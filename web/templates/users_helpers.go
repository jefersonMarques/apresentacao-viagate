package templates

import (
	"sort"
	"strings"

	"github.com/jefersonMarques/apresentacao-viagate/internal/access"
	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
)

type PermissionGroup struct {
	Name  string
	Items []domain.PermissionSetting
}

func PrimaryRole(user domain.User) string {
	for _, role := range []string{"super_admin", "admin", "user"} {
		if access.HasRole(user, role) {
			return role
		}
	}
	if len(user.Roles) == 0 {
		return "user"
	}
	return user.Roles[0]
}

func UserCan(user domain.User, permission string) bool { return access.Can(user, permission) }

func CanManageUser(actor, target domain.User) bool { return access.CanManageAccount(actor, target) }

func CanAssignAdminRoles(actor domain.User) bool { return access.IsSuperAdmin(actor) }

func PermissionVisibleToManager(actor domain.User, item domain.PermissionSetting) bool {
	if access.IsSuperAdmin(actor) {
		return true
	}
	if !access.IsAdmin(actor) {
		return false
	}
	if strings.HasPrefix(item.Code, "user.") || strings.HasPrefix(item.Code, "system.") || strings.HasPrefix(item.Code, "settings.") {
		return false
	}
	if item.Code == access.AuditRead || item.Code == access.AuditTechnicalRead {
		return false
	}
	return true
}

func PermissionGroups(actor domain.User, settings []domain.PermissionSetting) []PermissionGroup {
	index := map[string]int{}
	groups := []PermissionGroup{}
	for _, item := range settings {
		if !PermissionVisibleToManager(actor, item) {
			continue
		}
		position, ok := index[item.Group]
		if !ok {
			position = len(groups)
			index[item.Group] = position
			groups = append(groups, PermissionGroup{Name: item.Group})
		}
		groups[position].Items = append(groups[position].Items, item)
	}
	sort.SliceStable(groups, func(i, j int) bool { return groups[i].Name < groups[j].Name })
	return groups
}

func NotificationPreferenceEnabled(preferences []domain.NotificationPreference, eventType, scope string, fallback bool) bool {
	for _, item := range preferences {
		if item.EventType == eventType && item.Scope == scope {
			return item.Enabled
		}
	}
	return fallback
}

func UserHasPermission(user domain.User, code string) bool {
	return access.Can(user, code)
}
