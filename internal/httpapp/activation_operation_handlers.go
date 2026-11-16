package httpapp

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/postsale"
	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

func (a *App) saveActivationOperation(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	access, err := a.activationStore.AccessByToken(r.Context(), hashToken(token))
	if err != nil {
		http.Error(w, "Link de implantação inválido ou expirado.", http.StatusGone)
		return
	}
	if !access.Profile.RequiresPolicy || access.Section != "all" ||
		access.Profile.Status == "completed" || access.Profile.Status == "under_internal_setup" ||
		access.Profile.Status == "activated" {
		http.Error(w, "Este acesso não permite alterar a operação.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Dados inválidos.", http.StatusBadRequest)
		return
	}
	operation := postsale.Operation{
		Type: strings.ToLower(strings.TrimSpace(r.FormValue("operation_type"))),
		Insurer: strings.TrimSpace(r.FormValue("insurer")),
		StartDate: strings.TrimSpace(r.FormValue("policy_start_date")),
		EndDate: strings.TrimSpace(r.FormValue("policy_end_date")),
		BrokerCompany: strings.TrimSpace(r.FormValue("broker_company")),
		BrokerProducer: strings.TrimSpace(r.FormValue("broker_producer")),
	}
	if err := postsale.ValidateOperation(operation); err != nil {
		render(r.Context(), w, http.StatusBadRequest, templates.ActivationPage(access, token, "", "Informe uma operação e uma vigência válidas."))
		return
	}
	if err := a.postSaleStore.SaveOperation(r.Context(), access.Profile.ContractID, operation, postsale.Actor{
		Type: "customer", TokenID: access.TokenID, Name: "Acesso por link compartilhado", IP: requestIP(r), Agent: r.UserAgent(),
	}); err != nil {
		a.logger.Error("save public post-sale operation failed", "contract_id", access.Profile.ContractID, "error", err)
		http.Error(w, "Não foi possível salvar os dados da operação.", http.StatusConflict)
		return
	}
	http.Redirect(w, r, "/activation/"+token+"?saved=operation", http.StatusSeeOther)
}
