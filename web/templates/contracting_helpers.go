package templates

import (
	"strings"

	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
	"github.com/jefersonMarques/apresentacao-viagate/internal/platform/brfields"
)

func ContractingPolicyClass(hasPolicy bool) string {
	if hasPolicy {
		return "is-complete"
	}
	return "is-pending"
}

func ContractingPolicyTitle(hasPolicy bool) string {
	if hasPolicy {
		return "Apólice recebida"
	}
	return "Envie a apólice de seguros"
}

func ContractingPolicyDescription(hasPolicy bool) string {
	if hasPolicy {
		return "O arquivo já está vinculado à contratação. Você pode substituí-lo antes do envio final."
	}
	return "PDF, JPG ou PNG · máximo 15 MB."
}

func ContractingPolicyButton(hasPolicy bool) string {
	if hasPolicy {
		return "Substituir apólice"
	}
	return "Enviar apólice"
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
