package proposals

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSoftDeleteCascadePreservesSignedEvidence(t *testing.T) {
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

	schema := fmt.Sprintf("proposal_soft_delete_%d", time.Now().UTC().UnixNano())
	if _, err := adminPool.Exec(ctx, "create schema "+schema); err != nil {
		t.Fatalf("create test schema: %v", err)
	}
	defer func() {
		_, _ = adminPool.Exec(ctx, "drop schema if exists "+schema+" cascade")
	}()

	if _, err := adminPool.Exec(ctx, fmt.Sprintf(`
		create table %s.proposals (
			id uuid primary key,
			status text not null,
			is_default boolean not null default false,
			deleted_at timestamptz,
			deleted_by uuid,
			updated_by uuid,
			updated_at timestamptz not null default now()
		);
		create table %s.proposal_acceptances (
			id uuid primary key,
			proposal_id uuid not null,
			accepted_by_name text not null,
			deleted_at timestamptz,
			deleted_by uuid
		);
		create table %s.onboardings (
			id uuid primary key,
			proposal_acceptance_id uuid not null,
			deleted_at timestamptz,
			deleted_by uuid,
			updated_at timestamptz not null default now()
		);
		create table %s.uploaded_documents (
			id uuid primary key,
			onboarding_id uuid not null,
			deleted_at timestamptz,
			deleted_by uuid
		);
		create table %s.contracts (
			id uuid primary key,
			onboarding_id uuid not null,
			status text not null,
			fully_signed_at timestamptz,
			document_sha256 bytea,
			deleted_at timestamptz,
			deleted_by uuid,
			updated_at timestamptz not null default now()
		);
		create table %s.contract_signers (
			id uuid primary key,
			contract_id uuid not null,
			deleted_at timestamptz,
			deleted_by uuid
		);
		create table %s.activation_profiles (
			id uuid primary key,
			contract_id uuid not null,
			deleted_at timestamptz,
			deleted_by uuid,
			updated_at timestamptz not null default now()
		);
		create table %s.customer_sessions (
			id uuid primary key,
			proposal_acceptance_id uuid not null,
			revoked_at timestamptz
		);
		create table %s.customer_resume_tokens (
			id uuid primary key,
			proposal_acceptance_id uuid not null,
			revoked_at timestamptz
		);
		create table %s.activation_access_tokens (
			id uuid primary key,
			activation_id uuid not null,
			revoked_at timestamptz
		);
		create table %s.notification_outbox (
			id uuid primary key,
			status text not null,
			processing_at timestamptz,
			last_error text,
			dedupe_key text
		);
		create table %s.in_app_notifications (
			id uuid primary key,
			resource_type text,
			resource_id uuid,
			deleted_at timestamptz
		);
		create table %s.audit_events (
			id bigint generated always as identity primary key,
			actor_user_id uuid,
			actor_type text,
			event_type text,
			resource_type text,
			resource_id uuid,
			ip_address inet,
			user_agent text,
			metadata jsonb
		);
		create table %s.signature_events (
			id bigint generated always as identity primary key,
			contract_id uuid not null,
			event_type text not null
		);
		create table %s.contract_finalization_jobs (
			contract_id uuid primary key,
			status text not null
		);
	`,
		schema, schema, schema, schema, schema, schema, schema, schema,
		schema, schema, schema, schema, schema, schema, schema,
	)); err != nil {
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
		create or replace function prevent_proposal_acceptance_mutation()
		returns trigger language plpgsql as $
		begin
			if tg_op = 'DELETE' then
				raise exception 'proposal acceptances are immutable';
			end if;
			if (to_jsonb(new) - 'deleted_at' - 'deleted_by') is distinct from (to_jsonb(old) - 'deleted_at' - 'deleted_by') then
				raise exception 'proposal acceptance evidence is immutable';
			end if;
			return new;
		end;
		$;
		create trigger proposal_acceptances_immutable
		before update or delete on proposal_acceptances
		for each row execute function prevent_proposal_acceptance_mutation();
	`); err != nil {
		t.Fatalf("create proposal acceptance evidence trigger: %v", err)
	}

	const (
		proposalID   = "11111111-1111-1111-1111-111111111111"
		acceptanceID = "22222222-2222-2222-2222-222222222222"
		onboardingID = "33333333-3333-3333-3333-333333333333"
		documentID   = "44444444-4444-4444-4444-444444444444"
		contractID   = "55555555-5555-5555-5555-555555555555"
		signerID     = "66666666-6666-6666-6666-666666666666"
		activationID = "77777777-7777-7777-7777-777777777777"
		actorID      = "88888888-8888-8888-8888-888888888888"
	)

	documentHash := []byte{0x01, 0x02, 0x03, 0x04}
	seedStatements := []struct {
		query string
		args  []any
	}{
		{`insert into proposals(id,status,is_default) values($1,'accepted',true)`, []any{proposalID}},
		{`insert into proposal_acceptances(id,proposal_id,accepted_by_name) values($1,$2,'Cliente Original')`, []any{acceptanceID, proposalID}},
		{`insert into onboardings(id,proposal_acceptance_id) values($1,$2)`, []any{onboardingID, acceptanceID}},
		{`insert into uploaded_documents(id,onboarding_id) values($1,$2)`, []any{documentID, onboardingID}},
		{`insert into contracts(id,onboarding_id,status,fully_signed_at,document_sha256) values($1,$2,'signed',now(),$3)`, []any{contractID, onboardingID, documentHash}},
		{`insert into contract_signers(id,contract_id) values($1,$2)`, []any{signerID, contractID}},
		{`insert into activation_profiles(id,contract_id) values($1,$2)`, []any{activationID, contractID}},
		{`insert into customer_sessions(id,proposal_acceptance_id) values('99999999-9999-9999-9999-999999999999',$1)`, []any{acceptanceID}},
		{`insert into customer_resume_tokens(id,proposal_acceptance_id) values('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',$1)`, []any{acceptanceID}},
		{`insert into activation_access_tokens(id,activation_id) values('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',$1)`, []any{activationID}},
		{`insert into notification_outbox(id,status,dedupe_key) values('cccccccc-cccc-cccc-cccc-cccccccccccc','pending','contract-signature:' || $1::text)`, []any{contractID}},
		{`insert into in_app_notifications(id,resource_type,resource_id) values('dddddddd-dddd-dddd-dddd-dddddddddddd','contract',$1)`, []any{contractID}},
		{`insert into signature_events(contract_id,event_type) values($1,'contract.signed')`, []any{contractID}},
		{`insert into contract_finalization_jobs(contract_id,status) values($1,'pending')`, []any{contractID}},
	}
	for _, statement := range seedStatements {
		if _, err := pool.Exec(ctx, statement.query, statement.args...); err != nil {
			t.Fatalf("seed proposal lifecycle: %v", err)
		}
	}

	store := NewStore(pool)
	result, err := store.SoftDeleteCascade(ctx, SoftDeleteInput{
		ProposalID:  proposalID,
		ActorUserID: actorID,
		IPAddress:   net.ParseIP("127.0.0.1"),
		UserAgent:   "soft-delete-integration-test",
	})
	if err != nil {
		t.Fatalf("soft delete proposal lifecycle: %v", err)
	}

	if result.OriginalStatus != "accepted" ||
		result.Acceptances != 1 ||
		result.Onboardings != 1 ||
		result.Documents != 1 ||
		result.Contracts != 1 ||
		result.SignedContracts != 1 ||
		result.Signers != 1 ||
		result.Activations != 1 ||
		result.RevokedCustomerSessions != 1 ||
		result.RevokedResumeTokens != 1 ||
		result.RevokedActivationTokens != 1 ||
		result.CancelledNotifications != 1 ||
		result.HiddenInAppNotifications != 1 {
		t.Fatalf("unexpected cascade result: %+v", result)
	}

	for table, id := range map[string]string{
		"proposals": proposalID,
		"proposal_acceptances": acceptanceID,
		"onboardings": onboardingID,
		"uploaded_documents": documentID,
		"contracts": contractID,
		"contract_signers": signerID,
		"activation_profiles": activationID,
	} {
		var deleted bool
		query := "select deleted_at is not null from " + table + " where id=$1"
		if err := pool.QueryRow(ctx, query, id).Scan(&deleted); err != nil {
			t.Fatalf("load deletion state for %s: %v", table, err)
		}
		if !deleted {
			t.Fatalf("%s was not soft deleted", table)
		}
	}

	for table, id := range map[string]string{
		"customer_sessions": "99999999-9999-9999-9999-999999999999",
		"customer_resume_tokens": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		"activation_access_tokens": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	} {
		var revoked bool
		query := "select revoked_at is not null from " + table + " where id=$1"
		if err := pool.QueryRow(ctx, query, id).Scan(&revoked); err != nil {
			t.Fatalf("load revocation state for %s: %v", table, err)
		}
		if !revoked {
			t.Fatalf("%s access was not revoked", table)
		}
	}

	var notificationStatus string
	if err := pool.QueryRow(ctx, `select status from notification_outbox where id='cccccccc-cccc-cccc-cccc-cccccccccccc'`).Scan(&notificationStatus); err != nil {
		t.Fatalf("load notification outbox state: %v", err)
	}
	if notificationStatus != "cancelled" {
		t.Fatalf("pending notification should be cancelled, got %s", notificationStatus)
	}

	if _, err := pool.Exec(ctx, `update proposal_acceptances set accepted_by_name='Nome adulterado' where id=$1`, acceptanceID); err == nil {
		t.Fatal("proposal acceptance evidence mutation should remain blocked after soft delete")
	}
	var acceptedByName string
	if err := pool.QueryRow(ctx, `select accepted_by_name from proposal_acceptances where id=$1`, acceptanceID).Scan(&acceptedByName); err != nil {
		t.Fatalf("load retained proposal acceptance evidence: %v", err)
	}
	if acceptedByName != "Cliente Original" {
		t.Fatalf("proposal acceptance evidence was altered: %q", acceptedByName)
	}

	var hiddenNotification bool
	if err := pool.QueryRow(ctx, `select deleted_at is not null from in_app_notifications where id='dddddddd-dddd-dddd-dddd-dddddddddddd'`).Scan(&hiddenNotification); err != nil {
		t.Fatalf("load in-app notification state: %v", err)
	}
	if !hiddenNotification {
		t.Fatal("related in-app notification should be hidden")
	}

	var status, finalizationStatus string
	var storedHash []byte
	if err := pool.QueryRow(ctx, `select status,document_sha256 from contracts where id=$1`, contractID).Scan(&status, &storedHash); err != nil {
		t.Fatalf("load retained contract evidence: %v", err)
	}
	if status != "signed" || string(storedHash) != string(documentHash) {
		t.Fatalf("signed contract evidence was altered: status=%s hash=%x", status, storedHash)
	}
	if err := pool.QueryRow(ctx, `select status from contract_finalization_jobs where contract_id=$1`, contractID).Scan(&finalizationStatus); err != nil {
		t.Fatalf("load evidence finalization job: %v", err)
	}
	if finalizationStatus != "pending" {
		t.Fatalf("evidence finalization job must be retained, got %s", finalizationStatus)
	}

	var signatureEvents int
	if err := pool.QueryRow(ctx, `select count(*) from signature_events where contract_id=$1`, contractID).Scan(&signatureEvents); err != nil {
		t.Fatalf("count retained signature events: %v", err)
	}
	if signatureEvents != 1 {
		t.Fatalf("signature evidence must remain append-only, got %d events", signatureEvents)
	}

	var signedContracts string
	var evidenceRetained bool
	if err := pool.QueryRow(ctx, `
		select metadata->>'signed_contracts',(metadata->>'evidence_retained')::boolean
		from audit_events
		where resource_id=$1 and event_type='proposal.soft_deleted'
	`, proposalID).Scan(&signedContracts, &evidenceRetained); err != nil {
		t.Fatalf("load soft-delete audit event: %v", err)
	}
	if signedContracts != "1" || !evidenceRetained {
		t.Fatalf("unexpected soft-delete audit metadata: signed=%s retained=%v", signedContracts, evidenceRetained)
	}

	if _, err := store.SoftDeleteCascade(ctx, SoftDeleteInput{
		ProposalID: proposalID,
		ActorUserID: actorID,
	}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("second soft delete should report missing active proposal, got %v", err)
	}
}
