package httpapp

import (
	"testing"

	"github.com/jefersonMarques/apresentacao-viagate/internal/proposals"
)

func TestParseMoney(t *testing.T){
	tests:=[]struct{
		name string
		input string
		want float64
		wantErr bool
	}{
		{name:"empty",input:"",want:0},
		{name:"decimal point",input:"1234.56",want:1234.56},
		{name:"decimal comma",input:"1234,56",want:1234.56},
		{name:"brazilian thousands",input:"1.234,56",want:1234.56},
		{name:"formatted BRL",input:"R$ 1.234,56",want:1234.56},
		{name:"formatted BRL zero",input:"R$ 0,00",want:0},
		{name:"negative",input:"-1",wantErr:true},
		{name:"invalid",input:"abc",wantErr:true},
		{name:"nan",input:"NaN",wantErr:true},
		{name:"infinity",input:"Inf",wantErr:true},
	}
	for _,test:=range tests{
		t.Run(test.name,func(t *testing.T){
			got,err:=parseMoney(test.input)
			if test.wantErr{
				if err==nil{t.Fatalf("expected error, got value %v",got)}
				return
			}
			if err!=nil{t.Fatalf("unexpected error: %v",err)}
			if got!=test.want{t.Fatalf("got %v, want %v",got,test.want)}
		})
	}
}

func TestNormalizedConditionsRemovesBlankAndDuplicateValues(t *testing.T){
	got:=normalizedConditions([]string{"Condição A","Condição A","  Condição B  "},"\nCondição B\nCondição C\n")
	want:=[]string{"Condição A","Condição B","Condição C"}
	if len(got)!=len(want){t.Fatalf("got %d conditions, want %d: %#v",len(got),len(want),got)}
	for index:=range want{
		if got[index]!=want[index]{t.Fatalf("condition %d = %q, want %q",index,got[index],want[index])}
	}
}

func TestClearProposalPricesPreservesComposition(t *testing.T) {
	input := proposals.EditorInput{
		MinimumInvoice: 1500,
		SetupFee:       800,
		Items: []proposals.EditorItem{
			{CatalogID: "score-item-driver-register", Price: 12.5, IsOptional: false},
			{CatalogID: "auth-antt", Price: 4.9, IsOptional: true},
		},
	}

	clearProposalPrices(&input)

	if input.MinimumInvoice != 0 || input.SetupFee != 0 {
		t.Fatalf("protected proposal totals were not cleared: minimum=%v setup=%v", input.MinimumInvoice, input.SetupFee)
	}
	if len(input.Items) != 2 {
		t.Fatalf("product composition should be preserved, got %d items", len(input.Items))
	}
	for _, item := range input.Items {
		if item.Price != 0 {
			t.Fatalf("item %s price should be cleared, got %v", item.CatalogID, item.Price)
		}
	}
	if !input.Items[1].IsOptional {
		t.Fatal("optional product state should be preserved")
	}
}

func TestValidateProposalForPublishRequiresPricedProduct(t *testing.T) {
	base := proposals.EditorInput{
		ClientLegalName: "Cliente Teste Ltda",
		Content: map[string]any{
			"proposal": map[string]any{"contract_template_version_id": "template-version"},
		},
		Items: []proposals.EditorItem{{CatalogID: "score-item-driver-register", Price: 0}},
	}
	if err := validateProposalForPublish(base); err == nil {
		t.Fatal("proposal without a priced product must not be publishable")
	}

	base.Items[0].Price = 10
	if err := validateProposalForPublish(base); err != nil {
		t.Fatalf("proposal with a priced product should pass publish validation: %v", err)
	}
}
