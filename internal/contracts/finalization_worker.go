package contracts

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FinalizationWorker struct {
	pool      *pgxpool.Pool
	finalizer *Finalizer
	logger    *slog.Logger
}

func NewFinalizationWorker(pool *pgxpool.Pool, finalizer *Finalizer, logger *slog.Logger) *FinalizationWorker {
	return &FinalizationWorker{pool: pool, finalizer: finalizer, logger: logger}
}

func (w *FinalizationWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.process(ctx); err != nil {
				w.logger.Error("contract finalization worker failed", "error", err)
			}
		}
	}
}

func (w *FinalizationWorker) process(ctx context.Context) error {
	_, _ = w.pool.Exec(ctx, `
		update contract_finalization_jobs
		set status='failed',processing_at=null,available_at=now(),last_error='processing lease expired',updated_at=now()
		where status='processing' and processing_at<now()-interval '5 minutes'
	`)

	contractID, ok, err := w.claim(ctx)
	if err != nil || !ok {
		return err
	}

	jobCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	err = w.finalizer.Finalize(jobCtx, contractID)
	cancel()
	if err != nil {
		w.logger.Error("contract evidence finalization failed", "contract_id", contractID, "error", err)
		_, updateErr := w.pool.Exec(ctx, `
			update contract_finalization_jobs
			set status='failed',processing_at=null,last_error=$2,
			    available_at=now()+make_interval(secs=>least(3600,power(2,attempts)::int*15)),updated_at=now()
			where contract_id=$1 and status='processing'
		`, contractID, err.Error())
		if updateErr != nil {
			return fmt.Errorf("mark contract finalization failed: %w", updateErr)
		}
		return nil
	}

	_, err = w.pool.Exec(ctx, `
		update contract_finalization_jobs
		set status='completed',processing_at=null,completed_at=now(),last_error=null,updated_at=now()
		where contract_id=$1 and status='processing'
	`, contractID)
	if err != nil {
		return fmt.Errorf("mark contract finalization completed: %w", err)
	}
	w.logger.Info("contract evidence finalized", "contract_id", contractID)
	return nil
}

func (w *FinalizationWorker) claim(ctx context.Context) (string, bool, error) {
	tx, err := w.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return "", false, err
	}
	defer tx.Rollback(ctx)

	var contractID string
	err = tx.QueryRow(ctx, `
		select contract_id::text
		from contract_finalization_jobs
		where status in ('pending','failed') and available_at<=now() and attempts<8
		order by available_at,created_at
		for update skip locked
		limit 1
	`).Scan(&contractID)
	if err == pgx.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if _, err := tx.Exec(ctx, `
		update contract_finalization_jobs
		set status='processing',processing_at=now(),attempts=attempts+1,updated_at=now()
		where contract_id=$1
	`, contractID); err != nil {
		return "", false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", false, err
	}
	return contractID, true, nil
}
