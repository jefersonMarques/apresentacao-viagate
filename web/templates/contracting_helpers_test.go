package templates

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/jefersonMarques/apresentacao-viagate/internal/domain"
)

func TestContractingJourneyDoesNotRequestInsurance(t *testing.T) {
	for _, requiresPolicy := range []bool{true,false} {
		var output bytes.Buffer
		err := ContractingJourneyPage(domain.Onboarding{
			ID:"onboarding-test",RequiresPolicy:requiresPolicy,
		},"","","").Render(context.Background(),&output)
		if err!=nil{
			t.Fatalf("render contracting journey: %v",err)
		}
		markup:=output.String()
		for _,forbidden:=range []string{
			`data-contracting-step="insurance"`,
			`name="operation_type"`,
			`name="insurer"`,
			`name="policy_start_date"`,
			`name="policy_end_date"`,
		}{
			if strings.Contains(markup,forbidden){
				t.Errorf("contracting journey must not request %s",forbidden)
			}
		}
	}
}

func TestContractingReadyForReviewRequiresCompanyOnly(t *testing.T) {
	company:=domain.Onboarding{
		CNPJ:"39406000000121",LegalName:"Example Company",PostalCode:"82510000",
		Street:"Main Street",StreetNumber:"100",City:"Curitiba",State:"PR",RequiresPolicy:true,
	}
	if !ContractingReadyForReview(company){
		t.Fatal("company should be ready without insurance details")
	}
	company.CNPJ=""
	if ContractingReadyForReview(company){
		t.Fatal("company without CNPJ must not pass")
	}
}

func TestContractingOperationTypeLabel(t *testing.T) {
	for value,want:=range map[string]string{"normal":"Normal","avulsa":"Avulsa","":"Não informado"}{
		if got:=ContractingOperationTypeLabel(value);got!=want{
			t.Errorf("operation label=%q, want %q",got,want)
		}
	}
}
