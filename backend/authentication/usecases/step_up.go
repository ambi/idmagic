package usecases

// 高 sensitivity な self-service 操作のための step-up 再認証 (ADR-043 / wi-43)。
// パスワード変更・MFA factor 解除・primary email 変更・全セッション失効などは、
// セッションが乗っ取られた場合の被害を抑えるため「直近 N 分以内に password / MFA で
// 再認証済み」であることを要求する。判定の recency ソースは max(auth_time, step_up_at)。
// 新規ログイン直後 (auth_time が新しい) はそのまま step-up 済みとして扱う。

import (
	"context"
	"errors"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"

	"github.com/ambi/idmagic/backend/authentication/domain"
	authnports "github.com/ambi/idmagic/backend/authentication/ports"
	idmports "github.com/ambi/idmagic/backend/identitymanagement/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
)

// StepUpRecencySeconds は step-up が有効とみなされる窓 (5 分)。
const StepUpRecencySeconds = 300

// StepUpMethod は再認証に使える factor。
type StepUpMethod string

const (
	StepUpMethodPassword     StepUpMethod = "password"
	StepUpMethodTOTP         StepUpMethod = "totp"
	StepUpMethodWebAuthn     StepUpMethod = "webauthn"
	StepUpMethodRecoveryCode StepUpMethod = "recovery_code"
)

var (
	// ErrStepUpRequired は recency 窓を外れており再認証が必要なことを表す (handler が 403 に写す)。
	ErrStepUpRequired = errors.New("step-up authentication required")
	// ErrStepUpFailed は提示された factor (パスワード / TOTP コード) の検証に失敗したことを表す。
	ErrStepUpFailed = errors.New("step-up authentication failed")
	// ErrStepUpUnsupportedMethod は未対応 / 未登録の method を要求したことを表す。
	ErrStepUpUnsupportedMethod = errors.New("step-up method unsupported")
)

// StepUpSatisfied は authn が recency 窓内に強い (再)認証を済ませているかを判定する。
func StepUpSatisfied(authn *domain.AuthenticationContext, now time.Time) bool {
	if authn == nil || authn.AuthenticationPending {
		return false
	}
	recent := max(authn.AuthTime, authn.StepUpAt)
	if recent <= 0 {
		return false
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return now.Unix()-recent <= StepUpRecencySeconds
}

// AvailableStepUpMethods は user が step-up に使える method を返す。password は常に利用可能、
// totp は enrolled の場合のみ。
func AvailableStepUpMethods(user *idmdomain.User) []StepUpMethod {
	methods := []StepUpMethod{StepUpMethodPassword}
	if user != nil && user.MfaEnrolled {
		methods = append(methods, StepUpMethodTOTP)
	}
	return methods
}

// StepUpDeps は CompleteStepUp の依存。SessionManager は step_up_at の刻印に使う。
// WebAuthn / RecoveryCodeRepo は passkey / recovery code による step-up (wi-26) に使い、
// nil の場合その method は提示・受理しない。
type StepUpDeps struct {
	UserRepo         idmports.UserRepository
	PasswordHasher   authnports.PasswordHasher
	MfaFactorRepo    authnports.MfaFactorRepository
	WebAuthn         WebAuthnDeps
	RecoveryCodeRepo authnports.RecoveryCodeRepository
	SessionManager   *SessionManager
	Emit             func(spec.DomainEvent)
}

// stepUpMethods は sub が step-up に使える method を repo の実状から正確に算出する。password は
// 常に利用可能、totp / webauthn は enrolled 時、recovery_code は有効な残数がある時のみ。
func stepUpMethods(ctx context.Context, deps StepUpDeps, sub string) []StepUpMethod {
	methods := []StepUpMethod{StepUpMethodPassword}
	if deps.MfaFactorRepo != nil {
		if f, err := deps.MfaFactorRepo.Find(ctx, sub, spec.MfaFactorTOTP); err == nil && f != nil && f.Secret != nil && *f.Secret != "" {
			methods = append(methods, StepUpMethodTOTP)
		}
	}
	if deps.WebAuthn.RP != nil && deps.WebAuthn.CredentialRepo != nil {
		if creds, err := deps.WebAuthn.CredentialRepo.ListBySub(ctx, sub); err == nil && len(creds) > 0 {
			methods = append(methods, StepUpMethodWebAuthn)
		}
	}
	if deps.RecoveryCodeRepo != nil {
		if codes, err := deps.RecoveryCodeRepo.ListBySub(ctx, sub); err == nil {
			for _, code := range codes {
				if code.ConsumedAt == nil {
					methods = append(methods, StepUpMethodRecoveryCode)
					break
				}
			}
		}
	}
	return methods
}

// StepUpStart は利用可能な method を返し StepUpRequested を emit する。
func StepUpStart(
	ctx context.Context,
	deps StepUpDeps,
	sub, sessionID string,
) ([]StepUpMethod, error) {
	user, err := deps.UserRepo.FindBySub(ctx, sub)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	if deps.Emit != nil {
		deps.Emit(&domain.StepUpRequested{
			At: time.Now().UTC(), TenantID: tenancy.TenantID(ctx), UserID: sub, SessionID: sessionID,
		})
	}
	return stepUpMethods(ctx, deps, sub), nil
}

// CompleteStepUpInput は再認証の検証材料。method に応じて Password / Code / Assertion を使う。
type CompleteStepUpInput struct {
	Sub       string
	SessionID string
	Method    StepUpMethod
	Password  string
	Code      string
	Assertion []byte // method=webauthn のとき navigator.credentials.get の結果 JSON。
	Now       time.Time
}

// CompleteStepUp は提示された factor を検証し、成功すれば session に step_up_at を刻んで
// StepUpCompleted を emit する。検証失敗は ErrStepUpFailed。
func CompleteStepUp(ctx context.Context, deps StepUpDeps, in CompleteStepUpInput) error {
	now := in.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	user, err := deps.UserRepo.FindBySub(ctx, in.Sub)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}
	switch in.Method {
	case StepUpMethodPassword:
		if user.PasswordHash == "" {
			return ErrStepUpUnsupportedMethod
		}
		ok, verr := deps.PasswordHasher.Verify(in.Password, user.PasswordHash)
		if verr != nil {
			return verr
		}
		if !ok {
			return ErrStepUpFailed
		}
	case StepUpMethodTOTP:
		if !user.MfaEnrolled {
			return ErrStepUpUnsupportedMethod
		}
		result, verr := VerifyTOTPFactor(ctx, deps.MfaFactorRepo, in.Sub, in.Code, now)
		if verr != nil {
			return verr
		}
		if result == nil || !result.OK {
			return ErrStepUpFailed
		}
	case StepUpMethodWebAuthn:
		if deps.WebAuthn.RP == nil {
			return ErrStepUpUnsupportedMethod
		}
		// challenge は step-up challenge endpoint で session id をキーに発行済み。
		if _, verr := FinishWebAuthnAssertion(ctx, deps.WebAuthn, in.SessionID, in.Sub, in.Assertion, now); verr != nil {
			return ErrStepUpFailed
		}
	case StepUpMethodRecoveryCode:
		if deps.RecoveryCodeRepo == nil {
			return ErrStepUpUnsupportedMethod
		}
		recoveryDeps := RecoveryCodesDeps{UserRepo: deps.UserRepo, RecoveryCodeRepo: deps.RecoveryCodeRepo, Emit: deps.Emit}
		if _, verr := ConsumeRecoveryCode(ctx, recoveryDeps, in.Sub, in.Code, now); verr != nil {
			return ErrStepUpFailed
		}
	default:
		return ErrStepUpUnsupportedMethod
	}
	if deps.SessionManager != nil && in.SessionID != "" {
		if _, err := deps.SessionManager.RecordStepUp(ctx, in.SessionID, now); err != nil {
			return err
		}
	}
	if deps.Emit != nil {
		deps.Emit(&domain.StepUpCompleted{
			At: now, TenantID: tenancy.TenantID(ctx), UserID: in.Sub,
			SessionID: in.SessionID, Method: string(in.Method),
		})
	}
	return nil
}
