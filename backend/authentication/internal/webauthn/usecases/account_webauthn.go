package usecases

// self-service の WebAuthn / Passkey 登録・削除 (wi-26)。actor.sub == target.sub に
// 固定する。登録は attestation の検証 (RP ID / origin / challenge) を所持証明とし、削除は
// step-up を HTTP 層で要求する。1 ユーザーが複数 credential を持てる。

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	trusteddeviceusecases "github.com/ambi/idmagic/backend/authentication/internal/trusteddevice/usecases"
	authusecases "github.com/ambi/idmagic/backend/authentication/internal/usecases"
	"github.com/ambi/idmagic/backend/authentication/webauthn/domain"
	webauthnports "github.com/ambi/idmagic/backend/authentication/webauthn/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

const webAuthnRegistrationKeyPrefix = "reg:"

// StartWebAuthnRegistration は登録 challenge を発行し、SessionData を sub にひもづけて保存する。
// 既存 credential は exclude して同一 authenticator の二重登録を避ける。永続化はまだしない。
func StartWebAuthnRegistration(
	ctx context.Context,
	deps WebAuthnDeps,
	sub string,
) (*protocol.CredentialCreation, error) {
	if deps.RP == nil {
		return nil, ErrWebAuthnNotConfigured
	}
	user, err := authusecases.LoadSelfUser(ctx, deps.UserRepo, sub)
	if err != nil {
		return nil, err
	}
	stored, err := deps.CredentialRepo.ListBySub(ctx, sub)
	if err != nil {
		return nil, err
	}
	waUser, err := loadWebAuthnUser(user, stored)
	if err != nil {
		return nil, err
	}
	creation, session, err := deps.RP.rp.BeginRegistration(
		waUser,
		gowebauthn.WithExclusions(gowebauthn.Credentials(waUser.credentials).CredentialDescriptors()),
		gowebauthn.WithConveyancePreference(protocol.PreferNoAttestation),
		gowebauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			UserVerification: protocol.VerificationPreferred,
			ResidentKey:      protocol.ResidentKeyRequirementDiscouraged,
		}),
	)
	if err != nil {
		return nil, err
	}
	if err := saveCeremonySession(ctx, deps.SessionStore, webAuthnRegistrationKeyPrefix+sub, session); err != nil {
		return nil, err
	}
	return creation, nil
}

// FinishWebAuthnRegistration は attestation を検証して credential を永続化する。challenge は
// 一度きりで消費し、最初の credential で mfa_enrolled を true にする。
func FinishWebAuthnRegistration(
	ctx context.Context,
	deps WebAuthnDeps,
	sub string,
	body []byte,
	label *string,
	now time.Time,
) error {
	if deps.RP == nil {
		return ErrWebAuthnNotConfigured
	}
	if err := trusteddeviceusecases.RequireRevoker(deps.TrustedDevices); err != nil {
		return err
	}
	now = authusecases.NormalizedNow(now)
	user, err := authusecases.LoadSelfUser(ctx, deps.UserRepo, sub)
	if err != nil {
		return err
	}
	session, err := takeCeremonySession(ctx, deps.SessionStore, webAuthnRegistrationKeyPrefix+sub)
	if err != nil {
		return err
	}
	stored, err := deps.CredentialRepo.ListBySub(ctx, sub)
	if err != nil {
		return err
	}
	waUser, err := loadWebAuthnUser(user, stored)
	if err != nil {
		return err
	}
	parsed, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(body))
	if err != nil {
		return errors.Join(ErrWebAuthnVerification, err)
	}
	credential, err := deps.RP.rp.CreateCredential(waUser, *session, parsed)
	if err != nil {
		return errors.Join(ErrWebAuthnVerification, err)
	}
	specCred := fromWebAuthnCredential(sub, credential, sanitizeWebAuthnLabel(label), now)
	if err := specCred.Validate(); err != nil {
		return err
	}
	// 既存の credential ID は、持ち主を問わず登録の失敗にする（WebAuthn Level 3 の
	// 登録手順）。none attestation は秘密鍵の所持を証明しないため、他人の credential ID を
	// 名乗る応答も検証を通る。応答は検証の失敗と区別せず、登録済みかを知らせない。
	if err := deps.CredentialRepo.Save(ctx, specCred); errors.Is(err, webauthnports.ErrWebAuthnCredentialExists) {
		return errors.Join(ErrWebAuthnVerification, err)
	} else if err != nil {
		return err
	}
	if err := authusecases.SyncMfaEnrolled(ctx, deps.UserRepo, deps.MfaFactorRepo, deps.CredentialRepo, user, now); err != nil {
		return err
	}
	if deps.Emit != nil {
		deps.Emit(&authdomain.WebAuthnCredentialRegistered{At: now, TenantID: user.TenantID, UserID: user.ID})
	}
	return revokeOnCredentialChange(ctx, deps, user, now)
}

// revokeOnCredentialChange は認証要素の集合が変わった User の記憶済みの端末をすべて失効させる。
func revokeOnCredentialChange(ctx context.Context, deps WebAuthnDeps, user *userdomain.User, now time.Time) error {
	return trusteddeviceusecases.RevokeOnCredentialChange(
		ctx, deps.TrustedDevices, trusteddeviceusecases.EmitWithoutError(deps.Emit),
		user.TenantID, user.ID, spec.TrustedDeviceMfaChange, now,
	)
}

// RemoveWebAuthnCredential は指定 credential を解除する。存在しない / 別ユーザーの場合は
// ErrWebAuthnCredentialNotFound。解除後は残存する第二要素に応じて mfa_enrolled を再計算する。
func RemoveWebAuthnCredential(
	ctx context.Context,
	deps WebAuthnDeps,
	sub string,
	credentialID string,
	now time.Time,
) error {
	if err := trusteddeviceusecases.RequireRevoker(deps.TrustedDevices); err != nil {
		return err
	}
	now = authusecases.NormalizedNow(now)
	user, err := authusecases.LoadSelfUser(ctx, deps.UserRepo, sub)
	if err != nil {
		return err
	}
	existing, err := deps.CredentialRepo.FindByCredentialID(ctx, credentialID)
	if err != nil {
		return err
	}
	if existing == nil || existing.UserID != sub {
		return ErrWebAuthnCredentialNotFound
	}
	if err := deps.CredentialRepo.Delete(ctx, sub, credentialID); err != nil {
		return err
	}
	if err := authusecases.SyncMfaEnrolled(ctx, deps.UserRepo, deps.MfaFactorRepo, deps.CredentialRepo, user, now); err != nil {
		return err
	}
	if deps.Emit != nil {
		deps.Emit(&authdomain.WebAuthnCredentialRemoved{At: now, TenantID: user.TenantID, UserID: user.ID})
	}
	return revokeOnCredentialChange(ctx, deps, user, now)
}

// ListWebAuthnCredentials は self-service セキュリティ画面向けに登録済み credential を返す。
func ListWebAuthnCredentials(
	ctx context.Context,
	deps WebAuthnDeps,
	sub string,
) ([]*domain.WebAuthnCredential, error) {
	if _, err := authusecases.LoadSelfUser(ctx, deps.UserRepo, sub); err != nil {
		return nil, err
	}
	return deps.CredentialRepo.ListBySub(ctx, sub)
}

// sanitizeWebAuthnLabel は空白のみのラベルを nil に潰す。
func sanitizeWebAuthnLabel(label *string) *string {
	if label == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*label)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
