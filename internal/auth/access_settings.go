package auth

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
)

func (s *Store) ManagedUserByID(ctx context.Context, id string) (domain.User, error) {
	var user domain.User
	var roles, permissions []string
	err := s.pool.QueryRow(ctx, `
		select u.id::text,u.email::text,u.name,u.status::text,u.created_at,
		       coalesce(array_agg(distinct r.code) filter(where r.code is not null),'{}'),
		       coalesce(array(select permission_code from effective_user_permissions(u.id) order by permission_code),'{}')
		from users u
		left join user_roles ur on ur.user_id=u.id
		left join roles r on r.id=ur.role_id
		where u.id=$1
		group by u.id
	`, id).Scan(&user.ID, &user.Email, &user.Name, &user.Status, &user.CreatedAt, &roles, &permissions)
	if err != nil {
		return domain.User{}, err
	}
	user.Roles = roles
	user.Permissions = permissions
	return user, nil
}

func (s *Store) PermissionSettings(ctx context.Context, userID string) ([]domain.PermissionSetting, error) {
	rows, err := s.pool.Query(ctx, `
		select p.code,p.name,
		       exists(select 1 from effective_user_permissions($1) effective where effective.permission_code=p.code),
		       upo.allowed
		from permissions p
		left join user_permission_overrides upo on upo.permission_id=p.id and upo.user_id=$1
		order by p.code
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := []domain.PermissionSetting{}
	for rows.Next() {
		var item domain.PermissionSetting
		var override *bool
		if err := rows.Scan(&item.Code, &item.Name, &item.Effective, &override); err != nil {
			return nil, err
		}
		item.Override = override
		item.Group = permissionGroup(item.Code)
		settings = append(settings, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(settings, func(i, j int) bool {
		if settings[i].Group == settings[j].Group {
			return settings[i].Name < settings[j].Name
		}
		return settings[i].Group < settings[j].Group
	})
	return settings, nil
}

// SetPermissionOverrides changes only the supplied permission codes. This lets
// an Admin manage the permitted subset of a User account without erasing
// technical or migrated overrides that only a Superadmin may control.
func (s *Store) SetPermissionOverrides(ctx context.Context, targetUserID, actorUserID string, values map[string]bool) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for code, allowed := range values {
		command, err := tx.Exec(ctx, `
			insert into user_permission_overrides(user_id,permission_id,allowed,updated_by,updated_at)
			select $1,p.id,$3,$2,now() from permissions p where p.code=$4
			on conflict(user_id,permission_id) do update
			set allowed=excluded.allowed,updated_by=excluded.updated_by,updated_at=excluded.updated_at
		`, targetUserID, actorUserID, allowed, code)
		if err != nil {
			return err
		}
		if command.RowsAffected() != 1 {
			return fmt.Errorf("unknown permission %q", code)
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) NotificationPreferences(ctx context.Context, userID string) ([]domain.NotificationPreference, error) {
	rows, err := s.pool.Query(ctx, `
		select event_type,scope,enabled
		from user_notification_preferences
		where user_id=$1
		order by event_type,scope
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.NotificationPreference{}
	for rows.Next() {
		var item domain.NotificationPreference
		if err := rows.Scan(&item.EventType, &item.Scope, &item.Enabled); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ReplaceNotificationPreferences(ctx context.Context, userID string, values []domain.NotificationPreference) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `delete from user_notification_preferences where user_id=$1`, userID); err != nil {
		return err
	}
	for _, value := range values {
		if value.Scope != "own" && value.Scope != "all" {
			return fmt.Errorf("invalid notification scope")
		}
		if _, err := tx.Exec(ctx, `
			insert into user_notification_preferences(user_id,event_type,scope,enabled)
			values($1,$2,$3,$4)
		`, userID, value.EventType, value.Scope, value.Enabled); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func permissionGroup(code string) string {
	prefix, _, _ := strings.Cut(code, ".")
	switch prefix {
	case "proposal":
		return "Propostas"
	case "presentation":
		return "Apresentações"
	case "customer":
		return "Clientes"
	case "contract":
		return "Contratos"
	case "onboarding", "activation":
		return "Operação"
	case "user":
		return "Usuários"
	case "activity", "audit":
		return "Auditoria"
	case "notification":
		return "Notificações"
	case "settings", "system":
		return "Sistema"
	default:
		return "Outros"
	}
}
