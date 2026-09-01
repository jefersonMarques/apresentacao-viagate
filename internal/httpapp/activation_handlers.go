package httpapp

import (
	"fmt"
	"html"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/activation"
	"github.com/jefersonMarques/apresentacao-viagate/internal/notifications"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/security"
	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

const activationAccessTTL = 30 * 24 * time.Hour

func (a *App) activationPage(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	access, err := a.activationStore.AccessByToken(r.Context(), hashToken(token))
	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "Este link de ativação é inválido ou expirou.", http.StatusGone)
			return
		}
		a.logger.Error("load activation access failed", "error", err)
		http.Error(w, "Não foi possível carregar os dados para ativação.", http.StatusInternalServerError)
		return
	}
	message := ""
	switch r.URL.Query().Get("saved") {
	case "finance":
		message = "Dados financeiros salvos."
	case "goods":
		message = "Mercadorias salvas."
	case "users":
		message = "Usuários do sistema salvos."
	case "delegated":
		message = "Convite enviado para a pessoa indicada."
	case "completed":
		message = "Dados recebidos. A ViaGate já pode prosseguir com a implantação."
	}
	render(r.Context(), w, http.StatusOK, templates.ActivationPage(access, token, message, ""))
}

func (a *App) saveActivationSection(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	section := chi.URLParam(r, "section")
	access, err := a.activationStore.AccessByToken(r.Context(), hashToken(token))
	if err != nil {
		http.Error(w, "Link de ativação inválido ou expirado.", http.StatusGone)
		return
	}
	if err := r.ParseForm(); err != nil {
		render(r.Context(), w, http.StatusBadRequest, templates.ActivationPage(access, token, "", "Dados inválidos."))
		return
	}

	profile := access.Profile
	switch section {
	case "finance":
		profile.FinanceResponsibleName = strings.TrimSpace(r.FormValue("finance_name"))
		profile.FinanceResponsibleEmail = strings.ToLower(strings.TrimSpace(r.FormValue("finance_email")))
		if profile.FinanceResponsibleName == "" {
			render(r.Context(), w, http.StatusBadRequest, templates.ActivationPage(access, token, "", "Informe o responsável financeiro."))
			return
		}
		if _, err := mail.ParseAddress(profile.FinanceResponsibleEmail); err != nil {
			render(r.Context(), w, http.StatusBadRequest, templates.ActivationPage(access, token, "", "Informe um e-mail válido para o responsável financeiro."))
			return
		}
		phone, err := cleanPhone(r.FormValue("finance_phone"), true)
		if err != nil {
			render(r.Context(), w, http.StatusBadRequest, templates.ActivationPage(access, token, "", "Informe um telefone válido para o responsável financeiro."))
			return
		}
		profile.FinanceResponsiblePhone = phone
	case "goods":
		profile.Goods = nil
		for _, value := range r.Form["goods"] {
			value = strings.TrimSpace(value)
			if value != "" {
				profile.Goods = append(profile.Goods, value)
			}
		}
		if len(profile.Goods) == 0 {
			render(r.Context(), w, http.StatusBadRequest, templates.ActivationPage(access, token, "", "Informe pelo menos uma mercadoria transportada."))
			return
		}
	case "users":
		profile.SystemUsers = nil
		for _, row := range formValuesAligned(r.Form["system_user_name"], r.Form["system_user_phone"], r.Form["system_user_email"]) {
			if row[0] == "" && row[2] == "" {
				continue
			}
			if row[0] == "" || row[2] == "" {
				render(r.Context(), w, http.StatusBadRequest, templates.ActivationPage(access, token, "", "Preencha nome e e-mail de cada usuário do sistema."))
				return
			}
			if _, err := mail.ParseAddress(row[2]); err != nil {
				render(r.Context(), w, http.StatusBadRequest, templates.ActivationPage(access, token, "", "Há um e-mail inválido na lista de usuários."))
				return
			}
			phone, err := cleanPhone(row[1], false)
			if err != nil {
				render(r.Context(), w, http.StatusBadRequest, templates.ActivationPage(access, token, "", "Há um telefone inválido na lista de usuários."))
				return
			}
			profile.SystemUsers = append(profile.SystemUsers, activation.SystemUser{Name: row[0], Phone: phone, Email: strings.ToLower(row[2])})
		}
		if len(profile.SystemUsers) == 0 {
			render(r.Context(), w, http.StatusBadRequest, templates.ActivationPage(access, token, "", "Adicione pelo menos um usuário que utilizará o sistema."))
			return
		}
	default:
		http.NotFound(w, r)
		return
	}

	if err := a.activationStore.Save(r.Context(), access.TokenID, profile, section); err != nil {
		a.logger.Error("save activation section failed", "error", err, "activation_id", access.Profile.ID, "section", section)
		render(r.Context(), w, http.StatusConflict, templates.ActivationPage(access, token, "", "Não foi possível salvar esta etapa. Ela pode já ter sido concluída."))
		return
	}
	_, _ = a.pool.Exec(r.Context(), `
		insert into audit_events(actor_type,event_type,resource_type,resource_id,ip_address,user_agent,metadata)
		values('customer','activation.section_saved','activation',$1,$2,$3,jsonb_build_object('section',$4::text,'access_type',$5::text))
	`, access.Profile.ID, requestIP(r), r.UserAgent(), section, access.AccessType)
	http.Redirect(w, r, "/activation/"+token+"?saved="+section, http.StatusSeeOther)
}

func (a *App) delegateActivation(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	access, err := a.activationStore.AccessByToken(r.Context(), hashToken(token))
	if err != nil {
		http.Error(w, "Link de ativação inválido ou expirado.", http.StatusGone)
		return
	}
	if access.AccessType != "owner" || access.Section != "all" {
		http.Error(w, "Somente o responsável principal pode encaminhar esta etapa.", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "dados inválidos", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("delegate_name"))
	emailAddress, err := cleanEmail(r.FormValue("delegate_email"), true)
	if err != nil || name == "" {
		render(r.Context(), w, http.StatusBadRequest, templates.ActivationPage(access, token, "", "Informe nome e e-mail válidos da pessoa que continuará o preenchimento."))
		return
	}
	section := strings.TrimSpace(r.FormValue("delegate_section"))
	if section != "all" && section != "finance" && section != "goods" && section != "users" {
		section = "all"
	}
	plain, hash, err := security.RandomToken(32)
	if err != nil {
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}
	expiresAt := time.Now().Add(activationAccessTTL)
	if err := a.activationStore.CreateAccessToken(r.Context(), access.Profile.ID, "delegate", section, name, emailAddress, "", hash, expiresAt); err != nil {
		a.logger.Error("create activation delegation failed", "error", err, "activation_id", access.Profile.ID)
		http.Error(w, "Não foi possível criar o convite.", http.StatusInternalServerError)
		return
	}
	link := strings.TrimRight(a.cfg.BaseURL, "/") + "/activation/" + plain
	htmlBody := fmt.Sprintf(
		"<p>Olá, %s.</p><p>A contratação da <strong>%s</strong> com a ViaGate foi concluída e você foi indicado para complementar os dados necessários para ativação.</p><p><a href=\"%s\">Preencher dados para ativação</a></p><p>O link é individual e expira em 30 dias.</p>",
		html.EscapeString(name), html.EscapeString(access.Profile.LegalName), html.EscapeString(link),
	)
	if err := notifications.EnqueueWithOptions(r.Context(), a.pool, notifications.MessageOptions{
		Kind: "activation_access", ToName: name, ToEmail: emailAddress,
		Subject: "Dados para ativação ViaGate — " + access.Profile.LegalName,
		HTMLBody: htmlBody, TextBody: "Complete os dados para ativação em " + link,
		ExpiresAt: &expiresAt, Sensitive: true,
	}); err != nil {
		a.logger.Error("queue activation delegation failed", "error", err, "activation_id", access.Profile.ID)
		http.Error(w, "Convite criado, mas não foi possível colocá-lo na fila de e-mail.", http.StatusInternalServerError)
		return
	}
	_, _ = a.pool.Exec(r.Context(), `
		insert into audit_events(actor_type,event_type,resource_type,resource_id,ip_address,user_agent,metadata)
		values('customer','activation.delegated','activation',$1,$2,$3,jsonb_build_object('section',$4::text,'delegate_email',$5::text))
	`, access.Profile.ID, requestIP(r), r.UserAgent(), section, emailAddress)
	http.Redirect(w, r, "/activation/"+token+"?saved=delegated", http.StatusSeeOther)
}

func (a *App) submitActivation(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	access, err := a.activationStore.AccessByToken(r.Context(), hashToken(token))
	if err != nil {
		http.Error(w, "Link de ativação inválido ou expirado.", http.StatusGone)
		return
	}
	if err := a.activationStore.Submit(r.Context(), access.TokenID); err != nil {
		access, _ = a.activationStore.AccessByToken(r.Context(), hashToken(token))
		render(r.Context(), w, http.StatusBadRequest, templates.ActivationPage(access, token, "", "Ainda faltam informações. Preencha o contato financeiro, ao menos uma mercadoria e um usuário do sistema."))
		return
	}
	_, _ = a.pool.Exec(r.Context(), `
		insert into audit_events(actor_type,event_type,resource_type,resource_id,ip_address,user_agent,metadata)
		values('customer','activation.submitted','activation',$1,$2,$3,'{}')
	`, access.Profile.ID, requestIP(r), r.UserAgent())
	a.queueActivationCompletedNotification(r, access.Profile.ID)
	a.publishActivationEvent(r.Context(), access.Profile.ID, "activation.submitted", "Dados para ativação enviados")
	a.publishActivationActionRequired(r.Context(), access.Profile.ID)
	http.Redirect(w, r, "/activation/"+token+"?saved=completed", http.StatusSeeOther)
}

func (a *App) issueActivationAccess(w http.ResponseWriter, r *http.Request) {
	signerToken := chi.URLParam(r, "token")
	access, err := a.contractStore.SignerByPublicToken(r.Context(), signerToken)
	if err != nil || access.Signer.Status != "signed" || access.Contract.Status != "signed" {
		http.Error(w, "A ativação ainda não está disponível.", http.StatusConflict)
		return
	}
	profile, err := a.activationStore.EnsureForSignedContract(r.Context(), access.Contract.ID)
	if err != nil {
		a.logger.Error("ensure activation after signature failed", "error", err, "contract_id", access.Contract.ID)
		http.Error(w, "Não foi possível iniciar os dados para ativação.", http.StatusInternalServerError)
		return
	}
	plain, hash, err := security.RandomToken(32)
	if err != nil {
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}
	expiresAt := time.Now().Add(activationAccessTTL)
	if err := a.activationStore.CreateAccessToken(r.Context(), profile.ID, "owner", "all", access.Signer.Name, access.Signer.Email, access.Signer.ID, hash, expiresAt); err != nil {
		http.Error(w, "Não foi possível iniciar os dados para ativação.", http.StatusInternalServerError)
		return
	}
	link := strings.TrimRight(a.cfg.BaseURL, "/") + "/activation/" + plain
	htmlBody := fmt.Sprintf(
		"<p>Olá, %s.</p><p>O contrato da <strong>%s</strong> foi assinado. Para preparar sua operação, complete os dados de ativação.</p><p><a href=\"%s\">Continuar dados para ativação</a></p><p>Você também poderá encaminhar partes do preenchimento para alguém da sua equipe.</p>",
		html.EscapeString(access.Signer.Name), html.EscapeString(profile.LegalName), html.EscapeString(link),
	)
	if err := notifications.EnqueueWithOptions(r.Context(), a.pool, notifications.MessageOptions{
		DedupeKey: "activation-access:" + access.Contract.ID,
		Kind: "activation_access", ToName: access.Signer.Name, ToEmail: access.Signer.Email,
		Subject: "Próximo passo: dados para ativação ViaGate",
		HTMLBody: htmlBody, TextBody: "Continue os dados para ativação em " + link,
		ExpiresAt: &expiresAt, Sensitive: true,
	}); err != nil {
		a.logger.Error("queue activation owner access failed", "error", err, "activation_id", profile.ID)
	}
	http.Redirect(w, r, "/activation/"+plain+"?from=signature", http.StatusSeeOther)
}

func (a *App) queueActivationCompletedNotification(r *http.Request, activationID string) {
	var name, emailAddress, clientName string
	err := a.pool.QueryRow(r.Context(), `
		select u.name,u.email::text,o.legal_name
		from activation_profiles a
		join contracts c on c.id=a.contract_id
		join onboardings o on o.id=c.onboarding_id
		join proposal_acceptances pa on pa.id=o.proposal_acceptance_id
		join proposals p on p.id=pa.proposal_id
		join users u on u.id=p.created_by
		where a.id=$1
	`, activationID).Scan(&name, &emailAddress, &clientName)
	if err != nil {
		a.logger.Error("resolve activation notification recipient failed", "error", err, "activation_id", activationID)
		return
	}
	body := "<p>Os dados para ativação de <strong>" + html.EscapeString(clientName) + "</strong> foram concluídos.</p><p>A implantação interna já pode prosseguir.</p>"
	if err := notifications.EnqueueUnique(r.Context(), a.pool, "activation-completed:"+activationID, name, emailAddress, "Dados para ativação concluídos: "+clientName, body, "Os dados para ativação de "+clientName+" foram concluídos."); err != nil {
		a.logger.Error("queue activation completed notification failed", "error", err, "activation_id", activationID)
	}
}

func (a *App) adminActivations(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	items, err := a.activationStore.ListAdmin(r.Context())
	if err != nil {
		a.logger.Error("load activations failed", "error", err)
		http.Error(w, "Não foi possível carregar as ativações.", http.StatusInternalServerError)
		return
	}
	render(r.Context(), w, http.StatusOK, templates.ActivationsAdminPage(user, items))
}

func (a *App) adminActivationDetail(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	profile, err := a.activationStore.ByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Ativação não encontrada.", http.StatusNotFound)
		return
	}
	render(r.Context(), w, http.StatusOK, templates.ActivationAdminDetailPage(user, profile))
}

func (a *App) adminActivationStatus(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, "dados inválidos", http.StatusBadRequest)
		return
	}
	activationID := chi.URLParam(r, "id")
	status := strings.TrimSpace(r.FormValue("status"))
	if err := a.activationStore.SetInternalStatus(r.Context(), activationID, status); err != nil {
		http.Error(w, "Não foi possível atualizar esta implantação.", http.StatusConflict)
		return
	}
	_, _ = a.pool.Exec(r.Context(), `
		insert into audit_events(actor_user_id,actor_type,event_type,resource_type,resource_id,ip_address,user_agent,metadata)
		values($1,'user','activation.status_changed','activation',$2,$3,$4,jsonb_build_object('status',$5::text))
	`, user.ID, activationID, requestIP(r), r.UserAgent(), status)
	if status == "under_internal_setup" || status == "activated" {
		_, _ = a.pool.Exec(r.Context(), `
			update in_app_notifications
			set read_at=coalesce(read_at,now())
			where dedupe_key=$1 and read_at is null
		`, "activation.action-required:"+activationID)
	}
	if status == "activated" {
		a.publishActivationEvent(r.Context(), activationID, "activation.activated", "Operação liberada")
	}
	http.Redirect(w, r, "/admin/activations/"+activationID, http.StatusSeeOther)
}
