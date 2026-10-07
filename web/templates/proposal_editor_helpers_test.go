package templates

import (
	"testing"

	"github.com/jefersonMarques/apresentacao-viagate/internal/catalog"
	"github.com/jefersonMarques/apresentacao-viagate/internal/proposals"
)

func TestConditionCheckedDefaultsOnlyOnNewProposal(t *testing.T) {
	newProposal := proposals.EditorInput{}
	if !conditionChecked(newProposal, "Condição padrão") {
		t.Fatal("new proposal should start with managed conditions selected")
	}

	existing := proposals.EditorInput{ProposalID: "proposal-id"}
	if conditionChecked(existing, "Condição padrão") {
		t.Fatal("existing proposal with no saved conditions must remain empty")
	}

	existing.Conditions = []string{"Condição padrão"}
	if !conditionChecked(existing, "Condição padrão") {
		t.Fatal("saved condition should remain selected")
	}
}

func TestProposalCustomConditionsUsesManagedConditionList(t *testing.T) {
	input := proposals.EditorInput{
		Conditions: []string{"Condição gerenciada", "Condição negociada"},
		Content: map[string]any{
			"__ui_proposal_conditions": []catalog.Condition{
				{ID: "managed", Text: "Condição gerenciada", IsActive: true},
			},
		},
	}
	if got := ProposalCustomConditions(input); got != "Condição negociada" {
		t.Fatalf("unexpected custom conditions: %q", got)
	}
}

func TestProposalConditionGroupSelected(t *testing.T) {
	condition := catalog.Condition{Groups: []string{"score", "logistics"}}
	if !ProposalConditionGroupSelected(condition, "score") {
		t.Fatal("score group should be selected")
	}
	if ProposalConditionGroupSelected(condition, "monitoring") {
		t.Fatal("unrelated group should not be selected")
	}
}
