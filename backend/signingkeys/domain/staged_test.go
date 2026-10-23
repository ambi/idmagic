package domain

import (
	"os"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/testing_statematrix"
)

//spec:covers REQ-SIGNINGKEYS-006: SigningKeyLifecycle のマトリクスの確認付きの有効化を、すべての状態から予測すること。
func TestPreparedActivationMatchesTheStateMatrix(t *testing.T) {
	markdown, err := os.ReadFile("../../../docs/modules/signing-keys/features/lifecycle.md")
	if err != nil {
		t.Fatal(err)
	}
	machine, err := testing_statematrix.Read(string(markdown), "SigningKeyLifecycle")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	for _, state := range machine.States {
		t.Run(state, func(t *testing.T) {
			target := &SigningKey{Kid: "target"}
			keys := []*SigningKey{target}
			switch state {
			case "Active":
				target.Active = true
			case "Prepared":
				target.Prepared = true
			case "Verifying":
				at := now.Add(time.Hour)
				target.ExpiresAt = &at
				target.RetiredAt = &now
			case "Retired":
				target.ExpiresAt = &now
				target.RetiredAt = &now
			case "Archived":
				target.ArchivedAt = &now
			}
			if state != "Active" {
				keys = append(keys, &SigningKey{Kid: "current", Active: true})
			}
			expected := machine.Outcomes(state, "確認付きの有効化")
			if len(expected) != 1 {
				t.Fatal("activation must have one state outcome")
			}
			result, err := ActivatePrepared(keys, "target", now, 7*24*time.Hour)
			switch expected[0].Kind {
			case testing_statematrix.Transition:
				if err != nil || !result.Changed || result.Next.Prepared || !result.Next.Active || result.Previous.Active || !result.Previous.ExpiresAt.Equal(now.Add(7*24*time.Hour)) {
					t.Fatalf("transition: %+v %v", result, err)
				}
				if ActiveKey(keys) != result.Next || PreparedKey(keys) != nil || !target.Prepared || target.Active {
					t.Fatal("the stored selection or the previously read candidate changed incorrectly")
				}
			case testing_statematrix.NoOp:
				if err != nil || result.Changed {
					t.Fatalf("no-op: %+v %v", result, err)
				}
			case testing_statematrix.Refusal:
				if err == nil {
					t.Fatal("refusal became successful")
				}
			}
		})
	}
}

//spec:covers REQ-SIGNINGKEYS-006: WS-* と既定 SAML の共有 XML 鍵だけが即時切り替えを拒否すること。
func TestSharedXMLKeysRequirePrepublication(t *testing.T) {
	for _, tc := range []struct {
		usage KeyUsage
		scope string
		want  bool
	}{
		{KeyUsageXMLFederationSigning, "default", true},
		{KeyUsageXMLFederationSigning, "profile", false},
		{KeyUsageSigning, "default", false},
	} {
		if RequiresPrepublication(tc.usage, tc.scope) != tc.want {
			t.Fatalf("prepublication policy for %s/%s", tc.usage, tc.scope)
		}
	}
}

//spec:covers REQ-SIGNINGKEYS-006: 切り替えの猶予は有効化時点から始め、負の期間は状態を変えずに拒否すること。
func TestActivationGraceBoundary(t *testing.T) {
	now := time.Date(2026, 10, 24, 0, 0, 0, 0, time.UTC)
	old := &SigningKey{Kid: "old", Active: true}
	next := &SigningKey{Kid: "next", Prepared: true}
	keys := []*SigningKey{old, next}
	if _, err := ActivatePrepared(keys, next.Kid, now, -time.Nanosecond); err == nil || keys[0] != old || keys[1] != next {
		t.Fatal("negative grace was accepted or changed state")
	}
	result, err := ActivatePrepared(keys, next.Kid, now, 0)
	if err != nil || !result.Previous.ExpiresAt.Equal(now) || ActiveKey(keys).Kid != next.Kid || !old.Active || !next.Prepared {
		t.Fatalf("zero grace: %+v %v", result, err)
	}
}
