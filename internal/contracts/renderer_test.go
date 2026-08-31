package contracts

import (
	"strings"
	"testing"
)

func TestRendererTreatsVariablesAsLiteralText(t *testing.T) {
	renderer := NewRenderer()
	markdown := "# Contrato\n\nContratante: {client.legal_name}\n"
	data := Data{"client": map[string]any{"legal_name": "Empresa **NÃO ALTERAR** <script>alert(1)</script>"}}

	renderedMarkdown, renderedHTML, err := renderer.Render(markdown, data)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if strings.Contains(renderedHTML, "<script>") {
		t.Fatal("untrusted HTML must not be rendered")
	}
	if strings.Contains(renderedHTML, "<strong>NÃO ALTERAR</strong>") {
		t.Fatal("variable Markdown must not change document structure")
	}
	if !strings.Contains(renderedMarkdown, "\\*\\*NÃO ALTERAR\\*\\*") {
		t.Fatalf("expected Markdown markers to be escaped: %s", renderedMarkdown)
	}
}

func TestRendererKeepsTemplateConditionals(t *testing.T) {
	renderer := NewRenderer()
	markdown := "{% if products.cargo_score %}Cargo Score incluído{% endif %}{% if products.monitoring %}Monitoramento incluído{% endif %}"
	data := Data{"products": map[string]any{"cargo_score": true, "monitoring": false}}
	_, renderedHTML, err := renderer.Render(markdown, data)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if !strings.Contains(renderedHTML, "Cargo Score incluído") {
		t.Fatal("expected true conditional block")
	}
	if strings.Contains(renderedHTML, "Monitoramento incluído") {
		t.Fatal("false conditional block must be removed")
	}
}

func TestRendererAllowsTrustedPricingTableMarkdown(t *testing.T) {
	renderer := NewRenderer()
	markdown := "## Valores\n\n{proposal.pricing_table}\n"
	data := Data{
		"proposal": map[string]any{
			"pricing_table": RawMarkdown("| Serviço | Valor |\n|---|---:|\n| Consulta | R$ 4,00 |"),
		},
	}

	renderedMarkdown, renderedHTML, err := renderer.Render(markdown, data)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if !strings.Contains(renderedMarkdown, "| Consulta | R$ 4,00 |") {
		t.Fatalf("expected pricing table Markdown: %s", renderedMarkdown)
	}
	if !strings.Contains(renderedHTML, "<table>") {
		t.Fatalf("expected pricing table HTML: %s", renderedHTML)
	}
	if !strings.Contains(renderedHTML, "R$ 4,00") {
		t.Fatalf("expected proposal price in rendered HTML: %s", renderedHTML)
	}
}
