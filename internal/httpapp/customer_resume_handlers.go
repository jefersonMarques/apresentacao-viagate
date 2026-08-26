package httpapp

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func (a *App) resumeOnboarding(w http.ResponseWriter,r *http.Request){
	plain:=chi.URLParam(r,"token")
	acceptanceID,err:=a.proposalStore.CustomerSessionAcceptance(r.Context(),hashToken(plain))
	if err!=nil{http.Error(w,"Link de retomada inválido ou expirado.",http.StatusGone);return}
	onboarding,err:=a.onboardingStore.ByAcceptance(r.Context(),acceptanceID)
	if err!=nil{http.Error(w,"Cadastro não encontrado.",http.StatusNotFound);return}
	setSecureCookie(w,customerSessionCookie,plain,time.Now().Add(7*24*time.Hour),a.cfg.Environment=="production")
	_,_ = a.pool.Exec(r.Context(),`
		insert into audit_events(actor_type,event_type,resource_type,resource_id,ip_address,user_agent,metadata)
		values('customer','onboarding.resumed','onboarding',$1,$2,$3,'{}')
	`,onboarding.ID,requestIP(r),r.UserAgent())
	http.Redirect(w,r,"/onboarding/"+onboarding.ID,http.StatusSeeOther)
}
