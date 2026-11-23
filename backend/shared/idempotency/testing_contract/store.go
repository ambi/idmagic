// Package testing_contract は受付記録の保存に必要な、メモリと PostgreSQL の共通契約を検査する。
package testing_contract

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/idempotency"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func Run(t *testing.T, newStore func() idempotency.Store, tenantID, otherTenantID string) {
	t.Helper()
	now := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
	candidate := func() idempotency.Record {
		t.Helper()
		id, err := spec.NewUUIDv4()
		if err != nil {
			t.Fatal(err)
		}
		return idempotency.Record{ID: id, TenantID: tenantID, PrincipalID: "admin", KeyHash: id, Fingerprint: "fingerprint", CreatedAt: now, ExpiresAt: now.Add(idempotency.Retention)}
	}
	t.Run("one concurrent reservation", func(t *testing.T) {
		store := newStore()
		base := candidate()
		var acquired atomic.Int32
		var wait sync.WaitGroup
		ids := make(chan string, 20)
		for range 20 {
			request := candidate()
			request.KeyHash = base.KeyHash
			wait.Go(func() {
				record, claimed, err := store.Claim(t.Context(), request, now)
				if err != nil {
					t.Error(err)
					return
				}
				if claimed {
					acquired.Add(1)
				}
				ids <- record.ID
			})
		}
		wait.Wait()
		close(ids)
		if acquired.Load() != 1 {
			t.Fatalf("acquired=%d, want 1", acquired.Load())
		}
		winner := ""
		for id := range ids {
			if winner == "" {
				winner = id
			}
			if id != winner {
				t.Errorf("reservation=%s, want %s", id, winner)
			}
		}
	})
	t.Run("tenant and principal isolation", func(t *testing.T) {
		store := newStore()
		base := candidate()
		for _, scope := range []struct{ tenant, principal string }{{tenantID, "admin"}, {tenantID, "other-admin"}, {otherTenantID, "admin"}} {
			request := candidate()
			request.KeyHash, request.TenantID, request.PrincipalID = base.KeyHash, scope.tenant, scope.principal
			record, acquired, err := store.Claim(t.Context(), request, now)
			if err != nil || !acquired || record.ID != request.ID {
				t.Fatalf("scope=%+v record=%+v acquired=%v error=%v", scope, record, acquired, err)
			}
		}
	})
	t.Run("expiry and stale completion", func(t *testing.T) {
		store := newStore()
		first := candidate()
		_, _, err := store.Claim(t.Context(), first, now)
		if err != nil {
			t.Fatal(err)
		}
		second := candidate()
		second.KeyHash = first.KeyHash
		if _, acquired, err := store.Claim(t.Context(), second, first.ExpiresAt.Add(-time.Microsecond)); err != nil || acquired {
			t.Fatalf("before expiry acquired=%v error=%v", acquired, err)
		}
		second.CreatedAt, second.ExpiresAt = first.ExpiresAt, first.ExpiresAt.Add(idempotency.Retention)
		if _, acquired, err := store.Claim(t.Context(), second, first.ExpiresAt); err != nil || !acquired {
			t.Fatalf("at expiry acquired=%v error=%v", acquired, err)
		}
		first.KeyVersion, first.Ciphertext = 1, []byte("stale")
		if err := store.Complete(t.Context(), first); err == nil {
			t.Fatal("stale request overwrote a new reservation")
		}
		second.KeyVersion, second.Ciphertext = 2, []byte("encrypted-response")
		if err := store.Complete(t.Context(), second); err != nil {
			t.Fatal(err)
		}
		if err := store.Complete(t.Context(), second); err == nil {
			t.Fatal("completed response was overwritten")
		}
		found, acquired, err := store.Claim(t.Context(), second, second.CreatedAt)
		if err != nil || acquired || string(found.Ciphertext) != "encrypted-response" {
			t.Fatalf("record=%+v acquired=%v error=%v", found, acquired, err)
		}
		if err := store.PurgeExpired(t.Context(), second.ExpiresAt.Add(-time.Microsecond)); err != nil {
			t.Fatal(err)
		}
		found, acquired, err = store.Claim(t.Context(), second, second.CreatedAt)
		if err != nil || acquired || found.ID != second.ID {
			t.Fatalf("live record lost: acquired=%v error=%v", acquired, err)
		}
		if err := store.PurgeExpired(t.Context(), second.ExpiresAt); err != nil {
			t.Fatal(err)
		}
		third := candidate()
		// 過去の受付時刻で照会し、期限切れの置換では隠れる削除の欠落を検出する。
		third.KeyHash = first.KeyHash
		if _, acquired, err := store.Claim(t.Context(), third, third.CreatedAt); err != nil || !acquired {
			t.Fatalf("purged record acquired=%v error=%v", acquired, err)
		}
	})
	t.Run("reencryption excludes disabled keys before limiting", func(t *testing.T) {
		store := newStore()
		for _, version := range []int32{1, 2, 3} {
			record := candidate()
			if _, _, err := store.Claim(t.Context(), record, now); err != nil {
				t.Fatal(err)
			}
			record.KeyVersion, record.Ciphertext = version, []byte("ciphertext")
			if err := store.Complete(t.Context(), record); err != nil {
				t.Fatal(err)
			}
		}
		rows, err := store.ListPending(t.Context(), tenantID, 3, 1, []int32{1})
		if err != nil || len(rows) != 1 || rows[0].KeyVersion != 2 {
			t.Fatalf("rows=%+v error=%v", rows, err)
		}
		if count, err := store.PendingCount(t.Context(), tenantID, 3); err != nil || count < 2 {
			t.Fatalf("pending=%d error=%v", count, err)
		}
		row := rows[0]
		row.KeyVersion, row.Ciphertext = 3, []byte("new-ciphertext")
		if changed, err := store.ReplaceCipher(t.Context(), row, 2); err != nil || !changed {
			t.Fatalf("reencryption changed=%v error=%v", changed, err)
		}
		if changed, err := store.ReplaceCipher(t.Context(), row, 2); err != nil || changed {
			t.Fatalf("stale reencryption changed=%v error=%v", changed, err)
		}
	})
}
