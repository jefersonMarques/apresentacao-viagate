package proposals

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

type PublicProposal struct {
	ProposalID       string
	VersionID        string
	VersionNumber    int
	PublicToken      string
	Title            string
	ClientID         string
	ClientName       string
	ClientCNPJ       string
	PricingModel     string
	Content          map[string]any
	Conditions       []string
	MinimumInvoice   float64
	SetupFee         float64
	ContentHash      []byte
	ValidUntil       *time.Time
	Items            []Item
}

type Item struct {
	GroupName  string
	Label      string
	Unit       string
	Price      float64
	IsOptional bool
	SortOrder  int
}

type AcceptanceInput struct {
	Name              string
	Email             string
	CPF               string
	Phone             string
	Role              string
	AuthorityDeclared bool
	IPAddress         net.IP
	UserAgent         string
	SessionID         string
}

type AcceptanceResult struct {
	AcceptanceID string
	OnboardingID string
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) PublicByToken(ctx context.Context, token string) (PublicProposal, error) {
	var result PublicProposal
	var contentJSON []byte
	var conditionsJSON []byte
	err := s.pool.QueryRow(ctx, `
		select p.id::text, v.id::text, v.version_number, v.public_token::text, p.title,
		       c.id::text, c.legal_name, coalesce(c.cnpj,''), v.pricing_model,
		       v.content, v.conditions, v.minimum_invoice, v.setup_fee, v.content_hash, p.valid_until
		from proposal_versions v
		join proposals p on p.id=v.proposal_id
		join clients c on c.id=p.client_id
		where v.public_token=$1 and v.published_at is not null
		  and p.status in ('published','accepted')
	`, token).Scan(
		&result.ProposalID,
		&result.VersionID,
		&result.VersionNumber,
		&result.PublicToken,
		&result.Title,
		&result.ClientID,
		&result.ClientName,
		&result.ClientCNPJ,
		&result.PricingModel,
		&contentJSON,
		&conditionsJSON,
		&result.MinimumInvoice,
		&result.SetupFee,
		&result.ContentHash,
		&result.ValidUntil,
	)
	if err != nil {
		return PublicProposal{}, err
	}
	if result.ValidUntil != nil && result.ValidUntil.Before(time.Now().Truncate(24*time.Hour)) {
		return PublicProposal{}, fmt.Errorf("proposal expired")
	}
	if err := json.Unmarshal(contentJSON, &result.Content); err != nil {
		return PublicProposal{}, fmt.Errorf("decode proposal content: %w", err)
	}
	if len(conditionsJSON) > 0 {
		_ = json.Unmarshal(conditionsJSON, &result.Conditions)
	}

	rows, err := s.pool.Query(ctx, `
		select group_name,label,coalesce(unit,''),price,is_optional,sort_order
		from proposal_items
		where proposal_version_id=$1
		order by sort_order,id
	`, result.VersionID)
	if err != nil {
		return PublicProposal{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.GroupName,&item.Label,&item.Unit,&item.Price,&item.IsOptional,&item.SortOrder); err != nil {
			return PublicProposal{}, err
		}
		result.Items = append(result.Items,item)
	}
	return result, rows.Err()
}

func (s *Store) Accept(ctx context.Context, proposal PublicProposal, input AcceptanceInput) (AcceptanceResult, error) {
	if !input.AuthorityDeclared {
		return AcceptanceResult{}, fmt.Errorf("authority declaration is required")
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return AcceptanceResult{}, err
	}
	defer tx.Rollback(ctx)

	var currentStatus string
	err = tx.QueryRow(ctx, `select status::text from proposals where id=$1 for update`, proposal.ProposalID).Scan(&currentStatus)
	if err != nil {
		return AcceptanceResult{}, err
	}
	if currentStatus != "published" && currentStatus != "accepted" {
		return AcceptanceResult{}, fmt.Errorf("proposal is not available for acceptance")
	}

	var result AcceptanceResult
	err = tx.QueryRow(ctx, `
		insert into proposal_acceptances(
			proposal_id,proposal_version_id,proposal_hash,
			accepted_by_name,accepted_by_email,accepted_by_cpf,accepted_by_phone,accepted_by_role,
			authority_declared,acceptance_text_version,ip_address,user_agent,session_id
		) values ($1,$2,$3,$4,$5,$6,$7,$8,true,'v1',$9,$10,$11)
		on conflict (proposal_version_id) do update set proposal_version_id=excluded.proposal_version_id
		returning id::text
	`,
		proposal.ProposalID,
		proposal.VersionID,
		proposal.ContentHash,
		input.Name,
		input.Email,
		input.CPF,
		input.Phone,
		input.Role,
		nullableIP(input.IPAddress),
		input.UserAgent,
		input.SessionID,
	).Scan(&result.AcceptanceID)
	if err != nil {
		return AcceptanceResult{}, fmt.Errorf("record proposal acceptance: %w", err)
	}

	err = tx.QueryRow(ctx, `
		insert into onboardings(
			proposal_acceptance_id,client_id,status,cnpj,legal_name,trade_name,
			street,street_number,complement,district,city,state,postal_code,
			company_responsible_name,company_responsible_cpf,company_responsible_phone,
			company_responsible_email,company_responsible_role,company_responsible_authority_declared
		)
		select $1,c.id,'pending',coalesce(c.cnpj,''),c.legal_name,c.trade_name,
		       c.street,c.street_number,c.complement,c.district,c.city,c.state,c.postal_code,
		       $2,$3,$4,$5,$6,true
		from clients c where c.id=$7
		on conflict (proposal_acceptance_id) do update set updated_at=now()
		returning id::text
	`, result.AcceptanceID,input.Name,input.CPF,input.Phone,input.Email,input.Role,proposal.ClientID).Scan(&result.OnboardingID)
	if err != nil {
		return AcceptanceResult{}, fmt.Errorf("create onboarding: %w", err)
	}

	if _, err := tx.Exec(ctx, `update proposals set status='accepted',updated_at=now() where id=$1`, proposal.ProposalID); err != nil {
		return AcceptanceResult{}, err
	}

	if _, err := tx.Exec(ctx, `
		insert into audit_events(actor_type,event_type,resource_type,resource_id,ip_address,user_agent,metadata)
		values ('customer','proposal.accepted','proposal',$1,$2,$3,jsonb_build_object('acceptance_id',$4,'version',$5))
	`,proposal.ProposalID,nullableIP(input.IPAddress),input.UserAgent,result.AcceptanceID,proposal.VersionNumber); err != nil {
		return AcceptanceResult{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return AcceptanceResult{}, err
	}
	return result,nil
}

func (s *Store) CreateCustomerSession(ctx context.Context, acceptanceID string, tokenHash []byte, ip net.IP, userAgent string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		insert into customer_sessions(proposal_acceptance_id,token_hash,ip_address,user_agent,expires_at)
		values ($1,$2,$3,$4,$5)
	`,acceptanceID,tokenHash,nullableIP(ip),userAgent,expiresAt)
	return err
}

func (s *Store) CustomerSessionAcceptance(ctx context.Context, tokenHash []byte) (string,error) {
	var acceptanceID string
	err := s.pool.QueryRow(ctx, `
		select proposal_acceptance_id::text
		from customer_sessions
		where token_hash=$1 and revoked_at is null and expires_at > now()
	`,tokenHash).Scan(&acceptanceID)
	return acceptanceID,err
}

func nullableIP(ip net.IP) any {
	if ip == nil { return nil }
	return ip.String()
}
