package target

import "testing"

func TestBelowLimit(t *testing.T) {
	if !WithinLimit(1, 2) {
		t.Fatal("value below maximum was refused")
	}
}
