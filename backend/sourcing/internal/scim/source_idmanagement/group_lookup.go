package source_idmanagement

import (
	"context"

	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	scimports "github.com/ambi/idmagic/backend/sourcing/scim/ports"
)

// GroupLookup は対応レコードを、Group の詳細が必要とする有無へ変換する。
type GroupLookup struct {
	Repository scimports.ScimRepository
}

func (l GroupLookup) IsSCIMGroup(ctx context.Context, tenantID, groupID string) (bool, error) {
	ref, err := l.Repository.FindGroupRefByGroupID(ctx, tenantID, groupID)
	return ref != nil, err
}

var _ groupports.SCIMGroupLookup = GroupLookup{}
