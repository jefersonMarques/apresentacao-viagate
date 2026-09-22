package templates

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jefersonMarques/apresentacao-viagate/internal/catalog"
)

func CatalogActiveLabel(active bool) string {
	if active {
		return "ATIVO"
	}
	return "INATIVO"
}

func CatalogActiveClass(active bool) string {
	if active {
		return "active"
	}
	return "inactive"
}

func CatalogLifecycleLabel(active, archived, deleted bool) string {
	if deleted {
		return "EXCLUÍDO"
	}
	if archived {
		return "ARQUIVADO"
	}
	return CatalogActiveLabel(active)
}

func CatalogLifecycleClass(active, archived, deleted bool) string {
	if deleted {
		return "deleted"
	}
	if archived {
		return "archived"
	}
	return CatalogActiveClass(active)
}

func CatalogSortOrder(value int) string {
	return strconv.Itoa(value)
}

func CatalogValueOrDash(value string) string {
	if value == "" {
		return "—"
	}
	return value
}

func CatalogNonDeletedProductCount(category catalog.ManagedCategory) int {
	total := 0
	for _, product := range category.Products {
		if !product.IsDeleted {
			total++
		}
	}
	return total
}

func CatalogDependencyCount(product catalog.ManagedProduct) int {
	return len(product.DependencyGroups)
}

func CatalogDependencyModeName(index int) string {
	return fmt.Sprintf("dependency_mode_%d", index)
}

func CatalogDependencyProductName(index int) string {
	return fmt.Sprintf("dependency_product_%d", index)
}

func CatalogDependencySelected(group catalog.DependencyGroup, productID string) bool {
	for _, required := range group.RequiredProducts {
		if required.ID == productID {
			return true
		}
	}
	return false
}

func CatalogDependencyCandidateDisabled(category catalog.ManagedCategory, product catalog.ManagedProduct, selected bool) bool {
	if selected {
		return false
	}
	return category.IsArchived || category.IsDeleted || !product.IsActive || product.IsArchived || product.IsDeleted
}

func CatalogDependencySummary(groups []catalog.DependencyGroup) string {
	if len(groups) == 0 {
		return ""
	}
	parts := make([]string, 0, len(groups))
	for _, group := range groups {
		names := make([]string, 0, len(group.RequiredProducts))
		for _, required := range group.RequiredProducts {
			names = append(names, required.Name)
		}
		if len(names) == 0 {
			continue
		}
		connector := " e "
		if group.MatchMode == "any" {
			connector = " ou "
		}
		parts = append(parts, "("+strings.Join(names, connector)+")")
	}
	return strings.Join(parts, " e ")
}

func CatalogDependencyStatusSuffix(product catalog.ManagedProduct) string {
	switch {
	case product.IsDeleted:
		return " · excluído"
	case product.IsArchived:
		return " · arquivado"
	case !product.IsActive:
		return " · inativo"
	default:
		return ""
	}
}


func CatalogHasWritableCategory(categories []catalog.ManagedCategory) bool {
	for _, category := range categories {
		if !category.IsDeleted && !category.IsArchived {
			return true
		}
	}
	return false
}
