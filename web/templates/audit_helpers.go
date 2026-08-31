package templates

func AuditEventLabel(value string) string {
	switch value {
	case "proposal.published":
		return "Proposta publicada"
	case "proposal.accepted":
		return "Proposta aceita"
	case "onboarding.submitted":
		return "Dados da contratação enviados"
	case "onboarding.approved", "onboarding.auto_approved":
		return "Cadastro aprovado"
	case "document.uploaded":
		return "Documento enviado"
	case "contract.sent":
		return "Contrato enviado"
	case "contract.viewed":
		return "Contrato visualizado"
	case "otp.requested":
		return "Código de confirmação solicitado"
	case "otp.verified":
		return "Identidade confirmada"
	case "contract.signed":
		return "Contrato assinado"
	case "contract.evidence_finalized":
		return "Evidências da assinatura finalizadas"
	case "activation.section_saved":
		return "Dados de ativação salvos"
	case "activation.delegated":
		return "Ativação encaminhada para a equipe"
	case "activation.submitted":
		return "Dados para ativação concluídos"
	case "activation.status_changed":
		return "Etapa interna de implantação atualizada"
	case "user.invited":
		return "Usuário convidado"
	case "user.activated":
		return "Usuário ativado"
	case "user.password_reset":
		return "Senha redefinida"
	case "contract.artifact_downloaded":
		return "Documento contratual acessado"
	default:
		return "Evento registrado"
	}
}

func AuditResourceLabel(value string) string {
	switch value {
	case "proposal":
		return "Proposta"
	case "presentation":
		return "Apresentação"
	case "onboarding":
		return "Contratação"
	case "contract":
		return "Contrato"
	case "activation":
		return "Ativação"
	case "user":
		return "Usuário"
	case "contract_template":
		return "Modelo de contrato"
	default:
		return value
	}
}
