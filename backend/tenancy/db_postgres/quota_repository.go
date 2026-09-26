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

// usageCounter は一つの資源の利用量を加算、減算する問い合わせの組である。
// 列ごとに sqlc の問い合わせを分けたので、資源名から列を選ぶ対応はこの表だけが持つ。
type usageCounter struct {
	increment func(ctx context.Context, q *Queries, tenantID string, delta, limit int32) error
	decrement func(ctx context.Context, q *Queries, tenantID string, delta int32) error
}

var usageCounters = map[string]usageCounter{
	domain.ResourceUsers: {
		increment: func(ctx context.Context, q *Queries, tenantID string, delta, limit int32) error {
			_, err := q.IncrementTenantUsageUsers(ctx, IncrementTenantUsageUsersParams{TenantID: tenantID, Delta: delta, DefaultLimit: limit})
			return err
		},
		decrement: func(ctx context.Context, q *Queries, tenantID string, delta int32) error {
			return q.DecrementTenantUsageUsers(ctx, DecrementTenantUsageUsersParams{TenantID: tenantID, Delta: delta})
		},
	},
	domain.ResourceGroups: {
		increment: func(ctx context.Context, q *Queries, tenantID string, delta, limit int32) error {
			_, err := q.IncrementTenantUsageGroups(ctx, IncrementTenantUsageGroupsParams{TenantID: tenantID, Delta: delta, DefaultLimit: limit})
			return err
		},
		decrement: func(ctx context.Context, q *Queries, tenantID string, delta int32) error {
			return q.DecrementTenantUsageGroups(ctx, DecrementTenantUsageGroupsParams{TenantID: tenantID, Delta: delta})
		},
	},
	domain.ResourceAgents: {
		increment: func(ctx context.Context, q *Queries, tenantID string, delta, limit int32) error {
			_, err := q.IncrementTenantUsageAgents(ctx, IncrementTenantUsageAgentsParams{TenantID: tenantID, Delta: delta, DefaultLimit: limit})
			return err
		},
		decrement: func(ctx context.Context, q *Queries, tenantID string, delta int32) error {
			return q.DecrementTenantUsageAgents(ctx, DecrementTenantUsageAgentsParams{TenantID: tenantID, Delta: delta})
		},
	},
	domain.ResourceApplications: {
		increment: func(ctx context.Context, q *Queries, tenantID string, delta, limit int32) error {
			_, err := q.IncrementTenantUsageApplications(ctx, IncrementTenantUsageApplicationsParams{TenantID: tenantID, Delta: delta, DefaultLimit: limit})
			return err
		},
		decrement: func(ctx context.Context, q *Queries, tenantID string, delta int32) error {
			return q.DecrementTenantUsageApplications(ctx, DecrementTenantUsageApplicationsParams{TenantID: tenantID, Delta: delta})
		},
	},
	domain.ResourceOAuth2Clients: {
		increment: func(ctx context.Context, q *Queries, tenantID string, delta, limit int32) error {
			_, err := q.IncrementTenantUsageOAuth2Clients(ctx, IncrementTenantUsageOAuth2ClientsParams{TenantID: tenantID, Delta: delta, DefaultLimit: limit})
			return err
		},
		decrement: func(ctx context.Context, q *Queries, tenantID string, delta int32) error {
			return q.DecrementTenantUsageOAuth2Clients(ctx, DecrementTenantUsageOAuth2ClientsParams{TenantID: tenantID, Delta: delta})
		},
	},
	domain.ResourceActiveSessions: {
		increment: func(ctx context.Context, q *Queries, tenantID string, delta, limit int32) error {
			_, err := q.IncrementTenantUsageActiveSessions(ctx, IncrementTenantUsageActiveSessionsParams{TenantID: tenantID, Delta: delta, DefaultLimit: limit})
			return err
		},
		decrement: func(ctx context.Context, q *Queries, tenantID string, delta int32) error {
			return q.DecrementTenantUsageActiveSessions(ctx, DecrementTenantUsageActiveSessionsParams{TenantID: tenantID, Delta: delta})
		},
	},
	domain.ResourceConsents: {
		increment: func(ctx context.Context, q *Queries, tenantID string, delta, limit int32) error {
			_, err := q.IncrementTenantUsageConsents(ctx, IncrementTenantUsageConsentsParams{TenantID: tenantID, Delta: delta, DefaultLimit: limit})
			return err
		},
		decrement: func(ctx context.Context, q *Queries, tenantID string, delta int32) error {
			return q.DecrementTenantUsageConsents(ctx, DecrementTenantUsageConsentsParams{TenantID: tenantID, Delta: delta})
		},
	},
	domain.ResourceActiveJobs: {
		increment: func(ctx context.Context, q *Queries, tenantID string, delta, limit int32) error {
			_, err := q.IncrementTenantUsageActiveJobs(ctx, IncrementTenantUsageActiveJobsParams{TenantID: tenantID, Delta: delta, DefaultLimit: limit})
			return err
		},
		decrement: func(ctx context.Context, q *Queries, tenantID string, delta int32) error {
			return q.DecrementTenantUsageActiveJobs(ctx, DecrementTenantUsageActiveJobsParams{TenantID: tenantID, Delta: delta})
		},
	},
	domain.ResourceSsfStreams: {
		increment: func(ctx context.Context, q *Queries, tenantID string, delta, limit int32) error {
			_, err := q.IncrementTenantUsageSsfStreams(ctx, IncrementTenantUsageSsfStreamsParams{TenantID: tenantID, Delta: delta, DefaultLimit: limit})
			return err
		},
		decrement: func(ctx context.Context, q *Queries, tenantID string, delta int32) error {
			return q.DecrementTenantUsageSsfStreams(ctx, DecrementTenantUsageSsfStreamsParams{TenantID: tenantID, Delta: delta})
		},
	},
	"audit_events_retained": {
		increment: func(ctx context.Context, q *Queries, tenantID string, delta, limit int32) error {
			_, err := q.IncrementTenantUsageAuditEventsRetained(ctx, IncrementTenantUsageAuditEventsRetainedParams{TenantID: tenantID, Delta: delta, DefaultLimit: limit})
			return err
		},
		decrement: func(ctx context.Context, q *Queries, tenantID string, delta int32) error {
			return q.DecrementTenantUsageAuditEventsRetained(ctx, DecrementTenantUsageAuditEventsRetainedParams{TenantID: tenantID, Delta: delta})
		},
	},
	"export_artifacts_bytes": {
		increment: func(ctx context.Context, q *Queries, tenantID string, delta, limit int32) error {
			_, err := q.IncrementTenantUsageExportArtifactsBytes(ctx, IncrementTenantUsageExportArtifactsBytesParams{TenantID: tenantID, Delta: delta, DefaultLimit: limit})
			return err
		},
		decrement: func(ctx context.Context, q *Queries, tenantID string, delta int32) error {
			return q.DecrementTenantUsageExportArtifactsBytes(ctx, DecrementTenantUsageExportArtifactsBytesParams{TenantID: tenantID, Delta: delta})
		},
	},
}

// CheckAndIncrement atomically increments the usage counter for the given resource.
func (r *QuotaRepository) CheckAndIncrement(ctx context.Context, tenantID, resource string, delta int) error {
	counter, ok := usageCounters[resource]
	if !ok {
		return fmt.Errorf("unknown resource for quota increment: %s", resource)
	}
	queries := New(r.db)
	if err := queries.EnsureTenantUsage(ctx, tenantID); err != nil {
		return err
	}
	//nolint:gosec // 利用量の列は INT であり、呼び出し側の delta と既定の上限はその範囲に収まる
	err := counter.increment(ctx, queries, tenantID, int32(delta), int32(getDefaultQuota(resource)))
	if errors.Is(err, pgx.ErrNoRows) {
		return &domain.QuotaExceededError{TenantID: tenantID, Resource: resource}
	}
	return err
}

// Decrement atomically decreases the usage counter for the given resource.
func (r *QuotaRepository) Decrement(ctx context.Context, tenantID, resource string, delta int) error {
	counter, ok := usageCounters[resource]
	if !ok {
		return fmt.Errorf("unknown resource for quota decrement: %s", resource)
	}
	//nolint:gosec // 利用量の列は INT であり、呼び出し側の delta はその範囲に収まる
	return counter.decrement(ctx, New(r.db), tenantID, int32(delta))
}

// getDefaultQuota resolves the baseline limit for resource through
// domain.DefaultTenantQuota (the single source of truth shared with the
// memory backend's TenantQuota.EffectiveLimit) so the two persistence
// implementations cannot drift apart. Soft-quota / undefined resources fall
// back to 0 (unused by the Hard Quota enforcement paths that call this).
func getDefaultQuota(resource string) int {
	return domain.DefaultTenantQuota[resource]
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
		SsfStreams:           toPgtypeInt4(quota.SsfStreams),
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
		SsfStreams:           fromPgtype(row.SsfStreams),
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
		SsfStreams:           int(row.SsfStreams),
		AuditEventsRetained:  int(row.AuditEventsRetained),
		ExportArtifactsBytes: int(row.ExportArtifactsBytes),
	}, nil
}
