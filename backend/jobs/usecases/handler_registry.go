package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/ambi/idmagic/backend/jobs/domain"
)

// ErrHandlerNotRegistered is returned by HandlerRegistry.Lookup (and
// surfaced as the Job's failure) when no Handler is registered for a claimed
// Job's Kind. This should only happen if a worker is running with a stale
// binary that predates a JobKind added to spec/contexts/jobs.yaml.
var ErrHandlerNotRegistered = errors.New("jobs: no handler registered for job kind")

// Handler executes a claimed Job's business logic. It must be idempotent
// (JobHandlerIdempotency): at-least-once delivery means the same Job may be
// handed to a Handler more than once. Implementations call into the owning
// bounded context's usecases; Jobs itself holds no business logic.
type Handler func(ctx context.Context, job *domain.Job) (result json.RawMessage, err error)

// HandlerRegistry maps a JobKind to the Handler that executes it. A JobKind
// must first be added to spec/contexts/jobs.yaml (SCL-first) before a
// consumer WI registers its Handler here.
type HandlerRegistry struct {
	mu       sync.RWMutex
	handlers map[domain.JobKind]Handler
}

func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{handlers: map[domain.JobKind]Handler{}}
}

// Register adds h as the Handler for kind, overwriting any previous
// registration. It panics if kind is not a valid spec/contexts/jobs.yaml
// JobKind, since that is a programmer error caught at worker startup.
func (r *HandlerRegistry) Register(kind domain.JobKind, h Handler) {
	if !kind.Valid() {
		panic(fmt.Sprintf("jobs: cannot register handler for unknown JobKind %q", kind))
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[kind] = h
}

// Lookup returns the Handler registered for kind, or
// (nil, ErrHandlerNotRegistered).
func (r *HandlerRegistry) Lookup(kind domain.JobKind) (Handler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[kind]
	if !ok {
		return nil, ErrHandlerNotRegistered
	}
	return h, nil
}
