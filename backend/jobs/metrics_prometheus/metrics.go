// Package metrics_prometheus は Jobs のレーンを計測のラベルへ変換する。
package metrics_prometheus

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/jobs/usecases"
	sharedmetrics "github.com/ambi/idmagic/backend/shared/observability/metrics_prometheus"
)

// Metrics は JobsMetrics と、同じ provider による HTTP 計測と公開を提供する。
type Metrics struct {
	*sharedmetrics.Metrics
}

var _ usecases.JobsMetrics = (*Metrics)(nil)

func NewMetrics(serviceName, serviceVersion string) (*Metrics, error) {
	m, err := sharedmetrics.NewMetrics(serviceName, serviceVersion)
	if err != nil {
		return nil, err
	}
	return &Metrics{Metrics: m}, nil
}

func (m *Metrics) RecordJobClaimLatency(lane domain.ExecutionLane, latency time.Duration) {
	m.Metrics.RecordJobClaimLatency(string(lane), latency)
}

func (m *Metrics) RecordJobDuration(lane domain.ExecutionLane, outcome string, duration time.Duration) {
	m.Metrics.RecordJobDuration(string(lane), outcome, duration)
}

func (m *Metrics) RecordJobOutcome(lane domain.ExecutionLane, outcome string) {
	m.Metrics.RecordJobOutcome(string(lane), outcome)
}

func (m *Metrics) RecordJobRetry(lane domain.ExecutionLane) {
	m.Metrics.RecordJobRetry(string(lane))
}

func (m *Metrics) RecordJobQueueDepth(ctx context.Context, lane domain.ExecutionLane, status string, count int64) {
	m.Metrics.RecordJobQueueDepth(ctx, string(lane), status, count)
}
