package spec

// ResourceTypeMcpOperation は接続先ごとの操作割当の予約語彙である。
// 対象 ID は登録 ID + "/" + 操作の識別子とし、テナントのモデルでは上書きしない。
const ResourceTypeMcpOperation = "idmagic_mcp_operation"

const RelationMcpInvoke = "invoke"

// RelationMcpInvokeAnyTarget は、対象ごとの権限を持つ操作で全対象を許す範囲である。
// 特定の対象の範囲は、操作の ID + "/" + 対象 ID への RelationMcpInvoke で表す。
const RelationMcpInvokeAnyTarget = "invoke_any_target"

// McpTargetResourceID は、操作の割当の ID に対象 ID を付けた範囲の ID を返す。
func McpTargetResourceID(operationResourceID, targetID string) string {
	return operationResourceID + "/" + targetID
}
