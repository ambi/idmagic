package usecases

// 管理者による認証器リセット (第 2 層、wi-143)。管理者は対象 factor を
// 削除するだけで、代わりの factor を登録することはできない (なりすまし面を作らない
// ため)。TOTP / WebAuthn の両方が無くなった場合だけ、既存の管理者承認 enrollment
// bypass (IssueMfaEnrollmentBypass と同じ機構) を自動発行し、次回ログインを wi-127 の
// fail-closed な enrollment-required flow へ接続する。

import (
	"context"
	"errors"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	mfadomain "github.com/ambi/idmagic/backend/authentication/mfa/domain"
	recoveryports "github.com/ambi/idmagic/backend/authentication/recovery/ports"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

var ErrAuthenticatorResetNotAllowed = errors.New("authenticator reset is not allowed")

// reenrollmentBypassTTL is the lifetime of the enrollment bypass that
// ResetUserAuthenticators auto-issues when a reset leaves the target user
// with no second factor. It reuses IssueMfaEnrollmentBypass's [1m, 1h] bound.
const reenrollmentBypassTTL = time.Hour

// AuthenticatorResetDeps は MfaEnrollmentDeps に recovery code repository を足した
// もの。reset は factor 削除後に IssueMfaEnrollmentBypass をそのまま再利用するため
// MfaEnrollmentDeps を埋め込む。
type AuthenticatorResetDeps struct {
	MfaEnrollmentDeps
	RecoveryCodeRepo recoveryports.RecoveryCodeRepository
}

type AuthenticatorResetResult struct {
	MfaEnrolled          bool
	ReenrollmentRequired bool
	Bypass               *mfadomain.MfaEnrollmentBypass
}

func ResetUserAuthenticators(
	ctx context.Context,
	deps AuthenticatorResetDeps,
	actorID, userID string,
	targets []spec.AuthenticatorResetTarget,
	now time.Time,
) (*AuthenticatorResetResult, error) {
	if len(targets) == 0 {
		return nil, ErrAuthenticatorResetNotAllowed
	}
	for _, target := range targets {
		if !target.Valid() {
			return nil, ErrAuthenticatorResetNotAllowed
		}
	}
	user, err := deps.UserRepo.FindBySub(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.TenantID != tenancy.TenantID(ctx) || !user.IsActive() {
		return nil, ErrAuthenticatorResetNotAllowed
	}

	emitMfaEnrollmentEvent(deps.Emit, &authdomain.AuthenticatorResetRequested{
		At: now, TenantID: user.TenantID, ActorUserID: actorID, UserID: userID, Targets: targets,
	})

	for _, target := range targets {
		switch target {
		case spec.AuthenticatorResetTotp:
			if err := deps.MfaFactorRepo.Delete(ctx, userID, spec.MfaFactorTOTP); err != nil {
				return nil, err
			}
		case spec.AuthenticatorResetWebauthn:
			if deps.WebAuthnCredentialRepo == nil {
				continue
			}
			if err := deps.WebAuthnCredentialRepo.DeleteAllForSub(ctx, userID); err != nil {
				return nil, err
			}
		case spec.AuthenticatorResetRecoveryCode:
			if deps.RecoveryCodeRepo == nil {
				continue
			}
			if err := deps.RecoveryCodeRepo.DeleteAllForSub(ctx, userID); err != nil {
				return nil, err
			}
		}
	}

	now = authusecases.NormalizedNow(now)
	if err := authusecases.SyncMfaEnrolled(ctx, deps.UserRepo, deps.MfaFactorRepo, deps.WebAuthnCredentialRepo, user, now); err != nil {
		return nil, err
	}

	result := &AuthenticatorResetResult{MfaEnrolled: user.MfaEnrolled}
	if !user.MfaEnrolled {
		bypass, err := IssueMfaEnrollmentBypass(ctx, deps.MfaEnrollmentDeps, actorID, userID, reenrollmentBypassTTL, now)
		if err != nil && !errors.Is(err, ErrMfaEnrollmentAlreadyComplete) {
			return nil, err
		}
		if bypass != nil {
			result.ReenrollmentRequired = true
			result.Bypass = bypass
		}
	}

	emitMfaEnrollmentEvent(deps.Emit, &authdomain.AuthenticatorResetCompleted{
		At: now, TenantID: user.TenantID, ActorUserID: actorID, UserID: userID,
		Targets: targets, MfaEnrolled: result.MfaEnrolled, ReenrollmentRequired: result.ReenrollmentRequired,
	})
	return result, nil
}
