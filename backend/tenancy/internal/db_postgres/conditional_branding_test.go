package db_postgres

import (
	"errors"
	"sync"
	"testing"

	"github.com/ambi/idmagic/backend/shared/kernel"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

//spec:covers REQ-SYSTEM-037: 同じ表現を条件に書き込む二件のうち一件だけが確定し、拒否された編集は残らない。
func TestBrandingConditionalWritesHaveOneWinner(t *testing.T) {
	db := pgtest.Require(t)
	tenant := seedQuotaTenant(t.Context(), t, db)
	repo := &TenantBrandingRepository{Pool: db}
	now := pgtest.Now()
	initial := &tenancydomain.TenantBranding{TenantID: tenant.ID, ProductName: "Initial", CreatedAt: now, UpdatedAt: now}
	if err := repo.Save(t.Context(), initial); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var writers sync.WaitGroup
	for _, name := range []string{"First", "Second"} {
		writers.Go(func() {
			<-start
			_, err := repo.Update(t.Context(), tenant.ID, func(current *tenancydomain.TenantBranding) (*tenancydomain.TenantBranding, error) {
				if current == nil || current.ProductName != "Initial" {
					return nil, kernel.ErrPreconditionFailed
				}
				current.ProductName = name
				return current, nil
			})
			results <- err
		})
	}
	close(start)
	writers.Wait()
	close(results)
	succeeded, refused := 0, 0
	for err := range results {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, kernel.ErrPreconditionFailed):
			refused++
		default:
			t.Fatalf("unexpected write error: %v", err)
		}
	}
	stored, err := repo.FindByTenant(t.Context(), tenant.ID)
	if err != nil || stored == nil || stored.ProductName == "Initial" {
		t.Fatalf("stored=%+v error=%v", stored, err)
	}
	if succeeded != 1 || refused != 1 {
		t.Fatalf("successes=%d refusals=%d, want 1 each", succeeded, refused)
	}
}
