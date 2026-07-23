// Package ports は Application bounded context の永続境界 (port) を定義する (wi-69)。
package ports

import (
	"context"

	"github.com/ambi/idmagic/backend/application/domain"
)

// ApplicationRepository は Application aggregate の永続境界 (wi-69)。
type ApplicationRepository interface {
	// ListByTenant はテナント内の Application を name 昇順で返す。
	ListByTenant(ctx context.Context, tenantID string) ([]*domain.Application, error)
	// FindByID は application_id に一致する Application を返す。存在しなければ (nil, nil)。
	FindByID(ctx context.Context, tenantID, applicationID string) (*domain.Application, error)
	// FindByProtocol は protocol table の application_id relation を索引で逆引きする。
	// catalog 外 protocol record なら nil を返す。
	FindByProtocol(ctx context.Context, tenantID string, protocolType domain.ApplicationProtocolType, key string) (*domain.Application, error)
	// Save は Application を upsert する。
	Save(ctx context.Context, app *domain.Application) error
	// Create は Application 行と protocol row.application_id を原子的に作成する。
	Create(ctx context.Context, app *domain.Application) error
	// Delete は application_id に一致する Application を削除する (冪等)。
	Delete(ctx context.Context, tenantID, applicationID string) error
	// RemoveCategory はテナント内の全 Application の category_ids から指定カテゴリを除く
	// (カテゴリ削除時のクリーンアップ, wi-70)。
	RemoveCategory(ctx context.Context, tenantID, categoryID string) error
}

// ApplicationIconStore は Application icon blob の保存境界。
// Application aggregate は icon_object_key だけを持ち、binary 本体はこの store が所有する。
type ApplicationIconStore interface {
	Save(ctx context.Context, icon *domain.ApplicationIcon) error
	Find(ctx context.Context, tenantID, applicationID, objectKey string) (*domain.ApplicationIcon, error)
	DeleteByApplication(ctx context.Context, tenantID, applicationID string) error
}

// ApplicationCategoryRepository は ApplicationCategory の永続境界 (wi-70, ADR-069)。
type ApplicationCategoryRepository interface {
	// ListByTenant はテナント内のカテゴリを position 昇順 (同値は name 昇順) で返す。
	ListByTenant(ctx context.Context, tenantID string) ([]*domain.ApplicationCategory, error)
	// FindByID は category_id に一致するカテゴリを返す。存在しなければ (nil, nil)。
	FindByID(ctx context.Context, tenantID, categoryID string) (*domain.ApplicationCategory, error)
	// Save はカテゴリを upsert する。
	Save(ctx context.Context, category *domain.ApplicationCategory) error
	// Delete は category_id に一致するカテゴリを削除する (冪等)。
	Delete(ctx context.Context, tenantID, categoryID string) error
}

// SubjectRef は割当の対象 (user / group) を表す参照 (wi-69)。
type SubjectRef struct {
	Type domain.AssignmentSubjectType
	ID   string
}

// ApplicationOrderingRepository は利用者ごとのポータル手動並び順の永続境界 (wi-70, ADR-069)。
type ApplicationOrderingRepository interface {
	// Get は利用者の手動並び順 (application_id の順序列) を返す。未保存なら (nil, nil)。
	Get(ctx context.Context, tenantID, userID string) (*domain.ApplicationOrdering, error)
	// Save は利用者の手動並び順を upsert する。
	Save(ctx context.Context, ordering *domain.ApplicationOrdering) error
}

// AssignmentRepository は Application 割当の永続境界 (wi-69)。
type AssignmentRepository interface {
	// ListByApplication は Application の割当を subject 昇順で返す。
	ListByApplication(ctx context.Context, tenantID, applicationID string) ([]*domain.ApplicationAssignment, error)
	// ListBySubjects は指定 subject 群に一致する割当を返す (ポータル一覧・割当ゲート用)。
	ListBySubjects(ctx context.Context, tenantID string, subjects []SubjectRef) ([]*domain.ApplicationAssignment, error)
	// ListByTenant はテナント内のすべての Application 割当を返す。
	ListByTenant(ctx context.Context, tenantID string) ([]*domain.ApplicationAssignment, error)
	// Save は割当を upsert する。
	Save(ctx context.Context, assignment *domain.ApplicationAssignment) error
	// Delete は 1 件の割当を削除する (冪等)。
	Delete(ctx context.Context, tenantID, applicationID string, subjectType domain.AssignmentSubjectType, subjectID string) error
	// DeleteByApplication は Application の全割当を削除する (Application 削除時のクリーンアップ)。
	DeleteByApplication(ctx context.Context, tenantID, applicationID string) error
}

// SignInPolicyRepository は Application sign-in policy の永続境界。
type SignInPolicyRepository interface {
	// Get は application_id に一致する policy を返す。未設定なら (nil, nil)。
	Get(ctx context.Context, tenantID, applicationID string) (*domain.AppSignInPolicy, error)
	// ListByTenant はテナント内のすべての Application sign-in policy を返す。
	ListByTenant(ctx context.Context, tenantID string) ([]*domain.AppSignInPolicy, error)
	// Save は policy を upsert する。
	Save(ctx context.Context, policy *domain.AppSignInPolicy) error
	// Delete は application_id に一致する policy を削除する (冪等)。
	Delete(ctx context.Context, tenantID, applicationID string) error
}

// DefaultSignInPolicyRepository はテナント既定 sign-in policy の永続境界 (ADR-081)。
type DefaultSignInPolicyRepository interface {
	// Get は tenant_id に一致する既定 policy を返す。未設定なら (nil, nil)。
	Get(ctx context.Context, tenantID string) (*domain.TenantDefaultSignInPolicy, error)
	// Save は既定 policy を upsert する。
	Save(ctx context.Context, policy *domain.TenantDefaultSignInPolicy) error
}
