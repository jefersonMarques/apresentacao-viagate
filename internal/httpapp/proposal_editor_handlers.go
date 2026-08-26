package httpapp

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/catalog"
	"github.com/jefersonMarques/apresentacao-viagate/internal/proposals"
	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

func (a *App) newProposalPage(w http.ResponseWriter,r *http.Request){
	user,_:=currentUser(r.Context())
	validUntil:=time.Now().AddDate(0,0,15)
	input:=proposals.EditorInput{Title:"Proposta Comercial ViaGate",PricingModel:"per_item",ValidUntil:&validUntil}
	render(r.Context(),w,http.StatusOK,templates.ProposalEditorPage(user,input,proposals.SavedDraft{},"",""))
}

func (a *App) editProposalPage(w http.ResponseWriter,r *http.Request){
	user,_:=currentUser(r.Context())
	allowAll,_:=a.authStore.HasPermission(r.Context(),user.ID,"proposal.read_all")
	input,draft,err:=a.proposalStore.EditorByID(r.Context(),user.ID,chi.URLParam(r,"id"),allowAll)
	if err!=nil{http.Error(w,"proposta não encontrada ou acesso negado",http.StatusNotFound);return}
	message:=""
	if r.URL.Query().Get("saved")=="1"{message="Rascunho salvo."}
	if r.URL.Query().Get("published")=="1"{message="Versão publicada. Novas alterações criarão uma nova versão."}
	render(r.Context(),w,http.StatusOK,templates.ProposalEditorPage(user,input,draft,message,""))
}

func (a *App) saveProposal(w http.ResponseWriter,r *http.Request){
	user,_:=currentUser(r.Context())
	allowAll,_:=a.authStore.HasPermission(r.Context(),user.ID,"proposal.read_all")
	if err:=r.ParseForm();err!=nil{http.Error(w,"dados inválidos",http.StatusBadRequest);return}

	input,err:=a.proposalInputFromForm(r,user.Name,user.Email)
	if err!=nil{
		render(r.Context(),w,http.StatusBadRequest,templates.ProposalEditorPage(user,input,proposals.SavedDraft{},"",err.Error()))
		return
	}
	draft,err:=a.proposalStore.SaveDraft(r.Context(),user.ID,allowAll,input)
	if err!=nil{a.logger.Error("save proposal failed","error",err);render(r.Context(),w,http.StatusBadRequest,templates.ProposalEditorPage(user,input,proposals.SavedDraft{},"","Não foi possível salvar a proposta. Ela pode já ter sido aceita ou estar bloqueada."));return}

	if r.FormValue("action")=="publish"{
		if len(input.Items)==0{render(r.Context(),w,http.StatusBadRequest,templates.ProposalEditorPage(user,input,draft,"","Selecione ao menos um item para publicar."));return}
		if _,err:=a.proposalStore.Publish(r.Context(),user.ID,allowAll,draft.VersionID);err!=nil{a.logger.Error("publish proposal failed","error",err);render(r.Context(),w,http.StatusBadRequest,templates.ProposalEditorPage(user,input,draft,"","Não foi possível publicar a proposta."));return}
		http.Redirect(w,r,"/admin/proposals/"+draft.ProposalID+"/edit?published=1",http.StatusSeeOther)
		return
	}
	http.Redirect(w,r,"/admin/proposals/"+draft.ProposalID+"/edit?saved=1",http.StatusSeeOther)
}

func (a *App) proposalInputFromForm(r *http.Request,salespersonName,salespersonEmail string)(proposals.EditorInput,error){
	var input proposals.EditorInput
	input.ProposalID=strings.TrimSpace(r.FormValue("proposal_id"))
	input.ClientLegalName=strings.TrimSpace(r.FormValue("client_legal_name"));if input.ClientLegalName==""{return input,fmt.Errorf("Informe a razão social do cliente.")}
	input.ClientTradeName=strings.TrimSpace(r.FormValue("client_trade_name"))
	if value:=strings.TrimSpace(r.FormValue("client_cnpj"));value!=""{cnpj,err:=cleanCNPJ(value);if err!=nil{return input,err};input.ClientCNPJ=cnpj}
	clientEmail,err:=cleanEmail(r.FormValue("client_email"),false);if err!=nil{return input,fmt.Errorf("E-mail do cliente inválido.")};input.ClientEmail=clientEmail
	input.ClientPhone=strings.TrimSpace(r.FormValue("client_phone"))
	input.Title=strings.TrimSpace(r.FormValue("title"));if input.Title==""{input.Title="Proposta Comercial ViaGate"}
	if value:=strings.TrimSpace(r.FormValue("valid_until"));value!=""{date,err:=time.Parse("2006-01-02",value);if err!=nil{return input,fmt.Errorf("Validade inválida.")};input.ValidUntil=&date}
	input.PricingModel=strings.TrimSpace(r.FormValue("pricing_model"));if !validPricingModel(input.PricingModel){return input,fmt.Errorf("Modelo comercial inválido.")}

	minimumInvoice,err:=parseMoney(r.FormValue("minimum_invoice"));if err!=nil{return input,fmt.Errorf("Fatura mínima inválida.")};input.MinimumInvoice=minimumInvoice
	setupFee,err:=parseMoney(r.FormValue("setup_fee"));if err!=nil{return input,fmt.Errorf("Valor de implantação inválido.")};input.SetupFee=setupFee

	ids:=r.Form["catalog_id"];statuses:=r.Form["item_status"];prices:=r.Form["item_price"]
	for index,id:=range ids{
		status:="off";if index<len(statuses){status=statuses[index]}
		if status!="off"&&status!="included"&&status!="optional"{return input,fmt.Errorf("Status de item inválido.")}
		if status=="off"{continue}
		group,item,ok:=catalog.ItemByID(id);if !ok{return input,fmt.Errorf("Item comercial inválido: %s",id)}
		if !catalog.ModelAllows(item,input.PricingModel){continue}
		price:=0.0
		if index<len(prices){price,err=parseMoney(prices[index]);if err!=nil{return input,fmt.Errorf("Valor inválido para %s.",item.Label)}}
		input.Items=append(input.Items,proposals.EditorItem{CatalogID:item.ID,GroupName:group.Title,Label:item.Label,Unit:item.Unit,Price:price,IsOptional:status=="optional",SortOrder:index})
	}
	input.Conditions=normalizedConditions(r.Form["condition"],r.FormValue("custom_conditions"))
	validUntil:="";if input.ValidUntil!=nil{validUntil=input.ValidUntil.Format("2006-01-02")}
	input.Content=map[string]any{
		"proposal":map[string]any{"title":input.Title,"valid_until":validUntil},
		"client":map[string]any{"legal_name":input.ClientLegalName,"trade_name":input.ClientTradeName,"cnpj":input.ClientCNPJ,"email":input.ClientEmail,"phone":input.ClientPhone},
		"salesperson":map[string]any{"name":salespersonName,"email":salespersonEmail},
	}
	canonical:=struct{Content map[string]any `json:"content"`;PricingModel string `json:"pricing_model"`;MinimumInvoice float64 `json:"minimum_invoice"`;SetupFee float64 `json:"setup_fee"`;Conditions []string `json:"conditions"`;Items []proposals.EditorItem `json:"items"`}{input.Content,input.PricingModel,input.MinimumInvoice,input.SetupFee,input.Conditions,input.Items}
	encoded,_:=json.Marshal(canonical);hash:=sha256.Sum256(encoded);input.ContentHash=hash[:]
	return input,nil
}

func validPricingModel(value string)bool{for _,model:=range catalog.PricingModels{if model.ID==value{return true}};return false}

func parseMoney(value string)(float64,error){
	value=strings.TrimSpace(value)
	if value==""{return 0,nil}
	if strings.Contains(value,","){
		if strings.Contains(value,"."){value=strings.ReplaceAll(value,".","")}
		value=strings.ReplaceAll(value,",",".")
	}
	number,err:=strconv.ParseFloat(value,64)
	if err!=nil||math.IsNaN(number)||math.IsInf(number,0)||number<0{return 0,fmt.Errorf("invalid money")}
	return number,nil
}

func normalizedConditions(standard []string,custom string)[]string{
	seen:=map[string]bool{}
	result:=[]string{}
	appendValue:=func(value string){
		value=strings.TrimSpace(value)
		if value==""||seen[value]{return}
		seen[value]=true
		result=append(result,value)
	}
	for _,value:=range standard{appendValue(value)}
	for _,line:=range strings.Split(custom,"\n"){appendValue(line)}
	return result
}
