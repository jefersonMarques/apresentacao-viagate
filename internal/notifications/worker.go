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
	// Recupera mensagens que ficaram presas caso uma instância tenha encerrado
	// depois do claim e antes de concluir o envio.
	_,_ = w.pool.Exec(ctx,`
		update notification_outbox
		set status='pending',processing_at=null,available_at=now()
		where status='processing' and processing_at < now()-interval '5 minutes'
	`)

	messages,err:=w.claimBatch(ctx,20)
	if err!=nil{return err}
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
				set status='pending',processing_at=null,last_error=$2,
				    available_at=now() + make_interval(secs => least(3600, power(2,attempts)::int * 15))
				where id=$1 and status='processing'
			`,message.ID,err.Error())
			continue
		}
		if _,err := w.pool.Exec(ctx,`
			update notification_outbox
			set status='sent',sent_at=now(),processing_at=null,last_error=null
			where id=$1 and status='processing'
		`,message.ID); err != nil {
			return fmt.Errorf("mark notification sent: %w",err)
		}
	}
	return nil
}

func (w *Worker) claimBatch(ctx context.Context,limit int)([]queuedMessage,error){
	tx,err:=w.pool.BeginTx(ctx,pgx.TxOptions{IsoLevel:pgx.ReadCommitted})
	if err!=nil{return nil,err}
	defer tx.Rollback(ctx)

	rows,err:=tx.Query(ctx,`
		select id::text,recipient::text,coalesce(recipient_name,''),subject,html_body,coalesce(text_body,'')
		from notification_outbox
		where status='pending' and available_at<=now() and attempts<8
		order by created_at
		for update skip locked
		limit $1
	`,limit)
	if err!=nil{return nil,err}
	messages:=[]queuedMessage{}
	ids:=[]string{}
	for rows.Next(){
		var message queuedMessage
		if err:=rows.Scan(&message.ID,&message.Recipient,&message.RecipientName,&message.Subject,&message.HTMLBody,&message.TextBody);err!=nil{rows.Close();return nil,err}
		messages=append(messages,message)
		ids=append(ids,message.ID)
	}
	if err:=rows.Err();err!=nil{rows.Close();return nil,err}
	rows.Close()
	for _,id:=range ids{
		if _,err:=tx.Exec(ctx,`
			update notification_outbox
			set status='processing',processing_at=now(),attempts=attempts+1
			where id=$1
		`,id);err!=nil{return nil,err}
	}
	if err:=tx.Commit(ctx);err!=nil{return nil,err}
	return messages,nil
}

func Enqueue(ctx context.Context,pool *pgxpool.Pool,toName,toEmail,subject,htmlBody,textBody string) error {
	_,err := pool.Exec(ctx,`
		insert into notification_outbox(recipient,recipient_name,subject,html_body,text_body)
		values($1,nullif($2,''),$3,$4,nullif($5,''))
	`,toEmail,toName,subject,htmlBody,textBody)
	return err
}

func EnqueueUnique(ctx context.Context,pool *pgxpool.Pool,dedupeKey,toName,toEmail,subject,htmlBody,textBody string) error {
	_,err:=pool.Exec(ctx,`
		insert into notification_outbox(dedupe_key,recipient,recipient_name,subject,html_body,text_body)
		values(nullif($1,''),$2,nullif($3,''),$4,$5,nullif($6,''))
		on conflict (dedupe_key) where dedupe_key is not null do nothing
	`,dedupeKey,toEmail,toName,subject,htmlBody,textBody)
	return err
}
