package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication/trusteddevice/db_memory"
	"github.com/ambi/idmagic/backend/authentication/trusteddevice/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

const (
	testTenant = "11111111-1111-4111-8111-111111111111"
	testUser   = "22222222-2222-4222-8222-222222222222"
	testMaxAge = 30 * 24 * time.Hour
)

func testDeps() (Deps, *[]spec.DomainEvent) {
	events := &[]spec.DomainEvent{}
	return Deps{
		Repo: db_memory.NewTrustedDeviceRepository(),
		Emit: func(event spec.DomainEvent) { *events = append(*events, event) },
	}, events
}

func issue(t *testing.T, deps Deps, factor string, now time.Time) string {
	t.Helper()
	cookie, err := Issue(context.Background(), deps, testTenant, testUser, factor, "", testMaxAge, now)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	return cookie
}

// REQ-AUTHENTICATION-026: 記憶した端末は次のログインで第二要素を省略でき、利用のたびに
// verifier が回転する。
func TestEvaluateTrustsTheIssuedCookieAndRotatesIt(t *testing.T) {
	t.Parallel()
	deps, events := testDeps()
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	cookie := issue(t, deps, "otp", now)
	if cookie == "" {
		t.Fatal("Issue returned no cookie for a genuine second factor")
	}

	result, err := Evaluate(context.Background(), deps, testTenant, testUser, cookie, testMaxAge, now.Add(time.Hour))
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if !result.Trusted {
		t.Fatal("a freshly issued cookie must be trusted")
	}
	if result.RotatedCookie == "" || result.RotatedCookie == cookie {
		t.Fatalf("RotatedCookie = %q, want a rotated value", result.RotatedCookie)
	}
	if len(*events) != 1 {
		t.Fatalf("events = %d, want exactly one TrustedDeviceRegistered", len(*events))
	}
	if got := (*events)[0].EventType(); got != "TrustedDeviceRegistered" {
		t.Fatalf("event = %s, want TrustedDeviceRegistered", got)
	}
}

// REQ-AUTHENTICATION-026: 復旧コードでの成功と、テナントが機能を無効にしている場合は
// 記憶しない。
func TestIssueRefusesRecoveryCodeAndDisabledTenants(t *testing.T) {
	t.Parallel()
	deps, _ := testDeps()
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	if cookie := issue(t, deps, "rc", now); cookie != "" {
		t.Fatal("a recovery-code login must never remember the device")
	}
	if cookie := issue(t, deps, "pwd", now); cookie != "" {
		t.Fatal("a password-only login must never remember the device")
	}
	cookie, err := Issue(context.Background(), deps, testTenant, testUser, "otp", "", 0, now)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if cookie != "" {
		t.Fatal("a tenant with the feature disabled must never remember the device")
	}
}

// REQ-AUTHENTICATION-027: 回転前の古い cookie は次の正規利用で無効になる。
func TestEvaluateRejectsTheCookieFromBeforeRotation(t *testing.T) {
	t.Parallel()
	deps, _ := testDeps()
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	stolen := issue(t, deps, "otp", now)

	if _, err := Evaluate(context.Background(), deps, testTenant, testUser, stolen, testMaxAge, now.Add(time.Hour)); err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	result, err := Evaluate(context.Background(), deps, testTenant, testUser, stolen, testMaxAge, now.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if result.Trusted {
		t.Fatal("the pre-rotation cookie must stop working after a legitimate use")
	}
}

// REQ-AUTHENTICATION-027: 期限切れ、別テナント、別ユーザー、改竄した cookie は
// いずれも第二要素を省略できない。
func TestEvaluateFailsClosedOnEveryMismatch(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	otherTenant := "33333333-3333-4333-8333-333333333333"
	otherUser := "44444444-4444-4444-8444-444444444444"

	cases := []struct {
		name             string
		tenantID, userID string
		mutate           func(cookie string) string
		at               time.Time
	}{
		{"absolute expiry", testTenant, testUser, nil, now.Add(testMaxAge + time.Hour)},
		{"idle expiry", testTenant, testUser, nil, now.Add(31 * 24 * time.Hour)},
		{"other tenant", otherTenant, testUser, nil, now.Add(time.Hour)},
		{"other user", testTenant, otherUser, nil, now.Add(time.Hour)},
		{"tampered verifier", testTenant, testUser, func(cookie string) string {
			selector, _, _ := domain.ParseCookie(cookie)
			return domain.FormatCookie(selector, "forged-verifier")
		}, now.Add(time.Hour)},
		{"malformed cookie", testTenant, testUser, func(string) string { return "not-a-cookie" }, now.Add(time.Hour)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			deps, _ := testDeps()
			cookie := issue(t, deps, "otp", now)
			if tc.mutate != nil {
				cookie = tc.mutate(cookie)
			}
			result, err := Evaluate(context.Background(), deps, tc.tenantID, tc.userID, cookie, testMaxAge, tc.at)
			if err != nil {
				t.Fatalf("Evaluate: %v", err)
			}
			if result.Trusted {
				t.Fatal("the device must not be trusted")
			}
		})
	}
}

// idle 期限は絶対期限より短いテナントでも絶対期限を追い越さない (30 日の既定より
// 短い有効期間を設定したテナントでは絶対期限の側が先に効く)。
func TestEvaluateHonorsAShortTenantLifetime(t *testing.T) {
	t.Parallel()
	deps, _ := testDeps()
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	shortMaxAge := 24 * time.Hour
	cookie, err := Issue(context.Background(), deps, testTenant, testUser, "otp", "", shortMaxAge, now)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	result, err := Evaluate(context.Background(), deps, testTenant, testUser, cookie, shortMaxAge, now.Add(25*time.Hour))
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if result.Trusted {
		t.Fatal("a device past the tenant's short absolute lifetime must not be trusted")
	}
}

// REQ-AUTHENTICATION-028: 資格情報が変わると全デバイスが失効し、以後は第二要素が要る。
func TestRevokeAllForUserRevokesEveryDevice(t *testing.T) {
	t.Parallel()
	deps, events := testDeps()
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	first := issue(t, deps, "otp", now)
	second := issue(t, deps, "webauthn", now)

	if err := RevokeAllForUser(
		context.Background(), deps, testTenant, testUser, spec.TrustedDevicePasswordChange, now.Add(time.Hour),
	); err != nil {
		t.Fatalf("RevokeAllForUser: %v", err)
	}

	for _, cookie := range []string{first, second} {
		result, err := Evaluate(context.Background(), deps, testTenant, testUser, cookie, testMaxAge, now.Add(2*time.Hour))
		if err != nil {
			t.Fatalf("Evaluate: %v", err)
		}
		if result.Trusted {
			t.Fatal("a revoked device must never be trusted again")
		}
	}
	devices, err := ListActive(context.Background(), deps, testTenant, testUser, now.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	if len(devices) != 0 {
		t.Fatalf("active devices = %d, want 0", len(devices))
	}
	revoked := 0
	for _, event := range *events {
		if event.EventType() == "TrustedDeviceRevoked" {
			revoked++
		}
	}
	if revoked != 2 {
		t.Fatalf("TrustedDeviceRevoked events = %d, want 2", revoked)
	}
}

// 一括失効の再送は idempotent で、二重のイベントを出さない。
func TestRevokeAllForUserIsIdempotent(t *testing.T) {
	t.Parallel()
	deps, events := testDeps()
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	issue(t, deps, "otp", now)

	for range 2 {
		if err := RevokeAllForUser(
			context.Background(), deps, testTenant, testUser, spec.TrustedDeviceMfaChange, now.Add(time.Hour),
		); err != nil {
			t.Fatalf("RevokeAllForUser: %v", err)
		}
	}
	revoked := 0
	for _, event := range *events {
		if event.EventType() == "TrustedDeviceRevoked" {
			revoked++
		}
	}
	if revoked != 1 {
		t.Fatalf("TrustedDeviceRevoked events = %d, want 1", revoked)
	}
}

// REQ-AUTHENTICATION-029: 本人は個別に失効でき、再送は成功、他人のデバイスは見つからない。
func TestRevokeOneScopesToTheOwnerAndIsIdempotent(t *testing.T) {
	t.Parallel()
	deps, _ := testDeps()
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	issue(t, deps, "otp", now)
	devices, err := ListActive(context.Background(), deps, testTenant, testUser, now)
	if err != nil || len(devices) != 1 {
		t.Fatalf("ListActive = %v (err %v), want exactly one device", devices, err)
	}
	deviceID := devices[0].ID

	ctx := context.Background()
	if err := RevokeOne(ctx, deps, testTenant, testUser, deviceID, spec.TrustedDeviceSelfRevoke, now.Add(time.Hour)); err != nil {
		t.Fatalf("RevokeOne: %v", err)
	}
	if err := RevokeOne(ctx, deps, testTenant, testUser, deviceID, spec.TrustedDeviceSelfRevoke, now.Add(2*time.Hour)); err != nil {
		t.Fatalf("re-revoking must be idempotent: %v", err)
	}
	other := "44444444-4444-4444-8444-444444444444"
	if err := RevokeOne(ctx, deps, testTenant, other, deviceID, spec.TrustedDeviceSelfRevoke, now); !errors.Is(err, ErrTrustedDeviceNotFound) {
		t.Fatalf("err = %v, want ErrTrustedDeviceNotFound for another user's device", err)
	}
}

// Repo が未配線の環境では、発行も評価も静かに no-op になり MFA を弱めない。
func TestUnwiredRepositoryNeverTrustsADevice(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

	cookie, err := Issue(context.Background(), Deps{}, testTenant, testUser, "otp", "", testMaxAge, now)
	if err != nil || cookie != "" {
		t.Fatalf("Issue = %q (err %v), want no cookie", cookie, err)
	}
	result, err := Evaluate(context.Background(), Deps{}, testTenant, testUser, "a.b", testMaxAge, now)
	if err != nil || result.Trusted {
		t.Fatalf("Evaluate = %+v (err %v), want an untrusted result", result, err)
	}
}
