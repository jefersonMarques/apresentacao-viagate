package notifications

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestInAppNotificationScopePreferences(t *testing.T) {
	if os.Getenv("RUN_DB_INTEGRATION") != "1" {
		t.Skip("database integration test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer pool.Close()

	stamp := time.Now().UTC().UnixNano()
	ownerID := createNotificationTestUser(t, ctx, pool, fmt.Sprintf("notification-owner-%d@example.test", stamp), "Owner")
	otherID := createNotificationTestUser(t, ctx, pool, fmt.Sprintf("notification-other-%d@example.test", stamp), "Other")
	defer func() {
		_, _ = pool.Exec(ctx, `delete from users where id in ($1,$2)`, ownerID, otherID)
	}()

	if _, err := pool.Exec(ctx, `
		insert into user_permission_overrides(user_id,permission_id,allowed)
		select $1,p.id,true from permissions p where p.code='notification.receive_others'
	`, otherID); err != nil {
		t.Fatalf("grant receive-others permission: %v", err)
	}

	store := NewInAppStore(pool)
	publish := func(key string) {
		t.Helper()
		if err := store.Publish(ctx, Event{
			OwnerUserID: ownerID,
			EventType: "proposal.opened",
			Title: "Proposta aberta",
			ResourceType: "proposal",
			DedupeKey: key,
		}); err != nil {
			t.Fatalf("publish %s: %v", key, err)
		}
	}
	count := func(userID, key string) int {
		t.Helper()
		var value int
		if err := pool.QueryRow(ctx, `
			select count(*) from in_app_notifications
			where recipient_user_id=$1 and dedupe_key=$2
		`, userID, key).Scan(&value); err != nil {
			t.Fatalf("count notifications: %v", err)
		}
		return value
	}

	publish("scope-test:own-default")
	if got := count(ownerID, "scope-test:own-default"); got != 1 {
		t.Fatalf("owner should receive own event by default, got %d", got)
	}
	if got := count(otherID, "scope-test:own-default"); got != 0 {
		t.Fatalf("other user should not receive event by default, got %d", got)
	}

	if _, err := pool.Exec(ctx, `
		insert into user_notification_preferences(user_id,event_type,scope,enabled)
		values($1,'proposal.opened','all',true)
		on conflict(user_id,event_type,scope) do update set enabled=excluded.enabled,updated_at=now()
	`, otherID); err != nil {
		t.Fatalf("enable all-events preference: %v", err)
	}
	publish("scope-test:all-enabled")
	if got := count(otherID, "scope-test:all-enabled"); got != 1 {
		t.Fatalf("other user should receive event when all scope is enabled, got %d", got)
	}

	if _, err := pool.Exec(ctx, `
		insert into user_notification_preferences(user_id,event_type,scope,enabled)
		values($1,'proposal.opened','own',false)
		on conflict(user_id,event_type,scope) do update set enabled=excluded.enabled,updated_at=now()
	`, ownerID); err != nil {
		t.Fatalf("disable owner preference: %v", err)
	}
	publish("scope-test:own-disabled")
	if got := count(ownerID, "scope-test:own-disabled"); got != 0 {
		t.Fatalf("owner should not receive disabled own event, got %d", got)
	}
	if got := count(otherID, "scope-test:own-disabled"); got != 1 {
		t.Fatalf("other user with all scope enabled should still receive event, got %d", got)
	}
}

func createNotificationTestUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, email, name string) string {
	t.Helper()
	var userID string
	if err := pool.QueryRow(ctx, `
		insert into users(email,name,status) values($1,$2,'active') returning id::text
	`, email, name).Scan(&userID); err != nil {
		t.Fatalf("create notification test user: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		insert into user_roles(user_id,role_id)
		select $1,r.id from roles r where r.code='user'
	`, userID); err != nil {
		t.Fatalf("assign user role: %v", err)
	}
	return userID
}
