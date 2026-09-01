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
	preview := r.URL.Query().Get("preview") == "1"
	limit := 100
	if preview {
		limit = 6
	}
	items, err := notifications.NewInAppStore(a.pool).List(r.Context(), user.ID, limit)
	if err != nil {
		a.logger.Error("list in-app notifications failed", "user_id", user.ID, "error", err)
		http.Error(w, "não foi possível carregar as notificações", http.StatusInternalServerError)
		return
	}
	if preview {
		w.Header().Set("Cache-Control", "no-store")
		render(r.Context(), w, http.StatusOK, templates.NotificationPopover(items))
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
	if isHTMXRequest(r) {
		a.renderNotificationFeed(w, r, user.ID)
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
	if isHTMXRequest(r) {
		w.Header().Set("HX-Redirect", target)
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (a *App) markAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	if err := notifications.NewInAppStore(a.pool).MarkAllRead(r.Context(), user.ID); err != nil {
		http.Error(w, "não foi possível atualizar as notificações", http.StatusInternalServerError)
		return
	}
	if isHTMXRequest(r) {
		a.renderNotificationFeed(w, r, user.ID)
		return
	}
	http.Redirect(w, r, "/admin/notifications", http.StatusSeeOther)
}

func (a *App) renderNotificationFeed(w http.ResponseWriter, r *http.Request, userID string) {
	store := notifications.NewInAppStore(a.pool)
	items, err := store.List(r.Context(), userID, 100)
	if err != nil {
		a.logger.Error("reload in-app notifications failed", "user_id", userID, "error", err)
		http.Error(w, "não foi possível carregar as notificações", http.StatusInternalServerError)
		return
	}
	var unread int
	if err := a.pool.QueryRow(r.Context(), `select count(*) from in_app_notifications where recipient_user_id=$1 and read_at is null`, userID).Scan(&unread); err != nil {
		a.logger.Error("count unread notifications failed", "user_id", userID, "error", err)
		http.Error(w, "não foi possível atualizar as notificações", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	render(r.Context(), w, http.StatusOK, templates.NotificationFeedResponse(items, unread))
}
