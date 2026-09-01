package httpapp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/notifications"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/security"
	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

func (a *App) signaturePage(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	access, err := a.contractStore.SignerByPublicToken(r.Context(), token)
	if err != nil {
		http.Error(w, "Link de assinatura inválido.", http.StatusNotFound)
		return
	}

	message := ""
	if r.URL.Query().Get("otp") == "sent" {
		message = "Código solicitado. O envio para o e-mail do responsável pode levar alguns segundos."
	}
	if r.URL.Query().Get("signed") == "1" {
		message = "Contrato assinado com sucesso."
	}

	_, _ = a.pool.Exec(r.Context(), `
		insert into signature_events(contract_id,contract_signer_id,event_type,document_hash,ip_address,user_agent,metadata)
		values($1,$2,'contract.viewed',$3,$4,$5,'{}')
	`, access.Contract.ID, access.Signer.ID, access.Contract.DocumentSHA256, requestIP(r), r.UserAgent())
	a.publishContractEvent(r.Context(), access.Contract.ID, "contract.opened", "Contrato aberto", dailyEventDedupe(access.Contract.ID))

	render(r.Context(), w, http.StatusOK, templates.SignatureContractPage(access, token, message, ""))
}

func (a *App) sendSignatureOTP(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	access, err := a.contractStore.SignerByPublicToken(r.Context(), token)
	if err != nil {
		if wantsJSON(r) {
			writeJSONError(w, http.StatusNotFound, "Link de assinatura inválido.")
			return
		}
		http.Error(w, "Link inválido", http.StatusNotFound)
		return
	}
	if access.Signer.Status == "signed" {
		if wantsJSON(r) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "signed": true})
			return
		}
		http.Redirect(w, r, "/sign/"+token+"?signed=1", http.StatusSeeOther)
		return
	}

	var recent int
	_ = a.pool.QueryRow(r.Context(), `
		select count(*)
		from signature_challenges
		where contract_signer_id=$1 and created_at>now()-interval '10 minutes'
	`, access.Signer.ID).Scan(&recent)
	if recent >= 3 {
		if wantsJSON(r) {
			writeJSONError(w, http.StatusTooManyRequests, "Aguarde alguns minutos antes de solicitar outro código.")
			return
		}
		render(r.Context(), w, http.StatusTooManyRequests, templates.SignatureContractPage(access, token, "", "Aguarde alguns minutos antes de solicitar outro código."))
		return
	}

	otp, hash, err := security.RandomOTP()
	if err != nil {
		if wantsJSON(r) {
			writeJSONError(w, http.StatusInternalServerError, "Não foi possível gerar o código.")
			return
		}
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}
	expiresAt := time.Now().Add(a.cfg.Session.SignatureOTPTTL)
	if err := a.contractStore.CreateChallenge(r.Context(), access.Signer.ID, hash, expiresAt); err != nil {
		if wantsJSON(r) {
			writeJSONError(w, http.StatusInternalServerError, "Não foi possível gerar o código.")
			return
		}
		http.Error(w, "não foi possível gerar o código", http.StatusInternalServerError)
		return
	}

	_, _ = a.pool.Exec(r.Context(), `
		update notification_outbox
		set status='expired',processing_at=null,last_error='replaced by a newer OTP',
		    html_body='[conteúdo sensível substituído]',text_body='[conteúdo sensível substituído]'
		where dedupe_key like $1 and status in ('pending','processing')
	`, "signature-otp:"+access.Signer.ID+":%")

	_, _ = a.pool.Exec(r.Context(), `
		insert into signature_events(contract_id,contract_signer_id,event_type,document_hash,ip_address,user_agent,metadata)
		values($1,$2,'otp.requested',$3,$4,$5,jsonb_build_object('channel','email'))
	`, access.Contract.ID, access.Signer.ID, access.Contract.DocumentSHA256, requestIP(r), r.UserAgent())

	contractLink := strings.TrimRight(a.cfg.BaseURL, "/") + "/sign/" + token
	htmlBody := fmt.Sprintf(
		"<p>Seu código de confirmação para assinatura do contrato ViaGate é:</p><p style=\"font-size:28px;font-weight:800;letter-spacing:4px\">%s</p><p>O código expira em %s.</p><p><a href=\"%s\">Voltar ao contrato</a></p>",
		otp,
		a.cfg.Session.SignatureOTPTTL,
		contractLink,
	)
	if err := notifications.EnqueueWithOptions(r.Context(), a.pool, notifications.MessageOptions{
		DedupeKey: fmt.Sprintf("signature-otp:%s:%d", access.Signer.ID, time.Now().UTC().UnixNano()),
		Kind: "signature_otp", ToName: access.Signer.Name, ToEmail: access.Signer.Email,
		Subject: "Código de assinatura ViaGate", HTMLBody: htmlBody, TextBody: "Código: " + otp + "\nVoltar ao contrato: " + contractLink,
		ExpiresAt: &expiresAt, Sensitive: true,
	}); err != nil {
		a.logger.Error("enqueue signature OTP failed", "signer_id", access.Signer.ID, "recipient", access.Signer.Email, "error", err)
		if wantsJSON(r) {
			writeJSONError(w, http.StatusInternalServerError, "Não foi possível enviar o código.")
			return
		}
		http.Error(w, "não foi possível enviar o código", http.StatusInternalServerError)
		return
	}

	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"message": "Código enviado para " + maskEmail(access.Signer.Email),
			"retry_after_seconds": 60,
		})
		return
	}
	http.Redirect(w, r, "/sign/"+token+"?otp=sent", http.StatusSeeOther)
}

func (a *App) confirmSignature(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	access, err := a.contractStore.SignerByPublicToken(r.Context(), token)
	if err != nil {
		http.Error(w, "Link inválido", http.StatusNotFound)
		return
	}
	if access.Signer.Status == "signed" {
		http.Redirect(w, r, "/sign/"+token+"?signed=1", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "dados inválidos", http.StatusBadRequest)
		return
	}
	if r.FormValue("consent") != "1" {
		render(r.Context(), w, http.StatusBadRequest, templates.SignatureContractPage(access, token, "", "Confirme que leu e concorda em assinar o contrato."))
		return
	}

	otp := digits(r.FormValue("otp"))
	if len(otp) != 6 {
		render(r.Context(), w, http.StatusBadRequest, templates.SignatureContractPage(access, token, "", "Informe o código de 6 dígitos."))
		return
	}

	sessionID, err := newUUID()
	if err != nil {
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}
	contractID, fullySigned, err := a.contractStore.ConfirmAndSign(
		r.Context(), access.Signer.ID, security.HashToken(otp), access.Contract.DocumentSHA256,
		sessionID, requestIP(r), r.UserAgent(),
	)
	if err != nil {
		a.logger.Warn("contract confirmation failed", "signer_id", access.Signer.ID, "error", err)
		render(r.Context(), w, http.StatusUnauthorized, templates.SignatureContractPage(access, token, "", "Código inválido ou expirado. Solicite um novo código se necessário."))
		return
	}

	if fullySigned {
		_, _ = a.pool.Exec(r.Context(), `
			insert into notification_outbox(dedupe_key,recipient,recipient_name,subject,html_body,text_body)
			select 'contract-signed:'||c.id::text,u.email,u.name,'Contrato assinado: '||p.title,
			       '<p>O contrato de <strong>'||cl.legal_name||'</strong> foi assinado. O pacote técnico de evidências está sendo finalizado automaticamente.</p>',
			       'O contrato de '||cl.legal_name||' foi assinado.'
			from contracts c
			join onboardings o on o.id=c.onboarding_id
			join proposal_acceptances pa on pa.id=o.proposal_acceptance_id
			join proposals p on p.id=pa.proposal_id
			join clients cl on cl.id=o.client_id
			join users u on u.id=p.created_by
			where c.id=$1
			on conflict (dedupe_key) where dedupe_key is not null do nothing
		`, contractID)
		a.publishContractEvent(r.Context(), contractID, "contract.signed", "Contrato assinado", contractID)
		if err := a.queuePostSignatureActivation(r.Context(), access); err != nil {
			a.logger.Error("queue post-signature activation failed", "contract_id", contractID, "signer_id", access.Signer.ID, "error", err)
		}
	}

	http.Redirect(w, r, "/sign/"+token+"?signed=1", http.StatusSeeOther)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func maskEmail(value string) string {
	parts := strings.SplitN(strings.TrimSpace(value), "@", 2)
	if len(parts) != 2 || parts[0] == "" {
		return "o e-mail informado"
	}
	local := parts[0]
	visible := local[:1]
	if len(local) > 2 {
		visible += strings.Repeat("*", min(4, len(local)-2)) + local[len(local)-1:]
	} else {
		visible += "***"
	}
	return visible + "@" + parts[1]
}

func (a *App) downloadSignedContract(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	access, err := a.contractStore.SignerByPublicToken(r.Context(), token)
	if err != nil {
		http.Error(w, "Link inválido", http.StatusNotFound)
		return
	}
	filename := "contrato-viagate.pdf"
	if access.Contract.Status != "signed" {
		filename = "contrato-para-assinatura-viagate.pdf"
	}
	a.redirectPrivateArtifact(w, r, access.Contract.PDFStorageKey, filename)
}

func (a *App) downloadSignatureEvidence(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	keys, err := a.contractStore.ArtifactKeysBySignerToken(r.Context(), token)
	if err != nil {
		http.Error(w, "Evidências indisponíveis", http.StatusNotFound)
		return
	}
	if !keys.Finalized || keys.EvidenceKey == "" {
		http.Error(w, "O relatório de evidências ainda está sendo finalizado.", http.StatusConflict)
		return
	}
	a.redirectPrivateArtifact(w, r, keys.EvidenceKey, "evidencias-assinatura-viagate.pdf")
}

func (a *App) downloadSignaturePackage(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	keys, err := a.contractStore.ArtifactKeysBySignerToken(r.Context(), token)
	if err != nil {
		http.Error(w, "Pacote indisponível", http.StatusNotFound)
		return
	}
	if !keys.Finalized || keys.PackageKey == "" {
		http.Error(w, "O pacote final ainda está sendo finalizado.", http.StatusConflict)
		return
	}
	a.redirectPrivateArtifact(w, r, keys.PackageKey, "pacote-assinatura-viagate.zip")
}

func (a *App) redirectPrivateArtifact(w http.ResponseWriter, r *http.Request, key, filename string) {
	if strings.TrimSpace(key) == "" {
		http.Error(w, "Arquivo indisponível", http.StatusNotFound)
		return
	}
	downloadURL, err := a.storage.SignedAttachmentURL(r.Context(), key, filename, a.cfg.S3.DownloadTTL)
	if err != nil {
		a.logger.Error("presign private artifact failed", "error", err)
		http.Error(w, "Não foi possível disponibilizar o arquivo.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, downloadURL.String(), http.StatusTemporaryRedirect)
}
