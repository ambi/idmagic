package usecases

import (
	"context"
	"errors"

	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/sourcing/scim/domain"
	"github.com/ambi/idmagic/backend/sourcing/scim/ports"
)

var ErrNotFound = errors.New("SCIM resource not found")

// ErrDuplicate signals a uniqueness conflict (userName/displayName already
// used within the tenant). Wrapped with errors.Is-compatible context by
// callers; handlers map it to HTTP 409 with scimType "uniqueness".
var ErrDuplicate = errors.New("SCIM resource already exists")

// scimDeleted は SCIM の表現から消えている User を判定する
// (RFC7644-DELETE-SEMANTICS)。削除は soft delete なので User レコードは残り、
// lifecycle が PendingDeletion か Deleted になる。
//
// Disabled は消えていない。無効化は active: false として表現する状態であり、
// 削除と区別できなくなってはいけない。判定をここ 1 箇所に集めるのは、SCIM の
// 入口が User を読む経路を 7 つ持っており、どれか 1 つで判定を忘れるとその経路
// だけ削除済みの User が見え続けるからである。
func scimDeleted(user *userdomain.User) bool {
	return user.IsSoftDeleted() || user.IsDeleted()
}

type Usecases struct {
	ScimRepo  ports.ScimRepository
	UserRepo  userports.UserRepository
	GroupRepo groupports.GroupRepository
	Emit      func(spec.DomainEvent)
}

func NewUsecases(
	scimRepo ports.ScimRepository,
	userRepo userports.UserRepository,
	groupRepo groupports.GroupRepository,
	emit func(spec.DomainEvent),
) *Usecases {
	return &Usecases{
		ScimRepo:  scimRepo,
		UserRepo:  userRepo,
		GroupRepo: groupRepo,
		Emit:      emit,
	}
}

// findLiveUserByScimID は SCIM id から、SCIM の表現に残っている User を引く。
// 参照が無い、User が無い、削除済みのいずれも ErrNotFound になる。SCIM
// クライアントから見て、削除した id と一度も存在しなかった id は区別できては
// いけない (RFC7644-DELETE-SEMANTICS)。
func (u *Usecases) findLiveUserByScimID(ctx context.Context, tenantID, scimID string) (*userdomain.User, error) {
	ref, err := u.ScimRepo.FindUserRefByScimID(ctx, tenantID, scimID)
	if err != nil {
		return nil, err
	}
	if ref == nil {
		return nil, ErrNotFound
	}

	user, err := u.UserRepo.FindBySub(ctx, ref.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil || scimDeleted(user) {
		return nil, ErrNotFound
	}
	return user, nil
}

// liveUserByID は内部 User id から、SCIM の表現に残っている User を引く。応答へ
// 射影する前に参照先が消えていないかを見るために使う。消えている User は
// (nil, nil) を返し、呼び出し側はその参照を出さない。
func (u *Usecases) liveUserByID(ctx context.Context, userID string) (*userdomain.User, error) {
	user, err := u.UserRepo.FindBySub(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || scimDeleted(user) {
		return nil, nil //nolint:nilnil // 契約: SCIM から消えている User は (nil, nil)。
	}
	return user, nil
}

// resolveLiveUserID は書き込みが受け取った SCIM id を内部 User id へ解決する。
// 解決できない id と削除済みの User を区別せず、どちらも invalidValue で拒否する
// (RFC7644-DELETE-SEMANTICS)。kind は拒否の文面に入る参照の名前である。
func (u *Usecases) resolveLiveUserID(ctx context.Context, tenantID, kind, scimID string) (string, error) {
	user, err := u.findLiveUserByScimID(ctx, tenantID, scimID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", domain.NewMutationError("invalidValue", "%s %q does not resolve to a User in this tenant", kind, scimID)
		}
		return "", err
	}
	return user.ID, nil
}

// ListQuery is the normalized input to ListUsers/ListGroups (SCL
// interfaces.ListScimUsers / ListScimGroups). StartIndex/Count are nil when
// the caller omitted the corresponding query parameter; HasCount
// distinguishes an omitted count from an explicit 0.
type ListQuery struct {
	Filter     string
	StartIndex *int
	Count      *int
	HasCount   bool
}

// ListResult is a filtered, paginated SCIM collection page.
type ListResult struct {
	Total        int
	Items        []map[string]any
	StartIndex   int
	ItemsPerPage int
}
