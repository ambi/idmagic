package ports

import (
	"context"

	idmdomain "github.com/ambi/idmagic/backend/identitymanagement/domain"
)

// UserRepository は IdentityManagement が所有する User aggregate の永続化境界。
type UserRepository interface {
	// FindBySub は ADR-036 の tombstone (`deleted_at != null`) を除外する。
	// 既に削除された user を含めて引きたい場合は FindBySubIncludingDeleted を使う。
	FindBySub(ctx context.Context, sub string) (*idmdomain.User, error)
	// FindBySubIncludingDeleted は tombstone を含めて user を引く。
	// DeleteUser use case の冪等判定や監査経路から呼ばれる。
	FindBySubIncludingDeleted(ctx context.Context, sub string) (*idmdomain.User, error)
	FindByUsername(ctx context.Context, tenantID, username string) (*idmdomain.User, error)
	FindByEmail(ctx context.Context, tenantID, email string) (*idmdomain.User, error)
	FindAll(ctx context.Context, tenantID string) ([]*idmdomain.User, error)
	Save(ctx context.Context, user *idmdomain.User) error
}
