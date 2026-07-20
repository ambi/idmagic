package handlers_http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	authnports "github.com/ambi/idmagic/backend/authentication/session/ports"

	"github.com/labstack/echo/v5"
)

type spyMetrics struct {
	throttlePolicy  string
	throttleOutcome string
}

func (s *spyMetrics) BeginHTTPRequest(string, string) func(int)                      { return func(int) {} }
func (s *spyMetrics) RecordLoginOutcome(outcome, reasonClass, method string)         {}
func (s *spyMetrics) RecordTokenIssuance(grantType, outcome string, _ time.Duration) {}
func (s *spyMetrics) RecordLoginThrottle(policy, outcome string) {
	s.throttlePolicy, s.throttleOutcome = policy, outcome
}
func (s *spyMetrics) RecordQuotaExceeded(string) {}

type stubLoginThrottle struct {
	result authnports.LoginThrottleResult
	err    error
}

func (s stubLoginThrottle) TryAcquire(
	context.Context, authnports.LoginThrottleKind, string, time.Time,
) (authnports.LoginThrottleResult, error) {
	return s.result, s.err
}

func (stubLoginThrottle) RecordFailure(
	context.Context, authnports.LoginThrottleKind, string, time.Time,
) (authnports.LoginThrottleResult, error) {
	return authnports.LoginThrottleResult{}, nil
}

func (stubLoginThrottle) RecordSuccess(context.Context, authnports.LoginThrottleKind, string) error {
	return nil
}

type stubTenantSaltStore struct {
	salt []byte
	err  error
}

func (s stubTenantSaltStore) GetSalt(context.Context) ([]byte, error) {
	return s.salt, s.err
}

func TestExtractClientIPUsesOnlyTrustedForwardedHops(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/login", http.NoBody)
	req.Header.Set("X-Forwarded-For", "203.0.113.10, 198.51.100.20")
	if got := extractClientIP(req, 0); got != "" {
		t.Fatalf("trustedHops=0 returned %q", got)
	}
	if got := extractClientIP(req, 1); got != "203.0.113.10" {
		t.Fatalf("trustedHops=1 returned %q", got)
	}
	if got := extractClientIP(req, 2); got != "" {
		t.Fatalf("trustedHops=2 returned %q", got)
	}
}

// withEchoContext は d のメソッドを *echo.Context 経由で呼び出す小さな helper。
// acquireLoginThrottle / correlationHash は c.Request().Context() だけを使うため、
// 実リクエストの生成は httptest で十分。
func withEchoContext(t *testing.T, run func(c *echo.Context)) {
	t.Helper()
	e := echo.New()
	e.POST("/x", func(c *echo.Context) error {
		run(c)
		return nil
	})
	req := httptest.NewRequest(http.MethodPost, "/x", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
}

func TestAcquireLoginThrottleRecordsMetricOutcome(t *testing.T) {
	cases := []struct {
		name              string
		throttle          stubLoginThrottle
		wantAllowed       bool
		wantMetricOutcome string
	}{
		{
			name:              "allowed",
			throttle:          stubLoginThrottle{result: authnports.LoginThrottleResult{Allowed: true}},
			wantAllowed:       true,
			wantMetricOutcome: "allowed",
		},
		{
			name:              "throttled",
			throttle:          stubLoginThrottle{result: authnports.LoginThrottleResult{Allowed: false, RetryAfterSeconds: 30}},
			wantAllowed:       false,
			wantMetricOutcome: "throttled",
		},
		{
			name:              "store unavailable",
			throttle:          stubLoginThrottle{err: context.DeadlineExceeded},
			wantAllowed:       false,
			wantMetricOutcome: "store_unavailable",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spy := &spyMetrics{}
			d := Deps{LoginAttemptThrottle: tc.throttle}
			d.Metrics = spy
			withEchoContext(t, func(c *echo.Context) {
				result, _ := d.acquireLoginThrottle(c, authnports.LoginThrottleAccount, "alice")
				if result.Allowed != tc.wantAllowed {
					t.Fatalf("Allowed=%v, want %v", result.Allowed, tc.wantAllowed)
				}
			})
			if spy.throttlePolicy != string(authnports.LoginThrottleAccount) {
				t.Fatalf("policy=%q, want %q", spy.throttlePolicy, authnports.LoginThrottleAccount)
			}
			if spy.throttleOutcome != tc.wantMetricOutcome {
				t.Fatalf("outcome=%q, want %q", spy.throttleOutcome, tc.wantMetricOutcome)
			}
		})
	}
}

func TestAcquireLoginThrottleWithoutStoreAllowsAndSkipsMetric(t *testing.T) {
	spy := &spyMetrics{}
	d := Deps{}
	d.Metrics = spy
	withEchoContext(t, func(c *echo.Context) {
		result, err := d.acquireLoginThrottle(c, authnports.LoginThrottleIP, "203.0.113.10")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Allowed {
			t.Fatalf("expected Allowed=true when no throttle store is configured")
		}
	})
	if spy.throttleOutcome != "" {
		t.Fatalf("expected no metric recorded without a throttle store, got outcome=%q", spy.throttleOutcome)
	}
}

func TestCorrelationHashUsesTenantSaltWhenAvailable(t *testing.T) {
	d := Deps{TenantSaltStore: stubTenantSaltStore{salt: []byte("tenant-salt")}}
	var salted string
	withEchoContext(t, func(c *echo.Context) {
		salted = d.correlationHash(c, "alice")
	})
	if salted == "" || salted == hashThrottleKey("alice") {
		t.Fatalf("expected salted hash distinct from the unsalted fallback, got %q", salted)
	}
}

func TestCorrelationHashFallsBackToUnsaltedHashWithoutSaltStore(t *testing.T) {
	d := Deps{}
	var got string
	withEchoContext(t, func(c *echo.Context) {
		got = d.correlationHash(c, "alice")
	})
	if want := hashThrottleKey("alice"); got != want {
		t.Fatalf("correlationHash=%q, want fallback %q", got, want)
	}
}

func TestCorrelationHashFallsBackWhenSaltStoreErrors(t *testing.T) {
	d := Deps{TenantSaltStore: stubTenantSaltStore{err: context.DeadlineExceeded}}
	var got string
	withEchoContext(t, func(c *echo.Context) {
		got = d.correlationHash(c, "alice")
	})
	if want := hashThrottleKey("alice"); got != want {
		t.Fatalf("correlationHash=%q, want fallback %q", got, want)
	}
}

func TestWriteLoginThrottledReturnsRetryAfter(t *testing.T) {
	e := echo.New()
	e.POST("/login", func(c *echo.Context) error {
		return writeLoginThrottled(c, 900)
	})
	req := httptest.NewRequest(http.MethodPost, "/login", http.NoBody)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Retry-After"); got != "900" {
		t.Fatalf("Retry-After=%q", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control=%q", got)
	}
}
