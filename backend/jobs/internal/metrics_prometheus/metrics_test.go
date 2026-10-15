package metrics_prometheus_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/jobs/domain"
	jobsmetrics "github.com/ambi/idmagic/backend/jobs/internal/metrics_prometheus"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
)

//spec:covers REQ-SYSTEM-001: Worker の /metrics はレーンごとの時間、結果、再試行、キュー深度を保つ。
func TestWorkerJobMetricsPreserveLaneLabelsAndValues(t *testing.T) {
	for _, tc := range []struct {
		lane  domain.ExecutionLane
		label string
	}{
		{domain.LaneLatencySensitive, "latency_sensitive"},
		{domain.LaneDefault, "default"},
		{domain.LaneBulk, "bulk"},
	} {
		t.Run(tc.label, func(t *testing.T) {
			m, err := jobsmetrics.NewMetrics("worker-test", "test")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = m.Shutdown(t.Context()) })
			m.RecordJobClaimLatency(tc.lane, 250*time.Millisecond)
			m.RecordJobDuration(tc.lane, "failed", 1500*time.Millisecond)
			m.RecordJobOutcome(tc.lane, "succeeded")
			m.RecordJobRetry(tc.lane)
			m.RecordJobQueueDepth(t.Context(), tc.lane, "queued", 3)
			m.RecordJobQueueDepth(t.Context(), tc.lane, "running", 2)

			rec := httptest.NewRecorder()
			m.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", http.NoBody))
			if rec.Code != 200 {
				t.Fatalf("scrape status = %d, want 200", rec.Code)
			}
			parser := expfmt.NewTextParser(model.LegacyValidation)
			families, err := parser.TextToMetricFamilies(strings.NewReader(rec.Body.String()))
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []struct {
				name, outcome, status string
				value                 float64
			}{
				{"jobs_claim_latency_seconds", "", "", 0.25},
				{"jobs_duration_seconds", "failed", "", 1.5},
				{"jobs_outcome_total", "succeeded", "", 1},
				{"jobs_retry_total", "", "", 1},
				{"jobs_queue_depth", "", "queued", 3},
				{"jobs_queue_depth", "", "running", 2},
			} {
				found := false
				for _, sample := range families[want.name].GetMetric() {
					labels := map[string]string{}
					for _, label := range sample.GetLabel() {
						labels[label.GetName()] = label.GetValue()
					}
					if labels["lane"] != tc.label || labels["outcome"] != want.outcome || labels["status"] != want.status {
						continue
					}
					found = true
					var got float64
					switch {
					case sample.Counter != nil:
						got = sample.GetCounter().GetValue()
					case sample.Gauge != nil:
						got = sample.GetGauge().GetValue()
					case sample.Histogram != nil:
						got = sample.GetHistogram().GetSampleSum()
					default:
						t.Fatalf("%s: unexpected metric type", want.name)
					}
					if got != want.value {
						t.Errorf("%s %v = %v, want %v", want.name, labels, got, want.value)
					}
				}
				if !found {
					t.Errorf("missing %s{lane=%q,outcome=%q,status=%q}", want.name, tc.label, want.outcome, want.status)
				}
			}
		})
	}
}
