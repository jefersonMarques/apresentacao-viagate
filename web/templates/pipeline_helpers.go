package templates

import (
	"strings"

	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
)

func PipelineStage(item domain.PipelineItem) string {
	if item.ContractStatus=="signed"{return "Concluído"}
	if item.ContractStatus!=""{return "Contrato · "+humanStatus(item.ContractStatus)}
	if item.OnboardingStatus!=""{return "Cadastro · "+humanStatus(item.OnboardingStatus)}
	if item.ProposalStatus=="accepted"{return "Proposta aceita"}
	if item.ProposalStatus=="published"{return "Proposta enviada"}
	return "Proposta · "+humanStatus(item.ProposalStatus)
}

func humanStatus(value string) string {
	replacer:=strings.NewReplacer(
		"_"," ",
		"draft","rascunho",
		"published","publicada",
		"accepted","aceita",
		"pending","pendente",
		"in progress","em preenchimento",
		"submitted","enviado",
		"under review","em revisão",
		"correction requested","correção solicitada",
		"approved","aprovado",
		"generated","gerado",
		"sent","enviado",
		"partially signed","parcialmente assinado",
		"signed","assinado",
		"cancelled","cancelado",
	)
	return replacer.Replace(strings.ToLower(value))
}

func EventLabel(value string) string {
	labels:=map[string]string{
		"proposal.published":"Proposta publicada",
		"proposal.accepted":"Proposta aceita",
		"document.uploaded":"Documento enviado",
		"onboarding.submitted":"Dados da operação enviados",
		"contract.generation_failed":"Falha na geração do contrato",
		"contract.evidence_finalized":"Pacote de evidências finalizado",
		"contract_template.version_created":"Modelo de contrato atualizado",
	}
	if label,ok:=labels[value];ok{return label}
	return strings.ReplaceAll(value,"."," · ")
}

func EventActor(event domain.PipelineEvent) string {
	if strings.TrimSpace(event.ActorName)!=""{return event.ActorName}
	if strings.TrimSpace(event.ActorType)!=""{return event.ActorType}
	return "sistema"
}
