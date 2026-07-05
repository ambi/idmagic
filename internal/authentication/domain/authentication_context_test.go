package domain_test

import (
	"net/http"
	"testing"

	"github.com/ambi/idmagic/internal/authentication/domain"
)

// HTTPHeadersAdapter は標準 http.Header を framework 非依存の Headers に変換する。
func TestHTTPHeadersAdapterGet(t *testing.T) {
	h := http.Header{}
	h.Set("X-Forwarded-User", "alice")

	adapter := domain.HTTPHeadersAdapter{H: h}

	if got := adapter.Get("X-Forwarded-User"); got != "alice" {
		t.Errorf("Get(X-Forwarded-User) = %q, want alice", got)
	}
	// 大文字小文字は http.Header の canonical 化に従う。
	if got := adapter.Get("x-forwarded-user"); got != "alice" {
		t.Errorf("Get(x-forwarded-user) = %q, want alice", got)
	}
	if got := adapter.Get("X-Absent"); got != "" {
		t.Errorf("Get(X-Absent) = %q, want empty", got)
	}
}

// HTTPHeadersAdapter は domain.Headers インターフェースを満たす。
func TestHTTPHeadersAdapterImplementsHeaders(t *testing.T) {
	var _ domain.Headers = domain.HTTPHeadersAdapter{H: http.Header{}}
}
