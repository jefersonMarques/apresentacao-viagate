package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMigration000029PreservesPublishedProposalVersions(t *testing.T) {
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

	schema := fmt.Sprintf("migration_000029_%d", time.Now().UTC().UnixNano())
	if _, err := adminPool.Exec(ctx, "create schema "+schema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}
	defer func() {
		_, _ = adminPool.Exec(ctx, "drop schema if exists "+schema+" cascade")
	}()

	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse database config: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("open isolated pool: %v", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, `
		create table users (
			id uuid primary key
		);
		create table proposals (
			id uuid primary key,
			created_by uuid not null references users(id),
			deleted_at timestamptz,
			updated_at timestamptz not null default now()
		);
		create table proposal_versions (
			id uuid primary key,
			proposal_id uuid not null references proposals(id),
			created_by uuid not null references users(id),
			created_at timestamptz not null default now(),
			published_at timestamptz
		);
		create table proposal_acceptances (
			id uuid primary key,
			evidence_value text not null,
			deleted_at timestamptz,
			deleted_by uuid
		);

		create or replace function prevent_published_proposal_version_mutation()
		returns trigger language plpgsql as $published$
		begin
			if tg_op = 'DELETE' and old.published_at is not null then
				raise exception 'published proposal versions are immutable';
			end if;
			if tg_op = 'UPDATE' and old.published_at is not null then
				raise exception 'published proposal versions are immutable';
			end if;
			return case when tg_op = 'DELETE' then old else new end;
		end;
		$published$;

		create trigger proposal_versions_immutable
		before update or delete on proposal_versions
		for each row execute function prevent_published_proposal_version_mutation();
	`); err != nil {
		t.Fatalf("prepare migration schema: %v", err)
	}

	const (
		ownerID     = "11111111-1111-1111-1111-111111111111"
		editorID    = "22222222-2222-2222-2222-222222222222"
		proposalID  = "33333333-3333-3333-3333-333333333333"
		publishedID = "44444444-4444-4444-4444-444444444444"
		draftID     = "55555555-5555-5555-5555-555555555555"
		acceptID    = "66666666-6666-6666-6666-666666666666"
	)

	createdAt := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	if _, err := pool.Exec(ctx, `
		insert into users(id) values($1),($2);
		insert into proposals(id,created_by) values($3,$1);
		insert into proposal_versions(id,proposal_id,created_by,created_at,published_at)
		values
			($4,$3,$1,$6,now()),
			($5,$3,$1,$6,null);
		insert into proposal_acceptances(id,evidence_value)
		values($7,'original');
	`, ownerID, editorID, proposalID, publishedID, draftID, createdAt, acceptID); err != nil {
		t.Fatalf("seed migration scenario: %v", err)
	}

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file path")
	}
	migrationPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "migrations", "000029_proposal_audit_and_soft_delete_fix.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read migration 000029: %v", err)
	}

	if _, err := pool.Exec(ctx, string(migrationSQL)); err != nil {
		t.Fatalf("apply migration 000029 with published version present: %v", err)
	}

	var publishedUpdatedBy *string
	var publishedUpdatedAt *time.Time
	if err := pool.QueryRow(ctx, `
		select updated_by::text,updated_at
		from proposal_versions
		where id=$1
	`, publishedID).Scan(&publishedUpdatedBy, &publishedUpdatedAt); err != nil {
		t.Fatalf("load published version metadata: %v", err)
	}
	if publishedUpdatedBy != nil || publishedUpdatedAt != nil {
		t.Fatalf("published historical version should not be backfilled: updated_by=%v updated_at=%v", publishedUpdatedBy, publishedUpdatedAt)
	}

	var draftUpdatedBy string
	var draftUpdatedAt time.Time
	if err := pool.QueryRow(ctx, `
		select updated_by::text,updated_at
		from proposal_versions
		where id=$1
	`, draftID).Scan(&draftUpdatedBy, &draftUpdatedAt); err != nil {
		t.Fatalf("load draft metadata: %v", err)
	}
	if draftUpdatedBy != ownerID {
		t.Fatalf("draft editor backfill mismatch: got %s want %s", draftUpdatedBy, ownerID)
	}
	if !draftUpdatedAt.Equal(createdAt) {
		t.Fatalf("draft updated_at backfill mismatch: got %s want %s", draftUpdatedAt, createdAt)
	}

	var proposalUpdatedBy string
	if err := pool.QueryRow(ctx, `select updated_by::text from proposals where id=$1`, proposalID).Scan(&proposalUpdatedBy); err != nil {
		t.Fatalf("load proposal editor backfill: %v", err)
	}
	if proposalUpdatedBy != ownerID {
		t.Fatalf("proposal editor backfill mismatch: got %s want %s", proposalUpdatedBy, ownerID)
	}

	if _, err := pool.Exec(ctx, `update proposal_versions set updated_by=$2 where id=$1`, publishedID, editorID); err == nil {
		t.Fatal("published proposal version mutation should remain blocked")
	}

	if _, err := pool.Exec(ctx, `update proposal_acceptances set deleted_at=now(),deleted_by=$2 where id=$1`, acceptID, editorID); err != nil {
		t.Fatalf("soft-delete metadata should be allowed on acceptance: %v", err)
	}
	if _, err := pool.Exec(ctx, `update proposal_acceptances set evidence_value='tampered' where id=$1`, acceptID); err == nil {
		t.Fatal("acceptance evidence mutation should remain blocked")
	}

	if _, err := pool.Exec(ctx, `update proposals set created_by=$2 where id=$1`, proposalID, editorID); err == nil {
		t.Fatal("proposal owner reassignment should remain blocked")
	}
}
