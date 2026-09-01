package httpapp

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/access"
	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

func (a *App) pipelinePage(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	all := access.Can(user, access.ProposalReadAll)
	if !all && !access.Can(user, access.ProposalReadOwn) {
		http.Error(w, "acesso negado", http.StatusForbidden)
		return
	}
	items, err := a.pipelineStore.List(r.Context(), user.ID, all)
	if err != nil {
		a.logger.Error("load pipeline failed", "error", err)
		http.Error(w, "não foi possível carregar o pipeline", http.StatusInternalServerError)
		return
	}
	render(r.Context(), w, http.StatusOK, templates.PipelinePage(user, items, all))
}

func (a *App) pipelineDetailPage(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	proposalID := chi.URLParam(r, "proposalID")
	all := access.Can(user, access.ProposalReadAll)
	if !all && !access.Can(user, access.ProposalReadOwn) {
		http.Error(w, "acesso negado", http.StatusForbidden)
		return
	}
	var owner string
	if err := a.pool.QueryRow(r.Context(), `select created_by::text from proposals where id=$1`, proposalID).Scan(&owner); err != nil {
		http.Error(w, "negociação não encontrada", http.StatusNotFound)
		return
	}
	if !all && owner != user.ID {
		http.Error(w, "acesso negado", http.StatusForbidden)
		return
	}
	items, err := a.pipelineStore.List(r.Context(), user.ID, all)
	if err != nil {
		http.Error(w, "não foi possível carregar a negociação", http.StatusInternalServerError)
		return
	}
	selectedIndex := -1
	for index, item := range items {
		if item.ProposalID == proposalID {
			selectedIndex = index
			break
		}
	}
	if selectedIndex < 0 {
		http.Error(w, "negociação não encontrada", http.StatusNotFound)
		return
	}
	events, err := a.pipelineStore.Timeline(r.Context(), proposalID)
	if err != nil {
		a.logger.Error("load pipeline timeline failed", "error", err)
		http.Error(w, "não foi possível carregar o histórico", http.StatusInternalServerError)
		return
	}
	render(r.Context(), w, http.StatusOK, templates.PipelineDetailPage(user, items[selectedIndex], events))
}
