package contracts

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
)

type DeliveryAccess struct {
	ContractID     string
	ContractStatus string
	SignerID       string
	SignerToken    string
	SignerName     string
	SignerEmail    string
	SignerStatus   string
}

type AdminContractItem struct {
	ID             string
	ClientName     string
	ProposalTitle  string
	CommercialName string
	SignerName     string
	Status         string
	GeneratedAt    *time.Time
	SentAt         *time.Time
	FullySignedAt  *time.Time
	FinalizedAt    *time.Time
}

func (s *Store) ActiveByOnboarding(ctx context.Context, onboardingID string) (domain.Contract, error) {
	var contract domain.Contract
	err := s.pool.QueryRow(ctx, `
		select id::text,onboarding_id::text,proposal_version_id::text,template_version_id::text,status::text,
		       coalesce(rendered_markdown,''),coalesce(rendered_html,''),coalesce(pdf_storage_key,''),coalesce(document_sha256,'\\x'::bytea),
		       generated_at,sent_at,fully_signed_at,finalized_at,
		       coalesce(evidence_report_storage_key,''),coalesce(evidence_report_sha256,'\\x'::bytea),
		       coalesce(final_package_storage_key,''),coalesce(final_package_sha256,'\\x'::bytea)
		from contracts
		where onboarding_id=$1 and status<>'cancelled'
		order by created_at desc limit 1
	`, onboardingID).Scan(
		&contract.ID, &contract.OnboardingID, &contract.ProposalVersionID, &contract.TemplateVersionID, &contract.Status,
		&contract.RenderedMarkdown, &contract.RenderedHTML, &contract.PDFStorageKey, &contract.DocumentSHA256,
		&contract.GeneratedAt, &contract.SentAt, &contract.FullySignedAt, &contract.FinalizedAt,
		&contract.EvidenceReportStorageKey, &contract.EvidenceReportSHA256, &contract.FinalPackageStorageKey, &contract.FinalPackageSHA256,
	)
	return contract, err
}

func (s *Store) HasActiveForOnboarding(ctx context.Context, onboardingID string) (bool, error) {
	_, err := s.ActiveByOnboarding(ctx, onboardingID)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) DeliveryByOnboarding(ctx context.Context, onboardingID string) (DeliveryAccess, error) {
	var access DeliveryAccess
	err := s.pool.QueryRow(ctx, `
		select c.id::text,c.status::text,s.id::text,s.public_token::text,s.name,s.email::text,s.status::text
		from contracts c
		join contract_signers s on s.contract_id=c.id and s.signer_type='client'
		where c.onboarding_id=$1 and c.status<>'cancelled'
		order by c.created_at desc,s.sign_order,s.id
		limit 1
	`, onboardingID).Scan(&access.ContractID, &access.ContractStatus, &access.SignerID, &access.SignerToken, &access.SignerName, &access.SignerEmail, &access.SignerStatus)
	return access, err
}

func (s *Store) ListAdminContracts(ctx context.Context) ([]AdminContractItem, error) {
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
		order by coalesce(c.fully_signed_at,c.sent_at,c.generated_at,c.created_at) desc,c.id desc
	`)
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

func (s *Store) ArtifactKeysByContractID(ctx context.Context, contractID string) (ArtifactKeys, error) {
	var keys ArtifactKeys
	var finalizedAt *time.Time
	err := s.pool.QueryRow(ctx, `
		select coalesce(pdf_storage_key,''),
		       coalesce(evidence_report_storage_key,''),
		       coalesce(final_package_storage_key,''),
		       finalized_at
		from contracts
		where id=$1 and status<>'cancelled'
	`, contractID).Scan(&keys.ContractKey, &keys.EvidenceKey, &keys.PackageKey, &finalizedAt)
	if err != nil {
		return ArtifactKeys{}, err
	}
	keys.Finalized = finalizedAt != nil
	return keys, nil
}
