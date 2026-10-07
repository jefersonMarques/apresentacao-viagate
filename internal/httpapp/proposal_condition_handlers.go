package httpapp

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

func (a *App) proposalConditionsPage(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	conditions, err := a.catalogStore.ListProposalConditions(r.Context(), nil, true)
	if err != nil {
		a.logger.Error("load proposal special conditions failed", "error", err)
		http.Error(w, "não foi possível carregar as condições especiais", http.StatusInternalServerError)
		return
	}
	categories, err := a.catalogStore.ListAdmin(r.Context(), false)
	if err != nil {
		a.logger.Error("load product categories for proposal conditions failed", "error", err)
		http.Error(w, "não foi possível carregar as categorias de produtos", http.StatusInternalServerError)
		return
	}

	message := ""
	if r.URL.Query().Get("saved") == "1" {
		message = "Condição especial salva."
	}
	render(r.Context(), w, http.StatusOK, templates.ProposalConditionsPage(
		user,
		conditions,
		categories,
		message,
		strings.TrimSpace(r.URL.Query().Get("error")),
	))
}

func (a *App) saveProposalCondition(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, "dados inválidos", http.StatusBadRequest)
		return
	}

	sortOrder := 0
	if raw := strings.TrimSpace(r.FormValue("sort_order")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			redirectProposalConditions(w, r, "Ordem inválida.")
			return
		}
		sortOrder = value
	}

	id, err := a.catalogStore.SaveProposalCondition(
		r.Context(),
		strings.TrimSpace(r.FormValue("id")),
		strings.TrimSpace(r.FormValue("condition_text")),
		r.Form["group_code"],
		r.FormValue("is_active") == "1",
		sortOrder,
	)
	if err != nil {
		a.logger.Error("save proposal special condition failed", "user_id", user.ID, "condition_id", r.FormValue("id"), "error", err)
		redirectProposalConditions(w, r, "Não foi possível salvar a condição. Verifique se já existe uma condição com o mesmo texto.")
		return
	}

	_, _ = a.pool.Exec(r.Context(), `
		insert into audit_events(actor_user_id,event_type,resource_type,resource_id,metadata)
		values($1,'proposal_condition.saved','proposal_condition',$2,jsonb_build_object('active',$3::boolean))
	`, user.ID, id, r.FormValue("is_active") == "1")
	http.Redirect(w, r, "/admin/proposal-conditions?saved=1", http.StatusSeeOther)
}

func redirectProposalConditions(w http.ResponseWriter, r *http.Request, errorMessage string) {
	values := url.Values{}
	if strings.TrimSpace(errorMessage) != "" {
		values.Set("error", errorMessage)
	}
	target := "/admin/proposal-conditions"
	if encoded := values.Encode(); encoded != "" {
		target += "?" + encoded
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}
