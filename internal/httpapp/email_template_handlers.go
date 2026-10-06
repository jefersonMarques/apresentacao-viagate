package httpapp

import (
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/emailtemplates"
	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

func (a *App) emailTemplatesPage(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	items, err := a.emailTemplateStore.List(r.Context(), emailtemplates.PurposeProposalShare)
	if err != nil {
		http.Error(w, "não foi possível carregar os modelos de e-mail", http.StatusInternalServerError)
		return
	}

	selected := emailtemplates.Template{Purpose: emailtemplates.PurposeProposalShare, IsActive: true}
	if selectedID := strings.TrimSpace(r.URL.Query().Get("template")); selectedID != "" {
		selected, err = a.emailTemplateStore.ByID(r.Context(), selectedID)
		if err != nil || selected.Purpose != emailtemplates.PurposeProposalShare {
			http.Error(w, "modelo de e-mail não encontrado", http.StatusNotFound)
			return
		}
	}

	message := ""
	if r.URL.Query().Get("saved") == "1" {
		message = "Modelo de e-mail salvo."
	}
	render(r.Context(), w, http.StatusOK, templates.EmailTemplatesPage(
		user,
		items,
		emailtemplates.AllowedVariables(),
		selected,
		message,
	))
}

func (a *App) saveEmailTemplate(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, "dados inválidos", http.StatusBadRequest)
		return
	}

	input := emailtemplates.SaveInput{
		ID:              strings.TrimSpace(r.FormValue("template_id")),
		Purpose:         emailtemplates.PurposeProposalShare,
		Name:            strings.TrimSpace(r.FormValue("name")),
		Description:     strings.TrimSpace(r.FormValue("description")),
		SubjectTemplate: strings.TrimSpace(r.FormValue("subject_template")),
		HTMLTemplate:    strings.TrimSpace(r.FormValue("html_template")),
		TextTemplate:    strings.TrimSpace(r.FormValue("text_template")),
		IsActive:        r.FormValue("is_active") == "1",
		MakeDefault:     r.FormValue("make_default") == "1",
		UserID:          user.ID,
	}
	if input.Name == "" {
		http.Error(w, "nome do modelo é obrigatório", http.StatusBadRequest)
		return
	}
	if err := emailtemplates.Validate(input.SubjectTemplate, input.HTMLTemplate, input.TextTemplate); err != nil {
		http.Error(w, "modelo inválido: "+err.Error(), http.StatusBadRequest)
		return
	}

	templateID, err := a.emailTemplateStore.Save(r.Context(), input)
	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "modelo de e-mail não encontrado", http.StatusNotFound)
			return
		}
		a.logger.Error("save email template failed", "user_id", user.ID, "template_id", input.ID, "error", err)
		http.Error(w, "não foi possível salvar o modelo de e-mail", http.StatusInternalServerError)
		return
	}

	_, _ = a.pool.Exec(r.Context(), "insert into audit_events(actor_user_id,event_type,resource_type,resource_id,metadata,ip_address,user_agent) values($1,'email_template.saved','email_template',$2,jsonb_build_object('purpose',$3::text,'active',$4::boolean,'default_requested',$5::boolean),$6,$7)", user.ID, templateID, input.Purpose, input.IsActive, input.MakeDefault, requestIP(r), r.UserAgent())
	http.Redirect(w, r, "/admin/email-templates?saved=1&template="+templateID, http.StatusSeeOther)
}

func (a *App) previewEmailTemplate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "dados inválidos", http.StatusBadRequest)
		return
	}
	template := emailtemplates.Template{
		SubjectTemplate: strings.TrimSpace(r.FormValue("subject_template")),
		HTMLTemplate:    strings.TrimSpace(r.FormValue("html_template")),
		TextTemplate:    strings.TrimSpace(r.FormValue("text_template")),
	}
	if err := emailtemplates.Validate(template.SubjectTemplate, template.HTMLTemplate, template.TextTemplate); err != nil {
		http.Error(w, "modelo inválido: "+err.Error(), http.StatusBadRequest)
		return
	}
	draft, err := emailtemplates.Render(template, sampleEmailTemplateVariables())
	if err != nil {
		http.Error(w, "não foi possível renderizar o modelo: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src https: http: data:; style-src 'unsafe-inline'; font-src https: data:; base-uri 'none'; form-action 'none'")
	_, _ = fmt.Fprintf(w, "<!doctype html><html lang=\"pt-BR\"><head><meta charset=\"utf-8\"><title>%s</title></head><body style=\"margin:24px;background:#eef3f6\">%s</body></html>", html.EscapeString(draft.Subject), draft.HTMLBody)
}

func sampleEmailTemplateVariables() map[string]string {
	return map[string]string{
		"brand.logo_url":          "https://viagate.com.br/v1/assets/logo-viagate-color.svg",
		"client.display_name":      "Cliente Exemplo",
		"client.legal_name":        "Cliente Exemplo Ltda.",
		"client.trade_name":        "Cliente Exemplo",
		"client.email":             "contato@cliente.com.br",
		"contact.name":             "Mariana Souza",
		"contact.role":             "Gerente de Logística",
		"contact.email":            "mariana@cliente.com.br",
		"contact.greeting":         "Olá, Mariana Souza.",
		"contact.phone":            "(41) 99999-0000",
		"proposal.title":           "Proposta Comercial ViaGate",
		"proposal.url":             "https://viagate.com.br/p/exemplo",
		"proposal.valid_until":     "31/12/2026",
		"proposal.products_html":   "<table role=\"presentation\" width=\"100%\" cellspacing=\"0\" cellpadding=\"0\"><tr><td style=\"padding:10px 0;border-bottom:1px solid #e2e8ec\"><strong>Cargo Score</strong></td></tr><tr><td style=\"padding:10px 0;border-bottom:1px solid #e2e8ec\"><strong>Cargo Logística</strong><div style=\"margin-top:4px;color:#6f8290;font-size:12px\">Acompanhamento de viagens por aplicativo</div></td></tr></table>",
		"proposal.products_text":   "- Cargo Score\n- Cargo Logística — Acompanhamento de viagens por aplicativo",
		"proposal.products_count":  "2",
		"proposal.requires_policy": "Sim",
		"salesperson.name":         "Jeferson Marques",
		"salesperson.job_title":    "Executivo Comercial",
		"salesperson.email":        "comercial@viagate.com.br",
		"salesperson.phone":        "(41) 99999-0000",
		"salesperson.photo_url":    "https://viagate.com.br/v1/assets/logo-viagate-color.svg",
		"salesperson.linkedin":     "https://www.linkedin.com/",
		"salesperson.instagram":    "https://www.instagram.com/",
	}
}

