package observability

import (
	"context"
	"net/http"
	"time"

	"github.com/ambi/idmagic/backend/shared/adapters/http/support"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go.opentelemetry.io/otel/attribute"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

// Metrics is the OTel-Meter-backed implementation of support.Metrics and
// support.HTTPAbortMetrics, exported for pull-based scrape via the
// MetricsExposition interface (system.yaml, GET /metrics). It uses the OTel
// Prometheus exporter (an sdkmetric.Reader) rather than a second, competing
// prometheus client registry, per ADR-017. It is independent of Provider
// (tracing + OTLP push metrics): both may run at the same time, and Metrics
// works even when Provider (OBSERVABILITY=otel) is disabled, because a
// scrape endpoint needs no collector to be configured.
type Metrics struct {
	provider *sdkmetric.MeterProvider
	registry *prometheus.Registry

	httpRequests   metric.Int64Counter
	httpDuration   metric.Float64Histogram
	httpInFlight   metric.Int64UpDownCounter
	httpAborts     metric.Int64Counter
	detachedFailed metric.Int64Counter
	loginAttempts  metric.Int64Counter
	loginThrottle  metric.Int64Counter
	tokenIssuance  metric.Int64Counter
	tokenDuration  metric.Float64Histogram
}

// NewMetrics builds a dedicated Prometheus registry and OTel MeterProvider for
// pull-based scraping. It does not touch the process-global otel meter
// provider (that stays Provider's OTLP push meter, gated by OBSERVABILITY),
// so the two exporters never contend over the same instruments.
func NewMetrics(serviceName, serviceVersion string) (*Metrics, error) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	reader, err := otelprom.New(otelprom.WithRegisterer(registry))
	if err != nil {
		return nil, err
	}
	res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
		semconv.ServiceVersion(serviceVersion),
		semconv.ServiceNamespace("identity"),
	))
	if err != nil {
		return nil, err
	}
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(reader),
		sdkmetric.WithView(sdkmetric.NewView(
			sdkmetric.Instrument{Name: "http_request_duration_seconds"},
			sdkmetric.Stream{Aggregation: sdkmetric.AggregationExplicitBucketHistogram{
				Boundaries: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			}},
		)),
		sdkmetric.WithView(sdkmetric.NewView(
			sdkmetric.Instrument{Name: "oauth2_token_issuance_duration_seconds"},
			sdkmetric.Stream{Aggregation: sdkmetric.AggregationExplicitBucketHistogram{
				Boundaries: []float64{0.01, 0.025, 0.05, 0.1, 0.2, 0.3, 0.5, 1, 2, 5},
			}},
		)),
	)
	meter := provider.Meter(serviceName)

	m := &Metrics{provider: provider, registry: registry}
	if m.httpRequests, err = meter.Int64Counter("http_requests_total"); err != nil {
		return nil, err
	}
	if m.httpDuration, err = meter.Float64Histogram("http_request_duration_seconds", metric.WithUnit("s")); err != nil {
		return nil, err
	}
	if m.httpInFlight, err = meter.Int64UpDownCounter("http_requests_in_flight"); err != nil {
		return nil, err
	}
	if m.httpAborts, err = meter.Int64Counter("http_request_aborts_total"); err != nil {
		return nil, err
	}
	if m.detachedFailed, err = meter.Int64Counter("operation_detached_completion_failures_total"); err != nil {
		return nil, err
	}
	if m.loginAttempts, err = meter.Int64Counter("authn_login_attempts_total"); err != nil {
		return nil, err
	}
	if m.loginThrottle, err = meter.Int64Counter("authn_login_throttle_total"); err != nil {
		return nil, err
	}
	if m.tokenIssuance, err = meter.Int64Counter("oauth2_token_issuance_total"); err != nil {
		return nil, err
	}
	if m.tokenDuration, err = meter.Float64Histogram("oauth2_token_issuance_duration_seconds", metric.WithUnit("s")); err != nil {
		return nil, err
	}
	return m, nil
}

// Handler returns the OpenMetrics scrape handler for GET /metrics.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

func (m *Metrics) Shutdown(ctx context.Context) error {
	return m.provider.Shutdown(ctx)
}

func (m *Metrics) BeginHTTPRequest(route, method string) func(statusCode int) {
	ctx := context.Background()
	routeAttr, methodAttr := attribute.String("route", route), attribute.String("method", method)
	m.httpInFlight.Add(ctx, 1, metric.WithAttributes(routeAttr, methodAttr))
	start := time.Now()
	done := false
	return func(statusCode int) {
		if done {
			return
		}
		done = true
		m.httpInFlight.Add(ctx, -1, metric.WithAttributes(routeAttr, methodAttr))
		attrs := metric.WithAttributes(routeAttr, methodAttr, attribute.Int("status_code", statusCode))
		m.httpRequests.Add(ctx, 1, attrs)
		m.httpDuration.Record(ctx, time.Since(start).Seconds(), attrs)
	}
}

func (m *Metrics) RecordLoginOutcome(outcome, reasonClass, method string) {
	m.loginAttempts.Add(context.Background(), 1, metric.WithAttributes(
		attribute.String("outcome", outcome),
		attribute.String("reason_class", reasonClass),
		attribute.String("method", method),
	))
}

func (m *Metrics) RecordLoginThrottle(policy, outcome string) {
	m.loginThrottle.Add(context.Background(), 1, metric.WithAttributes(
		attribute.String("policy", policy),
		attribute.String("outcome", outcome),
	))
}

func (m *Metrics) RecordTokenIssuance(grantType, outcome string, duration time.Duration) {
	attrs := metric.WithAttributes(
		attribute.String("grant_type", grantType),
		attribute.String("outcome", outcome),
	)
	m.tokenIssuance.Add(context.Background(), 1, attrs)
	m.tokenDuration.Record(context.Background(), duration.Seconds(), attrs)
}

func (m *Metrics) IncHTTPAbort(kind support.HTTPAbortKind) {
	m.httpAborts.Add(context.Background(), 1, metric.WithAttributes(attribute.String("kind", string(kind))))
}

func (m *Metrics) IncDetachedCompletionFailure() {
	m.detachedFailed.Add(context.Background(), 1)
}
