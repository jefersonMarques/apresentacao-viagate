package proposals

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// PublishedByToken loads the immutable current published snapshot without
// applying customer-journey availability rules such as expiration or status.
// It is intended for internal artifact generation from an already published
// version.
func (s *Store) PublishedByToken(ctx context.Context, token string) (PublicProposal, error) {
	var result PublicProposal
	var contentJSON []byte
	var conditionsJSON []byte
	var currentTitle, currentClientName, currentClientCNPJ string
	var currentValidUntil *time.Time

	err := s.pool.QueryRow(ctx, `
		select p.id::text, v.id::text, v.version_number, v.public_token::text, p.title,p.status::text,
		       c.id::text, coalesce(c.legal_name,''), coalesce(c.cnpj,''), v.pricing_model,
		       v.content, v.conditions, v.minimum_invoice, v.setup_fee, v.content_hash, p.valid_until
		from proposal_versions v
		join proposals p on p.id=v.proposal_id
		join clients c on c.id=p.client_id
		where v.public_token=$1 and v.published_at is not null
		  and v.version_number=p.current_version
	`, token).Scan(
		&result.ProposalID,
		&result.VersionID,
		&result.VersionNumber,
		&result.PublicToken,
		&currentTitle,
		&result.Status,
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

	result.Title = snapshotString(result.Content, "proposal", "title", currentTitle)
	result.ClientName = snapshotString(result.Content, "client", "legal_name", currentClientName)
	result.ClientTradeName = snapshotString(result.Content, "client", "trade_name", "")
	result.ClientCNPJ = snapshotString(result.Content, "client", "cnpj", currentClientCNPJ)
	result.ValidUntil = currentValidUntil
	if value := snapshotString(result.Content, "proposal", "valid_until", ""); value != "" {
		if parsed, parseErr := time.Parse("2006-01-02", value); parseErr == nil {
			result.ValidUntil = &parsed
		}
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
		if err := rows.Scan(&item.GroupName, &item.Label, &item.Unit, &item.Price, &item.IsOptional, &item.SortOrder); err != nil {
			return PublicProposal{}, err
		}
		item.GroupName = strings.TrimSpace(item.GroupName)
		result.Items = append(result.Items, item)
	}
	return result, rows.Err()
}
