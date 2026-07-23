package usecases_test

// RED-GREEN for wi-160 T004.0: CheckQuotaAndIncrement / DecrementQuota had zero
// test coverage before this change even though the SCL scenario "Hard Quota を
// 超過したリソース作成は拒否される" (spec/contexts/tenancy.yaml) requires it.

import (
	"context"
	"errors"
	"testing"

	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenancyusecases "github.com/ambi/idmagic/backend/tenancy/usecases"
)

func TestCheckQuotaAndIncrement_underLimitSucceedsAndIncrementsUsage(t *testing.T) {
	repo := tenancymemory.NewQuotaRepository()
	ctx := context.Background()
	limit := 2
	if err := repo.SetQuota(ctx, "acme", &tenancydomain.TenantQuota{Groups: &limit}); err != nil {
		t.Fatalf("SetQuota: %v", err)
	}
	if err := tenancyusecases.CheckQuotaAndIncrement(ctx, repo, "acme", tenancydomain.ResourceGroups, 1); err != nil {
		t.Fatalf("CheckQuotaAndIncrement: %v", err)
	}
	usage, err := repo.GetUsage(ctx, "acme")
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if usage.Groups != 1 {
		t.Fatalf("expected usage.Groups == 1, got %d", usage.Groups)
	}
}

func TestCheckQuotaAndIncrement_atLimitReturnsQuotaExceededError(t *testing.T) {
	repo := tenancymemory.NewQuotaRepository()
	ctx := context.Background()
	limit := 1
	if err := repo.SetQuota(ctx, "acme", &tenancydomain.TenantQuota{Groups: &limit}); err != nil {
		t.Fatalf("SetQuota: %v", err)
	}
	if err := tenancyusecases.CheckQuotaAndIncrement(ctx, repo, "acme", tenancydomain.ResourceGroups, 1); err != nil {
		t.Fatalf("first CheckQuotaAndIncrement: %v", err)
	}
	err := tenancyusecases.CheckQuotaAndIncrement(ctx, repo, "acme", tenancydomain.ResourceGroups, 1)
	if err == nil {
		t.Fatal("expected QuotaExceededError, got nil")
	}
	var qErr *tenancydomain.QuotaExceededError
	if !errors.As(err, &qErr) {
		t.Fatalf("expected *domain.QuotaExceededError, got %T: %v", err, err)
	}
	if qErr.Resource != tenancydomain.ResourceGroups || qErr.TenantID != "acme" {
		t.Fatalf("unexpected error payload: %+v", qErr)
	}
	usage, err := repo.GetUsage(ctx, "acme")
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if usage.Groups != 1 {
		t.Fatalf("expected usage.Groups to stay at 1 after rejected increment, got %d", usage.Groups)
	}
}

func TestDecrementQuota_reducesUsage(t *testing.T) {
	repo := tenancymemory.NewQuotaRepository()
	ctx := context.Background()
	limit := 5
	if err := repo.SetQuota(ctx, "acme", &tenancydomain.TenantQuota{Users: &limit}); err != nil {
		t.Fatalf("SetQuota: %v", err)
	}
	if err := tenancyusecases.CheckQuotaAndIncrement(ctx, repo, "acme", tenancydomain.ResourceUsers, 3); err != nil {
		t.Fatalf("CheckQuotaAndIncrement: %v", err)
	}
	if err := tenancyusecases.DecrementQuota(ctx, repo, "acme", tenancydomain.ResourceUsers, 2); err != nil {
		t.Fatalf("DecrementQuota: %v", err)
	}
	usage, err := repo.GetUsage(ctx, "acme")
	if err != nil {
		t.Fatalf("GetUsage: %v", err)
	}
	if usage.Users != 1 {
		t.Fatalf("expected usage.Users == 1, got %d", usage.Users)
	}
}
