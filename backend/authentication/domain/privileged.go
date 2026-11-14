package domain

import (
	"errors"
	"time"
)

// PrivilegedAssurance は、制御面の操作が求める認証の保証の段階である (REQ-SYSTEM-033)。
type PrivilegedAssurance string

const (
	// PrivilegedRead は、システムコンソールへの入場と制御面の読み出しが求める段階である。
	PrivilegedRead PrivilegedAssurance = "privileged_read"
	// PrivilegedChange は、制御面の変更の操作が求める段階である。
	PrivilegedChange PrivilegedAssurance = "privileged_change"
)

// PrivilegedChangeMaxAgeSeconds は、制御面の変更の操作が許す、MFA の認証からの経過秒数である。
const PrivilegedChangeMaxAgeSeconds = 300

var (
	// ErrPrivilegedMFARequired は、認証が記憶済みの端末を除く MFA の方式を含まないことを表す。
	ErrPrivilegedMFARequired = errors.New("privileged operation requires multi-factor authentication")
	// ErrPrivilegedReauthenticationRequired は、MFA の認証が変更の操作には古いことを表す。
	ErrPrivilegedReauthenticationRequired = errors.New("privileged change requires a recent multi-factor authentication")
)

// EvaluatePrivilegedAuthentication は、認証が制御面の段階の保証を満たすかを判定する。
//
// MFA の方式は記憶済みの端末を除く第二要素（otp、webauthn、hwk、swk、rc）である。
// 変更の段階は、さらに auth_time が PrivilegedChangeMaxAgeSeconds の内にあることを求める。
// step_up_at は方式を記録せず、パスワードだけの再認証と区別できないので使わない。
func EvaluatePrivilegedAuthentication(authn *AuthenticationContext, level PrivilegedAssurance, now time.Time) error {
	if authn == nil || authn.AuthenticationPending || !authn.SecondFactorAuthenticated() {
		return ErrPrivilegedMFARequired
	}
	if level == PrivilegedChange &&
		(authn.AuthTime <= 0 || now.Unix()-authn.AuthTime > PrivilegedChangeMaxAgeSeconds) {
		return ErrPrivilegedReauthenticationRequired
	}
	return nil
}
