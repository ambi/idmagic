package ports

import (
	"context"
	"time"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
)

// GroupSourceOwnershipGuard は所有権をまとめて解決する。ガードが無い、または
// 失敗した場合を、計画器は fail-closed に解釈する。IdManagement が port を所有し、
// Sourcing の adapter が内向きに実装する。
type GroupSourceOwnershipGuard interface {
	SourceManagedGroupIDs(ctx context.Context, tenantID string, groupIDs []string) (map[string]bool, error)
}

// GroupImportRowMutation は受理した 1 行の完全な書き込み集合。adapter は Aggregate、
// dynamic rule、membership の cascade、クォータの増減、監査イベントを 1 つの
// トランザクションで確定する。Delete が真なら After は nil であり、Before が
// 消える対象である。
type GroupImportRowMutation struct {
	Before             *groupdomain.Group
	After              *groupdomain.Group
	Rule               *groupdomain.DynamicGroupRule
	Delete             bool
	RemovedMemberships []string
	Changed            []string
	ActorUserID        string
	AuditEventType     string
	ConsumesGroupQuota bool
	ReleasesGroupQuota bool
	// ReconcileGroupID は動的規則が変わった行で、再評価を同じトランザクション内に
	// 投入するための対象。空なら投入しない。
	ReconcileGroupID string
	ReconcileVersion int64
	Now              time.Time
}

type GroupImportRowCommitter interface {
	CommitGroupImportRow(ctx context.Context, mutation GroupImportRowMutation) error
}
