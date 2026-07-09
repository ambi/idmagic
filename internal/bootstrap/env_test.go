package bootstrap

import "testing"

func TestEnvInt32RejectsOutOfRangeValues(t *testing.T) {
	t.Setenv("BOOTSTRAP_INT32", "2147483648")

	if got := envInt32("BOOTSTRAP_INT32", 20); got != 20 {
		t.Fatalf("envInt32 out-of-range value = %d, want fallback 20", got)
	}
}

func TestEnvCircuitBreakerMinRequestsRejectsOutOfRangeValues(t *testing.T) {
	t.Setenv("BOOTSTRAP_UINT32", "4294967296")

	if got := envCircuitBreakerMinRequests("BOOTSTRAP_UINT32"); got != 10 {
		t.Fatalf("envCircuitBreakerMinRequests out-of-range value = %d, want fallback 10", got)
	}
}

func TestEnvCircuitBreakerMinRequestsRejectsNegativeValues(t *testing.T) {
	t.Setenv("BOOTSTRAP_UINT32", "-1")

	if got := envCircuitBreakerMinRequests("BOOTSTRAP_UINT32"); got != 10 {
		t.Fatalf("envCircuitBreakerMinRequests negative value = %d, want fallback 10", got)
	}
}
