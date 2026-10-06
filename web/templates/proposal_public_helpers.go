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

type ProposalConditionGroup struct {
	Title string
	Items []string
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
		key := ProposalItemGroupLabel(item)
		status := ProposalItemStatus(item)
		if current, ok := seen[key]; ok {
			if current.Status != status {
				current.Status = "Inclui opção adicional"
			}
			continue
		}
		title := key
		summary := strings.TrimSpace(item.GroupDescription)
		for _, group := range catalog.Groups {
			if group.ID == item.CategoryCode || group.ShortTitle == title || group.Title == item.GroupName {
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
			summary = "Solução contemplada nesta proposta, organizada para apoiar a operação apresentada."
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
		groupName := ProposalItemGroupLabel(item)
		index, ok := indexes[groupName]
		if !ok {
			index = len(groups)
			indexes[groupName] = index
			groups = append(groups, ProposalPriceGroup{Name: groupName, AllOptional: true})
		}
		groups[index].Items = append(groups[index].Items, item)
		if !item.IsOptional {
			groups[index].AllOptional = false
		}
	}
	return groups
}

func ProposalItemGroupLabel(item proposals.Item) string {
	if group, _, ok := catalog.ItemByID(item.ProductCode); ok {
		return group.ShortTitle
	}
	for _, group := range catalog.Groups {
		if group.ID == item.CategoryCode {
			return group.ShortTitle
		}
	}

	groupName := strings.TrimSpace(item.GroupName)
	normalized := strings.ToLower(groupName + " " + item.ProductCode)
	switch {
	case strings.Contains(normalized, "score") || strings.Contains(normalized, "analise cadastral") || strings.Contains(normalized, "análise cadastral"):
		return "Cargo Score"
	case strings.Contains(normalized, "truck") || strings.Contains(normalized, "logistica") || strings.Contains(normalized, "logística"):
		return "Cargo Truck"
	case strings.Contains(normalized, "auth") || strings.Contains(normalized, "autentic"):
		return "Consultas e autenticação"
	case strings.Contains(normalized, "preven"):
		return "Prevenção"
	case strings.Contains(normalized, "monitor"):
		return "Monitoramento de veículos"
	default:
		return groupName
	}
}

func ProposalItemScopeLabel(item proposals.Item) string {
	if item.IsOptional {
		return "Opcional"
	}
	return "Principal"
}

func ProposalDifferentialGridClass(items []ProposalDifferential) string {
	if len(items) == 4 {
		return "is-four"
	}
	return ""
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
		return "Contemplado"
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

	solutionLabel := proposalCountLabel(len(solutions), "solução", "soluções")
	includedLabel := proposalCountLabel(included, "produto ou serviço contemplado", "produtos e serviços contemplados")
	if optional > 0 {
		optionalLabel := proposalCountLabel(optional, "opção adicional", "opções adicionais")
		return fmt.Sprintf(
			"Esta proposta foi preparada para %s com %s, reunindo %s no escopo principal e %s para ampliar a solução conforme a necessidade da operação.",
			client,
			solutionLabel,
			includedLabel,
			optionalLabel,
		)
	}
	return fmt.Sprintf(
		"Esta proposta foi preparada para %s com %s e %s, reunindo em uma única jornada o escopo comercial, a contratação e a preparação da implantação.",
		client,
		solutionLabel,
		includedLabel,
	)
}

func ProposalProductPages(proposal proposals.PublicProposal) []ProposalProductPage {
	const maxItemsPerPage = 6
	total := len(proposal.Items)
	if total == 0 {
		return nil
	}

	pageCount := (total + maxItemsPerPage - 1) / maxItemsPerPage
	baseSize := total / pageCount
	remainder := total % pageCount

	pages := make([]ProposalProductPage, 0, pageCount)
	start := 0
	for pageIndex := 0; pageIndex < pageCount; pageIndex++ {
		size := baseSize
		if pageIndex < remainder {
			size++
		}
		end := start + size
		items := make([]proposals.Item, size)
		copy(items, proposal.Items[start:end])
		pages = append(pages, ProposalProductPage{
			Number: pageIndex + 1,
			Items:  items,
		})
		start = end
	}
	return pages
}

func ProposalDifferentials(proposal proposals.PublicProposal) []ProposalDifferential {
	selected := map[string]bool{}
	for _, item := range proposal.Items {
		category := strings.ToLower(strings.TrimSpace(item.CategoryCode))
		product := strings.ToLower(strings.TrimSpace(item.ProductCode))
		group := strings.ToLower(strings.TrimSpace(item.GroupName))

		if category != "" {
			selected[category] = true
		}
		switch {
		case strings.Contains(product, "score") || strings.Contains(group, "analise cadastral") || strings.Contains(group, "análise cadastral"):
			selected["score"] = true
		case strings.Contains(product, "auth") || strings.Contains(group, "autentic"):
			selected["authentication"] = true
		case strings.Contains(product, "truck") || strings.Contains(group, "logistica") || strings.Contains(group, "logística"):
			selected["logistics"] = true
		case strings.Contains(product, "prevention") || strings.Contains(group, "preven"):
			selected["prevention"] = true
		case strings.Contains(product, "monitoring") || strings.Contains(group, "monitoramento"):
			selected["monitoring"] = true
		}
	}

	result := []ProposalDifferential{}
	add := func(title, summary string) {
		result = append(result, ProposalDifferential{Title: title, Summary: summary})
	}

	if selected["score"] {
		add("Análise de risco mais estruturada", "Consultas cadastrais de motoristas e veículos organizadas para apoiar decisões com mais agilidade e consistência.")
		add("Biometria integrada ao processo", "Validação biométrica incorporada à jornada cadastral, reduzindo etapas manuais e apoiando a identificação do profissional.")
	}
	if selected["authentication"] {
		add("Consultas complementares", "Recursos adicionais de autenticação e pesquisa para aprofundar a análise conforme a necessidade da operação.")
	}
	if selected["logistics"] {
		add("Acompanhamento operacional", "Recursos para coletas, entregas, eventos de parada e acompanhamento de viagens pelo smartphone do motorista.")
	}
	if selected["prevention"] {
		add("Gestão preventiva", "Ferramentas complementares para acompanhar multas, débitos, restrições e histórico veicular quando contratadas.")
	}
	if selected["monitoring"] {
		add("Integração com monitoramento", "Opções para conectar o acompanhamento de veículos e viagens às rotinas operacionais da empresa.")
	}

	if ProposalOptionalCount(proposal) > 0 {
		add("Flexibilidade para evoluir a solução", "As opções adicionais podem ampliar o escopo conforme novas necessidades comerciais e operacionais surgirem.")
	} else if len(ProposalSolutions(proposal)) > 1 {
		add("Visão integrada da operação", "As soluções desta proposta se complementam em uma jornada única, reduzindo dispersão entre etapas e fornecedores.")
	}
	add("Implantação acompanhada", "A contratação segue uma jornada orientada, com aceite, contrato, assinatura e preparação da implantação em etapas claras.")

	if len(result) > 6 {
		result = result[:6]
	}
	return result
}

func ProposalHasOptionalItems(proposal proposals.PublicProposal) bool {
	return ProposalOptionalCount(proposal) > 0
}

func ProposalBillingUnitLabel(unit string) string {
	value := strings.TrimSpace(strings.ToLower(unit))
	if value == "" {
		return "Conforme utilização"
	}
	switch value {
	case "cadastro":
		return "Por cadastro"
	case "consulta":
		return "Por consulta"
	case "reanálise", "reanalise":
		return "Por reanálise"
	case "viagem":
		return "Por viagem"
	case "veículo", "veiculo":
		return "Por veículo"
	case "estado":
		return "Por estado"
	case "conjunto":
		return "Por conjunto"
	default:
		return "Por " + value
	}
}

func ProposalConditionGroups(conditions []string) []ProposalConditionGroup {
	if len(conditions) == 0 {
		return nil
	}

	type groupRule struct {
		title    string
		keywords []string
	}
	rules := []groupRule{
		{
			title:    "Prazos e operação",
			keywords: []string{"prazo", "retorno", "biometr", "operaç", "operac", "implant", "atendimento"},
		},
		{
			title:    "Serviços e integrações",
			keywords: []string{"integra", "aplicativo", "link web", "vitimologia", "logíst", "logist", "gestão de risco", "gerenciamento de risco"},
		},
		{
			title:    "Customizações e despesas",
			keywords: []string{"customiza", "fora do escopo", "hora técnica", "hora tecnica", "despesa", "deslocamento", "alimentação", "alimentacao", "hospedagem", "orçad", "orcad"},
		},
	}

	groups := make([]ProposalConditionGroup, 0, len(rules)+1)
	indexes := map[string]int{}
	add := func(title, item string) {
		index, ok := indexes[title]
		if !ok {
			index = len(groups)
			indexes[title] = index
			groups = append(groups, ProposalConditionGroup{Title: title})
		}
		groups[index].Items = append(groups[index].Items, item)
	}

	for _, condition := range conditions {
		text := strings.TrimSpace(condition)
		if text == "" {
			continue
		}
		normalized := strings.ToLower(text)
		title := "Condições gerais"
		for _, rule := range rules {
			matched := false
			for _, keyword := range rule.keywords {
				if strings.Contains(normalized, keyword) {
					title = rule.title
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
		add(title, text)
	}
	return groups
}

func ProposalClientNameClass(proposal proposals.PublicProposal) string {
	length := len([]rune(ProposalClientDisplayName(proposal)))
	switch {
	case length > 46:
		return "proposal-client-name proposal-client-name-xlong"
	case length > 30:
		return "proposal-client-name proposal-client-name-long"
	default:
		return "proposal-client-name"
	}
}

func ProposalJourneySteps(proposal proposals.PublicProposal) []ProposalJourneyStep {
	implementationSummary := "Cadastro dos responsáveis, usuários e informações necessárias para configuração da operação."
	if proposal.RequiresPolicy {
		implementationSummary = "Cadastro dos responsáveis e usuários, envio da apólice e informação das mercadorias transportadas para preparação da operação."
	}

	return []ProposalJourneyStep{
		{Number: "01", Title: "Aceite comercial", Summary: "A empresa confirma a proposta e inicia a jornada de contratação."},
		{Number: "02", Title: "Dados contratuais", Summary: "São confirmados os dados cadastrais e os responsáveis necessários para preparar o contrato."},
		{Number: "03", Title: "Assinatura digital", Summary: "O contrato segue para assinatura dos responsáveis, com registro e rastreabilidade de todo o processo."},
		{Number: "04", Title: "Preparação da implantação", Summary: implementationSummary},
		{Number: "05", Title: "Ativação", Summary: "Com as etapas obrigatórias concluídas, a ViaGate finaliza a configuração e libera a operação."},
	}
}

func ProposalPolicyRequirementLabel(proposal proposals.PublicProposal) string {
	if proposal.RequiresPolicy {
		return "Requer apólice e mercadorias"
	}
	return "Não requer apólice"
}

func ProposalItemDescription(item proposals.Item) string {
	if value := strings.TrimSpace(item.Description); value != "" {
		return value
	}
	if value := strings.TrimSpace(item.GroupDescription); value != "" {
		return value
	}
	return "Este produto ou serviço integra a solução comercial apresentada nesta proposta."
}


func ProposalSolutionStatusClass(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "opcional":
		return "optional"
	case "inclui opção adicional":
		return "mixed"
	default:
		return ""
	}
}

func ProposalItemStatusClass(item proposals.Item) string {
	if item.IsOptional {
		return "optional"
	}
	return ""
}


func proposalCountLabel(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}
