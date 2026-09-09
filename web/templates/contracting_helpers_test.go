package templates

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
)

func TestContractingJourneyRequiresExplicitOperationType(t *testing.T) {
	var output bytes.Buffer
	component := ContractingJourneyPage(domain.Onboarding{ID: "onboarding-test"}, false, "", "", "")
	if err := component.Render(context.Background(), &output); err != nil {
		t.Fatalf("render contracting journey: %v", err)
	}

	html := output.String()
	if !strings.Contains(html, "name=\"operation_type\" required") {
		t.Fatal("operation type select must be required")
	}
	if !strings.Contains(html, "value=\"\" disabled selected") {
		t.Fatal("empty operation type must render an explicit selected placeholder")
	}
	if strings.Contains(html, "value=\"normal\" selected") {
		t.Fatal("normal operation must not be selected when the persisted value is empty")
	}
}

func TestContractingOperationTypeLabel(t *testing.T) {
	tests := map[string]string{
		"normal": "Normal",
		"avulsa": "Avulsa",
		"":       "Não informado",
	}
	for value, want := range tests {
		if got := ContractingOperationTypeLabel(value); got != want {
			t.Fatalf("ContractingOperationTypeLabel(%q) = %q, want %q", value, got, want)
		}
	}
}

func TestContractingReadyForReview(t *testing.T) {
	complete := domain.Onboarding{
		OperationType:    "normal",
		Insurer:          "Seguradora Teste",
		PolicyStartDate:  "2026-09-01",
		PolicyEndDate:    "2027-09-01",
	}

	if !ContractingReadyForReview(complete, true) {
		t.Fatal("complete persisted insurance data with policy should be ready for review")
	}

	complete.OperationType = ""
	if ContractingReadyForReview(complete, true) {
		t.Fatal("missing operation type must block review")
	}
}
