package templates

import "github.com/jefersonMarques/apresentacao-viagate/internal/domain"

func PipelineCurrentCardClass(item domain.PipelineItem) string {
	if PipelineIsSigned(item) {
		return "pipeline-current-card signed"
	}
	return "pipeline-current-card"
}

func PipelineCurrentLabel(item domain.PipelineItem) string {
	if PipelineIsSigned(item) {
		return "PÓS-CONTRATAÇÃO"
	}
	return "ETAPA ATUAL"
}

func PipelineContractDownloadLabel(item domain.PipelineItem) string {
	if PipelineIsSigned(item) {
		return "Baixar contrato assinado"
	}
	return "Baixar contrato"
}
