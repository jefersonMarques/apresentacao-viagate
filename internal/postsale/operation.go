package postsale

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type Operation struct {
	Type           string
	Insurer        string
	StartDate      string
	EndDate        string
	BrokerCompany  string
	BrokerProducer string
	Saved          bool
}

type OperationVersion struct {
	Operation
	ActorName string
	ChangedAt time.Time
}

func ValidateOperation(operation Operation) error {
	if operation.Type != "" && operation.Type != "normal" && operation.Type != "avulsa" {
		return fmt.Errorf("invalid operation type")
	}
	start, startValid, err := optionalPolicyDate(operation.StartDate)
	if err != nil {
		return err
	}
	end, endValid, err := optionalPolicyDate(operation.EndDate)
	if err != nil {
		return err
	}
	if startValid && endValid && end.Before(start) {
		return fmt.Errorf("insurance end date must not precede the start date")
	}
	return nil
}

func OperationComplete(operation Operation) bool {
	return (operation.Type == "normal" || operation.Type == "avulsa") &&
		strings.TrimSpace(operation.Insurer) != "" &&
		operation.StartDate != "" && operation.EndDate != "" &&
		ValidateOperation(operation) == nil
}

func optionalPolicyDate(value string) (time.Time, bool, error) {
	if value == "" {
		return time.Time{}, false, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, false, fmt.Errorf("invalid insurance policy date")
	}
	return parsed, true, nil
}

func (s *Store) OperationByContract(ctx context.Context, contractID string) (Operation, error) {
	var operation Operation
	err := s.pool.QueryRow(ctx, `
		select coalesce(operation_type,''),coalesce(insurer,''),
		       coalesce(policy_start_date::text,''),coalesce(policy_end_date::text,''),
		       coalesce(broker_company,''),coalesce(broker_producer,'')
		from post_sale_operations where contract_id=$1
	`, contractID).Scan(&operation.Type,&operation.Insurer,&operation.StartDate,&operation.EndDate,
		&operation.BrokerCompany,&operation.BrokerProducer)
	if err == nil {
		operation.Saved = true
		return operation,nil
	}
	if err != pgx.ErrNoRows {
		return Operation{},err
	}

	err = s.pool.QueryRow(ctx, `
		select coalesce(o.operation_type,''),coalesce(o.insurer,''),
		       coalesce(o.policy_start_date::text,''),coalesce(o.policy_end_date::text,''),
		       coalesce(o.broker_company,''),coalesce(o.broker_producer,'')
		from contracts c
		join onboardings o on o.id=c.onboarding_id
		where c.id=$1 and c.deleted_at is null and o.deleted_at is null
	`, contractID).Scan(&operation.Type,&operation.Insurer,&operation.StartDate,&operation.EndDate,
		&operation.BrokerCompany,&operation.BrokerProducer)
	return operation,err
}

func (s *Store) SaveOperation(ctx context.Context, contractID string, operation Operation, actor Actor) error {
	operation.Type = strings.ToLower(strings.TrimSpace(operation.Type))
	operation.Insurer = strings.TrimSpace(operation.Insurer)
	operation.StartDate = strings.TrimSpace(operation.StartDate)
	operation.EndDate = strings.TrimSpace(operation.EndDate)
	operation.BrokerCompany = strings.TrimSpace(operation.BrokerCompany)
	operation.BrokerProducer = strings.TrimSpace(operation.BrokerProducer)

	if err := ValidateOperation(operation); err != nil {
		return err
	}
	if actor.Type != "user" || actor.UserID == "" {
        return fmt.Errorf("post-sale operation requires an authenticated commercial user")
    }

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var status string
	if err := tx.QueryRow(ctx, `
		select status::text from contracts
		where id=$1 and deleted_at is null for update
	`, contractID).Scan(&status); err != nil {
		return err
	}
	if status != "signed" {
		return fmt.Errorf("post-sale operation requires a signed contract")
	}

	var current Operation
	err = tx.QueryRow(ctx, `
		select coalesce(operation_type,''),coalesce(insurer,''),
		       coalesce(policy_start_date::text,''),coalesce(policy_end_date::text,''),
		       coalesce(broker_company,''),coalesce(broker_producer,'')
		from post_sale_operations where contract_id=$1 for update
	`, contractID).Scan(&current.Type,&current.Insurer,&current.StartDate,&current.EndDate,
		&current.BrokerCompany,&current.BrokerProducer)
	if err != nil && err != pgx.ErrNoRows {
		return err
	}
	if err == nil && current.Type == operation.Type && current.Insurer == operation.Insurer &&
		current.StartDate == operation.StartDate && current.EndDate == operation.EndDate &&
		current.BrokerCompany == operation.BrokerCompany && current.BrokerProducer == operation.BrokerProducer {
		return tx.Commit(ctx)
	}

	_, err = tx.Exec(ctx, `
		insert into post_sale_operations(
			contract_id,operation_type,insurer,policy_start_date,policy_end_date,
			broker_company,broker_producer,updated_by_user_id,updated_by_access_token_id
		) values($1,nullif($2,''),nullif($3,''),nullif($4,'')::date,nullif($5,'')::date,
			nullif($6,''),nullif($7,''),$8,$9)
		on conflict (contract_id) do update set
			operation_type=excluded.operation_type,
			insurer=excluded.insurer,
			policy_start_date=excluded.policy_start_date,
			policy_end_date=excluded.policy_end_date,
			broker_company=excluded.broker_company,
			broker_producer=excluded.broker_producer,
			updated_by_user_id=excluded.updated_by_user_id,
			updated_by_access_token_id=excluded.updated_by_access_token_id,
			updated_at=now()
	`,contractID,operation.Type,operation.Insurer,operation.StartDate,operation.EndDate,
		operation.BrokerCompany,operation.BrokerProducer,nullableID(actor.UserID),nullableID(actor.TokenID))
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		insert into post_sale_operation_history(
			contract_id,operation_type,insurer,policy_start_date,policy_end_date,
			broker_company,broker_producer,actor_type,actor_user_id,
			actor_access_token_id,actor_name,ip_address,user_agent
		) values($1,nullif($2,''),nullif($3,''),nullif($4,'')::date,nullif($5,'')::date,
			nullif($6,''),nullif($7,''),$8,$9,$10,$11,$12,$13)
	`,contractID,operation.Type,operation.Insurer,operation.StartDate,operation.EndDate,
		operation.BrokerCompany,operation.BrokerProducer,actor.Type,nullableID(actor.UserID),
		nullableID(actor.TokenID),actor.Name,actor.IP,actor.Agent)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) OperationHistory(ctx context.Context, contractID string) ([]OperationVersion,error) {
	rows,err := s.pool.Query(ctx,`
		select coalesce(h.operation_type,''),coalesce(h.insurer,''),
		       coalesce(h.policy_start_date::text,''),coalesce(h.policy_end_date::text,''),
		       coalesce(h.broker_company,''),coalesce(h.broker_producer,''),
		       coalesce(u.name,h.actor_name,'Sistema'),h.created_at
		from post_sale_operation_history h
		left join users u on u.id=h.actor_user_id
		where h.contract_id=$1
		order by h.created_at desc,h.id desc
		limit 100
	`,contractID)
	if err != nil {
		return nil,err
	}
	defer rows.Close()
	history := make([]OperationVersion,0)
	for rows.Next() {
		var version OperationVersion
		if err := rows.Scan(&version.Type,&version.Insurer,&version.StartDate,&version.EndDate,
			&version.BrokerCompany,&version.BrokerProducer,&version.ActorName,&version.ChangedAt); err != nil {
			return nil,err
		}
		version.Saved=true
		history=append(history,version)
	}
	return history,rows.Err()
}
