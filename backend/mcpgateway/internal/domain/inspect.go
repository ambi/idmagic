package domain

import "github.com/ambi/idmagic/backend/shared/protocol/mcp"

const (
	ProtocolVersion        = mcp.ProtocolVersion
	MethodToolsCall        = mcp.MethodToolsCall
	CodeParseError         = mcp.CodeParseError
	CodeInvalidRequest     = mcp.CodeInvalidRequest
	CodeMethodNotFound     = mcp.CodeMethodNotFound
	CodeHeaderMismatch     = mcp.CodeHeaderMismatch
	CodeUnsupportedVersion = mcp.CodeUnsupportedVersion
)

type (
	Headers   = mcp.Headers
	Message   = mcp.Message
	Violation = mcp.Violation
)

// Inspect は配布する Gateway と API に同じ MCP の解釈を適用する。
func Inspect(headers Headers, body []byte) (Message, *Violation) {
	return mcp.Inspect(headers, body)
}
