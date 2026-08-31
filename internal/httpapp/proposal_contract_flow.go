package httpapp

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	onboardingpkg "github.com/jefersonMarques/apresentacao-viagate/internal/onboarding"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/security"
	"github.com/jefersonMarques/apresentacao-viagate/internal/proposals"
	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

func (a *App) acceptProposalContractFlow(w http.ResponseWriter, r *http.Request, proposal proposals.PublicProposal) {
	if proposalSnapshotString(proposal.Content, "proposal", "contract_template_version_id") == "" {
		a.proposalContractFlowError(w, r, proposal, http.StatusConflict, "Esta proposta ainda não possui um modelo de contrato atribuído. Solicite uma nova versão ao comercial.")
		return
	}

	if err := r.ParseMultipartForm(15 << 20); err != nil {
		a.proposalContractFlowError(w, r, proposal, http.StatusBadRequest, "Não foi possível ler os dados. A apólice deve ter no máximo 15 MB.")
		return
	}

	cpf, err := cleanCPF(r.FormValue("responsible_cpf"))
	if err != nil {
		a.proposalContractFlowError(w, r, proposal, http.StatusBadRequest, err.Error())
		return
	}
	emailAddress, err := cleanEmail(r.FormValue("responsible_email"), true)
	if err != nil {
		a.proposalContractFlowError(w, r, proposal, http.StatusBadRequest, "Informe um e-mail válido para o responsável.")
		return
	}
	phone, err := cleanPhone(r.FormValue("responsible_phone"), true)
	if err != nil {
		a.proposalContractFlowError(w, r, proposal, http.StatusBadRequest, "Informe um telefone válido para o responsável.")
		return
	}
	responsibleName := strings.TrimSpace(r.FormValue("responsible_name"))
	if responsibleName == "" || r.FormValue("authority") != "1" {
		a.proposalContractFlowError(w, r, proposal, http.StatusBadRequest, "Preencha os dados do responsável e confirme a autorização para representar a empresa.")
		return
	}

	sessionID, err := newUUID()
	if err != nil {
		a.proposalContractFlowError(w, r, proposal, http.StatusInternalServerError, "Não foi possível iniciar a contratação.")
		return
	}
	acceptance := proposals.AcceptanceInput{
		Name:              responsibleName,
		Email:             emailAddress,
		CPF:               cpf,
		Phone:             phone,
		Role:              strings.TrimSpace(r.FormValue("responsible_role")),
		AuthorityDeclared: true,
		IPAddress:         requestIP(r),
		UserAgent:         r.UserAgent(),
		SessionID:         sessionID,
	}
	result, err := a.proposalStore.Accept(r.Context(), proposal, acceptance)
	if err != nil {
		a.logger.Error("inline proposal acceptance failed", "proposal_id", proposal.ProposalID, "error", err)
		a.proposalContractFlowError(w, r, proposal, http.StatusConflict, "Não foi possível registrar o aceite. Se esta proposta já foi aceita, use os mesmos dados do responsável.")
		return
	}

	plain, hash, err := security.RandomToken(32)
	if err == nil {
		expires := time.Now().Add(7 * 24 * time.Hour)
		if a.proposalStore.CreateCustomerSession(r.Context(), result.AcceptanceID, hash, requestIP(r), r.UserAgent(), expires) == nil {
			setSecureCookie(w, customerSessionCookie, plain, expires, a.cfg.Environment == "production")
		}
	}

	current, err := a.onboardingStore.ByAcceptance(r.Context(), result.AcceptanceID)
	if err != nil {
		a.proposalContractFlowError(w, r, proposal, http.StatusInternalServerError, "O aceite foi registrado, mas não foi possível preparar o cadastro da contratação.")
		return
	}
	if current.Status == "approved" {
		a.finishInlineContractDelivery(w, r, proposal, current.ID)
		return
	}
	if current.Status == "submitted" {
		if err := a.approveInlineOnboarding(r, current.ID); err != nil {
			a.proposalContractFlowError(w, r, proposal, http.StatusInternalServerError, "Os dados foram recebidos, mas não foi possível preparar o contrato agora. Tente novamente.")
			return
		}
		a.finishInlineContractDelivery(w, r, proposal, current.ID)
		return
	}

	cnpj, err := cleanCNPJ(r.FormValue("cnpj"))
	if err != nil {
		a.proposalContractFlowError(w, r, proposal, http.StatusBadRequest, err.Error())
		return
	}
	postalCode, err := cleanPostalCode(r.FormValue("postal_code"), true)
	if err != nil {
		a.proposalContractFlowError(w, r, proposal, http.StatusBadRequest, "Informe um CEP válido.")
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
	current.PolicyStartDate = strings.TrimSpace(r.FormValue("policy_start_date"))
	current.PolicyEndDate = strings.TrimSpace(r.FormValue("policy_end_date"))
	current.BrokerCompany = strings.TrimSpace(r.FormValue("broker_company"))
	current.BrokerProducer = strings.TrimSpace(r.FormValue("broker_producer"))
	current.CompanyResponsibleName = responsibleName
	current.CompanyResponsibleCPF = cpf
	current.CompanyResponsiblePhone = phone
	current.CompanyResponsibleEmail = emailAddress
	current.CompanyResponsibleRole = acceptance.Role
	current.AuthorityDeclared = true

	if current.LegalName == "" || current.Street == "" || current.StreetNumber == "" || current.City == "" || len(current.State) != 2 || current.Insurer == "" {
		a.proposalContractFlowError(w, r, proposal, http.StatusBadRequest, "Complete razão social, endereço e dados da seguradora para gerar o contrato.")
		return
	}
	if validationError := validateOnboarding(current); validationError != "" {
		a.proposalContractFlowError(w, r, proposal, http.StatusBadRequest, validationError)
		return
	}
	if err := a.onboardingStore.Save(r.Context(), current); err != nil {
		a.logger.Error("save inline onboarding failed", "onboarding_id", current.ID, "error", err)
		a.proposalContractFlowError(w, r, proposal, http.StatusConflict, "Não foi possível salvar os dados da contratação.")
		return
	}

	hasPolicy, policyErr := a.onboardingStore.HasPolicy(r.Context(), current.ID)
	if policyErr != nil {
		a.proposalContractFlowError(w, r, proposal, http.StatusInternalServerError, "Não foi possível validar a apólice.")
		return
	}
	file, header, fileErr := r.FormFile("insurance_policy")
	if fileErr == nil {
		defer file.Close()
		if err := a.storeInlineInsurancePolicy(r, current.ID, file, header.Filename); err != nil {
			a.proposalContractFlowError(w, r, proposal, http.StatusBadRequest, err.Error())
			return
		}
		hasPolicy = true
	}
	if !hasPolicy {
		a.proposalContractFlowError(w, r, proposal, http.StatusBadRequest, "Envie a apólice de seguros em PDF, JPG ou PNG.")
		return
	}

	if err := a.onboardingStore.Submit(r.Context(), current.ID); err != nil {
		a.proposalContractFlowError(w, r, proposal, http.StatusBadRequest, err.Error())
		return
	}
	a.queueOnboardingSubmittedNotification(r, current.ID)
	_, _ = a.pool.Exec(r.Context(), `
		insert into audit_events(actor_type,event_type,resource_type,resource_id,ip_address,user_agent,metadata)
		values('customer','onboarding.submitted','onboarding',$1,$2,$3,jsonb_build_object('source','proposal_modal'))
	`, current.ID, requestIP(r), r.UserAgent())

	if err := a.approveInlineOnboarding(r, current.ID); err != nil {
		a.proposalContractFlowError(w, r, proposal, http.StatusInternalServerError, "Os dados foram recebidos, mas não foi possível preparar o contrato agora. Tente novamente.")
		return
	}
	a.finishInlineContractDelivery(w, r, proposal, current.ID)
}

func (a *App) approveInlineOnboarding(r *http.Request, onboardingID string) error {
	command, err := a.pool.Exec(r.Context(), `
		update onboardings
		set status='approved',approved_at=coalesce(approved_at,now()),reviewed_at=now(),
		    review_notes='Aprovação automática após aceite e preenchimento completo na proposta.',updated_at=now()
		where id=$1 and status='submitted'
	`, onboardingID)
	if err != nil {
		a.logger.Error("approve inline onboarding failed", "onboarding_id", onboardingID, "error", err)
		return err
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("onboarding is not submitted")
	}
	_, _ = a.pool.Exec(r.Context(), `
		insert into audit_events(actor_type,event_type,resource_type,resource_id,metadata)
		values('system','onboarding.auto_approved','onboarding',$1,jsonb_build_object('source','proposal_modal'))
	`, onboardingID)
	return nil
}

func (a *App) storeInlineInsurancePolicy(r *http.Request, onboardingID string, file io.Reader, filename string) error {
	content, err := io.ReadAll(io.LimitReader(file, (15<<20)+1))
	if err != nil || len(content) == 0 || len(content) > 15<<20 {
		return fmt.Errorf("A apólice deve ter no máximo 15 MB.")
	}
	mimeType := http.DetectContentType(content[:min(512, len(content))])
	allowed := map[string]bool{"application/pdf": true, "image/jpeg": true, "image/png": true}
	if !allowed[mimeType] {
		return fmt.Errorf("Envie a apólice em PDF, JPG ou PNG.")
	}
	hash := sha256.Sum256(content)
	key := fmt.Sprintf("onboarding/%s/insurance_policy/%d-%s", onboardingID, time.Now().UTC().UnixNano(), sanitizeFilename(filename))
	if err := a.storage.Put(r.Context(), key, mimeType, bytes.NewReader(content), int64(len(content))); err != nil {
		a.logger.Error("inline policy upload failed", "onboarding_id", onboardingID, "error", err)
		return fmt.Errorf("Não foi possível armazenar a apólice. Tente novamente.")
	}
	document := onboardingpkg.Document{DocumentType: "insurance_policy", StorageKey: key, OriginalFilename: filename, MIMEType: mimeType, SizeBytes: int64(len(content)), SHA256: hash[:]}
	if err := a.onboardingStore.AddDocument(r.Context(), onboardingID, document); err != nil {
		_ = a.storage.Delete(r.Context(), key)
		return fmt.Errorf("Não foi possível registrar a apólice.")
	}
	_, _ = a.pool.Exec(r.Context(), `
		insert into audit_events(actor_type,event_type,resource_type,resource_id,ip_address,user_agent,metadata)
		values(
			'customer','document.uploaded','onboarding',$1,$2,$3,
			jsonb_build_object(
				'type','insurance_policy',
				'sha256',$4::text,
				'filename',$5::text,
				'source','proposal_modal'
			)
		)
	`, onboardingID, requestIP(r), r.UserAgent(), fmt.Sprintf("%x", hash[:]), filename)
	return nil
}

func (a *App) finishInlineContractDelivery(w http.ResponseWriter, r *http.Request, proposal proposals.PublicProposal, onboardingID string) {
	access, _, err := a.ensureContractDelivery(r.Context(), onboardingID)
	if err != nil {
		a.logger.Error("inline contract delivery failed", "onboarding_id", onboardingID, "error", err)
		a.queueContractGenerationFailure(r, onboardingID, err)
		a.proposalContractFlowError(w, r, proposal, http.StatusInternalServerError, "Os dados foram concluídos, mas o contrato não pôde ser gerado agora. Tente novamente em instantes.")
		return
	}
	signURL := "/sign/" + access.SignerToken
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(map[string]string{"sign_url": signURL})
		return
	}
	http.Redirect(w, r, signURL, http.StatusSeeOther)
}

func (a *App) proposalContractFlowError(w http.ResponseWriter, r *http.Request, proposal proposals.PublicProposal, status int, message string) {
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
		return
	}
	render(r.Context(), w, status, templates.PublicProposalViewerPage(proposal, message))
}

func wantsJSON(r *http.Request) bool {
	return strings.Contains(strings.ToLower(r.Header.Get("Accept")), "application/json")
}

func proposalSnapshotString(content map[string]any, section, key string) string {
	group, _ := content[section].(map[string]any)
	value, _ := group[key].(string)
	return strings.TrimSpace(value)
}
