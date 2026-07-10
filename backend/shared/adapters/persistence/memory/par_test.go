package memory

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestPARStore(t *testing.T) {
	ctx := context.Background()
	store := NewPARStore()

	t.Run("Save and Find", func(t *testing.T) {
		rec := &spec.PARRecord{
			RequestURI: "urn:ietf:params:oauth:request_uri:123",
			TenantID:   "tenant-1",
			ClientID:   "client-1",
			Parameters: map[string]string{"scope": "openid"},
			IssuedAt:   time.Now(),
			ExpiresAt:  time.Now().Add(5 * time.Minute),
			Used:       false,
		}

		err := store.Save(ctx, rec)
		if err != nil {
			t.Fatal(err)
		}

		found, err := store.Find(ctx, "urn:ietf:params:oauth:request_uri:123")
		if err != nil {
			t.Fatal(err)
		}
		if found == nil {
			t.Fatal("expected PAR record to be found")
		}
		if found.ClientID != "client-1" {
			t.Errorf("expected ClientID to be 'client-1', got %q", found.ClientID)
		}
	})

	t.Run("Consume", func(t *testing.T) {
		// 正常ケース
		consumed, err := store.Consume(ctx, "urn:ietf:params:oauth:request_uri:123")
		if err != nil {
			t.Fatal(err)
		}
		if consumed == nil {
			t.Fatal("expected to consume PAR record")
		}
		if !consumed.Used {
			t.Error("expected Used to be true after consume")
		}

		// すでに Used のレコードを再 Consume (失敗するべき)
		again, err := store.Consume(ctx, "urn:ietf:params:oauth:request_uri:123")
		if err != nil {
			t.Fatal(err)
		}
		if again != nil {
			t.Error("expected nil when consuming already used record")
		}

		// 存在しない RequestURI の Consume
		notfound, err := store.Consume(ctx, "urn:none")
		if err != nil {
			t.Fatal(err)
		}
		if notfound != nil {
			t.Error("expected nil when consuming non-existing record")
		}

		// 期限切れレコードの Consume
		expiredRec := &spec.PARRecord{
			RequestURI: "urn:expired",
			ExpiresAt:  time.Now().Add(-1 * time.Minute),
			Used:       false,
		}
		_ = store.Save(ctx, expiredRec)

		expired, err := store.Consume(ctx, "urn:expired")
		if err != nil {
			t.Fatal(err)
		}
		if expired != nil {
			t.Error("expected nil when consuming expired record")
		}
	})
}
