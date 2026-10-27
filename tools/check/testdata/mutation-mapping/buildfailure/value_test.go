package buildfailure

import "testing"

func TestValue(t *testing.T) {
	if got := Value(); got != 2 {
		t.Fatalf("Value() = %d, want 2", got)
	}
}
