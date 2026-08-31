package httpapp

import (
	"net/http"
	"strings"

	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

func (a *App) auditPage(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	entries, err := a.auditStore.List(r.Context(), query, 250)
	if err != nil {
		a.logger.Error("load audit log failed", "error", err)
		http.Error(w, "Não foi possível carregar a auditoria.", http.StatusInternalServerError)
		return
	}
	render(r.Context(), w, http.StatusOK, templates.AuditPage(user, entries, query))
}
