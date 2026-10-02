package httpapp

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/access"
	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
	"github.com/jefersonMarques/apresentacao-viagate/internal/emailtemplates"
	"github.com/jefersonMarques/apresentacao-viagate/internal/proposals"
)

type proposalEmailDraftResponse struct {
	To       string `json:"to"`
	Subject  string `json:"subject"`
	HTMLBody string `json:"html_body"`
	TextBody string `json:"text_body"`
}

func (a *App) proposalEmailDraft(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	proposalID := chi.URLParam(r, "id")
	allowAll := access.Can(user, access.ProposalReadAll)

	input, _, err := a.proposalStore.EditorByID(r.Context(), user.ID, proposalID, allowAll)
	if err != nil {
		http.Error(w, "proposta não encontrada ou acesso negado", http.StatusNotFound)
		return
	}

	var status, publicToken, ownerID string
	if err := a.pool.QueryRow(r.Context(), `
		select p.status::text,coalesce(v.public_token::text,''),p.created_by::text
		from proposals p
		left join proposal_versions v
		  on v.proposal_id=p.id
		 and v.version_number=p.current_version
		 and v.published_at is not null
		where p.id=$1
		  and p.deleted_at is null
	`, proposalID).Scan(&status, &publicToken, &ownerID); err != nil {
		http.Error(w, "não foi possível carregar a proposta", http.StatusInternalServerError)
		return
	}
	if status != "published" || publicToken == "" {
		http.Error(w, "publique a proposta antes de preparar o e-mail", http.StatusConflict)
		return
	}
	if strings.TrimSpace(input.ContactEmail) == "" {
		http.Error(w, "informe o e-mail do contato da negociação antes de abrir o cliente de e-mail", http.StatusUnprocessableEntity)
		return
	}

	seller, err := a.authStore.Profile(r.Context(), ownerID)
	if err != nil {
		a.logger.Error("load proposal seller profile for email draft failed", "proposal_id", proposalID, "seller_id", ownerID, "error", err)
		seller = proposalSellerFromContent(input.Content)
	}

	templateID := strings.TrimSpace(r.URL.Query().Get("template"))
	var emailTemplate emailtemplates.Template
	if templateID == "" {
		emailTemplate, err = a.emailTemplateStore.Default(r.Context(), emailtemplates.PurposeProposalShare)
	} else {
		emailTemplate, err = a.emailTemplateStore.ByID(r.Context(), templateID)
		if err == nil && (emailTemplate.Purpose != emailtemplates.PurposeProposalShare || !emailTemplate.IsActive) {
			http.Error(w, "modelo de e-mail indisponível", http.StatusNotFound)
			return
		}
	}
	if err != nil {
		http.Error(w, "nenhum modelo de e-mail ativo está disponível", http.StatusConflict)
		return
	}

	draft, err := emailtemplates.Render(
		emailTemplate,
		a.proposalEmailVariables(input, seller, publicToken),
	)
	if err != nil {
		a.logger.Error("render proposal email draft failed", "proposal_id", proposalID, "template_id", emailTemplate.ID, "error", err)
		http.Error(w, "não foi possível preparar o e-mail", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(proposalEmailDraftResponse{
		To:       input.ContactEmail,
		Subject:  draft.Subject,
		HTMLBody: draft.HTMLBody,
		TextBody: draft.TextBody,
	})
}

func (a *App) proposalEmailVariables(input proposals.EditorInput, seller domain.User, publicToken string) map[string]string {
	clientDisplayName := strings.TrimSpace(input.ClientTradeName)
	if clientDisplayName == "" {
		clientDisplayName = strings.TrimSpace(input.ClientLegalName)
	}
	if clientDisplayName == "" {
		clientDisplayName = "Cliente"
	}

	contactName := strings.TrimSpace(input.ContactName)
	contactGreeting := "Olá."
	if contactName != "" {
		contactGreeting = "Olá, " + contactName + "."
	}

	validUntil := ""
	if input.ValidUntil != nil {
		validUntil = input.ValidUntil.Format("02/01/2006")
	}

	baseURL := strings.TrimRight(a.cfg.BaseURL, "/")
	return map[string]string{
		"brand.logo_url":       baseURL + "/v1/assets/logo-viagate-color.svg",
		"client.display_name":   clientDisplayName,
		"client.legal_name":     strings.TrimSpace(input.ClientLegalName),
		"client.trade_name":     strings.TrimSpace(input.ClientTradeName),
		"client.email":          strings.TrimSpace(input.ClientEmail),
		"contact.name":          contactName,
		"contact.greeting":      contactGreeting,
		"contact.role":          strings.TrimSpace(input.ContactRole),
		"contact.email":         strings.TrimSpace(input.ContactEmail),
		"contact.phone":         strings.TrimSpace(input.ContactPhone),
		"proposal.title":        strings.TrimSpace(input.Title),
		"proposal.url":          baseURL + "/p/" + publicToken,
		"proposal.valid_until":  validUntil,
		"salesperson.name":      strings.TrimSpace(seller.Name),
		"salesperson.job_title": strings.TrimSpace(seller.JobTitle),
		"salesperson.email":     strings.TrimSpace(seller.Email),
		"salesperson.phone":     strings.TrimSpace(seller.Phone),
		"salesperson.photo_url": absoluteProposalEmailURL(baseURL, seller.PhotoURL),
		"salesperson.linkedin":  strings.TrimSpace(seller.LinkedInURL),
		"salesperson.instagram": strings.TrimSpace(seller.InstagramURL),
	}
}

func proposalSellerFromContent(content map[string]any) domain.User {
	group, _ := content["salesperson"].(map[string]any)
	value := func(key string) string {
		text, _ := group[key].(string)
		return strings.TrimSpace(text)
	}
	return domain.User{
		Name:         value("name"),
		Email:        value("email"),
		Phone:        value("phone"),
		JobTitle:     value("job_title"),
		PhotoURL:     value("photo_url"),
		LinkedInURL:  value("linkedin"),
		InstagramURL: value("instagram"),
	}
}

func absoluteProposalEmailURL(baseURL, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "http://") {
		return value
	}
	if strings.HasPrefix(value, "/") {
		return strings.TrimRight(baseURL, "/") + value
	}
	return value
}
