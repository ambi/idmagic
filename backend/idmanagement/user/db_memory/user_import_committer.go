package db_memory

import (
	"context"

	auditports "github.com/ambi/idmagic/backend/audit/ports"
	passwordports "github.com/ambi/idmagic/backend/authentication/password/ports"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// UserImportRowCommitter is the lightweight runtime adapter. Its mutation
// methods cannot fail after the quota precondition in the memory backend.
type UserImportRowCommitter struct {
	Users           userports.UserRepository
	PasswordHistory passwordports.PasswordHistoryRepository
	Quota           tenantports.QuotaRepository
	Audit           auditports.AuditEventRepository
}

func (c UserImportRowCommitter) CommitUserImportRow(ctx context.Context, mutation userports.UserImportRowMutation) error {
	if mutation.ConsumesUserQuota && c.Quota != nil {
		if err := c.Quota.CheckAndIncrement(ctx, mutation.After.TenantID, tenancydomain.ResourceUsers, 1); err != nil {
			return err
		}
	}
	if err := c.Users.Save(ctx, mutation.After); err != nil {
		if mutation.ConsumesUserQuota && c.Quota != nil {
			_ = c.Quota.Decrement(ctx, mutation.After.TenantID, tenancydomain.ResourceUsers, 1)
		}
		return err
	}
	if mutation.PasswordHistoryHash != "" && c.PasswordHistory != nil {
		if err := c.PasswordHistory.Add(ctx, mutation.After.ID, mutation.PasswordHistoryHash, mutation.Now); err != nil {
			return err
		}
	}
	if c.Audit != nil {
		auditID, err := spec.NewUUIDv4()
		if err != nil {
			return err
		}
		if err := c.Audit.Append(ctx, &auditports.AuditEventRecord{
			ID: auditID, TenantID: mutation.After.TenantID, Type: mutation.AuditEventType, OccurredAt: mutation.Now,
			Payload: map[string]any{"actorUserId": mutation.ActorUserID, "targetUserId": mutation.After.ID, "changedFields": mutation.Changed},
		}); err != nil {
			return err
		}
	}
	return nil
}

var _ userports.UserImportRowCommitter = UserImportRowCommitter{}
