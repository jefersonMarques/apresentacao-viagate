package httpapp

import (
	"context"
	"fmt"
	"time"

	"github.com/jefersonMarques/apresentacao-viagate/internal/notifications"
)

func (a *App) publishProposalEvent(ctx context.Context, proposalID, eventType, title, body, dedupeSuffix string) {
	var ownerID, proposalTitle, clientName string
	if err := a.pool.QueryRow(ctx, `
		select p.created_by::text,p.title,coalesce(nullif(c.trade_name,''),c.legal_name)
		from proposals p join clients c on c.id=p.client_id where p.id=$1
	`, proposalID).Scan(&ownerID, &proposalTitle, &clientName); err != nil {
		a.logger.Warn("load proposal event owner failed", "proposal_id", proposalID, "event_type", eventType, "error", err)
		return
	}
	if title == "" {
		title = proposalTitle
	}
	if body == "" {
		body = clientName
	}
	if dedupeSuffix == "" {
		dedupeSuffix = proposalID
	}
	store := notifications.NewInAppStore(a.pool)
	if err := store.Publish(ctx, notifications.Event{
		OwnerUserID: ownerID,
		EventType: eventType,
		Title: title,
		Body: body,
		ResourceType: "proposal",
		ResourceID: proposalID,
		TargetURL: "/admin/pipeline/" + proposalID,
		DedupeKey: eventType + ":" + dedupeSuffix,
	}); err != nil {
		a.logger.Warn("publish proposal notification failed", "proposal_id", proposalID, "event_type", eventType, "error", err)
	}
}

func (a *App) publishPresentationEvent(ctx context.Context, presentationID, eventType, dedupeSuffix string) {
	var ownerID, title, clientName string
	if err := a.pool.QueryRow(ctx, `
		select p.created_by::text,p.title,coalesce(nullif(c.trade_name,''),nullif(c.legal_name,''),'Cliente não identificado')
		from presentations p left join clients c on c.id=p.client_id where p.id=$1
	`, presentationID).Scan(&ownerID, &title, &clientName); err != nil {
		a.logger.Warn("load presentation event owner failed", "presentation_id", presentationID, "event_type", eventType, "error", err)
		return
	}
	if dedupeSuffix == "" {
		dedupeSuffix = presentationID
	}
	store := notifications.NewInAppStore(a.pool)
	if err := store.Publish(ctx, notifications.Event{
		OwnerUserID: ownerID,
		EventType: eventType,
		Title: "Apresentação aberta",
		Body: clientName + " · " + title,
		ResourceType: "presentation",
		ResourceID: presentationID,
		TargetURL: "/admin/presentations/" + presentationID + "/edit",
		DedupeKey: eventType + ":" + dedupeSuffix,
	}); err != nil {
		a.logger.Warn("publish presentation notification failed", "presentation_id", presentationID, "event_type", eventType, "error", err)
	}
}

func (a *App) publishContractEvent(ctx context.Context, contractID, eventType, title, dedupeSuffix string) {
	var ownerID, proposalID, clientName string
	if err := a.pool.QueryRow(ctx, `
		select p.created_by::text,p.id::text,coalesce(nullif(cl.trade_name,''),cl.legal_name)
		from contracts c
		join onboardings o on o.id=c.onboarding_id
		join proposal_acceptances pa on pa.id=o.proposal_acceptance_id
		join proposals p on p.id=pa.proposal_id
		join clients cl on cl.id=o.client_id
		where c.id=$1
	`, contractID).Scan(&ownerID, &proposalID, &clientName); err != nil {
		a.logger.Warn("load contract event owner failed", "contract_id", contractID, "event_type", eventType, "error", err)
		return
	}
	if dedupeSuffix == "" {
		dedupeSuffix = contractID
	}
	store := notifications.NewInAppStore(a.pool)
	if err := store.Publish(ctx, notifications.Event{
		OwnerUserID: ownerID,
		EventType: eventType,
		Title: title,
		Body: clientName,
		ResourceType: "contract",
		ResourceID: contractID,
		TargetURL: "/admin/pipeline/" + proposalID,
		DedupeKey: eventType + ":" + dedupeSuffix,
	}); err != nil {
		a.logger.Warn("publish contract notification failed", "contract_id", contractID, "event_type", eventType, "error", err)
	}
}

func dailyEventDedupe(resourceID string) string {
	return fmt.Sprintf("%s:%s", resourceID, time.Now().UTC().Format("2006-01-02"))
}
