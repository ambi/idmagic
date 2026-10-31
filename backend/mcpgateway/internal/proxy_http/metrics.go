package proxy_http

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics は Prometheus の指標である。ラベルは宣言した値に限り、呼び出し元に系列を増やさせない。
type Metrics struct {
	registry   *prometheus.Registry
	requests   *prometheus.CounterVec
	evaluation prometheus.Histogram
}

// NewMetrics は専用のレジストリを作る。
func NewMetrics() *Metrics {
	m := &Metrics{
		registry: prometheus.NewRegistry(),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "mcp_gateway_requests_total", Help: "MCP requests by method, tool and outcome.",
		}, []string{"method", "tool", "outcome"}),
		evaluation: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "mcp_gateway_evaluation_duration_seconds", Help: "Duration of one resource access evaluation.",
			Buckets: prometheus.DefBuckets,
		}),
	}
	m.registry.MustRegister(m.requests, m.evaluation)
	return m
}

// Request は一つの要求の結果を数える。
func (m *Metrics) Request(method, tool, outcome string) {
	m.requests.WithLabelValues(method, tool, outcome).Inc()
}

// Evaluation は評価の時間を記録する。
func (m *Metrics) Evaluation(elapsed time.Duration) { m.evaluation.Observe(elapsed.Seconds()) }

// Handler は指標の公開のハンドラーである。
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}
