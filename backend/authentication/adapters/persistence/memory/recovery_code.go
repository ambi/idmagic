package memory

import (
	"context"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/authentication/domain"
)

// =====================================================================
// RecoveryCodeRepository (Authentication) — wi-26 / ADR-087
// =====================================================================

type RecoveryCodeRepository struct {
	mu    sync.Mutex
	codes map[string][]*domain.RecoveryCode // key: sub
}

func NewRecoveryCodeRepository() *RecoveryCodeRepository {
	return &RecoveryCodeRepository{codes: map[string][]*domain.RecoveryCode{}}
}

func (r *RecoveryCodeRepository) ListBySub(
	_ context.Context,
	sub string,
) ([]*domain.RecoveryCode, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []*domain.RecoveryCode{}
	for _, c := range r.codes[sub] {
		out = append(out, cloneRecoveryCode(c))
	}
	return out, nil
}

func (r *RecoveryCodeRepository) ReplaceAll(
	_ context.Context,
	sub string,
	codes []*domain.RecoveryCode,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	stored := make([]*domain.RecoveryCode, 0, len(codes))
	for _, c := range codes {
		stored = append(stored, cloneRecoveryCode(c))
	}
	r.codes[sub] = stored
	return nil
}

func (r *RecoveryCodeRepository) MarkConsumed(
	_ context.Context,
	sub string,
	codeHash string,
	now time.Time,
) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.codes[sub] {
		if c.CodeHash == codeHash && c.ConsumedAt == nil {
			consumed := now
			c.ConsumedAt = &consumed
			return true, nil
		}
	}
	return false, nil
}

func (r *RecoveryCodeRepository) DeleteAllForSub(_ context.Context, sub string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.codes, sub)
	return nil
}

func cloneRecoveryCode(c *domain.RecoveryCode) *domain.RecoveryCode {
	if c == nil {
		return nil
	}
	out := *c
	if c.ConsumedAt != nil {
		consumed := *c.ConsumedAt
		out.ConsumedAt = &consumed
	}
	return &out
}
