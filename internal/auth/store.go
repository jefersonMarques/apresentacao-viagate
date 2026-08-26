package auth

import (
	"context"
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
	var roles []string
	err := s.pool.QueryRow(ctx, `
		select u.id::text, u.email::text, u.name, u.status::text, coalesce(u.password_hash,''),
		       coalesce(array_agg(r.code) filter (where r.code is not null), '{}')
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
	)
	if err != nil {
		return Credentials{}, err
	}
	result.User.Roles = roles
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
	var roles []string
	err := s.pool.QueryRow(ctx, `
		select u.id::text, u.email::text, u.name, u.status::text, u.created_at,
		       coalesce(array_agg(r.code) filter (where r.code is not null), '{}')
		from sessions s
		join users u on u.id = s.user_id
		left join user_roles ur on ur.user_id = u.id
		left join roles r on r.id = ur.role_id
		where s.token_hash = $1 and s.revoked_at is null and s.expires_at > now() and u.status = 'active'
		group by u.id
	`, tokenHash).Scan(&user.ID, &user.Email, &user.Name, &user.Status, &user.CreatedAt, &roles)
	if err != nil {
		return domain.User{}, err
	}
	user.Roles = roles
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
			select 1
			from user_roles ur
			join role_permissions rp on rp.role_id = ur.role_id
			join permissions p on p.id = rp.permission_id
			where ur.user_id = $1 and p.code = $2
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
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var userID string
	err = tx.QueryRow(ctx, `
		insert into users(email,name,status)
		values ($1,$2,'invited')
		on conflict (email) do update set name = excluded.name
		returning id::text
	`, email, name).Scan(&userID)
	if err != nil {
		return "", fmt.Errorf("upsert invited user: %w", err)
	}

	commandTag, err := tx.Exec(ctx, `
		insert into user_roles(user_id, role_id)
		select $1, id from roles where code = $2
		on conflict do nothing
	`, userID, roleCode)
	if err != nil {
		return "", fmt.Errorf("assign invited role: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		var roleExists bool
		if err := tx.QueryRow(ctx, `select exists(select 1 from roles where code=$1)`, roleCode).Scan(&roleExists); err != nil || !roleExists {
			return "", fmt.Errorf("invalid role %q", roleCode)
		}
	}

	_, err = tx.Exec(ctx, `
		insert into user_invitations(user_id, token_hash, expires_at, created_by)
		values ($1,$2,$3,$4)
	`, userID, tokenHash, expiresAt, createdBy)
	if err != nil {
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
		where i.token_hash=$1 and i.accepted_at is null and i.expires_at > now()
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
		where i.token_hash=$1 and i.accepted_at is null and i.expires_at > now()
		for update of i
	`, tokenHash).Scan(&user.ID, &user.Email, &user.Name)
	if err != nil {
		return domain.User{}, err
	}

	if _, err := tx.Exec(ctx, `update users set password_hash=$2,status='active',updated_at=now() where id=$1`, user.ID, passwordHash); err != nil {
		return domain.User{}, err
	}
	if _, err := tx.Exec(ctx, `update user_invitations set accepted_at=now() where token_hash=$1`, tokenHash); err != nil {
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
