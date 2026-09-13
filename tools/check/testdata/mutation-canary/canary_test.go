package mutationcanary

import "testing"

func TestDecorateShortValue(t *testing.T) {
	if got := decorate("a", 2); got != "value:a" {
		t.Fatalf("decorate short value = %q, want value:a", got)
	}
}
