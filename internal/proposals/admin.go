package proposals

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
)

func (s *Store) List(ctx context.Context, userID string, all bool) ([]domain.Proposal, error) {
	query := `
		select p.id::text,p.client_id::text,coalesce(nullif(c.trade_name,''),nullif(c.legal_name,''),'Cliente não identificado'),p.title,p.status::text,p.current_version,
		       coalesce(v.public_token::text,''),p.is_default,p.valid_until,p.created_by::text,u.name,p.updated_at
		from proposals p
		join clients c on c.id=p.client_id
		join users u on u.id=p.created_by
		left join proposal_versions v on v.proposal_id=p.id and v.version_number=p.current_version and v.published_at is not null
	`
	args := []any{}
	if !all {
		query += ` where p.created_by=$1`
		args = append(args, userID)
	}
	query += ` order by p.updated_at desc`
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []domain.Proposal
	for rows.Next() {
		var item domain.Proposal
		if err := rows.Scan(
			&item.ID,
			&item.ClientID,
			&item.ClientName,
			&item.Title,
			&item.Status,
			&item.CurrentVersion,
			&item.PublicToken,
			&item.IsDefault,
			&item.ValidUntil,
			&item.CreatedBy,
			&item.CreatedByName,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) Default(ctx context.Context) (*domain.Proposal, error) {
	var item domain.Proposal
	err := s.pool.QueryRow(ctx, `
		select p.id::text,p.client_id::text,coalesce(nullif(c.trade_name,''),nullif(c.legal_name,''),'Cliente não identificado'),p.title,p.status::text,p.current_version,
		       coalesce(v.public_token::text,''),p.is_default,p.valid_until,p.created_by::text,u.name,p.updated_at
		from proposals p
		join clients c on c.id=p.client_id
		join users u on u.id=p.created_by
		left join proposal_versions v on v.proposal_id=p.id and v.version_number=p.current_version and v.published_at is not null
		where p.is_default = true
		limit 1
	`).Scan(
		&item.ID,
		&item.ClientID,
		&item.ClientName,
		&item.Title,
		&item.Status,
		&item.CurrentVersion,
		&item.PublicToken,
		&item.IsDefault,
		&item.ValidUntil,
		&item.CreatedBy,
		&item.CreatedByName,
		&item.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Store) SetDefault(ctx context.Context, proposalID string) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var exists bool
	if err := tx.QueryRow(ctx, `select exists(select 1 from proposals where id=$1)`, proposalID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("proposal not found")
	}
	if _, err := tx.Exec(ctx, `update proposals set is_default=false where is_default=true and id<>$1`, proposalID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `update proposals set is_default=true,updated_at=now() where id=$1`, proposalID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) ClearDefault(ctx context.Context, proposalID string) error {
	result, err := s.pool.Exec(ctx, `update proposals set is_default=false,updated_at=now() where id=$1 and is_default=true`, proposalID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("default proposal not found")
	}
	return nil
}
