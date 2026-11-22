package resilience

import (
	"testing"
	"time"
)

func TestRandomDelayStaysWithinInclusiveLimit(t *testing.T) {
	for _, limit := range []time.Duration{0, time.Nanosecond, 30 * time.Second, 30 * time.Minute, 1<<63 - 1} {
		for range 100 {
			if delay := RandomDelay(limit); delay < 0 || delay > limit {
				t.Fatalf("RandomDelay(%v)=%v, want [0,%v]", limit, delay, limit)
			}
		}
	}
}
