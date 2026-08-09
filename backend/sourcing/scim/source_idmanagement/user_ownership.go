package source_idmanagement

import (
	"context"

	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	scimports "github.com/ambi/idmagic/backend/sourcing/scim/ports"
)

// UserOwnershipGuard adapts Sourcing ownership records to IdManagement's
// narrow fail-closed batch guard without reversing the context dependency.
type UserOwnershipGuard struct {
	Repository scimports.ScimRepository
}

func (g UserOwnershipGuard) SourceManagedUserIDs(ctx context.Context, tenantID string, userIDs []string) (map[string]bool, error) {
	refs, err := g.Repository.FindUserRefsByUserIDs(ctx, tenantID, userIDs)
	if err != nil {
		return nil, err
	}
	managed := make(map[string]bool, len(refs))
	for _, ref := range refs {
		managed[ref.UserID] = true
	}
	return managed, nil
}

var _ userports.UserSourceOwnershipGuard = UserOwnershipGuard{}
