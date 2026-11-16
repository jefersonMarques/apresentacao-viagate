package contracts

import "testing"

func TestContractInsuranceRequirements(t *testing.T) {
	if !templateRequiresInsuranceDetails("Seguradora: {insurance.insurer}") {
		t.Fatal("old contract must be identified")
	}
	if !templateRequiresInsuranceDetails("Operação: {operation.type}") {
		t.Fatal("old operation clause must be identified")
	}
	if templateRequiresInsuranceDetails("Dados operacionais coletados no pós-venda.") {
		t.Fatal("revised contract should not require insurance values")
	}
	if contractInsuranceDetailsComplete("normal", "", "2026-11-01", "2027-11-01") {
		t.Fatal("missing insurer cannot pass the old template gate")
	}
	if !contractInsuranceDetailsComplete("avulsa", "Insurance Company", "2026-11-01", "2027-11-01") {
		t.Fatal("complete legacy insurance details must remain compatible")
	}
}
