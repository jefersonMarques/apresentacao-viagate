package templates

import (
	"fmt"
	"strings"
	"time"

	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
	"github.com/jefersonMarques/apresentacao-viagate/internal/proposals"
)

func editorDate(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format("2006-01-02")
}

func proposalTitle(value string) string {
	if strings.TrimSpace(value) == "" {
		return "Proposta Comercial ViaGate"
	}
	return value
}

func editorPrice(value float64, found bool) string {
	if !found {
		return ""
	}
	return fmt.Sprintf("%.2f", value)
}

func editorFloat(value float64) string {
	if value == 0 {
		return ""
	}
	return fmt.Sprintf("%.2f", value)
}

func joinModels(values []string) string { return strings.Join(values, ",") }

func conditionChecked(current []string, value string) bool {
	for _, item := range current {
		if item == value {
			return true
		}
	}
	return len(current) == 0
}

func ProposalContractTemplateOptions(input proposals.EditorInput) []domain.ContractTemplate {
	if input.Content == nil {
		return nil
	}
	items, _ := input.Content["__ui_contract_template_options"].([]domain.ContractTemplate)
	return items
}

func ProposalContractTemplateID(input proposals.EditorInput) string {
	if input.Content == nil {
		return ""
	}
	proposal, _ := input.Content["proposal"].(map[string]any)
	value, _ := proposal["contract_template_id"].(string)
	return value
}

func proposalPublicURL(token string) string { return "/p/" + token }
