package ports

import (
	"context"
	"time"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
)

// GroupMembershipImportRowMutation は受理した 1 行の完全な書き込み集合。adapter は
// メンバーシップの追加または解除と監査イベントを 1 つのトランザクションで確定する。
// Release が真なら Member は nil であり、UserID が外す対象である。
type GroupMembershipImportRowMutation struct {
	TenantID string
	GroupID  string
	UserID   string
	// Release は解除の行で真になる。偽の行は Member を追加する。
	Release        bool
	Member         *groupdomain.GroupMember
	ActorUserID    string
	AuditEventType string
	Now            time.Time
}

type GroupMembershipImportRowCommitter interface {
	CommitGroupMembershipImportRow(ctx context.Context, mutation GroupMembershipImportRowMutation) error
}
