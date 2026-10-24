// Package ports は Directory Connector の永続化の契約を定める。
package ports

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
)

// ErrStaleConnector は、読んだ後にほかの更新が Connector を変えたことを表す。
var ErrStaleConnector = errors.New("directory connector: stale revision")

// ConnectorRepository は Directory Connector を保存する。テナントの外の Connector は存在しない
// ものとして扱い、Find と FindByEnrollmentTokenHash は見つからなければ nil を返す。
type ConnectorRepository interface {
	Create(ctx context.Context, connector domain.Connector) error
	// Update は、保存済みの Revision が connector.Revision の一つ前のときだけ保存し、
	// そうでなければ ErrStaleConnector を返す。
	Update(ctx context.Context, connector domain.Connector) error
	Find(ctx context.Context, tenantID, id string) (*domain.Connector, error)
	FindByEnrollmentTokenHash(ctx context.Context, tenantID, tokenHash string) (*domain.Connector, error)
	// List はテナントの Connector を作成時刻の昇順 (同時刻は ID の昇順) で返す。
	List(ctx context.Context, tenantID string) ([]domain.Connector, error)
	// TouchLastSeen は Active の Connector の最終接続時刻だけを書き、Revision を進めない。
	TouchLastSeen(ctx context.Context, tenantID, id string, at time.Time) error
}
