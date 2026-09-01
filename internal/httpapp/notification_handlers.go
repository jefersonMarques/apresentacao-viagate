package httpapp

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/notifications"
	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

func (a *App) notificationsPage(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	items, err := notifications.NewInAppStore(a.pool).List(r.Context(), user.ID, 100)
	if err != nil {
		a.logger.Error("list in-app notifications failed", "user_id", user.ID, "error", err)
		http.Error(w, "não foi possível carregar as notificações", http.StatusInternalServerError)
		return
	}
	render(r.Context(), w, http.StatusOK, templates.NotificationsPage(user, items))
}

func (a *App) markNotificationRead(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	if err := notifications.NewInAppStore(a.pool).MarkRead(r.Context(), user.ID, chi.URLParam(r, "id")); err != nil {
		http.Error(w, "não foi possível atualizar a notificação", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/notifications", http.StatusSeeOther)
}

func (a *App) openNotification(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, "dados inválidos", http.StatusBadRequest)
		return
	}
	if err := notifications.NewInAppStore(a.pool).MarkRead(r.Context(), user.ID, chi.URLParam(r, "id")); err != nil {
		http.Error(w, "não foi possível atualizar a notificação", http.StatusInternalServerError)
		return
	}
	target := strings.TrimSpace(r.FormValue("target"))
	if !strings.HasPrefix(target, "/admin/") && target != "/admin" {
		target = "/admin/notifications"
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (a *App) markAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	if err := notifications.NewInAppStore(a.pool).MarkAllRead(r.Context(), user.ID); err != nil {
		http.Error(w, "não foi possível atualizar as notificações", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/notifications", http.StatusSeeOther)
}
