package httpapp

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/catalog"
	"github.com/jefersonMarques/apresentacao-viagate/internal/presentations"
	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

func (a *App) newPresentationPage(w http.ResponseWriter,r *http.Request){
	user,_:=currentUser(r.Context())
	input:=presentations.EditorInput{
		Title:"Apresentação Institucional ViaGate",
		ShowClientIdentity:true,
		ShowContactSlide:true,
		SelectedModules:defaultPresentationModules(),
		SalespersonName:user.Name,
		SalespersonEmail:user.Email,
	}
	render(r.Context(),w,http.StatusOK,templates.PresentationEditorPage(user,input,presentations.SavedDraft{},"",""))
}

func (a *App) editPresentationPage(w http.ResponseWriter,r *http.Request){
	user,_:=currentUser(r.Context())
	allowAll,_:=a.authStore.HasPermission(r.Context(),user.ID,"presentation.read_all")
	input,draft,err:=a.presentationStore.EditorByID(r.Context(),user.ID,chi.URLParam(r,"id"),allowAll)
	if err!=nil{http.Error(w,"apresentação não encontrada ou acesso negado",http.StatusNotFound);return}
	message:=""
	if r.URL.Query().Get("saved")=="1"{message="Rascunho salvo."}
	if r.URL.Query().Get("published")=="1"{message="Versão publicada. Novas alterações criarão uma nova versão."}
	render(r.Context(),w,http.StatusOK,templates.PresentationEditorPage(user,input,draft,message,""))
}

func (a *App) savePresentation(w http.ResponseWriter,r *http.Request){
	user,_:=currentUser(r.Context())
	allowAll,_:=a.authStore.HasPermission(r.Context(),user.ID,"presentation.read_all")
	if err:=r.ParseForm();err!=nil{http.Error(w,"dados inválidos",http.StatusBadRequest);return}
	input,err:=presentationInputFromForm(r,user.Name,user.Email)
	if err!=nil{render(r.Context(),w,http.StatusBadRequest,templates.PresentationEditorPage(user,input,presentations.SavedDraft{},"",err.Error()));return}
	draft,err:=a.presentationStore.SaveDraft(r.Context(),user.ID,allowAll,input)
	if err!=nil{a.logger.Error("save presentation failed","error",err);render(r.Context(),w,http.StatusBadRequest,templates.PresentationEditorPage(user,input,presentations.SavedDraft{},"","Não foi possível salvar a apresentação."));return}
	if r.FormValue("action")=="publish"{
		if _,err:=a.presentationStore.Publish(r.Context(),user.ID,allowAll,draft.VersionID);err!=nil{a.logger.Error("publish presentation failed","error",err);render(r.Context(),w,http.StatusBadRequest,templates.PresentationEditorPage(user,input,draft,"","Não foi possível publicar a apresentação."));return}
		http.Redirect(w,r,"/admin/presentations/"+draft.PresentationID+"/edit?published=1",http.StatusSeeOther);return
	}
	http.Redirect(w,r,"/admin/presentations/"+draft.PresentationID+"/edit?saved=1",http.StatusSeeOther)
}

func (a *App) publicPresentationPage(w http.ResponseWriter,r *http.Request){
	token:=chi.URLParam(r,"token")
	presentation,err:=a.presentationStore.PublicByToken(r.Context(),token)
	if err!=nil{
		if err==pgx.ErrNoRows{http.Error(w,"Apresentação não encontrada.",http.StatusNotFound);return}
		http.Error(w,"Apresentação indisponível.",http.StatusGone);return
	}
	sessionID,_:=newUUID()
	_,_ = a.pool.Exec(r.Context(),`
		insert into document_events(document_kind,document_version_id,event_type,viewer_session,ip_address,user_agent)
		values('presentation',$1,'open',$2,$3,$4)
	`,presentation.VersionID,nullableUUID(sessionID),requestIP(r),r.UserAgent())
	render(r.Context(),w,http.StatusOK,templates.PublicPresentationPage(presentation))
}

func presentationInputFromForm(r *http.Request,salespersonName,salespersonEmail string)(presentations.EditorInput,error){
	contactEmail,err:=cleanEmail(r.FormValue("contact_email"),false)
	input:=presentations.EditorInput{
		PresentationID:strings.TrimSpace(r.FormValue("presentation_id")),
		ClientLegalName:strings.TrimSpace(r.FormValue("client_legal_name")),
		ClientTradeName:strings.TrimSpace(r.FormValue("client_trade_name")),
		Title:strings.TrimSpace(r.FormValue("title")),
		ContactName:strings.TrimSpace(r.FormValue("contact_name")),
		ContactRole:strings.TrimSpace(r.FormValue("contact_role")),
		ContactEmail:contactEmail,
		ShowClientIdentity:r.FormValue("show_client_identity")=="1",
		ShowContactSlide:r.FormValue("show_contact_slide")=="1",
		SalespersonName:salespersonName,
		SalespersonEmail:salespersonEmail,
	}
	if err!=nil{return input,fmt.Errorf("E-mail do contato inválido.")}
	if input.Title==""{input.Title="Apresentação Institucional ViaGate"}
	if value:=strings.TrimSpace(r.FormValue("client_cnpj"));value!=""{cnpj,err:=cleanCNPJ(value);if err!=nil{return input,err};input.ClientCNPJ=cnpj}
	allowed:=map[string]bool{};for _,group:=range catalog.Groups{allowed[group.ID]=true}
	for _,module:=range r.Form["module"]{module=strings.TrimSpace(module);if !allowed[module]{return input,fmt.Errorf("módulo institucional inválido")};input.SelectedModules=append(input.SelectedModules,module)}
	if len(input.SelectedModules)==0{input.SelectedModules=defaultPresentationModules()}
	canonical:=struct{
		ClientLegalName string `json:"client_legal_name"`
		ClientTradeName string `json:"client_trade_name"`
		ClientCNPJ string `json:"client_cnpj"`
		Title string `json:"title"`
		ContactName string `json:"contact_name"`
		ContactRole string `json:"contact_role"`
		ContactEmail string `json:"contact_email"`
		ShowClientIdentity bool `json:"show_client_identity"`
		ShowContactSlide bool `json:"show_contact_slide"`
		SelectedModules []string `json:"selected_modules"`
		SalespersonName string `json:"salesperson_name"`
		SalespersonEmail string `json:"salesperson_email"`
	}{input.ClientLegalName,input.ClientTradeName,input.ClientCNPJ,input.Title,input.ContactName,input.ContactRole,input.ContactEmail,input.ShowClientIdentity,input.ShowContactSlide,input.SelectedModules,input.SalespersonName,input.SalespersonEmail}
	encoded,_:=json.Marshal(canonical);hash:=sha256.Sum256(encoded);input.ContentHash=hash[:]
	return input,nil
}

func defaultPresentationModules()[]string{result:=make([]string,0,len(catalog.Groups));for _,group:=range catalog.Groups{result=append(result,group.ID)};return result}
