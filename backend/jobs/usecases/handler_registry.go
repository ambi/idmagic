package usecases

import (
	"errors"
	"fmt"
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
	handlers map[domain.JobKind]ports.Handler
}

func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{handlers: map[domain.JobKind]ports.Handler{}}
}

// Register adds h as the Handler for kind, overwriting any previous
// registration. It panics if kind is not a valid docs/modules/jobs/
// JobKind, since that is a programmer error caught at worker startup.
func (r *HandlerRegistry) Register(kind domain.JobKind, h ports.Handler) {
	if !kind.Valid() {
		panic(fmt.Sprintf("jobs: cannot register handler for unknown JobKind %q", kind))
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[kind] = h
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
	return h, nil
}
