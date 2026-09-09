package templates

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/jefersonMarques/apresentacao-viagate/internal/activation"
)

func TestActivationPageDisablesEmptyGoodsAndUsersActions(t *testing.T) {
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
