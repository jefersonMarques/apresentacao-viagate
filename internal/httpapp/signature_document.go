package httpapp

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (a *App) viewContractDocument(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	access, err := a.contractStore.SignerByPublicToken(r.Context(), token)
	if err != nil || access.Contract.PDFStorageKey == "" {
		http.Error(w, "Documento indisponível.", http.StatusNotFound)
		return
	}

	reader, _, _, err := a.storage.Get(r.Context(), access.Contract.PDFStorageKey)
	if err != nil {
		a.logger.Error("load contract PDF for signature viewer failed", "error", err, "contract_id", access.Contract.ID)
		http.Error(w, "Documento indisponível.", http.StatusInternalServerError)
		return
	}
	defer reader.Close()
	content, err := io.ReadAll(reader)
	if err != nil {
		http.Error(w, "Documento indisponível.", http.StatusInternalServerError)
		return
	}
	hash := sha256.Sum256(content)
	if !bytes.Equal(hash[:], access.Contract.DocumentSHA256) {
		a.logger.Error("contract PDF hash mismatch in signature viewer", "contract_id", access.Contract.ID)
		http.Error(w, "A integridade do documento não pôde ser confirmada.", http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="contrato-viagate.pdf"`)
	w.Header().Set("Cache-Control", "private, no-store, max-age=0")
	w.Header().Set("X-Frame-Options", "SAMEORIGIN")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'self'; sandbox")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
	_, _ = w.Write(content)
}
