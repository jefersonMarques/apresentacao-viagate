package proposals

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jefersonMarques/apresentacao-viagate/internal/legaltext"
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
	ClientTradeName  string
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
	var currentTitle,currentClientName,currentClientCNPJ string
	var currentValidUntil *time.Time
	err := s.pool.QueryRow(ctx, `
		select p.id::text, v.id::text, v.version_number, v.public_token::text, p.title,
		       c.id::text, coalesce(c.legal_name,''), coalesce(c.cnpj,''), v.pricing_model,
		       v.content, v.conditions, v.minimum_invoice, v.setup_fee, v.content_hash, p.valid_until
		from proposal_versions v
		join proposals p on p.id=v.proposal_id
		join clients c on c.id=p.client_id
		where v.public_token=$1 and v.published_at is not null
		  and v.version_number=p.current_version
		  and p.status in ('published','accepted')
	`, token).Scan(
		&result.ProposalID,
		&result.VersionID,
		&result.VersionNumber,
		&result.PublicToken,
		&currentTitle,
		&result.ClientID,
		&currentClientName,
		&currentClientCNPJ,
		&result.PricingModel,
		&contentJSON,
		&conditionsJSON,
		&result.MinimumInvoice,
		&result.SetupFee,
		&result.ContentHash,
		&currentValidUntil,
	)
	if err != nil {
		return PublicProposal{}, err
	}
	if err := json.Unmarshal(contentJSON, &result.Content); err != nil {
		return PublicProposal{}, fmt.Errorf("decode proposal content: %w", err)
	}
	result.Title=snapshotString(result.Content,"proposal","title",currentTitle)
	result.ClientName=snapshotString(result.Content,"client","legal_name",currentClientName)
	result.ClientTradeName=snapshotString(result.Content,"client","trade_name","")
	result.ClientCNPJ=snapshotString(result.Content,"client","cnpj",currentClientCNPJ)
	result.ValidUntil=currentValidUntil
	if value:=snapshotString(result.Content,"proposal","valid_until","");value!=""{
		if parsed,parseErr:=time.Parse("2006-01-02",value);parseErr==nil{result.ValidUntil=&parsed}
	}
	if result.ValidUntil != nil {
		today:=time.Now().In(time.Local);today=time.Date(today.Year(),today.Month(),today.Day(),0,0,0,0,today.Location())
		if result.ValidUntil.Before(today) { return PublicProposal{}, fmt.Errorf("proposal expired") }
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

func snapshotString(content map[string]any,section,key,fallback string) string {
	group,ok:=content[section].(map[string]any);if !ok{return fallback}
	value,ok:=group[key].(string);if !ok||strings.TrimSpace(value)==""{return fallback}
	return value
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
	var currentVersion int
	err = tx.QueryRow(ctx, `select status::text,current_version from proposals where id=$1 for update`, proposal.ProposalID).Scan(&currentStatus,&currentVersion)
	if err != nil {
		return AcceptanceResult{}, err
	}
	if currentStatus != "published" && currentStatus != "accepted" {
		return AcceptanceResult{}, fmt.Errorf("proposal is not available for acceptance")
	}
	if currentVersion!=proposal.VersionNumber{
		return AcceptanceResult{},fmt.Errorf("proposal version was superseded")
	}

	var existing AcceptanceResult
	var existingName,existingEmail,existingCPF string
	err=tx.QueryRow(ctx,`
		select pa.id::text,o.id::text,pa.accepted_by_name,pa.accepted_by_email::text,pa.accepted_by_cpf
		from proposal_acceptances pa join onboardings o on o.proposal_acceptance_id=pa.id
		where pa.proposal_version_id=$1
	`,proposal.VersionID).Scan(&existing.AcceptanceID,&existing.OnboardingID,&existingName,&existingEmail,&existingCPF)
	if err==nil{
		if strings.EqualFold(strings.TrimSpace(existingEmail),strings.TrimSpace(input.Email))&&existingCPF==input.CPF&&strings.EqualFold(strings.TrimSpace(existingName),strings.TrimSpace(input.Name)){
			if err:=tx.Commit(ctx);err!=nil{return AcceptanceResult{},err}
			return existing,nil
		}
		return AcceptanceResult{},fmt.Errorf("proposal version already accepted by another representative")
	}
	if err!=pgx.ErrNoRows{return AcceptanceResult{},err}

	acceptanceHash:=legaltext.SHA256(legaltext.ProposalAcceptanceText)
	var result AcceptanceResult
	err = tx.QueryRow(ctx, `
		insert into proposal_acceptances(
			proposal_id,proposal_version_id,proposal_hash,
			accepted_by_name,accepted_by_email,accepted_by_cpf,accepted_by_phone,accepted_by_role,
			authority_declared,acceptance_text_version,acceptance_text,acceptance_text_sha256,
			ip_address,user_agent,session_id
		) values ($1,$2,$3,$4,$5,$6,$7,$8,true,$9,$10,$11,$12,$13,$14)
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
		legaltext.ProposalAcceptanceVersion,
		legaltext.ProposalAcceptanceText,
		acceptanceHash,
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
			company_responsible_name,company_responsible_cpf,company_responsible_phone,
			company_responsible_email,company_responsible_role,company_responsible_authority_declared
		) values($1,$2,'pending',nullif($3,''),nullif($4,''),nullif($5,''),$6,$7,$8,$9,$10,true)
		returning id::text
	`,result.AcceptanceID,proposal.ClientID,proposal.ClientCNPJ,proposal.ClientName,proposal.ClientTradeName,input.Name,input.CPF,input.Phone,input.Email,input.Role).Scan(&result.OnboardingID)
	if err != nil {
		return AcceptanceResult{}, fmt.Errorf("create onboarding: %w", err)
	}

	if _, err := tx.Exec(ctx, `update proposals set status='accepted',updated_at=now() where id=$1`, proposal.ProposalID); err != nil {
		return AcceptanceResult{}, err
	}

	if _, err := tx.Exec(ctx, `
		insert into audit_events(actor_type,event_type,resource_type,resource_id,ip_address,user_agent,metadata)
		values ('customer','proposal.accepted','proposal',$1,$2,$3,
		        jsonb_build_object('acceptance_id',$4,'version',$5,'version_hash',$6,'acceptance_text_version',$7,'acceptance_text_sha256',$8))
	`,proposal.ProposalID,nullableIP(input.IPAddress),input.UserAgent,result.AcceptanceID,proposal.VersionNumber,fmt.Sprintf("%x",proposal.ContentHash),legaltext.ProposalAcceptanceVersion,fmt.Sprintf("%x",acceptanceHash)); err != nil {
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
