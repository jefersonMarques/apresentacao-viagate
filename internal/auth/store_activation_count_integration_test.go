package auth

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSessionUserCountsOnlyActivationsReadyForInternalSetup(t *testing.T) {
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

	schema := fmt.Sprintf("auth_activation_count_%d", time.Now().UTC().UnixNano())
	if _, err := adminPool.Exec(ctx, "create schema "+schema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}
	defer func() {
		_, _ = adminPool.Exec(ctx, "drop schema if exists "+schema+" cascade")
	}()

	if _, err := adminPool.Exec(ctx, fmt.Sprintf(`
		create table %s.users (
			id uuid primary key,
			email text not null,
			name text not null,
			status text not null,
			created_at timestamptz not null default now()
		);
		create table %s.roles (
			id uuid primary key,
			code text not null
		);
		create table %s.user_roles (
			user_id uuid not null,
			role_id uuid not null
		);
		create table %s.sessions (
			user_id uuid not null,
			token_hash bytea not null,
			revoked_at timestamptz,
			expires_at timestamptz not null
		);
		create table %s.in_app_notifications (
			recipient_user_id uuid not null,
			read_at timestamptz,
			deleted_at timestamptz
		);
		create table %s.proposals (
			id uuid primary key,
			deleted_at timestamptz
		);
		create table %s.proposal_acceptances (
			id uuid primary key,
			proposal_id uuid not null,
			deleted_at timestamptz
		);
		create table %s.onboardings (
			id uuid primary key,
			proposal_acceptance_id uuid not null,
			deleted_at timestamptz
		);
		create table %s.contracts (
			id uuid primary key,
			onboarding_id uuid not null,
			status text not null,
			fully_signed_at timestamptz,
			deleted_at timestamptz
		);
		create table %s.activation_profiles (
			id uuid primary key,
			contract_id uuid not null,
			status text not null,
			deleted_at timestamptz
		);
	`, schema, schema, schema, schema, schema, schema, schema, schema, schema, schema)); err != nil {
		t.Fatalf("create auth activation count tables: %v", err)
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

	const (
		managerID = "11111111-1111-1111-1111-111111111111"
		viewerID  = "22222222-2222-2222-2222-222222222222"
	)
	if _, err := pool.Exec(ctx, `
		create function effective_user_permissions(target_user_id uuid)
		returns table(permission_code text)
		language sql
		stable
		as $
			select 'activation.manage'::text
			where target_user_id = '11111111-1111-1111-1111-111111111111'::uuid
		$
	`); err != nil {
		t.Fatalf("create effective permissions function: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into users(id,email,name,status)
		values
			('11111111-1111-1111-1111-111111111111','manager@example.com','Manager','active'),
			('22222222-2222-2222-2222-222222222222','viewer@example.com','Viewer','active')
	`); err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into sessions(user_id,token_hash,expires_at)
		values
			('11111111-1111-1111-1111-111111111111',$1,now()+interval '1 hour'),
			('22222222-2222-2222-2222-222222222222',$2,now()+interval '1 hour')
	`, []byte("manager-token"), []byte("viewer-token")); err != nil {
		t.Fatalf("seed sessions: %v", err)
	}

	type activationSeed struct {
		base       string
		status     string
		contract   string
		fullySigned bool
		deleteProposal bool
		deleteActivation bool
	}
	seeds := []activationSeed{
		{base: "10000000", status: "completed", contract: "signed", fullySigned: true},
		{base: "20000000", status: "completed", contract: "sent", fullySigned: false},
		{base: "30000000", status: "completed", contract: "signed", fullySigned: false},
		{base: "40000000", status: "completed", contract: "signed", fullySigned: true, deleteProposal: true},
		{base: "50000000", status: "under_internal_setup", contract: "signed", fullySigned: true},
		{base: "60000000", status: "completed", contract: "signed", fullySigned: true, deleteActivation: true},
	}

	for _, seed := range seeds {
		proposalID := seed.base + "-0000-0000-0000-000000000001"
		acceptanceID := seed.base + "-0000-0000-0000-000000000002"
		onboardingID := seed.base + "-0000-0000-0000-000000000003"
		contractID := seed.base + "-0000-0000-0000-000000000004"
		activationID := seed.base + "-0000-0000-0000-000000000005"

		var proposalDeleted any
		if seed.deleteProposal {
			proposalDeleted = time.Now()
		}
		var activationDeleted any
		if seed.deleteActivation {
			activationDeleted = time.Now()
		}
		var fullySignedAt any
		if seed.fullySigned {
			fullySignedAt = time.Now()
		}

		if _, err := pool.Exec(ctx, `insert into proposals(id,deleted_at) values($1,$2)`, proposalID, proposalDeleted); err != nil {
			t.Fatalf("seed proposal %s: %v", seed.base, err)
		}
		if _, err := pool.Exec(ctx, `insert into proposal_acceptances(id,proposal_id) values($1,$2)`, acceptanceID, proposalID); err != nil {
			t.Fatalf("seed acceptance %s: %v", seed.base, err)
		}
		if _, err := pool.Exec(ctx, `insert into onboardings(id,proposal_acceptance_id) values($1,$2)`, onboardingID, acceptanceID); err != nil {
			t.Fatalf("seed onboarding %s: %v", seed.base, err)
		}
		if _, err := pool.Exec(ctx, `insert into contracts(id,onboarding_id,status,fully_signed_at) values($1,$2,$3,$4)`, contractID, onboardingID, seed.contract, fullySignedAt); err != nil {
			t.Fatalf("seed contract %s: %v", seed.base, err)
		}
		if _, err := pool.Exec(ctx, `insert into activation_profiles(id,contract_id,status,deleted_at) values($1,$2,$3,$4)`, activationID, contractID, seed.status, activationDeleted); err != nil {
			t.Fatalf("seed activation %s: %v", seed.base, err)
		}
	}

	store := NewStore(pool)
	manager, err := store.SessionUser(ctx, []byte("manager-token"))
	if err != nil {
		t.Fatalf("load manager session: %v", err)
	}
	if manager.ID != managerID {
		t.Fatalf("unexpected manager id: %s", manager.ID)
	}
	if manager.PendingActivations != 1 {
		t.Fatalf("expected exactly 1 activation ready for internal setup, got %d", manager.PendingActivations)
	}

	viewer, err := store.SessionUser(ctx, []byte("viewer-token"))
	if err != nil {
		t.Fatalf("load viewer session: %v", err)
	}
	if viewer.ID != viewerID {
		t.Fatalf("unexpected viewer id: %s", viewer.ID)
	}
	if viewer.PendingActivations != 0 {
		t.Fatalf("user without activation.manage should not receive activation badge, got %d", viewer.PendingActivations)
	}
}
