package activation

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

type SystemUser struct {
	Name  string
	Phone string
	Email string
}

type Profile struct {
	ID                       string
	ContractID               string
	ClientID                 string
	Status                   string
	LegalName                string
	TradeName                string
	CNPJ                     string
	Street                   string
	StreetNumber             string
	Complement               string
	District                 string
	City                     string
	State                    string
	PostalCode               string
	OperationType            string
	Insurer                  string
	PolicyStartDate          string
	PolicyEndDate            string
	BrokerCompany            string
	BrokerProducer           string
	CompanyResponsibleName   string
	CompanyResponsiblePhone  string
	CompanyResponsibleEmail  string
	FinanceResponsibleName   string
	FinanceResponsiblePhone  string
	FinanceResponsibleEmail  string
	Goods                    []string
	SystemUsers              []SystemUser
	SubmittedAt              *time.Time
	ActivatedAt              *time.Time
}

type Access struct {
	TokenID    string
	AccessType string
	Section    string
	Name       string
	Email      string
	Profile    Profile
}

type AdminItem struct {
	ProfileID       string
	ContractID      string
	ClientName      string
	CommercialName  string
	Status          string
	FullySignedAt   *time.Time
	SubmittedAt     *time.Time
	ActivatedAt     *time.Time
	UpdatedAt       time.Time
	GoodsCount      int
	SystemUserCount int
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) EnsureForSignedContract(ctx context.Context, contractID string) (Profile, error) {
	return s.EnsureForSignedContractWithExistingData(ctx, contractID)
}

func (s *Store) CreateAccessToken(ctx context.Context, activationID, accessType, section, name, email, signerID string, tokenHash []byte, expiresAt time.Time) error {
	if accessType != "owner" && accessType != "delegate" {
		return fmt.Errorf("invalid activation access type")
	}
	if section == "" {
		section = "all"
	}
	_, err := s.pool.Exec(ctx, `
		insert into activation_access_tokens(
			activation_id,token_hash,access_type,section,name,email,created_by_signer_id,expires_at
		) values($1,$2,$3,$4,nullif($5,''),nullif($6,''),nullif($7,'')::uuid,$8)
	`, activationID, tokenHash, accessType, section, name, email, signerID, expiresAt)
	return err
}

func (s *Store) AccessByToken(ctx context.Context, tokenHash []byte) (Access, error) {
	var access Access
	var profileID string
	err := s.pool.QueryRow(ctx, `
		select t.id::text,t.access_type,t.section,coalesce(t.name,''),coalesce(t.email::text,''),t.activation_id::text
		from activation_access_tokens t
		where t.token_hash=$1 and t.revoked_at is null and t.expires_at>now()
	`, tokenHash).Scan(&access.TokenID, &access.AccessType, &access.Section, &access.Name, &access.Email, &profileID)
	if err != nil {
		return Access{}, err
	}
	profile, err := s.ByID(ctx, profileID)
	if err != nil {
		return Access{}, err
	}
	access.Profile = profile
	_, _ = s.pool.Exec(ctx, `update activation_access_tokens set last_used_at=now() where id=$1`, access.TokenID)
	return access, nil
}

func (s *Store) ByID(ctx context.Context, id string) (Profile, error) {
	var profile Profile
	err := s.pool.QueryRow(ctx, `
		select a.id::text,a.contract_id::text,a.client_id::text,a.status,
		       o.legal_name,coalesce(o.trade_name,''),o.cnpj,
		       coalesce(o.street,''),coalesce(o.street_number,''),coalesce(o.complement,''),coalesce(o.district,''),
		       coalesce(o.city,''),coalesce(o.state,''),coalesce(o.postal_code,''),
		       coalesce(o.operation_type,''),coalesce(o.insurer,''),coalesce(o.policy_start_date::text,''),coalesce(o.policy_end_date::text,''),
		       coalesce(o.broker_company,''),coalesce(o.broker_producer,''),
		       o.company_responsible_name,o.company_responsible_phone,o.company_responsible_email::text,
		       coalesce(a.finance_responsible_name,''),coalesce(a.finance_responsible_phone,''),coalesce(a.finance_responsible_email::text,''),
		       a.submitted_at,a.activated_at
		from activation_profiles a
		join contracts c on c.id=a.contract_id
		join onboardings o on o.id=c.onboarding_id
		where a.id=$1
	`, id).Scan(
		&profile.ID, &profile.ContractID, &profile.ClientID, &profile.Status,
		&profile.LegalName, &profile.TradeName, &profile.CNPJ,
		&profile.Street, &profile.StreetNumber, &profile.Complement, &profile.District,
		&profile.City, &profile.State, &profile.PostalCode,
		&profile.OperationType, &profile.Insurer, &profile.PolicyStartDate, &profile.PolicyEndDate,
		&profile.BrokerCompany, &profile.BrokerProducer,
		&profile.CompanyResponsibleName, &profile.CompanyResponsiblePhone, &profile.CompanyResponsibleEmail,
		&profile.FinanceResponsibleName, &profile.FinanceResponsiblePhone, &profile.FinanceResponsibleEmail,
		&profile.SubmittedAt, &profile.ActivatedAt,
	)
	if err != nil {
		return Profile{}, err
	}

	goods, err := s.pool.Query(ctx, `select description from activation_goods where activation_id=$1 order by sort_order,id`, id)
	if err != nil {
		return Profile{}, err
	}
	for goods.Next() {
		var description string
		if err := goods.Scan(&description); err != nil {
			goods.Close()
			return Profile{}, err
		}
		profile.Goods = append(profile.Goods, description)
	}
	if err := goods.Err(); err != nil {
		goods.Close()
		return Profile{}, err
	}
	goods.Close()

	users, err := s.pool.Query(ctx, `select name,coalesce(phone,''),email::text from activation_system_users where activation_id=$1 order by sort_order,id`, id)
	if err != nil {
		return Profile{}, err
	}
	for users.Next() {
		var user SystemUser
		if err := users.Scan(&user.Name, &user.Phone, &user.Email); err != nil {
			users.Close()
			return Profile{}, err
		}
		profile.SystemUsers = append(profile.SystemUsers, user)
	}
	if err := users.Err(); err != nil {
		users.Close()
		return Profile{}, err
	}
	users.Close()
	return profile, nil
}

func (s *Store) Save(ctx context.Context, tokenID string, profile Profile, section string) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var activationID, allowedSection, status string
	err = tx.QueryRow(ctx, `
		select activation_id::text,section,a.status
		from activation_access_tokens t
		join activation_profiles a on a.id=t.activation_id
		where t.id=$1 and t.revoked_at is null and t.expires_at>now()
		for update of t,a
	`, tokenID).Scan(&activationID, &allowedSection, &status)
	if err != nil {
		return err
	}
	if status == "completed" || status == "under_internal_setup" || status == "activated" {
		return fmt.Errorf("activation data is already submitted")
	}
	if allowedSection != "all" && allowedSection != section {
		return fmt.Errorf("activation section not permitted")
	}

	switch section {
	case "finance":
		_, err = tx.Exec(ctx, `
			update activation_profiles
			set finance_responsible_name=nullif($2,''),finance_responsible_phone=nullif($3,''),finance_responsible_email=nullif($4,'')::citext,
			    status=case when status='pending' then 'in_progress' else status end,updated_at=now()
			where id=$1
		`, activationID, profile.FinanceResponsibleName, profile.FinanceResponsiblePhone, profile.FinanceResponsibleEmail)
	case "goods":
		if _, err = tx.Exec(ctx, `delete from activation_goods where activation_id=$1`, activationID); err == nil {
			for index, good := range profile.Goods {
				if _, err = tx.Exec(ctx, `insert into activation_goods(activation_id,description,sort_order) values($1,$2,$3)`, activationID, good, index); err != nil {
					break
				}
			}
		}
		if err == nil {
			_, err = tx.Exec(ctx, `update activation_profiles set status=case when status='pending' then 'in_progress' else status end,updated_at=now() where id=$1`, activationID)
		}
	case "users":
		if _, err = tx.Exec(ctx, `delete from activation_system_users where activation_id=$1`, activationID); err == nil {
			for index, user := range profile.SystemUsers {
				if _, err = tx.Exec(ctx, `insert into activation_system_users(activation_id,name,phone,email,sort_order) values($1,$2,nullif($3,''),$4,$5)`, activationID, user.Name, user.Phone, user.Email, index); err != nil {
					break
				}
			}
		}
		if err == nil {
			_, err = tx.Exec(ctx, `update activation_profiles set status=case when status='pending' then 'in_progress' else status end,updated_at=now() where id=$1`, activationID)
		}
	default:
		return fmt.Errorf("invalid activation section")
	}
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) Submit(ctx context.Context, tokenID string) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var activationID, status string
	var financeName, financeEmail, financePhone string
	err = tx.QueryRow(ctx, `
		select a.id::text,a.status,coalesce(a.finance_responsible_name,''),coalesce(a.finance_responsible_email::text,''),coalesce(a.finance_responsible_phone,'')
		from activation_access_tokens t
		join activation_profiles a on a.id=t.activation_id
		where t.id=$1 and t.revoked_at is null and t.expires_at>now()
		for update of t,a
	`, tokenID).Scan(&activationID, &status, &financeName, &financeEmail, &financePhone)
	if err != nil {
		return err
	}
	if status == "completed" || status == "under_internal_setup" || status == "activated" {
		return nil
	}
	if financeName == "" || financeEmail == "" || financePhone == "" {
		return fmt.Errorf("complete the financial contact")
	}
	var goodsCount, usersCount int
	if err := tx.QueryRow(ctx, `select count(*) from activation_goods where activation_id=$1`, activationID).Scan(&goodsCount); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `select count(*) from activation_system_users where activation_id=$1`, activationID).Scan(&usersCount); err != nil {
		return err
	}
	if goodsCount == 0 {
		return fmt.Errorf("add at least one transported good")
	}
	if usersCount == 0 {
		return fmt.Errorf("add at least one system user")
	}
	if _, err := tx.Exec(ctx, `update activation_profiles set status='completed',submitted_at=now(),updated_at=now() where id=$1`, activationID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) ListAdmin(ctx context.Context) ([]AdminItem, error) {
	rows, err := s.pool.Query(ctx, `
		select a.id::text,a.contract_id::text,o.legal_name,u.name,a.status,c.fully_signed_at,a.submitted_at,a.activated_at,a.updated_at,
		       (select count(*) from activation_goods g where g.activation_id=a.id),
		       (select count(*) from activation_system_users su where su.activation_id=a.id)
		from activation_profiles a
		join contracts c on c.id=a.contract_id
		join onboardings o on o.id=c.onboarding_id
		join proposal_acceptances pa on pa.id=o.proposal_acceptance_id
		join proposals p on p.id=pa.proposal_id
		join users u on u.id=p.created_by
		order by case a.status when 'completed' then 0 when 'in_progress' then 1 when 'pending' then 2 when 'under_internal_setup' then 3 else 4 end,a.updated_at desc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []AdminItem{}
	for rows.Next() {
		var item AdminItem
		if err := rows.Scan(&item.ProfileID, &item.ContractID, &item.ClientName, &item.CommercialName, &item.Status, &item.FullySignedAt, &item.SubmittedAt, &item.ActivatedAt, &item.UpdatedAt, &item.GoodsCount, &item.SystemUserCount); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) SetInternalStatus(ctx context.Context, activationID, status string) error {
	if status != "under_internal_setup" && status != "activated" {
		return fmt.Errorf("invalid activation status")
	}
	command, err := s.pool.Exec(ctx, `
		update activation_profiles
		set status=$2,activated_at=case when $2='activated' then coalesce(activated_at,now()) else activated_at end,updated_at=now()
		where id=$1 and status in ('completed','under_internal_setup')
	`, activationID, status)
	if err != nil {
		return err
	}
	if command.RowsAffected() != 1 {
		return fmt.Errorf("activation is not ready for this transition")
	}
	return nil
}
