package database

import (
    "context"
    "os"
    "strings"
    "testing"

    "github.com/jackc/pgx/v5/pgxpool"
)

func TestDefaultContractTemplateExcludesInsuranceAfterMigration(t *testing.T) {
    if os.Getenv("RUN_DB_INTEGRATION") != "1" {
        t.Skip("database integration test")
    }

    ctx := context.Background()
    pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
    if err != nil {
        t.Fatalf("connect to database: %v", err)
    }
    defer pool.Close()

    var currentVersion int
    var markdown string
    var historicalMarkdown string
    err = pool.QueryRow(ctx, `
        select t.current_version, v.markdown
        from contract_templates t
        join contract_template_versions v
          on v.contract_template_id=t.id
         and v.version_number=t.current_version
        where t.is_active=true and t.is_default=true
    `).Scan(&currentVersion, &markdown)
    if err != nil {
        t.Fatalf("load default contract template: %v", err)
    }

    forbidden := []string{
        "### 1.3. Operação",
        "### 1.4. Informações securitárias",
        "{operation.type}",
        "{insurance.",
    }

    for _, value := range forbidden {
        if strings.Contains(markdown, value) {
            t.Errorf("new contract contains forbidden insurance content %q", value)
        }
    }

    required := []string{
        "## 2. PREÇO E FORMA DE PAGAMENTO",
        "## 3. OBRIGAÇÕES DAS PARTES",
        "## 10. ACEITE E ASSINATURAS",
        "{proposal.pricing_table}",
        "{representative.cpf}",
    }
    for _, value := range required {
        if !strings.Contains(markdown, value) {
            t.Errorf("new contract is missing required content %q", value)
        }
    }

    if currentVersion <= 1 {
        t.Fatal("contract template has no preserved historical version")
    }

    err = pool.QueryRow(ctx, `
        select v.markdown
        from contract_templates t
        join contract_template_versions v
          on v.contract_template_id=t.id
         and v.version_number=t.current_version-1
        where t.is_active=true and t.is_default=true
    `).Scan(&historicalMarkdown)
    if err != nil {
        t.Fatalf("load historical contract template: %v", err)
    }
    if !strings.Contains(historicalMarkdown, "### 1.4. Informações securitárias") {
        t.Fatal("historical contract template must remain unchanged")
    }
}
