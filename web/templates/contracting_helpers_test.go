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
	component := ContractingJourneyPage(domain.Onboarding{ID: "onboarding-test", RequiresPolicy: true}, "", "", "")
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
		RequiresPolicy:   true,
		OperationType:    "normal",
		Insurer:          "Seguradora Teste",
		PolicyStartDate:  "2026-09-01",
		PolicyEndDate:    "2027-09-01",
	}

	if !ContractingReadyForReview(complete) {
		t.Fatal("complete persisted contract data should be ready for review without a policy upload")
	}

	complete.OperationType = ""
	if ContractingReadyForReview(complete) {
		t.Fatal("missing operation type must block review")
	}
}


func TestContractingReadyForReviewSkipsInsuranceWhenNotRequired(t *testing.T) {
	onboarding := domain.Onboarding{RequiresPolicy: false}
	if !ContractingReadyForReview(onboarding) {
		t.Fatal("insurance must not block review when proposal does not require policy")
	}

	var output bytes.Buffer
	component := ContractingJourneyPage(domain.Onboarding{ID: "onboarding-test", RequiresPolicy: false}, "", "", "")
	if err := component.Render(context.Background(), &output); err != nil {
		t.Fatalf("render contracting journey: %v", err)
	}
	html := output.String()
	if strings.Contains(html, `data-contracting-step="insurance"`) || strings.Contains(html, `name="insurer"`) {
		t.Fatal("insurance step must not render when proposal does not require policy")
	}
}
