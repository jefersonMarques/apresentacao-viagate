package pipeline

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) List(ctx context.Context, userID string, all bool) ([]domain.PipelineItem, error) {
	query := `
		select p.id::text,p.title,coalesce(nullif(cl.trade_name,''),nullif(cl.legal_name,''),'Cliente não identificado'),u.name,
		       coalesce(a.name,''),coalesce(ct.signed_by_name,''),p.status::text,
		       coalesce(o.id::text,''),coalesce(o.status::text,''),
		       coalesce(ct.id::text,''),coalesce(ct.status::text,''),
		       a.accepted_at,o.submitted_at,ct.fully_signed_at,
		       greatest(p.updated_at,coalesce(o.updated_at,p.updated_at),coalesce(ct.updated_at,p.updated_at))
		from proposals p
		join clients cl on cl.id=p.client_id
		join users u on u.id=p.created_by
		left join lateral (
			select pa.id,pa.name,pa.accepted_at
			from proposal_acceptances pa
			where pa.proposal_id=p.id
			order by pa.accepted_at desc limit 1
		) a on true
		left join onboardings o on o.proposal_acceptance_id=a.id
		left join lateral (
			select c.id,c.status,c.fully_signed_at,c.updated_at,
			       coalesce((
			           select cs.name
			           from contract_signers cs
			           where cs.contract_id=c.id and cs.status='signed'
			           order by cs.sign_order,cs.signed_at
			           limit 1
			       ),'') as signed_by_name
			from contracts c
			where c.onboarding_id=o.id
			order by c.created_at desc limit 1
		) ct on true
	`
	args := []any{}
	if !all {
		query += ` where p.created_by=$1`
		args = append(args, userID)
	}
	query += ` order by greatest(p.updated_at,coalesce(o.updated_at,p.updated_at),coalesce(ct.updated_at,p.updated_at)) desc`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.PipelineItem
	for rows.Next() {
		var item domain.PipelineItem
		if err := rows.Scan(
			&item.ProposalID,&item.ProposalTitle,&item.ClientName,&item.CommercialName,
			&item.CustomerResponsibleName,&item.SignedByName,&item.ProposalStatus,
			&item.OnboardingID,&item.OnboardingStatus,&item.ContractID,&item.ContractStatus,
			&item.AcceptedAt,&item.SubmittedAt,&item.FullySignedAt,&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) Timeline(ctx context.Context, proposalID string) ([]domain.PipelineEvent, error) {
	rows, err := s.pool.Query(ctx, `
		with resources as (
			select p.id as resource_id from proposals p where p.id=$1
			union
			select pa.id from proposal_acceptances pa where pa.proposal_id=$1
			union
			select o.id from onboardings o join proposal_acceptances pa on pa.id=o.proposal_acceptance_id where pa.proposal_id=$1
			union
			select c.id from contracts c join onboardings o on o.id=c.onboarding_id join proposal_acceptances pa on pa.id=o.proposal_acceptance_id where pa.proposal_id=$1
		)
		select a.event_type,a.actor_type,coalesce(u.name,''),a.metadata,a.created_at
		from audit_events a
		left join users u on u.id=a.actor_user_id
		where a.resource_id in (select resource_id from resources)
		order by a.created_at desc,a.id desc
	`, proposalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []domain.PipelineEvent
	for rows.Next() {
		var event domain.PipelineEvent
		var metadata []byte
		if err := rows.Scan(&event.EventType,&event.ActorType,&event.ActorName,&metadata,&event.CreatedAt); err != nil {
			return nil, err
		}
		if len(metadata) > 0 {
			var normalized any
			if json.Unmarshal(metadata,&normalized) == nil {
				event.MetadataJSON, _ = json.Marshal(normalized)
			}
		}
		events = append(events,event)
	}
	return events,rows.Err()
}
