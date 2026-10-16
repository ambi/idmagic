// Package testing_contract defines the persistence contract for recovery codes.
package testing_contract

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/recovery/domain"
	"github.com/ambi/idmagic/backend/authentication/recovery/ports"
)

type Fixture struct {
	Repository ports.RecoveryCodeRepository
	UserID     string
	Now        time.Time
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	if got, err := f.Repository.ListBySub(ctx, f.UserID); err != nil || len(got) != 0 {
		t.Fatalf("empty ListBySub = (%+v, %v)", got, err)
	}
	codes := []*domain.RecoveryCode{
		{UserID: f.UserID, CodeHash: "hash-a", GeneratedAt: f.Now},
		{UserID: f.UserID, CodeHash: "hash-b", GeneratedAt: f.Now.Add(time.Second)},
	}
	if err := f.Repository.ReplaceAll(ctx, f.UserID, codes); err != nil {
		t.Fatalf("ReplaceAll: %v", err)
	}
	got, err := f.Repository.ListBySub(ctx, f.UserID)
	if err != nil || len(got) != 2 {
		t.Fatalf("ListBySub = (%+v, %v), want two codes", got, err)
	}
	consumed, err := f.Repository.MarkConsumed(ctx, f.UserID, "hash-a", f.Now.Add(time.Minute))
	if err != nil || !consumed {
		t.Fatalf("first MarkConsumed = (%v, %v), want true", consumed, err)
	}
	consumed, err = f.Repository.MarkConsumed(ctx, f.UserID, "hash-a", f.Now.Add(2*time.Minute))
	if err != nil || consumed {
		t.Fatalf("second MarkConsumed = (%v, %v), want false", consumed, err)
	}
	if err := f.Repository.ReplaceAll(ctx, f.UserID, codes[1:]); err != nil {
		t.Fatalf("replacement ReplaceAll: %v", err)
	}
	if got, err := f.Repository.ListBySub(ctx, f.UserID); err != nil || len(got) != 1 || got[0].CodeHash != "hash-b" {
		t.Fatalf("replaced codes = (%+v, %v)", got, err)
	}
	if err := f.Repository.DeleteAllForSub(ctx, f.UserID); err != nil {
		t.Fatalf("DeleteAllForSub: %v", err)
	}
	if got, err := f.Repository.ListBySub(ctx, f.UserID); err != nil || len(got) != 0 {
		t.Fatalf("codes after delete = (%+v, %v)", got, err)
	}
}
