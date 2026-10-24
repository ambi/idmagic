package ports

import (
	"context"
	"time"

	directoryports "github.com/ambi/idmagic/backend/sourcing/directory/ports"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
)

// CorrelationRepository は Connector ごとの User の対応を保存する。対応の作成は User の作成と
// 同じトランザクションで行うので、組み立て地点の IdentityWrites が担う。
type CorrelationRepository interface {
	// Find は見つからなければ nil を返す。
	Find(ctx context.Context, tenantID, connectorID, externalID string) (*directoryports.UserCorrelation, error)
	// MarkSeen は対応が runID の実行に現れたことを記録する。
	MarkSeen(ctx context.Context, tenantID, connectorID, externalID, runID string) error
	// ListNotSeenIn は、Connector の対応のうち、runID の実行に現れなかったものを返す。
	ListNotSeenIn(ctx context.Context, tenantID, connectorID, runID string) ([]directoryports.UserCorrelation, error)
	// CorrelatedUserIDs は userIDs のうち、いずれかの Connector の対応を持つ User の ID を返す。
	CorrelatedUserIDs(ctx context.Context, tenantID string, userIDs []string) (map[string]bool, error)
}

// ComputerRepository は Connector と外部の不変 ID の組ごとにコンピューターを保存する。テナントの
// 有効なコンピューターの間で、SID と主体名をそれぞれ一意に保つ。
type ComputerRepository interface {
	// Find は見つからなければ nil を返す。
	Find(ctx context.Context, tenantID, connectorID, externalID string) (*domain.Computer, error)
	// FindEnabledHolders は、objectSID か principalKey を持つ有効なコンピューターを返す。
	FindEnabledHolders(ctx context.Context, tenantID, objectSID, principalKey string) ([]domain.Computer, error)
	// Save はコンピューターを作るか置き換える。一意性に反すれば domain.ErrComputerConflict を返す。
	Save(ctx context.Context, computer domain.Computer) error
	// MarkSeen はコンピューターが runID の実行に現れたことを記録する。
	MarkSeen(ctx context.Context, tenantID, connectorID, externalID, runID string) error
	// DisableNotSeenIn は、Connector の有効なコンピューターのうち、runID の実行に現れなかったものを
	// 無効にし、その数を返す。
	DisableNotSeenIn(ctx context.Context, tenantID, connectorID, runID string, at time.Time) (int, error)
	// FindEnabledBySID と FindEnabledByPrincipal は、見つからなければ nil を返す。
	FindEnabledBySID(ctx context.Context, tenantID, objectSID string) (*domain.Computer, error)
	FindEnabledByPrincipal(ctx context.Context, tenantID, samAccountName, realm string) (*domain.Computer, error)
}

// IngestionRunRepository は取り込みの実行と、その記録したバッチを保存する。
type IngestionRunRepository interface {
	// Start は run を保存し、同じ Connector の running の実行を superseded にする。
	Start(ctx context.Context, run domain.IngestionRun) error
	// Find は、Connector の実行でなければ nil を返す。
	Find(ctx context.Context, tenantID, connectorID, runID string) (*domain.IngestionRun, error)
	// List は Connector の実行を開始時刻の新しい順に limit 件まで返す。
	List(ctx context.Context, tenantID, connectorID string, limit int) ([]domain.IngestionRun, error)
	FindBatch(ctx context.Context, tenantID, runID, batchID string) (*domain.IngestionBatch, error)
	// RecordBatch はバッチを記録して実行の件数へ足し、記録したバッチを返す。同じ ID のバッチが
	// 先に記録されていれば、何も変えずにそれを返す。
	RecordBatch(ctx context.Context, batch domain.IngestionBatch) (domain.IngestionBatch, error)
	// ListBatches は実行のバッチを記録した順に返す。
	ListBatches(ctx context.Context, tenantID, runID string) ([]domain.IngestionBatch, error)
	// Complete は running の実行だけを completed にし、無効にした User とコンピューターの数を記録して、変えたかを返す。
	Complete(ctx context.Context, tenantID, runID string, completedAt time.Time, disabled, disabledComputers int) (bool, error)
}
