package proxy_http

import (
	"io"
	"net/http"
	"strings"
	"testing"

	mcpgatewaydomain "github.com/ambi/idmagic/backend/mcpgateway/internal/domain"
)

//spec:covers REQ-MCPGATEWAY-015: 古い評価応答や不正な許可一覧を素通しせず、上流を呼ぶ前に止める。
func TestCatalogToolListRequiresVerifiedNames(t *testing.T) {
	h := newHarness(t, accepted, func(options *Options) {
		options.Mapping = mcpgatewaydomain.Mapping{Catalog: true}
	})
	for _, names := range [][]string{nil, {"read", "read"}, {""}, strings.Split(strings.Repeat("x,", 100)+"x", ",")} {
		h.evaluator.result.AllowedTools = names
		request := toolRequest(t, h.gateway.URL, "read")
		body := `{"jsonrpc":"2.0","id":"call-1","method":"tools/list","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`
		request.Body = io.NopCloser(strings.NewReader(body))
		request.ContentLength = int64(len(body))
		request.Header.Set("Mcp-Method", "tools/list")
		request.Header.Del("Mcp-Name")
		response, _ := do(t, request)
		if response.StatusCode != http.StatusServiceUnavailable || h.calls.Load() != 0 {
			t.Fatalf("unverified list reached upstream: %d calls=%d", response.StatusCode, h.calls.Load())
		}
	}
}
