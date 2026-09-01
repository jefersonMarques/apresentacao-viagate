package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (s *Store) CreateManagedInvitation(ctx context.Context, email, name, roleCode string, tokenHash []byte, expiresAt time.Time, createdBy string) (string, error) {
	if !managedRole(roleCode) {
		return "", fmt.Errorf("invalid role")
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var roleID string
	if err := tx.QueryRow(ctx, `select id::text from roles where code=$1`, roleCode).Scan(&roleID); err != nil {
		return "", fmt.Errorf("invalid role")
	}

	var userID, status string
	err = tx.QueryRow(ctx, `select id::text,status::text from users where email=$1 for update`, email).Scan(&userID, &status)
	if err == pgx.ErrNoRows {
		if err := tx.QueryRow(ctx, `insert into users(email,name,status) values($1,$2,'invited') returning id::text`, email, name).Scan(&userID); err != nil {
			return "", err
		}
		status = "invited"
	} else if err != nil {
		return "", err
	} else {
		if status == "active" {
			return "", fmt.Errorf("user is already active")
		}
		if status == "disabled" {
			return "", fmt.Errorf("user is disabled; enable the existing account instead")
		}
		if _, err := tx.Exec(ctx, `update users set name=$2,updated_at=now() where id=$1`, userID, name); err != nil {
			return "", err
		}
	}

	if _, err := tx.Exec(ctx, `delete from user_roles where user_id=$1`, userID); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `delete from user_permission_overrides where user_id=$1`, userID); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `insert into user_roles(user_id,role_id) values($1,$2)`, userID, roleID); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `update user_invitations set expires_at=now() where user_id=$1 and accepted_at is null and expires_at>now()`, userID); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `insert into user_invitations(user_id,token_hash,expires_at,created_by) values($1,$2,$3,$4)`, userID, tokenHash, expiresAt, createdBy); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return userID, nil
}

func (s *Store) UpdateAccess(ctx context.Context, targetUserID, status, roleCode string) error {
	if status != "active" && status != "disabled" && status != "invited" {
		return fmt.Errorf("invalid user status")
	}
	if !managedRole(roleCode) {
		return fmt.Errorf("invalid role")
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var currentStatus, currentRole string
	if err := tx.QueryRow(ctx, `
		select u.status::text,coalesce((
			select r.code from user_roles ur join roles r on r.id=ur.role_id
			where ur.user_id=u.id and r.code in ('super_admin','admin','user')
			order by case r.code when 'super_admin' then 1 when 'admin' then 2 else 3 end
			limit 1
		),'user')
		from users u where u.id=$1 for update
	`, targetUserID).Scan(&currentStatus, &currentRole); err != nil {
		return err
	}
	_ = currentStatus

	removingLastSuperAdmin := currentRole == "super_admin" && (status != "active" || roleCode != "super_admin")
	if removingLastSuperAdmin {
		var activeSuperAdmins int
		if err := tx.QueryRow(ctx, `
			select count(distinct u.id)
			from users u
			join user_roles ur on ur.user_id=u.id
			join roles r on r.id=ur.role_id
			where u.status='active' and r.code='super_admin'
		`).Scan(&activeSuperAdmins); err != nil {
			return err
		}
		if activeSuperAdmins <= 1 {
			return fmt.Errorf("cannot remove the last active super admin")
		}
	}

	if _, err := tx.Exec(ctx, `update users set status=$2,updated_at=now() where id=$1`, targetUserID, status); err != nil {
		return err
	}
	if currentRole != roleCode {
		if _, err := tx.Exec(ctx, `delete from user_permission_overrides where user_id=$1`, targetUserID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `delete from user_roles where user_id=$1`, targetUserID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		insert into user_roles(user_id,role_id)
		select $1,id from roles where code=$2
	`, targetUserID, roleCode); err != nil {
		return err
	}
	if status == "disabled" {
		if _, err := tx.Exec(ctx, `update sessions set revoked_at=coalesce(revoked_at,now()) where user_id=$1 and revoked_at is null`, targetUserID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) UserByID(ctx context.Context, id string) (string, string, string, error) {
	var name, email, status string
	err := s.pool.QueryRow(ctx, `select name,email::text,status::text from users where id=$1`, id).Scan(&name, &email, &status)
	return name, email, status, err
}

func managedRole(role string) bool {
	return role == "user" || role == "admin" || role == "super_admin"
}
