package templates

import (
	"strings"

	"github.com/jefersonMarques/apresentacao-viagate/internal/activation"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/brfields"
)

func ActivationCanEdit(access activation.Access, section string) bool {
	if ActivationFinished(access.Profile) {
		return false
	}
	if section == "operation" {
		return false
	}
	if (section == "policy" || section == "goods") && !access.Profile.RequiresPolicy {
		return false
	}
	return access.Section == "all" || access.Section == section
}

func ActivationComplete(profile activation.Profile) bool {
	financeComplete := strings.TrimSpace(profile.FinanceResponsibleName) != "" &&
		strings.TrimSpace(profile.FinanceResponsibleEmail) != "" &&
		strings.TrimSpace(profile.FinanceResponsiblePhone) != ""
	if !financeComplete || len(profile.SystemUsers) == 0 {
		return false
	}
	if !profile.RequiresPolicy {
		return true
	}
	return profile.HasPolicy && len(profile.Goods) > 0
}

func ActivationContractSigned(profile activation.Profile) bool {
	return profile.ContractStatus == "signed" && profile.FullySignedAt != nil
}

func ActivationPolicyStatus(hasPolicy bool) string {
	if hasPolicy {
		return "Apólice recebida"
	}
	return "Apólice pendente"
}

func ActivationPolicyButton(hasPolicy bool) string {
	if hasPolicy {
		return "Substituir apólice"
	}
	return "Enviar apólice"
}

func ActivationContractStatus(profile activation.Profile) string {
	if ActivationContractSigned(profile) {
		return "Assinado"
	}
	return "Aguardando assinatura"
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

func ActivationOperationComplete(profile activation.Profile) bool {
	operationType := strings.ToLower(strings.TrimSpace(profile.OperationType))
	return (operationType == "normal" || operationType == "avulsa") &&
		strings.TrimSpace(profile.Insurer) != "" &&
		profile.PolicyStartDate != "" && profile.PolicyEndDate != ""
}

type ActivationStep struct {
    Key      string
    Label    string
    Complete bool
}

func ActivationFinished(profile activation.Profile) bool {
    return profile.Status == "completed" ||
        profile.Status == "under_internal_setup" ||
        profile.Status == "activated"
}

func ActivationCanView(access activation.Access, section string) bool {
    if section == "operation" {
        return false
    }
    if (section == "policy" || section == "goods") && !access.Profile.RequiresPolicy {
        return false
    }

    if access.Section == "all" {
        return true
    }

    return access.Section == section
}

func ActivationFinanceComplete(profile activation.Profile) bool {
    return strings.TrimSpace(profile.FinanceResponsibleName) != "" &&
        strings.TrimSpace(profile.FinanceResponsiblePhone) != "" &&
        strings.TrimSpace(profile.FinanceResponsibleEmail) != ""
}

func ActivationSteps(access activation.Access) []ActivationStep {
    profile := access.Profile
    candidates := []ActivationStep{
        {Key: "policy", Label: "Apólice", Complete: profile.HasPolicy},
        {Key: "finance", Label: "Financeiro", Complete: ActivationFinanceComplete(profile)},
        {Key: "goods", Label: "Mercadorias", Complete: len(profile.Goods) > 0},
        {Key: "users", Label: "Usuários", Complete: len(profile.SystemUsers) > 0},
    }

    steps := make([]ActivationStep, 0, len(candidates)+1)
    for _, step := range candidates {
        if ActivationCanView(access, step.Key) {
            steps = append(steps, step)
        }
    }

    if access.Section == "all" {
        steps = append(steps, ActivationStep{
            Key: "review", Label: "Revisão", Complete: ActivationComplete(profile),
        })
    }

    return steps
}
