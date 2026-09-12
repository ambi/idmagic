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
	recoverymemory "github.com/ambi/idmagic/backend/authentication/recovery/db_memory"
	recoveryusecases "github.com/ambi/idmagic/backend/authentication/recovery/usecases"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessiondomain "github.com/ambi/idmagic/backend/authentication/session/domain"
	totpmemory "github.com/ambi/idmagic/backend/authentication/totp/db_memory"
	totpdomain "github.com/ambi/idmagic/backend/authentication/totp/domain"
	trusteddevicememory "github.com/ambi/idmagic/backend/authentication/trusteddevice/db_memory"
	trusteddevicedomain "github.com/ambi/idmagic/backend/authentication/trusteddevice/domain"
	webauthnmemory "github.com/ambi/idmagic/backend/authentication/webauthn/db_memory"
	webauthndomain "github.com/ambi/idmagic/backend/authentication/webauthn/domain"
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

// 1 つ残らず消え、元のパスワードはもう照合されない。
//
//spec:covers GDPR-ERASURE: 削除要求を受けた利用者の資格情報は Authentication 側の保存先から
func TestCredentialErasureLeavesNothingAuthenticable(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 8, 12, 0, 0, 0, time.UTC)

	users := usermemory.NewUserRepository()
	history := passwordmemory.NewPasswordHistoryRepository()
	factors := totpmemory.NewMfaFactorRepository()
	sessions := sessionmemory.NewSessionStore()
	devices := trusteddevicememory.NewTrustedDeviceRepository()
	credentials := webauthnmemory.NewWebAuthnCredentialRepository()
	codes := recoverymemory.NewRecoveryCodeRepository()
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

	credentialLabel := "YubiKey"
	if err := credentials.Save(ctx, &webauthndomain.WebAuthnCredential{
		CredentialID: "credential-erasure", UserID: credentialSubject,
		PublicKey: "cG9zdC1xdWFudHVtLXB1YmxpYy1rZXk", Label: &credentialLabel, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	// リカバリコードは製品の生成経路から作る。平文はここでしか手に入らないので、消去前に
	// 実際に 1 本消費して「認証として成立する」ことまで確かめられる。
	recoveryDeps := recoveryusecases.RecoveryCodesDeps{UserRepo: users, RecoveryCodeRepo: codes}
	generated, err := recoveryusecases.GenerateRecoveryCodes(ctx, recoveryDeps, credentialSubject, now)
	if err != nil {
		t.Fatal(err)
	}

	// 前提: 消去前は資格情報が揃っていて、元のパスワードが照合される。
	if entries, err := history.Recent(ctx, credentialSubject, 10); err != nil || len(entries) == 0 {
		t.Fatalf("消去前のパスワード履歴=%v err=%v, want 1 件以上", entries, err)
	}
	if verified, err := hasher.Verify(credentialPassword, encoded); err != nil || !verified {
		t.Fatalf("消去前に元のパスワードが照合されない: verified=%v err=%v", verified, err)
	}
	if stored, err := credentials.ListBySub(ctx, credentialSubject); err != nil || len(stored) != 1 {
		t.Fatalf("消去前の WebAuthn 資格情報=%v err=%v, want 1 件", stored, err)
	}
	if _, err := recoveryusecases.ConsumeRecoveryCode(
		ctx, recoveryDeps, credentialSubject, generated.Codes[0], now,
	); err != nil {
		t.Fatalf("消去前にリカバリコードが消費できない: %v", err)
	}

	if err := userusecases.DeleteUser(ctx, userusecases.AdminUserDeps{
		UserRepo:               users,
		PasswordHistoryRepo:    history,
		MfaFactorRepo:          factors,
		SessionStore:           sessions,
		TrustedDeviceRepo:      devices,
		WebAuthnCredentialRepo: credentials,
		RecoveryCodeRepo:       codes,
		Emit:                   func(spec.DomainEvent) error { return nil },
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

	// WebAuthn: 空の一覧は、BeginWebAuthnAssertion と FinishWebAuthnAssertion が
	// ErrWebAuthnNoCredential を返す条件そのものである。credential id を知っていても引けない。
	if stored, err := credentials.ListBySub(ctx, credentialSubject); err != nil || len(stored) != 0 {
		t.Fatalf("消去後に残った WebAuthn 資格情報=%v err=%v", stored, err)
	}
	if found, err := credentials.FindByCredentialID(ctx, "credential-erasure"); err != nil || found != nil {
		t.Fatalf("消去後も credential id から引ける: %+v err=%v", found, err)
	}
	// リカバリコード: 製品自身が残数を答える経路で 0 になる。消費できるコードが 1 本も無い。
	// 消去後に ConsumeRecoveryCode を呼ぶ形は採らない。同関数は先に利用者を読むので、
	// Tombstone 化した利用者がコードの不在を覆い隠す。
	status, err := recoveryusecases.RecoveryCodeStatusFor(ctx, codes, credentialSubject)
	if err != nil {
		t.Fatal(err)
	}
	if status.Total != 0 || status.Remaining != 0 {
		t.Fatalf("消去後のリカバリコード total=%d remaining=%d, want どちらも 0", status.Total, status.Remaining)
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
