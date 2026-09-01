package httpapp

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
	"github.com/jefersonMarques/apresentacao-viagate/internal/notifications"
	onboardingpkg "github.com/jefersonMarques/apresentacao-viagate/internal/onboarding"
	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

func (a *App) onboardingPage(w http.ResponseWriter, r *http.Request) {
	onboarding, err := a.currentOnboarding(r)
	if err != nil {
		http.Error(w, "Cadastro não encontrado.", http.StatusForbidden)
		return
	}
	proposalURL := ""
	if proposalPath, pathErr := a.proposalPublicPathByOnboarding(r.Context(), onboarding.ID); pathErr == nil {
		proposalURL = proposalPath + "?view=proposal"
	}

	message := ""
	switch {
	case r.URL.Query().Get("accepted") == "1":
		message = "Proposta aceita. Agora confirme os dados necessários para prepararmos o contrato."
	case r.URL.Query().Get("saved") == "company":
		message = "Dados da empresa salvos."
	case r.URL.Query().Get("saved") == "insurance":
		message = "Dados do seguro salvos."
	case r.URL.Query().Get("saved") == "1":
		message = "Dados salvos."
	case r.URL.Query().Get("saved") == "document":
		message = "Apólice enviada com sucesso."
	case r.URL.Query().Get("submitted") == "1":
		message = "Dados e documentos recebidos com sucesso."
	case r.URL.Query().Get("contract_pending") == "1":
		message = "Cadastro aprovado. O contrato será encaminhado por e-mail assim que a preparação for concluída."
	case onboarding.Status == "correction_requested" && strings.TrimSpace(onboarding.ReviewNotes) != "":
		message = "Correção solicitada pela ViaGate: " + onboarding.ReviewNotes
	}

	if onboarding.Status == "submitted" || onboarding.Status == "under_review" || onboarding.Status == "approved" {
		render(r.Context(), w, http.StatusOK, templates.OnboardingStatusPage(onboarding, message, proposalURL))
		return
	}
	hasPolicy, err := a.onboardingStore.HasPolicy(r.Context(), onboarding.ID)
	if err != nil {
		a.logger.Error("load onboarding policy status failed", "onboarding_id", onboarding.ID, "error", err)
		http.Error(w, "Não foi possível carregar a contratação.", http.StatusInternalServerError)
		return
	}
	render(r.Context(), w, http.StatusOK, templates.ContractingJourneyPage(onboarding, hasPolicy, message, "", a.cfg.RequireOnboardingReview, proposalURL))
}

func (a *App) saveOnboarding(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "dados inválidos", http.StatusBadRequest)
		return
	}
	switch r.FormValue("step") {
	case "company":
		a.saveOnboardingCompany(w, r)
		return
	case "insurance":
		a.saveOnboardingInsurance(w, r)
		return
	}

	current, err := a.currentOnboarding(r)
	if err != nil {
		http.Error(w, "acesso negado", http.StatusForbidden)
		return
	}
	cnpj, err := cleanCNPJ(r.FormValue("cnpj"))
	if err != nil {
		a.renderContractingError(w, r, current, err.Error())
		return
	}
	cpf, err := cleanCPF(r.FormValue("responsible_cpf"))
	if err != nil {
		a.renderContractingError(w, r, current, err.Error())
		return
	}
	postalCode, err := cleanPostalCode(r.FormValue("postal_code"), false)
	if err != nil {
		a.renderContractingError(w, r, current, err.Error())
		return
	}
	responsiblePhone, err := cleanPhone(r.FormValue("responsible_phone"), true)
	if err != nil {
		a.renderContractingError(w, r, current, err.Error())
		return
	}

	current.CNPJ = cnpj
	current.LegalName = strings.TrimSpace(r.FormValue("legal_name"))
	current.TradeName = strings.TrimSpace(r.FormValue("trade_name"))
	current.Street = strings.TrimSpace(r.FormValue("street"))
	current.StreetNumber = strings.TrimSpace(r.FormValue("street_number"))
	current.Complement = strings.TrimSpace(r.FormValue("complement"))
	current.District = strings.TrimSpace(r.FormValue("district"))
	current.City = strings.TrimSpace(r.FormValue("city"))
	current.State = strings.ToUpper(strings.TrimSpace(r.FormValue("state")))
	current.PostalCode = postalCode
	current.OperationType = strings.ToLower(strings.TrimSpace(r.FormValue("operation_type")))
	current.Insurer = strings.TrimSpace(r.FormValue("insurer"))
	current.PolicyStartDate = r.FormValue("policy_start_date")
	current.PolicyEndDate = r.FormValue("policy_end_date")
	current.BrokerCompany = strings.TrimSpace(r.FormValue("broker_company"))
	current.BrokerProducer = strings.TrimSpace(r.FormValue("broker_producer"))
	current.CompanyResponsibleName = strings.TrimSpace(r.FormValue("responsible_name"))
	current.CompanyResponsibleCPF = cpf
	current.CompanyResponsiblePhone = responsiblePhone
	current.CompanyResponsibleEmail = strings.TrimSpace(strings.ToLower(r.FormValue("responsible_email")))
	current.CompanyResponsibleRole = strings.TrimSpace(r.FormValue("responsible_role"))
	current.AuthorityDeclared = r.FormValue("responsible_authority") == "1"

	if validationError := validateOnboarding(current); validationError != "" {
		a.renderContractingError(w, r, current, validationError)
		return
	}
	if err := a.onboardingStore.Save(r.Context(), current); err != nil {
		a.logger.Error("save onboarding failed", "error", err)
		a.renderContractingError(w, r, current, "Não foi possível salvar os dados. O cadastro pode já ter sido enviado para revisão.")
		return
	}
	http.Redirect(w, r, "/onboarding/"+current.ID+"?saved=1", http.StatusSeeOther)
}

func validateOnboarding(current domain.Onboarding) string {
	if current.LegalName == "" || current.CompanyResponsibleName == "" || current.CompanyResponsibleEmail == "" || current.CompanyResponsiblePhone == "" || !current.AuthorityDeclared {
		return "Complete os dados obrigatórios e a declaração do responsável."
	}
	if _, err := mail.ParseAddress(current.CompanyResponsibleEmail); err != nil {
		return "O e-mail do responsável é inválido."
	}
	if current.OperationType != "normal" && current.OperationType != "avulsa" {
		return "Selecione um tipo de operação válido."
	}
	if len(current.State) != 2 {
		return "UF inválida."
	}
	if current.PolicyStartDate == "" || current.PolicyEndDate == "" {
		return "Informe a vigência da apólice."
	}
	start, err1 := time.Parse("2006-01-02", current.PolicyStartDate)
	end, err2 := time.Parse("2006-01-02", current.PolicyEndDate)
	if err1 != nil || err2 != nil {
		return "A vigência da apólice é inválida."
	}
	if end.Before(start) {
		return "O fim da vigência da apólice não pode ser anterior ao início."
	}
	return ""
}

func (a *App) lookupCNPJ(w http.ResponseWriter, r *http.Request) {
	current, err := a.currentOnboarding(r)
	if err != nil {
		http.Error(w, "acesso negado", http.StatusForbidden)
		return
	}
	if current.Status != "pending" && current.Status != "in_progress" && current.Status != "correction_requested" {
		http.Error(w, "cadastro não está disponível para edição", http.StatusConflict)
		return
	}
	cnpj, err := cleanCNPJ(chi.URLParam(r, "cnpj"))
	if err != nil {
		http.Error(w, "CNPJ inválido", http.StatusBadRequest)
		return
	}
	company, err := a.registry.Lookup(r.Context(), cnpj)
	if err != nil {
		http.Error(w, "CNPJ não encontrado", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "private, max-age=300")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"cnpj": company.CNPJ, "legal_name": company.LegalName, "trade_name": company.TradeName, "email": company.Email, "phone": company.Phone,
		"street": company.Street, "number": company.Number, "complement": company.Complement, "district": company.District, "city": company.City, "state": company.State, "postal_code": company.PostalCode, "status": company.Status, "primary_cnae": company.PrimaryCNAE,
	})
}

func (a *App) uploadOnboardingDocument(w http.ResponseWriter, r *http.Request) {
	current, err := a.currentOnboarding(r)
	if err != nil {
		http.Error(w, "acesso negado", http.StatusForbidden)
		return
	}
	if current.Status != "pending" && current.Status != "in_progress" && current.Status != "correction_requested" {
		http.Error(w, "documentos bloqueados após o envio do cadastro", http.StatusConflict)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, (15<<20)+1)
	if err := r.ParseMultipartForm(15 << 20); err != nil {
		http.Error(w, "arquivo excede o limite de 15 MB", http.StatusRequestEntityTooLarge)
		return
	}
	file, header, err := r.FormFile("document")
	if err != nil {
		http.Error(w, "selecione a apólice", http.StatusBadRequest)
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, (15<<20)+1))
	if err != nil || len(content) == 0 || len(content) > 15<<20 {
		http.Error(w, "arquivo inválido", http.StatusBadRequest)
		return
	}
	mimeType := http.DetectContentType(content[:min(512, len(content))])
	allowed := map[string]bool{"application/pdf": true, "image/jpeg": true, "image/png": true}
	if !allowed[mimeType] {
		http.Error(w, "formato não permitido; envie PDF, JPG ou PNG", http.StatusUnsupportedMediaType)
		return
	}
	hash := sha256.Sum256(content)
	key := fmt.Sprintf("onboarding/%s/insurance_policy/%d-%s", current.ID, time.Now().UTC().UnixNano(), sanitizeFilename(header.Filename))
	if err := a.storage.Put(r.Context(), key, mimeType, bytes.NewReader(content), int64(len(content))); err != nil {
		a.logger.Error("upload policy to S3 failed", "error", err)
		http.Error(w, "não foi possível armazenar o arquivo", http.StatusInternalServerError)
		return
	}
	document := onboardingpkg.Document{DocumentType: "insurance_policy", StorageKey: key, OriginalFilename: header.Filename, MIMEType: mimeType, SizeBytes: int64(len(content)), SHA256: hash[:]}
	if err := a.onboardingStore.AddDocument(r.Context(), current.ID, document); err != nil {
		_ = a.storage.Delete(r.Context(), key)
		http.Error(w, "não foi possível registrar o arquivo", http.StatusInternalServerError)
		return
	}
	_, _ = a.pool.Exec(r.Context(), `insert into audit_events(actor_type,event_type,resource_type,resource_id,ip_address,user_agent,metadata) values('customer','document.uploaded','onboarding',$1,$2,$3,jsonb_build_object('type','insurance_policy','sha256',$4,'filename',$5))`, current.ID, requestIP(r), r.UserAgent(), fmt.Sprintf("%x", hash[:]), header.Filename)
	http.Redirect(w, r, "/onboarding/"+current.ID+"?saved=document", http.StatusSeeOther)
}

func (a *App) submitOnboarding(w http.ResponseWriter, r *http.Request) {
	current, err := a.currentOnboarding(r)
	if err != nil {
		http.Error(w, "acesso negado", http.StatusForbidden)
		return
	}
	if validationError := validateOnboarding(current); validationError != "" {
		a.renderContractingError(w, r, current, validationError)
		return
	}
	if err := a.onboardingStore.Submit(r.Context(), current.ID); err != nil {
		a.renderContractingError(w, r, current, "Complete as etapas e envie a apólice antes de continuar.")
		return
	}

	_, _ = a.pool.Exec(r.Context(), `
		insert into audit_events(actor_type,event_type,resource_type,resource_id,ip_address,user_agent,metadata)
		values('customer','onboarding.submitted','onboarding',$1,$2,$3,jsonb_build_object('review_required',$4))
	`, current.ID, requestIP(r), r.UserAgent(), a.cfg.RequireOnboardingReview)
	a.queueOnboardingSubmittedNotification(r, current.ID)
	a.publishOnboardingEvent(r.Context(), current.ID)

	if a.cfg.RequireOnboardingReview {
		http.Redirect(w, r, "/onboarding/"+current.ID+"?submitted=1", http.StatusSeeOther)
		return
	}

	command, err := a.pool.Exec(r.Context(), `
		update onboardings
		set status='approved',approved_at=coalesce(approved_at,now()),reviewed_at=now(),
		    review_notes='Aprovação automática: revisão interna desativada.',updated_at=now()
		where id=$1 and status='submitted'
	`, current.ID)
	if err != nil || command.RowsAffected() != 1 {
		a.logger.Error("auto approve onboarding failed", "error", err, "onboarding_id", current.ID)
		http.Redirect(w, r, "/onboarding/"+current.ID+"?contract_pending=1", http.StatusSeeOther)
		return
	}
	_, _ = a.pool.Exec(r.Context(), `insert into audit_events(actor_type,event_type,resource_type,resource_id,metadata) values('system','onboarding.auto_approved','onboarding',$1,'{}')`, current.ID)

	access, _, err := a.ensureContractDelivery(r.Context(), current.ID)
	if err != nil {
		a.logger.Error("automatic contract delivery failed", "error", err, "onboarding_id", current.ID)
		a.queueContractGenerationFailure(r, current.ID, err)
		http.Redirect(w, r, "/onboarding/"+current.ID+"?contract_pending=1", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/sign/"+access.SignerToken, http.StatusSeeOther)
}

func (a *App) queueOnboardingSubmittedNotification(r *http.Request, onboardingID string) {
	var name, emailAddress, proposalTitle, clientName string
	err := a.pool.QueryRow(r.Context(), `
		select u.name,u.email::text,p.title,o.legal_name
		from onboardings o
		join proposal_acceptances pa on pa.id=o.proposal_acceptance_id
		join proposals p on p.id=pa.proposal_id
		join users u on u.id=p.created_by
		where o.id=$1
	`, onboardingID).Scan(&name, &emailAddress, &proposalTitle, &clientName)
	if err != nil {
		a.logger.Error("resolve commercial for onboarding notification failed", "error", err, "onboarding_id", onboardingID)
		return
	}
	htmlBody := fmt.Sprintf("<p>O cadastro de <strong>%s</strong> foi enviado para a contratação da proposta <strong>%s</strong>.</p><p>Acesse o painel para revisar os dados e documentos.</p>", clientName, proposalTitle)
	if err := notifications.EnqueueUnique(r.Context(), a.pool, "onboarding-submitted:"+onboardingID, name, emailAddress, "Cadastro recebido: "+clientName, htmlBody, "O cadastro de "+clientName+" foi enviado e está disponível para revisão."); err != nil {
		a.logger.Error("queue onboarding submitted notification failed", "error", err, "onboarding_id", onboardingID)
	}
}

func (a *App) queueContractGenerationFailure(r *http.Request, onboardingID string, generationErr error) {
	_, err := a.pool.Exec(r.Context(), `
		insert into notification_outbox(dedupe_key,recipient,recipient_name,subject,html_body,text_body)
		select 'contract-generation-failed:'||o.id::text,u.email,u.name,'Falha na geração de contrato: '||p.title,
		       '<p>O cadastro de <strong>'||o.legal_name||'</strong> foi aprovado, mas a preparação do contrato precisa ser tentada novamente pelo painel.</p>',
		       'O cadastro de '||o.legal_name||' foi aprovado, mas a geração do contrato falhou.'
		from onboardings o
		join proposal_acceptances pa on pa.id=o.proposal_acceptance_id
		join proposals p on p.id=pa.proposal_id
		join users u on u.id=p.created_by
		where o.id=$1
		on conflict (dedupe_key) where dedupe_key is not null do nothing
	`, onboardingID)
	if err != nil {
		a.logger.Error("queue contract generation failure notification failed", "error", err, "onboarding_id", onboardingID, "generation_error", generationErr)
	}
}
