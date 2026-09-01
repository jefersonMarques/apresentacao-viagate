package httpapp

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/security"
	"github.com/jefersonMarques/apresentacao-viagate/internal/proposals"
)

type proposalJourney struct {
	State string `json:"state"`
	Label string `json:"label"`
	URL   string `json:"url"`
	Tone  string `json:"tone"`
}

func defaultProposalJourney() proposalJourney {
	return proposalJourney{State: "proposal", Label: "ACEITAR PROPOSTA", Tone: "primary"}
}

func (a *App) proposalJourneyForRequest(ctx context.Context, r *http.Request, proposal proposals.PublicProposal) proposalJourney {
	journey := defaultProposalJourney()
	cookie, err := r.Cookie(customerSessionCookie)
	if err != nil || cookie.Value == "" {
		return journey
	}
	acceptanceID, err := a.proposalStore.CustomerSessionAcceptance(ctx, hashToken(cookie.Value))
	if err != nil {
		return journey
	}

	var proposalID, onboardingID, onboardingStatus string
	if err := a.pool.QueryRow(ctx, `
		select pa.proposal_id::text,o.id::text,o.status::text
		from proposal_acceptances pa
		join onboardings o on o.proposal_acceptance_id=pa.id
		where pa.id=$1
	`, acceptanceID).Scan(&proposalID, &onboardingID, &onboardingStatus); err != nil || proposalID != proposal.ProposalID {
		return journey
	}

	if delivery, deliveryErr := a.contractStore.DeliveryByOnboarding(ctx, onboardingID); deliveryErr == nil {
		if delivery.ContractStatus == "signed" && delivery.SignerStatus == "signed" {
			return proposalJourney{State: "signed", Label: "VER CONTRATO ASSINADO", URL: "/sign/" + delivery.SignerToken + "?signed=1", Tone: "success"}
		}
		return proposalJourney{State: "signature", Label: "REVISAR E ASSINAR CONTRATO", URL: "/sign/" + delivery.SignerToken, Tone: "success"}
	}

	switch onboardingStatus {
	case "submitted", "under_review":
		return proposalJourney{State: "review", Label: "DADOS ENVIADOS", URL: "/onboarding/" + onboardingID, Tone: "success"}
	case "approved":
		return proposalJourney{State: "preparing_contract", Label: "CONTRATO EM PREPARAÇÃO", URL: "/onboarding/" + onboardingID, Tone: "success"}
	default:
		return proposalJourney{State: "contracting", Label: "CONTINUAR CONTRATAÇÃO", URL: "/onboarding/" + onboardingID, Tone: "success"}
	}
}

func (a *App) resumeCustomerJourney(w http.ResponseWriter, r *http.Request) {
	acceptanceID, err := a.proposalStore.CustomerJourneyAcceptance(r.Context(), hashToken(chi.URLParam(r, "token")))
	if err != nil {
		http.Error(w, "Este link para continuar a contratação é inválido ou expirou.", http.StatusGone)
		return
	}
	onboarding, err := a.onboardingStore.ByAcceptance(r.Context(), acceptanceID)
	if err != nil {
		http.Error(w, "Contratação não encontrada.", http.StatusNotFound)
		return
	}

	sessionPlain, sessionHash, err := security.RandomToken(32)
	if err != nil {
		http.Error(w, "não foi possível iniciar a sessão", http.StatusInternalServerError)
		return
	}
	expires := time.Now().Add(7 * 24 * time.Hour)
	if err := a.proposalStore.CreateCustomerSession(r.Context(), acceptanceID, sessionHash, requestIP(r), r.UserAgent(), expires); err != nil {
		http.Error(w, "não foi possível iniciar a sessão", http.StatusInternalServerError)
		return
	}
	setSecureCookie(w, customerSessionCookie, sessionPlain, expires, a.cfg.Environment == "production")

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
