package httpapp

import (
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/notifications"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/security"
	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

func (a *App) loginPage(w http.ResponseWriter,r *http.Request) {
	render(r.Context(),w,http.StatusOK,templates.Login(""))
}

func (a *App) login(w http.ResponseWriter,r *http.Request) {
	if err:=r.ParseForm();err!=nil { render(r.Context(),w,http.StatusBadRequest,templates.Login("Dados inválidos.")); return }
	email:=strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password:=r.FormValue("password")
	ip:=requestIP(r)

	allowed,err:=a.authStore.LoginAllowed(r.Context(),email,ip)
	if err!=nil{
		a.logger.Error("login rate limit check failed","error",err)
		render(r.Context(),w,http.StatusServiceUnavailable,templates.Login("Não foi possível validar o acesso agora. Tente novamente em instantes."))
		return
	}
	if !allowed{
		time.Sleep(250*time.Millisecond)
		w.Header().Set("Retry-After","900")
		render(r.Context(),w,http.StatusTooManyRequests,templates.Login("Muitas tentativas de acesso. Aguarde alguns minutos e tente novamente."))
		return
	}

	credentials,err:=a.authStore.FindCredentials(r.Context(),email)
	if err!=nil || credentials.User.Status!="active" || !security.VerifyPassword(credentials.PasswordHash,password) {
		if recordErr:=a.authStore.RecordLoginFailure(r.Context(),email,ip);recordErr!=nil{a.logger.Error("record login failure failed","error",recordErr)}
		time.Sleep(250*time.Millisecond)
		render(r.Context(),w,http.StatusUnauthorized,templates.Login("E-mail ou senha inválidos."))
		return
	}

	token,hash,err:=security.RandomToken(32)
	if err!=nil { http.Error(w,"não foi possível iniciar a sessão",http.StatusInternalServerError);return }
	expires:=time.Now().Add(a.cfg.Session.TTL)
	if err:=a.authStore.CreateSession(r.Context(),credentials.User.ID,hash,ip,r.UserAgent(),expires);err!=nil {
		a.logger.Error("create session failed","error",err)
		http.Error(w,"não foi possível iniciar a sessão",http.StatusInternalServerError)
		return
	}
	if err:=a.authStore.ClearLoginFailures(r.Context(),email,ip);err!=nil{a.logger.Warn("clear login failures failed","error",err,"user_id",credentials.User.ID)}
	setSecureCookie(w,a.cfg.Session.CookieName,token,expires,a.cfg.Environment=="production")
	_,_ = a.pool.Exec(r.Context(),`update users set last_login_at=now() where id=$1`,credentials.User.ID)
	http.Redirect(w,r,"/admin",http.StatusSeeOther)
}

func (a *App) logout(w http.ResponseWriter,r *http.Request) {
	if cookie,err:=r.Cookie(a.cfg.Session.CookieName);err==nil && cookie.Value!="" {
		_ = a.authStore.RevokeSession(r.Context(),hashToken(cookie.Value))
	}
	clearCookie(w,a.cfg.Session.CookieName)
	http.Redirect(w,r,"/login",http.StatusSeeOther)
}

func (a *App) invitationPage(w http.ResponseWriter,r *http.Request) {
	token:=chi.URLParam(r,"token")
	user,_,err:=a.authStore.Invitation(r.Context(),hashToken(token))
	if err!=nil {
		if err==pgx.ErrNoRows { http.Error(w,"Convite inválido ou expirado.",http.StatusGone);return }
		http.Error(w,"Não foi possível carregar o convite.",http.StatusInternalServerError);return
	}
	render(r.Context(),w,http.StatusOK,templates.InvitationPage(user.Name,user.Email,token,""))
}

func (a *App) acceptInvitation(w http.ResponseWriter,r *http.Request) {
	token:=chi.URLParam(r,"token")
	user,_,err:=a.authStore.Invitation(r.Context(),hashToken(token))
	if err!=nil { http.Error(w,"Convite inválido ou expirado.",http.StatusGone);return }
	if err:=r.ParseForm();err!=nil { http.Error(w,"dados inválidos",http.StatusBadRequest);return }
	password:=r.FormValue("password")
	if password!=r.FormValue("confirm_password") {
		render(r.Context(),w,http.StatusBadRequest,templates.InvitationPage(user.Name,user.Email,token,"As senhas são diferentes."));return
	}
	passwordHash,err:=security.HashPassword(password)
	if err!=nil { render(r.Context(),w,http.StatusBadRequest,templates.InvitationPage(user.Name,user.Email,token,err.Error()));return }
	activated,err:=a.authStore.AcceptInvitation(r.Context(),hashToken(token),passwordHash)
	if err!=nil {
		a.logger.Error("accept invitation failed","error",err)
		render(r.Context(),w,http.StatusInternalServerError,templates.InvitationPage(user.Name,user.Email,token,"Não foi possível ativar o acesso."));return
	}
	_,_ = a.pool.Exec(r.Context(),`insert into audit_events(actor_user_id,event_type,resource_type,resource_id,ip_address,user_agent,metadata) values($1,'user.activated','user',$1,$2,$3,'{}')`,activated.ID,requestIP(r),r.UserAgent())
	http.Redirect(w,r,"/login",http.StatusSeeOther)
}

func (a *App) inviteUser(w http.ResponseWriter,r *http.Request) {
	admin,_:=currentUser(r.Context())
	if err:=r.ParseForm();err!=nil { http.Error(w,"dados inválidos",http.StatusBadRequest);return }
	name:=strings.TrimSpace(r.FormValue("name"))
	emailAddress,err:=cleanEmail(r.FormValue("email"),true)
	if err!=nil{http.Redirect(w,r,"/admin/users?error="+queryEscape("Informe um e-mail válido."),http.StatusSeeOther);return}
	role:=strings.TrimSpace(r.FormValue("role"))
	if name=="" { http.Error(w,"nome é obrigatório",http.StatusBadRequest);return }

	token,hash,err:=security.RandomToken(32)
	if err!=nil { http.Error(w,"erro ao criar convite",http.StatusInternalServerError);return }
	userID,err:=a.authStore.CreateManagedInvitation(r.Context(),emailAddress,name,role,hash,time.Now().Add(a.cfg.Session.InviteTTL),admin.ID)
	if err!=nil {
		a.logger.Error("create invitation failed","error",err)
		http.Redirect(w,r,"/admin/users?error="+queryEscape("Não foi possível criar o convite: "+err.Error()),http.StatusSeeOther);return
	}
	link:=strings.TrimRight(a.cfg.BaseURL,"/")+"/invite/"+token
	htmlBody:=fmt.Sprintf("<p>Olá, %s.</p><p>Você recebeu um convite para acessar a plataforma comercial da ViaGate.</p><p><a href=\"%s\">Ativar meu acesso</a></p><p>O link é individual e expira em %s.</p>",html.EscapeString(name),html.EscapeString(link),a.cfg.Session.InviteTTL)
	if err:=notifications.Enqueue(r.Context(),a.pool,name,emailAddress,"Convite para a plataforma ViaGate",htmlBody,"Acesse "+link);err!=nil{
		a.logger.Error("queue invitation email failed","error",err,"user_id",userID)
		http.Redirect(w,r,"/admin/users?error="+queryEscape("Convite criado, mas o e-mail não entrou na fila do Brevo."),http.StatusSeeOther);return
	}
	_,_ = a.pool.Exec(r.Context(),`insert into audit_events(actor_user_id,event_type,resource_type,resource_id,ip_address,user_agent,metadata) values($1,'user.invited','user',$2,$3,$4,jsonb_build_object('role',$5))`,admin.ID,userID,requestIP(r),r.UserAgent(),role)
	http.Redirect(w,r,"/admin/users?invited=1",http.StatusSeeOther)
}
