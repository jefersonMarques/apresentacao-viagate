package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type PasswordResetUser struct {
	ID    string
	Name  string
	Email string
}

func (s *Store) CreatePasswordReset(ctx context.Context, email string, tokenHash []byte, expiresAt time.Time) (PasswordResetUser, bool, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return PasswordResetUser{}, false, err
	}
	defer tx.Rollback(ctx)

	var user PasswordResetUser
	err = tx.QueryRow(ctx, `select id::text,name,email::text from users where email=$1 and status='active' for update`, email).Scan(&user.ID, &user.Name, &user.Email)
	if err == pgx.ErrNoRows {
		if err := tx.Commit(ctx); err != nil {
			return PasswordResetUser{}, false, err
		}
		return PasswordResetUser{}, false, nil
	}
	if err != nil {
		return PasswordResetUser{}, false, err
	}

	if _, err := tx.Exec(ctx, `update password_reset_tokens set expires_at=now() where user_id=$1 and used_at is null and expires_at>now()`, user.ID); err != nil {
		return PasswordResetUser{}, false, err
	}
	if _, err := tx.Exec(ctx, `insert into password_reset_tokens(user_id,token_hash,expires_at) values($1,$2,$3)`, user.ID, tokenHash, expiresAt); err != nil {
		return PasswordResetUser{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return PasswordResetUser{}, false, err
	}
	return user, true, nil
}

func (s *Store) PasswordResetByToken(ctx context.Context, tokenHash []byte) (PasswordResetUser, error) {
	var user PasswordResetUser
	err := s.pool.QueryRow(ctx, `
		select u.id::text,u.name,u.email::text
		from password_reset_tokens t
		join users u on u.id=t.user_id
		where t.token_hash=$1 and t.used_at is null and t.expires_at>now() and u.status='active'
	`, tokenHash).Scan(&user.ID, &user.Name, &user.Email)
	return user, err
}

func (s *Store) ResetPassword(ctx context.Context, tokenHash []byte, passwordHash string) (PasswordResetUser, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return PasswordResetUser{}, err
	}
	defer tx.Rollback(ctx)

	var tokenID string
	var user PasswordResetUser
	err = tx.QueryRow(ctx, `
		select t.id::text,u.id::text,u.name,u.email::text
		from password_reset_tokens t
		join users u on u.id=t.user_id
		where t.token_hash=$1 and t.used_at is null and t.expires_at>now() and u.status='active'
		for update of t,u
	`, tokenHash).Scan(&tokenID, &user.ID, &user.Name, &user.Email)
	if err != nil {
		return PasswordResetUser{}, err
	}
	if _, err := tx.Exec(ctx, `update users set password_hash=$2,updated_at=now() where id=$1 and status='active'`, user.ID, passwordHash); err != nil {
		return PasswordResetUser{}, err
	}
	if _, err := tx.Exec(ctx, `update password_reset_tokens set used_at=now() where id=$1`, tokenID); err != nil {
		return PasswordResetUser{}, err
	}
	if _, err := tx.Exec(ctx, `update password_reset_tokens set expires_at=now() where user_id=$1 and id<>$2 and used_at is null and expires_at>now()`, user.ID, tokenID); err != nil {
		return PasswordResetUser{}, err
	}
	if _, err := tx.Exec(ctx, `update sessions set revoked_at=coalesce(revoked_at,now()) where user_id=$1 and revoked_at is null`, user.ID); err != nil {
		return PasswordResetUser{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return PasswordResetUser{}, fmt.Errorf("commit password reset: %w", err)
	}
	return user, nil
}
