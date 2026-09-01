package httpapp

import (
	"context"
	"net/http"

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
