package contracts

import (
	"os"
	"strings"
	"testing"
)

func loadCanonicalContractTemplate(t *testing.T) string {
	t.Helper()
	content, err := os.ReadFile("testdata/default_contract_template.md")
	if err != nil {
		t.Fatalf("read canonical contract template: %v", err)
	}
	return strings.TrimSpace(string(content))
}

func TestCanonicalContractTemplateMatchesMigration(t *testing.T) {
	expected := loadCanonicalContractTemplate(t)
	migration, err := os.ReadFile("../../migrations/000026_clean_contract_template.sql")
	if err != nil {
		t.Fatalf("read contract template migration: %v", err)
	}

	sql := string(migration)
	startToken := "$contract$\n"
	endToken := "\n$contract$::text"
	start := strings.Index(sql, startToken)
	if start < 0 {
		t.Fatal("contract template start marker not found in migration")
	}
	start += len(startToken)
	end := strings.Index(sql[start:], endToken)
	if end < 0 {
		t.Fatal("contract template end marker not found in migration")
	}
	actual := strings.TrimSpace(sql[start : start+end])
	if actual != expected {
		t.Fatal("migration contract template differs from canonical testdata")
	}
}

func TestCanonicalContractTemplateRendersMinimalProposalCleanly(t *testing.T) {
	template := loadCanonicalContractTemplate(t)
	if err := validateTemplateBooleanUsage(template); err != nil {
		t.Fatalf("canonical template boolean usage: %v", err)
	}

	data := contractValidationDataSets("Via Gateway Ltda", "39.406.000/0001-21")[1]
	renderedMarkdown, _, err := NewRenderer().Render(template, data)
	if err != nil {
		t.Fatalf("render canonical minimal template: %v", err)
	}

	for _, forbidden := range []string{
		" true",
		" false",
		"Nome fantasia:",
		"Cargo:",
		"Corretora |",
		"Produtor / Corretor responsável",
		"Valores de Cargo Truck",
		"Valores de Prevenção",
		"Valores de Monitoramento",
		"Valores de autenticação",
		"R$ 0,00",
	} {
		if strings.Contains(renderedMarkdown, forbidden) {
			t.Fatalf("minimal contract contains forbidden output %q:\n%s", forbidden, renderedMarkdown)
		}
	}

	for _, required := range []string{
		"Sem fatura mínima",
		"ISENTO",
		"Cadastro de motorista",
		"Cargo Score.",
	} {
		if !strings.Contains(renderedMarkdown, required) {
			t.Fatalf("minimal contract must contain %q:\n%s", required, renderedMarkdown)
		}
	}
}

func TestCanonicalContractTemplateOmitsCargoScoreClauseWhenNotContracted(t *testing.T) {
	template := loadCanonicalContractTemplate(t)
	data := contractPreviewData("Via Gateway Ltda", "39.406.000/0001-21")
	setPreviewValue(data, "products.cargo_score", false)

	renderedMarkdown, _, err := NewRenderer().Render(template, data)
	if err != nil {
		t.Fatalf("render canonical template without cargo score: %v", err)
	}
	if strings.Contains(renderedMarkdown, "**Cargo Score.**") {
		t.Fatal("cargo score clause must be omitted when the product is not contracted")
	}
}
