package contracts

import (
    "os"
    "strings"
    "testing"
)

func TestRevisedContractTemplateExcludesOperationalFields(t *testing.T) {
    data, err := os.ReadFile("testdata/default_contract_template.md")
    if err != nil {
        t.Fatalf("read baseline contract: %v", err)
    }

    baseline := string(data)
    operationStart := strings.Index(baseline, "### 1.3. Operação")
    insuranceStart := strings.Index(baseline, "### 1.4. Informações securitárias")
    pricingStart := strings.Index(baseline, "## 2. PREÇO E FORMA DE PAGAMENTO")
    if operationStart < 0 || insuranceStart <= operationStart || pricingStart <= insuranceStart {
        t.Fatal("legacy section boundaries are invalid")
    }

    revised := strings.TrimRight(baseline[:operationStart], "\n") +
        "\n\n" + baseline[pricingStart:]
    if templateRequiresInsuranceDetails(revised) {
        t.Fatal("revised template still requires insurance variables")
    }
    if !strings.Contains(revised, "## 3. OBRIGAÇÕES DAS PARTES") {
        t.Fatal("revised template lost general contractual obligations")
    }
    if err := validateTemplateBooleanUsage(revised); err != nil {
        t.Fatalf("invalid contract conditionals: %v", err)
    }
    for _, dataSet := range contractValidationDataSets("Via Gateway Ltda", "39.406.000/0001-21") {
        if _, _, err := NewRenderer().Render(revised, dataSet); err != nil {
            t.Fatalf("render contract without operation data: %v", err)
        }
    }
}
