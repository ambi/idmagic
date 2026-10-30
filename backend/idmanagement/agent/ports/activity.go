package ports

import (
	"context"
	"time"
)

// AgentActivityReader は Agent ごとの最終のトークン発行時刻を返す (REQ-IDMANAGEMENT-094)。
// 発行の記録は Audit が持ち、組み立ての地点が Audit の検索属性の読み取りを渡す。
// IdManagement は記録の保存形式を知らず、結果の時刻だけを受け取る。
type AgentActivityReader interface {
	// LastTokenIssuedAt は、agentIDs のうち発行の記録がある Agent について、最新の発行時刻を
	// 返す。記録が無い Agent は結果に含めない。
	LastTokenIssuedAt(ctx context.Context, tenantID string, agentIDs []string) (map[string]time.Time, error)
}
