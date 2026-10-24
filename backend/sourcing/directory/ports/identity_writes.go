// Package ports は、Directory Connector の取り込みが組み立て地点に求める書き込みの契約を定める。
package ports

import (
	"context"
	"errors"
	"time"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
)

// UserCorrelation は、Connector と外部の不変 ID の組から内部の User への対応である。
// LastSeenRunID は、この対応が最後に現れた取り込みの実行の ID である。
type UserCorrelation struct {
	TenantID      string
	ConnectorID   string
	ExternalID    string
	UserID        string
	LastSeenRunID string
}

// CorrelationWriter は、User の作成と同じトランザクションで対応を保存する。
type CorrelationWriter interface {
	SaveUserCorrelation(ctx context.Context, correlation UserCorrelation) error
}

// IdentityWrites は投影済みの属性を所有者の規則で保存する。
// CreateUser は作った User の ID を correlation に入れ、対応を一緒に確定する。
type IdentityWrites interface {
	CreateUser(ctx context.Context, correlation UserCorrelation, user *userdomain.User) (*userdomain.User, error)
	UpdateUser(ctx context.Context, user *userdomain.User) (*userdomain.User, error)
}

// UserLifecycle は、取り込みが User を止める、再開する、削除を予約するときに通す IdManagement の操作である。
type UserLifecycle interface {
	SetUserDisabled(ctx context.Context, tenantID, userID string, disabled bool, now time.Time) (changed bool, err error)
	ScheduleUserDeletion(ctx context.Context, tenantID, userID string, now time.Time) error
}

// IdentityWrites が User の作成と変更を拒否する理由。使用量の超過は IsQuotaExceeded を持つエラーで返す。
var (
	ErrUsernameConflict = errors.New("directory ingestion: username conflict")
	ErrEmailConflict    = errors.New("directory ingestion: email conflict")
	ErrInvalidAttribute = errors.New("directory ingestion: invalid attribute")
)
