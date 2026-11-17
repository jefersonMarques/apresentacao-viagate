package httpapp

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/contracts"
	"github.com/jefersonMarques/apresentacao-viagate/internal/postsale"
	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

func (a *App) signedPostSaleAccess(w http.ResponseWriter, r *http.Request) (contracts.SignerAccess, bool) {
	access, err := a.contractStore.SignerByPublicToken(r.Context(), chi.URLParam(r, "token"))
	if err != nil {
		http.Error(w, "Link inválido.", http.StatusNotFound)
		return contracts.SignerAccess{}, false
	}
	if access.Signer.Status != "signed" || access.Contract.Status != "signed" {
		http.Error(w, "O contato de implantação estará disponível após a assinatura.", http.StatusConflict)
		return contracts.SignerAccess{}, false
	}
	return access, true
}

func (a *App) postSaleContactPage(w http.ResponseWriter, r *http.Request) {
	access, ok := a.signedPostSaleAccess(w, r)
	if !ok {
		return
	}
	contact, err := a.postSaleStore.ByContract(r.Context(), access.Contract.ID)
	if err != nil {
		a.logger.Error("load implementation contact failed", "contract_id", access.Contract.ID, "error", err)
		http.Error(w, "Não foi possível carregar o contato de implantação.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	message := ""
	if r.URL.Query().Get("saved") == "1" {
		message = "Contato de implantação confirmado."
	}
	render(r.Context(), w, http.StatusOK, templates.PostSaleContactPage(chi.URLParam(r, "token"), contact, message, ""))
}

func (a *App) savePostSaleContact(w http.ResponseWriter, r *http.Request) {
	access, ok := a.signedPostSaleAccess(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Dados inválidos.", http.StatusBadRequest)
		return
	}

	contact := postsale.Contact{Mode: strings.TrimSpace(r.FormValue("contact_mode"))}
	if contact.Mode == "self" {
		contact.Name = access.Signer.Name
		contact.Email = access.Signer.Email
		if err := a.pool.QueryRow(r.Context(), `
			select o.company_responsible_phone
			from contracts c
			join onboardings o on o.id=c.onboarding_id
			where c.id=$1 and c.deleted_at is null and o.deleted_at is null
		`, access.Contract.ID).Scan(&contact.Phone); err != nil {
			http.Error(w, "Não foi possível localizar o telefone do responsável.", http.StatusInternalServerError)
			return
		}
	} else if contact.Mode == "team" {
		contact.Name = strings.TrimSpace(r.FormValue("contact_name"))
		phone, err := cleanPhone(r.FormValue("contact_phone"), true)
		if err != nil {
			render(r.Context(), w, http.StatusBadRequest, templates.PostSaleContactPage(chi.URLParam(r, "token"), contact, "", "Informe um telefone válido."))
			return
		}
		contact.Phone = phone
		email, err := cleanEmail(r.FormValue("contact_email"), false)
		if err != nil {
			render(r.Context(), w, http.StatusBadRequest, templates.PostSaleContactPage(chi.URLParam(r, "token"), contact, "", "Informe um e-mail válido ou deixe o campo vazio."))
			return
		}
		contact.Email = email
	} else {
		http.Error(w, "Selecione quem cuidará da implantação.", http.StatusBadRequest)
		return
	}

	if err := postsale.ValidateContact(contact); err != nil {
		render(r.Context(), w, http.StatusBadRequest, templates.PostSaleContactPage(chi.URLParam(r, "token"), contact, "", "Informe o nome e o telefone do contato de implantação."))
		return
	}
	if err := a.postSaleStore.Save(r.Context(), access.Contract.ID, contact, postsale.Actor{
		Type: "customer", SignerID: access.Signer.ID, IP: requestIP(r), Agent: r.UserAgent(),
	}); err != nil {
		a.logger.Error("save implementation contact failed", "contract_id", access.Contract.ID, "error", err)
		http.Error(w, "Não foi possível salvar o contato. Tente novamente.", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/sign/"+chi.URLParam(r, "token")+"/contact?saved=1", http.StatusSeeOther)
}
