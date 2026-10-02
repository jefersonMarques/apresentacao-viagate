package httpapp

import (
	"strings"
	"testing"

	"github.com/jefersonMarques/apresentacao-viagate/internal/config"
)

func TestPrepareProposalPDFDocumentUsesCommercialAssetRoutes(t *testing.T) {
	app := &App{cfg: config.Config{BaseURL: "https://viagate.com.br"}}
	document := app.prepareProposalPDFDocument("<html><head></head><body><main>Proposta</main></body></html>")

	for _, expected := range []string{
		`<base href="https://viagate.com.br/"/>`,
		`/commercial-assets/commercial-pdf.css`,
		`/commercial-assets/commercial-pdf.js`,
	} {
		if !strings.Contains(document, expected) {
			t.Fatalf("document should contain %q", expected)
		}
	}
	if strings.Contains(document, `href="/assets/commercial-pdf.css"`) || strings.Contains(document, `src="/assets/commercial-pdf.js"`) {
		t.Fatal("document still references the legacy PDF asset route")
	}
}
