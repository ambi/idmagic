package domain

import (
	"errors"
	"testing"
	"time"
)

//spec:covers REQ-SYSTEM-033: 制御面の読み出しは記憶済みの端末を除く MFA の方式を求め、変更はさらに auth_time が 5 分の内にあることを求め、境界の 300 秒を受け付けて 301 秒を拒否することを固定する。
func TestEvaluatePrivilegedAuthentication(t *testing.T) {
	now := time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)
	at := func(ago time.Duration) int64 { return now.Add(-ago).Unix() }
	for _, tc := range []struct {
		name  string
		authn *AuthenticationContext
		level PrivilegedAssurance
		want  error
	}{
		{"読み出し: otp", &AuthenticationContext{AMR: []string{"pwd", "otp"}, AuthTime: at(time.Hour)}, PrivilegedRead, nil},
		{"読み出し: webauthn", &AuthenticationContext{AMR: []string{"webauthn"}, AuthTime: at(time.Hour)}, PrivilegedRead, nil},
		{"読み出し: 復旧コード", &AuthenticationContext{AMR: []string{"pwd", "rc"}, AuthTime: at(time.Hour)}, PrivilegedRead, nil},
		{"読み出し: パスワードだけ", &AuthenticationContext{AMR: []string{"pwd"}, AuthTime: at(0)}, PrivilegedRead, ErrPrivilegedMFARequired},
		{"読み出し: 記憶済みの端末だけ", &AuthenticationContext{AMR: []string{"pwd", "tdev"}, AuthTime: at(0)}, PrivilegedRead, ErrPrivilegedMFARequired},
		{"読み出し: 上流の IdP", &AuthenticationContext{AMR: []string{"federated"}, AuthTime: at(0)}, PrivilegedRead, ErrPrivilegedMFARequired},
		{"読み出し: 認証待ち", &AuthenticationContext{AMR: []string{"pwd", "otp"}, AuthTime: at(0), AuthenticationPending: true}, PrivilegedRead, ErrPrivilegedMFARequired},
		{"読み出し: なし", nil, PrivilegedRead, ErrPrivilegedMFARequired},
		{"変更: 直近の otp", &AuthenticationContext{AMR: []string{"pwd", "otp"}, AuthTime: at(time.Minute)}, PrivilegedChange, nil},
		{"変更: 300 秒", &AuthenticationContext{AMR: []string{"pwd", "otp"}, AuthTime: at(300 * time.Second)}, PrivilegedChange, nil},
		{"変更: 301 秒", &AuthenticationContext{AMR: []string{"pwd", "otp"}, AuthTime: at(301 * time.Second)}, PrivilegedChange, ErrPrivilegedReauthenticationRequired},
		{"変更: auth_time なし", &AuthenticationContext{AMR: []string{"pwd", "otp"}}, PrivilegedChange, ErrPrivilegedReauthenticationRequired},
		{"変更: 直近のパスワードだけ", &AuthenticationContext{AMR: []string{"pwd"}, AuthTime: at(0)}, PrivilegedChange, ErrPrivilegedMFARequired},
		{"変更: step_up_at だけが新しい", &AuthenticationContext{AMR: []string{"pwd", "otp"}, AuthTime: at(time.Hour), StepUpAt: at(0)}, PrivilegedChange, ErrPrivilegedReauthenticationRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := EvaluatePrivilegedAuthentication(tc.authn, tc.level, now); !errors.Is(got, tc.want) || (tc.want == nil && got != nil) {
				t.Fatalf("EvaluatePrivilegedAuthentication = %v, want %v", got, tc.want)
			}
		})
	}
}
