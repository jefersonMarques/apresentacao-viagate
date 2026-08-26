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

func (a *App) adminOnboardings(w http.ResponseWriter,r *http.Request){
	user,_:=currentUser(r.Context())
	items,err:=a.onboardingStore.ListForReview(r.Context())
	if err!=nil{a.logger.Error("list onboardings failed","error",err);http.Error(w,"não foi possível carregar os cadastros",http.StatusInternalServerError);return}
	render(r.Context(),w,http.StatusOK,templates.OnboardingAdminListPage(user,items))
}

func (a *App) adminOnboardingDetail(w http.ResponseWriter,r *http.Request){
	user,_:=currentUser(r.Context())
	detail,err:=a.onboardingStore.AdminByID(r.Context(),chi.URLParam(r,"id"))
	if err!=nil{http.Error(w,"cadastro não encontrado",http.StatusNotFound);return}
	message:=""
	switch r.URL.Query().Get("result"){
	case "reviewed":message="Revisão registrada."
	case "contract":message="Contrato disponível e notificação de assinatura garantida na fila do Brevo."
	case "correction":message="Correção solicitada e cliente notificado."
	}
	errorMessage:=r.URL.Query().Get("error")
	render(r.Context(),w,http.StatusOK,templates.OnboardingAdminDetailPage(user,detail,message,errorMessage))
}

func (a *App) reviewOnboarding(w http.ResponseWriter,r *http.Request){
	user,_:=currentUser(r.Context())
	id:=chi.URLParam(r,"id")
	if err:=r.ParseForm();err!=nil{http.Error(w,"dados inválidos",http.StatusBadRequest);return}
	status:=strings.TrimSpace(r.FormValue("status"))
	notes:=strings.TrimSpace(r.FormValue("notes"))
	detail,err:=a.onboardingStore.AdminByID(r.Context(),id)
	if err!=nil{http.Error(w,"cadastro não encontrado",http.StatusNotFound);return}
	if status=="correction_requested"&&notes==""{http.Redirect(w,r,"/admin/onboardings/"+id+"?error="+queryEscape("Informe o que precisa ser corrigido."),http.StatusSeeOther);return}
	if err:=a.onboardingStore.Review(r.Context(),id,user.ID,status,notes);err!=nil{
		a.logger.Error("review onboarding failed","error",err,"onboarding_id",id)
		http.Redirect(w,r,"/admin/onboardings/"+id+"?error="+queryEscape("Não foi possível alterar o status: "+err.Error()),http.StatusSeeOther);return
	}
	_,_ = a.pool.Exec(r.Context(),`
		insert into audit_events(actor_user_id,event_type,resource_type,resource_id,ip_address,user_agent,metadata)
		values($1,'onboarding.reviewed','onboarding',$2,$3,$4,jsonb_build_object('status',$5,'notes',$6))
	`,user.ID,id,requestIP(r),r.UserAgent(),status,notes)

	if status=="correction_requested"{
		if err:=a.sendOnboardingCorrection(r,id,detail.Onboarding.ProposalAcceptanceID,detail.Onboarding.CompanyResponsibleName,detail.Onboarding.CompanyResponsibleEmail,notes);err!=nil{
			a.logger.Error("queue onboarding correction failed","error",err,"onboarding_id",id)
			http.Redirect(w,r,"/admin/onboardings/"+id+"?error="+queryEscape("Correção registrada, mas o e-mail não entrou na fila."),http.StatusSeeOther);return
		}
		http.Redirect(w,r,"/admin/onboardings/"+id+"?result=correction",http.StatusSeeOther);return
	}
	if status=="approved"{
		if _,_,err:=a.ensureContractDelivery(r.Context(),id);err!=nil{
			a.logger.Error("approved onboarding contract delivery failed","error",err,"onboarding_id",id)
			http.Redirect(w,r,"/admin/onboardings/"+id+"?error="+queryEscape("Cadastro aprovado, mas a geração/envio do contrato falhou. Use Tentar novamente."),http.StatusSeeOther);return
		}
	}
	http.Redirect(w,r,"/admin/onboardings/"+id+"?result=reviewed",http.StatusSeeOther)
}

func (a *App) retryOnboardingContract(w http.ResponseWriter,r *http.Request){
	user,_:=currentUser(r.Context())
	id:=chi.URLParam(r,"id")
	access,created,err:=a.ensureContractDelivery(r.Context(),id)
	if err!=nil{
		a.logger.Error("manual contract retry failed","error",err,"onboarding_id",id)
		_,_ = a.pool.Exec(r.Context(),`insert into audit_events(actor_user_id,event_type,resource_type,resource_id,metadata) values($1,'contract.retry_failed','onboarding',$2,jsonb_build_object('error',$3))`,user.ID,id,err.Error())
		http.Redirect(w,r,"/admin/onboardings/"+id+"?error="+queryEscape("Não foi possível gerar ou reenviar o contrato: "+err.Error()),http.StatusSeeOther);return
	}
	_,_ = a.pool.Exec(r.Context(),`
		insert into audit_events(actor_user_id,event_type,resource_type,resource_id,metadata)
		values($1,'contract.retry_completed','onboarding',$2,jsonb_build_object('contract_id',$3,'created',$4))
	`,user.ID,id,access.ContractID,created)
	http.Redirect(w,r,"/admin/onboardings/"+id+"?result=contract",http.StatusSeeOther)
}

func (a *App) adminOnboardingDocument(w http.ResponseWriter,r *http.Request){
	id:=chi.URLParam(r,"id")
	document,err:=a.onboardingStore.DocumentByID(r.Context(),id,chi.URLParam(r,"documentID"))
	if err!=nil{http.Error(w,"documento não encontrado",http.StatusNotFound);return}
	url,err:=a.storage.SignedDownloadURL(r.Context(),document.StorageKey,a.cfg.S3.DownloadTTL)
	if err!=nil{a.logger.Error("presign onboarding document failed","error",err);http.Error(w,"não foi possível disponibilizar o documento",http.StatusInternalServerError);return}
	w.Header().Set("Cache-Control","no-store")
	http.Redirect(w,r,url.String(),http.StatusTemporaryRedirect)
}

func (a *App) sendOnboardingCorrection(r *http.Request,onboardingID,acceptanceID,name,emailAddress,notes string) error {
	plain,hash,err:=security.RandomToken(32);if err!=nil{return err}
	expires:=time.Now().Add(7*24*time.Hour)
	if err:=a.proposalStore.CreateCustomerSession(r.Context(),acceptanceID,hash,requestIP(r),r.UserAgent(),expires);err!=nil{return err}
	link:=strings.TrimRight(a.cfg.BaseURL,"/")+"/onboarding/resume/"+plain
	htmlBody:=fmt.Sprintf("<p>Olá, %s.</p><p>Precisamos de um ajuste nas informações enviadas para a implantação ViaGate.</p><p><strong>Solicitação:</strong> %s</p><p><a href=\"%s\">Revisar e corrigir cadastro</a></p>",name,notes,link)
	return notifications.Enqueue(r.Context(),a.pool,name,emailAddress,"Ajuste necessário no cadastro ViaGate",htmlBody,"Ajuste solicitado: "+notes+"\nAcesse: "+link)
}

func queryEscape(value string) string {
	replacer:=strings.NewReplacer("%","%25"," ","%20","?","%3F","&","%26","=","%3D","#","%23","\n","%0A")
	return replacer.Replace(value)
}
