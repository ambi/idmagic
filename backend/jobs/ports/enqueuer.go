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

// SettlementHandler は成功、再試行、最終失敗を保存した後に、その保存結果を受け取る。
// 保存失敗、取り消し、リース喪失では呼ばない。保存と通知は同じトランザクションではなく、
// 保存後のプロセス停止による通知の欠落を許容する用途に限る。
type SettlementHandler func(ctx context.Context, job *domain.Job)

// HandlerRegistrar はジョブの種類ごとに処理と確定後の通知を一緒に登録する契約である。
// after は登録順に同期で呼ぶ。通知が不要な種類では省略する。
type HandlerRegistrar interface {
	Register(kind domain.JobKind, h Handler, after ...SettlementHandler)
}

// Canceler はほかのモジュールと管理の経路がジョブを取り消す契約である。
// 取り消しの遷移、active_jobs の減算、JobCanceled の発行は Jobs が行う。
// 終端に達したジョブは ErrJobAlreadyTerminal で拒否し、使用量もイベントも変えない。
type Canceler interface {
	Cancel(ctx context.Context, jobID string, now time.Time) (*domain.Job, error)
}
