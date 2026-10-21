package ports

import (
	"context"

	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
)

// AgentChange は、Update が最新の Agent に当てる変更である。false を返すと保存しない。
// 行を固定している間に呼ばれるので、外部の作用を起こさず、ほかの集約も読まない。
type AgentChange func(agent *agentdomain.Agent) (bool, error)

// AgentRepository は tenant-scoped な Agent 集約とその OAuth2Client 束縛を永続化する。
// すべての操作はテナント境界に閉じ、cross-tenant 参照は use case 側で
// reject する。Agent は自身の資格情報を持たず、AgentCredentialBinding で既存
// OAuth2Client に束縛してトークンを得る。
type AgentRepository interface {
	ListAll(ctx context.Context, tenantID string) ([]*agentdomain.Agent, error)
	// ListPage returns up to limit agents for tenantID ordered by (name, id)
	// ascending, strictly after the keyset (afterName, afterID). Pass "", ""
	// for the first page. Backs ListAgents keyset pagination (wi-159).
	ListPage(ctx context.Context, tenantID, afterName, afterID string, limit int) ([]*agentdomain.Agent, error)
	ListPageBefore(ctx context.Context, tenantID, beforeName, beforeID string, limit int) ([]*agentdomain.Agent, error)
	Count(ctx context.Context, tenantID string) (int64, error)
	FindByID(ctx context.Context, tenantID, id string) (*agentdomain.Agent, error)
	Save(ctx context.Context, agent *agentdomain.Agent) error
	// Update は Agent の行を固定して読み直し、change をその最新の値に当てて保存し、保存した
	// Agent を返す。Agent がなければ change を呼ばずに nil を返す。先に読んだ Agent を Save
	// すると、その間の別の書き込み (所有者の停止に伴う無効化や停止など) を書き戻すので、
	// Agent を変える操作はこれを使う。
	Update(ctx context.Context, tenantID, id string, change AgentChange) (*agentdomain.Agent, error)
	Delete(ctx context.Context, tenantID, id string) error

	ListBindings(ctx context.Context, tenantID, agentID string) ([]*agentdomain.AgentCredentialBinding, error)
	// AddBinding は束縛を追加し、新規追加なら true を返す。既に束縛済みなら false を
	// 返し no-op とする (冪等)。
	AddBinding(ctx context.Context, binding *agentdomain.AgentCredentialBinding) (bool, error)
	// RemoveBinding は束縛を削除し、削除されたなら true を返す。非束縛なら false を
	// 返し no-op とする (冪等)。
	RemoveBinding(ctx context.Context, tenantID, agentID, clientID string) (bool, error)
	// FindByClientID は指定 client に束縛された Agent を返す。束縛がなければ nil を
	// 返す。トークン発行経路の status gate が呼ぶ。
	FindByClientID(ctx context.Context, tenantID, clientID string) (*agentdomain.Agent, error)
}
