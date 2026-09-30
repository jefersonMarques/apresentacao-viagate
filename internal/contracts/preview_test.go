package contracts

import (
	"strings"
	"testing"
)

func TestValidateTemplateBooleanUsageRequiresConditionals(t *testing.T) {
	if err := validateTemplateBooleanUsage("Cargo Score: {products.cargo_score}"); err == nil {
		t.Fatal("direct boolean interpolation must be rejected")
	}
	if err := validateTemplateBooleanUsage("{% if products.cargo_score %}Cargo Score{% endif %}"); err != nil {
		t.Fatalf("conditional boolean usage should be valid: %v", err)
	}
}

func TestMinimalValidationDataHidesOptionalValues(t *testing.T) {
	sets := contractValidationDataSets("ViaGate", "39.406.000/0001-21")
	if len(sets) != 2 {
		t.Fatalf("expected full and minimal validation sets, got %d", len(sets))
	}
	minimal := sets[1]
	for _, path := range []string{
		"client.trade_name",
		"representative.role",
		"insurance.broker_company",
		"insurance.broker_producer",
		"proposal.valid_until",
	} {
		if value := resolve(minimal, path); strings.TrimSpace(value.(string)) != "" {
			t.Fatalf("%s should be empty in minimal validation data: %v", path, value)
		}
	}
	if resolve(minimal, "proposal.setup_fee") != "ISENTO" {
		t.Fatalf("minimal setup fee must exercise exemption state")
	}
}
