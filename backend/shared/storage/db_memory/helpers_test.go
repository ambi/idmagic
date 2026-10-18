package db_memory_test

import (
	"testing"

	"github.com/ambi/idmagic/backend/shared/storage/db_memory"
)

func TestScopedKeyUsesOnlyTheCallerFallback(t *testing.T) {
	for _, tc := range []struct{ scope, fallback, want string }{
		{"tenant-a", "tenant-b", "tenant-a|item"},
		{"", "tenant-b", "tenant-b|item"},
		{"", "", "|item"},
	} {
		if got := db_memory.ScopedKey(tc.scope, "item", tc.fallback); got != tc.want {
			t.Fatalf("ScopedKey(%q, item, %q) = %q, want %q", tc.scope, tc.fallback, got, tc.want)
		}
	}
}
