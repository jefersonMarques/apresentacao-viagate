package httpapp

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (a *App) proposalFromSignature(w http.ResponseWriter, r *http.Request) {
	access, err := a.contractStore.SignerByPublicToken(r.Context(), chi.URLParam(r, "token"))
	if err != nil {
		http.Error(w, "Link de assinatura inválido.", http.StatusNotFound)
		return
	}
	path, err := a.proposalPublicPathByContract(r.Context(), access.Contract.ID)
	if err != nil {
		http.Error(w, "Proposta não encontrada.", http.StatusNotFound)
		return
	}
	http.Redirect(w, r, path+"?view=proposal", http.StatusSeeOther)
}

func (a *App) proposalFromActivation(w http.ResponseWriter, r *http.Request) {
	access, err := a.activationStore.AccessByToken(r.Context(), hashToken(chi.URLParam(r, "token")))
	if err != nil {
		http.Error(w, "Link de ativação inválido ou expirado.", http.StatusGone)
		return
	}
	path, err := a.proposalPublicPathByContract(r.Context(), access.Profile.ContractID)
	if err != nil {
		http.Error(w, "Proposta não encontrada.", http.StatusNotFound)
		return
	}
	http.Redirect(w, r, path+"?view=proposal", http.StatusSeeOther)
}
