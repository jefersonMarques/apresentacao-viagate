package httpapp

import (
    "errors"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/jefersonMarques/apresentacao-viagate/internal/platform/companyregistry"
)

func TestCompanyLookupFailureStatus(t *testing.T) {
    app := &App{}
    tests := []struct {
        name     string
        err      error
        status   int
        contains string
        json     bool
    }{
        {
            name: "not found", err: companyregistry.ErrNotFound,
            status: http.StatusNotFound, contains: "não encontrado",
        },
        {
            name: "provider unavailable", err: companyregistry.ErrUnavailable,
            status: http.StatusBadGateway, contains: "indisponível",
        },
        {
            name: "public JSON unavailable", err: errors.New("network failure"),
            status: http.StatusBadGateway, contains: "\"error\"", json: true,
        },
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            recorder := httptest.NewRecorder()
            app.respondCompanyLookupFailure(recorder, tc.err, tc.json)
            if recorder.Code != tc.status {
                t.Errorf("status %d, want %d", recorder.Code, tc.status)
            }
            if !strings.Contains(recorder.Body.String(), tc.contains) {
                t.Errorf("missing expected message: %q", recorder.Body.String())
            }
            if recorder.Header().Get("Cache-Control") != "no-store" {
                t.Error("lookup errors must not be cached")
            }
        })
    }
}
