package httpapp

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (a *App) downloadCustomerSignatureReceipt(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	access, err := a.contractStore.SignerByPublicToken(r.Context(), token)
	if err != nil || access.Signer.Status != "signed" || access.Contract.Status != "signed" {
		http.Error(w, "Comprovante de assinatura indisponível.", http.StatusNotFound)
		return
	}

	pdf, err := a.contractFinalizer.CustomerSignatureReceipt(r.Context(), access.Contract.ID)
	if err != nil {
		a.logger.Error("render customer signature receipt failed", "contract_id", access.Contract.ID, "error", err)
		http.Error(w, "Não foi possível preparar o comprovante de assinatura.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=\"comprovante-assinatura-viagate.pdf\"")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdf)
}
