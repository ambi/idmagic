package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// WorkloadTrustBundleRepository は WorkloadTrustBundle を PostgreSQL に
// 永続化する。すべての参照はテナント境界に閉じる。クエリは sqlc 生成。
type WorkloadTrustBundleRepository struct{ Pool sharedpg.DB }

func textPtrOrNil(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func textOrNilPtr(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func timestamptzPtrOrNil(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	tm := t.Time
	return &tm
}

func timestamptzOrNilPtr(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func trustBundleFromRow(row *WorkloadTrustBundle) (*workloaddomain.WorkloadTrustBundle, error) {
	b := &workloaddomain.WorkloadTrustBundle{
		ID: row.ID, TenantID: row.TenantID, Name: row.Name, TrustDomain: row.TrustDomain,
		Issuer: row.Issuer, JWKSURI: textPtrOrNil(row.JwksUri),
		MaxSubjectTokenTTLSeconds: int(row.MaxSubjectTokenTtlSeconds),
		Status:                    workloaddomain.WorkloadTrustBundleStatus(row.Status),
		CreatedAt:                 row.CreatedAt, UpdatedAt: &row.UpdatedAt,
		JWKSCachedAt: timestamptzPtrOrNil(row.JwksCachedAt),
	}
	if len(row.Jwks) > 0 {
		var jwks map[string]any
		if err := json.Unmarshal(row.Jwks, &jwks); err != nil {
			return nil, err
		}
		b.JWKS = jwks
	}
	if err := json.Unmarshal(row.AcceptedAudiences, &b.AcceptedAudiences); err != nil {
		return nil, err
	}
	return b, b.Validate()
}

func (r *WorkloadTrustBundleRepository) ListAll(ctx context.Context, tenantID string) ([]*workloaddomain.WorkloadTrustBundle, error) {
	rows, err := New(r.Pool).ListWorkloadTrustBundlesByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*workloaddomain.WorkloadTrustBundle, 0, len(rows))
	for _, row := range rows {
		b, err := trustBundleFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

func (r *WorkloadTrustBundleRepository) FindByID(ctx context.Context, tenantID, id string) (*workloaddomain.WorkloadTrustBundle, error) {
	row, err := New(r.Pool).FindWorkloadTrustBundleByID(ctx, FindWorkloadTrustBundleByIDParams{TenantID: tenantID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return trustBundleFromRow(row)
}

func (r *WorkloadTrustBundleRepository) FindByIssuer(ctx context.Context, tenantID, issuer string) (*workloaddomain.WorkloadTrustBundle, error) {
	row, err := New(r.Pool).FindWorkloadTrustBundleByIssuer(ctx, FindWorkloadTrustBundleByIssuerParams{TenantID: tenantID, Issuer: issuer})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return trustBundleFromRow(row)
}

func (r *WorkloadTrustBundleRepository) Save(ctx context.Context, b *workloaddomain.WorkloadTrustBundle) error {
	audiences := b.AcceptedAudiences
	if audiences == nil {
		audiences = []string{}
	}
	audiencesJSON, err := json.Marshal(audiences)
	if err != nil {
		return err
	}
	var jwksJSON []byte
	if b.JWKS != nil {
		jwksJSON, err = json.Marshal(b.JWKS)
		if err != nil {
			return err
		}
	}
	updatedAt := time.Now().UTC()
	if b.UpdatedAt != nil {
		updatedAt = *b.UpdatedAt
	}
	return New(r.Pool).SaveWorkloadTrustBundle(ctx, SaveWorkloadTrustBundleParams{
		ID: b.ID, TenantID: b.TenantID, Name: b.Name, TrustDomain: b.TrustDomain, Issuer: b.Issuer,
		JwksUri: textOrNilPtr(b.JWKSURI), Jwks: jwksJSON, AcceptedAudiences: audiencesJSON,
		MaxSubjectTokenTtlSeconds: int32(b.MaxSubjectTokenTTLSeconds), //nolint:gosec // G115: TTL seconds is admin-configured and validated positive, well under int32 max
		Status:                    string(b.Status),
		CreatedAt:                 b.CreatedAt, UpdatedAt: updatedAt, JwksCachedAt: timestamptzOrNilPtr(b.JWKSCachedAt),
	})
}

func (r *WorkloadTrustBundleRepository) Delete(ctx context.Context, tenantID, id string) error {
	return New(r.Pool).DeleteWorkloadTrustBundle(ctx, DeleteWorkloadTrustBundleParams{TenantID: tenantID, ID: id})
}
