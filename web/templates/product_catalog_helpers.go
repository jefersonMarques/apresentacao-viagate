package templates

import "strconv"

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

func CatalogSortOrder(value int) string {
	return strconv.Itoa(value)
}
