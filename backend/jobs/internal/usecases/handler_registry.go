package usecases

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/jobs/ports"
)

// ErrHandlerNotRegistered is returned by HandlerRegistry.Lookup (and
// surfaced as the Job's failure) when no Handler is registered for a claimed
// Job's Kind. This should only happen if a worker is running with a stale
// binary that predates a JobKind added to docs/modules/jobs/.
var ErrHandlerNotRegistered = errors.New("jobs: no handler registered for job kind")

var _ ports.HandlerRegistrar = (*HandlerRegistry)(nil)

// HandlerRegistry maps a JobKind to the Handler that executes it. A JobKind
// must first be added to docs/modules/jobs/ (SCL-first) before a
// consumer WI registers its Handler here.
type HandlerRegistry struct {
	mu       sync.RWMutex
	handlers map[domain.JobKind]handlerRegistration
}

type handlerRegistration struct {
	handler ports.Handler
	after   []ports.SettlementHandler
}

func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{handlers: map[domain.JobKind]handlerRegistration{}}
}

// Register は起動時に、種類ごとのハンドラーと確定後の通知をまとめて登録する。
// 再登録は以前の通知も置き換える。不明な種類と nil の通知は構築時の誤りとして拒否する。
func (r *HandlerRegistry) Register(kind domain.JobKind, h ports.Handler, after ...ports.SettlementHandler) {
	if !kind.Valid() {
		panic(fmt.Sprintf("jobs: cannot register handler for unknown JobKind %q", kind))
	}
	for _, notify := range after {
		if notify == nil {
			panic("jobs: 確定後の通知処理が未設定")
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[kind] = handlerRegistration{handler: h, after: slices.Clone(after)}
}

// Lookup returns the Handler registered for kind, or
// (nil, ErrHandlerNotRegistered).
func (r *HandlerRegistry) Lookup(kind domain.JobKind) (ports.Handler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[kind]
	if !ok {
		return nil, ErrHandlerNotRegistered
	}
	return h.handler, nil
}

func (r *HandlerRegistry) notifySettlement(ctx context.Context, job *domain.Job) {
	r.mu.RLock()
	after := r.handlers[job.Kind].after
	r.mu.RUnlock()
	for _, notify := range after {
		notify(ctx, job)
	}
}
