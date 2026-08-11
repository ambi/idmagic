package ports

import (
	"context"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

// UserRepository は IdManagement が所有する User aggregate の永続化境界。
type UserRepository interface {
	// FindBySub は tombstone (`deleted_at != null`) を除外する。
	// 既に削除された user を含めて引きたい場合は FindBySubIncludingDeleted を使う。
	FindBySub(ctx context.Context, sub string) (*userdomain.User, error)
	// FindBySubIncludingDeleted は tombstone を含めて user を引く。
	// DeleteUser use case の冪等判定や監査経路から呼ばれる。
	FindBySubIncludingDeleted(ctx context.Context, sub string) (*userdomain.User, error)
	FindByUsername(ctx context.Context, tenantID, username string) (*userdomain.User, error)
	FindByEmail(ctx context.Context, tenantID, email string) (*userdomain.User, error)
	FindAll(ctx context.Context, tenantID string) ([]*userdomain.User, error)
	// ListPage returns up to limit non-deleted users for tenantID ordered by
	// (preferred_username, id) ascending, strictly after the keyset
	// (afterUsername, afterID). Pass "", "" for the first page. Backs
	// ListAdminUsers keyset pagination (wi-159).
	ListPage(ctx context.Context, tenantID, afterUsername, afterID string, limit int) ([]*userdomain.User, error)
	ListPageBefore(ctx context.Context, tenantID, beforeUsername, beforeID string, limit int) ([]*userdomain.User, error)
	ListPageFiltered(ctx context.Context, tenantID, query string, status *idmdomain.UserStatus, afterUsername, afterID string, limit int) ([]*userdomain.User, error)
	ListPageBeforeFiltered(ctx context.Context, tenantID, query string, status *idmdomain.UserStatus, beforeUsername, beforeID string, limit int) ([]*userdomain.User, error)
	Count(ctx context.Context, tenantID string) (int64, error)
	CountFiltered(ctx context.Context, tenantID, query string, status *idmdomain.UserStatus) (int64, error)
	Save(ctx context.Context, user *userdomain.User) error
}
