package httpapp

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5"
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
	if cookie, err := r.Cookie(customerSessionCookie); err == nil && cookie.Value != "" {
		if acceptanceID, sessionErr := a.proposalStore.CustomerSessionAcceptance(ctx, hashToken(cookie.Value)); sessionErr == nil {
			if journey, ok := a.proposalJourneyForAcceptance(ctx, acceptanceID, proposal); ok {
				return journey
			}
		}
	}

	acceptanceID, err := a.proposalAcceptanceForVersion(ctx, proposal.VersionID)
	if err != nil {
		return defaultProposalJourney()
	}
	journey, ok := a.proposalJourneyForAcceptance(ctx, acceptanceID, proposal)
	if !ok {
		return defaultProposalJourney()
	}
	return journey
}

func (a *App) proposalAcceptanceForVersion(ctx context.Context, proposalVersionID string) (string, error) {
	var acceptanceID string
	err := a.pool.QueryRow(ctx, `
		select pa.id::text
		from proposal_acceptances pa
		where pa.proposal_version_id=$1
		order by pa.accepted_at desc
		limit 1
	`, proposalVersionID).Scan(&acceptanceID)
	return acceptanceID, err
}

func (a *App) proposalJourneyForAcceptance(ctx context.Context, acceptanceID string, proposal proposals.PublicProposal) (proposalJourney, bool) {
	var proposalID, onboardingID, onboardingStatus string
	if err := a.pool.QueryRow(ctx, `
		select pa.proposal_id::text,o.id::text,o.status::text
		from proposal_acceptances pa
		join onboardings o on o.proposal_acceptance_id=pa.id
		where pa.id=$1
	`, acceptanceID).Scan(&proposalID, &onboardingID, &onboardingStatus); err != nil || proposalID != proposal.ProposalID {
		return proposalJourney{}, false
	}

	if delivery, deliveryErr := a.contractStore.DeliveryByOnboarding(ctx, onboardingID); deliveryErr == nil {
		if delivery.ContractStatus == "signed" && delivery.SignerStatus == "signed" {
			return proposalJourney{State: "signed", Label: "CONTINUAR PARA ATIVAÇÃO", URL: "/sign/" + delivery.SignerToken + "?signed=1", Tone: "success"}, true
		}
		return proposalJourney{State: "signature", Label: "REVISAR E ASSINAR CONTRATO", URL: "/sign/" + delivery.SignerToken, Tone: "success"}, true
	} else if deliveryErr != pgx.ErrNoRows {
		return proposalJourney{}, false
	}

	switch onboardingStatus {
	case "submitted", "under_review":
		return proposalJourney{State: "review", Label: "ACOMPANHAR CONTRATAÇÃO", URL: "/onboarding/" + onboardingID, Tone: "success"}, true
	case "approved":
		return proposalJourney{State: "preparing_contract", Label: "CONTRATO EM PREPARAÇÃO", URL: "/onboarding/" + onboardingID, Tone: "success"}, true
	default:
		return proposalJourney{State: "contracting", Label: "CONTINUAR CONTRATAÇÃO", URL: "/onboarding/" + onboardingID, Tone: "success"}, true
	}
}

func (a *App) proposalPublicPathByOnboarding(ctx context.Context, onboardingID string) (string, error) {
	var token string
	err := a.pool.QueryRow(ctx, `
		select pv.public_token::text
		from onboardings o
		join proposal_acceptances pa on pa.id=o.proposal_acceptance_id
		join proposal_versions pv on pv.id=pa.proposal_version_id
		where o.id=$1
	`, onboardingID).Scan(&token)
	if err != nil {
		return "", err
	}
	return "/p/" + token, nil
}

func (a *App) proposalPublicPathByContract(ctx context.Context, contractID string) (string, error) {
	var onboardingID string
	if err := a.pool.QueryRow(ctx, `select onboarding_id::text from contracts where id=$1`, contractID).Scan(&onboardingID); err != nil {
		return "", err
	}
	return a.proposalPublicPathByOnboarding(ctx, onboardingID)
}
