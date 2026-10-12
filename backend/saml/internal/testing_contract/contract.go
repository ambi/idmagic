// Package testing_contract defines the shared SAML persistence contract.
package testing_contract

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/saml/ports"
)

type Fixture struct {
	Replay  ports.AuthnRequestReplayStore
	TenantA string
	TenantB string
	Now     time.Time
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	if ok, err := f.Replay.RecordIfNew(ctx, f.TenantA, "sp", "request", time.Minute, f.Now); err != nil || !ok {
		t.Fatalf("first RecordIfNew = (%v, %v)", ok, err)
	}
	if ok, err := f.Replay.RecordIfNew(ctx, f.TenantA, "sp", "request", time.Minute, f.Now); err != nil || ok {
		t.Fatalf("duplicate RecordIfNew = (%v, %v)", ok, err)
	}
	if ok, err := f.Replay.RecordIfNew(ctx, f.TenantB, "sp", "request", time.Minute, f.Now); err != nil || !ok {
		t.Fatalf("other tenant RecordIfNew = (%v, %v)", ok, err)
	}
	if ok, err := f.Replay.RecordIfNew(ctx, f.TenantA, "sp", "request", time.Minute, f.Now.Add(2*time.Minute)); err != nil || !ok {
		t.Fatalf("post-expiry RecordIfNew = (%v, %v)", ok, err)
	}

	var wait sync.WaitGroup
	var mu sync.Mutex
	winners := 0
	for range 8 {
		wait.Go(func() {
			ok, err := f.Replay.RecordIfNew(ctx, f.TenantA, "sp", "parallel", time.Minute, f.Now)
			if err != nil {
				t.Errorf("parallel RecordIfNew: %v", err)
				return
			}
			if ok {
				mu.Lock()
				winners++
				mu.Unlock()
			}
		})
	}
	wait.Wait()
	if winners != 1 {
		t.Fatalf("parallel winners = %d, want 1", winners)
	}
}
