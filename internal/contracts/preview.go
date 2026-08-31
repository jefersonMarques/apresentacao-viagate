package contracts

import (
	"context"
	"strings"
)

func (g *Generator) PreviewTemplate(ctx context.Context, markdown string) ([]byte, error) {
	markdown = ensureProposalFinancialTerms(markdown)
	_, renderedHTML, err := g.renderer.Render(markdown, contractPreviewData(g.company.LegalName, g.company.CNPJ))
	if err != nil {
		return nil, err
	}
	return g.pdf.Render(ctx, renderedHTML)
}

func (g *Generator) ValidateTemplate(ctx context.Context, markdown string) error {
	_, err := g.PreviewTemplate(ctx, markdown)
	return err
}

func contractPreviewData(companyName, companyCNPJ string) Data {
	data := Data{}
	for _, variable := range AllowedVariables() {
		setPreviewValue(data, variable, previewValue(variable))
	}
	setPreviewValue(data, "client.legal_name", "Transportadora Exemplo Ltda.")
	setPreviewValue(data, "client.trade_name", "Transportadora Exemplo")
	setPreviewValue(data, "client.cnpj", "12.345.678/0001-90")
	setPreviewValue(data, "client.address", "Rua Exemplo, 100, Centro, Curitiba, PR, 80000-000")
	setPreviewValue(data, "client.city", "Curitiba")
	setPreviewValue(data, "client.state", "PR")
	setPreviewValue(data, "representative.name", "Maria da Silva")
	setPreviewValue(data, "representative.cpf", "123.456.789-00")
	setPreviewValue(data, "representative.email", "maria@cliente.exemplo")
	setPreviewValue(data, "representative.phone", "(41) 99999-9999")
	setPreviewValue(data, "representative.role", "Diretora")
	setPreviewValue(data, "proposal.pricing_model", "Item + conjunto")
	setPreviewValue(data, "proposal.pricing_table", RawMarkdown("| Serviço | Valor |\n| --- | ---: |\n| Cargo Score | R$ 1,50 |\n| Cargo Truck | R$ 2,50 |"))
	setPreviewValue(data, "proposal.minimum_invoice", "R$ 500,00")
	setPreviewValue(data, "proposal.setup_fee", "R$ 1.000,00")
	setPreviewValue(data, "proposal.accepted_at", "31/08/2026 15:00")
	setPreviewValue(data, "proposal.valid_until", "15/09/2026")
	setPreviewValue(data, "operation.type", "Normal")
	setPreviewValue(data, "insurance.insurer", "Seguradora Exemplo")
	setPreviewValue(data, "insurance.policy_start_date", "01/09/2026")
	setPreviewValue(data, "insurance.policy_end_date", "31/08/2027")
	setPreviewValue(data, "insurance.broker_company", "Corretora Exemplo")
	setPreviewValue(data, "insurance.broker_producer", "Produtor Exemplo")
	if strings.TrimSpace(companyName) == "" {
		companyName = "ViaGate Tecnologia e Gerenciamento de Riscos"
	}
	if strings.TrimSpace(companyCNPJ) == "" {
		companyCNPJ = "00.000.000/0001-00"
	}
	setPreviewValue(data, "viagate.legal_name", companyName)
	setPreviewValue(data, "viagate.cnpj", companyCNPJ)
	return data
}

func previewValue(variable string) any {
	if strings.HasPrefix(variable, "products.") {
		return true
	}
	if strings.HasPrefix(variable, "pricing.") {
		return "R$ 1,00"
	}
	return "Exemplo"
}

func setPreviewValue(data Data, path string, value any) {
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return
	}
	current := map[string]any(data)
	for _, part := range parts[:len(parts)-1] {
		next, ok := current[part].(map[string]any)
		if !ok {
			next = map[string]any{}
			current[part] = next
		}
		current = next
	}
	current[parts[len(parts)-1]] = value
}
