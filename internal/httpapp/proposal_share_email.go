package httpapp

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/access"
	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
	"github.com/jefersonMarques/apresentacao-viagate/internal/notifications"
	"github.com/jefersonMarques/apresentacao-viagate/internal/proposals"
)

func (a *App) shareProposalByEmail(w http.ResponseWriter, r *http.Request) {
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
	`, proposalID).Scan(&status, &publicToken, &ownerID); err != nil {
		http.Error(w, "não foi possível carregar a proposta", http.StatusInternalServerError)
		return
	}
	if status != "published" || publicToken == "" {
		http.Error(w, "publique a proposta antes de compartilhá-la", http.StatusConflict)
		return
	}
	if strings.TrimSpace(input.ContactEmail) == "" {
		http.Error(w, "informe o e-mail do contato da negociação antes de enviar", http.StatusUnprocessableEntity)
		return
	}

	seller, err := a.authStore.Profile(r.Context(), ownerID)
	if err != nil {
		a.logger.Error("load proposal seller profile for email failed", "proposal_id", proposalID, "seller_id", ownerID, "error", err)
		seller = proposalSellerFromContent(input.Content)
	}

	subject, htmlBody, textBody := buildProposalShareEmail(a.cfg.BaseURL, input, seller, publicToken)
	if err := notifications.EnqueueWithOptions(r.Context(), a.pool, notifications.MessageOptions{
		Kind:      "proposal_share",
		ToName:    input.ContactName,
		ToEmail:   input.ContactEmail,
		Subject:   subject,
		HTMLBody:  htmlBody,
		TextBody:  textBody,
		Sensitive: true,
	}); err != nil {
		a.logger.Error("queue proposal share email failed", "proposal_id", proposalID, "recipient", input.ContactEmail, "error", err)
		http.Error(w, "não foi possível agendar o envio do e-mail", http.StatusInternalServerError)
		return
	}

	_, _ = a.pool.Exec(r.Context(), `
		insert into audit_events(actor_user_id,event_type,resource_type,resource_id,metadata)
		values($1,'proposal.share_email_queued','proposal',$2,jsonb_build_object('recipient',$3::text))
	`, user.ID, proposalID, input.ContactEmail)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"queued":true,"message":"E-mail adicionado à fila de envio."}`))
}

func buildProposalShareEmail(baseURL string, input proposals.EditorInput, seller domain.User, publicToken string) (string, string, string) {
	clientName := strings.TrimSpace(input.ClientTradeName)
	if clientName == "" {
		clientName = strings.TrimSpace(input.ClientLegalName)
	}
	if clientName == "" {
		clientName = "sua empresa"
	}
	recipientName := strings.TrimSpace(input.ContactName)
	greeting := "Olá."
	if recipientName != "" {
		greeting = "Olá, " + recipientName + "."
	}

	baseURL = strings.TrimRight(baseURL, "/")
	proposalURL := baseURL + "/p/" + publicToken
	photoURL := absoluteProposalEmailURL(baseURL, seller.PhotoURL)

	signaturePhoto := ""
	if photoURL != "" {
		signaturePhoto = fmt.Sprintf(`<td style="width:72px;padding-right:14px;vertical-align:top"><img src="%s" alt="" width="64" height="64" style="display:block;width:64px;height:64px;object-fit:cover;border:0"></td>`, html.EscapeString(photoURL))
	}
	sellerName := strings.TrimSpace(seller.Name)
	if sellerName == "" {
		sellerName = "Equipe ViaGate"
	}
	roleLine := ""
	if strings.TrimSpace(seller.JobTitle) != "" {
		roleLine = fmt.Sprintf(`<div style="margin-top:3px;color:#6f8290;font-size:12px">%s</div>`, html.EscapeString(seller.JobTitle))
	}
	contactParts := []string{}
	if strings.TrimSpace(seller.Phone) != "" {
		contactParts = append(contactParts, html.EscapeString(seller.Phone))
	}
	if strings.TrimSpace(seller.Email) != "" {
		contactParts = append(contactParts, html.EscapeString(seller.Email))
	}
	contactLine := ""
	if len(contactParts) > 0 {
		contactLine = fmt.Sprintf(`<div style="margin-top:7px;color:#536875;font-size:12px">%s</div>`, strings.Join(contactParts, " &nbsp;·&nbsp; "))
	}

	subject := "Proposta ViaGate — " + clientName
	htmlBody := fmt.Sprintf(`<!doctype html>
<html lang="pt-BR">
<body style="margin:0;padding:0;background:#eef3f6;font-family:Arial,Helvetica,sans-serif;color:#102637">
<table role="presentation" width="100%%" cellspacing="0" cellpadding="0" style="background:#eef3f6;padding:28px 12px">
<tr><td align="center">
<table role="presentation" width="620" cellspacing="0" cellpadding="0" style="width:100%%;max-width:620px;background:#ffffff;border:1px solid #d6e0e6">
<tr><td style="background:#071827;padding:24px 28px;border-bottom:4px solid #ff6b18">
<div style="color:#ff6b18;font-size:11px;font-weight:700;letter-spacing:2px">VIAGATE</div>
<div style="margin-top:7px;color:#ffffff;font-size:22px;font-weight:700">Proposta comercial</div>
</td></tr>
<tr><td style="padding:30px 28px">
<p style="margin:0 0 16px;font-size:15px;line-height:1.6">%s</p>
<p style="margin:0 0 12px;font-size:14px;line-height:1.65">Preparei a proposta comercial da ViaGate para <strong>%s</strong>.</p>
<p style="margin:0 0 24px;color:#536875;font-size:13px;line-height:1.65">No link abaixo você pode revisar a solução, os produtos, valores e condições comerciais apresentados para a negociação.</p>
<table role="presentation" cellspacing="0" cellpadding="0"><tr><td style="background:#ff6b18"><a href="%s" style="display:inline-block;padding:13px 20px;color:#ffffff;text-decoration:none;font-size:13px;font-weight:700">Abrir proposta</a></td></tr></table>
<p style="margin:22px 0 0;color:#82919a;font-size:11px;line-height:1.6">Se o botão não abrir, copie este endereço no navegador:<br><a href="%s" style="color:#536875;word-break:break-all">%s</a></p>
</td></tr>
<tr><td style="padding:22px 28px;border-top:1px solid #e2e8ec;background:#f8fafb">
<table role="presentation" cellspacing="0" cellpadding="0"><tr>
%s
<td style="vertical-align:top">
<div style="color:#102637;font-size:14px;font-weight:700">%s</div>
%s
%s
</td>
</tr></table>
</td></tr>
</table>
</td></tr>
</table>
</body>
</html>`,
		html.EscapeString(greeting),
		html.EscapeString(clientName),
		html.EscapeString(proposalURL),
		html.EscapeString(proposalURL),
		html.EscapeString(proposalURL),
		signaturePhoto,
		html.EscapeString(sellerName),
		roleLine,
		contactLine,
	)
	textBody := fmt.Sprintf("%s\n\nPreparei a proposta comercial da ViaGate para %s.\n\nAcesse: %s\n\n%s", greeting, clientName, proposalURL, proposalSellerTextSignature(seller))
	return subject, htmlBody, textBody
}

func proposalSellerFromContent(content map[string]any) domain.User {
	group, _ := content["salesperson"].(map[string]any)
	value := func(key string) string {
		text, _ := group[key].(string)
		return strings.TrimSpace(text)
	}
	return domain.User{
		Name:       value("name"),
		Email:      value("email"),
		Phone:      value("phone"),
		JobTitle:   value("job_title"),
		PhotoURL:   value("photo_url"),
		LinkedInURL: value("linkedin"),
		InstagramURL: value("instagram"),
	}
}

func proposalSellerTextSignature(seller domain.User) string {
	lines := []string{strings.TrimSpace(seller.Name)}
	if strings.TrimSpace(seller.JobTitle) != "" {
		lines = append(lines, strings.TrimSpace(seller.JobTitle))
	}
	if strings.TrimSpace(seller.Phone) != "" {
		lines = append(lines, strings.TrimSpace(seller.Phone))
	}
	if strings.TrimSpace(seller.Email) != "" {
		lines = append(lines, strings.TrimSpace(seller.Email))
	}
	filtered := lines[:0]
	for _, line := range lines {
		if line != "" {
			filtered = append(filtered, line)
		}
	}
	if len(filtered) == 0 {
		return "Equipe ViaGate"
	}
	return strings.Join(filtered, "\n")
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

func queueProposalShareEmail(ctx context.Context) {
	_ = ctx
}
