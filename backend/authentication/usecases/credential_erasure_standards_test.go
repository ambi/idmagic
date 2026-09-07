package usecases_test

// docs/standards.md の GDPR-ERASURE のうち、Authentication が担う資格情報の破棄を観測する。
// IdManagement が担う UserLifecycle の Purge 遷移は
// backend/idmanagement/user/usecases/erasure_standards_test.go が別に観測する。
// 行は 2 つの Context を名指しているので、片方に注記を置いただけではもう片方が素通りする。
//
// 破棄は IdManagement の Purge から cascade で走るが、消える対象は Authentication の
// 資格情報である。したがって観測は Authentication 側の保存先を 1 つずつ読み直し、
// さらに「元のパスワードがもう照合されない」ことまで確かめる。行が消えたかどうかだけを
// 読むと、行を消したうえでハッシュを残す実装を通してしまう。

import (
	"context"
	"testing"
	"time"

	passwordmemory "github.com/ambi/idmagic/backend/authentication/password/db_memory"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessiondomain "github.com/ambi/idmagic/backend/authentication/session/domain"
	totpmemory "github.com/ambi/idmagic/backend/authentication/totp/db_memory"
	totpdomain "github.com/ambi/idmagic/backend/authentication/totp/domain"
	trusteddevicememory "github.com/ambi/idmagic/backend/authentication/trusteddevice/db_memory"
	trusteddevicedomain "github.com/ambi/idmagic/backend/authentication/trusteddevice/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	"github.com/ambi/idmagic/backend/shared/security/testing_passwords"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const (
	credentialSubject  = "user-credential-erasure"
	credentialPassword = "erasure-password-4821"
)

// GDPR-ERASURE: 削除要求を受けた利用者の資格情報は Authentication 側の保存先から
// 1 つ残らず消え、元のパスワードはもう照合されない。
func TestCredentialErasureLeavesNothingAuthenticable(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)

	users := usermemory.NewUserRepository()
	history := passwordmemory.NewPasswordHistoryRepository()
	factors := totpmemory.NewMfaFactorRepository()
	sessions := sessionmemory.NewSessionStore()
	devices := trusteddevicememory.NewTrustedDeviceRepository()
	// 本番と同じ Argon2id を使う。偽のハッシャーに差し替えると、照合できなくなったことの
	// 観測が実装ではなくテストダブルの性質になる。
	hasher := testing_passwords.NewHasher()

	encoded, err := hasher.Hash(credentialPassword)
	if err != nil {
		t.Fatal(err)
	}
	users.Seed(&userdomain.User{
		ID: credentialSubject, TenantID: tenancydomain.DefaultTenantID,
		PreferredUsername: "credential-subject", PasswordHash: encoded,
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour),
	})
	if err := history.Add(ctx, credentialSubject, encoded, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	secret := "JBSWY3DPEHPK3PXP"
	if err := factors.Save(ctx, &totpdomain.MfaFactor{
		UserID: credentialSubject, Type: spec.MfaFactorTOTP, Secret: &secret, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := sessions.Save(ctx, &sessiondomain.LoginSession{
		ID: "session-erasure", TenantID: tenancydomain.DefaultTenantID, UserID: credentialSubject,
		AuthTime: now.Unix(), AMR: []string{"pwd"}, ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if err := devices.Save(ctx, &trusteddevicedomain.TrustedDevice{
		ID: "device-erasure", TenantID: tenancydomain.DefaultTenantID, UserID: credentialSubject,
		Selector: "selector-erasure", VerifierHash: "verifier-erasure",
		CreatedAt: now, LastUsedAt: now, ExpiresAt: now.Add(24 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	// 前提: 消去前は資格情報が揃っていて、元のパスワードが照合される。
	if entries, err := history.Recent(ctx, credentialSubject, 10); err != nil || len(entries) == 0 {
		t.Fatalf("消去前のパスワード履歴=%v err=%v, want 1 件以上", entries, err)
	}
	if verified, err := hasher.Verify(credentialPassword, encoded); err != nil || !verified {
		t.Fatalf("消去前に元のパスワードが照合されない: verified=%v err=%v", verified, err)
	}

	if err := userusecases.DeleteUser(ctx, userusecases.AdminUserDeps{
		UserRepo:            users,
		PasswordHistoryRepo: history,
		MfaFactorRepo:       factors,
		SessionStore:        sessions,
		TrustedDeviceRepo:   devices,
		Emit:                func(spec.DomainEvent) error { return nil },
	}, userusecases.DeleteUserInput{
		ActorUserID: "admin", Sub: credentialSubject, Reason: "erasure request", Now: now,
	}); err != nil {
		t.Fatal(err)
	}

	if entries, err := history.Recent(ctx, credentialSubject, 10); err != nil || len(entries) != 0 {
		t.Fatalf("消去後に残ったパスワード履歴=%v err=%v", entries, err)
	}
	if got, err := factors.ListBySub(ctx, credentialSubject); err != nil || len(got) != 0 {
		t.Fatalf("消去後に残った MFA 要素=%v err=%v", got, err)
	}
	if session, err := sessions.Find(ctx, "session-erasure"); err != nil || session != nil {
		t.Fatalf("消去後に残ったセッション=%+v err=%v", session, err)
	}
	if device, err := devices.FindBySelector(ctx, tenancydomain.DefaultTenantID, "selector-erasure"); err != nil || device != nil {
		t.Fatalf("消去後に残った信頼済みデバイス=%+v err=%v", device, err)
	}

	// 残るのは行の有無だけではない。保管されたパスワードそのものが、元のパスワードを
	// 受け付けない値に置き換わっている。
	tombstone, err := users.FindBySubIncludingDeleted(ctx, credentialSubject)
	if err != nil {
		t.Fatal(err)
	}
	if tombstone == nil {
		t.Fatal("tombstone が消えている")
	}
	if tombstone.PasswordHash == encoded {
		t.Fatal("消去後もパスワードハッシュが元のまま残っている")
	}
	verified, err := hasher.Verify(credentialPassword, tombstone.PasswordHash)
	if err == nil && verified {
		t.Fatal("消去後も元のパスワードが照合される")
	}
}
