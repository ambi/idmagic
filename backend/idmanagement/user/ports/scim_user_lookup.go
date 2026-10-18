package ports

import "context"

// SCIMUserLookup は User の詳細の由来表示に使う照会である。
// Sourcing の保存型と書き込み操作を、表示の呼び出し側へ渡さない。
type SCIMUserLookup interface {
	IsSCIMUser(ctx context.Context, tenantID, userID string) (bool, error)
}
