package httpapp

import (
	"net/http"
	"strings"
	"time"

	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

func (a *App) saveOnboardingCompany(w http.ResponseWriter, r *http.Request) {
	current, err := a.currentOnboarding(r)
	if err != nil {
		http.Error(w, "acesso negado", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "dados inválidos", http.StatusBadRequest)
		return
	}
	cnpj, err := cleanCNPJ(r.FormValue("cnpj"))
	if err != nil {
		a.renderContractingError(w, r, current, err.Error())
		return
	}
	postalCode, err := cleanPostalCode(r.FormValue("postal_code"), true)
	if err != nil {
		a.renderContractingError(w, r, current, err.Error())
		return
	}

	current.CNPJ = cnpj
	current.LegalName = strings.TrimSpace(r.FormValue("legal_name"))
	current.TradeName = strings.TrimSpace(r.FormValue("trade_name"))
	current.Street = strings.TrimSpace(r.FormValue("street"))
	current.StreetNumber = strings.TrimSpace(r.FormValue("street_number"))
	current.Complement = strings.TrimSpace(r.FormValue("complement"))
	current.District = strings.TrimSpace(r.FormValue("district"))
	current.City = strings.TrimSpace(r.FormValue("city"))
	current.State = strings.ToUpper(strings.TrimSpace(r.FormValue("state")))
	current.PostalCode = postalCode
	if current.LegalName == "" || current.Street == "" || current.StreetNumber == "" || current.City == "" || len(current.State) != 2 {
		a.renderContractingError(w, r, current, "Complete razão social, endereço, cidade e UF para continuar.")
		return
	}
	if err := a.onboardingStore.Save(r.Context(), current); err != nil {
		a.logger.Error("save contracting company failed", "onboarding_id", current.ID, "error", err)
		a.renderContractingError(w, r, current, "Não foi possível salvar os dados da empresa. Tente novamente.")
		return
	}
	http.Redirect(w, r, "/onboarding/"+current.ID+"?saved=company", http.StatusSeeOther)
}

func (a *App) saveOnboardingInsurance(w http.ResponseWriter, r *http.Request) {
	current, err := a.currentOnboarding(r)
	if err != nil {
		http.Error(w, "acesso negado", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "dados inválidos", http.StatusBadRequest)
		return
	}
	operationType := strings.ToLower(strings.TrimSpace(r.FormValue("operation_type")))
	if operationType != "normal" && operationType != "avulsa" {
		a.renderContractingError(w, r, current, "Selecione um tipo de operação válido.")
		return
	}
	insurer := strings.TrimSpace(r.FormValue("insurer"))
	if insurer == "" {
		a.renderContractingError(w, r, current, "Informe a seguradora.")
		return
	}
	startValue := strings.TrimSpace(r.FormValue("policy_start_date"))
	endValue := strings.TrimSpace(r.FormValue("policy_end_date"))
	start, startErr := time.Parse("2006-01-02", startValue)
	end, endErr := time.Parse("2006-01-02", endValue)
	if startErr != nil || endErr != nil || end.Before(start) {
		a.renderContractingError(w, r, current, "Informe uma vigência de apólice válida.")
		return
	}

	current.OperationType = operationType
	current.Insurer = insurer
	current.PolicyStartDate = startValue
	current.PolicyEndDate = endValue
	current.BrokerCompany = strings.TrimSpace(r.FormValue("broker_company"))
	current.BrokerProducer = strings.TrimSpace(r.FormValue("broker_producer"))
	if err := a.onboardingStore.Save(r.Context(), current); err != nil {
		a.logger.Error("save contracting insurance failed", "onboarding_id", current.ID, "error", err)
		a.renderContractingError(w, r, current, "Não foi possível salvar os dados do seguro. Tente novamente.")
		return
	}
	http.Redirect(w, r, "/onboarding/"+current.ID+"?saved=insurance", http.StatusSeeOther)
}

func (a *App) renderContractingError(w http.ResponseWriter, r *http.Request, current interface{ GetID() string }, message string) {
	// Kept only as a compile-time guard against accidental generic use.
	http.Error(w, message, http.StatusBadRequest)
}

func (a *App) renderContractingPageError(w http.ResponseWriter, r *http.Request, currentOnboardingID string, message string) {
	current, err := a.currentOnboarding(r)
	if err != nil || current.ID != currentOnboardingID {
		http.Error(w, message, http.StatusBadRequest)
		return
	}
	hasPolicy, _ := a.onboardingStore.HasPolicy(r.Context(), current.ID)
	render(r.Context(), w, http.StatusBadRequest, templates.ContractingJourneyPage(current, hasPolicy, "", message, a.cfg.RequireOnboardingReview))
}
