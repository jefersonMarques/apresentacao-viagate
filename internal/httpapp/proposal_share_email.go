package httpapp

import (
	"context"
	"encoding/json"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

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

	var status, publicToken, ownerID, currentVersionID string
	var requiresPolicy bool
	var publishedContentJSON []byte
	if err := a.pool.QueryRow(r.Context(), `
		select p.status::text,coalesce(v.public_token::text,''),p.created_by::text,
		       coalesce(v.id::text,''),coalesce(v.requires_policy,false),coalesce(v.content,'{}'::jsonb)
		from proposals p
		left join proposal_versions v
		  on v.proposal_id=p.id
		 and v.version_number=p.current_version
		 and v.published_at is not null
		where p.id=$1
		  and p.deleted_at is null
	`, proposalID).Scan(&status, &publicToken, &ownerID, &currentVersionID, &requiresPolicy, &publishedContentJSON); err != nil {
		http.Error(w, "não foi possível carregar a proposta", http.StatusInternalServerError)
		return
	}
	if status != "published" || publicToken == "" {
		http.Error(w, "publique a proposta antes de preparar o e-mail", http.StatusConflict)
		return
	}

	var publishedContent map[string]any
	if err := json.Unmarshal(publishedContentJSON, &publishedContent); err != nil {
		a.logger.Error("decode published proposal content for email draft failed", "proposal_id", proposalID, "version_id", currentVersionID, "error", err)
		http.Error(w, "não foi possível carregar os dados publicados da proposta", http.StatusInternalServerError)
		return
	}
	input = proposalEmailInputFromPublishedContent(input, publishedContent)
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

	products, err := a.proposalEmailProducts(r.Context(), currentVersionID)
	if err != nil {
		a.logger.Error("load proposal products for email draft failed", "proposal_id", proposalID, "version_id", currentVersionID, "error", err)
		http.Error(w, "não foi possível carregar os produtos da proposta", http.StatusInternalServerError)
		return
	}

	variables := a.proposalEmailVariables(input, seller, publicToken)
	for key, value := range proposalEmailProductVariables(products, requiresPolicy) {
		variables[key] = value
	}
	draft, err := emailtemplates.Render(
		emailTemplate,
		variables,
	)
	if err != nil {
		a.logger.Error("render proposal email draft failed", "proposal_id", proposalID, "template_id", emailTemplate.ID, "error", err)
		http.Error(w, "não foi possível preparar o e-mail", http.StatusInternalServerError)
		return
	}

	if draft.TextBody == "" {
		draft.TextBody = "Acesse a proposta: " + variables["proposal.url"]
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

func proposalEmailInputFromPublishedContent(input proposals.EditorInput, content map[string]any) proposals.EditorInput {
	input.Content = content
	if value, ok := proposalEmailSnapshotValue(content, "proposal", "title"); ok {
		input.Title = value
	}
	if value, ok := proposalEmailSnapshotValue(content, "client", "legal_name"); ok {
		input.ClientLegalName = value
	}
	if value, ok := proposalEmailSnapshotValue(content, "client", "trade_name"); ok {
		input.ClientTradeName = value
	}
	if value, ok := proposalEmailSnapshotValue(content, "client", "email"); ok {
		input.ClientEmail = value
	}
	if value, ok := proposalEmailSnapshotValue(content, "contact", "name"); ok {
		input.ContactName = value
	}
	if value, ok := proposalEmailSnapshotValue(content, "contact", "role"); ok {
		input.ContactRole = value
	}
	if value, ok := proposalEmailSnapshotValue(content, "contact", "email"); ok {
		input.ContactEmail = value
	}
	if value, ok := proposalEmailSnapshotValue(content, "contact", "phone"); ok {
		input.ContactPhone = value
	}
	if rawValidUntil, ok := proposalEmailSnapshotValue(content, "proposal", "valid_until"); ok {
		input.ValidUntil = nil
		if rawValidUntil != "" {
			if parsed, err := time.Parse("2006-01-02", rawValidUntil); err == nil {
				input.ValidUntil = &parsed
			}
		}
	}
	return input
}

func proposalEmailSnapshotValue(content map[string]any, section, key string) (string, bool) {
	group, ok := content[section].(map[string]any)
	if !ok {
		return "", false
	}
	raw, exists := group[key]
	if !exists {
		return "", false
	}
	value, ok := raw.(string)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(value), true
}

type proposalEmailProduct struct {
	Label       string
	Description string
	GroupName   string
	IsOptional  bool
}

func (a *App) proposalEmailProducts(ctx context.Context, versionID string) ([]proposalEmailProduct, error) {
	if strings.TrimSpace(versionID) == "" {
		return nil, nil
	}
	rows, err := a.pool.Query(ctx, `
		select label,
		       coalesce(metadata->>'product_description',''),
		       group_name,
		       is_optional
		from proposal_items
		where proposal_version_id=$1
		order by sort_order,id
	`, versionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []proposalEmailProduct{}
	for rows.Next() {
		var item proposalEmailProduct
		if err := rows.Scan(&item.Label, &item.Description, &item.GroupName, &item.IsOptional); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func proposalEmailProductVariables(items []proposalEmailProduct, requiresPolicy bool) map[string]string {
	var htmlBody strings.Builder
	var textBody strings.Builder

	if len(items) == 0 {
		htmlBody.WriteString(`<div style="padding:14px 16px;background:#f7f9fa;color:#536875;font-size:13px;line-height:1.6">Consulte a proposta para visualizar os produtos e serviços contemplados.</div>`)
		textBody.WriteString("Consulte a proposta para visualizar os produtos e serviços contemplados.")
	} else {
		htmlBody.WriteString(`<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="width:100%;border-collapse:collapse">`)
		for index, item := range items {
			if index > 0 {
				textBody.WriteString("\n")
			}
			label := strings.TrimSpace(item.Label)
			description := strings.TrimSpace(item.Description)
			groupName := strings.TrimSpace(item.GroupName)

			htmlBody.WriteString(`<tr><td style="padding:12px 0;border-bottom:1px solid #e2e8ec">`)
			if groupName != "" {
				htmlBody.WriteString(`<div style="margin-bottom:3px;color:#82919a;font-size:10px;font-weight:700;letter-spacing:.08em;text-transform:uppercase">`)
				htmlBody.WriteString(html.EscapeString(groupName))
				htmlBody.WriteString(`</div>`)
			}
			htmlBody.WriteString(`<strong style="color:#102637;font-size:13px">`)
			htmlBody.WriteString(html.EscapeString(label))
			htmlBody.WriteString(`</strong>`)
			if item.IsOptional {
				htmlBody.WriteString(` <span style="display:inline-block;margin-left:6px;padding:2px 6px;background:#eef3f6;color:#536875;font-size:9px;font-weight:700;text-transform:uppercase">Opcional</span>`)
			}
			if description != "" {
				htmlBody.WriteString(`<div style="margin-top:4px;color:#6f8290;font-size:12px;line-height:1.55">`)
				htmlBody.WriteString(html.EscapeString(description))
				htmlBody.WriteString(`</div>`)
			}
			htmlBody.WriteString(`</td></tr>`)

			textBody.WriteString("- ")
			textBody.WriteString(label)
			if item.IsOptional {
				textBody.WriteString(" (Opcional)")
			}
			if description != "" {
				textBody.WriteString(" — ")
				textBody.WriteString(description)
			}
		}
		htmlBody.WriteString(`</table>`)
	}

	requiresPolicyText := "Não"
	if requiresPolicy {
		requiresPolicyText = "Sim"
	}
	return map[string]string{
		"proposal.products_html":   htmlBody.String(),
		"proposal.products_text":   textBody.String(),
		"proposal.products_count":  strconv.Itoa(len(items)),
		"proposal.requires_policy": requiresPolicyText,
	}
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
		"brand.logo_url":       baseURL + "/v1/assets/logo-viagate-email.png",
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
