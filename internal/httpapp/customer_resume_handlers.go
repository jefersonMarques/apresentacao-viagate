package httpapp

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/security"
)

func (a *App) resumeOnboarding(w http.ResponseWriter,r *http.Request){
	plain:=chi.URLParam(r,"token")
	acceptanceID,err:=a.proposalStore.ConsumeCustomerResumeToken(r.Context(),hashToken(plain))
	if err!=nil{http.Error(w,"Link de retomada inválido, expirado ou já utilizado.",http.StatusGone);return}
	onboarding,err:=a.onboardingStore.ByAcceptance(r.Context(),acceptanceID)
	if err!=nil{http.Error(w,"Cadastro não encontrado.",http.StatusNotFound);return}
	if onboarding.Status!="correction_requested"{http.Error(w,"Este cadastro não possui correção pendente.",http.StatusGone);return}

	sessionPlain,sessionHash,err:=security.RandomToken(32)
	if err!=nil{http.Error(w,"não foi possível iniciar a sessão",http.StatusInternalServerError);return}
	expires:=time.Now().Add(7*24*time.Hour)
	if err:=a.proposalStore.CreateCustomerSession(r.Context(),acceptanceID,sessionHash,requestIP(r),r.UserAgent(),expires);err!=nil{http.Error(w,"não foi possível iniciar a sessão",http.StatusInternalServerError);return}
	setSecureCookie(w,customerSessionCookie,sessionPlain,expires,a.cfg.Environment=="production")
	_,_ = a.pool.Exec(r.Context(),`
		insert into audit_events(actor_type,event_type,resource_type,resource_id,ip_address,user_agent,metadata)
		values('customer','onboarding.resumed','onboarding',$1,$2,$3,jsonb_build_object('one_time_link',true))
	`,onboarding.ID,requestIP(r),r.UserAgent())
	http.Redirect(w,r,"/onboarding/"+onboarding.ID,http.StatusSeeOther)
}
