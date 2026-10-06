package templates

import (
	"strings"
	"testing"

	"github.com/jefersonMarques/apresentacao-viagate/internal/proposals"
)

func TestProposalSolutionsPreferPublishedSnapshotDescriptions(t *testing.T) {
	proposal := proposals.PublicProposal{
		Items: []proposals.Item{
			{
				CategoryCode:     "custom-category",
				GroupName:        "Gestão de risco",
				GroupDescription: "Descrição congelada na proposta publicada.",
				Label:            "Produto A",
			},
			{
				CategoryCode:     "custom-category",
				GroupName:        "Gestão de risco",
				GroupDescription: "Descrição alterada não deve criar outro grupo.",
				Label:            "Produto B",
				IsOptional:       true,
			},
		},
	}

	solutions := ProposalSolutions(proposal)
	if len(solutions) != 1 {
		t.Fatalf("expected one solution group, got %d", len(solutions))
	}
	if solutions[0].Summary != "Descrição congelada na proposta publicada." {
		t.Fatalf("unexpected snapshot summary: %q", solutions[0].Summary)
	}
	if solutions[0].Status != "Incluído + opcional" {
		t.Fatalf("unexpected mixed status: %q", solutions[0].Status)
	}
}

func TestProposalProductPagesChunkDetailWithoutDroppingItems(t *testing.T) {
	proposal := proposals.PublicProposal{}
	for i := 0; i < 7; i++ {
		proposal.Items = append(proposal.Items, proposals.Item{Label: string(rune('A' + i))})
	}

	pages := ProposalProductPages(proposal)
	if len(pages) != 2 {
		t.Fatalf("expected two detail pages, got %d", len(pages))
	}
	if len(pages[0].Items) != 6 || len(pages[1].Items) != 1 {
		t.Fatalf("unexpected page sizes: %d and %d", len(pages[0].Items), len(pages[1].Items))
	}
}

func TestProposalExecutiveSummaryReflectsIncludedAndOptionalItems(t *testing.T) {
	proposal := proposals.PublicProposal{
		ClientTradeName: "Cliente Teste",
		Items: []proposals.Item{
			{GroupName: "Cargo Score", Label: "Produto 1"},
			{GroupName: "Cargo Score", Label: "Produto 2", IsOptional: true},
		},
	}

	summary := ProposalExecutiveSummary(proposal)
	if !strings.Contains(summary, "Cliente Teste") || !strings.Contains(summary, "1 produto(s) ou serviço(s) incluído(s)") || !strings.Contains(summary, "1 opção(ões) adicional(is)") {
		t.Fatalf("unexpected executive summary: %q", summary)
	}
}

func TestProposalJourneyReflectsPolicyRequirement(t *testing.T) {
	withPolicy := ProposalJourneySteps(proposals.PublicProposal{RequiresPolicy: true})
	withoutPolicy := ProposalJourneySteps(proposals.PublicProposal{RequiresPolicy: false})

	if !strings.Contains(withPolicy[3].Summary, "apólice") || !strings.Contains(withPolicy[3].Summary, "mercadorias") {
		t.Fatalf("policy journey should mention policy requirements: %q", withPolicy[3].Summary)
	}
	if strings.Contains(withoutPolicy[3].Summary, "apólice") || strings.Contains(withoutPolicy[3].Summary, "mercadorias") {
		t.Fatalf("journey without policy must not request policy data: %q", withoutPolicy[3].Summary)
	}
}

func TestProposalDifferentialsFollowSelectedModules(t *testing.T) {
	proposal := proposals.PublicProposal{
		Items: []proposals.Item{
			{CategoryCode: "logistics", GroupName: "Cargo Truck"},
		},
	}
	differentials := ProposalDifferentials(proposal)

	found := false
	for _, differential := range differentials {
		if differential.Title == "Acompanhamento operacional" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("logistics proposal should include operational tracking differential")
	}
}
