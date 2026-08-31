package proposals

import (
	"context"
	"encoding/json"
)

// DuplicateSourceByID returns the reusable commercial content of a proposal.
// Accepted proposals are always copied from the immutable version that was accepted.
func (s *Store) DuplicateSourceByID(ctx context.Context, userID, proposalID string, allowAll bool) (EditorInput, error) {
	input, _, err := s.EditorByID(ctx, userID, proposalID, allowAll)
	if err != nil {
		return EditorInput{}, err
	}

	var status string
	var currentVersion int
	if err := s.pool.QueryRow(ctx, `
		select status::text,current_version
		from proposals
		where id=$1
	`, proposalID).Scan(&status, &currentVersion); err != nil {
		return EditorInput{}, err
	}
	if status != "accepted" || currentVersion <= 0 {
		return input, nil
	}

	var versionID string
	var contentJSON, conditionsJSON []byte
	if err := s.pool.QueryRow(ctx, `
		select id::text,pricing_model,minimum_invoice,setup_fee,content,conditions,content_hash
		from proposal_versions
		where proposal_id=$1 and version_number=$2 and published_at is not null
		limit 1
	`, proposalID, currentVersion).Scan(
		&versionID,
		&input.PricingModel,
		&input.MinimumInvoice,
		&input.SetupFee,
		&contentJSON,
		&conditionsJSON,
		&input.ContentHash,
	); err != nil {
		return EditorInput{}, err
	}

	input.Content = nil
	input.Conditions = nil
	input.Items = nil
	input.OperationContext = ""
	input.CustomerPriorities = nil
	input.SolutionTitle = ""
	input.SolutionScope = nil

	if len(contentJSON) > 0 {
		_ = json.Unmarshal(contentJSON, &input.Content)
		input.ClientLogoURL = contentString(input.Content, "client", "logo_url")
		input.ContactName = contentString(input.Content, "contact", "name")
		input.ContactRole = contentString(input.Content, "contact", "role")
		input.ContactEmail = contentString(input.Content, "contact", "email")
		input.ContactPhone = contentString(input.Content, "contact", "phone")
		input.OperationContext = contentRootString(input.Content, "operation_context")
		input.CustomerPriorities = contentRootStrings(input.Content, "customer_priorities")
		input.SolutionTitle = contentRootString(input.Content, "solution_title")
		input.SolutionScope = contentRootStrings(input.Content, "solution_scope")
	}
	if len(conditionsJSON) > 0 {
		_ = json.Unmarshal(conditionsJSON, &input.Conditions)
	}

	rows, err := s.pool.Query(ctx, `
		select coalesce(metadata->>'catalog_id',''),group_name,label,coalesce(unit,''),price,is_optional,sort_order
		from proposal_items
		where proposal_version_id=$1
		order by sort_order,id
	`, versionID)
	if err != nil {
		return EditorInput{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var item EditorItem
		if err := rows.Scan(&item.CatalogID, &item.GroupName, &item.Label, &item.Unit, &item.Price, &item.IsOptional, &item.SortOrder); err != nil {
			return EditorInput{}, err
		}
		input.Items = append(input.Items, item)
	}
	if err := rows.Err(); err != nil {
		return EditorInput{}, err
	}
	return input, nil
}
