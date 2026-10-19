package deps_http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	mfadomain "github.com/ambi/idmagic/backend/authentication/mfa/domain"
	sessiondomain "github.com/ambi/idmagic/backend/authentication/session/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

type pendingAccountAuthentication struct {
	pending         bool
	authenticatedAt int64
}

func (pendingAccountAuthentication) Subject() string          { return "pending-subject" }
func (a pendingAccountAuthentication) AuthenticatedAt() int64 { return a.authenticatedAt }
func (pendingAccountAuthentication) Session() string          { return "pending-session" }
func (a pendingAccountAuthentication) Pending() bool          { return a.pending }

type pendingAccountSessions struct {
	authentication support.Authentication
	err            error
}

func (s pendingAccountSessions) ResolveSession(context.Context, http.Header) (support.Authentication, error) {
	return s.authentication, s.err
}

func (pendingAccountSessions) EndSession(context.Context, string) error { return nil }

//spec:covers REQ-IDMANAGEMENT-053: pending の認証結果は、直近の認証時刻を持っていても本人のメール変更へ進めない。
func TestAccountIdentityRejectsPendingBeforeConvertingAuthentication(t *testing.T) {
	a := AccountIdentity{Authenticator: &support.Authenticator{Sessions: pendingAccountSessions{
		authentication: pendingAccountAuthentication{pending: true, authenticatedAt: time.Now().UTC().Unix()},
	}}}
	c := echo.New().NewContext(httptest.NewRequest(http.MethodPost, "/", http.NoBody), httptest.NewRecorder())
	sub, err := a.RequireStepUpSub(c)
	if sub != "" || !errors.Is(err, support.ErrAdminAuthenticationRequired) {
		t.Fatalf("subject=%q error=%v, want refusal of pending authentication", sub, err)
	}
}

func TestAccountIdentityPreservesFreshnessAndAuthenticationErrors(t *testing.T) {
	now := time.Now().UTC()
	failure := errors.New("session lookup failed")
	for _, tc := range []struct {
		name           string
		authentication support.Authentication
		err            error
		wantErr        error
		wantSub        string
	}{
		{name: "fresh", authentication: pendingAccountAuthentication{authenticatedAt: now.Unix()}, wantSub: "pending-subject"},
		{name: "absent", wantErr: support.ErrAdminAuthenticationRequired},
		{name: "expired", authentication: pendingAccountAuthentication{authenticatedAt: now.Add(-time.Hour).Unix()}, wantErr: mfadomain.ErrStepUpRequired},
		{name: "lookup failure", err: failure, wantErr: failure},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := AccountIdentity{Authenticator: &support.Authenticator{Sessions: pendingAccountSessions{authentication: tc.authentication, err: tc.err}}}
			c := echo.New().NewContext(httptest.NewRequest(http.MethodPost, "/", http.NoBody), httptest.NewRecorder())
			sub, err := a.RequireStepUpSub(c)
			if sub != tc.wantSub || !errors.Is(err, tc.wantErr) {
				t.Fatalf("subject=%q error=%v, want subject=%q error=%v", sub, err, tc.wantSub, tc.wantErr)
			}
		})
	}
}

func TestAccountIdentityMapsOnlyAuthenticationErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"authentication", support.ErrAdminAuthenticationRequired, http.StatusUnauthorized, "authentication_required"},
		{"step-up", mfadomain.ErrStepUpRequired, http.StatusForbidden, "step_up_required"},
		{"session", sessiondomain.ErrSessionNotFound, http.StatusNotFound, "session_not_found"},
		{"other", errors.New("other domain error"), 0, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c := echo.New().NewContext(httptest.NewRequest(http.MethodGet, "/", http.NoBody), recorder)
			handled, err := (AccountIdentity{}).WriteAuthenticationError(c, tc.err)
			if tc.status == 0 {
				if handled || err != nil || recorder.Body.Len() != 0 {
					t.Fatalf("handled=%v error=%v body=%s, want unchanged unknown error", handled, err, recorder.Body.String())
				}
				return
			}
			if !handled || err != nil || recorder.Code != tc.status || !strings.Contains(recorder.Body.String(), "urn:idmagic:error:"+tc.code) {
				t.Fatalf("handled=%v error=%v status=%d body=%s, want %d %s", handled, err, recorder.Code, recorder.Body.String(), tc.status, tc.code)
			}
			if recorder.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("Cache-Control=%q, want no-store", recorder.Header().Get("Cache-Control"))
			}
		})
	}
}
