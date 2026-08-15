package domain_test

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
)

// REQ-OAUTH2-049: 委譲モードは act チェーンと principal 種別から一意に決まる。
func TestDeriveDelegationMode(t *testing.T) {
	cases := []struct {
		name string
		in   domain.DelegationSubject
		want domain.DelegationMode
	}{
		{
			name: "autonomous: agent acting as its own principal",
			in: domain.DelegationSubject{
				Sub: "agent-client", ClientID: "agent-client", PrincipalType: domain.PrincipalTypeAgent,
			},
			want: domain.DelegationModeAutonomous,
		},
		{
			name: "autonomous: client credentials without an agent binding",
			in:   domain.DelegationSubject{Sub: "svc-client", ClientID: "svc-client"},
			want: domain.DelegationModeAutonomous,
		},
		{
			name: "on_behalf_of: an agent acts for a user",
			in: domain.DelegationSubject{
				Sub: "user-alice", ClientID: "agent-client",
				Act: map[string]any{"sub": "agent-client"},
			},
			want: domain.DelegationModeOnBehalfOf,
		},
		{
			name: "on_behalf_of: a multi-step delegation chain",
			in: domain.DelegationSubject{
				Sub: "user-alice", ClientID: "agent-b",
				Act: map[string]any{"sub": "agent-b", "act": map[string]any{"sub": "agent-a"}},
			},
			want: domain.DelegationModeOnBehalfOf,
		},
		{
			name: "direct: a user's own token",
			in:   domain.DelegationSubject{Sub: "user-alice", ClientID: "portal"},
			want: domain.DelegationModeDirect,
		},
		{
			// workload identity の自己交換は sub と act.sub が同じクライアントになる。
			// これを代行と数えると自律実行が利用者の代理として記録されてしまう。
			name: "autonomous: an act chain that only points at the subject is not delegation",
			in: domain.DelegationSubject{
				Sub: "agent-client", ClientID: "agent-client", PrincipalType: domain.PrincipalTypeAgent,
				Act: map[string]any{"sub": "agent-client"},
			},
			want: domain.DelegationModeAutonomous,
		},
		{
			// 入れ子の内側だけが別主体でも代行である。外側だけを見て打ち切らない。
			name: "on_behalf_of: only the inner actor differs from the subject",
			in: domain.DelegationSubject{
				Sub: "agent-b", ClientID: "agent-b",
				Act: map[string]any{"sub": "agent-b", "act": map[string]any{"sub": "user-alice"}},
			},
			want: domain.DelegationModeOnBehalfOf,
		},
		{
			name: "autonomous: an unknown subject is never reported as a user's direct access",
			in:   domain.DelegationSubject{ClientID: "portal"},
			want: domain.DelegationModeAutonomous,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := domain.DeriveDelegationMode(tc.in); got != tc.want {
				t.Fatalf("DeriveDelegationMode() = %q, want %q", got, tc.want)
			}
		})
	}
}

// act チェーンが自己参照で閉じていても導出は停止する。subject 自身を指す環なので
// 「異なる行為者」による早期脱出は起きず、走査の上限だけが停止を保証する。
func TestDeriveDelegationModeTerminatesOnSelfReferentialAct(t *testing.T) {
	act := map[string]any{"sub": "agent-a"}
	act["act"] = act
	done := make(chan domain.DelegationMode, 1)
	go func() {
		done <- domain.DeriveDelegationMode(domain.DelegationSubject{
			Sub: "agent-a", ClientID: "agent-a", Act: act,
		})
	}()
	select {
	case got := <-done:
		// 上限に達したチェーンは深さの検査が拒否する側の入力なので、代行として扱う。
		if got != domain.DelegationModeOnBehalfOf {
			t.Fatalf("DeriveDelegationMode() = %q, want %q", got, domain.DelegationModeOnBehalfOf)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("DeriveDelegationMode did not terminate on a self-referential act chain")
	}
}
