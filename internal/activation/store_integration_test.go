package activation

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestInternalSetupRequiresSignedContract(t *testing.T) {
	if os.Getenv("RUN_DB_INTEGRATION") != "1" {
		t.Skip("database integration test")
	}

	ctx := context.Background()
	databaseURL := os.Getenv("DATABASE_URL")
	adminPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open admin database: %v", err)
	}
	defer adminPool.Close()

	schema := fmt.Sprintf("activation_signature_gate_%d", time.Now().UTC().UnixNano())
	if _, err := adminPool.Exec(ctx, "create schema "+schema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}
	defer func() {
		_, _ = adminPool.Exec(ctx, "drop schema if exists "+schema+" cascade")
	}()

	if _, err := adminPool.Exec(ctx, fmt.Sprintf(`
		create table %s.contracts (
			id text primary key,
			status text not null,
			fully_signed_at timestamptz,
			deleted_at timestamptz
		);

		create table %s.activation_profiles (
			id text primary key,
			contract_id text not null references %s.contracts(id),
			status text not null,
			deleted_at timestamptz,
			activated_at timestamptz,
			updated_at timestamptz not null default now()
		);
	`, schema, schema, schema)); err != nil {
		t.Fatalf("create test tables: %v", err)
	}

	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse database config: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("open isolated test pool: %v", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, `
		insert into contracts(id,status,fully_signed_at)
		values
			('unsigned-contract','sent',null),
			('signed-contract','signed',now());

		insert into activation_profiles(id,contract_id,status)
		values
			('unsigned-activation','unsigned-contract','completed'),
			('signed-activation','signed-contract','completed');
	`); err != nil {
		t.Fatalf("seed activation states: %v", err)
	}

	store := NewStore(pool)

	ready, err := store.ReadyForInternalSetup(ctx, "unsigned-activation")
	if err != nil {
		t.Fatalf("check unsigned activation readiness: %v", err)
	}
	if ready {
		t.Fatal("unsigned contract must not be ready for internal setup")
	}
	if err := store.SetInternalStatus(ctx, "unsigned-activation", "under_internal_setup"); err == nil {
		t.Fatal("unsigned contract must not enter internal setup")
	}

	ready, err = store.ReadyForInternalSetup(ctx, "signed-activation")
	if err != nil {
		t.Fatalf("check signed activation readiness: %v", err)
	}
	if !ready {
		t.Fatal("signed contract with completed data should be ready for internal setup")
	}
	if err := store.SetInternalStatus(ctx, "signed-activation", "under_internal_setup"); err != nil {
		t.Fatalf("signed contract should enter internal setup: %v", err)
	}
}
