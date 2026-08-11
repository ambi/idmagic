package ports

import (
	"context"

	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
)

// WorkloadTrustBundleRepository は tenant-scoped な WorkloadTrustBundle を永続化する。
// すべての操作はテナント境界に閉じ、cross-tenant 参照は use case 側で
// reject する。
type WorkloadTrustBundleRepository interface {
	ListAll(ctx context.Context, tenantID string) ([]*workloaddomain.WorkloadTrustBundle, error)
	FindByID(ctx context.Context, tenantID, id string) (*workloaddomain.WorkloadTrustBundle, error)
	// FindByIssuer はテナント内で issuer に一致する WorkloadTrustBundle を返す (issuer は
	// テナント内で一意)。無ければ nil を返す。VerifyWorkloadAttestation の起点となる
	// issuer 解決に使う (未登録 issuer は fail-closed で拒否する)。
	FindByIssuer(ctx context.Context, tenantID, issuer string) (*workloaddomain.WorkloadTrustBundle, error)
	Save(ctx context.Context, bundle *workloaddomain.WorkloadTrustBundle) error
	// Delete は WorkloadTrustBundle 単体を削除する。配下の AgentWorkloadBinding の
	// cascade 削除は usecase 層が AgentWorkloadBindingRepository を通じて行う責務。
	Delete(ctx context.Context, tenantID, id string) error
}

// AgentWorkloadBindingRepository は tenant-scoped な AgentWorkloadBinding を永続化する。
type AgentWorkloadBindingRepository interface {
	ListByTrustBundle(ctx context.Context, tenantID, trustBundleID string) ([]*workloaddomain.AgentWorkloadBinding, error)
	FindByID(ctx context.Context, tenantID, id string) (*workloaddomain.AgentWorkloadBinding, error)
	Save(ctx context.Context, binding *workloaddomain.AgentWorkloadBinding) error
	Delete(ctx context.Context, tenantID, id string) error
}
