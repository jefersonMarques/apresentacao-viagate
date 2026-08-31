package contracts

import (
	"strings"
	"testing"
)

func TestBuildProposalPricingDataUsesProposalSnapshot(t *testing.T) {
	items := []proposalPricingItem{
		{
			CatalogID:  "score-item-driver-register",
			GroupName:  "Cargo Score | Análise cadastral",
			Label:      "Cadastro | Motorista — Frota, agregado e terceiro",
			Unit:       "cadastro",
			Price:      25,
			IsOptional: false,
		},
		{
			CatalogID:  "monitoring-trip",
			GroupName:  "Monitoramento de veículos | Integração com gerenciadora",
			Label:      "Viagem Avulsa",
			Unit:       "viagem",
			Price:      1234.5,
			IsOptional: true,
		},
	}

	data := buildProposalPricingData(items)

	if data.Products["cargo_score"] != true {
		t.Fatal("cargo score must be enabled when the proposal contains a score item")
	}
	if data.Products["monitoring"] != true {
		t.Fatal("monitoring must be enabled even when the proposal item is optional")
	}
	if data.Prices["score_item_driver_register"] != "R$ 25,00" {
		t.Fatalf("unexpected score price: %v", data.Prices["score_item_driver_register"])
	}
	if data.Prices["monitoring_trip"] != "R$ 1.234,50" {
		t.Fatalf("unexpected monitoring price: %v", data.Prices["monitoring_trip"])
	}
	if !strings.Contains(string(data.PricingTable), "Opcional | R$ 1.234,50") {
		t.Fatalf("pricing table must preserve optional status and proposal price: %s", data.PricingTable)
	}
}

func TestFormatBRL(t *testing.T) {
	cases := map[float64]string{
		0:        "R$ 0,00",
		4:        "R$ 4,00",
		25.5:     "R$ 25,50",
		1234.56:  "R$ 1.234,56",
		1234567:  "R$ 1.234.567,00",
		-1234.56: "-R$ 1.234,56",
	}
	for value, expected := range cases {
		if actual := formatBRL(value); actual != expected {
			t.Fatalf("formatBRL(%v) = %q, expected %q", value, actual, expected)
		}
	}
}
