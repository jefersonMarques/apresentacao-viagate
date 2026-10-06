package templates

import (
	"fmt"
	"strings"

	"github.com/jefersonMarques/apresentacao-viagate/internal/catalog"
	"github.com/jefersonMarques/apresentacao-viagate/internal/proposals"
)

type ProposalSolution struct {
	Title   string
	Summary string
	Status  string
}

type ProposalProductPage struct {
	Number int
	Items  []proposals.Item
}

type ProposalDifferential struct {
	Title   string
	Summary string
}

type ProposalJourneyStep struct {
	Number  string
	Title   string
	Summary string
}

type ProposalPriceGroup struct {
	Name        string
	Items       []proposals.Item
	AllOptional bool
}

func ProposalSolutions(proposal proposals.PublicProposal) []ProposalSolution {
	seen := map[string]*ProposalSolution{}
	order := []string{}
	for _, item := range proposal.Items {
		key := item.GroupName
		status := ProposalItemStatus(item)
		if current, ok := seen[key]; ok {
			if current.Status != status {
				current.Status = "Incluído + opcional"
			}
			continue
		}
		title := strings.TrimSpace(item.GroupName)
		summary := strings.TrimSpace(item.GroupDescription)
		for _, group := range catalog.Groups {
			if group.ID == item.CategoryCode || group.Title == key {
				if title == "" || title == key {
					title = group.ShortTitle
				}
				if summary == "" {
					summary = group.Summary
				}
				break
			}
		}
		if title == "" {
			title = "Solução ViaGate"
		}
		if summary == "" {
			summary = "Solução selecionada e configurada para esta negociação."
		}
		seen[key] = &ProposalSolution{Title: title, Summary: summary, Status: status}
		order = append(order, key)
	}
	result := make([]ProposalSolution, 0, len(order))
	for _, key := range order {
		result = append(result, *seen[key])
	}
	return result
}

func ProposalPriceGroups(proposal proposals.PublicProposal) []ProposalPriceGroup {
	indexes := map[string]int{}
	groups := []ProposalPriceGroup{}
	for _, item := range proposal.Items {
		index, ok := indexes[item.GroupName]
		if !ok {
			index = len(groups)
			indexes[item.GroupName] = index
			groups = append(groups, ProposalPriceGroup{Name: item.GroupName, AllOptional: true})
		}
		groups[index].Items = append(groups[index].Items, item)
		if !item.IsOptional {
			groups[index].AllOptional = false
		}
	}
	return groups
}

func ProposalModelCards(value string) []catalog.PricingModel {
	result := []catalog.PricingModel{}
	for _, model := range catalog.PricingModels {
		if value == "item_and_bundle" {
			if model.ID == "per_item" || model.ID == "bundle" {
				result = append(result, model)
			}
			continue
		}
		if model.ID == value {
			result = append(result, model)
		}
	}
	return result
}

func ProposalItemStatus(value any, found ...bool) string {
	switch item := value.(type) {
	case proposals.Item:
		if item.IsOptional {
			return "Opcional"
		}
		return "Proposto"
	case proposals.EditorItem:
		exists := len(found) > 0 && found[0]
		if !exists {
			return "off"
		}
		if item.IsOptional {
			return "optional"
		}
		return "included"
	default:
		return ""
	}
}

func ProposalModelLabel(value string) string {
	for _, model := range catalog.PricingModels {
		if model.ID == value {
			return model.Title
		}
	}
	return value
}

func ProposalSalesperson(proposal proposals.PublicProposal, key string) string {
	group, ok := proposal.Content["salesperson"].(map[string]any)
	if !ok {
		return ""
	}
	value, _ := group[key].(string)
	return strings.TrimSpace(value)
}

func ProposalSalespersonRole(proposal proposals.PublicProposal) string {
	if value := ProposalSalesperson(proposal, "role"); value != "" {
		return value
	}
	return ProposalSalesperson(proposal, "job_title")
}

func ProposalContentString(proposal proposals.PublicProposal, key string) string {
	value, _ := proposal.Content[key].(string)
	return strings.TrimSpace(value)
}

func ProposalContentStrings(proposal proposals.PublicProposal, key string) []string {
	values, ok := proposal.Content[key].([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
			result = append(result, strings.TrimSpace(text))
		}
	}
	return result
}

func ProposalSectionString(proposal proposals.PublicProposal, section, key string) string {
	group, ok := proposal.Content[section].(map[string]any)
	if !ok {
		return ""
	}
	value, _ := group[key].(string)
	return strings.TrimSpace(value)
}

func ProposalClientDisplayName(value any) string {
	switch proposal := value.(type) {
	case proposals.PublicProposal:
		if strings.TrimSpace(proposal.ClientTradeName) != "" {
			return proposal.ClientTradeName
		}
		return proposal.ClientName
	case proposals.EditorInput:
		if strings.TrimSpace(proposal.ClientTradeName) != "" {
			return proposal.ClientTradeName
		}
		if strings.TrimSpace(proposal.ClientLegalName) != "" {
			return proposal.ClientLegalName
		}
		return "Novo cliente"
	default:
		return ""
	}
}


func ProposalIncludedCount(proposal proposals.PublicProposal) int {
	count := 0
	for _, item := range proposal.Items {
		if !item.IsOptional {
			count++
		}
	}
	return count
}

func ProposalOptionalCount(proposal proposals.PublicProposal) int {
	count := 0
	for _, item := range proposal.Items {
		if item.IsOptional {
			count++
		}
	}
	return count
}

func ProposalExecutiveSummary(proposal proposals.PublicProposal) string {
	solutions := ProposalSolutions(proposal)
	included := ProposalIncludedCount(proposal)
	optional := ProposalOptionalCount(proposal)
	client := ProposalClientDisplayName(proposal)

	if len(solutions) == 0 {
		return fmt.Sprintf("Esta proposta foi preparada para %s com condições comerciais e uma jornada de contratação estruturada pela ViaGate.", client)
	}

	if optional > 0 {
		return fmt.Sprintf(
			"Esta proposta foi estruturada para %s com %d frente(s) de solução, %d produto(s) ou serviço(s) incluído(s) e %d opção(ões) adicional(is), preservando flexibilidade para a evolução da operação.",
			client,
			len(solutions),
			included,
			optional,
		)
	}
	return fmt.Sprintf(
		"Esta proposta foi estruturada para %s com %d frente(s) de solução e %d produto(s) ou serviço(s) incluído(s), reunindo em uma única jornada o escopo comercial, a contratação e a preparação da implantação.",
		client,
		len(solutions),
		included,
	)
}

func ProposalProductPages(proposal proposals.PublicProposal) []ProposalProductPage {
	const pageSize = 6
	if len(proposal.Items) == 0 {
		return nil
	}
	pages := make([]ProposalProductPage, 0, (len(proposal.Items)+pageSize-1)/pageSize)
	for start := 0; start < len(proposal.Items); start += pageSize {
		end := start + pageSize
		if end > len(proposal.Items) {
			end = len(proposal.Items)
		}
		items := make([]proposals.Item, end-start)
		copy(items, proposal.Items[start:end])
		pages = append(pages, ProposalProductPage{
			Number: len(pages) + 1,
			Items:  items,
		})
	}
	return pages
}

func ProposalDifferentials(proposal proposals.PublicProposal) []ProposalDifferential {
	selected := map[string]bool{}
	for _, item := range proposal.Items {
		selected[item.CategoryCode] = true
	}

	result := []ProposalDifferential{}
	add := func(title, summary string) {
		result = append(result, ProposalDifferential{Title: title, Summary: summary})
	}

	if selected["score"] {
		add("Risco cadastral em uma única jornada", "Cadastro e consulta de motoristas, veículos e colaboradores com validações estruturadas para apoiar a tomada de decisão.")
		add("Biometria com prova de vida", "Fluxo digital de identificação integrado ao processo cadastral, reduzindo etapas manuais e dependência de cópias de documentos.")
	}
	if selected["authentication"] {
		add("Consultas complementares", "Camada adicional de autenticação e consultas para aprofundar a análise conforme a necessidade da operação.")
	}
	if selected["logistics"] {
		add("Acompanhamento operacional", "Recursos para coletas, entregas, eventos de parada e acompanhamento de viagens pelo smartphone do motorista.")
	}
	if selected["prevention"] {
		add("Gestão preventiva", "Ferramentas complementares para multas, débitos, restrições e histórico veicular, quando contratadas.")
	}
	if selected["monitoring"] {
		add("Integração com monitoramento", "Opções para integrar o acompanhamento de veículos e viagens às rotinas operacionais já utilizadas pela empresa.")
	}

	add("Composição modular", "Produtos principais e opcionais podem ser combinados conforme a necessidade comercial e operacional de cada cliente.")
	add("Implantação acompanhada", "A contratação segue uma jornada orientada, com aceite, contrato, assinatura e preparação da implantação em etapas claras.")

	if len(result) > 6 {
		result = result[:6]
	}
	return result
}

func ProposalJourneySteps(proposal proposals.PublicProposal) []ProposalJourneyStep {
	implementationSummary := "Cadastro dos responsáveis, usuários e informações necessárias para configuração da operação."
	if proposal.RequiresPolicy {
		implementationSummary = "Cadastro dos responsáveis e usuários, envio da apólice e informação das mercadorias transportadas para preparação da operação."
	}

	return []ProposalJourneyStep{
		{Number: "01", Title: "Aceite comercial", Summary: "O cliente confirma formalmente esta versão da proposta e inicia a jornada de contratação."},
		{Number: "02", Title: "Dados contratuais", Summary: "São confirmados os dados cadastrais e os responsáveis necessários para geração do contrato."},
		{Number: "03", Title: "Assinatura digital", Summary: "O contrato segue para assinatura dos responsáveis definidos, preservando rastreabilidade e evidências."},
		{Number: "04", Title: "Preparação da implantação", Summary: implementationSummary},
		{Number: "05", Title: "Ativação", Summary: "Com as etapas obrigatórias concluídas, a equipe ViaGate realiza a configuração interna e libera a operação."},
	}
}

func ProposalPolicyRequirementLabel(proposal proposals.PublicProposal) string {
	if proposal.RequiresPolicy {
		return "Apólice e mercadorias exigidas"
	}
	return "Sem exigência de apólice"
}

func ProposalItemDescription(item proposals.Item) string {
	if value := strings.TrimSpace(item.Description); value != "" {
		return value
	}
	if value := strings.TrimSpace(item.GroupDescription); value != "" {
		return value
	}
	return "Produto ou serviço selecionado para compor o escopo desta proposta."
}
