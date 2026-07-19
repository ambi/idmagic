package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"
)

func scrape(t *testing.T, m *Metrics) string {
	t.Helper()
	req := httptest.NewRequest("GET", "/metrics", http.NoBody)
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("scrape status = %d, want 200", rec.Code)
	}
	return rec.Body.String()
}

func TestMetricsExposesREDAndGoldenSignals(t *testing.T) {
	t.Parallel()

	m, err := NewMetrics("test-service", "0.0.0-test")
	if err != nil {
		t.Fatalf("NewMetrics() error = %v", err)
	}
	t.Cleanup(func() { _ = m.Shutdown(t.Context()) })

	observe := m.BeginHTTPRequest("/api/auth/login", "POST")
	observe(401)
	m.RecordLoginOutcome("failure", "invalid_credentials", "password")
	m.RecordLoginThrottle("account", "throttled")
	m.RecordTokenIssuance("client_credentials", "success", 42*time.Millisecond)
	m.IncHTTPAbort(support.HTTPAbortClientAborted)
	m.IncDetachedCompletionFailure()
	m.RecordJobClaimLatency(domain.LaneBulk, 250*time.Millisecond)
	m.RecordJobOutcome(domain.LaneBulk, "succeeded")
	m.RecordJobRetry(domain.LaneDefault)
	m.RecordJobQueueDepth(t.Context(), domain.LaneLatencySensitive, "queued", 3)

	body := scrape(t, m)

	for _, want := range []string{
		`http_requests_total{`,
		`route="/api/auth/login"`,
		`status_code="401"`,
		`http_request_duration_seconds`,
		`authn_login_attempts_total{`,
		`outcome="failure"`,
		`reason_class="invalid_credentials"`,
		`authn_login_throttle_total{`,
		`policy="account"`,
		`oauth2_token_issuance_total{`,
		`grant_type="client_credentials"`,
		`oauth2_token_issuance_duration_seconds`,
		`http_request_aborts_total{`,
		`kind="client_aborted"`,
		`operation_detached_completion_failures_total`,
		`jobs_claim_latency_seconds`,
		`jobs_outcome_total{`,
		`lane="bulk"`,
		`jobs_retry_total{`,
		`jobs_queue_depth{`,
		`lane="latency_sensitive"`,
		`status="queued"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("scrape output missing %q\n---\n%s", want, body)
		}
	}
}

// TestMetricsForbidsHighCardinalityLabels guards the cardinality budget
// (MetricsExposition, system.yaml): tenant_id/user_id/client_id/raw path
// values must never appear as label keys, regardless of what a future
// instrumentation call site is tempted to pass in.
func TestMetricsForbidsHighCardinalityLabels(t *testing.T) {
	t.Parallel()

	m, err := NewMetrics("test-service", "0.0.0-test")
	if err != nil {
		t.Fatalf("NewMetrics() error = %v", err)
	}
	t.Cleanup(func() { _ = m.Shutdown(t.Context()) })

	observe := m.BeginHTTPRequest("/realms/:tenant_id/token", "POST")
	observe(200)
	m.RecordLoginOutcome("success", "", "password")
	m.RecordLoginThrottle("ip", "allowed")
	m.RecordTokenIssuance("authorization_code", "success", time.Millisecond)
	m.RecordJobClaimLatency(domain.LaneBulk, time.Second)
	m.RecordJobOutcome(domain.LaneBulk, "succeeded")
	m.RecordJobRetry(domain.LaneDefault)
	m.RecordJobQueueDepth(t.Context(), domain.LaneBulk, "running", 1)

	body := scrape(t, m)

	for _, forbidden := range []string{"tenant_id=", "user_id=", "client_id=", "job_id="} {
		if strings.Contains(body, forbidden) {
			t.Errorf("scrape output must not expose high-cardinality label %q\n---\n%s", forbidden, body)
		}
	}
}
