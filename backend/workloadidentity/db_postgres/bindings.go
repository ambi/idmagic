package db_postgres

import (
	"context"
	"errors"
	"time"

	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
	"github.com/jackc/pgx/v5"
)

// AgentWorkloadBindingRepository は AgentWorkloadBinding (ADR-053) を PostgreSQL に
// 永続化する。クエリは sqlc 生成 (ADR-090)。
type AgentWorkloadBindingRepository struct{ Pool sharedpg.DB }

func bindingFromRow(row *AgentWorkloadBinding) (*workloaddomain.AgentWorkloadBinding, error) {
	b := &workloaddomain.AgentWorkloadBinding{
		ID: row.ID, TenantID: row.TenantID, TrustBundleID: row.TrustBundleID,
		SubjectPattern: row.SubjectPattern, AgentID: row.AgentID,
		Status:    workloaddomain.AgentWorkloadBindingStatus(row.Status),
		CreatedAt: row.CreatedAt, UpdatedAt: &row.UpdatedAt,
		DisabledAt: timestamptzPtrOrNil(row.DisabledAt),
	}
	return b, b.Validate()
}

func (r *AgentWorkloadBindingRepository) ListByTrustBundle(ctx context.Context, tenantID, trustBundleID string) ([]*workloaddomain.AgentWorkloadBinding, error) {
	rows, err := New(r.Pool).ListAgentWorkloadBindingsByTrustBundle(ctx, ListAgentWorkloadBindingsByTrustBundleParams{
		TenantID: tenantID, TrustBundleID: trustBundleID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*workloaddomain.AgentWorkloadBinding, 0, len(rows))
	for _, row := range rows {
		b, err := bindingFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}

func (r *AgentWorkloadBindingRepository) FindByID(ctx context.Context, tenantID, id string) (*workloaddomain.AgentWorkloadBinding, error) {
	row, err := New(r.Pool).FindAgentWorkloadBindingByID(ctx, FindAgentWorkloadBindingByIDParams{TenantID: tenantID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return bindingFromRow(row)
}

func (r *AgentWorkloadBindingRepository) Save(ctx context.Context, b *workloaddomain.AgentWorkloadBinding) error {
	updatedAt := time.Now().UTC()
	if b.UpdatedAt != nil {
		updatedAt = *b.UpdatedAt
	}
	return New(r.Pool).SaveAgentWorkloadBinding(ctx, SaveAgentWorkloadBindingParams{
		ID: b.ID, TenantID: b.TenantID, TrustBundleID: b.TrustBundleID, SubjectPattern: b.SubjectPattern,
		AgentID: b.AgentID, Status: string(b.Status), CreatedAt: b.CreatedAt, UpdatedAt: updatedAt,
		DisabledAt: timestamptzOrNilPtr(b.DisabledAt),
	})
}

func (r *AgentWorkloadBindingRepository) Delete(ctx context.Context, tenantID, id string) error {
	return New(r.Pool).DeleteAgentWorkloadBinding(ctx, DeleteAgentWorkloadBindingParams{TenantID: tenantID, ID: id})
}
