package proposals

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPublicByTokenRedirectsSupersededPublishedVersion(t *testing.T) {
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

	schema := fmt.Sprintf("proposal_public_link_%d", time.Now().UTC().UnixNano())
	if _, err := adminPool.Exec(ctx, "create schema "+schema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}
	defer func() {
		_, _ = adminPool.Exec(ctx, "drop schema if exists "+schema+" cascade")
	}()

	if _, err := adminPool.Exec(ctx, fmt.Sprintf(`
		create table %s.clients (
			id uuid primary key,
			legal_name text,
			cnpj varchar(14)
		);
		create table %s.proposals (
			id uuid primary key,
			client_id uuid not null,
			title text not null,
			status text not null,
			current_version integer not null,
			valid_until date,
			deleted_at timestamptz
		);
		create table %s.proposal_versions (
			id uuid primary key,
			proposal_id uuid not null,
			version_number integer not null,
			public_token uuid not null,
			pricing_model text not null,
			content jsonb not null,
			conditions jsonb not null default '[]'::jsonb,
			minimum_invoice numeric(14,2) not null default 0,
			setup_fee numeric(14,2) not null default 0,
			requires_policy boolean not null default false,
			content_hash bytea not null,
			published_at timestamptz
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
	`, schema, schema, schema, schema)); err != nil {
		t.Fatalf("create proposal public-link tables: %v", err)
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

	const (
		clientID    = "10000000-0000-0000-0000-000000000001"
		proposalID  = "20000000-0000-0000-0000-000000000001"
		versionOne  = "30000000-0000-0000-0000-000000000001"
		versionTwo  = "30000000-0000-0000-0000-000000000002"
		oldToken    = "40000000-0000-0000-0000-000000000001"
		currentToken = "40000000-0000-0000-0000-000000000002"
	)

	if _, err := pool.Exec(ctx, `
		insert into clients(id,legal_name,cnpj)
		values($1,'Cliente Teste Ltda.','12345678000199')
	`, clientID); err != nil {
		t.Fatalf("seed client: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into proposals(id,client_id,title,status,current_version)
		values($1,$2,'Proposta Teste','published',2)
	`, proposalID, clientID); err != nil {
		t.Fatalf("seed proposal: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into proposal_versions(
			id,proposal_id,version_number,public_token,pricing_model,content,conditions,content_hash,published_at
		) values
			($1,$2,1,$3,'catalog','{"proposal":{"title":"Versão 1"}}','[]',decode('01','hex'),now()),
			($4,$2,2,$5,'catalog','{"proposal":{"title":"Versão 2"}}','[]',decode('02','hex'),now())
	`, versionOne, proposalID, oldToken, versionTwo, currentToken); err != nil {
		t.Fatalf("seed proposal versions: %v", err)
	}

	store := NewStore(pool)

	_, err = store.PublicByToken(ctx, oldToken)
	var superseded *SupersededProposalError
	if !errors.As(err, &superseded) {
		t.Fatalf("old published token should be superseded, got %v", err)
	}
	if superseded.CurrentToken != currentToken {
		t.Fatalf("unexpected current token: got %s want %s", superseded.CurrentToken, currentToken)
	}

	current, err := store.PublicByToken(ctx, currentToken)
	if err != nil {
		t.Fatalf("current published token should resolve: %v", err)
	}
	if current.VersionNumber != 2 || current.PublicToken != currentToken || current.Title != "Versão 2" {
		t.Fatalf("unexpected current proposal: %+v", current)
	}
}
