package templates

import (
	"sort"
	"strings"
	"time"

	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
)

type PipelineTimelineStep struct {
	Key         string
	Title       string
	Description string
	Actor       string
	OccurredAt  *time.Time
	Completed   bool
	Current     bool
	Planned     bool
	Signed      bool
}

type pipelineStepDefinition struct {
	Key         string
	Title       string
	Description string
	EventTypes  []string
}

var pipelineStepDefinitions = []pipelineStepDefinition{
	{
		Key:         "proposal_published",
		Title:       "Proposta publicada",
		Description: "A proposta comercial foi disponibilizada para análise do cliente.",
		EventTypes:  []string{"proposal.published"},
	},
	{
		Key:         "proposal_accepted",
		Title:       "Proposta aceita",
		Description: "O responsável confirmou o aceite da proposta comercial.",
		EventTypes:  []string{"proposal.accepted"},
	},
	{
		Key:         "document_uploaded",
		Title:       "Documento enviado",
		Description: "A apólice ou documento obrigatório foi anexado ao cadastro.",
		EventTypes:  []string{"document.uploaded"},
	},
	{
		Key:         "onboarding_submitted",
		Title:       "Dados da operação enviados",
		Description: "O cliente concluiu o preenchimento e enviou os dados operacionais.",
		EventTypes:  []string{"onboarding.submitted"},
	},
	{
		Key:         "onboarding_approved",
		Title:       "Cadastro aprovado",
		Description: "Os dados foram validados e o cadastro foi liberado para a contratação.",
		EventTypes:  []string{"onboarding.approved", "onboarding.auto_approved"},
	},
	{
		Key:         "contract_sent",
		Title:       "Contrato enviado para assinatura",
		Description: "O contrato foi gerado e disponibilizado ao responsável para assinatura eletrônica.",
		EventTypes:  []string{"contract.sent"},
	},
	{
		Key:         "contract_signed",
		Title:       "Contrato assinado",
		Description: "O responsável concluiu a assinatura eletrônica do contrato.",
		EventTypes:  []string{"contract.signed"},
	},
	{
		Key:         "evidence_finalized",
		Title:       "Pacote de evidências finalizado",
		Description: "O sistema consolidou o contrato assinado e a trilha final de evidências.",
		EventTypes:  []string{"contract.evidence_finalized"},
	},
}

func PipelineStage(item domain.PipelineItem) string {
	if PipelineIsSigned(item) {
		return "Contrato assinado"
	}
	if item.ContractStatus != "" {
		return "Contrato · " + humanStatus(item.ContractStatus)
	}
	if item.OnboardingStatus != "" {
		return "Cadastro · " + humanStatus(item.OnboardingStatus)
	}
	if item.ProposalStatus == "accepted" {
		return "Proposta aceita"
	}
	if item.ProposalStatus == "published" {
		return "Proposta enviada"
	}
	return "Proposta · " + humanStatus(item.ProposalStatus)
}

func PipelineIsSigned(item domain.PipelineItem) bool {
	return item.ContractStatus == "signed" || item.FullySignedAt != nil
}

func PipelineStageBadgeClass(item domain.PipelineItem) string {
	if PipelineIsSigned(item) {
		return "badge success pipeline-signed-badge"
	}
	return "badge"
}

func PipelineRowClass(item domain.PipelineItem) string {
	if PipelineIsSigned(item) {
		return "pipeline-row-signed"
	}
	return ""
}

func PipelineStageDescription(item domain.PipelineItem) string {
	switch {
	case PipelineIsSigned(item):
		return "A assinatura eletrônica foi concluída. A contratação está formalizada e vinculada à trilha de evidências."
	case item.ContractStatus == "sent" || item.ContractStatus == "partially_signed":
		return "O contrato já foi enviado e está aguardando a conclusão da assinatura eletrônica."
	case item.ContractStatus == "generated":
		return "O contrato foi gerado e está sendo preparado para envio ao responsável."
	case item.OnboardingStatus == "approved":
		return "O cadastro foi aprovado e a preparação do contrato é a próxima etapa."
	case item.OnboardingStatus == "under_review":
		return "Os dados enviados pelo cliente estão em revisão pela equipe ViaGate."
	case item.OnboardingStatus == "submitted":
		return "O cadastro foi enviado e aguarda validação."
	case item.OnboardingStatus != "":
		return "O cliente está concluindo os dados necessários para a contratação."
	case item.ProposalStatus == "accepted":
		return "A proposta foi aceita e o cadastro operacional é a próxima etapa."
	case item.ProposalStatus == "published":
		return "A proposta está disponível e aguarda o aceite do cliente."
	default:
		return "A negociação está em andamento."
	}
}

func PipelineSignedBy(item domain.PipelineItem) string {
	if value := strings.TrimSpace(item.SignedByName); value != "" {
		return value
	}
	if value := strings.TrimSpace(item.CustomerResponsibleName); value != "" {
		return value
	}
	return "Responsável da empresa"
}

func PipelineCommercials(items []domain.PipelineItem) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, item := range items {
		value := strings.TrimSpace(item.CommercialName)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func PipelineStages(items []domain.PipelineItem) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, item := range items {
		value := PipelineStage(item)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func PipelineSearch(item domain.PipelineItem) string {
	return strings.Join([]string{item.ClientName, item.ProposalTitle, item.CommercialName, PipelineStage(item)}, " ")
}

func PipelineTimeline(item domain.PipelineItem, events []domain.PipelineEvent) []PipelineTimelineStep {
	steps := make([]PipelineTimelineStep, 0, len(pipelineStepDefinitions))

	for _, definition := range pipelineStepDefinitions {
		event, found := findPipelineEvent(events, definition.EventTypes)
		step := PipelineTimelineStep{
			Key:         definition.Key,
			Title:       definition.Title,
			Description: definition.Description,
			Signed:      definition.Key == "contract_signed",
		}

		if found {
			occurredAt := event.CreatedAt
			step.OccurredAt = &occurredAt
			step.Actor = PipelineEventActor(event, item)
			step.Completed = true
		} else {
			step.Completed = inferPipelineStepCompletion(definition.Key, item)
			step.OccurredAt = inferPipelineStepTime(definition.Key, item)
			step.Actor = inferredPipelineActor(definition.Key, item)
		}

		steps = append(steps, step)
	}

	currentIndex := -1
	for index := range steps {
		if !steps[index].Completed {
			currentIndex = index
			break
		}
	}
	if currentIndex < 0 && len(steps) > 0 {
		currentIndex = len(steps) - 1
	}

	for index := range steps {
		steps[index].Current = index == currentIndex
		steps[index].Planned = !steps[index].Completed && !steps[index].Current
	}

	return steps
}

func PipelineTimelineStepClass(step PipelineTimelineStep) string {
	classes := []string{"pipeline-timeline-step"}
	if step.Completed {
		classes = append(classes, "completed")
	}
	if step.Current {
		classes = append(classes, "current")
	}
	if step.Planned {
		classes = append(classes, "planned")
	}
	if step.Signed && step.Completed {
		classes = append(classes, "signed")
	}
	return strings.Join(classes, " ")
}

func PipelineTimelineStatus(step PipelineTimelineStep) string {
	switch {
	case step.Signed && step.Completed:
		return "Assinado"
	case step.Current && !step.Completed:
		return "Etapa atual"
	case step.Completed:
		return "Concluído"
	default:
		return "Pendente"
	}
}

func PipelineTimelineStatusClass(step PipelineTimelineStep) string {
	switch {
	case step.Signed && step.Completed:
		return "pipeline-step-status signed"
	case step.Current:
		return "pipeline-step-status current"
	case step.Completed:
		return "pipeline-step-status completed"
	default:
		return "pipeline-step-status pending"
	}
}

func PipelineTimelineMeta(step PipelineTimelineStep) string {
	if step.OccurredAt != nil {
		when := step.OccurredAt.Format("02/01/2006 às 15:04")
		if strings.TrimSpace(step.Actor) != "" {
			return when + " · por " + step.Actor
		}
		return when
	}
	if step.Current {
		return "Aguardando conclusão desta etapa"
	}
	if step.Planned {
		return "Será realizado após as etapas anteriores"
	}
	if strings.TrimSpace(step.Actor) != "" {
		return "Concluído · por " + step.Actor
	}
	return "Concluído"
}

func findPipelineEvent(events []domain.PipelineEvent, eventTypes []string) (domain.PipelineEvent, bool) {
	for _, event := range events {
		for _, eventType := range eventTypes {
			if event.EventType == eventType {
				return event, true
			}
		}
	}
	return domain.PipelineEvent{}, false
}

func inferPipelineStepCompletion(key string, item domain.PipelineItem) bool {
	switch key {
	case "proposal_published":
		return item.ProposalStatus == "published" || item.ProposalStatus == "accepted" || item.AcceptedAt != nil
	case "proposal_accepted":
		return item.AcceptedAt != nil || item.ProposalStatus == "accepted"
	case "document_uploaded":
		return item.SubmittedAt != nil || onboardingHasReached(item.OnboardingStatus, "submitted")
	case "onboarding_submitted":
		return item.SubmittedAt != nil || onboardingHasReached(item.OnboardingStatus, "submitted")
	case "onboarding_approved":
		return item.OnboardingStatus == "approved" || item.ContractID != ""
	case "contract_sent":
		return item.ContractStatus == "sent" || item.ContractStatus == "partially_signed" || PipelineIsSigned(item)
	case "contract_signed":
		return PipelineIsSigned(item)
	case "evidence_finalized":
		return false
	default:
		return false
	}
}

func inferPipelineStepTime(key string, item domain.PipelineItem) *time.Time {
	switch key {
	case "proposal_accepted":
		return item.AcceptedAt
	case "onboarding_submitted":
		return item.SubmittedAt
	case "contract_signed":
		return item.FullySignedAt
	default:
		return nil
	}
}

func inferredPipelineActor(key string, item domain.PipelineItem) string {
	customer := strings.TrimSpace(item.CustomerResponsibleName)
	if customer == "" {
		customer = "Cliente"
	}

	switch key {
	case "proposal_published":
		if strings.TrimSpace(item.CommercialName) != "" {
			return item.CommercialName
		}
		return "Equipe ViaGate"
	case "proposal_accepted", "document_uploaded", "onboarding_submitted":
		return customer
	case "onboarding_approved", "contract_sent", "evidence_finalized":
		return "Sistema ViaGate"
	case "contract_signed":
		return PipelineSignedBy(item)
	default:
		return ""
	}
}

func onboardingHasReached(status, target string) bool {
	order := map[string]int{
		"":                     0,
		"pending":              1,
		"in_progress":          2,
		"correction_requested": 2,
		"submitted":            3,
		"under_review":         4,
		"approved":             5,
	}
	return order[status] >= order[target]
}

func humanStatus(value string) string {
	normalized := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "_", " ")
	labels := map[string]string{
		"draft":                "rascunho",
		"published":            "publicada",
		"accepted":             "aceita",
		"pending":              "pendente",
		"in progress":          "em preenchimento",
		"submitted":            "enviado",
		"under review":         "em revisão",
		"correction requested": "correção solicitada",
		"approved":             "aprovado",
		"generated":            "gerado",
		"sent":                 "enviado",
		"partially signed":     "parcialmente assinado",
		"signed":               "assinado",
		"cancelled":            "cancelado",
	}
	if label, ok := labels[normalized]; ok {
		return label
	}
	return normalized
}

func EventLabel(value string) string {
	labels := map[string]string{
		"proposal.published":                "Proposta publicada",
		"proposal.accepted":                 "Proposta aceita",
		"document.uploaded":                 "Documento enviado",
		"onboarding.submitted":              "Dados da operação enviados",
		"onboarding.approved":               "Cadastro aprovado",
		"onboarding.auto_approved":          "Cadastro aprovado automaticamente",
		"contract.sent":                     "Contrato enviado para assinatura",
		"contract.signed":                   "Contrato assinado",
		"contract.generation_failed":        "Falha na geração do contrato",
		"contract.evidence_finalized":       "Pacote de evidências finalizado",
		"contract_template.version_created": "Modelo de contrato atualizado",
	}
	if label, ok := labels[value]; ok {
		return label
	}
	return "Atualização registrada"
}

func PipelineEventActor(event domain.PipelineEvent, item domain.PipelineItem) string {
	if strings.TrimSpace(event.ActorName) != "" {
		return event.ActorName
	}

	switch strings.ToLower(strings.TrimSpace(event.ActorType)) {
	case "customer":
		if strings.TrimSpace(item.CustomerResponsibleName) != "" {
			return item.CustomerResponsibleName
		}
		return "Cliente"
	case "system":
		return "Sistema ViaGate"
	case "user":
		if strings.TrimSpace(item.CommercialName) != "" {
			return item.CommercialName
		}
		return "Equipe ViaGate"
	default:
		return "Equipe ViaGate"
	}
}

func EventActor(event domain.PipelineEvent) string {
	if strings.TrimSpace(event.ActorName) != "" {
		return event.ActorName
	}
	switch strings.ToLower(strings.TrimSpace(event.ActorType)) {
	case "customer":
		return "Cliente"
	case "system":
		return "Sistema ViaGate"
	case "user":
		return "Equipe ViaGate"
	default:
		return "Sistema ViaGate"
	}
}
