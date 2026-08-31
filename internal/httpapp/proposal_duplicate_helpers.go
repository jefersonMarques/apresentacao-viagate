package httpapp

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
	"github.com/jefersonMarques/apresentacao-viagate/internal/proposals"
)

func (a *App) prepareDuplicatedProposalInput(ctx context.Context, source proposals.EditorInput, salesperson domain.User) proposals.EditorInput {
	validUntil := time.Now().AddDate(0, 0, 15)
	contractTemplateID := proposalContentString(source.Content, "proposal", "contract_template_id")
	contractTemplateVersionID := ""
	if contractTemplateID != "" {
		if currentVersionID, err := a.resolveProposalContractTemplateVersion(ctx, contractTemplateID); err == nil {
			contractTemplateVersionID = currentVersionID
		}
	}

	sensitiveValues := []string{
		source.ClientLegalName,
		source.ClientTradeName,
		source.ClientCNPJ,
		source.ClientEmail,
		source.ClientPhone,
		source.ContactName,
		source.ContactEmail,
		source.ContactPhone,
	}

	title := scrubDuplicatedProposalText(strings.TrimSpace(source.Title), sensitiveValues)
	if title == "" {
		title = "Proposta Comercial ViaGate"
	}
	if !strings.HasSuffix(strings.ToLower(title), "(cópia)") {
		title += " (cópia)"
	}

	solutionScope := make([]string, 0, len(source.SolutionScope))
	for _, value := range source.SolutionScope {
		if cleaned := scrubDuplicatedProposalText(value, sensitiveValues); cleaned != "" {
			solutionScope = append(solutionScope, cleaned)
		}
	}
	conditions := make([]string, 0, len(source.Conditions))
	for _, value := range source.Conditions {
		if cleaned := scrubDuplicatedProposalText(value, sensitiveValues); cleaned != "" {
			conditions = append(conditions, cleaned)
		}
	}

	input := proposals.EditorInput{
		Title:          title,
		ValidUntil:     &validUntil,
		SolutionTitle:  scrubDuplicatedProposalText(source.SolutionTitle, sensitiveValues),
		SolutionScope:  solutionScope,
		PricingModel:   source.PricingModel,
		MinimumInvoice: source.MinimumInvoice,
		SetupFee:       source.SetupFee,
		Conditions:     conditions,
		Items:          append([]proposals.EditorItem(nil), source.Items...),
	}
	if input.PricingModel == "" {
		input.PricingModel = "per_item"
	}

	input.Content = map[string]any{
		"proposal": map[string]any{
			"title":                        input.Title,
			"valid_until":                  validUntil.Format("2006-01-02"),
			"contract_template_id":         contractTemplateID,
			"contract_template_version_id": contractTemplateVersionID,
		},
		"client": map[string]any{
			"legal_name": "", "trade_name": "", "company_name": "", "cnpj": "", "email": "", "phone": "", "logo_url": "",
			"street": "", "street_number": "", "complement": "", "district": "", "city": "", "state": "", "postal_code": "",
		},
		"contact":             map[string]any{"name": "", "role": "", "email": "", "phone": ""},
		"operation_context":   "",
		"customer_priorities": []string{},
		"solution_title":      input.SolutionTitle,
		"solution_scope":      input.SolutionScope,
		"salesperson": map[string]any{
			"name": salesperson.Name, "email": salesperson.Email, "phone": salesperson.Phone,
			"job_title": salesperson.JobTitle, "role": salesperson.JobTitle, "photo_url": salesperson.PhotoURL,
			"linkedin": salesperson.LinkedInURL, "instagram": salesperson.InstagramURL,
		},
	}

	canonical := struct {
		Content        map[string]any         `json:"content"`
		PricingModel   string                 `json:"pricing_model"`
		MinimumInvoice float64                `json:"minimum_invoice"`
		SetupFee       float64                `json:"setup_fee"`
		Conditions     []string               `json:"conditions"`
		Items          []proposals.EditorItem `json:"items"`
	}{input.Content, input.PricingModel, input.MinimumInvoice, input.SetupFee, input.Conditions, input.Items}
	encoded, _ := json.Marshal(canonical)
	hash := sha256.Sum256(encoded)
	input.ContentHash = hash[:]
	return input
}

func scrubDuplicatedProposalText(value string, sensitiveValues []string) string {
	cleaned := strings.TrimSpace(value)
	for _, sensitive := range sensitiveValues {
		sensitive = strings.TrimSpace(sensitive)
		if sensitive == "" {
			continue
		}
		pattern, err := regexp.Compile(`(?i)` + regexp.QuoteMeta(sensitive))
		if err != nil {
			continue
		}
		cleaned = pattern.ReplaceAllString(cleaned, "")
	}
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	cleaned = strings.Trim(cleaned, " -–—·,;:/")
	return strings.TrimSpace(cleaned)
}
