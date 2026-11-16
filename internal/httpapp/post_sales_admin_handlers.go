package httpapp

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/access"
	"github.com/jefersonMarques/apresentacao-viagate/internal/postsale"
	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

func (a *App) postSalesPage(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	all := access.Can(user, access.ProposalReadAll)
	if !all && !access.Can(user, access.ProposalReadOwn) {
		http.Error(w, "acesso negado", http.StatusForbidden)
		return
	}
	items, err := a.postSaleStore.List(r.Context(), user.ID, all)
	if err != nil {
		a.logger.Error("list post-sale customers failed", "error", err)
		http.Error(w, "Não foi possível carregar o pós-venda.", http.StatusInternalServerError)
		return
	}
	render(r.Context(), w, http.StatusOK, templates.PostSalesPage(user, items))
}

func (a *App) postSaleDetail(w http.ResponseWriter, r *http.Request) (postsale.Detail, bool) {
	user, _ := currentUser(r.Context())
	all := access.Can(user, access.ProposalReadAll)
	if !all && !access.Can(user, access.ProposalReadOwn) {
		http.Error(w, "acesso negado", http.StatusForbidden)
		return postsale.Detail{}, false
	}
	detail, err := a.postSaleStore.Detail(r.Context(), chi.URLParam(r, "contractID"), user.ID, all)
	if err == pgx.ErrNoRows {
		http.Error(w, "Cliente não encontrado ou sem acesso.", http.StatusNotFound)
		return postsale.Detail{}, false
	}
	if err != nil {
		a.logger.Error("load post-sale customer failed", "error", err)
		http.Error(w, "Não foi possível carregar o pós-venda.", http.StatusInternalServerError)
		return postsale.Detail{}, false
	}
	return detail, true
}

func (a *App) postSaleDetailPage(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	detail, ok := a.postSaleDetail(w, r)
	if !ok {
		return
	}
	message := ""
	if r.URL.Query().Get("saved") == "1" {
		message = "Contato atualizado e registrado no histórico."
	}
	if r.URL.Query().Get("saved") == "operation" {
		message = "Dados operacionais salvos com histórico."
	}
	render(r.Context(), w, http.StatusOK, templates.PostSaleDetailPage(user, detail, message, ""))
}

func (a *App) updatePostSaleContact(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	detail, ok := a.postSaleDetail(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Dados inválidos.", http.StatusBadRequest)
		return
	}
	phone, err := cleanPhone(r.FormValue("contact_phone"), true)
	if err != nil {
		render(r.Context(), w, http.StatusBadRequest, templates.PostSaleDetailPage(user, detail, "", "Informe um telefone válido."))
		return
	}
	email, err := cleanEmail(r.FormValue("contact_email"), false)
	if err != nil {
		render(r.Context(), w, http.StatusBadRequest, templates.PostSaleDetailPage(user, detail, "", "Informe um e-mail válido ou deixe o campo vazio."))
		return
	}
	contact := postsale.Contact{
		Mode: "team", Name: strings.TrimSpace(r.FormValue("contact_name")),
		Phone: phone, Email: email,
	}
	if err := postsale.ValidateContact(contact); err != nil {
		render(r.Context(), w, http.StatusBadRequest, templates.PostSaleDetailPage(user, detail, "", "Informe nome e telefone."))
		return
	}
	if err := a.postSaleStore.Save(r.Context(), detail.ContractID, contact, postsale.Actor{
		Type: "user", UserID: user.ID, IP: requestIP(r), Agent: r.UserAgent(),
	}); err != nil {
		a.logger.Error("update post-sale contact failed", "error", err)
		http.Error(w, "Não foi possível salvar o contato.", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/post-sales/"+detail.ContractID+"?saved=1", http.StatusSeeOther)
}

func (a *App) updatePostSaleOperation(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	detail, ok := a.postSaleDetail(w, r)
	if !ok {
		return
	}
	if !detail.RequiresPolicy {
		http.Error(w, "Esta contratação não exige dados de seguro.", http.StatusConflict)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Dados inválidos.", http.StatusBadRequest)
		return
	}
	operation := postsale.Operation{
		Type: strings.ToLower(strings.TrimSpace(r.FormValue("operation_type"))),
		Insurer: strings.TrimSpace(r.FormValue("insurer")),
		StartDate: strings.TrimSpace(r.FormValue("policy_start_date")),
		EndDate: strings.TrimSpace(r.FormValue("policy_end_date")),
		BrokerCompany: strings.TrimSpace(r.FormValue("broker_company")),
		BrokerProducer: strings.TrimSpace(r.FormValue("broker_producer")),
	}
	if err := postsale.ValidateOperation(operation); err != nil {
		detail.Operation = operation
		render(r.Context(), w, http.StatusBadRequest, templates.PostSaleDetailPage(user, detail, "", "Verifique o tipo de operação e a vigência da apólice."))
		return
	}
	if err := a.postSaleStore.SaveOperation(r.Context(), detail.ContractID, operation, postsale.Actor{
		Type: "user", UserID: user.ID, Name: user.Name, IP: requestIP(r), Agent: r.UserAgent(),
	}); err != nil {
		a.logger.Error("save post-sale operation failed", "error", err, "contract_id", detail.ContractID)
		http.Error(w, "Não foi possível salvar os dados operacionais.", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/post-sales/"+detail.ContractID+"?saved=operation", http.StatusSeeOther)
}
