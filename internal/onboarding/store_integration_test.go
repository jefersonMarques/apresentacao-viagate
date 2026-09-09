package onboarding

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAutoApproveIsIdempotent(t *testing.T) {
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

	schema := fmt.Sprintf("onboarding_autoapprove_%d", time.Now().UTC().UnixNano())
	if _, err := adminPool.Exec(ctx, "create schema "+schema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}
	defer func() {
		_, _ = adminPool.Exec(ctx, "drop schema if exists "+schema+" cascade")
	}()

	if _, err := adminPool.Exec(ctx, fmt.Sprintf(`
		create table %s.onboardings (
			id text primary key,
			status text not null,
			approved_at timestamptz,
			reviewed_at timestamptz,
			review_notes text,
			updated_at timestamptz not null default now()
		);

		create table %s.audit_events (
			actor_type text,
			event_type text,
			resource_type text,
			resource_id text,
			metadata jsonb
		);
	`, schema, schema)); err != nil {
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

	if _, err := pool.Exec(ctx, `insert into onboardings(id,status) values('submitted-case','submitted'),('pending-case','pending')`); err != nil {
		t.Fatalf("seed onboardings: %v", err)
	}

	store := NewStore(pool)
	changed, err := store.AutoApprove(ctx, "submitted-case", "integration_test")
	if err != nil {
		t.Fatalf("first auto approval: %v", err)
	}
	if !changed {
		t.Fatal("first auto approval should report a state change")
	}

	changed, err = store.AutoApprove(ctx, "submitted-case", "integration_test")
	if err != nil {
		t.Fatalf("second auto approval: %v", err)
	}
	if changed {
		t.Fatal("second auto approval must be idempotent")
	}

	var status string
	var auditCount int
	if err := pool.QueryRow(ctx, `select status from onboardings where id='submitted-case'`).Scan(&status); err != nil {
		t.Fatalf("load approved onboarding: %v", err)
	}
	if status != "approved" {
		t.Fatalf("expected approved status, got %s", status)
	}
	if err := pool.QueryRow(ctx, `select count(*) from audit_events where resource_id='submitted-case' and event_type='onboarding.auto_approved'`).Scan(&auditCount); err != nil {
		t.Fatalf("count audit events: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected one auto-approval audit event, got %d", auditCount)
	}

	if _, err := store.AutoApprove(ctx, "pending-case", "integration_test"); err == nil {
		t.Fatal("pending onboarding must not be auto-approved")
	}
}
