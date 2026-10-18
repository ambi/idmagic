package source_idmanagement

import (
	"context"

	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	scimports "github.com/ambi/idmagic/backend/sourcing/scim/ports"
)

// UserLookup は対応レコードを、User の詳細が必要とする有無へ変換する。
type UserLookup struct {
	Repository scimports.ScimRepository
}

func (l UserLookup) IsSCIMUser(ctx context.Context, tenantID, userID string) (bool, error) {
	ref, err := l.Repository.FindUserRefByUserID(ctx, tenantID, userID)
	return ref != nil, err
}

var _ userports.SCIMUserLookup = UserLookup{}
