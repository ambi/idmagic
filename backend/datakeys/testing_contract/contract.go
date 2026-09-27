// Package testing_contract defines the shared data-key lifecycle contract.
package testing_contract

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/datakeys/domain"
	"github.com/ambi/idmagic/backend/datakeys/ports"
)

type Fixture struct {
	Repository ports.DataKeyRepository
	TenantID   string
	Now        time.Time
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	v1, err := f.Repository.Bootstrap(ctx, f.TenantID, []byte("wrapped-1"), "master-1", f.Now)
	if err != nil || v1 == nil || v1.Version != 1 || v1.Status != domain.DataKeyStatusActive {
		t.Fatalf("Bootstrap = (%+v, %v)", v1, err)
	}
	if _, err := f.Repository.Bootstrap(ctx, f.TenantID, []byte("wrapped-again"), "master-1", f.Now); !errors.Is(err, domain.ErrDataKeyAlreadyBootstrapped) {
		t.Fatalf("second Bootstrap error = %v", err)
	}
	v2, previous, err := f.Repository.Rotate(ctx, f.TenantID, []byte("wrapped-2"), "master-1", f.Now.Add(time.Minute))
	if err != nil || v2 == nil || previous == nil || v2.Version != 2 || previous.Status != domain.DataKeyStatusRetiring {
		t.Fatalf("Rotate = (%+v, %+v, %v)", v2, previous, err)
	}
	if active, err := f.Repository.FindActive(ctx, f.TenantID); err != nil || active == nil || active.Version != 2 {
		t.Fatalf("FindActive = (%+v, %v)", active, err)
	}
	if _, err := f.Repository.Disable(ctx, f.TenantID, 2, f.Now); !errors.Is(err, domain.ErrDataKeyIsActive) {
		t.Fatalf("Disable active error = %v", err)
	}
	if _, err := f.Repository.Disable(ctx, f.TenantID, 1, f.Now.Add(2*time.Minute)); err != nil {
		t.Fatalf("Disable retiring: %v", err)
	}
	destroyed, err := f.Repository.Destroy(ctx, f.TenantID, 1, f.Now.Add(3*time.Minute))
	if err != nil || destroyed == nil || destroyed.Status != domain.DataKeyStatusDestroyed || destroyed.WrappedDEK != nil {
		t.Fatalf("Destroy = (%+v, %v)", destroyed, err)
	}
	all, err := f.Repository.ListAll(ctx, f.TenantID)
	if err != nil || len(all) != 2 || all[0].Version != 2 || all[1].Version != 1 {
		t.Fatalf("ListAll = (%+v, %v)", all, err)
	}
}
