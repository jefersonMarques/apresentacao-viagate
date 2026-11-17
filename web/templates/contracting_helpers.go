package templates

import (
	"strings"

	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/brfields"
)

func ContractingOperationTypeLabel(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "normal":
		return "Normal"
	case "avulsa":
		return "Avulsa"
	default:
		return "Não informado"
	}
}

func ContractingReadyForReview(onboarding domain.Onboarding) bool {
	return len(strings.TrimSpace(onboarding.CNPJ)) == 14 &&
		strings.TrimSpace(onboarding.LegalName) != "" &&
		strings.TrimSpace(onboarding.PostalCode) != "" &&
		strings.TrimSpace(onboarding.Street) != "" &&
		strings.TrimSpace(onboarding.StreetNumber) != "" &&
		strings.TrimSpace(onboarding.City) != "" &&
		len(strings.TrimSpace(onboarding.State)) == 2
}

func ContractingAddress(onboarding domain.Onboarding) string {
	parts := []string{}
	street := strings.TrimSpace(onboarding.Street)
	if street != "" {
		if number := strings.TrimSpace(onboarding.StreetNumber); number != "" {
			street += ", " + number
		}
		parts = append(parts, street)
	}
	if district := strings.TrimSpace(onboarding.District); district != "" {
		parts = append(parts, district)
	}
	cityState := strings.TrimSpace(onboarding.City)
	if state := strings.TrimSpace(onboarding.State); state != "" {
		if cityState != "" {
			cityState += "/"
		}
		cityState += state
	}
	if cityState != "" {
		parts = append(parts, cityState)
	}
	if postalCode := brfields.FormatPostalCode(onboarding.PostalCode); postalCode != "" {
		parts = append(parts, postalCode)
	}
	if len(parts) == 0 {
		return "Ainda não informado"
	}
	return strings.Join(parts, " · ")
}
