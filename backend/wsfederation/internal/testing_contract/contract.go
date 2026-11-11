// Package testing_contract defines the shared WS-Federation persistence contract.
package testing_contract

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/wsfederation/domain"
	"github.com/ambi/idmagic/backend/wsfederation/ports"
)

type Fixture struct {
	Repository ports.WsFedRelyingPartyRepository
	TenantA    string
	TenantB    string
	Now        time.Time
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	if got, err := f.Repository.FindByWtrealm(ctx, f.TenantA, "missing"); err != nil || got != nil {
		t.Fatalf("missing FindByWtrealm = (%+v, %v)", got, err)
	}
	rp := &domain.WsFedRelyingParty{
		TenantID: f.TenantA, Wtrealm: "urn:example:rp", DisplayName: "Example RP",
		ReplyURLs: []string{"https://rp.example/callback"}, Audience: "urn:example:audience",
		TokenType: domain.TokenTypeSAML11, CreatedAt: f.Now, UpdatedAt: f.Now,
	}
	if err := f.Repository.Save(ctx, rp); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := f.Repository.FindByWtrealm(ctx, f.TenantA, rp.Wtrealm)
	if err != nil || got == nil || got.DisplayName != rp.DisplayName || len(got.ReplyURLs) != 1 {
		t.Fatalf("FindByWtrealm = (%+v, %v), want round trip", got, err)
	}
	if leaked, err := f.Repository.FindByWtrealm(ctx, f.TenantB, rp.Wtrealm); err != nil || leaked != nil {
		t.Fatalf("other tenant FindByWtrealm = (%+v, %v)", leaked, err)
	}
	second := *rp
	second.Wtrealm = "urn:example:second"
	if err := f.Repository.Save(ctx, &second); err != nil {
		t.Fatalf("Save second: %v", err)
	}
	list, err := f.Repository.ListAll(ctx, f.TenantA)
	if err != nil || len(list) != 2 || list[0].Wtrealm != rp.Wtrealm || list[1].Wtrealm != second.Wtrealm {
		t.Fatalf("ListAll = (%+v, %v), want stable wtrealm order", list, err)
	}
	if err := f.Repository.Delete(ctx, f.TenantA, rp.Wtrealm); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if got, err := f.Repository.FindByWtrealm(ctx, f.TenantA, rp.Wtrealm); err != nil || got != nil {
		t.Fatalf("Find after Delete = (%+v, %v)", got, err)
	}
	runUpdate(t, f, second)
}

// runUpdate は、Update が最新の RP を読み直して結果を当て、失敗では変えず、同時の更新を失わないことを確かめる。
func runUpdate(t *testing.T, f Fixture, stored domain.WsFedRelyingParty) {
	t.Helper()
	ctx := context.Background()
	if err := f.Repository.Update(ctx, f.TenantA, "missing", func(rp *domain.WsFedRelyingParty) (ports.RelyingPartyOutcome, error) {
		if rp != nil {
			t.Fatalf("Update of a missing RP passed %+v", rp)
		}
		return ports.RelyingPartyUnchanged, nil
	}); err != nil {
		t.Fatalf("Update missing: %v", err)
	}
	if err := f.Repository.Update(ctx, f.TenantB, stored.Wtrealm, func(rp *domain.WsFedRelyingParty) (ports.RelyingPartyOutcome, error) {
		if rp != nil {
			t.Fatalf("Update in another tenant read %+v", rp)
		}
		return ports.RelyingPartyUnchanged, nil
	}); err != nil {
		t.Fatalf("Update other tenant: %v", err)
	}

	refused := errors.New("refused")
	if err := f.Repository.Update(ctx, f.TenantA, stored.Wtrealm, func(rp *domain.WsFedRelyingParty) (ports.RelyingPartyOutcome, error) {
		rp.DisplayName = "must not be saved"
		return ports.RelyingPartySaved, refused
	}); !errors.Is(err, refused) {
		t.Fatalf("Update with an error = %v, want the change's error", err)
	}
	if got, err := f.Repository.FindByWtrealm(ctx, f.TenantA, stored.Wtrealm); err != nil || got == nil || got.DisplayName != stored.DisplayName {
		t.Fatalf("Find after a refused Update = (%+v, %v), want unchanged", got, err)
	}

	// 同時の更新は、それぞれが直前の更新の結果を読み直すので、一つも失われない。
	const writers = 8
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for range writers {
		wg.Go(func() {
			errs <- f.Repository.Update(ctx, f.TenantA, stored.Wtrealm, func(rp *domain.WsFedRelyingParty) (ports.RelyingPartyOutcome, error) {
				rp.ReplyURLs = append(rp.ReplyURLs, "https://rp.example/callback/"+strconv.Itoa(len(rp.ReplyURLs)))
				return ports.RelyingPartySaved, nil
			})
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent Update: %v", err)
		}
	}
	got, err := f.Repository.FindByWtrealm(ctx, f.TenantA, stored.Wtrealm)
	if err != nil || got == nil || len(got.ReplyURLs) != len(stored.ReplyURLs)+writers {
		t.Fatalf("Find after concurrent Updates = (%+v, %v), want %d reply URLs", got, err, len(stored.ReplyURLs)+writers)
	}

	if err := f.Repository.Update(ctx, f.TenantA, stored.Wtrealm, func(*domain.WsFedRelyingParty) (ports.RelyingPartyOutcome, error) {
		return ports.RelyingPartyDeleted, nil
	}); err != nil {
		t.Fatalf("Update delete: %v", err)
	}
	if got, err := f.Repository.FindByWtrealm(ctx, f.TenantA, stored.Wtrealm); err != nil || got != nil {
		t.Fatalf("Find after an Update delete = (%+v, %v)", got, err)
	}
}
