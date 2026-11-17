package ports

import (
	"context"
	"errors"
	"time"

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
	// FindSPIFFEByTrustDomain はテナント内で trustDomain を持つ spiffe の
	// WorkloadTrustBundle を返す (spiffe の信頼設定の間でトラストドメインは一意)。
	// oidc の信頼設定は、同じラベルを持っていても返さない。無ければ nil を返す。
	FindSPIFFEByTrustDomain(ctx context.Context, tenantID, trustDomain string) (*workloaddomain.WorkloadTrustBundle, error)
	Save(ctx context.Context, bundle *workloaddomain.WorkloadTrustBundle) error
	// RecordJWKSFetched は jwksURI から取得できた鍵の集合と取得時刻だけを記録し、取得中の
	// 設定変更を保持する。行の jwks_uri が jwksURI と異なるか、保存済みの時刻が取得時刻より
	// 新しければ書かず、古い取得元や遅れて終わった取得が新しい内容を書き換えない。
	// 存在しない行は作らない。
	RecordJWKSFetched(ctx context.Context, tenantID, id string, jwksURI *string, fetched workloaddomain.JWKSSnapshot) error
	// Delete は WorkloadTrustBundle 単体を削除する。配下の AgentWorkloadBinding の
	// cascade 削除は usecase 層が AgentWorkloadBindingRepository を通じて行う責務。
	Delete(ctx context.Context, tenantID, id string) error
}

// AgentWorkloadBindingRepository は tenant-scoped な AgentWorkloadBinding を永続化する。
type AgentWorkloadBindingRepository interface {
	ListByTrustBundle(ctx context.Context, tenantID, trustBundleID string) ([]*workloaddomain.AgentWorkloadBinding, error)
	FindByID(ctx context.Context, tenantID, id string) (*workloaddomain.AgentWorkloadBinding, error)
	Save(ctx context.Context, binding *workloaddomain.AgentWorkloadBinding) error
	// UpdateMapping は対応付けだけを書き、並行した状態変更を保つ。削除済みの行は作らない。
	UpdateMapping(ctx context.Context, tenantID, id string, pattern, agentID *string, at time.Time) (*workloaddomain.AgentWorkloadBinding, error)
	Delete(ctx context.Context, tenantID, id string) error
}

var ErrBindingPatternConflict = errors.New("subject_pattern already registered for this trust bundle")
