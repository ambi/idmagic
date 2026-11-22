package usecases_test

import (
	"testing"

	sessionusecases "github.com/ambi/idmagic/backend/authentication/internal/session/usecases"
)

// decision 4: /end_session の id_token_hint fallback は browser cookie から
// sid を読み取るだけで、失効はしない (revoke の有無は呼び出し側が決める)。
func TestSessionIDFromCookieExtractsWithoutRevoking(t *testing.T) {
	m := sessionusecases.NewSessionManager(nil)
	if got := m.SessionIDFromCookie("other=1; " + sessionusecases.SessionCookie + "=abc-123; foo=bar"); got != "abc-123" {
		t.Fatalf("SessionIDFromCookie=%q, want abc-123", got)
	}
	if got := m.SessionIDFromCookie(""); got != "" {
		t.Fatalf("SessionIDFromCookie('')=%q, want empty", got)
	}
}
