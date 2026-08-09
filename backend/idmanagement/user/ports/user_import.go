package ports

import (
	"context"
	"time"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

// EffectiveUserAttributeSchemaReader supplies the attribute definitions that
// may appear as custom:<key> columns without exposing Tenancy internals to the
// User import use case.
type EffectiveUserAttributeSchemaReader interface {
	EffectiveUserAttributeDefs(ctx context.Context, tenantID string) ([]userdomain.UserAttributeDef, error)
}

// UserSourceOwnershipGuard resolves source ownership in batches. A missing or
// failed guard is interpreted fail-closed by the import planner.
type UserSourceOwnershipGuard interface {
	SourceManagedUserIDs(ctx context.Context, tenantID string, userIDs []string) (map[string]bool, error)
}

// UserImportRowMutation is the complete atomic write set for one accepted CSV
// row. The adapter commits the aggregate, password history/quota side effects,
// and audit event in one transaction.
type UserImportRowMutation struct {
	Before              *userdomain.User
	After               *userdomain.User
	Changed             []string
	ActorUserID         string
	AuditEventType      string
	PasswordHistoryHash string
	ConsumesUserQuota   bool
	Now                 time.Time
}

type UserImportRowCommitter interface {
	CommitUserImportRow(ctx context.Context, mutation UserImportRowMutation) error
}
