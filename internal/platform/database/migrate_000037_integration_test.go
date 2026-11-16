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

    tx, err := pool.Begin(ctx)
    if err != nil {
        t.Fatalf("begin template migration test: %v", err)
    }
    defer tx.Rollback(ctx)

    _, err = tx.Exec(ctx, `
        create temp table contract_templates (
            id uuid primary key,
            created_by uuid not null,
            current_version integer not null,
            is_active boolean not null,
            is_default boolean not null,
            updated_at timestamptz
        );
        create temp table contract_template_versions (
            contract_template_id uuid not null,
            version_number integer not null,
            markdown text not null,
            template_hash bytea,
            created_by uuid not null
        );
    `)
    if err != nil {
        t.Fatalf("create isolated contract tables: %v", err)
    }

    original := strings.Join([]string{
        "# Example Contract",
        "",
        "## 1. OBJETO",
        "",
        "### 1.2. Produtos e serviços contratados",
        "",
        "Produtos da proposta.",
        "",
        "### 1.3. Operação",
        "",
        "**Tipo de operação:** {operation.type}",
        "",
        "### 1.4. Informações securitárias",
        "",
        "| Seguradora | {insurance.insurer} |",
        "",
        "## 2. PREÇO E FORMA DE PAGAMENTO",
        "",
        "{proposal.pricing_table}",
        "",
        "## 3. OBRIGAÇÕES DAS PARTES",
        "",
        "As partes observarão as obrigações contratuais.",
        "",
        "## 10. ACEITE E ASSINATURAS",
        "",
        "{representative.cpf}",
    }, "\n")

    _, err = tx.Exec(ctx, `
        insert into contract_templates(id,created_by,current_version,is_active,is_default)
        values ('aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa','bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb',1,true,true)
    `)
    if err != nil {
        t.Fatalf("seed template header: %v", err)
    }

    _, err = tx.Exec(ctx, `
        insert into contract_template_versions(
            contract_template_id,version_number,markdown,template_hash,created_by
        ) values (
            'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',1,$1,digest($1,'sha256'),
            'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb'
        )
    `, original)
    if err != nil {
        t.Fatalf("seed isolated contract template: %v", err)
    }

    migration, err := os.ReadFile("../../../migrations/000037_contract_without_operational_insurance.sql")
    if err != nil {
        t.Fatalf("read contract template migration: %v", err)
    }
    if _, err := tx.Exec(ctx, string(migration)); err != nil {
        t.Fatalf("apply contract template migration: %v", err)
    }

    var version int
    err = tx.QueryRow(ctx, `
        select current_version from contract_templates
        where is_active=true and is_default=true
    `).Scan(&version)
    if err != nil || version != 2 {
        t.Fatalf("version after migration = %d (error: %v), want 2", version, err)
    }

    var revised, preserved string
    err = tx.QueryRow(ctx, `
        select markdown from contract_template_versions
        where contract_template_id='aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'
          and version_number=2
    `).Scan(&revised)
    if err != nil {
        t.Fatalf("load revised contract: %v", err)
    }

    err = tx.QueryRow(ctx, `
        select markdown from contract_template_versions
        where contract_template_id='aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'
          and version_number=1
    `).Scan(&preserved)
    if err != nil {
        t.Fatalf("load historical contract: %v", err)
    }
    if preserved != original {
        t.Fatal("migration changed the historical contract version")
    }

    for _, value := range []string{
        "### 1.3. Operação",
        "### 1.4. Informações securitárias",
        "{operation.type}",
        "{insurance.",
    } {
        if strings.Contains(revised, value) {
            t.Errorf("revised contract contains forbidden section or field %q", value)
        }
    }

    for _, value := range []string{
        "### 1.2. Produtos e serviços contratados",
        "## 2. PREÇO E FORMA DE PAGAMENTO",
        "## 3. OBRIGAÇÕES DAS PARTES",
        "## 10. ACEITE E ASSINATURAS",
        "{proposal.pricing_table}",
    } {
        if !strings.Contains(revised, value) {
            t.Errorf("revised contract lost content %q", value)
        }
    }
}
