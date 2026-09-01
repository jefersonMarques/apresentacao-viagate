package httpapp

import (
	"crypto/sha256"
	"net/http"
	"strings"

	"github.com/jefersonMarques/apresentacao-viagate/internal/access"
	"github.com/jefersonMarques/apresentacao-viagate/internal/contracts"
	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

func (a *App) dashboard(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())

	proposalItems := []domain.Proposal{}
	if proposalAll := access.Can(user, access.ProposalReadAll); proposalAll || access.Can(user, access.ProposalReadOwn) {
		items, err := a.proposalStore.List(r.Context(), user.ID, proposalAll)
		if err != nil {
			http.Error(w, "não foi possível carregar os registros", http.StatusInternalServerError)
			return
		}
		proposalItems = items
	}

	presentationItems := []domain.Presentation{}
	if presentationAll := access.Can(user, access.PresentationReadAll); presentationAll || access.Can(user, access.PresentationReadOwn) {
		items, err := a.presentationStore.List(r.Context(), user.ID, presentationAll)
		if err != nil {
			http.Error(w, "não foi possível carregar os registros", http.StatusInternalServerError)
			return
		}
		presentationItems = items
	}

	records := templates.DashboardRecords(proposalItems, presentationItems)
	render(r.Context(), w, http.StatusOK, templates.AdminDashboardPage(user, records))
}

func (a *App) usersPage(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	items, err := a.authStore.ListUsers(r.Context())
	if err != nil {
		http.Error(w, "não foi possível carregar os usuários", http.StatusInternalServerError)
		return
	}

	selected := domain.User{}
	settings := []domain.PermissionSetting{}
	preferences := []domain.NotificationPreference{}
	if selectedID := strings.TrimSpace(r.URL.Query().Get("user")); selectedID != "" {
		selected, err = a.authStore.ManagedUserByID(r.Context(), selectedID)
		if err != nil {
			http.Error(w, "usuário não encontrado", http.StatusNotFound)
			return
		}
		settings, err = a.authStore.PermissionSettings(r.Context(), selectedID)
		if err != nil {
			http.Error(w, "não foi possível carregar as permissões", http.StatusInternalServerError)
			return
		}
		preferences, err = a.authStore.NotificationPreferences(r.Context(), selectedID)
		if err != nil {
			http.Error(w, "não foi possível carregar as preferências", http.StatusInternalServerError)
			return
		}
	}

	message := ""
	if r.URL.Query().Get("invited") == "1" {
		message = "Convite criado e colocado na fila de envio do Brevo."
	}
	if r.URL.Query().Get("updated") == "1" {
		message = "Acesso do usuário atualizado."
	}
	if r.URL.Query().Get("permissions") == "1" {
		message = "Permissões individuais atualizadas."
	}
	if r.URL.Query().Get("notifications") == "1" {
		message = "Preferências de notificação atualizadas."
	}
	render(r.Context(), w, http.StatusOK, templates.UsersManagementPage(user, items, selected, settings, preferences, message, r.URL.Query().Get("error")))
}

func (a *App) contractTemplatesPage(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	items, err := a.contractStore.ListTemplates(r.Context())
	if err != nil {
		http.Error(w, "não foi possível carregar os modelos", http.StatusInternalServerError)
		return
	}

	selectedID := strings.TrimSpace(r.URL.Query().Get("template"))
	selectedName := ""
	selectedDescription := ""
	selectedMarkdown := ""
	if selectedID != "" {
		found := false
		for _, item := range items {
			if item.ID != selectedID {
				continue
			}
			found = true
			selectedName = item.Name
			selectedDescription = item.Description
			break
		}
		if !found {
			http.Error(w, "modelo de contrato não encontrado", http.StatusNotFound)
			return
		}
		version, versionErr := a.contractStore.LatestTemplateVersion(r.Context(), selectedID)
		if versionErr != nil {
			a.logger.Error("load latest contract template failed", "template_id", selectedID, "error", versionErr)
			http.Error(w, "não foi possível carregar a versão atual do modelo", http.StatusInternalServerError)
			return
		}
		selectedMarkdown = version.Markdown
	}

	message := ""
	if r.URL.Query().Get("saved") == "1" {
		message = "Nova versão do modelo salva após validação completa."
	}
	render(r.Context(), w, http.StatusOK, templates.ContractTemplatesPage(
		user,
		items,
		contracts.AllowedVariables(),
		message,
		selectedID,
		selectedName,
		selectedDescription,
		selectedMarkdown,
	))
}

func (a *App) saveContractTemplate(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, "dados inválidos", http.StatusBadRequest)
		return
	}
	templateID := strings.TrimSpace(r.FormValue("template_id"))
	name := strings.TrimSpace(r.FormValue("name"))
	description := strings.TrimSpace(r.FormValue("description"))
	markdown := strings.TrimSpace(r.FormValue("markdown"))
	if name == "" || markdown == "" {
		http.Error(w, "nome e conteúdo são obrigatórios", http.StatusBadRequest)
		return
	}

	if err := a.contractGenerator.ValidateTemplate(r.Context(), markdown); err != nil {
		a.logger.Warn("contract template validation failed", "error", err, "user_id", user.ID)
		http.Error(w, "O modelo não pode ser salvo porque a prévia falhou: "+err.Error(), http.StatusBadRequest)
		return
	}

	hash := sha256.Sum256([]byte(markdown))
	version, err := a.contractStore.SaveTemplateVersion(r.Context(), templateID, name, description, markdown, hash[:], user.ID)
	if err != nil {
		a.logger.Error("save contract template failed", "error", err)
		http.Error(w, "não foi possível salvar o modelo", http.StatusBadRequest)
		return
	}
	if r.FormValue("make_default") == "1" {
		if err := a.contractStore.SetDefaultTemplate(r.Context(), version.ContractTemplateID); err != nil {
			a.logger.Error("set default template failed", "error", err)
			http.Error(w, "modelo salvo, mas não foi possível defini-lo como padrão", http.StatusInternalServerError)
			return
		}
	}
	_, _ = a.pool.Exec(r.Context(), `
		insert into audit_events(actor_user_id,event_type,resource_type,resource_id,metadata)
		values($1,'contract_template.version_created','contract_template',$2,jsonb_build_object('version',$3::integer))
	`, user.ID, version.ContractTemplateID, version.VersionNumber)
	http.Redirect(w, r, "/admin/contracts/templates?saved=1&template="+version.ContractTemplateID, http.StatusSeeOther)
}

func (a *App) adminProposals(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	all := access.Can(user, access.ProposalReadAll)
	if !all && !access.Can(user, access.ProposalReadOwn) {
		http.Error(w, "acesso negado", http.StatusForbidden)
		return
	}
	items, err := a.proposalStore.List(r.Context(), user.ID, all)
	if err != nil {
		http.Error(w, "não foi possível carregar as propostas", http.StatusInternalServerError)
		return
	}
	render(r.Context(), w, http.StatusOK, templates.ProposalListPage(user, items))
}

func (a *App) adminPresentations(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	all := access.Can(user, access.PresentationReadAll)
	if !all && !access.Can(user, access.PresentationReadOwn) {
		http.Error(w, "acesso negado", http.StatusForbidden)
		return
	}
	items, err := a.presentationStore.List(r.Context(), user.ID, all)
	if err != nil {
		http.Error(w, "não foi possível carregar as apresentações", http.StatusInternalServerError)
		return
	}
	render(r.Context(), w, http.StatusOK, templates.PresentationListPage(user, items))
}
