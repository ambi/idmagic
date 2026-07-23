package usecases

// wi-160 T004.6 RED tests for the SCL scenario "Hard Quota を超過したリソース
// 作成は拒否される" (spec/contexts/tenancy.yaml), applied to active_sessions.

import (
	"context"
	"errors"
	"testing"
	"time"

	memory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func TestSessionManagerCreate_rejectsWhenHardQuotaExceeded(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC)
	store := memory.NewSessionStore()
	store.Clock = func() time.Time { return now }
	quotaRepo := tenancymemory.NewQuotaRepository()
	limit := 1
	if err := quotaRepo.SetQuota(ctx, tenancydomain.DefaultTenantID, &tenancydomain.TenantQuota{ActiveSessions: &limit}); err != nil {
		t.Fatalf("SetQuota: %v", err)
	}
	manager := NewSessionManager(store)
	manager.QuotaRepo = quotaRepo

	if _, err := manager.Create(ctx, "user-1", []string{"pwd"}, now); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	_, err := manager.Create(ctx, "user-2", []string{"pwd"}, now)
	var qErr *tenancydomain.QuotaExceededError
	if !errors.As(err, &qErr) {
		t.Fatalf("expected *domain.QuotaExceededError, got %v", err)
	}
	if qErr.Resource != tenancydomain.ResourceActiveSessions {
		t.Fatalf("unexpected resource: %s", qErr.Resource)
	}
}

func TestRevokeOwnSession_decrementsQuotaUsage(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC)
	store := memory.NewSessionStore()
	store.Clock = func() time.Time { return now }
	quotaRepo := tenancymemory.NewQuotaRepository()
	limit := 1
	if err := quotaRepo.SetQuota(ctx, tenancydomain.DefaultTenantID, &tenancydomain.TenantQuota{ActiveSessions: &limit}); err != nil {
		t.Fatalf("SetQuota: %v", err)
	}
	manager := NewSessionManager(store)
	manager.QuotaRepo = quotaRepo

	created, err := manager.Create(ctx, "user-1", []string{"pwd"}, now)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	deps := SessionDeps{Store: store, QuotaRepo: quotaRepo}
	if err := RevokeOwnSession(ctx, deps, "user-1", created.SessionID, now); err != nil {
		t.Fatalf("RevokeOwnSession: %v", err)
	}
	if _, err := manager.Create(ctx, "user-2", []string{"pwd"}, now); err != nil {
		t.Fatalf("expected create to succeed after revoke freed quota, got %v", err)
	}

	// 冪等な再 Revoke は二重減算しない。
	if err := RevokeOwnSession(ctx, deps, "user-1", created.SessionID, now); err != nil {
		t.Fatalf("idempotent re-revoke: %v", err)
	}
	if _, err := manager.Create(ctx, "user-3", []string{"pwd"}, now); err == nil {
		t.Fatal("expected idempotent re-revoke to not free a second quota slot")
	}
}
