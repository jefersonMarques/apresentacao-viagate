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
	if solutions[0].Status != "Inclui opção adicional" {
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
	if len(pages[0].Items) != 4 || len(pages[1].Items) != 3 {
		t.Fatalf("expected balanced 4/3 pages, got %d and %d", len(pages[0].Items), len(pages[1].Items))
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
	if !strings.Contains(summary, "Cliente Teste") || !strings.Contains(summary, "1 produto ou serviço contemplado") || !strings.Contains(summary, "1 opção adicional") {
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


func TestProposalPublicCopyUsesCustomerFacingTerms(t *testing.T) {
	included := proposals.Item{Label: "Cargo Score"}
	optional := proposals.Item{Label: "Histórico Veicular", IsOptional: true}

	if status := ProposalItemStatus(included); status != "Contemplado" {
		t.Fatalf("unexpected included customer-facing status: %q", status)
	}
	if status := ProposalItemStatus(optional); status != "Opcional" {
		t.Fatalf("unexpected optional customer-facing status: %q", status)
	}

	description := ProposalItemDescription(included)
	if strings.Contains(strings.ToLower(description), "selecionado") || strings.Contains(strings.ToLower(description), "snapshot") {
		t.Fatalf("fallback description exposes internal terminology: %q", description)
	}
}

func TestProposalPolicyRequirementLabelIsCustomerFacing(t *testing.T) {
	if got := ProposalPolicyRequirementLabel(proposals.PublicProposal{RequiresPolicy: true}); got != "Requer apólice e mercadorias" {
		t.Fatalf("unexpected policy label: %q", got)
	}
	if got := ProposalPolicyRequirementLabel(proposals.PublicProposal{RequiresPolicy: false}); got != "Não requer apólice" {
		t.Fatalf("unexpected no-policy label: %q", got)
	}
}


func TestProposalProductPagesBalanceElevenItems(t *testing.T) {
	proposal := proposals.PublicProposal{}
	for i := 0; i < 11; i++ {
		proposal.Items = append(proposal.Items, proposals.Item{Label: string(rune('A' + i))})
	}
	pages := ProposalProductPages(proposal)
	if len(pages) != 2 || len(pages[0].Items) != 6 || len(pages[1].Items) != 5 {
		t.Fatalf("expected balanced 6/5 pages, got %#v", []int{len(pages[0].Items), len(pages[1].Items)})
	}
}

func TestProposalItemGroupLabelMapsLegacyCommercialNames(t *testing.T) {
	item := proposals.Item{
		GroupName:   "ANALISE CADASTRAL | CONJUNTO",
		ProductCode: "score-bundle-register",
	}
	if got := ProposalItemGroupLabel(item); got != "Cargo Score" {
		t.Fatalf("unexpected commercial group name: %q", got)
	}
}

func TestProposalDifferentialsAvoidModularCopyWithoutOptions(t *testing.T) {
	proposal := proposals.PublicProposal{
		Items: []proposals.Item{
			{CategoryCode: "logistics", GroupName: "Cargo Truck"},
			{CategoryCode: "prevention", GroupName: "Prevenção"},
		},
	}
	for _, differential := range ProposalDifferentials(proposal) {
		if differential.Title == "Flexibilidade para evoluir a solução" {
			t.Fatal("proposal without optional items must not advertise optional flexibility")
		}
	}
}

func TestProposalConditionGroupsOrganizeCommercialConditions(t *testing.T) {
	groups := ProposalConditionGroups([]string{
		"O retorno ocorre em até 10 minutos.",
		"Integrações dependem de homologação.",
		"Customizações fora do escopo serão orçadas.",
		"Condição específica adicional.",
	})
	if len(groups) != 4 {
		t.Fatalf("expected four condition groups, got %d", len(groups))
	}
	got := map[string]int{}
	for _, group := range groups {
		got[group.Title] = len(group.Items)
	}
	if got["Prazos e operação"] != 1 || got["Serviços e integrações"] != 1 || got["Customizações e despesas"] != 1 || got["Condições gerais"] != 1 {
		t.Fatalf("unexpected condition grouping: %#v", got)
	}
}

func TestProposalBillingUnitLabelUsesCustomerLanguage(t *testing.T) {
	tests := map[string]string{
		"consulta":  "Por consulta",
		"viagem":    "Por viagem",
		"veículo":   "Por veículo",
		"reanálise": "Por reanálise",
	}
	for input, want := range tests {
		if got := ProposalBillingUnitLabel(input); got != want {
			t.Fatalf("unit %q: got %q want %q", input, got, want)
		}
	}
}
