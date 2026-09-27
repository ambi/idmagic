// Package testing_contract defines the shared authentication persistence contract.
package testing_contract

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/ports"
)

type Fixture struct {
	Store   ports.AuthEventBucketStore
	TenantA string
	TenantB string
	Now     time.Time
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	t.Run("aggregate and isolate tenants", func(t *testing.T) {
		f := newFixture(t)
		ctx := context.Background()
		first, err := f.Store.Record(ctx, ports.AuthEventBucketFailedLogin, f.TenantA, "key-a", f.Now)
		if err != nil || !first.FirstInWindow || first.Bucket.Count != 1 {
			t.Fatalf("first Record = (%+v, %v)", first, err)
		}
		second, err := f.Store.Record(ctx, ports.AuthEventBucketFailedLogin, f.TenantA, "key-a", f.Now.Add(time.Second))
		if err != nil || second.FirstInWindow || second.Bucket.Count != 2 {
			t.Fatalf("second Record = (%+v, %v)", second, err)
		}
		if _, err := f.Store.Record(ctx, ports.AuthEventBucketFailedLogin, f.TenantB, "key-a", f.Now); err != nil {
			t.Fatalf("other tenant Record: %v", err)
		}
		for tenantID, want := range map[string]int{f.TenantA: 2, f.TenantB: 1} {
			got, err := f.Store.List(ctx, tenantID, time.Time{}, "", 10)
			if err != nil || len(got) != 1 || got[0].Count != want {
				t.Fatalf("List(%q) = (%+v, %v), want count %d", tenantID, got, err, want)
			}
		}
	})

	t.Run("window ordering and keyset continuation", func(t *testing.T) {
		f := newFixture(t)
		ctx := context.Background()
		for i := range 3 {
			at := f.Now.Add(time.Duration(i) * ports.AuthEventBucketWindow)
			if _, err := f.Store.Record(ctx, ports.AuthEventBucketFailedLogin, f.TenantA, "key", at); err != nil {
				t.Fatalf("Record #%d: %v", i, err)
			}
		}
		first, err := f.Store.List(ctx, f.TenantA, time.Time{}, "", 2)
		if err != nil || len(first) != 2 || !first[0].WindowStart.After(first[1].WindowStart) {
			t.Fatalf("first page = (%+v, %v)", first, err)
		}
		cursor := first[len(first)-1]
		next, err := f.Store.List(ctx, f.TenantA, cursor.WindowStart, string(cursor.Kind)+"|"+cursor.KeyHash, 2)
		if err != nil || len(next) != 1 || !next[0].WindowStart.Before(cursor.WindowStart) {
			t.Fatalf("next page = (%+v, %v)", next, err)
		}
	})
}
