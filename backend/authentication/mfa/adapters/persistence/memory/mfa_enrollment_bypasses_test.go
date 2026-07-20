package memory

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/mfa/domain"
)

func TestMfaEnrollmentBypassConsumeIsSingleUse(t *testing.T) {
	repo := NewMfaEnrollmentBypassRepository()
	now := time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC)
	bypass := &domain.MfaEnrollmentBypass{ID: "id", TenantID: "tenant", UserID: "user", IssuedBy: "admin", IssuedAt: now, ExpiresAt: now.Add(time.Minute)}
	if err := repo.Save(context.Background(), bypass); err != nil {
		t.Fatal(err)
	}
	first, err := repo.ConsumeActive(context.Background(), "tenant", "user", now.Add(time.Second))
	if err != nil || first == nil || first.ConsumedAt == nil {
		t.Fatalf("first consume = %#v, %v", first, err)
	}
	second, err := repo.ConsumeActive(context.Background(), "tenant", "user", now.Add(2*time.Second))
	if err != nil || second != nil {
		t.Fatalf("second consume = %#v, %v", second, err)
	}
}

func TestMfaEnrollmentBypassExpiresOnce(t *testing.T) {
	repo := NewMfaEnrollmentBypassRepository()
	now := time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC)
	bypass := &domain.MfaEnrollmentBypass{ID: "expired", TenantID: "tenant", UserID: "user", IssuedBy: "admin", IssuedAt: now, ExpiresAt: now.Add(time.Minute)}
	if err := repo.Save(context.Background(), bypass); err != nil {
		t.Fatal(err)
	}
	first, err := repo.ExpireOpen(context.Background(), "tenant", "user", now.Add(2*time.Minute))
	if err != nil || first == nil || first.ExpiredAt == nil {
		t.Fatalf("expire=%#v err=%v", first, err)
	}
	second, err := repo.ExpireOpen(context.Background(), "tenant", "user", now.Add(3*time.Minute))
	if err != nil || second != nil {
		t.Fatalf("second expire=%#v err=%v", second, err)
	}
}
