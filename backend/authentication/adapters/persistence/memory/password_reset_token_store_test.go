package memory

import (
	"context"
	"sync"
	"testing"
	"time"

	authnports "github.com/ambi/idmagic/backend/authentication/ports"
)

func TestPasswordResetTokenStoreInvalidatesPreviousTokenForSubject(t *testing.T) {
	store := NewPasswordResetTokenStore()
	now := time.Now().UTC()
	for _, hash := range []string{"old", "new"} {
		if err := store.Save(context.Background(), authnports.PasswordResetTokenRecord{
			Sub: "user", TokenHash: hash, CreatedAt: now, ExpiresAt: now.Add(time.Minute),
		}); err != nil {
			t.Fatal(err)
		}
	}
	old, _ := store.Consume(context.Background(), "old", now)
	next, _ := store.Consume(context.Background(), "new", now)
	if old != nil || next == nil {
		t.Fatalf("old=%#v new=%#v", old, next)
	}
}

func TestPasswordResetTokenStoreConsumeSucceedsOnceConcurrently(t *testing.T) {
	store := NewPasswordResetTokenStore()
	now := time.Now().UTC()
	if err := store.Save(context.Background(), authnports.PasswordResetTokenRecord{
		Sub: "user", TokenHash: "token", CreatedAt: now, ExpiresAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan *authnports.PasswordResetTokenRecord, 2)
	for range 2 {
		wg.Go(func() {
			record, _ := store.Consume(context.Background(), "token", now)
			results <- record
		})
	}
	wg.Wait()
	close(results)
	successes := 0
	for record := range results {
		if record != nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successful consumes=%d, want 1", successes)
	}

	// 存在しないトークンの Consume (nil, nil)
	notFound, err := store.Consume(context.Background(), "non-existent-hash", now)
	if err != nil {
		t.Fatal(err)
	}
	if notFound != nil {
		t.Error("expected nil for non-existing token")
	}

	// 期限切れトークンの Consume (nil, nil)
	expiredRecord := authnports.PasswordResetTokenRecord{
		Sub: "user-exp", TokenHash: "token-exp", CreatedAt: now, ExpiresAt: now.Add(-1 * time.Minute),
	}
	_ = store.Save(context.Background(), expiredRecord)
	expired, err := store.Consume(context.Background(), "token-exp", now)
	if err != nil {
		t.Fatal(err)
	}
	if expired != nil {
		t.Error("expected nil for expired token")
	}
}
