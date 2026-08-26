package onboarding

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
)

type Store struct {
	pool *pgxpool.Pool
}

type Document struct {
	ID               string
	DocumentType     string
	StorageKey       string
	OriginalFilename string
	MIMEType         string
	SizeBytes        int64
	SHA256           []byte
}

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) ByAcceptance(ctx context.Context, acceptanceID string) (domain.Onboarding, error) {
	var o domain.Onboarding
	err := s.pool.QueryRow(ctx, `
		select id::text,proposal_acceptance_id::text,client_id::text,status::text,
		       cnpj,legal_name,coalesce(trade_name,''),coalesce(street,''),coalesce(street_number,''),
		       coalesce(complement,''),coalesce(district,''),coalesce(city,''),coalesce(state,''),coalesce(postal_code,''),
		       coalesce(operation_type,''),coalesce(insurer,''),
		       coalesce(policy_start_date::text,''),coalesce(policy_end_date::text,''),
		       coalesce(broker_company,''),coalesce(broker_producer,''),
		       company_responsible_name,company_responsible_cpf,company_responsible_phone,
		       company_responsible_email::text,coalesce(company_responsible_role,''),company_responsible_authority_declared,
		       coalesce(finance_responsible_name,''),coalesce(finance_responsible_phone,''),coalesce(finance_responsible_email::text,'')
		from onboardings where proposal_acceptance_id=$1
	`, acceptanceID).Scan(
		&o.ID,&o.ProposalAcceptanceID,&o.ClientID,&o.Status,
		&o.CNPJ,&o.LegalName,&o.TradeName,&o.Street,&o.StreetNumber,&o.Complement,&o.District,&o.City,&o.State,&o.PostalCode,
		&o.OperationType,&o.Insurer,&o.PolicyStartDate,&o.PolicyEndDate,&o.BrokerCompany,&o.BrokerProducer,
		&o.CompanyResponsibleName,&o.CompanyResponsibleCPF,&o.CompanyResponsiblePhone,&o.CompanyResponsibleEmail,&o.CompanyResponsibleRole,&o.AuthorityDeclared,
		&o.FinanceResponsibleName,&o.FinanceResponsiblePhone,&o.FinanceResponsibleEmail,
	)
	if err != nil { return domain.Onboarding{}, err }

	goods, err := s.pool.Query(ctx, `select description from onboarding_goods where onboarding_id=$1 order by sort_order,id`, o.ID)
	if err != nil { return domain.Onboarding{}, err }
	for goods.Next() {
		var value string
		if err := goods.Scan(&value); err != nil { goods.Close(); return domain.Onboarding{}, err }
		o.Goods = append(o.Goods,value)
	}
	goods.Close()

	users, err := s.pool.Query(ctx, `select name,coalesce(phone,''),email::text from onboarding_system_users where onboarding_id=$1 order by sort_order,id`, o.ID)
	if err != nil { return domain.Onboarding{}, err }
	for users.Next() {
		var user domain.OnboardingSystemUser
		if err := users.Scan(&user.Name,&user.Phone,&user.Email); err != nil { users.Close(); return domain.Onboarding{}, err }
		o.SystemUsers = append(o.SystemUsers,user)
	}
	users.Close()
	return o, users.Err()
}

func (s *Store) Save(ctx context.Context, o domain.Onboarding) error {
	if !o.AuthorityDeclared {
		return fmt.Errorf("responsible authority declaration is required")
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil { return err }
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		update onboardings set
		  status=case when status='pending' then 'in_progress' else status end,
		  cnpj=$2,legal_name=$3,trade_name=nullif($4,''),street=nullif($5,''),street_number=nullif($6,''),
		  complement=nullif($7,''),district=nullif($8,''),city=nullif($9,''),state=nullif($10,''),postal_code=nullif($11,''),
		  operation_type=nullif($12,''),insurer=nullif($13,''),policy_start_date=nullif($14,'')::date,policy_end_date=nullif($15,'')::date,
		  broker_company=nullif($16,''),broker_producer=nullif($17,''),
		  company_responsible_name=$18,company_responsible_cpf=$19,company_responsible_phone=$20,
		  company_responsible_email=$21,company_responsible_role=nullif($22,''),company_responsible_authority_declared=$23,
		  finance_responsible_name=nullif($24,''),finance_responsible_phone=nullif($25,''),finance_responsible_email=nullif($26,'')::citext,
		  updated_at=now()
		where id=$1 and status in ('pending','in_progress','correction_requested')
	`,o.ID,o.CNPJ,o.LegalName,o.TradeName,o.Street,o.StreetNumber,o.Complement,o.District,o.City,o.State,o.PostalCode,
		o.OperationType,o.Insurer,o.PolicyStartDate,o.PolicyEndDate,o.BrokerCompany,o.BrokerProducer,
		o.CompanyResponsibleName,o.CompanyResponsibleCPF,o.CompanyResponsiblePhone,o.CompanyResponsibleEmail,o.CompanyResponsibleRole,o.AuthorityDeclared,
		o.FinanceResponsibleName,o.FinanceResponsiblePhone,o.FinanceResponsibleEmail)
	if err != nil { return err }

	if _, err := tx.Exec(ctx, `delete from onboarding_goods where onboarding_id=$1`,o.ID); err != nil { return err }
	for index, description := range o.Goods {
		if description == "" { continue }
		if _, err := tx.Exec(ctx, `insert into onboarding_goods(onboarding_id,description,sort_order) values ($1,$2,$3)`,o.ID,description,index); err != nil { return err }
	}

	if _, err := tx.Exec(ctx, `delete from onboarding_system_users where onboarding_id=$1`,o.ID); err != nil { return err }
	for index, user := range o.SystemUsers {
		if user.Name == "" || user.Email == "" { continue }
		if _, err := tx.Exec(ctx, `insert into onboarding_system_users(onboarding_id,name,phone,email,sort_order) values ($1,$2,nullif($3,''),$4,$5)`,o.ID,user.Name,user.Phone,user.Email,index); err != nil { return err }
	}

	return tx.Commit(ctx)
}

func (s *Store) AddDocument(ctx context.Context, onboardingID string, document Document) error {
	_, err := s.pool.Exec(ctx, `
		insert into uploaded_documents(onboarding_id,document_type,storage_key,original_filename,mime_type,size_bytes,sha256)
		values ($1,$2,$3,$4,$5,$6,$7)
	`,onboardingID,document.DocumentType,document.StorageKey,document.OriginalFilename,document.MIMEType,document.SizeBytes,document.SHA256)
	return err
}

func (s *Store) HasPolicy(ctx context.Context, onboardingID string) (bool,error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `select exists(select 1 from uploaded_documents where onboarding_id=$1 and document_type='insurance_policy' and status='uploaded')`,onboardingID).Scan(&exists)
	return exists,err
}

func (s *Store) Submit(ctx context.Context, onboardingID string) error {
	policy, err := s.HasPolicy(ctx,onboardingID)
	if err != nil { return err }
	if !policy { return fmt.Errorf("insurance policy is required") }

	command, err := s.pool.Exec(ctx, `
		update onboardings set status='submitted',submitted_at=now(),updated_at=now()
		where id=$1 and status in ('pending','in_progress','correction_requested')
	`,onboardingID)
	if err != nil { return err }
	if command.RowsAffected() != 1 { return fmt.Errorf("onboarding cannot be submitted in its current state") }
	return nil
}
