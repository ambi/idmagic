package resilience

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var ErrCircuitOpen = errors.New("circuit breaker is open")

type State string

const (
	StateClosed   State = "closed"
	StateOpen     State = "open"
	StateHalfOpen State = "half-open"
)

type Settings struct {
	Name             string
	FailureThreshold float64       // 失敗率しきい値 (例: 0.5)
	Cooldown         time.Duration // オープンからハーフオープンへの遷移時間
	MinRequests      uint32        // 判定を開始する最小リクエスト数
}

type CircuitBreaker struct {
	name             string
	failureThreshold float64
	cooldown         time.Duration
	minRequests      uint32

	mu     sync.Mutex
	state  State
	expiry time.Time
	counts counts

	// metrics
	stateCounter metric.Int64UpDownCounter
}

type counts struct {
	Requests uint32
	Failures uint32
}

func NewCircuitBreaker(settings Settings) *CircuitBreaker {
	minReqs := settings.MinRequests
	if minReqs == 0 {
		minReqs = 5
	}
	cooldown := settings.Cooldown
	if cooldown == 0 {
		cooldown = 15 * time.Second
	}
	threshold := settings.FailureThreshold
	if threshold <= 0 || threshold > 1 {
		threshold = 0.5
	}

	meter := otel.Meter("resilience")
	stateCounter, _ := meter.Int64UpDownCounter(
		"circuit_breaker_state",
		metric.WithDescription("Current state of the circuit breaker: 1 for open, 2 for half-open, 0 for closed"),
	)

	cb := &CircuitBreaker{
		name:             settings.Name,
		failureThreshold: threshold,
		cooldown:         cooldown,
		minRequests:      minReqs,
		state:            StateClosed,
		stateCounter:     stateCounter,
	}

	// 初期状態をメトリックに記録 (Closed = 0)
	if cb.stateCounter != nil {
		cb.stateCounter.Add(context.Background(), 0, metric.WithAttributes(attribute.String("name", cb.name)))
	}

	return cb
}

func (cb *CircuitBreaker) Execute(run func() error) error {
	if err := cb.beforeRequest(); err != nil {
		return err
	}

	defer func() {
		if err := recover(); err != nil {
			cb.afterRequest(false)
			panic(err)
		}
	}()

	err := run()
	cb.afterRequest(err == nil)
	return err
}

func (cb *CircuitBreaker) beforeRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state := cb.currentState(now)

	if state == StateOpen {
		return ErrCircuitOpen
	}
	return nil
}

func (cb *CircuitBreaker) afterRequest(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state := cb.currentState(now)

	cb.counts.Requests++
	if !success {
		cb.counts.Failures++
	}

	switch state {
	case StateClosed:
		if cb.counts.Requests >= cb.minRequests {
			failureRate := float64(cb.counts.Failures) / float64(cb.counts.Requests)
			if failureRate >= cb.failureThreshold {
				cb.setState(StateOpen, now.Add(cb.cooldown))
			}
		}
	case StateHalfOpen:
		if !success {
			cb.setState(StateOpen, now.Add(cb.cooldown))
		} else {
			cb.setState(StateClosed, time.Time{})
		}
	}
}

func (cb *CircuitBreaker) currentState(now time.Time) State {
	if cb.state == StateOpen && now.After(cb.expiry) {
		cb.setState(StateHalfOpen, time.Time{})
	}
	return cb.state
}

func (cb *CircuitBreaker) setState(state State, expiry time.Time) {
	if cb.state == state {
		return
	}
	cb.state = state
	cb.expiry = expiry
	cb.counts = counts{}

	if cb.stateCounter != nil {
		var val int64
		switch state {
		case StateClosed:
			val = 0
		case StateOpen:
			val = 1
		case StateHalfOpen:
			val = 2
		}
		cb.stateCounter.Add(context.Background(), val, metric.WithAttributes(attribute.String("name", cb.name)))
	}
}

// RetryWithBackoff は与えられた操作を Exponential Backoff でリトライする。
func RetryWithBackoff(ctx context.Context, op func() error) error {
	const maxAttempts = 5
	backoff := 500 * time.Millisecond
	const maxBackoff = 5 * time.Second
	const factor = 2.0

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := op()
		if err == nil {
			return nil
		}
		lastErr = err

		if attempt == maxAttempts || ctx.Err() != nil {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		backoff = time.Duration(float64(backoff) * factor)
		backoff = min(backoff, maxBackoff)
	}
	if lastErr != nil {
		return lastErr
	}
	return errors.New("retry failed")
}
