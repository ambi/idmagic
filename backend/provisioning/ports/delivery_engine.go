package ports

import (
	"context"

	"github.com/ambi/idmagic/backend/provisioning/domain"
)

// AttributeSource resolves a source aggregate's current attributes for the
// mapping engine (spec/contexts/provisioning.yaml models.AttributeMappingRule
// source_kind=attribute). exists=false means the source aggregate itself is
// gone (e.g. hard-deleted), distinct from individual missing attributes.
type AttributeSource interface {
	ResolveAttributes(ctx context.Context, tenantID string, sourceType domain.ProvisioningSourceType, sourceID string) (attrs map[string]any, exists bool, err error)
}

// ProvisioningTargetClient is the protocol seam (decision 2): each
// protocol feature slice (backend/provisioning/client_scim, a future .../entraid)
// implements this against its own wire format. The delivery engine usecase
// depends only on this port, never on a concrete protocol package.
type ProvisioningTargetClient interface {
	Discover(ctx context.Context) (domain.ProvisioningCapabilities, error)

	CreateUser(ctx context.Context, rules []domain.AttributeMappingRule, attrs map[string]any) (remoteID string, etag *string, err error)
	UpdateUser(ctx context.Context, remoteID string, rules []domain.AttributeMappingRule, attrs map[string]any, supportsPatch bool) (etag *string, err error)
	DeleteUser(ctx context.Context, remoteID string) error
	SearchUserByAttribute(ctx context.Context, attribute, value string) (remoteID string, found bool, err error)

	CreateGroup(ctx context.Context, rules []domain.AttributeMappingRule, attrs map[string]any) (remoteID string, etag *string, err error)
	UpdateGroup(ctx context.Context, remoteID string, rules []domain.AttributeMappingRule, attrs map[string]any, supportsPatch bool) (etag *string, err error)
	DeleteGroup(ctx context.Context, remoteID string) error
	SearchGroupByAttribute(ctx context.Context, attribute, value string) (remoteID string, found bool, err error)
	// PatchGroupMembers sends one incremental add/remove against a Group's
	// `members`. op is "add" or "remove". The Group's own attributes go through
	// UpdateGroup; membership moves separately because it is a set operation and
	// replacing it wholesale would delete downstream members IdMagic cannot see.
	PatchGroupMembers(ctx context.Context, remoteGroupID, op string, remoteUserIDs []string) error
}

// GroupMemberSource reads a Group's current direct members. Membership is not a
// mapped attribute — the mapping engine produces a resource document, while
// membership is pushed as its own PATCH — so the delivery engine reads it
// through this port rather than through AttributeSource.
type GroupMemberSource interface {
	ListMemberUserIDs(ctx context.Context, tenantID, groupID string) ([]string, error)
}
