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
		Title:       "Apólice enviada",
		Description: "A apólice necessária para a contratação foi anexada.",
		EventTypes:  []string{"document.uploaded"},
	},
	{
		Key:         "onboarding_submitted",
		Title:       "Dados para contratação enviados",
		Description: "O cliente enviou os dados necessários para a ViaGate preparar o contrato.",
		EventTypes:  []string{"onboarding.submitted"},
	},
	{
		Key:         "onboarding_approved",
		Title:       "Contratação aprovada",
		Description: "Os dados foram validados e o contrato pôde ser preparado.",
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
		Key:         "activation_data",
		Title:       "Dados para ativação",
		Description: "Financeiro, mercadorias e usuários iniciais foram informados para preparar a operação.",
		EventTypes:  []string{"activation.submitted"},
	},
	{
		Key:         "internal_setup",
		Title:       "Implantação interna",
		Description: "A equipe ViaGate está configurando os dados recebidos e preparando a operação.",
		EventTypes:  nil,
	},
	{
		Key:         "activated",
		Title:       "Operação liberada",
		Description: "A implantação foi concluída e a operação está liberada para uso.",
		EventTypes:  nil,
	},
}

func PipelineStage(item domain.PipelineItem) string {
	if PipelineIsSigned(item) {
		switch item.ActivationStatus {
		case "activated":
			return "Operação liberada"
		case "under_internal_setup":
			return "Implantação interna"
		case "completed":
			return "Dados para ativação recebidos"
		case "in_progress":
			return "Dados para ativação · em preenchimento"
		case "pending", "":
			return "Dados para ativação"
		default:
			return "Dados para ativação"
		}
	}
	if item.ContractStatus != "" {
		return "Contrato · " + humanStatus(item.ContractStatus)
	}
	if item.OnboardingStatus != "" {
		return "Contratação · " + humanStatus(item.OnboardingStatus)
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

func PipelineIsActivated(item domain.PipelineItem) bool {
	return item.ActivationStatus == "activated" || item.ActivatedAt != nil
}

func PipelineStageBadgeClass(item domain.PipelineItem) string {
	if PipelineIsActivated(item) {
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
	case PipelineIsActivated(item):
		return "A implantação foi concluída e a operação está liberada. O contrato permanece preservado como foi assinado."
	case item.ActivationStatus == "under_internal_setup":
		return "Os dados para ativação foram recebidos e a equipe ViaGate está preparando a operação."
	case item.ActivationStatus == "completed":
		return "O cliente concluiu os dados para ativação. A implantação interna é a próxima etapa."
	case item.ActivationStatus == "in_progress":
		return "O contrato está assinado e o cliente ou alguém da equipe dele está complementando os dados para ativação."
	case PipelineIsSigned(item):
		return "A contratação está formalizada. Agora aguardamos os dados operacionais necessários para preparar a ativação."
	case item.ContractStatus == "sent" || item.ContractStatus == "partially_signed":
		return "O contrato já foi enviado e está aguardando a conclusão da assinatura eletrônica."
	case item.ContractStatus == "generated":
		return "O contrato foi gerado e está sendo preparado para envio ao responsável."
	case item.OnboardingStatus == "approved":
		return "Os dados para contratação foram aprovados e a preparação do contrato é a próxima etapa."
	case item.OnboardingStatus == "under_review":
		return "Os dados enviados pelo cliente estão em revisão pela equipe ViaGate."
	case item.OnboardingStatus == "submitted":
		return "Os dados para contratação foram enviados e aguardam validação."
	case item.OnboardingStatus != "":
		return "O cliente está concluindo somente os dados necessários para preparar o contrato."
	case item.ProposalStatus == "accepted":
		return "A proposta foi aceita e os dados para contratação são a próxima etapa."
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

	for index := range steps {
		steps[index].Current = currentIndex >= 0 && index == currentIndex
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
	case "activation_data":
		return item.ActivationStatus == "completed" || item.ActivationStatus == "under_internal_setup" || PipelineIsActivated(item)
	case "internal_setup":
		return PipelineIsActivated(item)
	case "activated":
		return PipelineIsActivated(item)
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
	case "activation_data":
		return item.ActivationSubmittedAt
	case "activated":
		return item.ActivatedAt
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
	case "proposal_accepted", "document_uploaded", "onboarding_submitted", "activation_data":
		return customer
	case "onboarding_approved", "contract_sent":
		return "Sistema ViaGate"
	case "contract_signed":
		return PipelineSignedBy(item)
	case "internal_setup", "activated":
		return "Equipe ViaGate"
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
		"expired":              "expirada",
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
		"completed":            "dados recebidos",
		"under internal setup": "em implantação",
		"activated":            "operação liberada",
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
		"document.uploaded":                 "Apólice enviada",
		"onboarding.submitted":              "Dados para contratação enviados",
		"onboarding.approved":               "Contratação aprovada",
		"onboarding.auto_approved":          "Contratação aprovada automaticamente",
		"contract.sent":                     "Contrato enviado para assinatura",
		"contract.signed":                   "Contrato assinado",
		"contract.generation_failed":        "Falha na geração do contrato",
		"contract.evidence_finalized":       "Evidências da assinatura finalizadas",
		"contract_template.version_created": "Modelo de contrato atualizado",
		"activation.section_saved":          "Dados para ativação atualizados",
		"activation.delegated":              "Preenchimento encaminhado para a equipe do cliente",
		"activation.submitted":              "Dados para ativação concluídos",
		"activation.status_changed":         "Etapa de implantação atualizada",
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
