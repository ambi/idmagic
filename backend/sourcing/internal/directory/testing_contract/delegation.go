package testing_contract

import (
	"errors"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/ports"
)

type PasswordFixture struct {
	Store                                                  ports.PasswordChallengeStore
	TenantID, OtherTenantID, ConnectorID, OtherConnectorID string
}

// RunPasswordChallenges は SQL とメモリで同じ配送、単回性と期限を観測する。
func RunPasswordChallenges(t *testing.T, f PasswordFixture) {
	t.Helper()
	ctx := t.Context()
	now := time.Now().UTC().Truncate(time.Microsecond)
	makeItem := func() domain.PasswordChallenge {
		return domain.PasswordChallenge{ID: uuid.NewV4().String(), TenantID: f.TenantID, ConnectorID: f.ConnectorID, Revision: 7, KeyID: "encryption-key", ExpiresAt: now.Add(5 * time.Second), EncryptedRequest: "encrypted-only"}
	}
	item := makeItem()
	if err := f.Store.Enqueue(ctx, item, now); err != nil {
		t.Fatal(err)
	}
	for _, attempt := range []struct {
		tenant, connector string
		revision          int64
	}{{f.OtherTenantID, f.ConnectorID, 7}, {f.TenantID, f.OtherConnectorID, 7}, {f.TenantID, f.ConnectorID, 8}} {
		got, err := f.Store.Claim(ctx, attempt.tenant, attempt.connector, attempt.revision, now)
		if err != nil || got != nil {
			t.Fatalf("invalid claim: %+v %v", got, err)
		}
	}
	if err := f.Store.Complete(ctx, f.TenantID, f.ConnectorID, item.ID, 7, domain.PasswordAccepted, now); !errors.Is(err, ports.ErrDelegationUnavailable) {
		t.Fatalf("unclaimed completion: %v", err)
	}
	var wg sync.WaitGroup
	claimed := make(chan *domain.PasswordChallenge, 8)
	for range 8 {
		wg.Go(func() {
			got, err := f.Store.Claim(ctx, f.TenantID, f.ConnectorID, 7, now)
			if err != nil {
				t.Error(err)
			}
			if got != nil {
				claimed <- got
			}
		})
	}
	wg.Wait()
	close(claimed)
	if len(claimed) != 1 {
		t.Fatalf("claim count = %d", len(claimed))
	}
	got := <-claimed
	if got.ID != item.ID || got.EncryptedRequest != "encrypted-only" {
		t.Fatalf("claim = %+v", got)
	}
	stored, err := f.Store.Find(ctx, f.TenantID, item.ID)
	if err != nil || stored == nil || !stored.Delivered || stored.EncryptedRequest != "" {
		t.Fatalf("ciphertext retained: %+v %v", stored, err)
	}
	for _, r := range []struct {
		tenant, connector string
		revision          int64
		result            domain.PasswordResult
		at                time.Time
	}{
		{f.OtherTenantID, f.ConnectorID, 7, domain.PasswordAccepted, now},
		{f.TenantID, f.OtherConnectorID, 7, domain.PasswordAccepted, now},
		{f.TenantID, f.ConnectorID, 8, domain.PasswordAccepted, now},
		{f.TenantID, f.ConnectorID, 7, "unknown", now},
		{f.TenantID, f.ConnectorID, 7, domain.PasswordAccepted, item.ExpiresAt},
		{f.TenantID, f.ConnectorID, 7, domain.PasswordAccepted, item.ExpiresAt.Add(time.Microsecond)},
	} {
		if err := f.Store.Complete(ctx, r.tenant, r.connector, item.ID, r.revision, r.result, r.at); !errors.Is(err, ports.ErrDelegationUnavailable) {
			t.Fatalf("invalid completion %+v: %v", r, err)
		}
	}
	for range 2 {
		if err := f.Store.Complete(ctx, f.TenantID, f.ConnectorID, item.ID, 7, domain.PasswordAccepted, item.ExpiresAt.Add(-time.Microsecond)); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.Store.Complete(ctx, f.TenantID, f.ConnectorID, item.ID, 7, domain.PasswordInvalid, now); !errors.Is(err, ports.ErrDelegationUnavailable) {
		t.Fatalf("conflicting completion: %v", err)
	}
	if err := f.Store.Sweep(ctx, item.ExpiresAt); err != nil {
		t.Fatal(err)
	}
	if got, err := f.Store.Find(ctx, f.TenantID, item.ID); err != nil || got != nil {
		t.Fatalf("expired item: %+v %v", got, err)
	}
	for range domain.MaxPasswordChallenges {
		if err := f.Store.Enqueue(ctx, makeItem(), now); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.Store.Enqueue(ctx, makeItem(), now); !errors.Is(err, ports.ErrDelegationUnavailable) {
		t.Fatalf("queue limit: %v", err)
	}
}
