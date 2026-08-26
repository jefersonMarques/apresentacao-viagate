package httpapp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
	"github.com/jefersonMarques/apresentacao-viagate/internal/notifications"
	onboardingpkg "github.com/jefersonMarques/apresentacao-viagate/internal/onboarding"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/security"
	"github.com/jefersonMarques/apresentacao-viagate/internal/proposals"
	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

const customerAcceptanceContextKey contextKey = "customer_acceptance_id"

func (a *App) publicProposalPage(w http.ResponseWriter,r *http.Request) {
	token:=chi.URLParam(r,"token")
	proposal,err:=a.proposalStore.PublicByToken(r.Context(),token)
	if err!=nil {
		if err==pgx.ErrNoRows { http.Error(w,"Proposta não encontrada.",http.StatusNotFound);return }
		http.Error(w,"Esta proposta não está mais disponível.",http.StatusGone);return
	}
	render(r.Context(),w,http.StatusOK,templates.PublicProposalPage(proposal,""))
}

func (a *App) acceptProposal(w http.ResponseWriter,r *http.Request) {
	token:=chi.URLParam(r,"token")
	proposal,err:=a.proposalStore.PublicByToken(r.Context(),token)
	if err!=nil { http.Error(w,"Proposta não disponível.",http.StatusGone);return }
	if err:=r.ParseForm();err!=nil { http.Error(w,"dados inválidos",http.StatusBadRequest);return }
	cpf,err:=cleanCPF(r.FormValue("cpf"))
	if err!=nil { render(r.Context(),w,http.StatusBadRequest,templates.PublicProposalPage(proposal,err.Error()));return }
	if r.FormValue("authority")!="1" { render(r.Context(),w,http.StatusBadRequest,templates.PublicProposalPage(proposal,"É necessário confirmar a autorização para representar a empresa."));return }
	sessionID,err:=newUUID();if err!=nil{http.Error(w,"erro interno",http.StatusInternalServerError);return}
	input:=proposals.AcceptanceInput{
		Name:strings.TrimSpace(r.FormValue("name")),
		Email:strings.TrimSpace(strings.ToLower(r.FormValue("email"))),
		CPF:cpf,
		Phone:strings.TrimSpace(r.FormValue("phone")),
		Role:strings.TrimSpace(r.FormValue("role")),
		AuthorityDeclared:true,
		IPAddress:requestIP(r),UserAgent:r.UserAgent(),SessionID:sessionID,
	}
	if input.Name==""||input.Email==""||input.Phone=="" { render(r.Context(),w,http.StatusBadRequest,templates.PublicProposalPage(proposal,"Preencha todos os dados do responsável."));return }
	result,err:=a.proposalStore.Accept(r.Context(),proposal,input)
	if err!=nil { a.logger.Error("proposal acceptance failed","error",err);render(r.Context(),w,http.StatusBadRequest,templates.PublicProposalPage(proposal,"Não foi possível registrar o aceite."));return }

	plain,hash,err:=security.RandomToken(32);if err!=nil{http.Error(w,"erro interno",http.StatusInternalServerError);return}
	expires:=time.Now().Add(7*24*time.Hour)
	if err:=a.proposalStore.CreateCustomerSession(r.Context(),result.AcceptanceID,hash,requestIP(r),r.UserAgent(),expires);err!=nil{http.Error(w,"não foi possível iniciar o cadastro",http.StatusInternalServerError);return}
	setSecureCookie(w,customerSessionCookie,plain,expires,a.cfg.Environment=="production")

	_,_ = a.pool.Exec(r.Context(),`
		insert into notification_outbox(recipient,recipient_name,subject,html_body,text_body)
		select u.email,u.name,'Proposta aceita: '||p.title,
		       '<p>A proposta de <strong>'||c.legal_name||'</strong> foi aceita por '||$2||'.</p>',
		       'A proposta de '||c.legal_name||' foi aceita por '||$2||'.'
		from proposals p join clients c on c.id=p.client_id join users u on u.id=p.created_by where p.id=$1
	`,proposal.ProposalID,input.Name)
	http.Redirect(w,r,"/onboarding/"+result.OnboardingID,http.StatusSeeOther)
}

func (a *App) customerSessionRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		cookie,err:=r.Cookie(customerSessionCookie)
		if err!=nil||cookie.Value==""{http.Error(w,"Sessão do cliente expirada. Abra novamente o link da proposta.",http.StatusUnauthorized);return}
		acceptanceID,err:=a.proposalStore.CustomerSessionAcceptance(r.Context(),hashToken(cookie.Value))
		if err!=nil{clearCookie(w,customerSessionCookie);http.Error(w,"Sessão do cliente expirada.",http.StatusUnauthorized);return}
		ctx:=context.WithValue(r.Context(),customerAcceptanceContextKey,acceptanceID)
		next.ServeHTTP(w,r.WithContext(ctx))
	})
}

func (a *App) currentOnboarding(r *http.Request)(domain.Onboarding,error){
	acceptanceID,ok:=r.Context().Value(customerAcceptanceContextKey).(string)
	if !ok||acceptanceID==""{return domain.Onboarding{},fmt.Errorf("customer session unavailable")}
	onboarding,err:=a.onboardingStore.ByAcceptance(r.Context(),acceptanceID)
	if err!=nil{return domain.Onboarding{},err}
	if pathID:=chi.URLParam(r,"id");pathID!=""&&pathID!=onboarding.ID{return domain.Onboarding{},fmt.Errorf("onboarding access denied")}
	return onboarding,nil
}

func (a *App) onboardingPage(w http.ResponseWriter,r *http.Request) {
	onboarding,err:=a.currentOnboarding(r);if err!=nil{http.Error(w,"Cadastro não encontrado.",http.StatusForbidden);return}
	message:=""
	switch r.URL.Query().Get("saved") {case "1":message="Dados salvos.";case "document":message="Apólice enviada com sucesso."}
	render(r.Context(),w,http.StatusOK,templates.OnboardingPage(onboarding,message,""))
}

func (a *App) saveOnboarding(w http.ResponseWriter,r *http.Request) {
	current,err:=a.currentOnboarding(r);if err!=nil{http.Error(w,"acesso negado",http.StatusForbidden);return}
	if err:=r.ParseForm();err!=nil{http.Error(w,"dados inválidos",http.StatusBadRequest);return}
	cnpj,err:=cleanCNPJ(r.FormValue("cnpj"));if err!=nil{render(r.Context(),w,http.StatusBadRequest,templates.OnboardingPage(current,"",err.Error()));return}
	cpf,err:=cleanCPF(r.FormValue("responsible_cpf"));if err!=nil{render(r.Context(),w,http.StatusBadRequest,templates.OnboardingPage(current,"",err.Error()));return}

	current.CNPJ=cnpj
	current.LegalName=strings.TrimSpace(r.FormValue("legal_name"));current.TradeName=strings.TrimSpace(r.FormValue("trade_name"))
	current.Street=strings.TrimSpace(r.FormValue("street"));current.StreetNumber=strings.TrimSpace(r.FormValue("street_number"));current.Complement=strings.TrimSpace(r.FormValue("complement"));current.District=strings.TrimSpace(r.FormValue("district"));current.City=strings.TrimSpace(r.FormValue("city"));current.State=strings.ToUpper(strings.TrimSpace(r.FormValue("state")));current.PostalCode=strings.TrimSpace(r.FormValue("postal_code"))
	current.OperationType=strings.TrimSpace(r.FormValue("operation_type"));current.Insurer=strings.TrimSpace(r.FormValue("insurer"));current.PolicyStartDate=r.FormValue("policy_start_date");current.PolicyEndDate=r.FormValue("policy_end_date");current.BrokerCompany=strings.TrimSpace(r.FormValue("broker_company"));current.BrokerProducer=strings.TrimSpace(r.FormValue("broker_producer"))
	current.CompanyResponsibleName=strings.TrimSpace(r.FormValue("responsible_name"));current.CompanyResponsibleCPF=cpf;current.CompanyResponsiblePhone=strings.TrimSpace(r.FormValue("responsible_phone"));current.CompanyResponsibleEmail=strings.TrimSpace(strings.ToLower(r.FormValue("responsible_email")));current.CompanyResponsibleRole=strings.TrimSpace(r.FormValue("responsible_role"));current.AuthorityDeclared=r.FormValue("responsible_authority")=="1"
	current.FinanceResponsibleName=strings.TrimSpace(r.FormValue("finance_name"));current.FinanceResponsiblePhone=strings.TrimSpace(r.FormValue("finance_phone"));current.FinanceResponsibleEmail=strings.TrimSpace(strings.ToLower(r.FormValue("finance_email")))
	current.Goods=nil
	for _,value:=range r.Form["goods"]{if value=strings.TrimSpace(value);value!=""{current.Goods=append(current.Goods,value)}}
	current.SystemUsers=nil
	for _,row:=range formValuesAligned(r.Form["system_user_name"],r.Form["system_user_phone"],r.Form["system_user_email"]){if row[0]!=""&&row[2]!=""{current.SystemUsers=append(current.SystemUsers,domain.OnboardingSystemUser{Name:row[0],Phone:row[1],Email:strings.ToLower(row[2])})}}
	if current.LegalName==""||current.CompanyResponsibleName==""||current.CompanyResponsibleEmail==""||!current.AuthorityDeclared{render(r.Context(),w,http.StatusBadRequest,templates.OnboardingPage(current,"","Complete os dados obrigatórios e a declaração do responsável."));return}
	if err:=a.onboardingStore.Save(r.Context(),current);err!=nil{a.logger.Error("save onboarding failed","error",err);render(r.Context(),w,http.StatusBadRequest,templates.OnboardingPage(current,"","Não foi possível salvar os dados."));return}
	http.Redirect(w,r,"/onboarding/"+current.ID+"?saved=1",http.StatusSeeOther)
}

func (a *App) lookupCNPJ(w http.ResponseWriter,r *http.Request) {
	if _,err:=a.currentOnboarding(r);err!=nil{http.Error(w,"acesso negado",http.StatusForbidden);return}
	company,err:=a.registry.Lookup(r.Context(),chi.URLParam(r,"cnpj"));if err!=nil{http.Error(w,"CNPJ não encontrado",http.StatusNotFound);return}
	w.Header().Set("Content-Type","application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"cnpj":company.CNPJ,"legal_name":company.LegalName,"trade_name":company.TradeName,"email":company.Email,"phone":company.Phone,
		"street":company.Street,"number":company.Number,"complement":company.Complement,"district":company.District,"city":company.City,"state":company.State,"postal_code":company.PostalCode,"status":company.Status,"primary_cnae":company.PrimaryCNAE,
	})
}

func (a *App) uploadOnboardingDocument(w http.ResponseWriter,r *http.Request) {
	current,err:=a.currentOnboarding(r);if err!=nil{http.Error(w,"acesso negado",http.StatusForbidden);return}
	r.Body=http.MaxBytesReader(w,r.Body,16<<20)
	if err:=r.ParseMultipartForm(16<<20);err!=nil{http.Error(w,"arquivo excede o limite de 15 MB",http.StatusRequestEntityTooLarge);return}
	file,header,err:=r.FormFile("document");if err!=nil{http.Error(w,"selecione a apólice",http.StatusBadRequest);return};defer file.Close()
	content,err:=io.ReadAll(io.LimitReader(file,15<<20+1));if err!=nil||len(content)==0||len(content)>15<<20{http.Error(w,"arquivo inválido",http.StatusBadRequest);return}
	mimeType:=http.DetectContentType(content[:min(512,len(content))])
	allowed:=map[string]bool{"application/pdf":true,"image/jpeg":true,"image/png":true}
	if !allowed[mimeType]{http.Error(w,"formato não permitido; envie PDF, JPG ou PNG",http.StatusUnsupportedMediaType);return}
	hash:=sha256.Sum256(content)
	key:=fmt.Sprintf("onboarding/%s/insurance_policy/%d-%s",current.ID,time.Now().UTC().UnixNano(),sanitizeFilename(header.Filename))
	if err:=a.storage.Put(r.Context(),key,mimeType,bytes.NewReader(content),int64(len(content)));err!=nil{a.logger.Error("upload policy to S3 failed","error",err);http.Error(w,"não foi possível armazenar o arquivo",http.StatusInternalServerError);return}
	document:=onboardingpkg.Document{DocumentType:"insurance_policy",StorageKey:key,OriginalFilename:header.Filename,MIMEType:mimeType,SizeBytes:int64(len(content)),SHA256:hash[:]}
	if err:=a.onboardingStore.AddDocument(r.Context(),current.ID,document);err!=nil{_ = a.storage.Delete(r.Context(),key);http.Error(w,"não foi possível registrar o arquivo",http.StatusInternalServerError);return}
	_,_ = a.pool.Exec(r.Context(),`insert into audit_events(actor_type,event_type,resource_type,resource_id,ip_address,user_agent,metadata) values('customer','document.uploaded','onboarding',$1,$2,$3,jsonb_build_object('type','insurance_policy','sha256',$4))`,current.ID,requestIP(r),r.UserAgent(),fmt.Sprintf("%x",hash[:]))
	http.Redirect(w,r,"/onboarding/"+current.ID+"?saved=document",http.StatusSeeOther)
}

func (a *App) submitOnboarding(w http.ResponseWriter,r *http.Request) {
	current,err:=a.currentOnboarding(r);if err!=nil{http.Error(w,"acesso negado",http.StatusForbidden);return}
	if err:=a.onboardingStore.Submit(r.Context(),current.ID);err!=nil{render(r.Context(),w,http.StatusBadRequest,templates.OnboardingPage(current,"",err.Error()));return}
	generated,err:=a.contractGenerator.GenerateForOnboarding(r.Context(),current.ID)
	if err!=nil{a.logger.Error("generate contract failed","error",err,"onboarding_id",current.ID);http.Error(w,"Dados recebidos, mas não foi possível gerar o contrato. O comercial foi avisado.",http.StatusInternalServerError);return}
	link:=strings.TrimRight(a.cfg.BaseURL,"/")+"/sign/"+generated.SignerToken
	htmlBody:=fmt.Sprintf("<p>Olá, %s.</p><p>Os dados da operação foram recebidos e o contrato está pronto para assinatura.</p><p><a href=\"%s\">Revisar e assinar contrato</a></p>",generated.SignerName,link)
	_ = notifications.Enqueue(r.Context(),a.pool,generated.SignerName,generated.SignerEmail,"Contrato ViaGate disponível para assinatura",htmlBody,"Contrato disponível em "+link)
	_ = a.contractStore.MarkSent(r.Context(),generated.ContractID)
	_,_ = a.pool.Exec(r.Context(),`insert into audit_events(actor_type,event_type,resource_type,resource_id,ip_address,user_agent,metadata) values('customer','onboarding.submitted','onboarding',$1,$2,$3,jsonb_build_object('contract_id',$4))`,current.ID,requestIP(r),r.UserAgent(),generated.ContractID)
	http.Redirect(w,r,"/sign/"+generated.SignerToken,http.StatusSeeOther)
}

func (a *App) signaturePage(w http.ResponseWriter,r *http.Request) {
	token:=chi.URLParam(r,"token")
	access,err:=a.contractStore.SignerByPublicToken(r.Context(),token);if err!=nil{http.Error(w,"Link de assinatura inválido.",http.StatusNotFound);return}
	message:="";if r.URL.Query().Get("otp")=="sent"{message="Código enviado para o e-mail do responsável."};if r.URL.Query().Get("signed")=="1"{message="Contrato assinado com sucesso."}
	_,_ = a.pool.Exec(r.Context(),`insert into signature_events(contract_id,contract_signer_id,event_type,document_hash,ip_address,user_agent,metadata) values($1,$2,'contract.viewed',$3,$4,$5,'{}')`,access.Contract.ID,access.Signer.ID,access.Contract.DocumentSHA256,requestIP(r),r.UserAgent())
	render(r.Context(),w,http.StatusOK,templates.SignaturePage(access,token,message,""))
}

func (a *App) sendSignatureOTP(w http.ResponseWriter,r *http.Request) {
	token:=chi.URLParam(r,"token")
	access,err:=a.contractStore.SignerByPublicToken(r.Context(),token);if err!=nil{http.Error(w,"Link inválido",http.StatusNotFound);return}
	if access.Signer.Status=="signed"{http.Redirect(w,r,"/sign/"+token+"?signed=1",http.StatusSeeOther);return}
	var recent int
	_ = a.pool.QueryRow(r.Context(),`select count(*) from signature_challenges where contract_signer_id=$1 and created_at>now()-interval '10 minutes'`,access.Signer.ID).Scan(&recent)
	if recent>=3{render(r.Context(),w,http.StatusTooManyRequests,templates.SignaturePage(access,token,"","Aguarde alguns minutos antes de solicitar outro código."));return}
	otp,hash,err:=security.RandomOTP();if err!=nil{http.Error(w,"erro interno",http.StatusInternalServerError);return}
	if err:=a.contractStore.CreateChallenge(r.Context(),access.Signer.ID,hash,time.Now().Add(a.cfg.Session.SignatureOTPTTL));err!=nil{http.Error(w,"não foi possível gerar o código",http.StatusInternalServerError);return}
	_,_ = a.pool.Exec(r.Context(),`insert into identity_verifications(contract_signer_id,mode,status,provider) values($1,'email_otp','pending','brevo')`,access.Signer.ID)
	htmlBody:=fmt.Sprintf("<p>Seu código de confirmação para assinatura do contrato ViaGate é:</p><p style=\"font-size:28px;font-weight:800;letter-spacing:4px\">%s</p><p>O código expira em %s.</p>",otp,a.cfg.Session.SignatureOTPTTL)
	if err:=notifications.Enqueue(r.Context(),a.pool,access.Signer.Name,access.Signer.Email,"Código de assinatura ViaGate",htmlBody,"Código: "+otp);err!=nil{http.Error(w,"não foi possível enviar o código",http.StatusInternalServerError);return}
	http.Redirect(w,r,"/sign/"+token+"?otp=sent",http.StatusSeeOther)
}

func (a *App) confirmSignature(w http.ResponseWriter,r *http.Request) {
	token:=chi.URLParam(r,"token")
	access,err:=a.contractStore.SignerByPublicToken(r.Context(),token);if err!=nil{http.Error(w,"Link inválido",http.StatusNotFound);return}
	if err:=r.ParseForm();err!=nil{http.Error(w,"dados inválidos",http.StatusBadRequest);return}
	if r.FormValue("consent")!="1"{render(r.Context(),w,http.StatusBadRequest,templates.SignaturePage(access,token,"","Confirme que leu e concorda em assinar o contrato."));return}
	otp:=digits(r.FormValue("otp"));if len(otp)!=6{render(r.Context(),w,http.StatusBadRequest,templates.SignaturePage(access,token,"","Informe o código de 6 dígitos."));return}
	if err:=a.contractStore.VerifyChallenge(r.Context(),access.Signer.ID,security.HashToken(otp));err!=nil{render(r.Context(),w,http.StatusUnauthorized,templates.SignaturePage(access,token,"","Código inválido ou expirado."));return}
	sessionID,err:=newUUID();if err!=nil{http.Error(w,"erro interno",http.StatusInternalServerError);return}
	if err:=a.contractStore.Sign(r.Context(),access.Signer.ID,access.Contract.DocumentSHA256,sessionID,requestIP(r),r.UserAgent());err!=nil{a.logger.Error("contract sign failed","error",err);render(r.Context(),w,http.StatusConflict,templates.SignaturePage(access,token,"","Não foi possível concluir a assinatura. O documento pode ter sido alterado ou a validação expirou."));return}
	_,_ = a.pool.Exec(r.Context(),`update identity_verifications set status='verified',verified_at=now(),evidence=evidence||jsonb_build_object('session_id',$2,'ip',$3,'user_agent',$4) where contract_signer_id=$1 and mode='email_otp' and status='pending'`,access.Signer.ID,sessionID,requestIP(r),r.UserAgent())
	http.Redirect(w,r,"/sign/"+token+"?signed=1",http.StatusSeeOther)
}
