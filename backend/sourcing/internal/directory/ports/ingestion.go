package ports

import (
	"context"
	"errors"
	"time"

	directoryports "github.com/ambi/idmagic/backend/sourcing/directory/ports"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
)

var ErrRunBusy = errors.New("directory ingestion: connector operation is busy")

// CorrelationRepository は Connector ごとの User の対応を保存する。対応の作成は User の作成と
// 同じトランザクションで行うので、組み立て地点の IdentityWrites が担う。
type CorrelationRepository interface {
	List(ctx context.Context, tenantID, connectorID string) ([]directoryports.UserCorrelation, error)
	// Find は見つからなければ nil を返す。
	Find(ctx context.Context, tenantID, connectorID, externalID string) (*directoryports.UserCorrelation, error)
	// MarkSeen は対応が runID の実行に現れたことを記録する。
	MarkSeen(ctx context.Context, tenantID, connectorID, externalID, runID string) error
	// ListNotSeenIn は、Connector の対応のうち、runID の実行に現れなかったものを返す。
	ListNotSeenIn(ctx context.Context, tenantID, connectorID, runID string) ([]directoryports.UserCorrelation, error)
	// CorrelatedUserIDs は userIDs のうち、いずれかの Connector の対応を持つ User の ID を返す。
	CorrelatedUserIDs(ctx context.Context, tenantID string, userIDs []string) (map[string]bool, error)
}

// IngestionRunRepository は取り込みの実行と、その記録したバッチを保存する。
type IngestionRunRepository interface {
	// WithConnectorLock はプロセスをまたいで同じ Connector の変更を排他し、競合は ErrRunBusy で返す。
	WithConnectorLock(ctx context.Context, tenantID, connectorID string, apply func() error) error
	// SaveProgress は実行の状態、保留と無効化件数を保存する。バッチの集計を書き戻さない。
	SaveProgress(ctx context.Context, run domain.IngestionRun) error
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
	// Complete は running の実行だけを completed にし、変えたかを返す。
	Complete(ctx context.Context, tenantID, runID string, completedAt time.Time, disabled int) (bool, error)
}
