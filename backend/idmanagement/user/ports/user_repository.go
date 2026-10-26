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
	// ListPurgeCandidates は、テナントの PendingDeletion の User と、完全削除を終えていない
	// (PendingPurge を持つ) Tombstone を返す。猶予期間の判定は呼び出し側が行う。
	ListPurgeCandidates(ctx context.Context, tenantID string) ([]*userdomain.User, error)
	Count(ctx context.Context, tenantID string) (int64, error)
	// CountHoldingQuota は、テナントの User の使用量を占める User (User.HoldsQuota) を数える。
	CountHoldingQuota(ctx context.Context, tenantID string) (int64, error)
	CountFiltered(ctx context.Context, tenantID, query string, status *idmdomain.UserStatus) (int64, error)
	Save(ctx context.Context, user *userdomain.User) error
	// Update は sub の User の行を固定して読み直し、change をその最新の値に当てて保存し、
	// 保存した User を返す。先に読んだ User を Save すると、その間の別の書き込み (無効化など) を
	// 書き戻すので、一部の値だけを記録する書き手はこれを使う。
	Update(ctx context.Context, sub string, change UserChange) (*userdomain.User, error)
}

// UserChange は、Update が最新の User に当てる変更である。false を返すと保存しない。
// 行を固定している間に呼ばれるので、外部の作用を起こさない。
type UserChange func(user *userdomain.User) (bool, error)

// UserWriter は、呼び出し側のトランザクションに参加して User を保存する。
// ほかのモジュールが自分の書き込みと User の変更を一緒に確定するときに使い、
// 確定と巻き戻しは呼び出し側のトランザクションが決める。
type UserWriter interface {
	Save(ctx context.Context, user *userdomain.User) error
	// Update は呼び出し側のトランザクションの中で、UserRepository.Update と同じことを行う。
	Update(ctx context.Context, sub string, change UserChange) (*userdomain.User, error)
}
