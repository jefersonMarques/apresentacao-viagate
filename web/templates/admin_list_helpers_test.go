package templates

import (
	"testing"

	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
)

func TestProposalRegularCountExcludesDefaultTemplates(t *testing.T) {
	items := []domain.Proposal{
		{ID: "default-1", IsDefault: true},
		{ID: "proposal-1"},
		{ID: "default-2", IsDefault: true},
		{ID: "proposal-2"},
	}

	if got := ProposalRegularCount(items); got != 2 {
		t.Fatalf("regular proposal count = %d, want 2", got)
	}
}
