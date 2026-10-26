package db_postgres

import (
	"context"
	"errors"
	"fmt"
	"math"

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

// QuotaRepositoryInTx は、呼び出し側のトランザクションで使用量を加減算する QuotaRepository を返す。
// ほかのモジュールの書き込みと使用量を同じトランザクションで確定させるため、組み立て地点が渡す。
func QuotaRepositoryInTx(tx pgx.Tx) tenantports.QuotaRepository {
	return NewQuotaRepository(tx)
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
}

// CheckAndIncrement atomically increments the usage counter for the given resource.
func (r *QuotaRepository) CheckAndIncrement(ctx context.Context, tenantID, resource string, delta int) error {
	if domain.IsSoftQuota(resource) {
		if delta < 0 {
			return errors.New("soft quota: negative increment")
		}
		queries := New(r.db)
		if err := queries.EnsureTenantUsage(ctx, tenantID); err != nil {
			return err
		}
		if resource == domain.ResourceAuditEventsRetained {
			_, err := queries.IncrementTenantUsageAuditEventsRetained(ctx, IncrementTenantUsageAuditEventsRetainedParams{TenantID: tenantID, Delta: int64(delta)})
			return err
		}
		_, err := queries.IncrementTenantUsageExportArtifactsBytes(ctx, IncrementTenantUsageExportArtifactsBytesParams{TenantID: tenantID, Delta: int64(delta)})
		return err
	}
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
	if domain.IsSoftQuota(resource) {
		if delta < 0 {
			return errors.New("soft quota: negative decrement")
		}
		if resource == domain.ResourceAuditEventsRetained {
			return New(r.db).DecrementTenantUsageAuditEventsRetained(ctx, DecrementTenantUsageAuditEventsRetainedParams{TenantID: tenantID, Delta: int64(delta)})
		}
		return New(r.db).DecrementTenantUsageExportArtifactsBytes(ctx, DecrementTenantUsageExportArtifactsBytesParams{TenantID: tenantID, Delta: int64(delta)})
	}
	counter, ok := usageCounters[resource]
	if !ok {
		return fmt.Errorf("unknown resource for quota decrement: %s", resource)
	}
	//nolint:gosec // 利用量の列は INT であり、呼び出し側の delta はその範囲に収まる
	return counter.decrement(ctx, New(r.db), tenantID, int32(delta))
}

// getDefaultQuota は Hard の SQL に渡す既定値をドメインの同じ定義から解決する。
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
		AuditEventsRetained:  toPgtypeInt8(quota.AuditEventsRetained),
		ExportArtifactsBytes: toPgtypeInt8(quota.ExportArtifactsBytes),
	})
}

func (r *QuotaRepository) SetSoftUsage(ctx context.Context, tenantID string, measured domain.SoftQuotaUsage) error {
	if err := measured.Validate(); err != nil {
		return err
	}
	return New(r.db).SetSoftTenantUsage(ctx, SetSoftTenantUsageParams{
		TenantID: tenantID, AuditEventsRetained: int64(measured.AuditEventsRetained), ExportArtifactsBytes: int64(measured.ExportArtifactsBytes),
	})
}

// RecountUsage は自分のトランザクションで使用量の行を固定するので、トランザクションを始められる
// 接続で構築したリポジトリだけが受け付ける。count は別の接続で数えてよい。
func (r *QuotaRepository) RecountUsage(ctx context.Context, tenantID, resource string, count func(context.Context) (int, error)) (int, error) {
	if resource != domain.ResourceUsers {
		return 0, fmt.Errorf("usage recount is not supported for %s", resource)
	}
	beginner, ok := r.db.(interface {
		Begin(ctx context.Context) (pgx.Tx, error)
	})
	if !ok {
		return 0, errors.New("usage recount: requires a database that begins transactions")
	}
	tx, err := beginner.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // commit path makes rollback a no-op
	queries := New(tx)
	if err := queries.EnsureTenantUsage(ctx, tenantID); err != nil {
		return 0, err
	}
	previous, err := queries.LockTenantUsageUsers(ctx, tenantID)
	if err != nil {
		return 0, err
	}
	counted, err := count(ctx)
	if err != nil {
		return 0, err
	}
	if counted < 0 || counted > math.MaxInt32 {
		return 0, fmt.Errorf("usage recount: count %d is outside the usage column", counted)
	}
	if err := queries.SetTenantUsageUsers(ctx, SetTenantUsageUsersParams{TenantID: tenantID, Users: int32(counted)}); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return int(previous), nil
}

func toPgtypeInt4(v *int) pgtype.Int4 {
	if v == nil {
		return pgtype.Int4{Valid: false}
	}
	//nolint:gosec // we trust our input size
	return pgtype.Int4{Int32: int32(*v), Valid: true}
}

func toPgtypeInt8(v *int) pgtype.Int8 {
	if v == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: int64(*v), Valid: true}
}

func fromPgtypeInt8(v pgtype.Int8) *int {
	if !v.Valid {
		return nil
	}
	value := int(v.Int64)
	return &value
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
		AuditEventsRetained:  fromPgtypeInt8(row.AuditEventsRetained),
		ExportArtifactsBytes: fromPgtypeInt8(row.ExportArtifactsBytes),
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
