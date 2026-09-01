package auth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
)

type Store struct {
	pool *pgxpool.Pool
}

type Credentials struct {
	User         domain.User
	PasswordHash string
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) FindCredentials(ctx context.Context, email string) (Credentials, error) {
	var result Credentials
	var roles, permissions []string
	err := s.pool.QueryRow(ctx, `
		select u.id::text, u.email::text, u.name, u.status::text, coalesce(u.password_hash,''),
		       coalesce(array_agg(distinct r.code) filter (where r.code is not null), '{}'),
		       coalesce(array(select permission_code from effective_user_permissions(u.id) order by permission_code), '{}')
		from users u
		left join user_roles ur on ur.user_id = u.id
		left join roles r on r.id = ur.role_id
		where u.email = $1
		group by u.id
	`, email).Scan(
		&result.User.ID,
		&result.User.Email,
		&result.User.Name,
		&result.User.Status,
		&result.PasswordHash,
		&roles,
		&permissions,
	)
	if err != nil {
		return Credentials{}, err
	}
	result.User.Roles = roles
	result.User.Permissions = permissions
	return result, nil
}

func (s *Store) CreateSession(ctx context.Context, userID string, tokenHash []byte, ip net.IP, userAgent string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		insert into sessions(user_id, token_hash, ip_address, user_agent, expires_at)
		values ($1,$2,$3,$4,$5)
	`, userID, tokenHash, nullableIP(ip), userAgent, expiresAt)
	return err
}

func (s *Store) SessionUser(ctx context.Context, tokenHash []byte) (domain.User, error) {
	var user domain.User
	var roles, permissions []string
	err := s.pool.QueryRow(ctx, `
		select u.id::text, u.email::text, u.name, u.status::text, u.created_at,
		       coalesce(array_agg(distinct r.code) filter (where r.code is not null), '{}'),
		       coalesce(array(select permission_code from effective_user_permissions(u.id) order by permission_code), '{}'),
		       (select count(*)::int from in_app_notifications n where n.recipient_user_id=u.id and n.read_at is null)
		from sessions s
		join users u on u.id = s.user_id
		left join user_roles ur on ur.user_id = u.id
		left join roles r on r.id = ur.role_id
		where s.token_hash = $1 and s.revoked_at is null and s.expires_at > now() and u.status = 'active'
		group by u.id
	`, tokenHash).Scan(&user.ID, &user.Email, &user.Name, &user.Status, &user.CreatedAt, &roles, &permissions, &user.UnreadNotifications)
	if err != nil {
		return domain.User{}, err
	}
	user.Roles = roles
	user.Permissions = permissions
	return user, nil
}

func (s *Store) RevokeSession(ctx context.Context, tokenHash []byte) error {
	_, err := s.pool.Exec(ctx, `update sessions set revoked_at = now() where token_hash = $1 and revoked_at is null`, tokenHash)
	return err
}

func (s *Store) HasPermission(ctx context.Context, userID, permission string) (bool, error) {
	var allowed bool
	err := s.pool.QueryRow(ctx, `
		select exists(
			select 1 from effective_user_permissions($1) where permission_code=$2
		)
	`, userID, permission).Scan(&allowed)
	return allowed, err
}

func (s *Store) ListUsers(ctx context.Context) ([]domain.User, error) {
	rows, err := s.pool.Query(ctx, `
		select u.id::text, u.email::text, u.name, u.status::text, u.created_at,
		       coalesce(array_agg(r.code order by r.code) filter (where r.code is not null), '{}')
		from users u
		left join user_roles ur on ur.user_id = u.id
		left join roles r on r.id = ur.role_id
		group by u.id
		order by u.created_at desc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []domain.User{}
	for rows.Next() {
		var user domain.User
		var roles []string
		if err := rows.Scan(&user.ID, &user.Email, &user.Name, &user.Status, &user.CreatedAt, &roles); err != nil {
			return nil, err
		}
		user.Roles = roles
		users = append(users, user)
	}
	return users, rows.Err()
}

func (s *Store) CreateInvitation(ctx context.Context, email, name, roleCode string, tokenHash []byte, expiresAt time.Time, createdBy string) (string, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var roleID string
	if err := tx.QueryRow(ctx, `select id::text from roles where code=$1`, roleCode).Scan(&roleID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("invalid role %q", roleCode)
		}
		return "", err
	}

	var userID, status string
	err = tx.QueryRow(ctx, `select id::text,status::text from users where email=$1 for update`, email).Scan(&userID, &status)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if err := tx.QueryRow(ctx, `insert into users(email,name,status) values($1,$2,'invited') returning id::text`, email, name).Scan(&userID); err != nil {
			return "", fmt.Errorf("create invited user: %w", err)
		}
	case err != nil:
		return "", err
	case status != "invited":
		return "", fmt.Errorf("user already exists; update the existing account instead of sending a new invitation")
	default:
		if _, err := tx.Exec(ctx, `update users set name=$2,updated_at=now() where id=$1 and status='invited'`, userID, name); err != nil {
			return "", err
		}
	}

	if _, err := tx.Exec(ctx, `delete from user_roles where user_id=$1`, userID); err != nil {
		return "", fmt.Errorf("reset invited role: %w", err)
	}
	if _, err := tx.Exec(ctx, `insert into user_roles(user_id,role_id) values($1,$2)`, userID, roleID); err != nil {
		return "", fmt.Errorf("assign invited role: %w", err)
	}
	if _, err := tx.Exec(ctx, `update user_invitations set expires_at=now() where user_id=$1 and accepted_at is null and expires_at>now()`, userID); err != nil {
		return "", fmt.Errorf("expire prior invitations: %w", err)
	}

	if _, err = tx.Exec(ctx, `
		insert into user_invitations(user_id, token_hash, expires_at, created_by)
		values ($1,$2,$3,$4)
	`, userID, tokenHash, expiresAt, createdBy); err != nil {
		return "", fmt.Errorf("create invitation: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return userID, nil
}

func (s *Store) Invitation(ctx context.Context, tokenHash []byte) (domain.User, time.Time, error) {
	var user domain.User
	var expiresAt time.Time
	err := s.pool.QueryRow(ctx, `
		select u.id::text, u.email::text, u.name, u.status::text, i.expires_at
		from user_invitations i
		join users u on u.id = i.user_id
		where i.token_hash=$1 and i.accepted_at is null and i.expires_at > now() and u.status='invited'
	`, tokenHash).Scan(&user.ID, &user.Email, &user.Name, &user.Status, &expiresAt)
	return user, expiresAt, err
}

func (s *Store) AcceptInvitation(ctx context.Context, tokenHash []byte, passwordHash string) (domain.User, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return domain.User{}, err
	}
	defer tx.Rollback(ctx)

	var user domain.User
	err = tx.QueryRow(ctx, `
		select u.id::text, u.email::text, u.name
		from user_invitations i
		join users u on u.id=i.user_id
		where i.token_hash=$1 and i.accepted_at is null and i.expires_at > now() and u.status='invited'
		for update of i,u
	`, tokenHash).Scan(&user.ID, &user.Email, &user.Name)
	if err != nil {
		return domain.User{}, err
	}

	command, err := tx.Exec(ctx, `update users set password_hash=$2,status='active',updated_at=now() where id=$1 and status='invited'`, user.ID, passwordHash)
	if err != nil {
		return domain.User{}, err
	}
	if command.RowsAffected() != 1 {
		return domain.User{}, fmt.Errorf("invited account is no longer available")
	}
	if _, err := tx.Exec(ctx, `update user_invitations set accepted_at=now() where token_hash=$1`, tokenHash); err != nil {
		return domain.User{}, err
	}
	if _, err := tx.Exec(ctx, `update user_invitations set expires_at=now() where user_id=$1 and token_hash<>$2 and accepted_at is null and expires_at>now()`, user.ID, tokenHash); err != nil {
		return domain.User{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, err
	}
	user.Status = "active"
	return user, nil
}

func nullableIP(ip net.IP) any {
	if ip == nil {
		return nil
	}
	return ip.String()
}
