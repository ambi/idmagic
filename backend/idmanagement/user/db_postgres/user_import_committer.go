package db_postgres

import (
	"context"
	"encoding/json"

	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	tenancypostgres "github.com/ambi/idmagic/backend/tenancy/db_postgres"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// UserImportRowCommitter persists the complete row write set and its safe
// audit record in one PostgreSQL transaction.
type UserImportRowCommitter struct{ Pool sharedpg.DB }

func (c UserImportRowCommitter) CommitUserImportRow(ctx context.Context, mutation userports.UserImportRowMutation) error {
	tx, err := c.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // commit path makes rollback a no-op
	if mutation.ConsumesUserQuota {
		if err := tenancypostgres.NewQuotaRepository(tx).CheckAndIncrement(ctx, mutation.After.TenantID, tenancydomain.ResourceUsers, 1); err != nil {
			return err
		}
	}
	roles, err := json.Marshal(mutation.After.Roles)
	if err != nil {
		return err
	}
	lifecycle, err := json.Marshal(mutation.After.Lifecycle)
	if err != nil {
		return err
	}
	attributes, err := json.Marshal(mutation.After.Attributes)
	if err != nil {
		return err
	}
	if err := New(tx).SaveUser(ctx, SaveUserParams{
		ID: mutation.After.ID, TenantID: mutation.After.TenantID, PreferredUsername: mutation.After.PreferredUsername,
		PasswordHash: mutation.After.PasswordHash, Name: textOrNil(mutation.After.Name), GivenName: textOrNil(mutation.After.GivenName),
		FamilyName: textOrNil(mutation.After.FamilyName), Email: textOrNil(mutation.After.Email), EmailVerified: mutation.After.EmailVerified,
		MfaEnrolled: mutation.After.MfaEnrolled, CreatedAt: mutation.After.CreatedAt, UpdatedAt: mutation.After.UpdatedAt,
		Roles: roles, Lifecycle: lifecycle, Attributes: attributes,
	}); err != nil {
		return err
	}
	if mutation.PasswordHistoryHash != "" {
		historyID, err := spec.NewUUIDv4()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO password_history (id, user_id, encoded, created_at) VALUES ($1, $2, $3, $4)`, historyID, mutation.After.ID, mutation.PasswordHistoryHash, mutation.Now); err != nil {
			return err
		}
	}
	auditID, err := spec.NewUUIDv4()
	if err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{
		"actorUserId": mutation.ActorUserID, "targetUserId": mutation.After.ID, "changedFields": mutation.Changed,
	})
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO audit_events (id, tenant_id, type, user_id, occurred_at, payload)
        VALUES ($1, $2, $3, $4, $5, $6)`, auditID, mutation.After.TenantID, mutation.AuditEventType, mutation.After.ID, mutation.Now, payload); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var _ userports.UserImportRowCommitter = UserImportRowCommitter{}
