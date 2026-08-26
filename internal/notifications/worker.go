package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/email"
)

type Worker struct {
	pool   *pgxpool.Pool
	mailer *email.Brevo
	logger *slog.Logger
}

type queuedMessage struct {
	ID            string
	Recipient     string
	RecipientName string
	Subject       string
	HTMLBody      string
	TextBody      string
}

func NewWorker(pool *pgxpool.Pool, mailer *email.Brevo, logger *slog.Logger) *Worker {
	return &Worker{pool: pool, mailer: mailer, logger: logger}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				w.logger.Error("notification outbox batch failed", "error", err)
			}
		}
	}
}

func (w *Worker) processBatch(ctx context.Context) error {
	rows, err := w.pool.Query(ctx, `
		select id::text,recipient::text,coalesce(recipient_name,''),subject,html_body,coalesce(text_body,'')
		from notification_outbox
		where status='pending' and available_at<=now() and attempts<8
		order by created_at
		limit 20
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	messages := []queuedMessage{}
	for rows.Next() {
		var message queuedMessage
		if err := rows.Scan(&message.ID,&message.Recipient,&message.RecipientName,&message.Subject,&message.HTMLBody,&message.TextBody); err != nil {
			return err
		}
		messages=append(messages,message)
	}
	if err := rows.Err(); err != nil { return err }

	for _, message := range messages {
		sendCtx,cancel := context.WithTimeout(ctx,10*time.Second)
		err := w.mailer.Send(sendCtx,email.Message{
			ToName:message.RecipientName,
			ToEmail:message.Recipient,
			Subject:message.Subject,
			HTMLBody:message.HTMLBody,
			TextBody:message.TextBody,
		})
		cancel()
		if err != nil {
			_,_ = w.pool.Exec(ctx,`
				update notification_outbox
				set attempts=attempts+1,last_error=$2,
				    available_at=now() + make_interval(secs => least(3600, power(2,attempts+1)::int * 15))
				where id=$1
			`,message.ID,err.Error())
			continue
		}
		if _,err := w.pool.Exec(ctx,`update notification_outbox set status='sent',sent_at=now(),attempts=attempts+1,last_error=null where id=$1`,message.ID); err != nil {
			return fmt.Errorf("mark notification sent: %w",err)
		}
	}
	return nil
}

func Enqueue(ctx context.Context,pool *pgxpool.Pool,toName,toEmail,subject,htmlBody,textBody string) error {
	_,err := pool.Exec(ctx,`
		insert into notification_outbox(recipient,recipient_name,subject,html_body,text_body)
		values($1,nullif($2,''),$3,$4,nullif($5,''))
	`,toEmail,toName,subject,htmlBody,textBody)
	return err
}
