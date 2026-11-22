// Package ports は Application モジュールの永続境界 (port) を定義する (wi-69)。
package ports

import (
	"context"
	"errors"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

var ErrApplicationNotFound = errors.New("application not found")

// ApplicationRepository は Application aggregate の永続境界 (wi-69)。
type ApplicationRepository interface {
	// ListAll はテナント内の Application を name 昇順で返す。
	ListAll(ctx context.Context, tenantID string) ([]*appdomain.Application, error)
	// ListPage returns up to limit applications for tenantID ordered by
	// (name, id) ascending, strictly after the keyset (afterName, afterID).
	// Pass "", "" for the first page. Backs ListAdminApplications keyset
	// pagination (wi-159).
	ListPage(ctx context.Context, tenantID, afterName, afterID string, limit int) ([]*appdomain.Application, error)
	ListPageBefore(ctx context.Context, tenantID, beforeName, beforeID string, limit int) ([]*appdomain.Application, error)
	Count(ctx context.Context, tenantID string) (int64, error)
	// FindByID は application_id に一致する Application を返す。存在しなければ (nil, nil)。
	FindByID(ctx context.Context, tenantID, applicationID string) (*appdomain.Application, error)
	// FindByProtocol は protocol table の application_id relation を索引で逆引きする。
	// catalog 外 protocol record なら nil を返す。
	FindByProtocol(ctx context.Context, tenantID string, protocolType appdomain.ApplicationProtocolType, key string) (*appdomain.Application, error)
	// Save は Application を upsert する。
	Save(ctx context.Context, app *appdomain.Application) error
	// Create は Application 行と protocol row.application_id を原子的に作成する。
	Create(ctx context.Context, app *appdomain.Application) error
	// Delete は application_id に一致する Application を削除する (冪等)。
	Delete(ctx context.Context, tenantID, applicationID string) error
	// DeleteWithRelated は関連状態と使用量を一つの確定境界で削除する。
	DeleteWithRelated(ctx context.Context, app *appdomain.Application, participants DeletionParticipants) error
	// RemoveCategory はテナント内の全 Application の category_ids から指定カテゴリを除く
	// (カテゴリ削除時のクリーンアップ, wi-70)。
	RemoveCategory(ctx context.Context, tenantID, categoryID string) error
}

// DeletionParticipants は、削除に伴って同じ確定境界へ参加する操作である。
type DeletionParticipants struct {
	Assignments    AssignmentRepository
	Policy         SignInPolicyDeleter
	Icons          ApplicationIconStore
	Quota          tenantports.QuotaRepository
	DeleteProtocol func(context.Context, *appdomain.Application) (int, error)
}

// ApplicationIconStore は Application icon blob の保存境界。
// Application aggregate は icon_object_key だけを持ち、binary 本体はこの store が所有する。
type ApplicationIconStore interface {
	Save(ctx context.Context, icon *appdomain.ApplicationIcon) error
	Find(ctx context.Context, tenantID, applicationID, objectKey string) (*appdomain.ApplicationIcon, error)
	DeleteByApplication(ctx context.Context, tenantID, applicationID string) error
}

// ApplicationCategoryRepository は ApplicationCategory の永続境界 (wi-70)。
type ApplicationCategoryRepository interface {
	// ListAll はテナント内のカテゴリを position 昇順 (同値は name 昇順) で返す。
	ListAll(ctx context.Context, tenantID string) ([]*appdomain.ApplicationCategory, error)
	// FindByID は category_id に一致するカテゴリを返す。存在しなければ (nil, nil)。
	FindByID(ctx context.Context, tenantID, categoryID string) (*appdomain.ApplicationCategory, error)
	// Save はカテゴリを upsert する。
	Save(ctx context.Context, category *appdomain.ApplicationCategory) error
	// Delete は category_id に一致するカテゴリを削除する (冪等)。
	Delete(ctx context.Context, tenantID, categoryID string) error
}

// SubjectRef は割当の対象 (user / group) を表す参照 (wi-69)。
type SubjectRef struct {
	Type appdomain.AssignmentSubjectType
	ID   string
}

// ApplicationOrderingRepository は利用者ごとのポータル手動並び順の永続境界 (wi-70)。
type ApplicationOrderingRepository interface {
	// Get は利用者の手動並び順 (application_id の順序列) を返す。未保存なら (nil, nil)。
	Get(ctx context.Context, tenantID, userID string) (*appdomain.ApplicationOrdering, error)
	// Save は利用者の手動並び順を upsert する。
	Save(ctx context.Context, ordering *appdomain.ApplicationOrdering) error
}

// AssignmentRepository は Application 割当の永続境界 (wi-69)。
type AssignmentRepository interface {
	// ListByApplication は Application の割当を subject 昇順で返す。
	ListByApplication(ctx context.Context, tenantID, applicationID string) ([]*appdomain.ApplicationAssignment, error)
	// ListPageByApplication returns up to limit assignments for (tenantID, applicationID)
	// ordered by (subject_type, subject_id) ascending, strictly after the
	// keyset (afterSubjectType, afterSubjectID). Pass "", "" for the first
	// page. Backs ListApplicationAssignments keyset pagination (wi-159).
	ListPageByApplication(ctx context.Context, tenantID, applicationID, afterSubjectType, afterSubjectID string, limit int) ([]*appdomain.ApplicationAssignment, error)
	ListPageBeforeByApplication(ctx context.Context, tenantID, applicationID, beforeSubjectType, beforeSubjectID string, limit int) ([]*appdomain.ApplicationAssignment, error)
	// ListBySubjects は指定 subject 群に一致する割当を返す (ポータル一覧・割当ゲート用)。
	ListBySubjects(ctx context.Context, tenantID string, subjects []SubjectRef) ([]*appdomain.ApplicationAssignment, error)
	// ListAll はテナント内のすべての Application 割当を返す。
	ListAll(ctx context.Context, tenantID string) ([]*appdomain.ApplicationAssignment, error)
	// Save は割当を upsert する。
	Save(ctx context.Context, assignment *appdomain.ApplicationAssignment) error
	// Delete は 1 件の割当を削除する (冪等)。
	Delete(ctx context.Context, tenantID, applicationID string, subjectType appdomain.AssignmentSubjectType, subjectID string) error
	// DeleteByApplication は Application の全割当を削除する (Application 削除時のクリーンアップ)。
	DeleteByApplication(ctx context.Context, tenantID, applicationID string) error
}
