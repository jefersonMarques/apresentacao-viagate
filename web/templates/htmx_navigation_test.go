package templates

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
)

func TestPipelineNavigationUsesHTMXWithHTMLFallback(t *testing.T) {
	item := domain.PipelineItem{
		ProposalID:    "proposal-123",
		ClientName:    "Sample Customer",
		ProposalTitle: "Sample Proposal",
	}

	var output bytes.Buffer

	err := PipelinePage(domain.User{}, []domain.PipelineItem{item}, false).Render(context.Background(), &output)
	if err != nil {
		t.Fatalf("render pipeline page: %v", err)
	}

	markup := output.String()
	requiredAttributes := []string{
		`id="adminContent"`,
		`src="/commercial-assets/vendor/htmx.min.js"`,
		`src="/commercial-assets/htmx-config.js"`,
		`href="/admin/pipeline/proposal-123"`,
		`hx-get="/admin/pipeline/proposal-123"`,
		`hx-target="#adminContent"`,
		`hx-select="#adminContent"`,
		`hx-push-url="true"`,
	}

	for _, attribute := range requiredAttributes {
		if !strings.Contains(markup, attribute) {
			t.Errorf("expected pipeline markup to include %q", attribute)
		}
	}
}
