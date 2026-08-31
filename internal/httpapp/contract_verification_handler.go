package httpapp

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/contracts"
	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

func (a *App) contractVerificationPage(w http.ResponseWriter, r *http.Request) {
	token := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "token")))
	if len(token) != 64 {
		http.Error(w, "Código de verificação inválido.", http.StatusNotFound)
		return
	}

	record, err := a.contractStore.VerificationByToken(r.Context(), token)
	if err != nil {
		http.Error(w, "Documento não encontrado para este código de verificação.", http.StatusNotFound)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	render(r.Context(), w, http.StatusOK, templates.ContractVerificationPage(record, contracts.VerificationCode(token)))
}
