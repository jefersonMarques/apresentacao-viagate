package httpapp

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/access"
	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
	notificationpkg "github.com/jefersonMarques/apresentacao-viagate/internal/notifications"
)

func (a *App) updateUserAccess(w http.ResponseWriter, r *http.Request) {
	actor, _ := currentUser(r.Context())
	targetID := chi.URLParam(r, "id")
	target, err := a.authStore.ManagedUserByID(r.Context(), targetID)
	if err != nil {
		http.Error(w, "usuário não encontrado", http.StatusNotFound)
		return
	}
	if !access.CanManageAccount(actor, target) {
		http.Error(w, "acesso negado", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "dados inválidos", http.StatusBadRequest)
		return
	}

	switch strings.TrimSpace(r.FormValue("mode")) {
	case "permissions":
		a.updateUserPermissions(w, r, actor, target)
	case "notifications":
		a.updateUserNotificationPreferences(w, r, actor, target)
	default:
		a.updateUserProfileAccess(w, r, actor, target)
	}
}

func (a *App) updateUserProfileAccess(w http.ResponseWriter, r *http.Request, actor, target domain.User) {
	if !access.Can(actor, access.UserStatusManage) {
		http.Error(w, "acesso negado", http.StatusForbidden)
		return
	}
	status := strings.TrimSpace(r.FormValue("status"))
	role := strings.TrimSpace(r.FormValue("role"))
	if !access.CanAssignRole(actor, role) {
		http.Error(w, "você não pode atribuir este perfil", http.StatusForbidden)
		return
	}
	if target.Status == "invited" && status == "active" {
		http.Redirect(w, r, "/admin/users?user="+target.ID+"&error="+queryEscape("Usuários convidados precisam ativar a conta pelo link antes de ficarem ativos."), http.StatusSeeOther)
		return
	}
	if err := a.authStore.UpdateAccess(r.Context(), target.ID, status, role); err != nil {
		a.logger.Error("update user access failed", "error", err, "target_user_id", target.ID)
		http.Redirect(w, r, "/admin/users?user="+target.ID+"&error="+queryEscape("Não foi possível alterar o acesso: "+err.Error()), http.StatusSeeOther)
		return
	}
	_, _ = a.pool.Exec(r.Context(), `
		insert into audit_events(actor_user_id,event_type,resource_type,resource_id,ip_address,user_agent,metadata)
		values($1,'user.access_updated','user',$2,$3,$4,jsonb_build_object('status',$5,'role',$6))
	`, actor.ID, target.ID, requestIP(r), r.UserAgent(), status, role)
	http.Redirect(w, r, "/admin/users?user="+target.ID+"&updated=1", http.StatusSeeOther)
}

func (a *App) updateUserPermissions(w http.ResponseWriter, r *http.Request, actor, target domain.User) {
	if !access.Can(actor, access.UserPermissionsManage) || !access.IsUser(target) {
		http.Error(w, "acesso negado", http.StatusForbidden)
		return
	}
	checked := stringSet(r.Form["permission"])
	values := map[string]bool{}
	for _, code := range r.Form["managed_permission"] {
		code = strings.TrimSpace(code)
		if code == "" || !permissionManageableBy(actor, code) {
			continue
		}
		values[code] = checked[code]
	}
	if err := a.authStore.SetPermissionOverrides(r.Context(), target.ID, actor.ID, values); err != nil {
		a.logger.Error("update user permissions failed", "error", err, "target_user_id", target.ID)
		http.Redirect(w, r, "/admin/users?user="+target.ID+"&error="+queryEscape("Não foi possível atualizar as permissões."), http.StatusSeeOther)
		return
	}
	_, _ = a.pool.Exec(r.Context(), `
		insert into audit_events(actor_user_id,event_type,resource_type,resource_id,ip_address,user_agent,metadata)
		values($1,'user.permissions_updated','user',$2,$3,$4,jsonb_build_object('managed_count',$5::integer))
	`, actor.ID, target.ID, requestIP(r), r.UserAgent(), len(values))
	http.Redirect(w, r, "/admin/users?user="+target.ID+"&permissions=1", http.StatusSeeOther)
}

func (a *App) updateUserNotificationPreferences(w http.ResponseWriter, r *http.Request, actor, target domain.User) {
	own := stringSet(r.Form["notify_own"])
	all := stringSet(r.Form["notify_all"])
	preferences := []domain.NotificationPreference{}
	seen := map[string]bool{}
	for _, eventType := range r.Form["notification_event"] {
		eventType = strings.TrimSpace(eventType)
		if eventType == "" || seen[eventType] || !notificationpkg.IsSupportedEvent(eventType) {
			continue
		}
		seen[eventType] = true
		preferences = append(preferences, domain.NotificationPreference{EventType: eventType, Scope: "own", Enabled: own[eventType]})
		if access.Can(target, access.NotificationReceiveOthers) {
			preferences = append(preferences, domain.NotificationPreference{EventType: eventType, Scope: "all", Enabled: all[eventType]})
		}
	}
	if err := a.authStore.ReplaceNotificationPreferences(r.Context(), target.ID, preferences); err != nil {
		a.logger.Error("update notification preferences failed", "error", err, "target_user_id", target.ID)
		http.Redirect(w, r, "/admin/users?user="+target.ID+"&error="+queryEscape("Não foi possível atualizar as notificações."), http.StatusSeeOther)
		return
	}
	_, _ = a.pool.Exec(r.Context(), `
		insert into audit_events(actor_user_id,event_type,resource_type,resource_id,ip_address,user_agent,metadata)
		values($1,'user.notification_preferences_updated','user',$2,$3,$4,'{}')
	`, actor.ID, target.ID, requestIP(r), r.UserAgent())
	http.Redirect(w, r, "/admin/users?user="+target.ID+"&notifications=1", http.StatusSeeOther)
}

func permissionManageableBy(actor domain.User, code string) bool {
	if access.IsSuperAdmin(actor) {
		return true
	}
	if !access.IsAdmin(actor) {
		return false
	}
	if strings.HasPrefix(code, "user.") || strings.HasPrefix(code, "system.") || strings.HasPrefix(code, "settings.") {
		return false
	}
	return code != access.AuditRead && code != access.AuditTechnicalRead
}

func stringSet(values []string) map[string]bool {
	result := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result[value] = true
		}
	}
	return result
}
