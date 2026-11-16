package templates

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jefersonMarques/apresentacao-viagate/internal/activation"
)

func TestActivationPageDisablesEmptyGoodsAndUsersActions(t *testing.T) {
	access := activation.Access{
		Section: "all",
		Profile: activation.Profile{
			ID:             "activation-test",
			Status:         "in_progress",
			RequiresPolicy: true,
		},
	}

	var output bytes.Buffer
	if err := ActivationPage(access, "token-test", "", "").Render(context.Background(), &output); err != nil {
		t.Fatalf("render activation page: %v", err)
	}

	html := output.String()
	for _, section := range []string{"goods", "users"} {
		needle := `data-activation-section-save="` + section + `"`
		index := strings.Index(html, needle)
		if index < 0 {
			t.Fatalf("save button for %s not rendered", section)
		}
		windowEnd := min(len(html), index+240)
		if !strings.Contains(html[index:windowEnd], "disabled") {
			t.Fatalf("save button for %s must be disabled without persisted items", section)
		}
	}
}

func TestActivationCompleteRequiresGoodsAndUsers(t *testing.T) {
	profile := activation.Profile{
		HasPolicy:               true,
		RequiresPolicy:          true,
		OperationType:           "normal",
		Insurer:                 "Insurer",
		PolicyStartDate:         "2026-11-01",
		PolicyEndDate:           "2027-11-01",
		FinanceResponsibleName:  "Financeiro",
		FinanceResponsibleEmail: "financeiro@example.com",
		FinanceResponsiblePhone: "11999999999",
	}
	if ActivationComplete(profile) {
		t.Fatal("activation must not be complete without goods and users")
	}

	profile.Goods = []string{"Queijos"}
	if ActivationComplete(profile) {
		t.Fatal("activation must not be complete without a system user")
	}

	profile.SystemUsers = []activation.SystemUser{{Name: "Maria", Email: "maria@example.com"}}
	if !ActivationComplete(profile) {
		t.Fatal("activation should be complete with finance, one good and one user")
	}
}


func TestActivationCompleteRequiresPolicy(t *testing.T) {
	profile := activation.Profile{
		RequiresPolicy:          true,
		OperationType:           "normal",
		Insurer:                 "Insurer",
		PolicyStartDate:         "2026-11-01",
		PolicyEndDate:           "2027-11-01",
		FinanceResponsibleName:  "Financeiro",
		FinanceResponsibleEmail: "financeiro@example.com",
		FinanceResponsiblePhone: "11999999999",
		Goods:                   []string{"Queijos"},
		SystemUsers:             []activation.SystemUser{{Name: "Maria", Email: "maria@example.com"}},
	}
	if ActivationComplete(profile) {
		t.Fatal("activation must not be complete without an insurance policy")
	}

	profile.HasPolicy = true
	if !ActivationComplete(profile) {
		t.Fatal("activation should be complete when policy and operational data are present")
	}
	profile.Insurer = ""
	if ActivationComplete(profile) {
		t.Fatal("activation must require insurance operation details")
	}
}

func TestActivationContractSigned(t *testing.T) {
	now := time.Now()
	if ActivationContractSigned(activation.Profile{ContractStatus: "sent"}) {
		t.Fatal("sent contract must not unlock internal implementation")
	}
	if !ActivationContractSigned(activation.Profile{ContractStatus: "signed", FullySignedAt: &now}) {
		t.Fatal("fully signed contract should unlock internal implementation")
	}
}


func TestActivationCompleteSkipsPolicyAndGoodsWhenNotRequired(t *testing.T) {
	profile := activation.Profile{
		RequiresPolicy:          false,
		FinanceResponsibleName:  "Financeiro",
		FinanceResponsibleEmail: "financeiro@example.com",
		FinanceResponsiblePhone: "11999999999",
		SystemUsers:             []activation.SystemUser{{Name: "Maria", Email: "maria@example.com"}},
	}
	if !ActivationComplete(profile) {
		t.Fatal("activation should be complete without policy and goods when proposal does not require them")
	}
}

func TestActivationPageHidesPolicyAndGoodsWhenNotRequired(t *testing.T) {
	access := activation.Access{
		Section: "all",
		Profile: activation.Profile{
			ID:     "activation-test",
			Status: "in_progress",
		},
	}
	var output bytes.Buffer
	if err := ActivationPage(access, "token-test", "", "").Render(context.Background(), &output); err != nil {
		t.Fatalf("render activation page: %v", err)
	}
	html := output.String()
	if strings.Contains(html, `id="apolice"`) || strings.Contains(html, `id="mercadorias"`) {
		t.Fatal("policy and goods sections must be hidden when not required")
	}
}

func TestActivationStepsMatchInsuranceRequirements(t *testing.T) {
    insured := activation.Access{
        Section: "all",
        Profile: activation.Profile{RequiresPolicy: true},
    }
    insuredSteps := ActivationSteps(insured)
    expected := []string{"operation", "policy", "finance", "goods", "users", "review"}
    if len(insuredSteps) != len(expected) {
        t.Fatalf("insured steps = %d, want %d", len(insuredSteps), len(expected))
    }
    for index, key := range expected {
        if insuredSteps[index].Key != key {
            t.Errorf("step %d = %q, want %q", index, insuredSteps[index].Key, key)
        }
    }

    regular := activation.Access{Section: "all"}
    regularSteps := ActivationSteps(regular)
    expectedRegular := []string{"finance", "users", "review"}
    if len(regularSteps) != len(expectedRegular) {
        t.Fatalf("regular steps = %d, want %d", len(regularSteps), len(expectedRegular))
    }
    for index, key := range expectedRegular {
        if regularSteps[index].Key != key {
            t.Errorf("regular step %d = %q, want %q", index, regularSteps[index].Key, key)
        }
    }
}

func TestActivationPageLimitsDelegateToAuthorizedSection(t *testing.T) {
    access := activation.Access{
        Section: "finance",
        AccessType: "delegate",
        Profile: activation.Profile{
            LegalName: "Test Company",
            CompanyResponsibleEmail: "signer@example.com",
            Status: "in_progress",
            RequiresPolicy: true,
            Insurer: "Example Insurer",
            Goods: []string{"Restricted Good"},
        },
    }

    var output bytes.Buffer
    if err := ActivationPage(access, "token-test", "", "").Render(context.Background(), &output); err != nil {
        t.Fatalf("render scoped activation: %v", err)
    }
    html := output.String()
    for _, forbidden := range []string{
        `data-activation-step="operation"`,
        `data-activation-step="policy"`,
        `data-activation-step="goods"`,
        `data-activation-step="users"`,
        `data-activation-step="review"`,
        "Restricted Good",
        "Example Insurer",
        "signer@example.com",
        "/activation/token-test/proposal",
        "/activation/token-test/submit",
    } {
        if strings.Contains(html, forbidden) {
            t.Errorf("scoped activation exposed forbidden content %q", forbidden)
        }
    }
    if !strings.Contains(html, `data-activation-step="finance"`) {
        t.Fatal("finance delegate must access the finance step")
    }
}

func TestActivationReviewContainsEditActionsAndConfirmation(t *testing.T) {
    access := activation.Access{
        Section: "all",
        Profile: activation.Profile{
            Status: "in_progress",
            RequiresPolicy: true,
        },
    }
    var output bytes.Buffer
    if err := ActivationPage(access, "token-test", "", "").Render(context.Background(), &output); err != nil {
        t.Fatalf("render review: %v", err)
    }
    html := output.String()
    for _, required := range []string{
        `data-activation-step="review"`,
        `data-activation-edit="operation"`,
        `data-activation-edit="policy"`,
        `data-activation-edit="finance"`,
        `data-activation-edit="goods"`,
        `data-activation-edit="users"`,
        "Confirmar dados da implantação",
    } {
        if !strings.Contains(html, required) {
            t.Errorf("review missing action %q", required)
        }
    }
    if strings.Contains(html, `/activation/token-test/proposal`) {
        t.Fatal("review must not link to the proposal")
    }
}

func TestActivationPageShowsSuccessAfterConfirmation(t *testing.T) {
    access := activation.Access{
        Section: "all",
        Profile: activation.Profile{Status: "completed"},
    }
    var output bytes.Buffer
    if err := ActivationPage(access, "token-test", "", "").Render(context.Background(), &output); err != nil {
        t.Fatalf("render completed activation: %v", err)
    }
    html := output.String()
    if !strings.Contains(html, "Dados enviados com sucesso!") {
        t.Fatal("completed activation must show success message")
    }
    if strings.Contains(html, `data-activation-step="review"`) ||
        strings.Contains(html, `/activation/token-test/submit`) {
        t.Fatal("completed activation must not show a new submit action")
    }
}

func TestActivationDateLabel(t *testing.T) {
    if actual := ActivationDateLabel("2026-11-16"); actual != "16/11/2026" {
        t.Fatalf("formatted policy date = %q", actual)
    }
    if actual := ActivationDateLabel("invalid"); actual != "invalid" {
        t.Fatalf("invalid date must not be silently rewritten: %q", actual)
    }
}
