package ports

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ambi/idmagic/backend/jobs/domain"
)

// Enqueuer はほかのモジュールがジョブを投入する契約である。
// レーンの決定、既定値の補完、重複排除、active_jobs の確認、JobEnqueued の発行は Jobs が行う。
type Enqueuer interface {
	Enqueue(ctx context.Context, input EnqueueInput, now time.Time) (*domain.Job, error)
}

// Handler は取得したジョブの業務処理を実行する。少なくとも 1 回配送されるので、
// 同じジョブを複数回受け取っても結果が変わらないように実装する（JobHandlerIdempotency）。
type Handler func(ctx context.Context, job *domain.Job) (result json.RawMessage, err error)

// HandlerRegistrar はジョブの種類ごとに処理を登録する契約である。
type HandlerRegistrar interface {
	Register(kind domain.JobKind, h Handler)
}
