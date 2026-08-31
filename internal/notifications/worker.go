package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
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
	Sensitive     bool
	ExpiresAt     *time.Time
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
	_, _ = w.pool.Exec(ctx, `
		update notification_outbox
		set status='pending',processing_at=null,available_at=now()
		where status='processing' and processing_at < now()-interval '5 minutes'
		  and (expires_at is null or expires_at>now())
	`)
	_, _ = w.pool.Exec(ctx, `
		update notification_outbox
		set status='expired',processing_at=null,last_error='message expired before delivery',
		    html_body=case when sensitive then '[conteúdo sensível expirado]' else html_body end,
		    text_body=case when sensitive then '[conteúdo sensível expirado]' else text_body end
		where status in ('pending','processing') and expires_at is not null and expires_at<=now()
	`)

	messages, err := w.claimBatch(ctx, 20)
	if err != nil {
		return err
	}
	for _, message := range messages {
		if message.ExpiresAt != nil && !message.ExpiresAt.After(time.Now()) {
			_, _ = w.pool.Exec(ctx, `
				update notification_outbox
				set status='expired',processing_at=null,last_error='message expired before delivery',
				    html_body=case when sensitive then '[conteúdo sensível expirado]' else html_body end,
				    text_body=case when sensitive then '[conteúdo sensível expirado]' else text_body end
				where id=$1 and status='processing'
			`, message.ID)
			continue
		}

		sendCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err := w.mailer.Send(sendCtx, email.Message{
			ToName:   message.RecipientName,
			ToEmail:  message.Recipient,
			Subject:  message.Subject,
			HTMLBody: message.HTMLBody,
			TextBody: message.TextBody,
		})
		cancel()
		if err != nil {
			w.logger.Error(
				"notification delivery failed",
				"notification_id", message.ID,
				"recipient", message.Recipient,
				"subject", message.Subject,
				"error", err,
			)
			_, _ = w.pool.Exec(ctx, `
				update notification_outbox
				set status='pending',processing_at=null,last_error=$2,
				    available_at=now() + make_interval(secs => least(3600, power(2,attempts)::int * 15))
				where id=$1 and status='processing'
			`, message.ID, err.Error())
			continue
		}
		if _, err := w.pool.Exec(ctx, `
			update notification_outbox
			set status='sent',sent_at=now(),processing_at=null,last_error=null,
			    html_body=case when sensitive then '[conteúdo sensível removido após envio]' else html_body end,
			    text_body=case when sensitive then '[conteúdo sensível removido após envio]' else text_body end
			where id=$1 and status='processing'
		`, message.ID); err != nil {
			return fmt.Errorf("mark notification sent: %w", err)
		}
		w.logger.Info(
			"notification delivered to provider",
			"notification_id", message.ID,
			"recipient", message.Recipient,
			"subject", message.Subject,
		)
	}
	return nil
}

func (w *Worker) claimBatch(ctx context.Context, limit int) ([]queuedMessage, error) {
	tx, err := w.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		select id::text,recipient::text,coalesce(recipient_name,''),subject,html_body,coalesce(text_body,''),sensitive,expires_at
		from notification_outbox
		where status='pending' and available_at<=now() and attempts<8
		  and (expires_at is null or expires_at>now())
		order by created_at
		for update skip locked
		limit $1
	`, limit)
	if err != nil {
		return nil, err
	}
	messages := []queuedMessage{}
	ids := []string{}
	for rows.Next() {
		var message queuedMessage
		if err := rows.Scan(&message.ID, &message.Recipient, &message.RecipientName, &message.Subject, &message.HTMLBody, &message.TextBody, &message.Sensitive, &message.ExpiresAt); err != nil {
			rows.Close()
			return nil, err
		}
		messages = append(messages, message)
		ids = append(ids, message.ID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	for _, id := range ids {
		if _, err := tx.Exec(ctx, `
			update notification_outbox
			set status='processing',processing_at=now(),attempts=attempts+1
			where id=$1
		`, id); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return messages, nil
}

func Enqueue(ctx context.Context, pool *pgxpool.Pool, toName, toEmail, subject, htmlBody, textBody string) error {
	return EnqueueWithOptions(ctx, pool, MessageOptions{
		ToName: toName, ToEmail: toEmail, Subject: subject, HTMLBody: htmlBody, TextBody: textBody,
	})
}

func EnqueueUnique(ctx context.Context, pool *pgxpool.Pool, dedupeKey, toName, toEmail, subject, htmlBody, textBody string) error {
	return EnqueueWithOptions(ctx, pool, MessageOptions{
		DedupeKey: dedupeKey, ToName: toName, ToEmail: toEmail, Subject: subject, HTMLBody: htmlBody, TextBody: textBody,
	})
}

type MessageOptions struct {
	DedupeKey string
	Kind      string
	ToName    string
	ToEmail   string
	Subject   string
	HTMLBody  string
	TextBody  string
	ExpiresAt *time.Time
	Sensitive bool
}

func EnqueueWithOptions(ctx context.Context, pool *pgxpool.Pool, options MessageOptions) error {
	kind := options.Kind
	if kind == "" {
		kind = "generic"
	}
	_, err := pool.Exec(ctx, `
		insert into notification_outbox(
			dedupe_key,kind,recipient,recipient_name,subject,html_body,text_body,expires_at,sensitive
		) values(nullif($1,''),$2,$3,nullif($4,''),$5,$6,nullif($7,''),$8,$9)
		on conflict (dedupe_key) where dedupe_key is not null do nothing
	`, options.DedupeKey, kind, options.ToEmail, options.ToName, options.Subject, options.HTMLBody, options.TextBody, options.ExpiresAt, options.Sensitive)
	return err
}
