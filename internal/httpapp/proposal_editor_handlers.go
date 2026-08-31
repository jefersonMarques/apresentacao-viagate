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
	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
	"github.com/jefersonMarques/apresentacao-viagate/internal/proposals"
	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

func (a *App) newProposalPage(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	duplicateID := strings.TrimSpace(r.URL.Query().Get("duplicate"))
	if duplicateID != "" {
		allowAll, _ := a.authStore.HasPermission(r.Context(), user.ID, "proposal.read_all")
		source, err := a.proposalStore.DuplicateSourceByID(r.Context(), user.ID, duplicateID, allowAll)
		if err != nil {
			http.Error(w, "proposta não encontrada ou acesso negado", http.StatusNotFound)
			return
		}
		salesperson, profileErr := a.authStore.Profile(r.Context(), user.ID)
		if profileErr != nil {
			a.logger.Error("load commercial profile for proposal duplicate failed", "user_id", user.ID, "error", profileErr)
			salesperson = user
		}
		input := a.prepareDuplicatedProposalInput(r.Context(), source, salesperson)
		input = a.decorateProposalContractOptions(r.Context(), input)
		render(r.Context(), w, http.StatusOK, templates.ProposalEditorPage(
			user,
			input,
			proposals.SavedDraft{},
			"Cópia preparada como nova proposta. Os dados do cliente, contato, contexto da operação e prioridades foram removidos. Revise e salve para criar o novo rascunho.",
			"",
		))
		return
	}

	validUntil := time.Now().AddDate(0, 0, 15)
	input := proposals.EditorInput{Title: "Proposta Comercial ViaGate", PricingModel: "per_item", ValidUntil: &validUntil}
	input = a.decorateProposalContractOptions(r.Context(), input)
	render(r.Context(), w, http.StatusOK, templates.ProposalEditorPage(user, input, proposals.SavedDraft{}, "", ""))
}

func (a *App) editProposalPage(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	allowAll, _ := a.authStore.HasPermission(r.Context(), user.ID, "proposal.read_all")
	proposalID := chi.URLParam(r, "id")
	input, draft, err := a.proposalStore.EditorByID(r.Context(), user.ID, proposalID, allowAll)
	if err != nil {
		http.Error(w, "proposta não encontrada ou acesso negado", http.StatusNotFound)
		return
	}

	var status, currentPublicToken string
	if err := a.pool.QueryRow(r.Context(), `
		select p.status::text,coalesce(v.public_token::text,'')
		from proposals p
		left join proposal_versions v
		  on v.proposal_id=p.id
		 and v.version_number=p.current_version
		 and v.published_at is not null
		where p.id=$1
	`, proposalID).Scan(&status, &currentPublicToken); err != nil {
		http.Error(w, "não foi possível carregar a proposta", http.StatusInternalServerError)
		return
	}
	if status == "accepted" {
		if currentPublicToken != "" {
			http.Redirect(w, r, "/p/"+currentPublicToken, http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "/admin/proposals", http.StatusSeeOther)
		}
		return
	}
	if status == "cancelled" {
		http.Redirect(w, r, "/admin/proposals", http.StatusSeeOther)
		return
	}

	input = a.decorateProposalContractOptions(r.Context(), input)
	message := ""
	if r.URL.Query().Get("saved") == "1" {
		message = "Rascunho salvo."
	}
	if r.URL.Query().Get("published") == "1" {
		message = "Versão publicada. Novas alterações criarão uma nova versão."
	}
	render(r.Context(), w, http.StatusOK, templates.ProposalEditorPage(user, input, draft, message, ""))
}

func (a *App) saveProposal(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r.Context())
	allowAll, _ := a.authStore.HasPermission(r.Context(), user.ID, "proposal.read_all")
	if err := r.ParseForm(); err != nil {
		http.Error(w, "dados inválidos", http.StatusBadRequest)
		return
	}

	action := strings.TrimSpace(r.URL.Query().Get("action"))
	if action == "" {
		action = strings.TrimSpace(r.FormValue("action"))
	}
	if action == "" {
		action = "draft"
	}
	if action != "draft" && action != "publish" {
		http.Error(w, "ação inválida", http.StatusBadRequest)
		return
	}

	salesperson, profileErr := a.authStore.Profile(r.Context(), user.ID)
	if profileErr != nil {
		a.logger.Error("load commercial profile failed", "user_id", user.ID, "error", profileErr)
		salesperson = user
	}
	input, err := a.proposalInputFromForm(r, salesperson)
	if err != nil {
		input = a.decorateProposalContractOptions(r.Context(), input)
		render(r.Context(), w, http.StatusBadRequest, templates.ProposalEditorPage(user, input, proposals.SavedDraft{}, "", err.Error()))
		return
	}

	draft, err := a.proposalStore.SaveDraft(r.Context(), user.ID, allowAll, input)
	if err != nil {
		a.logger.Error("save proposal failed", "user_id", user.ID, "proposal_id", input.ProposalID, "action", action, "error", err)
		message := "Não foi possível salvar a proposta. Ela pode já ter sido aceita ou estar bloqueada."
		if a.cfg.Environment != "production" {
			message += " Detalhe: " + err.Error()
		}
		input = a.decorateProposalContractOptions(r.Context(), input)
		render(r.Context(), w, http.StatusBadRequest, templates.ProposalEditorPage(user, input, proposals.SavedDraft{}, "", message))
		return
	}
	if err := a.persistProposalContractAssignment(r.Context(), draft, input); err != nil {
		a.logger.Error("persist proposal contract assignment failed", "proposal_id", draft.ProposalID, "version_id", draft.VersionID, "error", err)
		message := "O rascunho foi salvo, mas não foi possível vincular o modelo de contrato."
		if a.cfg.Environment != "production" {
			message += " Detalhe: " + err.Error()
		}
		input = a.decorateProposalContractOptions(r.Context(), input)
		render(r.Context(), w, http.StatusBadRequest, templates.ProposalEditorPage(user, input, draft, "Rascunho salvo.", message))
		return
	}

	if action == "publish" {
		if err := validateProposalForPublish(input); err != nil {
			input = a.decorateProposalContractOptions(r.Context(), input)
			render(r.Context(), w, http.StatusBadRequest, templates.ProposalEditorPage(user, input, draft, "Rascunho salvo antes da validação de publicação.", err.Error()))
			return
		}
		if _, err := a.proposalStore.Publish(r.Context(), user.ID, allowAll, draft.VersionID); err != nil {
			a.logger.Error("publish proposal failed", "user_id", user.ID, "proposal_id", draft.ProposalID, "version_id", draft.VersionID, "error", err)
			message := "Não foi possível publicar a proposta."
			if a.cfg.Environment != "production" {
				message += " Detalhe: " + err.Error()
			}
			input = a.decorateProposalContractOptions(r.Context(), input)
			render(r.Context(), w, http.StatusBadRequest, templates.ProposalEditorPage(user, input, draft, "", message))
			return
		}
		http.Redirect(w, r, "/admin/proposals/"+draft.ProposalID+"/edit?published=1", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/admin/proposals/"+draft.ProposalID+"/edit?saved=1", http.StatusSeeOther)
}

func validateProposalForPublish(input proposals.EditorInput) error {
	if strings.TrimSpace(input.ClientLegalName) == "" && strings.TrimSpace(input.ClientTradeName) == "" {
		return fmt.Errorf("Informe a razão social ou o nome fantasia do cliente antes de publicar.")
	}
	if len(input.Items) == 0 {
		return fmt.Errorf("Informe o valor de ao menos um produto antes de publicar.")
	}
	if proposalContentString(input.Content, "proposal", "contract_template_version_id") == "" {
		return fmt.Errorf("Selecione o modelo de contrato desta proposta antes de publicar.")
	}
	return nil
}

func (a *App) proposalInputFromForm(r *http.Request, salesperson domain.User) (proposals.EditorInput, error) {
	contractTemplateID := strings.TrimSpace(r.FormValue("contract_template_id"))
	input := proposals.EditorInput{
		ProposalID:         strings.TrimSpace(r.FormValue("proposal_id")),
		ClientLegalName:    strings.TrimSpace(r.FormValue("client_legal_name")),
		ClientTradeName:    strings.TrimSpace(r.FormValue("client_trade_name")),
		ClientCNPJ:         strings.TrimSpace(r.FormValue("client_cnpj")),
		ClientEmail:        strings.TrimSpace(r.FormValue("client_email")),
		ClientPhone:        strings.TrimSpace(r.FormValue("client_phone")),
		ClientLogoURL:      strings.TrimSpace(r.FormValue("client_logo_url")),
		ClientStreet:       strings.TrimSpace(r.FormValue("client_street")),
		ClientStreetNumber: strings.TrimSpace(r.FormValue("client_street_number")),
		ClientComplement:   strings.TrimSpace(r.FormValue("client_complement")),
		ClientDistrict:     strings.TrimSpace(r.FormValue("client_district")),
		ClientCity:         strings.TrimSpace(r.FormValue("client_city")),
		ClientState:        strings.ToUpper(strings.TrimSpace(r.FormValue("client_state"))),
		ClientPostalCode:   strings.TrimSpace(r.FormValue("client_postal_code")),
		ContactName:        strings.TrimSpace(r.FormValue("contact_name")),
		ContactRole:        strings.TrimSpace(r.FormValue("contact_role")),
		ContactEmail:       strings.TrimSpace(r.FormValue("contact_email")),
		ContactPhone:       strings.TrimSpace(r.FormValue("contact_phone")),
		Title:              strings.TrimSpace(r.FormValue("title")),
		OperationContext:   strings.TrimSpace(r.FormValue("operation_context")),
		CustomerPriorities: multilineValues(r.FormValue("customer_priorities")),
		SolutionTitle:      strings.TrimSpace(r.FormValue("solution_title")),
		SolutionScope:      multilineValues(r.FormValue("solution_scope")),
		PricingModel:       strings.TrimSpace(r.FormValue("pricing_model")),
		Content: map[string]any{
			"proposal": map[string]any{"contract_template_id": contractTemplateID},
		},
	}

	if input.Title == "" {
		input.Title = "Proposta Comercial ViaGate"
	}
	if input.PricingModel == "" {
		input.PricingModel = "per_item"
	}

	if input.ClientCNPJ != "" {
		cnpj, err := cleanCNPJ(input.ClientCNPJ)
		if err != nil {
			return input, err
		}
		input.ClientCNPJ = cnpj
	}
	if input.ClientEmail != "" {
		clientEmail, err := cleanEmail(input.ClientEmail, false)
		if err != nil {
			return input, fmt.Errorf("E-mail do cliente inválido.")
		}
		input.ClientEmail = clientEmail
	}
	if input.ClientPhone != "" {
		clientPhone, err := cleanPhone(input.ClientPhone, false)
		if err != nil {
			return input, fmt.Errorf("Telefone do cliente inválido.")
		}
		input.ClientPhone = clientPhone
	}
	if input.ClientPostalCode != "" {
		postalCode, err := cleanPostalCode(input.ClientPostalCode, false)
		if err != nil {
			return input, fmt.Errorf("CEP do cliente inválido.")
		}
		input.ClientPostalCode = postalCode
	}
	if input.ClientState != "" && len(input.ClientState) != 2 {
		return input, fmt.Errorf("UF do cliente inválida.")
	}
	if input.ClientLogoURL != "" {
		logoURL, err := cleanCommercialImageURL(input.ClientLogoURL)
		if err != nil {
			return input, fmt.Errorf("Logo do cliente inválido. Envie a imagem novamente.")
		}
		input.ClientLogoURL = logoURL
	}
	if input.ContactEmail != "" {
		contactEmail, err := cleanEmail(input.ContactEmail, false)
		if err != nil {
			return input, fmt.Errorf("E-mail do contato inválido.")
		}
		input.ContactEmail = contactEmail
	}
	if input.ContactPhone != "" {
		contactPhone, err := cleanPhone(input.ContactPhone, false)
		if err != nil {
			return input, fmt.Errorf("Telefone do contato inválido.")
		}
		input.ContactPhone = contactPhone
	}
	if value := strings.TrimSpace(r.FormValue("valid_until")); value != "" {
		date, err := time.Parse("2006-01-02", value)
		if err != nil {
			return input, fmt.Errorf("Validade inválida.")
		}
		input.ValidUntil = &date
	}
	if !validPricingModel(input.PricingModel) {
		return input, fmt.Errorf("Modelo comercial inválido.")
	}

	contractTemplateVersionID, err := a.resolveProposalContractTemplateVersion(r.Context(), contractTemplateID)
	if err != nil {
		return input, err
	}

	minimumInvoice, err := parseMoney(r.FormValue("minimum_invoice"))
	if err != nil {
		return input, fmt.Errorf("Fatura mínima inválida.")
	}
	input.MinimumInvoice = minimumInvoice
	setupFee, err := parseMoney(r.FormValue("setup_fee"))
	if err != nil {
		return input, fmt.Errorf("Valor de implantação inválido.")
	}
	input.SetupFee = setupFee

	ids := r.Form["catalog_id"]
	statuses := r.Form["item_status"]
	prices := r.Form["item_price"]
	for index, id := range ids {
		status := "off"
		if index < len(statuses) {
			status = strings.TrimSpace(statuses[index])
		}
		if status != "off" && status != "included" && status != "optional" {
			return input, fmt.Errorf("Status de item inválido.")
		}
		priceValue := ""
		if index < len(prices) {
			priceValue = strings.TrimSpace(prices[index])
		}
		if status == "off" || priceValue == "" {
			continue
		}
		group, item, ok := catalog.ItemByID(id)
		if !ok {
			return input, fmt.Errorf("Item comercial inválido: %s", id)
		}
		if !catalog.ModelAllows(item, input.PricingModel) {
			continue
		}
		price, err := parseMoney(priceValue)
		if err != nil {
			return input, fmt.Errorf("Valor inválido para %s.", item.Label)
		}
		input.Items = append(input.Items, proposals.EditorItem{CatalogID: item.ID, GroupName: group.Title, Label: item.Label, Unit: item.Unit, Price: price, IsOptional: status == "optional", SortOrder: index})
	}
	input.Conditions = normalizedConditions(r.Form["condition"], r.FormValue("custom_conditions"))
	validUntil := ""
	if input.ValidUntil != nil {
		validUntil = input.ValidUntil.Format("2006-01-02")
	}
	input.Content = map[string]any{
		"proposal": map[string]any{
			"title": input.Title,
			"valid_until": validUntil,
			"contract_template_id": contractTemplateID,
			"contract_template_version_id": contractTemplateVersionID,
		},
		"client": map[string]any{
			"legal_name": input.ClientLegalName, "trade_name": input.ClientTradeName, "company_name": input.ClientLegalName,
			"cnpj": input.ClientCNPJ, "email": input.ClientEmail, "phone": input.ClientPhone, "logo_url": input.ClientLogoURL,
			"street": input.ClientStreet, "street_number": input.ClientStreetNumber, "complement": input.ClientComplement,
			"district": input.ClientDistrict, "city": input.ClientCity, "state": input.ClientState, "postal_code": input.ClientPostalCode,
		},
		"contact":             map[string]any{"name": input.ContactName, "role": input.ContactRole, "email": input.ContactEmail, "phone": input.ContactPhone},
		"operation_context":   input.OperationContext,
		"customer_priorities": input.CustomerPriorities,
		"solution_title":      input.SolutionTitle,
		"solution_scope":      input.SolutionScope,
		"salesperson": map[string]any{
			"name": salesperson.Name, "email": salesperson.Email, "phone": salesperson.Phone,
			"job_title": salesperson.JobTitle, "role": salesperson.JobTitle, "photo_url": salesperson.PhotoURL,
			"linkedin": salesperson.LinkedInURL, "instagram": salesperson.InstagramURL,
		},
	}
	canonical := struct {
		Content        map[string]any         `json:"content"`
		PricingModel   string                 `json:"pricing_model"`
		MinimumInvoice float64                `json:"minimum_invoice"`
		SetupFee       float64                `json:"setup_fee"`
		Conditions     []string               `json:"conditions"`
		Items          []proposals.EditorItem `json:"items"`
	}{input.Content, input.PricingModel, input.MinimumInvoice, input.SetupFee, input.Conditions, input.Items}
	encoded, _ := json.Marshal(canonical)
	hash := sha256.Sum256(encoded)
	input.ContentHash = hash[:]
	return input, nil
}

func proposalContentString(content map[string]any, section, key string) string {
	group, _ := content[section].(map[string]any)
	value, _ := group[key].(string)
	return strings.TrimSpace(value)
}

func validPricingModel(value string) bool {
	for _, model := range catalog.PricingModels {
		if model.ID == value {
			return true
		}
	}
	return false
}

func parseMoney(value string) (float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	if strings.Contains(value, "-") {
		return 0, fmt.Errorf("invalid money")
	}
	value = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(value, "R$", ""), "\u00a0", ""))
	value = strings.ReplaceAll(value, " ", "")
	if strings.Contains(value, ",") {
		value = strings.ReplaceAll(value, ".", "")
		value = strings.ReplaceAll(value, ",", ".")
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) || number < 0 {
		return 0, fmt.Errorf("invalid money")
	}
	return number, nil
}

func normalizedConditions(standard []string, custom string) []string {
	seen := map[string]bool{}
	result := []string{}
	appendValue := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		result = append(result, value)
	}
	for _, value := range standard {
		appendValue(value)
	}
	for _, line := range strings.Split(custom, "\n") {
		appendValue(line)
	}
	return result
}

func multilineValues(value string) []string {
	result := []string{}
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}
