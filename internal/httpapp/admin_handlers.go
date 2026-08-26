package httpapp

import (
	"crypto/sha256"
	"net/http"
	"strings"

	"github.com/jefersonMarques/apresentacao-viagate/internal/contracts"
	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

func (a *App) dashboard(w http.ResponseWriter,r *http.Request) {
	user,_:=currentUser(r.Context())
	render(r.Context(),w,http.StatusOK,templates.Dashboard(user))
}

func (a *App) usersPage(w http.ResponseWriter,r *http.Request) {
	user,_:=currentUser(r.Context())
	items,err:=a.authStore.ListUsers(r.Context())
	if err!=nil { http.Error(w,"não foi possível carregar os usuários",http.StatusInternalServerError);return }
	message:=""
	if r.URL.Query().Get("invited")=="1" { message="Convite criado e colocado na fila de envio do Brevo." }
	if r.URL.Query().Get("updated")=="1" { message="Acesso do usuário atualizado." }
	render(r.Context(),w,http.StatusOK,templates.UsersManagementPage(user,items,message,r.URL.Query().Get("error")))
}

func (a *App) contractTemplatesPage(w http.ResponseWriter,r *http.Request) {
	user,_:=currentUser(r.Context())
	items,err:=a.contractStore.ListTemplates(r.Context())
	if err!=nil { http.Error(w,"não foi possível carregar os modelos",http.StatusInternalServerError);return }
	message:=""
	if r.URL.Query().Get("saved")=="1" { message="Nova versão do modelo salva." }
	render(r.Context(),w,http.StatusOK,templates.ContractTemplatesPage(user,items,contracts.AllowedVariables(),message))
}

func (a *App) saveContractTemplate(w http.ResponseWriter,r *http.Request) {
	user,_:=currentUser(r.Context())
	if err:=r.ParseForm();err!=nil { http.Error(w,"dados inválidos",http.StatusBadRequest);return }
	templateID:=strings.TrimSpace(r.FormValue("template_id"))
	name:=strings.TrimSpace(r.FormValue("name"))
	description:=strings.TrimSpace(r.FormValue("description"))
	markdown:=strings.TrimSpace(r.FormValue("markdown"))
	if name==""||markdown=="" { http.Error(w,"nome e conteúdo são obrigatórios",http.StatusBadRequest);return }

	hash:=sha256.Sum256([]byte(markdown))
	version,err:=a.contractStore.SaveTemplateVersion(r.Context(),templateID,name,description,markdown,hash[:],user.ID)
	if err!=nil { a.logger.Error("save contract template failed","error",err);http.Error(w,"não foi possível salvar o modelo",http.StatusBadRequest);return }
	if r.FormValue("make_default")=="1" {
		if err:=a.contractStore.SetDefaultTemplate(r.Context(),version.ContractTemplateID);err!=nil { a.logger.Error("set default template failed","error",err);http.Error(w,"modelo salvo, mas não foi possível defini-lo como padrão",http.StatusInternalServerError);return }
	}
	_,_ = a.pool.Exec(r.Context(),`
		insert into audit_events(actor_user_id,event_type,resource_type,resource_id,metadata)
		values($1,'contract_template.version_created','contract_template',$2,jsonb_build_object('version',$3))
	`,user.ID,version.ContractTemplateID,version.VersionNumber)
	http.Redirect(w,r,"/admin/contracts/templates?saved=1",http.StatusSeeOther)
}

func (a *App) adminProposals(w http.ResponseWriter,r *http.Request) {
	user,_:=currentUser(r.Context())
	all:=false
	if allowed,err:=a.authStore.HasPermission(r.Context(),user.ID,"proposal.read_all");err==nil { all=allowed }
	items,err:=a.proposalStore.List(r.Context(),user.ID,all)
	if err!=nil { http.Error(w,"não foi possível carregar as propostas",http.StatusInternalServerError);return }
	render(r.Context(),w,http.StatusOK,templates.ProposalListPage(user,items))
}

func (a *App) adminPresentations(w http.ResponseWriter,r *http.Request) {
	user,_:=currentUser(r.Context())
	all:=false
	if allowed,err:=a.authStore.HasPermission(r.Context(),user.ID,"presentation.read_all");err==nil { all=allowed }
	items,err:=a.presentationStore.List(r.Context(),user.ID,all)
	if err!=nil { http.Error(w,"não foi possível carregar as apresentações",http.StatusInternalServerError);return }
	render(r.Context(),w,http.StatusOK,templates.PresentationListPage(user,items))
}
