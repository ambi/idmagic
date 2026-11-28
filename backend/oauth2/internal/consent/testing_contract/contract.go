// Package testing_contract defines the shared consent repository contract.
package testing_contract

import (
	"context"
	"testing"
	"time"

	consentports "github.com/ambi/idmagic/backend/oauth2/consent/ports"
	consentdomain "github.com/ambi/idmagic/backend/oauth2/internal/consent/domain"
)

type Fixture struct {
	Repository consentports.ConsentRepository
	TenantID   string
	Consent    *consentdomain.Consent
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	if err := f.Repository.Save(ctx, f.TenantID, f.Consent); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := f.Repository.Find(ctx, f.TenantID, f.Consent.UserID, f.Consent.ClientID)
	if err != nil || got == nil || got.State != consentdomain.ConsentGranted {
		t.Fatalf("Find = (%+v, %v)", got, err)
	}
	all, err := f.Repository.FindAll(ctx, f.TenantID)
	if err != nil || len(all) != 1 {
		t.Fatalf("FindAll = (%+v, %v)", all, err)
	}
	revokedAt := f.Consent.GrantedAt.Add(time.Minute)
	if err := f.Repository.Revoke(ctx, f.TenantID, f.Consent.UserID, f.Consent.ClientID, revokedAt); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	// 撤回済みの同意を撤回し直しても、最初の撤回の時刻を保つ。
	if err := f.Repository.Revoke(ctx, f.TenantID, f.Consent.UserID, f.Consent.ClientID, revokedAt.Add(time.Minute)); err != nil {
		t.Fatalf("Revoke again: %v", err)
	}
	got, err = f.Repository.Find(ctx, f.TenantID, f.Consent.UserID, f.Consent.ClientID)
	if err != nil || got == nil || got.State != consentdomain.ConsentRevoked || got.RevokedAt == nil || !got.RevokedAt.Equal(revokedAt) {
		t.Fatalf("Find after revoke = (%+v, %v), want revoked at %v", got, err, revokedAt)
	}
	// 撤回した行は撤回の時点で consents の使用量から減らしてあるので、削除の件数に数えない。
	if unrevoked, err := f.Repository.DeleteAllForSub(ctx, f.Consent.UserID); err != nil || unrevoked != 0 {
		t.Fatalf("DeleteAllForSub of a revoked consent = (%d, %v), want (0, nil)", unrevoked, err)
	}
	if got, err := f.Repository.Find(ctx, f.TenantID, f.Consent.UserID, f.Consent.ClientID); err != nil || got != nil {
		t.Fatalf("Find after DeleteAllForSub = (%+v, %v), want (nil, nil)", got, err)
	}
	// 付与済みの行は使用量を占めているので数える。
	if err := f.Repository.Save(ctx, f.TenantID, f.Consent); err != nil {
		t.Fatalf("Save again: %v", err)
	}
	if unrevoked, err := f.Repository.DeleteAllForSub(ctx, f.Consent.UserID); err != nil || unrevoked != 1 {
		t.Fatalf("DeleteAllForSub of a granted consent = (%d, %v), want (1, nil)", unrevoked, err)
	}
	// 期限切れは保存した状態ではない。実時計で期限を過ぎた同意も Granted のまま読み戻し、
	// 呼び出し側が StateAt で判定する。
	expired := *f.Consent
	expired.GrantedAt = f.Consent.GrantedAt.Add(-2 * time.Hour)
	expired.ExpiresAt = f.Consent.GrantedAt.Add(-time.Hour)
	if err := f.Repository.Save(ctx, f.TenantID, &expired); err != nil {
		t.Fatalf("Save expired: %v", err)
	}
	got, err = f.Repository.Find(ctx, f.TenantID, expired.UserID, expired.ClientID)
	if err != nil || got == nil || got.State != consentdomain.ConsentGranted || got.StateAt(f.Consent.GrantedAt) != consentdomain.ConsentExpired {
		t.Fatalf("Find of an expired consent = (%+v, %v), want stored Granted judged Expired", got, err)
	}
	if unrevoked, err := f.Repository.DeleteAllForSub(ctx, f.Consent.UserID); err != nil || unrevoked != 1 {
		t.Fatalf("DeleteAllForSub of an expired consent = (%d, %v), want (1, nil)", unrevoked, err)
	}
}
