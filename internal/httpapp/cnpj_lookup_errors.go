package httpapp

import (
    "encoding/json"
    "errors"
    "net/http"

    "github.com/jefersonMarques/apresentacao-viagate/internal/platform/companyregistry"
)

func (a *App) respondCompanyLookupFailure(
    w http.ResponseWriter,
    err error,
    jsonResponse bool,
) {
    status := http.StatusBadGateway
    message := "Serviço de consulta de CNPJ indisponível. Tente novamente."

    if errors.Is(err, companyregistry.ErrNotFound) {
        status = http.StatusNotFound
        message = "CNPJ não encontrado nas bases consultadas."
    }

    if a.logger != nil {
        a.logger.Warn("company registry lookup failed", "error", err, "status", status)
    }

    w.Header().Set("Cache-Control", "no-store")
    if jsonResponse {
        w.Header().Set("Content-Type", "application/json; charset=utf-8")
        w.WriteHeader(status)
        _ = json.NewEncoder(w).Encode(map[string]string{"error": message})
        return
    }

    http.Error(w, message, status)
}
