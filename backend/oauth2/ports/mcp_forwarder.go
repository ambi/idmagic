package ports

import (
	"context"
	"errors"
	"net/http"
)

var (
	ErrMcpForwardDenied      = errors.New("MCP forwarding denied")
	ErrMcpForwardUnavailable = errors.New("MCP forwarding unavailable")
)

// McpForwardRequest は OAuth2 の評価を通った呼び出し元と、登録に束縛した要求である。
// Caller は提示トークンの検証結果だけを渡し、受信 JSON からは作らない。
type McpForwardRequest struct {
	TenantID, ConnectorID, URL string
	Caller                     *IntrospectionResult
	Headers                    http.Header
	Body                       []byte
}

// McpForwarder は所有者の規則を確認し、資格情報を外へ返さず上流の応答を開く。
type McpForwarder interface {
	ForwardMCP(ctx context.Context, request McpForwardRequest) (*http.Response, error)
}
