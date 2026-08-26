package httpapp

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (a *App) updateUserAccess(w http.ResponseWriter,r *http.Request){
	admin,_:=currentUser(r.Context())
	targetID:=chi.URLParam(r,"id")
	if targetID==admin.ID{
		http.Redirect(w,r,"/admin/users?error="+queryEscape("Você não pode alterar o próprio perfil ou desativar o próprio acesso."),http.StatusSeeOther);return
	}
	if err:=r.ParseForm();err!=nil{http.Error(w,"dados inválidos",http.StatusBadRequest);return}
	status:=strings.TrimSpace(r.FormValue("status"))
	role:=strings.TrimSpace(r.FormValue("role"))
	_,_,currentStatus,err:=a.authStore.UserByID(r.Context(),targetID)
	if err!=nil{http.Error(w,"usuário não encontrado",http.StatusNotFound);return}
	if currentStatus=="invited"&&status=="active"{
		http.Redirect(w,r,"/admin/users?error="+queryEscape("Usuários convidados precisam ativar a conta pelo link antes de ficarem ativos."),http.StatusSeeOther);return
	}
	if err:=a.authStore.UpdateAccess(r.Context(),targetID,status,role);err!=nil{
		a.logger.Error("update user access failed","error",err,"target_user_id",targetID)
		http.Redirect(w,r,"/admin/users?error="+queryEscape("Não foi possível alterar o acesso: "+err.Error()),http.StatusSeeOther);return
	}
	_,_ = a.pool.Exec(r.Context(),`
		insert into audit_events(actor_user_id,event_type,resource_type,resource_id,ip_address,user_agent,metadata)
		values($1,'user.access_updated','user',$2,$3,$4,jsonb_build_object('status',$5,'role',$6))
	`,admin.ID,targetID,requestIP(r),r.UserAgent(),status,role)
	http.Redirect(w,r,"/admin/users?updated=1",http.StatusSeeOther)
}
