package usecases

import (
	"errors"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"

	"github.com/ambi/idmagic/backend/oauth2/domain"

	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestAdminConsents(t *testing.T) {
	ctx := tenantContext(tenancydomain.DefaultTenantID)
	consentRepo := oauth2memory.NewConsentRepository()
	var emitted []spec.DomainEvent
	emitFunc := func(e spec.DomainEvent) {
		emitted = append(emitted, e)
	}

	deps := ConsentDeps{
		ConsentRepo: consentRepo,
		Emit:        emitFunc,
	}

	now := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)

	t.Run("ListConsents", func(t *testing.T) {
		// 初期は空
		list, err := ListConsents(ctx, deps)
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 0 {
			t.Errorf("expected empty consent list, got %d", len(list))
		}

		// 1つ保存
		consent := &domain.Consent{
			UserID:   "user-1",
			ClientID: "client-1",
			Scopes:   []string{"openid"},
		}
		_ = consentRepo.Save(ctx, tenancydomain.DefaultTenantID, consent)

		list, err = ListConsents(ctx, deps)
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != 1 {
			t.Fatalf("expected 1 consent, got %d", len(list))
		}
		if list[0].UserID != "user-1" {
			t.Errorf("expected UserID 'user-1', got %q", list[0].UserID)
		}
	})

	t.Run("GetConsent", func(t *testing.T) {
		// 正常
		consent, err := GetConsent(ctx, deps, "user-1", "client-1")
		if err != nil {
			t.Fatal(err)
		}
		if consent.UserID != "user-1" || consent.ClientID != "client-1" {
			t.Errorf("mismatched consent values: %+v", consent)
		}

		// 存在しない
		_, err = GetConsent(ctx, deps, "user-none", "client-none")
		if !errors.Is(err, ErrConsentNotFound) {
			t.Errorf("expected ErrConsentNotFound, got %v", err)
		}
	})

	t.Run("RevokeConsent", func(t *testing.T) {
		emitted = nil
		// 正常Revoke
		err := RevokeConsent(ctx, deps, "admin-1", "user-1", "client-1", now)
		if err != nil {
			t.Fatal(err)
		}

		// 状態が Revoked に変化していることの確認
		consent, err := GetConsent(ctx, deps, "user-1", "client-1")
		if err != nil {
			t.Fatal(err)
		}
		if consent.State != domain.ConsentRevoked {
			t.Errorf("expected state to be ConsentRevoked, got %v", consent.State)
		}
		if consent.RevokedAt == nil {
			t.Error("expected RevokedAt to be non-nil")
		}

		// イベントが Emit されたこと
		if len(emitted) != 1 {
			t.Fatalf("expected 1 event, got %d", len(emitted))
		}
		ev, ok := emitted[0].(*domain.ConsentRevokedEvent)
		if !ok {
			t.Fatalf("expected ConsentRevokedEvent, got %T", emitted[0])
		}
		if ev.ActorUserID != "admin-1" || ev.UserID != "user-1" || ev.ClientID != "client-1" {
			t.Errorf("unexpected event content: %+v", ev)
		}

		// 存在しないConsentのRevokeはErrConsentNotFoundになる
		err = RevokeConsent(ctx, deps, "admin-1", "user-none", "client-none", now)
		if !errors.Is(err, ErrConsentNotFound) {
			t.Errorf("expected ErrConsentNotFound for non-existing consent, got %v", err)
		}
	})
}
