package httpapp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/security"
	"github.com/jefersonMarques/apresentacao-viagate/internal/proposals"
	"github.com/jefersonMarques/apresentacao-viagate/web/templates"
)

const customerAcceptanceContextKey contextKey = "customer_acceptance_id"

func (a *App) publicProposalPage(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	proposal, err := a.proposalStore.PublicByToken(r.Context(), token)
	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "Proposta não encontrada.", http.StatusNotFound)
			return
		}
		http.Error(w, "Esta proposta não está mais disponível.", http.StatusGone)
		return
	}
	if wantsJSON(r) {
		a.writeProposalContractData(w, r, proposal)
		return
	}

	// O link público da proposta é a única porta de entrada da jornada do
	// cliente. Depois do aceite, abrir o mesmo link retoma automaticamente o
	// ponto correto. ?view=proposal é a saída explícita para consultar a
	// proposta original sem avançar a jornada.
	if proposal.Status == "accepted" && r.URL.Query().Get("view") != "proposal" {
		if acceptanceID, acceptanceErr := a.proposalAcceptanceForVersion(r.Context(), proposal.VersionID); acceptanceErr == nil {
			if journey, ok := a.proposalJourneyForAcceptance(r.Context(), acceptanceID, proposal); ok && journey.URL != "" {
				if sessionErr := a.startCustomerSession(w, r, acceptanceID); sessionErr != nil {
					a.logger.Error("start customer journey session from proposal failed", "proposal_id", proposal.ProposalID, "error", sessionErr)
					http.Error(w, "Não foi possível retomar a contratação.", http.StatusInternalServerError)
					return
				}
				http.Redirect(w, r, journey.URL, http.StatusSeeOther)
				return
			}
		}
	}

	sessionID, _ := newUUID()
	_, _ = a.pool.Exec(r.Context(), `
		insert into document_events(document_kind,document_version_id,event_type,viewer_session,ip_address,user_agent)
		values('proposal',$1,'open',$2,$3,$4)
	`, proposal.VersionID, nullableUUID(sessionID), requestIP(r), r.UserAgent())
	a.publishProposalEvent(r.Context(), proposal.ProposalID, "proposal.opened", "Proposta aberta", "", dailyEventDedupe(proposal.VersionID))
	render(r.Context(), w, http.StatusOK, templates.PublicProposalViewerPage(proposal, ""))
}

func (a *App) acceptProposal(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	proposal, err := a.proposalStore.PublicByToken(r.Context(), token)
	if err != nil {
		http.Error(w, "Proposta não disponível.", http.StatusGone)
		return
	}
	if strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "multipart/form-data") {
		// Compatibility for proposal links already open during rollout. New viewers
		// use the short acceptance flow below and collect contracting data afterward.
		a.acceptProposalContractFlow(w, r, proposal)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "dados inválidos", http.StatusBadRequest)
		return
	}
	cpf, err := cleanCPF(r.FormValue("cpf"))
	if err != nil {
		render(r.Context(), w, http.StatusBadRequest, templates.PublicProposalViewerPage(proposal, err.Error()))
		return
	}
	emailAddress, err := cleanEmail(r.FormValue("email"), true)
	if err != nil {
		render(r.Context(), w, http.StatusBadRequest, templates.PublicProposalViewerPage(proposal, "Informe um e-mail válido para o responsável."))
		return
	}
	phone, err := cleanPhone(r.FormValue("phone"), true)
	if err != nil {
		render(r.Context(), w, http.StatusBadRequest, templates.PublicProposalViewerPage(proposal, "Informe um telefone válido para o responsável."))
		return
	}
	if r.FormValue("authority") != "1" {
		render(r.Context(), w, http.StatusBadRequest, templates.PublicProposalViewerPage(proposal, "É necessário confirmar a autorização para representar a empresa."))
		return
	}
	sessionID, err := newUUID()
	if err != nil {
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}
	input := proposals.AcceptanceInput{
		Name: strings.TrimSpace(r.FormValue("name")), Email: emailAddress, CPF: cpf, Phone: phone,
		Role: strings.TrimSpace(r.FormValue("role")), AuthorityDeclared: true,
		IPAddress: requestIP(r), UserAgent: r.UserAgent(), SessionID: sessionID,
	}
	if input.Name == "" {
		render(r.Context(), w, http.StatusBadRequest, templates.PublicProposalViewerPage(proposal, "Preencha todos os dados do responsável."))
		return
	}
	result, err := a.proposalStore.Accept(r.Context(), proposal, input)
	if err != nil {
		a.logger.Error("proposal acceptance failed", "error", err)
		render(r.Context(), w, http.StatusBadRequest, templates.PublicProposalViewerPage(proposal, "Não foi possível registrar o aceite. Se esta proposta já foi aceita, abra novamente o mesmo link da proposta para continuar."))
		return
	}

	plain, hash, err := security.RandomToken(32)
	if err != nil {
		http.Error(w, "erro interno", http.StatusInternalServerError)
		return
	}
	expires := time.Now().Add(7 * 24 * time.Hour)
	if err := a.proposalStore.CreateCustomerSession(r.Context(), result.AcceptanceID, hash, requestIP(r), r.UserAgent(), expires); err != nil {
		http.Error(w, "não foi possível iniciar o cadastro", http.StatusInternalServerError)
		return
	}
	setSecureCookie(w, customerSessionCookie, plain, expires, a.cfg.Environment == "production")

	if err := a.queueCustomerJourneyEmail(r.Context(), result.AcceptanceID, input.Name, input.Email); err != nil {
		a.logger.Error("queue customer contracting journey email failed", "acceptance_id", result.AcceptanceID, "error", err)
	}
	_, _ = a.pool.Exec(r.Context(), `
		insert into notification_outbox(recipient,recipient_name,subject,html_body,text_body)
		select u.email,u.name,'Proposta aceita: '||p.title,
		       '<p>A proposta de <strong>'||coalesce(nullif(c.trade_name,''),nullif(c.legal_name,''),'cliente')||'</strong> foi aceita por '||$2||'.</p>',
		       'A proposta de '||coalesce(nullif(c.trade_name,''),nullif(c.legal_name,''),'cliente')||' foi aceita por '||$2||'.'
		from proposals p join clients c on c.id=p.client_id join users u on u.id=p.created_by where p.id=$1
	`, proposal.ProposalID, input.Name)
	a.publishProposalEvent(r.Context(), proposal.ProposalID, "proposal.accepted", "Proposta aceita", "Aceita por "+input.Name, proposal.VersionID)

	// Retorna ao mesmo link que o cliente recebeu originalmente. O GET decide
	// a etapa correta no servidor e cria uma nova sessão quando necessário.
	nextURL := "/p/" + proposal.PublicToken
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(map[string]string{"next_url": nextURL})
		return
	}
	http.Redirect(w, r, nextURL, http.StatusSeeOther)
}

func (a *App) customerSessionRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(customerSessionCookie)
		if err != nil || cookie.Value == "" {
			http.Error(w, "Sessão do cliente expirada. Abra novamente o link da proposta.", http.StatusUnauthorized)
			return
		}
		acceptanceID, err := a.proposalStore.CustomerSessionAcceptance(r.Context(), hashToken(cookie.Value))
		if err != nil {
			clearCookie(w, customerSessionCookie)
			http.Error(w, "Sessão do cliente expirada. Abra novamente o link da proposta.", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), customerAcceptanceContextKey, acceptanceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *App) currentOnboarding(r *http.Request) (domain.Onboarding, error) {
	acceptanceID, ok := r.Context().Value(customerAcceptanceContextKey).(string)
	if !ok || acceptanceID == "" {
		return domain.Onboarding{}, http.ErrNoCookie
	}
	onboarding, err := a.onboardingStore.ByAcceptance(r.Context(), acceptanceID)
	if err != nil {
		return domain.Onboarding{}, err
	}
	if pathID := chi.URLParam(r, "id"); pathID != "" && pathID != onboarding.ID {
		return domain.Onboarding{}, pgx.ErrNoRows
	}
	return onboarding, nil
}
