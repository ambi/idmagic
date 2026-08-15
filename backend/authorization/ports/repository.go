// Package ports は Authorization Context の永続化と外部解決の抽象を宣言する。
package ports

import (
	"context"

	"github.com/ambi/idmagic/backend/authorization/domain"
)

// RelationTupleFilter は一覧の絞り込み。空のフィールドは絞り込まない。
type RelationTupleFilter struct {
	ResourceType string
	ResourceID   string
	Relation     string
	SubjectType  string
	SubjectID    string
}

// TupleWrite は 1 トランザクションで適用する差分。DeleteObjects に挙げた
// オブジェクトは、リソース側・主体側の双方のタプルを取り除く。
type TupleWrite struct {
	Writes        []domain.RelationTuple
	Deletes       []domain.RelationTuple
	DeleteObjects []domain.ObjectRef
}

// TupleWriteResult は適用件数と、適用後のテナントの書き込み版。
type TupleWriteResult struct {
	WrittenCount int
	DeletedCount int
	Version      int64
}

// RelationTupleRepository はテナント境界を持つ関係タプルのストア。
// 実装は memory と PostgreSQL の 2 つで、同じ契約テストを共有する。
type RelationTupleRepository interface {
	domain.TupleReader

	// List は絞り込んだタプルを limit 件まで返す。
	List(ctx context.Context, tenantID string, filter RelationTupleFilter, limit int) ([]domain.RelationTuple, error)
	// ListResourceIDs は、そのテナント・そのリソース型に現れる識別子を limit 件まで返す。
	// ListAccessibleResources の走査対象になる。
	ListResourceIDs(ctx context.Context, tenantID, resourceType string, limit int) ([]string, error)
	// Write は差分を 1 トランザクションで適用し、書き込み版を進める。
	Write(ctx context.Context, tenantID string, write TupleWrite) (TupleWriteResult, error)
	// Version はテナントの現在の書き込み版を返す。書き込みのないテナントは 0。
	Version(ctx context.Context, tenantID string) (int64, error)
}

// AuthorizationModelRepository は追記のみの認可モデルの版を保持する。
type AuthorizationModelRepository interface {
	// Publish は次の版を採番して保存し、テナントの書き込み版を進める。
	// 呼び出し側は ID / TenantID / ResourceTypes / CreatedAt を埋め、Version は
	// この実装が決める。
	Publish(ctx context.Context, model *domain.AuthorizationModel) (*domain.AuthorizationModel, int64, error)
	// Latest は最新版を返す。1 版も無ければ nil を返す。
	Latest(ctx context.Context, tenantID string) (*domain.AuthorizationModel, error)
	// FindByVersion は指定した版を返す。無ければ nil を返す。
	FindByVersion(ctx context.Context, tenantID string, version int) (*domain.AuthorizationModel, error)
}

// PrincipalStatusResolver は代行チェーン上のプリンシパルが有効かどうかを解決する。
// Agent の状態は IdManagement が正であり、Authorization はポート越しに問い合わせる
// だけで判断の実体を持たない。解決できない場合は有効とみなさない。
type PrincipalStatusResolver interface {
	IsPrincipalActive(ctx context.Context, tenantID, principalType, principalID string) (bool, error)
}
