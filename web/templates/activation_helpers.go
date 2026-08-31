package templates

import (
	"strings"

	"github.com/jefersonMarques/apresentacao-viagate/internal/activation"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/brfields"
)

func ActivationCanEdit(access activation.Access, section string) bool {
	if access.Profile.Status == "completed" || access.Profile.Status == "under_internal_setup" || access.Profile.Status == "activated" {
		return false
	}
	return access.Section == "all" || access.Section == section
}

func ActivationComplete(profile activation.Profile) bool {
	return strings.TrimSpace(profile.FinanceResponsibleName) != "" &&
		strings.TrimSpace(profile.FinanceResponsibleEmail) != "" &&
		strings.TrimSpace(profile.FinanceResponsiblePhone) != "" &&
		len(profile.Goods) > 0 && len(profile.SystemUsers) > 0
}

func activationDoneClass(done bool) string {
	if done {
		return "done"
	}
	return "pending"
}

func ActivationStatusLabel(status string) string {
	switch status {
	case "pending":
		return "Aguardando dados"
	case "in_progress":
		return "Em preenchimento"
	case "completed":
		return "Dados recebidos"
	case "under_internal_setup":
		return "Em implantação"
	case "activated":
		return "Operação liberada"
	default:
		return "Em andamento"
	}
}

func ActivationAddress(profile activation.Profile) string {
	parts := []string{}
	for _, value := range []string{profile.Street, profile.StreetNumber, profile.Complement, profile.District, profile.City, profile.State, brfields.FormatPostalCode(profile.PostalCode)} {
		value = strings.TrimSpace(value)
		if value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, ", ")
}

func ActivationSectionLabel(section string) string {
	switch section {
	case "finance":
		return "dados financeiros"
	case "goods":
		return "mercadorias transportadas"
	case "users":
		return "usuários do sistema"
	default:
		return "todos os dados para ativação"
	}
}

func humanOperationType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "normal":
		return "Normal"
	case "avulsa", "avulso":
		return "Avulsa"
	case "":
		return "Não informada"
	default:
		return strings.TrimSpace(value)
	}
}
