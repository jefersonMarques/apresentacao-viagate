package httpapp

import (
	"context"
	"fmt"

	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
	"github.com/jefersonMarques/apresentacao-viagate/internal/proposals"
)

const proposalContractTemplateOptionsKey = "__ui_contract_template_options"

func (a *App) decorateProposalContractOptions(ctx context.Context, input proposals.EditorInput) proposals.EditorInput {
	items, err := a.proposalContractTemplates(ctx)
	if err != nil {
		a.logger.Error("load proposal contract templates failed", "error", err)
		return input
	}
	if input.Content == nil {
		input.Content = map[string]any{}
	}
	input.Content[proposalContractTemplateOptionsKey] = items
	proposal, _ := input.Content["proposal"].(map[string]any)
	if proposal == nil {
		proposal = map[string]any{}
		input.Content["proposal"] = proposal
	}
	if proposal["contract_template_id"] == nil || proposal["contract_template_id"] == "" {
		for _, item := range items {
			if item.IsDefault {
				proposal["contract_template_id"] = item.ID
				break
			}
		}
	}
	return input
}

func (a *App) proposalContractTemplates(ctx context.Context) ([]domain.ContractTemplate, error) {
	rows, err := a.pool.Query(ctx, `
		select id::text,name,coalesce(description,''),current_version,is_active,is_default
		from contract_templates
		where is_active=true and current_version>0
		order by is_default desc,name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.ContractTemplate
	for rows.Next() {
		var item domain.ContractTemplate
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.CurrentVersion, &item.IsActive, &item.IsDefault); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (a *App) resolveProposalContractTemplateVersion(ctx context.Context, templateID string) (string, error) {
	if templateID == "" {
		return "", nil
	}
	var versionID string
	err := a.pool.QueryRow(ctx, `
		select v.id::text
		from contract_templates t
		join contract_template_versions v on v.contract_template_id=t.id and v.version_number=t.current_version
		where t.id=$1 and t.is_active=true and t.current_version>0
	`, templateID).Scan(&versionID)
	if err != nil {
		return "", fmt.Errorf("modelo de contrato inválido ou sem versão ativa")
	}
	return versionID, nil
}

func (a *App) persistProposalContractAssignment(ctx context.Context, draft proposals.SavedDraft, input proposals.EditorInput) error {
	templateID := proposalContentString(input.Content, "proposal", "contract_template_id")
	versionID := proposalContentString(input.Content, "proposal", "contract_template_version_id")
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		update proposals
		set contract_template_id=nullif($2,'')::uuid
		where id=$1
	`, draft.ProposalID, templateID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		update proposal_versions
		set contract_template_version_id=nullif($2,'')::uuid
		where id=$1 and published_at is null
	`, draft.VersionID, versionID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
