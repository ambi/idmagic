package tokenvault

import (
	"testing"

	"github.com/ambi/idmagic/backend/shared/spec"
)

//spec:covers REQ-TOKENVAULT-016: 必須の監査が未配線の構成は MCP の代理転送を提供しない。
func TestManagedMcpForwarderRequiresAudit(t *testing.T) {
	module := NewMemoryModule()
	if module.McpForwarder(nil) != nil {
		t.Fatal("provided forwarding without audit")
	}
	if module.McpForwarder(func(spec.DomainEvent) {}) == nil {
		t.Fatal("audited forwarding is unavailable")
	}
}
