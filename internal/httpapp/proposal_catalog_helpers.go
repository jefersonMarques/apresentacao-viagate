package httpapp

import (
	"context"

	"github.com/jefersonMarques/apresentacao-viagate/internal/proposals"
)

func (a *App) decorateProposalEditor(ctx context.Context, input proposals.EditorInput) proposals.EditorInput {
	input = a.decorateProposalContractOptions(ctx, input)
	return a.decorateProposalCatalog(ctx, input)
}

func (a *App) decorateProposalCatalog(ctx context.Context, input proposals.EditorInput) proposals.EditorInput {
	selectedCodes := make([]string, 0, len(input.Items))
	for _, item := range input.Items {
		if item.CatalogID != "" {
			selectedCodes = append(selectedCodes, item.CatalogID)
		}
	}

	categories, err := a.catalogStore.ListForProposal(ctx, selectedCodes)
	if err != nil {
		a.logger.Error("load proposal product catalog failed", "proposal_id", input.ProposalID, "error", err)
		categories = nil
	}
	if input.Content == nil {
		input.Content = map[string]any{}
	}
	if len(input.SelectedCategoryCodes) == 0 {
		selectedProducts := map[string]bool{}
		for _, item := range input.Items {
			selectedProducts[item.CatalogID] = true
		}
		for _, category := range categories {
			for _, product := range category.Products {
				if selectedProducts[product.Code] {
					input.SelectedCategoryCodes = append(input.SelectedCategoryCodes, category.Code)
					break
				}
			}
		}
	}
	input.Content["__ui_product_catalog"] = categories
	return input
}
