package contracts

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
)

type Store struct {
	pool *pgxpool.Pool
}

type SignerAccess struct {
	Signer   domain.ContractSigner
	Contract domain.Contract
	HTML     string
}

type ArtifactKeys struct {
	ContractKey string
	EvidenceKey string
	PackageKey  string
	Finalized   bool
}

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) ListTemplates(ctx context.Context) ([]domain.ContractTemplate,error) {
	rows,err := s.pool.Query(ctx, `select id::text,name,coalesce(description,''),current_version,is_active from contract_templates order by name`)
	if err != nil { return nil,err }
	defer rows.Close()
	var items []domain.ContractTemplate
	for rows.Next() {
		var item domain.ContractTemplate
		if err := rows.Scan(&item.ID,&item.Name,&item.Description,&item.CurrentVersion,&item.IsActive); err != nil { return nil,err }
		items=append(items,item)
	}
	return items,rows.Err()
}

func (s *Store) SaveTemplateVersion(ctx context.Context, templateID,name,description,markdown string, hash []byte, userID string) (domain.ContractTemplateVersion,error) {
	tx,err := s.pool.BeginTx(ctx,pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil { return domain.ContractTemplateVersion{},err }
	defer tx.Rollback(ctx)

	if templateID == "" {
		if err := tx.QueryRow(ctx, `insert into contract_templates(name,description,created_by) values($1,nullif($2,''),$3) returning id::text`,name,description,userID).Scan(&templateID); err != nil {
			return domain.ContractTemplateVersion{},err
		}
	}

	var nextVersion int
	if err := tx.QueryRow(ctx, `select current_version+1 from contract_templates where id=$1 for update`,templateID).Scan(&nextVersion); err != nil {
		return domain.ContractTemplateVersion{},err
	}

	var version domain.ContractTemplateVersion
	err = tx.QueryRow(ctx, `
		insert into contract_template_versions(contract_template_id,version_number,markdown,template_hash,created_by)
		values($1,$2,$3,$4,$5)
		returning id::text,contract_template_id::text,version_number,markdown,template_hash
	`,templateID,nextVersion,markdown,hash,userID).Scan(&version.ID,&version.ContractTemplateID,&version.VersionNumber,&version.Markdown,&version.TemplateHash)
	if err != nil { return domain.ContractTemplateVersion{},err }

	if _,err := tx.Exec(ctx, `update contract_templates set name=$2,description=nullif($3,''),current_version=$4,updated_at=now() where id=$1`,templateID,name,description,nextVersion); err != nil {
		return domain.ContractTemplateVersion{},err
	}
	if err := tx.Commit(ctx); err != nil { return domain.ContractTemplateVersion{},err }
	return version,nil
}

func (s *Store) SetDefaultTemplate(ctx context.Context,templateID string) error {
	tx,err:=s.pool.Begin(ctx);if err!=nil{return err};defer tx.Rollback(ctx)
	if _,err:=tx.Exec(ctx,`update contract_templates set is_default=false where is_default=true`);err!=nil{return err}
	command,err:=tx.Exec(ctx,`update contract_templates set is_default=true where id=$1 and is_active=true`,templateID);if err!=nil{return err}
	if command.RowsAffected()!=1{return fmt.Errorf("contract template not found")}
	return tx.Commit(ctx)
}

func (s *Store) LatestTemplateVersion(ctx context.Context, templateID string) (domain.ContractTemplateVersion,error) {
	var v domain.ContractTemplateVersion
	err := s.pool.QueryRow(ctx, `
		select v.id::text,v.contract_template_id::text,v.version_number,v.markdown,v.template_hash
		from contract_template_versions v
		join contract_templates t on t.id=v.contract_template_id and t.current_version=v.version_number
		where t.id=$1 and t.is_active=true
	`,templateID).Scan(&v.ID,&v.ContractTemplateID,&v.VersionNumber,&v.Markdown,&v.TemplateHash)
	return v,err
}

func (s *Store) CreateGeneratedContract(ctx context.Context,onboardingID,proposalVersionID,templateVersionID,markdown,html,storageKey string,documentHash []byte,createdBy string) (domain.Contract,error) {
	var contract domain.Contract
	err := s.pool.QueryRow(ctx, `
		insert into contracts(onboarding_id,proposal_version_id,template_version_id,status,rendered_markdown,rendered_html,pdf_storage_key,document_sha256,generated_at,created_by)
		values($1,$2,$3,'generated',$4,$5,$6,$7,now(),$8)
		returning id::text,onboarding_id::text,proposal_version_id::text,template_version_id::text,status::text,rendered_markdown,rendered_html,pdf_storage_key,document_sha256,generated_at
	`,onboardingID,proposalVersionID,templateVersionID,markdown,html,storageKey,documentHash,createdBy).Scan(
		&contract.ID,&contract.OnboardingID,&contract.ProposalVersionID,&contract.TemplateVersionID,&contract.Status,
		&contract.RenderedMarkdown,&contract.RenderedHTML,&contract.PDFStorageKey,&contract.DocumentSHA256,&contract.GeneratedAt,
	)
	return contract,err
}

func (s *Store) AddSigner(ctx context.Context,contractID,signerType,name,email,cpf,role string,signOrder int) (domain.ContractSigner,error) {
	var signer domain.ContractSigner
	err := s.pool.QueryRow(ctx, `
		insert into contract_signers(contract_id,signer_type,name,email,cpf,role,sign_order)
		values($1,$2,$3,$4,$5,nullif($6,''),$7)
		returning id::text,contract_id::text,signer_type,name,email::text,cpf,coalesce(role,''),sign_order,status::text
	`,contractID,signerType,name,email,cpf,role,signOrder).Scan(&signer.ID,&signer.ContractID,&signer.SignerType,&signer.Name,&signer.Email,&signer.CPF,&signer.Role,&signer.SignOrder,&signer.Status)
	return signer,err
}

func (s *Store) SignerByPublicToken(ctx context.Context,token string) (SignerAccess,error) {
	var access SignerAccess
	err := s.pool.QueryRow(ctx, `
		select s.id::text,s.contract_id::text,s.signer_type,s.name,s.email::text,s.cpf,coalesce(s.role,''),s.sign_order,s.status::text,s.signed_at,
		       c.id::text,c.onboarding_id::text,c.proposal_version_id::text,c.template_version_id::text,c.status::text,
		       coalesce(c.rendered_markdown,''),coalesce(c.rendered_html,''),coalesce(c.pdf_storage_key,''),c.document_sha256,
		       coalesce(c.evidence_report_storage_key,''),c.evidence_report_sha256,
		       coalesce(c.final_package_storage_key,''),c.final_package_sha256,
		       c.generated_at,c.sent_at,c.fully_signed_at,c.finalized_at
		from contract_signers s join contracts c on c.id=s.contract_id
		where s.public_token=$1 and c.status in ('generated','sent','partially_signed','signed')
	`,token).Scan(
		&access.Signer.ID,&access.Signer.ContractID,&access.Signer.SignerType,&access.Signer.Name,&access.Signer.Email,&access.Signer.CPF,&access.Signer.Role,&access.Signer.SignOrder,&access.Signer.Status,&access.Signer.SignedAt,
		&access.Contract.ID,&access.Contract.OnboardingID,&access.Contract.ProposalVersionID,&access.Contract.TemplateVersionID,&access.Contract.Status,
		&access.Contract.RenderedMarkdown,&access.Contract.RenderedHTML,&access.Contract.PDFStorageKey,&access.Contract.DocumentSHA256,
		&access.Contract.EvidenceReportStorageKey,&access.Contract.EvidenceReportSHA256,&access.Contract.FinalPackageStorageKey,&access.Contract.FinalPackageSHA256,
		&access.Contract.GeneratedAt,&access.Contract.SentAt,&access.Contract.FullySignedAt,&access.Contract.FinalizedAt,
	)
	access.HTML=access.Contract.RenderedHTML
	return access,err
}

func (s *Store) SignerPublicToken(ctx context.Context,signerID string) (string,error) {
	var token string
	err := s.pool.QueryRow(ctx, `select public_token::text from contract_signers where id=$1`,signerID).Scan(&token)
	return token,err
}

func (s *Store) ArtifactKeysBySignerToken(ctx context.Context,token string) (ArtifactKeys,error) {
	var keys ArtifactKeys
	var finalizedAt *time.Time
	err:=s.pool.QueryRow(ctx,`
		select c.pdf_storage_key,coalesce(c.evidence_report_storage_key,''),coalesce(c.final_package_storage_key,''),c.finalized_at
		from contract_signers s join contracts c on c.id=s.contract_id
		where s.public_token=$1 and s.status='signed' and c.status='signed'
	`,token).Scan(&keys.ContractKey,&keys.EvidenceKey,&keys.PackageKey,&finalizedAt)
	if err!=nil{return ArtifactKeys{},err}
	keys.Finalized=finalizedAt!=nil
	return keys,nil
}

func (s *Store) CreateChallenge(ctx context.Context,signerID string,otpHash []byte,expiresAt time.Time) error {
	tx,err:=s.pool.Begin(ctx)
	if err != nil { return err }
	defer tx.Rollback(ctx)
	if _,err:=tx.Exec(ctx,`delete from signature_challenges where contract_signer_id=$1 and verified_at is null`,signerID);err!=nil{return err}
	if _,err:=tx.Exec(ctx,`insert into signature_challenges(contract_signer_id,otp_hash,expires_at) values($1,$2,$3)`,signerID,otpHash,expiresAt);err!=nil{return err}
	if _,err:=tx.Exec(ctx,`update contract_signers set status='otp_sent' where id=$1 and status in ('pending','otp_sent')`,signerID);err!=nil{return err}
	return tx.Commit(ctx)
}

func (s *Store) VerifyChallenge(ctx context.Context,signerID string,otpHash []byte) error {
	tx,err:=s.pool.BeginTx(ctx,pgx.TxOptions{IsoLevel:pgx.Serializable})
	if err!=nil{return err}
	defer tx.Rollback(ctx)
	var challengeID string
	var expected []byte
	var attempts int
	err=tx.QueryRow(ctx,`
		select id::text,otp_hash,attempts from signature_challenges
		where contract_signer_id=$1 and verified_at is null and expires_at>now()
		order by created_at desc limit 1 for update
	`,signerID).Scan(&challengeID,&expected,&attempts)
	if err!=nil{return err}
	if attempts>=5{return fmt.Errorf("maximum OTP attempts reached")}
	if !bytes.Equal(expected,otpHash){
		_,_ = tx.Exec(ctx,`update signature_challenges set attempts=attempts+1 where id=$1`,challengeID)
		_ = tx.Commit(ctx)
		return fmt.Errorf("invalid OTP")
	}
	if _,err:=tx.Exec(ctx,`update signature_challenges set verified_at=now() where id=$1`,challengeID);err!=nil{return err}
	if _,err:=tx.Exec(ctx,`update contract_signers set status='verified' where id=$1`,signerID);err!=nil{return err}
	return tx.Commit(ctx)
}

func (s *Store) Sign(ctx context.Context,signerID string,documentHash []byte,sessionID string,ip net.IP,userAgent string) (string,bool,error) {
	tx,err:=s.pool.BeginTx(ctx,pgx.TxOptions{IsoLevel:pgx.Serializable})
	if err!=nil{return "",false,err}
	defer tx.Rollback(ctx)

	var contractID string
	var storedHash []byte
	var status string
	err=tx.QueryRow(ctx,`
		select c.id::text,c.document_sha256,s.status::text
		from contract_signers s join contracts c on c.id=s.contract_id
		where s.id=$1 for update of s,c
	`,signerID).Scan(&contractID,&storedHash,&status)
	if err!=nil{return "",false,err}
	if status!="verified" { return "",false,fmt.Errorf("signer identity is not verified") }
	if !bytes.Equal(storedHash,documentHash){return "",false,fmt.Errorf("contract document hash changed")}

	if _,err:=tx.Exec(ctx,`update contract_signers set status='signed',signed_at=now(),signed_document_hash=$2,signature_session_id=$3 where id=$1`,signerID,documentHash,sessionID);err!=nil{return "",false,err}
	if _,err:=tx.Exec(ctx,`
		insert into signature_events(contract_id,contract_signer_id,event_type,document_hash,ip_address,user_agent,session_id)
		values($1,$2,'contract.signed',$3,$4,$5,$6)
	`,contractID,signerID,documentHash,nullableIP(ip),userAgent,sessionID);err!=nil{return "",false,err}

	var pending int
	if err:=tx.QueryRow(ctx,`select count(*) from contract_signers where contract_id=$1 and status<>'signed'`,contractID).Scan(&pending);err!=nil{return "",false,err}
	fullySigned:=pending==0
	if fullySigned {
		if _,err:=tx.Exec(ctx,`update contracts set status='signed',fully_signed_at=now(),updated_at=now() where id=$1`,contractID);err!=nil{return "",false,err}
	} else {
		if _,err:=tx.Exec(ctx,`update contracts set status='partially_signed',updated_at=now() where id=$1`,contractID);err!=nil{return "",false,err}
	}
	if err:=tx.Commit(ctx);err!=nil{return "",false,err}
	return contractID,fullySigned,nil
}

func (s *Store) MarkSent(ctx context.Context,contractID string) error {
	_,err:=s.pool.Exec(ctx,`update contracts set status='sent',sent_at=coalesce(sent_at,now()),updated_at=now() where id=$1 and status in ('generated','sent')`,contractID)
	return err
}

func nullableIP(ip net.IP) any { if ip==nil{return nil}; return ip.String() }
