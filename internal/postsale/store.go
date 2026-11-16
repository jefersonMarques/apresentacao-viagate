package postsale

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

type Contact struct {
	Mode  string
	Name  string
	Phone string
	Email string
	Saved bool
}

type Actor struct {
	Type     string
	SignerID string
	UserID   string
	TokenID  string
	Name     string
	IP       net.IP
	Agent    string
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func ValidateContact(contact Contact) error {
	if contact.Mode != "self" && contact.Mode != "team" {
		return fmt.Errorf("invalid post-sale contact mode")
	}
	if strings.TrimSpace(contact.Name) == "" || strings.TrimSpace(contact.Phone) == "" {
		return fmt.Errorf("post-sale contact name and phone are required")
	}
	return nil
}

func (s *Store) ByContract(ctx context.Context, contractID string) (Contact, error) {
	var contact Contact
	err := s.pool.QueryRow(ctx, `
		select contact_mode, contact_name, contact_phone, coalesce(contact_email::text,'')
		from post_sale_contacts
		where contract_id=$1
	`, contractID).Scan(&contact.Mode, &contact.Name, &contact.Phone, &contact.Email)
	if err == pgx.ErrNoRows {
		return Contact{}, nil
	}
	if err != nil {
		return Contact{}, err
	}
	contact.Saved = true
	return contact, nil
}

func (s *Store) Save(ctx context.Context, contractID string, contact Contact, actor Actor) error {
	if err := ValidateContact(contact); err != nil {
		return err
	}
	if actor.Type == "customer" && actor.SignerID == "" {
		return fmt.Errorf("missing customer signer")
	}
	if actor.Type == "user" && actor.UserID == "" {
		return fmt.Errorf("missing staff user")
	}
	if actor.Type != "customer" && actor.Type != "user" {
		return fmt.Errorf("invalid contact actor")
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var status string
	if err := tx.QueryRow(ctx, `
		select status::text from contracts
		where id=$1 and deleted_at is null
		for update
	`, contractID).Scan(&status); err != nil {
		return err
	}
	if status != "signed" {
		return fmt.Errorf("contract must be signed before selecting an implementation contact")
	}

	var current Contact
	err = tx.QueryRow(ctx, `
		select contact_mode, contact_name, contact_phone, coalesce(contact_email::text,'')
		from post_sale_contacts where contract_id=$1
		for update
	`, contractID).Scan(&current.Mode, &current.Name, &current.Phone, &current.Email)
	if err != nil && err != pgx.ErrNoRows {
		return err
	}
	if err == nil && current.Mode == contact.Mode && current.Name == contact.Name &&
		current.Phone == contact.Phone && current.Email == contact.Email {
		return tx.Commit(ctx)
	}

	_, err = tx.Exec(ctx, `
		insert into post_sale_contacts (
			contract_id,contact_mode,contact_name,contact_phone,contact_email,
			updated_by_signer_id,updated_by_user_id
		) values ($1,$2,$3,$4,nullif($5,'')::citext,$6,$7)
		on conflict (contract_id) do update set
			contact_mode=excluded.contact_mode,
			contact_name=excluded.contact_name,
			contact_phone=excluded.contact_phone,
			contact_email=excluded.contact_email,
			updated_by_signer_id=excluded.updated_by_signer_id,
			updated_by_user_id=excluded.updated_by_user_id,
			updated_at=now()
	`, contractID, contact.Mode, contact.Name, contact.Phone, contact.Email,
		nullableID(actor.SignerID), nullableID(actor.UserID))
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		insert into post_sale_contact_history (
			contract_id,contact_mode,contact_name,contact_phone,contact_email,
			actor_type,actor_signer_id,actor_user_id,ip_address,user_agent
		) values ($1,$2,$3,$4,nullif($5,'')::citext,$6,$7,$8,$9,$10)
	`, contractID, contact.Mode, contact.Name, contact.Phone, contact.Email,
		actor.Type, nullableID(actor.SignerID), nullableID(actor.UserID), actor.IP, actor.Agent)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func nullableID(value string) any {
	if value == "" {
		return nil
	}
	return value
}

type Item struct {
	ContractID       string
	ClientName       string
	SalespersonName  string
	ProposalTitle    string
	ContactName      string
	SignedAt         time.Time
	HasContact       bool
	ActivationStatus string
}

type ContactVersion struct {
	Contact
	ActorName string
	ChangedAt time.Time
}

type Detail struct {
	Item
	CNPJ               string
	ResponsibleName    string
	ResponsibleEmail   string
	ResponsiblePhone   string
	RequiresPolicy     bool
	Contact            Contact
	Operation          Operation
	OperationHistory   []OperationVersion
	History            []ContactVersion
}

func (s *Store) List(ctx context.Context, userID string, all bool) ([]Item, error) {
	rows, err := s.pool.Query(ctx, `
		select c.id::text, o.legal_name, u.name, p.title, c.fully_signed_at,
		       coalesce(ps.contact_name,''), ps.contract_id is not null,
		       coalesce(a.status,'')
		from contracts c
		join onboardings o on o.id=c.onboarding_id
		join proposal_acceptances pa on pa.id=o.proposal_acceptance_id
		join proposals p on p.id=pa.proposal_id
		join users u on u.id=p.created_by
		left join post_sale_contacts ps on ps.contract_id=c.id
		left join activation_profiles a on a.contract_id=c.id and a.deleted_at is null
		where c.status='signed'
		  and c.deleted_at is null and o.deleted_at is null
		  and pa.deleted_at is null and p.deleted_at is null
		  and ($2 or p.created_by::text=$1)
		order by case when a.status in ('activated','under_internal_setup','completed') then 1 else 0 end,
		         c.fully_signed_at desc
	`, userID, all)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]Item, 0)
	for rows.Next() {
		var item Item
		if err := rows.Scan(
			&item.ContractID, &item.ClientName, &item.SalespersonName,
			&item.ProposalTitle, &item.SignedAt, &item.ContactName, &item.HasContact,
			&item.ActivationStatus,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) Detail(ctx context.Context, contractID, userID string, all bool) (Detail, error) {
	var detail Detail
	err := s.pool.QueryRow(ctx, `
		select c.id::text, o.legal_name, u.name, p.title, c.fully_signed_at,
		       o.cnpj, o.company_responsible_name, o.company_responsible_email::text,
		       o.company_responsible_phone, pv.requires_policy
		from contracts c
		join onboardings o on o.id=c.onboarding_id
		join proposal_acceptances pa on pa.id=o.proposal_acceptance_id
		join proposals p on p.id=pa.proposal_id
		join proposal_versions pv on pv.id=pa.proposal_version_id
		join users u on u.id=p.created_by
		where c.id=$1 and c.status='signed'
		  and c.deleted_at is null and o.deleted_at is null
		  and pa.deleted_at is null and p.deleted_at is null
		  and ($3 or p.created_by::text=$2)
	`, contractID, userID, all).Scan(
		&detail.ContractID, &detail.ClientName, &detail.SalespersonName,
		&detail.ProposalTitle, &detail.SignedAt,
		&detail.CNPJ, &detail.ResponsibleName, &detail.ResponsibleEmail, &detail.ResponsiblePhone, &detail.RequiresPolicy,
	)
	if err != nil {
		return Detail{}, err
	}
	detail.Contact, err = s.ByContract(ctx, contractID)
	if err != nil {
		return Detail{}, err
	}
	detail.Operation, err = s.OperationByContract(ctx, contractID)
	if err != nil {
		return Detail{}, err
	}
	detail.OperationHistory, err = s.OperationHistory(ctx, contractID)
	if err != nil {
		return Detail{}, err
	}
	rows, err := s.pool.Query(ctx, `
		select h.contact_mode,h.contact_name,h.contact_phone,coalesce(h.contact_email::text,''),
		       coalesce(u.name,cs.name,'Sistema'),h.created_at
		from post_sale_contact_history h
		left join users u on u.id=h.actor_user_id
		left join contract_signers cs on cs.id=h.actor_signer_id
		where h.contract_id=$1
		order by h.created_at desc,h.id desc
		limit 100
	`, contractID)
	if err != nil {
		return Detail{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var version ContactVersion
		if err := rows.Scan(&version.Mode,&version.Name,&version.Phone,&version.Email,&version.ActorName,&version.ChangedAt); err != nil {
			return Detail{}, err
		}
		version.Saved = true
		detail.History = append(detail.History,version)
	}
	if err := rows.Err(); err != nil {
		return Detail{}, err
	}
	return detail,nil
}

func StatusLabel(item Item) string {
	switch item.ActivationStatus {
	case "activated":
		return "Operação liberada"
	case "under_internal_setup":
		return "Em implantação"
	case "completed":
		return "Pronto para ativação"
	default:
		if item.HasContact {
			return "Em preparação"
		}
		return "Aguardando contato"
	}
}
