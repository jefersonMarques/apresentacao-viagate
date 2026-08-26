package httpapp

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/notifications"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/security"
	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

func (a *App) signaturePage(w http.ResponseWriter,r *http.Request) {
	token:=chi.URLParam(r,"token")
	access,err:=a.contractStore.SignerByPublicToken(r.Context(),token);if err!=nil{http.Error(w,"Link de assinatura inválido.",http.StatusNotFound);return}
	message:=""
	if r.URL.Query().Get("otp")=="sent"{message="Código enviado para o e-mail do responsável."}
	if r.URL.Query().Get("signed")=="1"{message="Contrato assinado com sucesso."}
	if access.Contract.Status=="signed"&&access.Contract.FinalizedAt==nil{
		if err:=a.contractFinalizer.Finalize(r.Context(),access.Contract.ID);err!=nil{a.logger.Error("contract evidence finalization failed","error",err,"contract_id",access.Contract.ID)}else{access,_=a.contractStore.SignerByPublicToken(r.Context(),token)}
	}
	_,_ = a.pool.Exec(r.Context(),`insert into signature_events(contract_id,contract_signer_id,event_type,document_hash,ip_address,user_agent,metadata) values($1,$2,'contract.viewed',$3,$4,$5,'{}')`,access.Contract.ID,access.Signer.ID,access.Contract.DocumentSHA256,requestIP(r),r.UserAgent())
	render(r.Context(),w,http.StatusOK,templates.SignatureContractPage(access,token,message,""))
}

func (a *App) sendSignatureOTP(w http.ResponseWriter,r *http.Request) {
	token:=chi.URLParam(r,"token")
	access,err:=a.contractStore.SignerByPublicToken(r.Context(),token);if err!=nil{http.Error(w,"Link inválido",http.StatusNotFound);return}
	if access.Signer.Status=="signed"{http.Redirect(w,r,"/sign/"+token+"?signed=1",http.StatusSeeOther);return}
	var recent int
	_ = a.pool.QueryRow(r.Context(),`select count(*) from signature_challenges where contract_signer_id=$1 and created_at>now()-interval '10 minutes'`,access.Signer.ID).Scan(&recent)
	if recent>=3{render(r.Context(),w,http.StatusTooManyRequests,templates.SignatureContractPage(access,token,"","Aguarde alguns minutos antes de solicitar outro código."));return}
	otp,hash,err:=security.RandomOTP();if err!=nil{http.Error(w,"erro interno",http.StatusInternalServerError);return}
	if err:=a.contractStore.CreateChallenge(r.Context(),access.Signer.ID,hash,time.Now().Add(a.cfg.Session.SignatureOTPTTL));err!=nil{http.Error(w,"não foi possível gerar o código",http.StatusInternalServerError);return}
	_,_ = a.pool.Exec(r.Context(),`insert into identity_verifications(contract_signer_id,mode,status,provider) values($1,'email_otp','pending','brevo')`,access.Signer.ID)
	_,_ = a.pool.Exec(r.Context(),`insert into signature_events(contract_id,contract_signer_id,event_type,document_hash,ip_address,user_agent,metadata) values($1,$2,'otp.requested',$3,$4,$5,jsonb_build_object('channel','email'))`,access.Contract.ID,access.Signer.ID,access.Contract.DocumentSHA256,requestIP(r),r.UserAgent())
	htmlBody:=fmt.Sprintf("<p>Seu código de confirmação para assinatura do contrato ViaGate é:</p><p style=\"font-size:28px;font-weight:800;letter-spacing:4px\">%s</p><p>O código expira em %s.</p>",otp,a.cfg.Session.SignatureOTPTTL)
	if err:=notifications.Enqueue(r.Context(),a.pool,access.Signer.Name,access.Signer.Email,"Código de assinatura ViaGate",htmlBody,"Código: "+otp);err!=nil{http.Error(w,"não foi possível enviar o código",http.StatusInternalServerError);return}
	http.Redirect(w,r,"/sign/"+token+"?otp=sent",http.StatusSeeOther)
}

func (a *App) confirmSignature(w http.ResponseWriter,r *http.Request) {
	token:=chi.URLParam(r,"token")
	access,err:=a.contractStore.SignerByPublicToken(r.Context(),token);if err!=nil{http.Error(w,"Link inválido",http.StatusNotFound);return}
	if access.Signer.Status=="signed"{http.Redirect(w,r,"/sign/"+token+"?signed=1",http.StatusSeeOther);return}
	if err:=r.ParseForm();err!=nil{http.Error(w,"dados inválidos",http.StatusBadRequest);return}
	if r.FormValue("consent")!="1"{render(r.Context(),w,http.StatusBadRequest,templates.SignatureContractPage(access,token,"","Confirme que leu e concorda em assinar o contrato."));return}
	otp:=digits(r.FormValue("otp"));if len(otp)!=6{render(r.Context(),w,http.StatusBadRequest,templates.SignatureContractPage(access,token,"","Informe o código de 6 dígitos."));return}
	if err:=a.contractStore.VerifyChallenge(r.Context(),access.Signer.ID,security.HashToken(otp));err!=nil{render(r.Context(),w,http.StatusUnauthorized,templates.SignatureContractPage(access,token,"","Código inválido ou expirado."));return}
	sessionID,err:=newUUID();if err!=nil{http.Error(w,"erro interno",http.StatusInternalServerError);return}
	_,_ = a.pool.Exec(r.Context(),`update identity_verifications set status='verified',verified_at=now(),evidence=evidence||jsonb_build_object('session_id',$2,'ip',$3,'user_agent',$4) where contract_signer_id=$1 and mode='email_otp' and status='pending'`,access.Signer.ID,sessionID,requestIP(r),r.UserAgent())
	_,_ = a.pool.Exec(r.Context(),`insert into signature_events(contract_id,contract_signer_id,event_type,document_hash,ip_address,user_agent,session_id,metadata) values($1,$2,'otp.verified',$3,$4,$5,$6,jsonb_build_object('channel','email'))`,access.Contract.ID,access.Signer.ID,access.Contract.DocumentSHA256,requestIP(r),r.UserAgent(),sessionID)
	contractID,fullySigned,err:=a.contractStore.Sign(r.Context(),access.Signer.ID,access.Contract.DocumentSHA256,sessionID,requestIP(r),r.UserAgent())
	if err!=nil{a.logger.Error("contract sign failed","error",err);render(r.Context(),w,http.StatusConflict,templates.SignatureContractPage(access,token,"","Não foi possível concluir a assinatura. O documento pode ter sido alterado ou a validação expirou."));return}
	if fullySigned{
		if err:=a.contractFinalizer.Finalize(r.Context(),contractID);err!=nil{a.logger.Error("contract evidence finalization failed","error",err,"contract_id",contractID)}
		_,_ = a.pool.Exec(r.Context(),`
			insert into notification_outbox(recipient,recipient_name,subject,html_body,text_body)
			select u.email,u.name,'Contrato assinado: '||p.title,
			       '<p>O contrato de <strong>'||cl.legal_name||'</strong> foi assinado e a trilha de evidências foi registrada.</p>',
			       'O contrato de '||cl.legal_name||' foi assinado.'
			from contracts c join onboardings o on o.id=c.onboarding_id join proposal_acceptances pa on pa.id=o.proposal_acceptance_id join proposals p on p.id=pa.proposal_id join clients cl on cl.id=o.client_id join users u on u.id=p.created_by where c.id=$1
		`,contractID)
	}
	http.Redirect(w,r,"/sign/"+token+"?signed=1",http.StatusSeeOther)
}

func (a *App) downloadSignedContract(w http.ResponseWriter,r *http.Request) {
	token:=chi.URLParam(r,"token")
	access,err:=a.contractStore.SignerByPublicToken(r.Context(),token);if err!=nil{http.Error(w,"Link inválido",http.StatusNotFound);return}
	a.redirectPrivateArtifact(w,r,access.Contract.PDFStorageKey)
}

func (a *App) downloadSignatureEvidence(w http.ResponseWriter,r *http.Request) {
	token:=chi.URLParam(r,"token")
	keys,err:=a.contractStore.ArtifactKeysBySignerToken(r.Context(),token);if err!=nil{http.Error(w,"Evidências indisponíveis",http.StatusNotFound);return}
	if !keys.Finalized||keys.EvidenceKey==""{http.Error(w,"O relatório de evidências ainda está sendo finalizado.",http.StatusConflict);return}
	a.redirectPrivateArtifact(w,r,keys.EvidenceKey)
}

func (a *App) downloadSignaturePackage(w http.ResponseWriter,r *http.Request) {
	token:=chi.URLParam(r,"token")
	keys,err:=a.contractStore.ArtifactKeysBySignerToken(r.Context(),token);if err!=nil{http.Error(w,"Pacote indisponível",http.StatusNotFound);return}
	if !keys.Finalized||keys.PackageKey==""{http.Error(w,"O pacote final ainda está sendo finalizado.",http.StatusConflict);return}
	a.redirectPrivateArtifact(w,r,keys.PackageKey)
}

func (a *App) redirectPrivateArtifact(w http.ResponseWriter,r *http.Request,key string) {
	if strings.TrimSpace(key)==""{http.Error(w,"Arquivo indisponível",http.StatusNotFound);return}
	url,err:=a.storage.SignedDownloadURL(r.Context(),key,a.cfg.S3.DownloadTTL);if err!=nil{a.logger.Error("presign private artifact failed","error",err);http.Error(w,"Não foi possível disponibilizar o arquivo.",http.StatusInternalServerError);return}
	w.Header().Set("Cache-Control","no-store")
	http.Redirect(w,r,url.String(),http.StatusTemporaryRedirect)
}
