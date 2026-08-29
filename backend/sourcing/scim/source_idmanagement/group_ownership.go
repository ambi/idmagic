package source_idmanagement

import (
	"context"

	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	scimports "github.com/ambi/idmagic/backend/sourcing/scim/ports"
)

// GroupOwnershipGuard adapts Sourcing ownership records to IdManagement's
// narrow fail-closed batch guard without reversing the context dependency.
// A failure here is interpreted by the caller as "externally owned", so a CSV
// row never silently overrides a stronger upstream authority.
type GroupOwnershipGuard struct {
	Repository scimports.ScimRepository
}

func (g GroupOwnershipGuard) SourceManagedGroupIDs(ctx context.Context, tenantID string, groupIDs []string) (map[string]bool, error) {
	refs, err := g.Repository.FindGroupRefsByGroupIDs(ctx, tenantID, groupIDs)
	if err != nil {
		return nil, err
	}
	managed := make(map[string]bool, len(refs))
	for _, ref := range refs {
		managed[ref.GroupID] = true
	}
	return managed, nil
}

var _ groupports.GroupSourceOwnershipGuard = GroupOwnershipGuard{}
