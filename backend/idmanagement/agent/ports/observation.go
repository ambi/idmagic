package ports

import (
	"context"
	"errors"
	"time"

	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
)

var (
	// ErrObservationLimitExceeded は保存するとテナントの上限を超える。
	ErrObservationLimitExceeded = errors.New("agent observation limit exceeded")
	// ErrObservationAgentMissing は確認先の Agent が決定の確定の時点で存在しない。
	ErrObservationAgentMissing = errors.New("agent observation target agent is missing")
)

// ObservationImportCounts は一つのバッチの適用の結果である。
type ObservationImportCounts struct {
	Created, Updated, Unchanged int
}

// ObservationFilter は一覧の条件である。Decision が空なら絞り込まない。
type ObservationFilter struct {
	Decision agentdomain.ObservationDecisionKind
}

// AgentObservationRepository は外部の観測を永続化する。すべての操作はテナント境界に閉じる。
// 確認は Agent の削除と同じ確定で解除する。
type AgentObservationRepository interface {
	// Import はバッチを順に適用する。既存の行は ObservedAt が新しいときだけ属性を更新し、決定を
	// 変えない。全件を確定するか一件も確定せず、上限を超えるなら ErrObservationLimitExceeded を返す。
	Import(ctx context.Context, tenantID string, batch []*agentdomain.AgentObservation, maxObservations int) (ObservationImportCounts, error)
	FindByID(ctx context.Context, tenantID, id string) (*agentdomain.AgentObservation, error)
	// ListPage は (Subject, ID) の昇順で、キーセットより後を返す。
	ListPage(ctx context.Context, tenantID string, filter ObservationFilter, afterSubject, afterID string, limit int) ([]*agentdomain.AgentObservation, error)
	// ListPageBefore は (Subject, ID) の昇順の結果を、キーセットより前の範囲から返す。
	ListPageBefore(ctx context.Context, tenantID string, filter ObservationFilter, beforeSubject, beforeID string, limit int) ([]*agentdomain.AgentObservation, error)
	Count(ctx context.Context, tenantID string, filter ObservationFilter) (int64, error)
	// ListConfirmed はテナントの確認した観測をすべて返す。棚卸しが読む。
	ListConfirmed(ctx context.Context, tenantID string) ([]*agentdomain.AgentObservation, error)
	// Confirm は未決定の行だけを確認し、更新したかを返す。Agent がなければ
	// ErrObservationAgentMissing を返す。
	Confirm(ctx context.Context, tenantID, id, agentID, actorUserID string, at time.Time) (bool, error)
	// Exclude は未決定の行だけを除外し、更新したかを返す。
	Exclude(ctx context.Context, tenantID, id, actorUserID string, at time.Time) (bool, error)
	// Reopen は行を未決定へ戻し、戻す前の観測を返す。行がなければ nil を返す。
	Reopen(ctx context.Context, tenantID, id string) (*agentdomain.AgentObservation, error)
	// PurgeExpired は ObservedAt が cutoff より前の、確認していない観測を消し、件数を返す。
	PurgeExpired(ctx context.Context, cutoff time.Time) (int64, error)
}

// WorkloadSubjectMatcher は、外部の issuer と subject に一致する有効な関連付けの Agent を返す。
// 関連付けの保存型と照合の規則は WorkloadIdentity が持ち、IdManagement は結果だけを受け取る。
type WorkloadSubjectMatcher interface {
	AgentsForSubject(ctx context.Context, tenantID, issuer, subject string) ([]string, error)
}
