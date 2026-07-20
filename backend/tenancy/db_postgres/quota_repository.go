package db_postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

type QuotaRepository struct {
	db DBTX
}

var _ tenantports.QuotaRepository = (*QuotaRepository)(nil)

func NewQuotaRepository(db DBTX) *QuotaRepository {
	return &QuotaRepository{db: db}
}

// CheckAndIncrement atomically increments the usage counter for the given resource.
func (r *QuotaRepository) CheckAndIncrement(ctx context.Context, tenantID, resource string, delta int) error {
	var stmt string
	// Validate resource and construct the query
	switch resource {
	case "users", "groups", "agents", "applications", "oauth2_clients", "active_sessions", "consents", "active_jobs", "audit_events_retained", "export_artifacts_bytes":
		// Ensure tenant_usages row exists
		_, err := r.db.Exec(ctx, `
			INSERT INTO tenant_usages (tenant_id) VALUES ($1) ON CONFLICT DO NOTHING
		`, tenantID)
		if err != nil {
			return err
		}

		// The column name is safe because it comes from the switch statement above.
		stmt = fmt.Sprintf(`
			UPDATE tenant_usages 
			SET %[1]s = %[1]s + $2 
			WHERE tenant_id = $1 
			  AND (SELECT COALESCE((SELECT %[1]s FROM tenant_quotas WHERE tenant_id = $1), %d)) >= %[1]s + $2
			RETURNING %[1]s
		`, resource, getDefaultQuota(resource))
	default:
		return fmt.Errorf("unknown resource for quota increment: %s", resource)
	}

	var newVal int
	err := r.db.QueryRow(ctx, stmt, tenantID, delta).Scan(&newVal)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &domain.QuotaExceededError{TenantID: tenantID, Resource: resource}
		}
		return err
	}
	return nil
}

// Decrement atomically decreases the usage counter for the given resource.
func (r *QuotaRepository) Decrement(ctx context.Context, tenantID, resource string, delta int) error {
	var stmt string
	switch resource {
	case "users", "groups", "agents", "applications", "oauth2_clients", "active_sessions", "consents", "active_jobs", "audit_events_retained", "export_artifacts_bytes":
		stmt = fmt.Sprintf(`
			UPDATE tenant_usages 
			SET %[1]s = GREATEST(0, %[1]s - $2) 
			WHERE tenant_id = $1
		`, resource)
	default:
		return fmt.Errorf("unknown resource for quota decrement: %s", resource)
	}
	_, err := r.db.Exec(ctx, stmt, tenantID, delta)
	return err
}

func getDefaultQuota(resource string) int {
	switch resource {
	case "users":
		return 10000
	case "groups":
		return 1000
	case "agents":
		return 100
	case "applications":
		return 50
	case "oauth2_clients":
		return 100
	case "active_sessions":
		return 50000
	case "consents":
		return 10000
	case "active_jobs":
		return 10
	default:
		return 0 // fallback for soft quotas or undefined
	}
}

// SetQuota explicitly sets the quota for a tenant.
func (r *QuotaRepository) SetQuota(ctx context.Context, tenantID string, quota *domain.TenantQuota) error {
	queries := New(r.db)
	return queries.UpsertTenantQuota(ctx, UpsertTenantQuotaParams{
		TenantID:             tenantID,
		Users:                toPgtypeInt4(quota.Users),
		Groups:               toPgtypeInt4(quota.Groups),
		Agents:               toPgtypeInt4(quota.Agents),
		Applications:         toPgtypeInt4(quota.Applications),
		Oauth2Clients:        toPgtypeInt4(quota.OAuth2Clients),
		ActiveSessions:       toPgtypeInt4(quota.ActiveSessions),
		Consents:             toPgtypeInt4(quota.Consents),
		ActiveJobs:           toPgtypeInt4(quota.ActiveJobs),
		AuditEventsRetained:  toPgtypeInt4(quota.AuditEventsRetained),
		ExportArtifactsBytes: toPgtypeInt4(quota.ExportArtifactsBytes),
	})
}

func toPgtypeInt4(v *int) pgtype.Int4 {
	if v == nil {
		return pgtype.Int4{Valid: false}
	}
	//nolint:gosec // we trust our input size
	return pgtype.Int4{Int32: int32(*v), Valid: true}
}

func (r *QuotaRepository) GetQuota(ctx context.Context, tenantID string) (*domain.TenantQuota, error) {
	queries := New(r.db)
	row, err := queries.GetTenantQuota(ctx, tenantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &domain.TenantQuota{}, nil
		}
		return nil, err
	}

	fromPgtype := func(v pgtype.Int4) *int {
		if !v.Valid {
			return nil
		}
		val := int(v.Int32)
		return &val
	}

	return &domain.TenantQuota{
		Users:                fromPgtype(row.Users),
		Groups:               fromPgtype(row.Groups),
		Agents:               fromPgtype(row.Agents),
		Applications:         fromPgtype(row.Applications),
		OAuth2Clients:        fromPgtype(row.Oauth2Clients),
		ActiveSessions:       fromPgtype(row.ActiveSessions),
		Consents:             fromPgtype(row.Consents),
		ActiveJobs:           fromPgtype(row.ActiveJobs),
		AuditEventsRetained:  fromPgtype(row.AuditEventsRetained),
		ExportArtifactsBytes: fromPgtype(row.ExportArtifactsBytes),
	}, nil
}

func (r *QuotaRepository) GetUsage(ctx context.Context, tenantID string) (*domain.TenantUsage, error) {
	queries := New(r.db)
	row, err := queries.GetTenantUsage(ctx, tenantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &domain.TenantUsage{}, nil
		}
		return nil, err
	}

	return &domain.TenantUsage{
		Users:                int(row.Users),
		Groups:               int(row.Groups),
		Agents:               int(row.Agents),
		Applications:         int(row.Applications),
		OAuth2Clients:        int(row.Oauth2Clients),
		ActiveSessions:       int(row.ActiveSessions),
		Consents:             int(row.Consents),
		ActiveJobs:           int(row.ActiveJobs),
		AuditEventsRetained:  int(row.AuditEventsRetained),
		ExportArtifactsBytes: int(row.ExportArtifactsBytes),
	}, nil
}
