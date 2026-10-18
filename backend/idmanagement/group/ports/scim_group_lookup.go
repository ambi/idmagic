package ports

import "context"

// SCIMGroupLookup は Group の詳細の由来表示に使う照会である。
type SCIMGroupLookup interface {
	IsSCIMGroup(ctx context.Context, tenantID, groupID string) (bool, error)
}
