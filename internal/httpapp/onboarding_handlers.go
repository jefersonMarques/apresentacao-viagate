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
	onboardingpkg "github.com/jefersonMarques/apresentacao-viagate/internal/onboarding"
	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

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
	current.OperationType=strings.ToLower(strings.TrimSpace(r.FormValue("operation_type")));current.Insurer=strings.TrimSpace(r.FormValue("insurer"));current.PolicyStartDate=r.FormValue("policy_start_date");current.PolicyEndDate=r.FormValue("policy_end_date");current.BrokerCompany=strings.TrimSpace(r.FormValue("broker_company"));current.BrokerProducer=strings.TrimSpace(r.FormValue("broker_producer"))
	current.CompanyResponsibleName=strings.TrimSpace(r.FormValue("responsible_name"));current.CompanyResponsibleCPF=cpf;current.CompanyResponsiblePhone=strings.TrimSpace(r.FormValue("responsible_phone"));current.CompanyResponsibleEmail=strings.TrimSpace(strings.ToLower(r.FormValue("responsible_email")));current.CompanyResponsibleRole=strings.TrimSpace(r.FormValue("responsible_role"));current.AuthorityDeclared=r.FormValue("responsible_authority")=="1"
	current.FinanceResponsibleName=strings.TrimSpace(r.FormValue("finance_name"));current.FinanceResponsiblePhone=strings.TrimSpace(r.FormValue("finance_phone"));current.FinanceResponsibleEmail=strings.TrimSpace(strings.ToLower(r.FormValue("finance_email")))
	current.Goods=nil
	for _,value:=range r.Form["goods"]{if value=strings.TrimSpace(value);value!=""{current.Goods=append(current.Goods,value)}}
	current.SystemUsers=nil
	for _,row:=range formValuesAligned(r.Form["system_user_name"],r.Form["system_user_phone"],r.Form["system_user_email"]){if row[0]!=""&&row[2]!=""{current.SystemUsers=append(current.SystemUsers,domain.OnboardingSystemUser{Name:row[0],Phone:row[1],Email:strings.ToLower(row[2])})}}

	if validationError:=validateOnboarding(current);validationError!=""{render(r.Context(),w,http.StatusBadRequest,templates.OnboardingPage(current,"",validationError));return}
	if err:=a.onboardingStore.Save(r.Context(),current);err!=nil{a.logger.Error("save onboarding failed","error",err);render(r.Context(),w,http.StatusBadRequest,templates.OnboardingPage(current,"","Não foi possível salvar os dados."));return}
	http.Redirect(w,r,"/onboarding/"+current.ID+"?saved=1",http.StatusSeeOther)
}

func validateOnboarding(current domain.Onboarding) string {
	if current.LegalName==""||current.CompanyResponsibleName==""||current.CompanyResponsibleEmail==""||current.CompanyResponsiblePhone==""||!current.AuthorityDeclared{return "Complete os dados obrigatórios e a declaração do responsável."}
	if _,err:=mail.ParseAddress(current.CompanyResponsibleEmail);err!=nil{return "O e-mail do responsável é inválido."}
	if current.FinanceResponsibleEmail!=""{if _,err:=mail.ParseAddress(current.FinanceResponsibleEmail);err!=nil{return "O e-mail do responsável financeiro é inválido."}}
	if current.OperationType!="normal"&&current.OperationType!="avulsa"{return "Selecione um tipo de operação válido."}
	if len(current.State)>2{return "UF inválida."}
	if current.PolicyStartDate!=""&&current.PolicyEndDate!=""{start,err1:=time.Parse("2006-01-02",current.PolicyStartDate);end,err2:=time.Parse("2006-01-02",current.PolicyEndDate);if err1!=nil||err2!=nil{return "A vigência da apólice é inválida."};if end.Before(start){return "O fim da vigência da apólice não pode ser anterior ao início."}}
	for _,user:=range current.SystemUsers{if _,err:=mail.ParseAddress(user.Email);err!=nil{return "Há um e-mail inválido na lista de usuários do sistema."}}
	return ""
}

func (a *App) lookupCNPJ(w http.ResponseWriter,r *http.Request) {
	if _,err:=a.currentOnboarding(r);err!=nil{http.Error(w,"acesso negado",http.StatusForbidden);return}
	company,err:=a.registry.Lookup(r.Context(),chi.URLParam(r,"cnpj"));if err!=nil{http.Error(w,"CNPJ não encontrado",http.StatusNotFound);return}
	w.Header().Set("Content-Type","application/json")
	w.Header().Set("Cache-Control","private, max-age=300")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"cnpj":company.CNPJ,"legal_name":company.LegalName,"trade_name":company.TradeName,"email":company.Email,"phone":company.Phone,
		"street":company.Street,"number":company.Number,"complement":company.Complement,"district":company.District,"city":company.City,"state":company.State,"postal_code":company.PostalCode,"status":company.Status,"primary_cnae":company.PrimaryCNAE,
	})
}

func (a *App) uploadOnboardingDocument(w http.ResponseWriter,r *http.Request) {
	current,err:=a.currentOnboarding(r);if err!=nil{http.Error(w,"acesso negado",http.StatusForbidden);return}
	r.Body=http.MaxBytesReader(w,r.Body,(15<<20)+1)
	if err:=r.ParseMultipartForm(15<<20);err!=nil{http.Error(w,"arquivo excede o limite de 15 MB",http.StatusRequestEntityTooLarge);return}
	file,header,err:=r.FormFile("document");if err!=nil{http.Error(w,"selecione a apólice",http.StatusBadRequest);return};defer file.Close()
	content,err:=io.ReadAll(io.LimitReader(file,(15<<20)+1));if err!=nil||len(content)==0||len(content)>15<<20{http.Error(w,"arquivo inválido",http.StatusBadRequest);return}
	mimeType:=http.DetectContentType(content[:min(512,len(content))])
	allowed:=map[string]bool{"application/pdf":true,"image/jpeg":true,"image/png":true}
	if !allowed[mimeType]{http.Error(w,"formato não permitido; envie PDF, JPG ou PNG",http.StatusUnsupportedMediaType);return}
	hash:=sha256.Sum256(content)
	key:=fmt.Sprintf("onboarding/%s/insurance_policy/%d-%s",current.ID,time.Now().UTC().UnixNano(),sanitizeFilename(header.Filename))
	if err:=a.storage.Put(r.Context(),key,mimeType,bytes.NewReader(content),int64(len(content)));err!=nil{a.logger.Error("upload policy to S3 failed","error",err);http.Error(w,"não foi possível armazenar o arquivo",http.StatusInternalServerError);return}
	document:=onboardingpkg.Document{DocumentType:"insurance_policy",StorageKey:key,OriginalFilename:header.Filename,MIMEType:mimeType,SizeBytes:int64(len(content)),SHA256:hash[:]}
	if err:=a.onboardingStore.AddDocument(r.Context(),current.ID,document);err!=nil{_ = a.storage.Delete(r.Context(),key);http.Error(w,"não foi possível registrar o arquivo",http.StatusInternalServerError);return}
	_,_ = a.pool.Exec(r.Context(),`insert into audit_events(actor_type,event_type,resource_type,resource_id,ip_address,user_agent,metadata) values('customer','document.uploaded','onboarding',$1,$2,$3,jsonb_build_object('type','insurance_policy','sha256',$4,'filename',$5))`,current.ID,requestIP(r),r.UserAgent(),fmt.Sprintf("%x",hash[:]),header.Filename)
	http.Redirect(w,r,"/onboarding/"+current.ID+"?saved=document",http.StatusSeeOther)
}

func (a *App) submitOnboarding(w http.ResponseWriter,r *http.Request) {
	current,err:=a.currentOnboarding(r);if err!=nil{http.Error(w,"acesso negado",http.StatusForbidden);return}
	if err:=a.onboardingStore.Submit(r.Context(),current.ID);err!=nil{render(r.Context(),w,http.StatusBadRequest,templates.OnboardingPage(current,"",err.Error()));return}

	access,_,err:=a.ensureContractDelivery(r.Context(),current.ID)
	if err!=nil{
		a.logger.Error("contract delivery failed","error",err,"onboarding_id",current.ID)
		_,_ = a.pool.Exec(r.Context(),`insert into audit_events(actor_type,event_type,resource_type,resource_id,metadata) values('system','contract.generation_failed','onboarding',$1,jsonb_build_object('error',$2))`,current.ID,err.Error())
		_,_ = a.pool.Exec(r.Context(),`
			insert into notification_outbox(dedupe_key,recipient,recipient_name,subject,html_body,text_body)
			select 'contract-generation-failed:'||o.id::text,u.email,u.name,'Falha na geração de contrato: '||p.title,
			       '<p>O onboarding de <strong>'||c.legal_name||'</strong> foi enviado, mas o contrato precisa ser gerado novamente pelo painel.</p>',
			       'O onboarding de '||c.legal_name||' foi enviado, mas a geração automática do contrato falhou.'
			from onboardings o join proposal_acceptances pa on pa.id=o.proposal_acceptance_id join proposals p on p.id=pa.proposal_id join clients c on c.id=o.client_id join users u on u.id=p.created_by where o.id=$1
			on conflict (dedupe_key) where dedupe_key is not null do nothing
		`,current.ID)
		http.Error(w,"Dados recebidos com sucesso. O comercial foi avisado para concluir a geração do contrato.",http.StatusAccepted);return
	}
	_,_ = a.pool.Exec(r.Context(),`insert into audit_events(actor_type,event_type,resource_type,resource_id,ip_address,user_agent,metadata) values('customer','onboarding.submitted','onboarding',$1,$2,$3,jsonb_build_object('contract_id',$4))`,current.ID,requestIP(r),r.UserAgent(),access.ContractID)
	http.Redirect(w,r,"/sign/"+access.SignerToken,http.StatusSeeOther)
}
