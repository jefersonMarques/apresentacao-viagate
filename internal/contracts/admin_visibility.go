package contracts

import "context"

func (s *Store) ListVisibleContracts(ctx context.Context, userID string, allowAll bool) ([]AdminContractItem, error) {
	rows, err := s.pool.Query(ctx, `
		select c.id::text,
		       coalesce(nullif(cl.trade_name,''),nullif(cl.legal_name,''),'Cliente não identificado'),
		       p.title,u.name,
		       coalesce((
		           select cs.name
		           from contract_signers cs
		           where cs.contract_id=c.id and cs.signer_type='client'
		           order by cs.sign_order,cs.id
		           limit 1
		       ),''),
		       c.status::text,c.generated_at,c.sent_at,c.fully_signed_at,c.finalized_at
		from contracts c
		join onboardings o on o.id=c.onboarding_id
		join proposal_acceptances pa on pa.id=o.proposal_acceptance_id
		join proposals p on p.id=pa.proposal_id
		join clients cl on cl.id=o.client_id
		join users u on u.id=p.created_by
		where c.status<>'cancelled'
		  and ($2::boolean or p.created_by=$1)
		order by coalesce(c.fully_signed_at,c.sent_at,c.generated_at,c.created_at) desc,c.id desc
	`, userID, allowAll)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []AdminContractItem{}
	for rows.Next() {
		var item AdminContractItem
		if err := rows.Scan(
			&item.ID,
			&item.ClientName,
			&item.ProposalTitle,
			&item.CommercialName,
			&item.SignerName,
			&item.Status,
			&item.GeneratedAt,
			&item.SentAt,
			&item.FullySignedAt,
			&item.FinalizedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
