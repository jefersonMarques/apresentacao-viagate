package activation

import (
    "context"
    "fmt"
    "os"
    "testing"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
)

func newActivationIntegrationStore(t *testing.T) (context.Context, *Store) {
    t.Helper()
    if os.Getenv("RUN_DB_INTEGRATION") != "1" {
        t.Skip("database integration test")
    }

    ctx := context.Background()
    databaseURL := os.Getenv("DATABASE_URL")
    adminPool, err := pgxpool.New(ctx, databaseURL)
    if err != nil {
        t.Fatalf("open database: %v", err)
    }
    t.Cleanup(adminPool.Close)

    schema := fmt.Sprintf("activation_operation_gate_%d", time.Now().UTC().UnixNano())
    if _, err := adminPool.Exec(ctx, "create schema "+schema); err != nil {
        t.Fatalf("create test schema: %v", err)
    }
    t.Cleanup(func() {
        _, _ = adminPool.Exec(context.Background(), "drop schema if exists "+schema+" cascade")
    })

    tables := fmt.Sprintf(`
        create table %s.proposal_versions (
            id text primary key,
            requires_policy boolean not null
        );
        create table %s.proposal_acceptances (
            id text primary key,
            proposal_version_id text not null
        );
        create table %s.onboardings (
            id text primary key,
            proposal_acceptance_id text not null,
            operation_type text,
            insurer text,
            policy_start_date date,
            policy_end_date date,
            deleted_at timestamptz
        );
        create table %s.contracts (
            id text primary key,
            onboarding_id text not null,
            status text not null,
            fully_signed_at timestamptz,
            deleted_at timestamptz
        );
        create table %s.activation_profiles (
            id text primary key,
            contract_id text not null,
            status text not null,
            finance_responsible_name text,
            finance_responsible_email text,
            finance_responsible_phone text,
            submitted_at timestamptz,
            activated_at timestamptz,
            updated_at timestamptz not null default now(),
            deleted_at timestamptz
        );
        create table %s.post_sale_operations (
            contract_id text primary key,
            operation_type text,
            insurer text,
            policy_start_date date,
            policy_end_date date
        );
        create table %s.activation_access_tokens (
            id text primary key,
            activation_id text not null,
            section text not null,
            revoked_at timestamptz,
            expires_at timestamptz not null
        );
        create table %s.activation_goods (
            activation_id text not null,
            description text not null
        );
        create table %s.activation_system_users (
            activation_id text not null,
            name text not null,
            email text not null
        );
        create table %s.uploaded_documents (
            onboarding_id text not null,
            document_type text not null,
            status text not null,
            deleted_at timestamptz
        );
    `, schema, schema, schema, schema, schema, schema, schema, schema, schema, schema)
    if _, err := adminPool.Exec(ctx, tables); err != nil {
        t.Fatalf("create isolated activation tables: %v", err)
    }

    cfg, err := pgxpool.ParseConfig(databaseURL)
    if err != nil {
        t.Fatalf("parse test database configuration: %v", err)
    }
    cfg.ConnConfig.RuntimeParams["search_path"] = schema
    pool, err := pgxpool.NewWithConfig(ctx, cfg)
    if err != nil {
        t.Fatalf("open isolated test pool: %v", err)
    }
    t.Cleanup(pool.Close)

    seed := `
        insert into proposal_versions(id,requires_policy)
        values ('regular-version',false),('insured-version',true);

        insert into proposal_acceptances(id,proposal_version_id)
        values ('regular-acceptance','regular-version'),
               ('insured-acceptance','insured-version');

        insert into onboardings(id,proposal_acceptance_id)
        values ('regular-onboarding','regular-acceptance'),
               ('insured-onboarding','insured-acceptance');

        insert into contracts(id,onboarding_id,status,fully_signed_at)
        values ('unsigned-contract','regular-onboarding','sent',null),
               ('signed-contract','regular-onboarding','signed',now()),
               ('insured-contract','insured-onboarding','signed',now());

        insert into activation_profiles(
            id,contract_id,status,finance_responsible_name,
            finance_responsible_email,finance_responsible_phone
        ) values
            ('unsigned-activation','unsigned-contract','completed',
             'Finance','finance@example.com','41999999999'),
            ('signed-activation','signed-contract','completed',
             'Finance','finance@example.com','41999999999'),
            ('insured-activation','insured-contract','completed',
             'Finance','finance@example.com','41999999999'),
            ('insured-submitting','insured-contract','in_progress',
             'Finance','finance@example.com','41999999999');

        insert into activation_access_tokens(
            id,activation_id,section,expires_at
        ) values
            ('insured-token','insured-submitting','all',now()+interval '1 day'),
            ('restricted-token','insured-submitting','finance',now()+interval '1 day');

        insert into activation_goods(activation_id,description)
        values ('insured-submitting','Refrigerated goods');

        insert into activation_system_users(activation_id,name,email)
        values ('insured-submitting','Operator','operator@example.com');

        insert into uploaded_documents(onboarding_id,document_type,status)
        values ('insured-onboarding','insurance_policy','uploaded');
    `
    if _, err := pool.Exec(ctx, seed); err != nil {
        t.Fatalf("seed activation states: %v", err)
    }

    return ctx, NewStore(pool)
}

func TestInternalSetupRequiresSignedContractAndCommercialOperation(t *testing.T) {
    ctx, store := newActivationIntegrationStore(t)

    unsignedReady, err := store.ReadyForInternalSetup(ctx, "unsigned-activation")
    if err != nil {
        t.Fatalf("check unsigned readiness: %v", err)
    }
    if unsignedReady {
        t.Fatal("unsigned contract must not be ready for internal setup")
    }
    if err := store.SetInternalStatus(ctx, "unsigned-activation", "under_internal_setup"); err == nil {
        t.Fatal("unsigned contract must not start internal setup")
    }

    regularReady, err := store.ReadyForInternalSetup(ctx, "signed-activation")
    if err != nil || !regularReady {
        t.Fatalf("signed non-insured activation should be ready: ready=%v err=%v", regularReady, err)
    }
    if err := store.SetInternalStatus(ctx, "signed-activation", "under_internal_setup"); err != nil {
        t.Fatalf("signed non-insured activation should start: %v", err)
    }

    insuredReady, err := store.ReadyForInternalSetup(ctx, "insured-activation")
    if err != nil {
        t.Fatalf("check insured readiness without operation: %v", err)
    }
    if insuredReady {
        t.Fatal("insured activation must wait for the commercial operation")
    }
    if err := store.SetInternalStatus(ctx, "insured-activation", "under_internal_setup"); err == nil {
        t.Fatal("direct status transition must reject missing commercial operation")
    }

    _, err = store.pool.Exec(ctx, `
        insert into post_sale_operations(
            contract_id,operation_type,insurer,policy_start_date,policy_end_date
        ) values (
            'insured-contract','normal','Insurer','2026-11-01','2027-11-01'
        )
    `)
    if err != nil {
        t.Fatalf("save commercial operation in fixture: %v", err)
    }

    insuredReady, err = store.ReadyForInternalSetup(ctx, "insured-activation")
    if err != nil || !insuredReady {
        t.Fatalf("insured activation should be ready after commercial input: ready=%v err=%v", insuredReady, err)
    }
    if err := store.SetInternalStatus(ctx, "insured-activation", "under_internal_setup"); err != nil {
        t.Fatalf("insured activation should start after commercial input: %v", err)
    }
}

func TestSubmitCustomerDataWithoutCommercialOperation(t *testing.T) {
    ctx, store := newActivationIntegrationStore(t)

    if err := store.Submit(ctx, "restricted-token"); err == nil {
        t.Fatal("restricted access cannot confirm all customer sections")
    }
    if err := store.Submit(ctx, "insured-token"); err != nil {
        t.Fatalf("customer confirmation must not depend on commercial operation: %v", err)
    }

    var status string
    var submittedAt *time.Time
    err := store.pool.QueryRow(ctx, `
        select status,submitted_at
        from activation_profiles where id='insured-submitting'
    `).Scan(&status,&submittedAt)
    if err != nil {
        t.Fatalf("load submitted activation: %v", err)
    }
    if status != "completed" || submittedAt == nil {
        t.Fatalf("customer submission not recorded: status=%q submitted_at=%v", status, submittedAt)
    }

    ready, err := store.ReadyForInternalSetup(ctx, "insured-submitting")
    if err != nil {
        t.Fatalf("check commercial readiness after customer submit: %v", err)
    }
    if ready {
        t.Fatal("customer submission alone must not unlock internal setup")
    }
    if err := store.SetInternalStatus(ctx, "insured-submitting", "under_internal_setup"); err == nil {
        t.Fatal("direct internal transition must remain blocked")
    }

    _, err = store.pool.Exec(ctx, `
        insert into post_sale_operations(
            contract_id,operation_type,insurer,policy_start_date,policy_end_date
        ) values ('insured-contract','normal','Insurer','2026-11-01','2027-11-01')
    `)
    if err != nil {
        t.Fatalf("save commercial operation: %v", err)
    }

    ready, err = store.ReadyForInternalSetup(ctx, "insured-submitting")
    if err != nil || !ready {
        t.Fatalf("commercial completion after customer submission should unlock setup: ready=%v err=%v",ready,err)
    }
    if err := store.SetInternalStatus(ctx, "insured-submitting", "under_internal_setup"); err != nil {
        t.Fatalf("start internal setup after commercial completion: %v",err)
    }
}
