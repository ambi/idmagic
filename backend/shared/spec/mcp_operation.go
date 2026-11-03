package spec

// ResourceTypeMcpOperation は接続先ごとの操作割当の予約語彙である。
// 対象 ID は登録 ID + "/" + 操作の識別子とし、テナントのモデルでは上書きしない。
const ResourceTypeMcpOperation = "idmagic_mcp_operation"

const RelationMcpInvoke = "invoke"
