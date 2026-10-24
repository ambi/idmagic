package domain_test

import (
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	"github.com/ambi/idmagic/backend/signinpolicy/domain"
)

//spec:covers REQ-APPLICATION-009: 再認証の期限は直近の認証事実から測り、期限を過ぎた場合と認証時刻が不明な場合に追加認証を要求する。
func TestReauthenticationDeadlineUsesTheLatestKnownAuthentication(t *testing.T) {
	now := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name     string
		authTime int64
		stepUpAt int64
		maxAge   int
		decision domain.PolicyDecision
	}{
		{name: "期限と同じ秒", authTime: now.Unix() - 300, maxAge: 300, decision: domain.PolicyAllow},
		{name: "期限の次の秒", authTime: now.Unix() - 301, maxAge: 300, decision: domain.PolicyStepUpRequired},
		{name: "直近の追加認証", authTime: now.Unix() - 600, stepUpAt: now.Unix() - 300, maxAge: 300, decision: domain.PolicyAllow},
		{name: "直近の初回認証", authTime: now.Unix() - 300, stepUpAt: now.Unix() - 600, maxAge: 300, decision: domain.PolicyAllow},
		// 長い期限でも、未知の時刻 0 を Unix epoch に成立した認証として扱わない。
		{name: "認証時刻が不明", maxAge: int(now.Unix()) + 300, decision: domain.PolicyStepUpRequired},
	} {
		t.Run(test.name, func(t *testing.T) {
			policy := &domain.AppSignInPolicy{Rules: []domain.SignInRule{{
				Enabled: true, RequiredAuthn: domain.RequiredAuthnLevel{Strength: domain.RequiredAuthnPassword},
				Condition: domain.AccessCondition{ReauthMaxAgeSeconds: &test.maxAge},
			}}}
			authn := &authdomain.AuthenticationContext{UserID: "alice", AuthTime: test.authTime, StepUpAt: test.stepUpAt}
			if got := domain.EvaluateSignInPolicy(policy, authn, "", now); got.Decision != test.decision {
				t.Fatalf("再認証の判定 = %s、期待値 %s", got.Decision, test.decision)
			}
		})
	}
}
