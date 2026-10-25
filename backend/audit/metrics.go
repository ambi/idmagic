package audit

import (
	"context"
	"time"

	"github.com/ambi/idmagic/backend/audit/ports"
	"github.com/prometheus/client_golang/prometheus"
)

type exportMetrics struct {
	operations *prometheus.CounterVec
	limits     *prometheus.CounterVec
	bytes      *prometheus.Desc
	store      ports.ExportArtifactStore
}

func newExportMetrics(store ports.ExportArtifactStore) *exportMetrics {
	return &exportMetrics{
		operations: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "idmagic_audit_export_operations_total", Help: "Audit export lifecycle operations."}, []string{"operation", "outcome"}),
		limits:     prometheus.NewCounterVec(prometheus.CounterOpts{Name: "idmagic_audit_export_limits_total", Help: "Audit exports that exceeded their configured limits."}, []string{"limit"}),
		bytes:      prometheus.NewDesc("idmagic_audit_export_artifact_bytes", "Retained published audit export artifact bytes.", nil, nil), store: store,
	}
}

func (m *exportMetrics) Record(operation, outcome string) {
	if m != nil {
		m.operations.WithLabelValues(operation, outcome).Inc()
	}
}

func (m *exportMetrics) Limit(kind string) {
	if m != nil {
		m.limits.WithLabelValues(kind).Inc()
	}
}

func (m *exportMetrics) Describe(ch chan<- *prometheus.Desc) {
	m.operations.Describe(ch)
	m.limits.Describe(ch)
	ch <- m.bytes
}

func (m *exportMetrics) Collect(ch chan<- prometheus.Metric) {
	m.operations.Collect(ch)
	m.limits.Collect(ch)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	size, err := m.store.TotalBytes(ctx)
	if err != nil {
		ch <- prometheus.NewInvalidMetric(m.bytes, err)
		return
	}
	ch <- prometheus.MustNewConstMetric(m.bytes, prometheus.GaugeValue, float64(size))
}

func (m Module) ExportCollector() prometheus.Collector {
	if m.metrics == nil {
		return nil
	}
	return m.metrics
}
