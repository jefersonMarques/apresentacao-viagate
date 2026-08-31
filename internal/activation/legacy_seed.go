package activation

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// EnsureForSignedContractWithExistingData creates the post-signature activation
// record and, only on its first creation, carries forward operational data that
// an in-flight customer may already have supplied in the former onboarding form.
func (s *Store) EnsureForSignedContractWithExistingData(ctx context.Context, contractID string) (Profile, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return Profile{}, err
	}
	defer tx.Rollback(ctx)

	var activationID, onboardingID string
	var inserted bool
	err = tx.QueryRow(ctx, `
		with source as (
			select c.id as contract_id,o.id as onboarding_id,o.client_id,
			       o.finance_responsible_name,o.finance_responsible_phone,o.finance_responsible_email
			from contracts c
			join onboardings o on o.id=c.onboarding_id
			where c.id=$1 and c.status='signed' and c.fully_signed_at is not null
		), inserted as (
			insert into activation_profiles(
				contract_id,client_id,finance_responsible_name,finance_responsible_phone,finance_responsible_email,
				status
			)
			select contract_id,client_id,finance_responsible_name,finance_responsible_phone,finance_responsible_email,
			       case when nullif(trim(coalesce(finance_responsible_name,'')),'') is not null then 'in_progress' else 'pending' end
			from source
			on conflict (contract_id) do nothing
			returning id
		)
		select coalesce(i.id,a.id)::text,s.onboarding_id::text,(i.id is not null)
		from source s
		left join inserted i on true
		left join activation_profiles a on a.contract_id=s.contract_id
	`, contractID).Scan(&activationID, &onboardingID, &inserted)
	if err != nil {
		return Profile{}, fmt.Errorf("ensure activation profile: %w", err)
	}

	if inserted {
		if _, err := tx.Exec(ctx, `
			insert into activation_goods(activation_id,description,sort_order)
			select $1,description,sort_order
			from onboarding_goods
			where onboarding_id=$2 and nullif(trim(description),'') is not null
			order by sort_order,id
		`, activationID, onboardingID); err != nil {
			return Profile{}, fmt.Errorf("seed activation goods: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			insert into activation_system_users(activation_id,name,phone,email,sort_order)
			select $1,name,phone,email,sort_order
			from onboarding_system_users
			where onboarding_id=$2
			order by sort_order,id
		`, activationID, onboardingID); err != nil {
			return Profile{}, fmt.Errorf("seed activation system users: %w", err)
		}

		var goodsCount, usersCount int
		if err := tx.QueryRow(ctx, `select count(*) from activation_goods where activation_id=$1`, activationID).Scan(&goodsCount); err != nil {
			return Profile{}, err
		}
		if err := tx.QueryRow(ctx, `select count(*) from activation_system_users where activation_id=$1`, activationID).Scan(&usersCount); err != nil {
			return Profile{}, err
		}
		if goodsCount > 0 || usersCount > 0 {
			if _, err := tx.Exec(ctx, `update activation_profiles set status='in_progress',updated_at=now() where id=$1 and status='pending'`, activationID); err != nil {
				return Profile{}, err
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Profile{}, err
	}
	return s.ByID(ctx, activationID)
}
