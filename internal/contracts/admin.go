package contracts

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
)

type DeliveryAccess struct {
	ContractID   string
	ContractStatus string
	SignerID     string
	SignerToken  string
	SignerName   string
	SignerEmail  string
	SignerStatus string
}

func (s *Store) SetDefaultTemplate(ctx context.Context, templateID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil { return err }
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `update contract_templates set is_default=false where is_default=true`); err != nil { return err }
	if _, err := tx.Exec(ctx, `update contract_templates set is_default=true where id=$1 and is_active=true`, templateID); err != nil { return err }
	return tx.Commit(ctx)
}

func (s *Store) TemplateIsDefault(ctx context.Context, templateID string) (bool,error) {
	var value bool
	err := s.pool.QueryRow(ctx, `select is_default from contract_templates where id=$1`,templateID).Scan(&value)
	return value,err
}

func (s *Store) ActiveByOnboarding(ctx context.Context,onboardingID string)(domain.Contract,error){
	var contract domain.Contract
	err:=s.pool.QueryRow(ctx,`
		select id::text,onboarding_id::text,proposal_version_id::text,template_version_id::text,status::text,
		       coalesce(rendered_markdown,''),coalesce(rendered_html,''),coalesce(pdf_storage_key,''),coalesce(document_sha256,'\\x'::bytea),
		       generated_at,sent_at,fully_signed_at,finalized_at,
		       coalesce(evidence_report_storage_key,''),coalesce(evidence_report_sha256,'\\x'::bytea),
		       coalesce(final_package_storage_key,''),coalesce(final_package_sha256,'\\x'::bytea)
		from contracts
		where onboarding_id=$1 and status<>'cancelled'
		order by created_at desc limit 1
	`,onboardingID).Scan(
		&contract.ID,&contract.OnboardingID,&contract.ProposalVersionID,&contract.TemplateVersionID,&contract.Status,
		&contract.RenderedMarkdown,&contract.RenderedHTML,&contract.PDFStorageKey,&contract.DocumentSHA256,
		&contract.GeneratedAt,&contract.SentAt,&contract.FullySignedAt,&contract.FinalizedAt,
		&contract.EvidenceReportStorageKey,&contract.EvidenceReportSHA256,&contract.FinalPackageStorageKey,&contract.FinalPackageSHA256,
	)
	return contract,err
}

func (s *Store) HasActiveForOnboarding(ctx context.Context,onboardingID string)(bool,error){
	_,err:=s.ActiveByOnboarding(ctx,onboardingID)
	if err==pgx.ErrNoRows{return false,nil}
	if err!=nil{return false,err}
	return true,nil
}

func (s *Store) DeliveryByOnboarding(ctx context.Context,onboardingID string)(DeliveryAccess,error){
	var access DeliveryAccess
	err:=s.pool.QueryRow(ctx,`
		select c.id::text,c.status::text,s.id::text,s.public_token::text,s.name,s.email::text,s.status::text
		from contracts c
		join contract_signers s on s.contract_id=c.id and s.signer_type='client'
		where c.onboarding_id=$1 and c.status<>'cancelled'
		order by c.created_at desc,s.sign_order,s.id
		limit 1
	`,onboardingID).Scan(&access.ContractID,&access.ContractStatus,&access.SignerID,&access.SignerToken,&access.SignerName,&access.SignerEmail,&access.SignerStatus)
	return access,err
}
