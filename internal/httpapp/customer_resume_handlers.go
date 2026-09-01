package httpapp

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/security"
)

func (a *App) resumeOnboarding(w http.ResponseWriter, r *http.Request) {
	plain := chi.URLParam(r, "token")
	tokenHash := hashToken(plain)

	if acceptanceID, err := a.proposalStore.CustomerJourneyAcceptance(r.Context(), tokenHash); err == nil {
		a.resumeContractingJourney(w, r, acceptanceID)
		return
	}

	acceptanceID, err := a.proposalStore.ConsumeCustomerResumeToken(r.Context(), tokenHash)
	if err != nil {
		http.Error(w, "Link de retomada inválido, expirado ou já utilizado.", http.StatusGone)
		return
	}
	onboarding, err := a.onboardingStore.ByAcceptance(r.Context(), acceptanceID)
	if err != nil {
		http.Error(w, "Cadastro não encontrado.", http.StatusNotFound)
		return
	}
	if onboarding.Status != "correction_requested" {
		http.Error(w, "Este cadastro não possui correção pendente.", http.StatusGone)
		return
	}
	if err := a.startCustomerSession(w, r, acceptanceID); err != nil {
		http.Error(w, "não foi possível iniciar a sessão", http.StatusInternalServerError)
		return
	}
	_, _ = a.pool.Exec(r.Context(), `
		insert into audit_events(actor_type,event_type,resource_type,resource_id,ip_address,user_agent,metadata)
		values('customer','onboarding.resumed','onboarding',$1,$2,$3,jsonb_build_object('one_time_link',true))
	`, onboarding.ID, requestIP(r), r.UserAgent())
	http.Redirect(w, r, "/onboarding/"+onboarding.ID, http.StatusSeeOther)
}

func (a *App) resumeContractingJourney(w http.ResponseWriter, r *http.Request, acceptanceID string) {
	onboarding, err := a.onboardingStore.ByAcceptance(r.Context(), acceptanceID)
	if err != nil {
		http.Error(w, "Contratação não encontrada.", http.StatusNotFound)
		return
	}
	if err := a.startCustomerSession(w, r, acceptanceID); err != nil {
		http.Error(w, "não foi possível iniciar a sessão", http.StatusInternalServerError)
		return
	}
	_, _ = a.pool.Exec(r.Context(), `
		insert into audit_events(actor_type,event_type,resource_type,resource_id,ip_address,user_agent,metadata)
		values('customer','customer_journey.resumed','onboarding',$1,$2,$3,jsonb_build_object('reusable_link',true))
	`, onboarding.ID, requestIP(r), r.UserAgent())

	if delivery, deliveryErr := a.contractStore.DeliveryByOnboarding(r.Context(), onboarding.ID); deliveryErr == nil {
		target := "/sign/" + delivery.SignerToken
		if delivery.ContractStatus == "signed" && delivery.SignerStatus == "signed" {
			target += "?signed=1"
		}
		http.Redirect(w, r, target, http.StatusSeeOther)
		return
	} else if deliveryErr != pgx.ErrNoRows {
		a.logger.Warn("resolve journey contract failed", "onboarding_id", onboarding.ID, "error", deliveryErr)
	}

	http.Redirect(w, r, "/onboarding/"+onboarding.ID, http.StatusSeeOther)
}

func (a *App) startCustomerSession(w http.ResponseWriter, r *http.Request, acceptanceID string) error {
	sessionPlain, sessionHash, err := security.RandomToken(32)
	if err != nil {
		return err
	}
	expires := time.Now().Add(7 * 24 * time.Hour)
	if err := a.proposalStore.CreateCustomerSession(r.Context(), acceptanceID, sessionHash, requestIP(r), r.UserAgent(), expires); err != nil {
		return err
	}
	setSecureCookie(w, customerSessionCookie, sessionPlain, expires, a.cfg.Environment == "production")
	return nil
}
