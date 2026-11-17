package workloadidentity

import "github.com/prometheus/client_golang/prometheus"

// jwksFallbackMetrics は、JWKS の取得に失敗して最後に取得できた鍵を使った回数を、鍵の
// 出所 (プロセスのメモリか、信頼設定に保存した鍵か) ごとに数える (REQ-WORKLOADIDENTITY-018)。
type jwksFallbackMetrics struct {
	uses *prometheus.CounterVec
}

func newJWKSFallbackMetrics() *jwksFallbackMetrics {
	return &jwksFallbackMetrics{uses: prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "idmagic_workload_jwks_fallback_total",
		Help: "Workload attestation verifications that used last-known-good JWKS after a fetch failure, by key source.",
	}, []string{"source"})}
}

func (m *jwksFallbackMetrics) observe(source string) {
	if m != nil {
		m.uses.WithLabelValues(source).Inc()
	}
}

// JWKSFallbackCollector は、最後に取得できた鍵を使った回数の指標を返す。指標を持たない
// 組み立てでは nil を返す。
func (m Module) JWKSFallbackCollector() prometheus.Collector {
	if m.jwksFallback == nil {
		return nil
	}
	return m.jwksFallback.uses
}
