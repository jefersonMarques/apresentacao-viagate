package httpapp

import (
	"fmt"
	"net/http"
	"strings"
)

func (a *App) previewContractTemplate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Dados inválidos.", http.StatusBadRequest)
		return
	}
	markdown := strings.TrimSpace(r.FormValue("markdown"))
	if markdown == "" {
		http.Error(w, "Informe o conteúdo do modelo.", http.StatusBadRequest)
		return
	}
	pdf, err := a.contractGenerator.PreviewTemplate(r.Context(), markdown)
	if err != nil {
		http.Error(w, "O modelo não pôde ser renderizado: "+err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `inline; filename="preview-contrato-viagate.pdf"`)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdf)))
	_, _ = w.Write(pdf)
}
