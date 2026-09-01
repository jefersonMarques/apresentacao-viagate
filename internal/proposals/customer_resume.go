package proposals

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Store) CreateCustomerResumeToken(ctx context.Context, acceptanceID string, tokenHash []byte, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		insert into customer_resume_tokens(proposal_acceptance_id,token_hash,expires_at,purpose)
		values($1,$2,$3,'correction')
	`, acceptanceID, tokenHash, expiresAt)
	return err
}

func (s *Store) ConsumeCustomerResumeToken(ctx context.Context, tokenHash []byte) (string, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var tokenID, acceptanceID string
	if err := tx.QueryRow(ctx, `
		select id::text,proposal_acceptance_id::text
		from customer_resume_tokens
		where token_hash=$1 and purpose='correction' and used_at is null and revoked_at is null and expires_at>now()
		for update
	`, tokenHash).Scan(&tokenID, &acceptanceID); err != nil {
		return "", err
	}
	command, err := tx.Exec(ctx, `update customer_resume_tokens set used_at=now(),last_used_at=now() where id=$1 and used_at is null`, tokenID)
	if err != nil {
		return "", err
	}
	if command.RowsAffected() != 1 {
		return "", fmt.Errorf("resume token was already consumed")
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return acceptanceID, nil
}

func (s *Store) CreateCustomerJourneyToken(ctx context.Context, acceptanceID string, tokenHash []byte, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		insert into customer_resume_tokens(proposal_acceptance_id,token_hash,expires_at,purpose)
		values($1,$2,$3,'journey')
	`, acceptanceID, tokenHash, expiresAt)
	return err
}

func (s *Store) CustomerJourneyAcceptance(ctx context.Context, tokenHash []byte) (string, error) {
	var tokenID, acceptanceID string
	err := s.pool.QueryRow(ctx, `
		select id::text,proposal_acceptance_id::text
		from customer_resume_tokens
		where token_hash=$1 and purpose='journey' and revoked_at is null and expires_at>now()
	`, tokenHash).Scan(&tokenID, &acceptanceID)
	if err != nil {
		return "", err
	}
	_, _ = s.pool.Exec(ctx, `update customer_resume_tokens set last_used_at=now() where id=$1`, tokenID)
	return acceptanceID, nil
}
