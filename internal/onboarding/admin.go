package onboarding

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
)

type AdminItem struct {
	ID             string
	OwnerUserID    string
	LegalName      string
	CNPJ           string
	Status         string
	ProposalTitle  string
	CommercialName string
	SubmittedAt    *time.Time
	ContractID     string
	ContractStatus string
	UpdatedAt      time.Time
}

type AdminDetail struct {
	Onboarding          domain.Onboarding
	OwnerUserID         string
	ProposalTitle       string
	CommercialName      string
	Documents           []Document
	ContractID          string
	ContractStatus      string
	ContractFinalizedAt *time.Time
	ReviewNotes         string
	SubmittedAt         *time.Time
	ReviewedAt          *time.Time
}

func (s *Store) ListVisible(ctx context.Context, userID string, allowAll bool) ([]AdminItem, error) {
	rows, err := s.pool.Query(ctx, `
		select o.id::text,p.created_by::text,o.legal_name,o.cnpj,o.status::text,p.title,u.name,o.submitted_at,
		       coalesce(c.id::text,''),coalesce(c.status::text,''),o.updated_at
		from onboardings o
		join proposal_acceptances pa on pa.id=o.proposal_acceptance_id
		join proposals p on p.id=pa.proposal_id
		join users u on u.id=p.created_by
		left join contracts c on c.onboarding_id=o.id and c.status<>'cancelled'
		where ($2::boolean or p.created_by=$1)
		order by coalesce(o.submitted_at,o.updated_at) desc
	`, userID, allowAll)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []AdminItem{}
	for rows.Next() {
		var item AdminItem
		if err := rows.Scan(&item.ID, &item.OwnerUserID, &item.LegalName, &item.CNPJ, &item.Status, &item.ProposalTitle, &item.CommercialName, &item.SubmittedAt, &item.ContractID, &item.ContractStatus, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ListForReview(ctx context.Context) ([]AdminItem, error) {
	return s.ListVisible(ctx, "00000000-0000-0000-0000-000000000000", true)
}

func (s *Store) AdminByID(ctx context.Context, id string) (AdminDetail, error) {
	var acceptanceID string
	var detail AdminDetail
	err := s.pool.QueryRow(ctx, `
		select o.proposal_acceptance_id::text,p.created_by::text,p.title,u.name,coalesce(o.review_notes,''),o.submitted_at,o.reviewed_at,
		       coalesce(c.id::text,''),coalesce(c.status::text,''),c.finalized_at
		from onboardings o
		join proposal_acceptances pa on pa.id=o.proposal_acceptance_id
		join proposals p on p.id=pa.proposal_id
		join users u on u.id=p.created_by
		left join contracts c on c.onboarding_id=o.id and c.status<>'cancelled'
		where o.id=$1
	`, id).Scan(
		&acceptanceID,
		&detail.OwnerUserID,
		&detail.ProposalTitle,
		&detail.CommercialName,
		&detail.ReviewNotes,
		&detail.SubmittedAt,
		&detail.ReviewedAt,
		&detail.ContractID,
		&detail.ContractStatus,
		&detail.ContractFinalizedAt,
	)
	if err != nil {
		return AdminDetail{}, err
	}
	detail.Onboarding, err = s.ByAcceptance(ctx, acceptanceID)
	if err != nil {
		return AdminDetail{}, err
	}

	rows, err := s.pool.Query(ctx, `
		select id::text,document_type,storage_key,original_filename,mime_type,size_bytes,sha256
		from uploaded_documents where onboarding_id=$1 order by uploaded_at desc,id desc
	`, id)
	if err != nil {
		return AdminDetail{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var document Document
		if err := rows.Scan(&document.ID, &document.DocumentType, &document.StorageKey, &document.OriginalFilename, &document.MIMEType, &document.SizeBytes, &document.SHA256); err != nil {
			return AdminDetail{}, err
		}
		detail.Documents = append(detail.Documents, document)
	}
	return detail, rows.Err()
}

func (s *Store) Review(ctx context.Context, id, userID, status, notes string) error {
	if status != "under_review" && status != "correction_requested" && status != "approved" {
		return fmt.Errorf("invalid onboarding review status")
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var current string
	var hasContract bool
	if err := tx.QueryRow(ctx, `
		select o.status::text,exists(select 1 from contracts c where c.onboarding_id=o.id and c.status<>'cancelled')
		from onboardings o where o.id=$1 for update
	`, id).Scan(&current, &hasContract); err != nil {
		return err
	}

	switch status {
	case "under_review":
		if current != "submitted" && current != "under_review" {
			return fmt.Errorf("onboarding cannot enter review from %s", current)
		}
	case "correction_requested":
		if hasContract {
			return fmt.Errorf("cannot request correction after contract generation")
		}
		if current != "submitted" && current != "under_review" {
			return fmt.Errorf("onboarding cannot request correction from %s", current)
		}
	case "approved":
		if current != "submitted" && current != "under_review" && current != "approved" {
			return fmt.Errorf("onboarding cannot be approved from %s", current)
		}
	}

	_, err = tx.Exec(ctx, `
		update onboardings set status=$2,review_notes=nullif($3,''),reviewed_by=$4,reviewed_at=now(),
		       approved_at=case when $2='approved' then coalesce(approved_at,now()) else approved_at end,
		       updated_at=now()
		where id=$1
	`, id, status, notes, userID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
