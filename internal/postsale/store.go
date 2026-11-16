package postsale

import (
	"context"
	"fmt"
	"net"
	"strings"

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
