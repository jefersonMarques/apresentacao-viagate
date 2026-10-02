package proposals

import (
	"context"
	"fmt"
	"net"

	"github.com/jackc/pgx/v5"
)

type SoftDeleteInput struct {
	ProposalID  string
	ActorUserID string
	IPAddress   net.IP
	UserAgent   string
}

type SoftDeleteResult struct {
	OriginalStatus            string
	Acceptances               int64
	Onboardings               int64
	Documents                 int64
	Contracts                 int64
	SignedContracts           int64
	Signers                   int64
	Activations               int64
	RevokedCustomerSessions   int64
	RevokedResumeTokens       int64
	RevokedActivationTokens   int64
	CancelledNotifications    int64
	HiddenInAppNotifications  int64
}

func (s *Store) SoftDeleteCascade(ctx context.Context, input SoftDeleteInput) (SoftDeleteResult, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return SoftDeleteResult{}, err
	}
	defer tx.Rollback(ctx)

	var result SoftDeleteResult
	if err := tx.QueryRow(ctx, `
		select status::text
		from proposals
		where id=$1 and deleted_at is null
		for update
	`, input.ProposalID).Scan(&result.OriginalStatus); err != nil {
		return SoftDeleteResult{}, err
	}

	if err := tx.QueryRow(ctx, `
		select count(*)
		from contracts c
		join onboardings o on o.id=c.onboarding_id
		join proposal_acceptances pa on pa.id=o.proposal_acceptance_id
		where pa.proposal_id=$1
		  and c.deleted_at is null
		  and c.status='signed'
		  and c.fully_signed_at is not null
	`, input.ProposalID).Scan(&result.SignedContracts); err != nil {
		return SoftDeleteResult{}, err
	}

	command, err := tx.Exec(ctx, `
		update customer_sessions cs
		set revoked_at=coalesce(cs.revoked_at,now())
		from proposal_acceptances pa
		where cs.proposal_acceptance_id=pa.id
		  and pa.proposal_id=$1
		  and cs.revoked_at is null
	`, input.ProposalID)
	if err != nil {
		return SoftDeleteResult{}, err
	}
	result.RevokedCustomerSessions = command.RowsAffected()

	command, err = tx.Exec(ctx, `
		update customer_resume_tokens crt
		set revoked_at=coalesce(crt.revoked_at,now())
		from proposal_acceptances pa
		where crt.proposal_acceptance_id=pa.id
		  and pa.proposal_id=$1
		  and crt.revoked_at is null
	`, input.ProposalID)
	if err != nil {
		return SoftDeleteResult{}, err
	}
	result.RevokedResumeTokens = command.RowsAffected()

	command, err = tx.Exec(ctx, `
		update activation_access_tokens aat
		set revoked_at=coalesce(aat.revoked_at,now())
		from activation_profiles ap
		join contracts c on c.id=ap.contract_id
		join onboardings o on o.id=c.onboarding_id
		join proposal_acceptances pa on pa.id=o.proposal_acceptance_id
		where aat.activation_id=ap.id
		  and pa.proposal_id=$1
		  and aat.revoked_at is null
	`, input.ProposalID)
	if err != nil {
		return SoftDeleteResult{}, err
	}
	result.RevokedActivationTokens = command.RowsAffected()


	command, err = tx.Exec(ctx, `
		with resources as (
			select $1::uuid as id
			union
			select pa.id from proposal_acceptances pa where pa.proposal_id=$1
			union
			select o.id
			from onboardings o
			join proposal_acceptances pa on pa.id=o.proposal_acceptance_id
			where pa.proposal_id=$1
			union
			select c.id
			from contracts c
			join onboardings o on o.id=c.onboarding_id
			join proposal_acceptances pa on pa.id=o.proposal_acceptance_id
			where pa.proposal_id=$1
			union
			select ap.id
			from activation_profiles ap
			join contracts c on c.id=ap.contract_id
			join onboardings o on o.id=c.onboarding_id
			join proposal_acceptances pa on pa.id=o.proposal_acceptance_id
			where pa.proposal_id=$1
		)
		update notification_outbox n
		set status='cancelled',
		    processing_at=null,
		    last_error='proposal soft deleted'
		where n.status in ('pending','processing')
		  and n.dedupe_key is not null
		  and exists(
			select 1
			from resources r
			where position(r.id::text in n.dedupe_key)>0
		  )
	`, input.ProposalID)
	if err != nil {
		return SoftDeleteResult{}, err
	}
	result.CancelledNotifications = command.RowsAffected()

	command, err = tx.Exec(ctx, `
		with resources as (
			select 'proposal'::text as resource_type,$1::uuid as id
			union all
			select 'onboarding',o.id
			from onboardings o
			join proposal_acceptances pa on pa.id=o.proposal_acceptance_id
			where pa.proposal_id=$1
			union all
			select 'contract',c.id
			from contracts c
			join onboardings o on o.id=c.onboarding_id
			join proposal_acceptances pa on pa.id=o.proposal_acceptance_id
			where pa.proposal_id=$1
			union all
			select 'activation',ap.id
			from activation_profiles ap
			join contracts c on c.id=ap.contract_id
			join onboardings o on o.id=c.onboarding_id
			join proposal_acceptances pa on pa.id=o.proposal_acceptance_id
			where pa.proposal_id=$1
		)
		update in_app_notifications n
		set deleted_at=coalesce(n.deleted_at,now())
		where n.deleted_at is null
		  and exists(
			select 1
			from resources r
			where r.resource_type=n.resource_type and r.id=n.resource_id
		  )
	`, input.ProposalID)
	if err != nil {
		return SoftDeleteResult{}, err
	}
	result.HiddenInAppNotifications = command.RowsAffected()

	command, err = tx.Exec(ctx, `
		update uploaded_documents d
		set deleted_at=coalesce(d.deleted_at,now()),
		    deleted_by=coalesce(d.deleted_by,$2)
		from onboardings o
		join proposal_acceptances pa on pa.id=o.proposal_acceptance_id
		where d.onboarding_id=o.id
		  and pa.proposal_id=$1
		  and d.deleted_at is null
	`, input.ProposalID, input.ActorUserID)
	if err != nil {
		return SoftDeleteResult{}, err
	}
	result.Documents = command.RowsAffected()

	command, err = tx.Exec(ctx, `
		update contract_signers signer
		set deleted_at=coalesce(signer.deleted_at,now()),
		    deleted_by=coalesce(signer.deleted_by,$2)
		from contracts c
		join onboardings o on o.id=c.onboarding_id
		join proposal_acceptances pa on pa.id=o.proposal_acceptance_id
		where signer.contract_id=c.id
		  and pa.proposal_id=$1
		  and signer.deleted_at is null
	`, input.ProposalID, input.ActorUserID)
	if err != nil {
		return SoftDeleteResult{}, err
	}
	result.Signers = command.RowsAffected()

	command, err = tx.Exec(ctx, `
		update activation_profiles ap
		set deleted_at=coalesce(ap.deleted_at,now()),
		    deleted_by=coalesce(ap.deleted_by,$2),
		    updated_at=now()
		from contracts c
		join onboardings o on o.id=c.onboarding_id
		join proposal_acceptances pa on pa.id=o.proposal_acceptance_id
		where ap.contract_id=c.id
		  and pa.proposal_id=$1
		  and ap.deleted_at is null
	`, input.ProposalID, input.ActorUserID)
	if err != nil {
		return SoftDeleteResult{}, err
	}
	result.Activations = command.RowsAffected()

	command, err = tx.Exec(ctx, `
		update contracts c
		set deleted_at=coalesce(c.deleted_at,now()),
		    deleted_by=coalesce(c.deleted_by,$2),
		    updated_at=now()
		from onboardings o
		join proposal_acceptances pa on pa.id=o.proposal_acceptance_id
		where c.onboarding_id=o.id
		  and pa.proposal_id=$1
		  and c.deleted_at is null
	`, input.ProposalID, input.ActorUserID)
	if err != nil {
		return SoftDeleteResult{}, err
	}
	result.Contracts = command.RowsAffected()

	command, err = tx.Exec(ctx, `
		update onboardings o
		set deleted_at=coalesce(o.deleted_at,now()),
		    deleted_by=coalesce(o.deleted_by,$2),
		    updated_at=now()
		from proposal_acceptances pa
		where o.proposal_acceptance_id=pa.id
		  and pa.proposal_id=$1
		  and o.deleted_at is null
	`, input.ProposalID, input.ActorUserID)
	if err != nil {
		return SoftDeleteResult{}, err
	}
	result.Onboardings = command.RowsAffected()

	command, err = tx.Exec(ctx, `
		update proposal_acceptances
		set deleted_at=coalesce(deleted_at,now()),
		    deleted_by=coalesce(deleted_by,$2)
		where proposal_id=$1
		  and deleted_at is null
	`, input.ProposalID, input.ActorUserID)
	if err != nil {
		return SoftDeleteResult{}, err
	}
	result.Acceptances = command.RowsAffected()

	command, err = tx.Exec(ctx, `
		update proposals
		set deleted_at=now(),
		    deleted_by=$2,
		    updated_by=$2,
		    is_default=false,
		    updated_at=now()
		where id=$1 and deleted_at is null
	`, input.ProposalID, input.ActorUserID)
	if err != nil {
		return SoftDeleteResult{}, err
	}
	if command.RowsAffected() != 1 {
		return SoftDeleteResult{}, fmt.Errorf("proposal was already deleted")
	}

	if _, err := tx.Exec(ctx, `
		insert into audit_events(
			actor_user_id,actor_type,event_type,resource_type,resource_id,ip_address,user_agent,metadata
		)
		values(
			$2,'user','proposal.soft_deleted','proposal',$1,$3,$4,
			jsonb_build_object(
				'original_status',$5::text,
				'acceptances',$6::bigint,
				'onboardings',$7::bigint,
				'documents',$8::bigint,
				'contracts',$9::bigint,
				'signed_contracts',$10::bigint,
				'signers',$11::bigint,
				'activations',$12::bigint,
				'revoked_customer_sessions',$13::bigint,
				'revoked_resume_tokens',$14::bigint,
				'revoked_activation_tokens',$15::bigint,
				'cancelled_notifications',$16::bigint,
				'hidden_in_app_notifications',$17::bigint,
				'evidence_retained',true
			)
		)
	`,
		input.ProposalID,
		input.ActorUserID,
		input.IPAddress,
		input.UserAgent,
		result.OriginalStatus,
		result.Acceptances,
		result.Onboardings,
		result.Documents,
		result.Contracts,
		result.SignedContracts,
		result.Signers,
		result.Activations,
		result.RevokedCustomerSessions,
		result.RevokedResumeTokens,
		result.RevokedActivationTokens,
		result.CancelledNotifications,
		result.HiddenInAppNotifications,
	); err != nil {
		return SoftDeleteResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return SoftDeleteResult{}, err
	}
	return result, nil
}
