package proposals

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestProposalDraftOwnershipAndAudit(t *testing.T) {
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

	schema := fmt.Sprintf("proposal_audit_%d", time.Now().UTC().UnixNano())
	if _, err := adminPool.Exec(ctx, "create schema "+schema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}
	defer func() {
		_, _ = adminPool.Exec(ctx, "drop schema if exists "+schema+" cascade")
	}()

	if _, err := adminPool.Exec(ctx, fmt.Sprintf(`
		create table %s.clients (
			id uuid primary key default gen_random_uuid(),
			legal_name text not null,
			trade_name text,
			cnpj varchar(14),
			email citext,
			phone text,
			street text,
			street_number text,
			complement text,
			district text,
			city text,
			state char(2),
			postal_code text,
			created_by uuid not null,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now()
		);
		create table %s.proposals (
			id uuid primary key default gen_random_uuid(),
			client_id uuid not null,
			title text not null,
			status text not null default 'draft',
			current_version integer not null default 0,
			valid_until date,
			is_default boolean not null default false,
			created_by uuid not null,
			updated_by uuid,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now(),
			deleted_at timestamptz
		);
		create table %s.proposal_versions (
			id uuid primary key default gen_random_uuid(),
			proposal_id uuid not null,
			version_number integer not null,
			public_token uuid not null default gen_random_uuid(),
			pricing_model text not null,
			content jsonb not null,
			conditions jsonb not null default '[]'::jsonb,
			minimum_invoice numeric(14,2) not null default 0,
			setup_fee numeric(14,2) not null default 0,
			content_hash bytea not null,
			requires_policy boolean not null default false,
			published_at timestamptz,
			created_by uuid not null,
			updated_by uuid,
			created_at timestamptz not null default now(),
			updated_at timestamptz not null default now()
		);
		create table %s.proposal_items (
			id uuid primary key default gen_random_uuid(),
			proposal_version_id uuid not null,
			group_name text not null,
			label text not null,
			unit text,
			price numeric(14,4) not null default 0,
			is_optional boolean not null default false,
			sort_order integer not null default 0,
			metadata jsonb not null default '{}'::jsonb
		);
		create table %s.audit_events (
			id bigint generated always as identity primary key,
			actor_user_id uuid,
			event_type text not null,
			resource_type text not null,
			resource_id uuid,
			metadata jsonb not null default '{}'::jsonb,
			created_at timestamptz not null default now()
		);
	`, schema, schema, schema, schema, schema)); err != nil {
		t.Fatalf("create proposal audit test tables: %v", err)
	}

	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse database config: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("open isolated test pool: %v", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, `
		create or replace function protect_proposal_owner()
		returns trigger language plpgsql as $$
		begin
			if new.created_by is distinct from old.created_by then
				raise exception 'proposal owner is immutable';
			end if;
			return new;
		end;
		$$;
		create trigger proposals_owner_immutable
		before update on proposals
		for each row execute function protect_proposal_owner();

		create or replace function protect_proposal_version_creator()
		returns trigger language plpgsql as $$
		begin
			if new.created_by is distinct from old.created_by then
				raise exception 'proposal version creator is immutable';
			end if;
			return new;
		end;
		$$;
		create trigger proposal_versions_creator_immutable
		before update on proposal_versions
		for each row execute function protect_proposal_version_creator();
	`); err != nil {
		t.Fatalf("create ownership triggers: %v", err)
	}

	const (
		ownerID  = "11111111-1111-1111-1111-111111111111"
		editorID = "22222222-2222-2222-2222-222222222222"
	)

	store := NewStore(pool)
	first, err := store.SaveDraft(ctx, ownerID, false, EditorInput{
		ClientLegalName: "Cliente Original Ltda.",
		Title:           "Proposta Original",
		PricingModel:    "catalog",
		Content:         map[string]any{"proposal": map[string]any{"title": "Proposta Original"}},
		ContentHash:     []byte{0x01, 0x02},
	})
	if err != nil {
		t.Fatalf("create proposal draft: %v", err)
	}

	_, err = store.SaveDraft(ctx, editorID, true, EditorInput{
		ProposalID:      first.ProposalID,
		ClientLegalName: "Cliente Original Ltda.",
		Title:           "Proposta Revisada",
		PricingModel:    "catalog",
		Content:         map[string]any{"proposal": map[string]any{"title": "Proposta Revisada"}},
		ContentHash:     []byte{0x03, 0x04},
	})
	if err != nil {
		t.Fatalf("edit proposal draft as another user: %v", err)
	}

	var proposalOwner, proposalEditor string
	if err := pool.QueryRow(ctx, `
		select created_by::text,updated_by::text
		from proposals
		where id=$1
	`, first.ProposalID).Scan(&proposalOwner, &proposalEditor); err != nil {
		t.Fatalf("load proposal ownership: %v", err)
	}
	if proposalOwner != ownerID {
		t.Fatalf("proposal owner changed: got %s want %s", proposalOwner, ownerID)
	}
	if proposalEditor != editorID {
		t.Fatalf("latest proposal editor not tracked: got %s want %s", proposalEditor, editorID)
	}

	var versionCreator, versionEditor string
	if err := pool.QueryRow(ctx, `
		select created_by::text,updated_by::text
		from proposal_versions
		where id=$1
	`, first.VersionID).Scan(&versionCreator, &versionEditor); err != nil {
		t.Fatalf("load proposal version ownership: %v", err)
	}
	if versionCreator != ownerID {
		t.Fatalf("proposal version creator changed: got %s want %s", versionCreator, ownerID)
	}
	if versionEditor != editorID {
		t.Fatalf("latest version editor not tracked: got %s want %s", versionEditor, editorID)
	}

	rows, err := pool.Query(ctx, `
		select actor_user_id::text,coalesce(metadata->>'previous_hash',''),metadata->>'current_hash'
		from audit_events
		where resource_id=$1 and event_type='proposal.draft_saved'
		order by id
	`, first.ProposalID)
	if err != nil {
		t.Fatalf("load proposal draft audit: %v", err)
	}
	defer rows.Close()

	type auditRow struct {
		actor        string
		previousHash string
		currentHash  string
	}
	var auditRows []auditRow
	for rows.Next() {
		var row auditRow
		if err := rows.Scan(&row.actor, &row.previousHash, &row.currentHash); err != nil {
			t.Fatalf("scan proposal draft audit: %v", err)
		}
		auditRows = append(auditRows, row)
	}
	if len(auditRows) != 2 {
		t.Fatalf("expected 2 proposal draft audit events, got %d", len(auditRows))
	}
	if auditRows[0].actor != ownerID || auditRows[0].currentHash != "0102" {
		t.Fatalf("unexpected first audit event: %+v", auditRows[0])
	}
	if auditRows[1].actor != editorID || auditRows[1].previousHash != "0102" || auditRows[1].currentHash != "0304" {
		t.Fatalf("unexpected second audit event: %+v", auditRows[1])
	}

	if _, err := pool.Exec(ctx, `update proposals set created_by=$2 where id=$1`, first.ProposalID, editorID); err == nil {
		t.Fatal("database should reject proposal owner reassignment")
	}
	if _, err := pool.Exec(ctx, `update proposal_versions set created_by=$2 where id=$1`, first.VersionID, editorID); err == nil {
		t.Fatal("database should reject proposal version creator reassignment")
	}
}
