package support_http_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	rlports "github.com/ambi/idmagic/backend/shared/ratelimit/ports"
	"github.com/labstack/echo/v5"
)

type stubRateLimiter struct {
	result rlports.RateLimitResult
	err    error
}

func (s stubRateLimiter) Allow(context.Context, string, string, time.Time) (rlports.RateLimitResult, error) {
	return s.result, s.err
}

type rateLimitMetricsSpy struct {
	calls []struct{ policy, outcome string }
}

func (s *rateLimitMetricsSpy) BeginHTTPRequest(string, string) func(int)         { return func(int) {} }
func (s *rateLimitMetricsSpy) RecordLoginOutcome(string, string, string)         {}
func (s *rateLimitMetricsSpy) RecordLoginThrottle(string, string)                {}
func (s *rateLimitMetricsSpy) RecordTokenIssuance(string, string, time.Duration) {}
func (s *rateLimitMetricsSpy) RecordQuotaExceeded(string)                        {}
func (s *rateLimitMetricsSpy) RecordEndpointRateLimit(policy, outcome string) {
	s.calls = append(s.calls, struct{ policy, outcome string }{policy, outcome})
}

func withRateLimitEchoContext(t *testing.T, fn func(c *echo.Context, rec *httptest.ResponseRecorder)) {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/token", http.NoBody)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	fn(c, rec)
}

func TestCheckRateLimitNilLimiterAllows(t *testing.T) {
	withRateLimitEchoContext(t, func(c *echo.Context, rec *httptest.ResponseRecorder) {
		blocked, err := support.CheckRateLimit(c, nil, nil, "token", "key")
		if err != nil || blocked {
			t.Fatalf("nil limiter should allow: blocked=%v err=%v", blocked, err)
		}
		if rec.Body.Len() != 0 {
			t.Fatalf("nil limiter should not write a response body, got %q", rec.Body.String())
		}
	})
}

func TestCheckRateLimitAllowedDoesNotWriteResponse(t *testing.T) {
	withRateLimitEchoContext(t, func(c *echo.Context, rec *httptest.ResponseRecorder) {
		limiter := stubRateLimiter{result: rlports.RateLimitResult{Allowed: true}}
		blocked, err := support.CheckRateLimit(c, limiter, nil, "token", "key")
		if err != nil || blocked {
			t.Fatalf("allowed request should not block: blocked=%v err=%v", blocked, err)
		}
		if rec.Body.Len() != 0 {
			t.Fatalf("allowed request should not write a response body, got %q", rec.Body.String())
		}
	})
}

// TestCheckRateLimitBlockedWrites429WithRetryAfter also guards against the bug this two-value
// signature exists to prevent: WriteRateLimited returns nil on a successful write, so a caller
// using `if err := CheckRateLimit(...); err != nil` alone would silently fall through on a
// blocked-but-successfully-written request. blocked must be true even though err is nil here.
func TestCheckRateLimitBlockedWrites429WithRetryAfter(t *testing.T) {
	withRateLimitEchoContext(t, func(c *echo.Context, rec *httptest.ResponseRecorder) {
		limiter := stubRateLimiter{result: rlports.RateLimitResult{Allowed: false, RetryAfterSeconds: 42}}
		blocked, err := support.CheckRateLimit(c, limiter, nil, "token", "key")
		if err != nil {
			t.Fatalf("a successful 429 write should not itself error: %v", err)
		}
		if !blocked {
			t.Fatalf("blocked=false, want true (caller must stop even though err is nil)")
		}
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("status=%d, want 429", rec.Code)
		}
		if got := rec.Header().Get("Retry-After"); got != "42" {
			t.Fatalf("Retry-After=%q, want 42", got)
		}
		if got := rec.Body.String(); got == "" {
			t.Fatalf("expected a JSON body, got empty")
		}
	})
}

func TestCheckRateLimitStoreErrorPropagates(t *testing.T) {
	withRateLimitEchoContext(t, func(c *echo.Context, rec *httptest.ResponseRecorder) {
		wantErr := errors.New("store unreachable")
		limiter := stubRateLimiter{err: wantErr}
		blocked, err := support.CheckRateLimit(c, limiter, nil, "token", "key")
		if !errors.Is(err, wantErr) {
			t.Fatalf("err=%v, want %v (fail-closed: propagate rather than allow)", err, wantErr)
		}
		if blocked {
			t.Fatalf("blocked should be false on a store error; the caller stops via err, not blocked")
		}
		if rec.Body.Len() != 0 {
			t.Fatalf("store error should not itself write a response body, got %q", rec.Body.String())
		}
	})
}

func TestCheckRateLimitRecordsMetricOutcome(t *testing.T) {
	cases := []struct {
		name        string
		limiter     rlports.RateLimiter
		wantOutcome string
	}{
		{"allowed", stubRateLimiter{result: rlports.RateLimitResult{Allowed: true}}, "allowed"},
		{"rate_limited", stubRateLimiter{result: rlports.RateLimitResult{Allowed: false, RetryAfterSeconds: 1}}, "rate_limited"},
		{"store_unavailable", stubRateLimiter{err: errors.New("down")}, "store_unavailable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withRateLimitEchoContext(t, func(c *echo.Context, _ *httptest.ResponseRecorder) {
				spy := &rateLimitMetricsSpy{}
				_, _ = support.CheckRateLimit(c, tc.limiter, spy, "token", "key")
				if len(spy.calls) != 1 || spy.calls[0].policy != "token" || spy.calls[0].outcome != tc.wantOutcome {
					t.Fatalf("calls=%+v, want one {token %s}", spy.calls, tc.wantOutcome)
				}
			})
		})
	}
}
