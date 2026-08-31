package maintenance

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Worker struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewWorker(pool *pgxpool.Pool, logger *slog.Logger) *Worker {
	return &Worker{pool: pool, logger: logger}
}

func (w *Worker) Run(ctx context.Context) {
	w.cleanup(ctx)
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.cleanup(ctx)
		}
	}
}

func (w *Worker) cleanup(ctx context.Context) {
	statements := []struct {
		name string
		sql  string
	}{
		{name: "expired proposals", sql: `update proposals set status='expired',updated_at=now() where status='published' and valid_until is not null and valid_until<current_date`},
		{name: "login failures", sql: `delete from login_failures where created_at < now()-interval '24 hours'`},
		{name: "admin sessions", sql: `delete from sessions where expires_at < now()-interval '7 days' or (revoked_at is not null and revoked_at < now()-interval '7 days')`},
		{name: "customer sessions", sql: `delete from customer_sessions where expires_at < now()-interval '7 days' or (revoked_at is not null and revoked_at < now()-interval '7 days')`},
		{name: "customer resume tokens", sql: `delete from customer_resume_tokens where expires_at < now()-interval '7 days' or (used_at is not null and used_at < now()-interval '7 days')`},
		{name: "password reset tokens", sql: `delete from password_reset_tokens where expires_at < now()-interval '7 days' or (used_at is not null and used_at < now()-interval '7 days')`},
		{name: "activation access tokens", sql: `delete from activation_access_tokens where expires_at < now()-interval '30 days' or (revoked_at is not null and revoked_at < now()-interval '30 days')`},
		{name: "sensitive expired notifications", sql: `update notification_outbox set html_body='[conteúdo sensível expirado]',text_body='[conteúdo sensível expirado]' where sensitive=true and expires_at is not null and expires_at<=now() and (html_body not like '[conteúdo sensível%' or coalesce(text_body,'') not like '[conteúdo sensível%')`},
	}
	for _, statement := range statements {
		cleanupCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		_, err := w.pool.Exec(cleanupCtx, statement.sql)
		cancel()
		if err != nil {
			w.logger.Error("maintenance cleanup failed", "resource", statement.name, "error", err)
		}
	}
}
