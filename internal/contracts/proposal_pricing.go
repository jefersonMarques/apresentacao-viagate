package contracts

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jefersonMarques/apresentacao-viagate/internal/catalog"
)

type proposalPricingItem struct {
	CatalogID  string
	GroupName  string
	Label      string
	Unit       string
	Price      float64
	IsOptional bool
}

type proposalPricingData struct {
	Products     map[string]any
	Prices       map[string]any
	PricingTable RawMarkdown
}

func loadProposalPricing(ctx context.Context, pool *pgxpool.Pool, proposalVersionID string) (proposalPricingData, error) {
	rows, err := pool.Query(ctx, `
		select coalesce(metadata->>'catalog_id',''),group_name,label,coalesce(unit,''),price,is_optional
		from proposal_items
		where proposal_version_id=$1
		order by sort_order,id
	`, proposalVersionID)
	if err != nil {
		return proposalPricingData{}, fmt.Errorf("load proposal pricing: %w", err)
	}
	defer rows.Close()

	items := make([]proposalPricingItem, 0)
	for rows.Next() {
		var item proposalPricingItem
		if err := rows.Scan(&item.CatalogID, &item.GroupName, &item.Label, &item.Unit, &item.Price, &item.IsOptional); err != nil {
			return proposalPricingData{}, fmt.Errorf("scan proposal pricing: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return proposalPricingData{}, fmt.Errorf("iterate proposal pricing: %w", err)
	}

	return buildProposalPricingData(items), nil
}

func buildProposalPricingData(items []proposalPricingItem) proposalPricingData {
	products := map[string]any{
		"cargo_score": false,
		"cargo_truck": false,
		"prevention":  false,
		"monitoring":  false,
	}
	prices := make(map[string]any)

	for _, group := range catalog.Groups {
		for _, item := range group.Items {
			prices[pricingVariableKey(item.ID)] = ""
		}
	}

	for _, item := range items {
		groupID := proposalItemGroupID(item)
		switch groupID {
		case "score":
			products["cargo_score"] = true
		case "logistics":
			products["cargo_truck"] = true
		case "prevention":
			products["prevention"] = true
		case "monitoring":
			products["monitoring"] = true
		}

		if item.CatalogID != "" {
			prices[pricingVariableKey(item.CatalogID)] = formatBRL(item.Price)
		}
	}

	return proposalPricingData{
		Products:     products,
		Prices:       prices,
		PricingTable: RawMarkdown(buildPricingTable(items)),
	}
}

func proposalItemGroupID(item proposalPricingItem) string {
	if item.CatalogID != "" {
		if group, _, ok := catalog.ItemByID(item.CatalogID); ok {
			return group.ID
		}
	}

	group := strings.ToLower(strings.TrimSpace(item.GroupName))
	switch {
	case strings.Contains(group, "score"):
		return "score"
	case strings.Contains(group, "logística"), strings.Contains(group, "logistica"), strings.Contains(group, "truck"):
		return "logistics"
	case strings.Contains(group, "preven"):
		return "prevention"
	case strings.Contains(group, "monitoramento"):
		return "monitoring"
	default:
		return ""
	}
}

func buildPricingTable(items []proposalPricingItem) string {
	if len(items) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("| Grupo | Serviço | Unidade | Condição | Valor |\n")
	builder.WriteString("|---|---|---|---|---:|\n")
	for _, item := range items {
		status := "Proposto"
		if item.IsOptional {
			status = "Opcional"
		}
		builder.WriteString("| ")
		builder.WriteString(escapeMarkdownTableCell(item.GroupName))
		builder.WriteString(" | ")
		builder.WriteString(escapeMarkdownTableCell(item.Label))
		builder.WriteString(" | ")
		builder.WriteString(escapeMarkdownTableCell(item.Unit))
		builder.WriteString(" | ")
		builder.WriteString(status)
		builder.WriteString(" | ")
		builder.WriteString(formatBRL(item.Price))
		builder.WriteString(" |\n")
	}
	return strings.TrimSpace(builder.String())
}

func pricingVariableKey(catalogID string) string {
	return strings.ReplaceAll(strings.TrimSpace(catalogID), "-", "_")
}

func pricingVariableNames() []string {
	variables := make([]string, 0)
	for _, group := range catalog.Groups {
		for _, item := range group.Items {
			variables = append(variables, "pricing."+pricingVariableKey(item.ID))
		}
	}
	return variables
}

func formatBRL(value float64) string {
	negative := value < 0
	cents := int64(math.Round(math.Abs(value) * 100))
	integerPart := cents / 100
	decimalPart := cents % 100

	digits := fmt.Sprintf("%d", integerPart)
	if len(digits) > 3 {
		var groups []string
		for len(digits) > 3 {
			index := len(digits) - 3
			groups = append([]string{digits[index:]}, groups...)
			digits = digits[:index]
		}
		groups = append([]string{digits}, groups...)
		digits = strings.Join(groups, ".")
	}

	prefix := "R$ "
	if negative {
		prefix = "-R$ "
	}
	return fmt.Sprintf("%s%s,%02d", prefix, digits, decimalPart)
}

func escapeMarkdownTableCell(value string) string {
	value = strings.Map(func(r rune) rune {
		switch r {
		case '\r', '\n', '\t':
			return ' '
		default:
			if r < 32 || r == 127 {
				return -1
			}
			return r
		}
	}, value)
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "|", "\\|")
	return strings.TrimSpace(value)
}
