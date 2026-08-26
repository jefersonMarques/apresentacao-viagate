package httpapp

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

func (a *App) profilePage(w http.ResponseWriter,r *http.Request) {
	current,_:=currentUser(r.Context())
	profile,err:=a.authStore.Profile(r.Context(),current.ID)
	if err!=nil{http.Error(w,"não foi possível carregar o perfil",http.StatusInternalServerError);return}
	message:=""
	if r.URL.Query().Get("saved")=="1"{message="Perfil atualizado. Os novos dados serão usados nas próximas versões salvas."}
	render(r.Context(),w,http.StatusOK,templates.ProfilePage(current,profile,message,""))
}

func (a *App) updateProfile(w http.ResponseWriter,r *http.Request) {
	current,_:=currentUser(r.Context())
	if err:=r.ParseForm();err!=nil{http.Error(w,"dados inválidos",http.StatusBadRequest);return}
	name:=strings.TrimSpace(r.FormValue("name"))
	jobTitle:=strings.TrimSpace(r.FormValue("job_title"))
	email,err:=cleanEmail(r.FormValue("email"),true)
	if err!=nil||name==""{
		profile,_:=a.authStore.Profile(r.Context(),current.ID)
		profile.Name=name;profile.Email=strings.TrimSpace(r.FormValue("email"));profile.JobTitle=jobTitle
		render(r.Context(),w,http.StatusBadRequest,templates.ProfilePage(current,profile,"","Informe nome e e-mail válidos."));return
	}
	phone,err:=cleanPhone(r.FormValue("phone"),false)
	if err!=nil{
		profile,_:=a.authStore.Profile(r.Context(),current.ID)
		profile.Name=name;profile.Email=email;profile.Phone=strings.TrimSpace(r.FormValue("phone"));profile.JobTitle=jobTitle
		render(r.Context(),w,http.StatusBadRequest,templates.ProfilePage(current,profile,"","Telefone inválido."));return
	}
	photoURL,err:=cleanProfileURL(r.FormValue("photo_url"))
	if err!=nil{a.renderProfileValidationError(w,r,current.ID,name,email,phone,jobTitle,"URL da foto inválida.");return}
	linkedInURL,err:=cleanProfileURL(r.FormValue("linkedin_url"))
	if err!=nil{a.renderProfileValidationError(w,r,current.ID,name,email,phone,jobTitle,"URL do LinkedIn inválida.");return}
	instagramURL,err:=cleanProfileURL(r.FormValue("instagram_url"))
	if err!=nil{a.renderProfileValidationError(w,r,current.ID,name,email,phone,jobTitle,"URL do Instagram inválida.");return}

	if err:=a.authStore.UpdateProfile(r.Context(),current.ID,name,email,phone,jobTitle,photoURL,linkedInURL,instagramURL);err!=nil{
		a.logger.Error("update profile failed","user_id",current.ID,"error",err)
		profile,_:=a.authStore.Profile(r.Context(),current.ID)
		render(r.Context(),w,http.StatusBadRequest,templates.ProfilePage(current,profile,"","Não foi possível salvar o perfil. Verifique se o e-mail já está em uso."));return
	}
	_,_=a.pool.Exec(r.Context(),`
		insert into audit_events(actor_user_id,event_type,resource_type,resource_id,metadata)
		values($1,'user.profile_updated','user',$1,jsonb_build_object(
			'email',$2,'phone_set',$3,'job_title_set',$4,'photo_set',$5,'linkedin_set',$6,'instagram_set',$7
		))
	`,current.ID,email,phone!="",jobTitle!="",photoURL!="",linkedInURL!="",instagramURL!="")
	http.Redirect(w,r,"/admin/profile?saved=1",http.StatusSeeOther)
}

func (a *App) renderProfileValidationError(w http.ResponseWriter,r *http.Request,userID,name,email,phone,jobTitle,message string){
	profile,_:=a.authStore.Profile(r.Context(),userID)
	profile.Name=name;profile.Email=email;profile.Phone=phone;profile.JobTitle=jobTitle
	profile.PhotoURL=strings.TrimSpace(r.FormValue("photo_url"))
	profile.LinkedInURL=strings.TrimSpace(r.FormValue("linkedin_url"))
	profile.InstagramURL=strings.TrimSpace(r.FormValue("instagram_url"))
	current,_:=currentUser(r.Context())
	render(r.Context(),w,http.StatusBadRequest,templates.ProfilePage(current,profile,"",message))
}

func cleanProfileURL(value string)(string,error){
	value=strings.TrimSpace(value)
	if value==""{return "",nil}
	parsed,err:=url.ParseRequestURI(value)
	if err!=nil||parsed.Host==""||(parsed.Scheme!="https"&&parsed.Scheme!="http"){return "",errors.New("invalid URL")}
	return parsed.String(),nil
}
