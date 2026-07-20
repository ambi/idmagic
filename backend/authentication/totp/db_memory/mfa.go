package db_memory

import (
	"context"
	"strings"
	"sync"

	"github.com/ambi/idmagic/backend/authentication/totp/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// =====================================================================
// MfaFactorRepository (Authentication)
// =====================================================================

type MfaFactorRepository struct {
	mu      sync.RWMutex
	factors map[string]*domain.MfaFactor
}

func NewMfaFactorRepository() *MfaFactorRepository {
	return &MfaFactorRepository{factors: map[string]*domain.MfaFactor{}}
}

func (r *MfaFactorRepository) ListBySub(_ context.Context, sub string) ([]*domain.MfaFactor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := []*domain.MfaFactor{}
	for _, factor := range r.factors {
		if factor.UserID == sub {
			out = append(out, cloneMfaFactor(factor))
		}
	}
	return out, nil
}

func (r *MfaFactorRepository) Find(
	_ context.Context,
	sub string,
	factorType spec.MfaFactorType,
) (*domain.MfaFactor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return cloneMfaFactor(r.factors[mfaFactorKey(sub, factorType)]), nil
}

func (r *MfaFactorRepository) Save(_ context.Context, factor *domain.MfaFactor) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factors[mfaFactorKey(factor.UserID, factor.Type)] = cloneMfaFactor(factor)
	return nil
}

func (r *MfaFactorRepository) Delete(_ context.Context, sub string, factorType spec.MfaFactorType) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.factors, mfaFactorKey(sub, factorType))
	return nil
}

func (r *MfaFactorRepository) DeleteAllForSub(_ context.Context, sub string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	prefix := sub + "|"
	for key := range r.factors {
		if strings.HasPrefix(key, prefix) {
			delete(r.factors, key)
		}
	}
	return nil
}

func mfaFactorKey(sub string, factorType spec.MfaFactorType) string {
	return sub + "|" + string(factorType)
}

func cloneMfaFactor(factor *domain.MfaFactor) *domain.MfaFactor {
	if factor == nil {
		return nil
	}
	out := *factor
	if factor.Secret != nil {
		secret := *factor.Secret
		out.Secret = &secret
	}
	if factor.Label != nil {
		label := *factor.Label
		out.Label = &label
	}
	if factor.LastUsedAt != nil {
		lastUsedAt := *factor.LastUsedAt
		out.LastUsedAt = &lastUsedAt
	}
	return &out
}
