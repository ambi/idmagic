package memory

import (
	"context"
	"sync"
	"time"

	"github.com/ambi/idmagic/backend/authentication/mfa/domain"
)

type MfaEnrollmentBypassRepository struct {
	mu    sync.Mutex
	items map[string]*domain.MfaEnrollmentBypass
}

func NewMfaEnrollmentBypassRepository() *MfaEnrollmentBypassRepository {
	return &MfaEnrollmentBypassRepository{items: map[string]*domain.MfaEnrollmentBypass{}}
}

func (r *MfaEnrollmentBypassRepository) Save(_ context.Context, bypass *domain.MfaEnrollmentBypass) error {
	if err := bypass.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, item := range r.items {
		if item.TenantID == bypass.TenantID && item.UserID == bypass.UserID && item.ConsumedAt == nil && item.RevokedAt == nil && item.ExpiredAt == nil {
			now := bypass.IssuedAt
			item.RevokedAt = &now
		}
	}
	r.items[bypass.ID] = cloneMfaEnrollmentBypass(bypass)
	return nil
}

func (r *MfaEnrollmentBypassRepository) ExpireOpen(_ context.Context, tenantID, userID string, now time.Time) (*domain.MfaEnrollmentBypass, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item := r.findOpen(tenantID, userID)
	if item == nil || now.Before(item.ExpiresAt) {
		return nil, nil
	}
	item.ExpiredAt = &now
	return cloneMfaEnrollmentBypass(item), nil
}

func (r *MfaEnrollmentBypassRepository) FindActive(_ context.Context, tenantID, userID string, now time.Time) (*domain.MfaEnrollmentBypass, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return cloneMfaEnrollmentBypass(r.findActive(tenantID, userID, now)), nil
}

func (r *MfaEnrollmentBypassRepository) ConsumeActive(_ context.Context, tenantID, userID string, now time.Time) (*domain.MfaEnrollmentBypass, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item := r.findActive(tenantID, userID, now)
	if item == nil {
		return nil, nil
	}
	item.ConsumedAt = &now
	return cloneMfaEnrollmentBypass(item), nil
}

func (r *MfaEnrollmentBypassRepository) RevokeActive(_ context.Context, tenantID, userID string, now time.Time) (*domain.MfaEnrollmentBypass, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item := r.findOpen(tenantID, userID)
	if item == nil {
		return nil, nil
	}
	item.RevokedAt = &now
	return cloneMfaEnrollmentBypass(item), nil
}

func (r *MfaEnrollmentBypassRepository) findActive(tenantID, userID string, now time.Time) *domain.MfaEnrollmentBypass {
	for _, item := range r.items {
		if item.TenantID == tenantID && item.UserID == userID && item.Available(now) {
			return item
		}
	}
	return nil
}

func (r *MfaEnrollmentBypassRepository) findOpen(tenantID, userID string) *domain.MfaEnrollmentBypass {
	for _, item := range r.items {
		if item.TenantID == tenantID && item.UserID == userID && item.ConsumedAt == nil && item.RevokedAt == nil && item.ExpiredAt == nil {
			return item
		}
	}
	return nil
}

func cloneMfaEnrollmentBypass(in *domain.MfaEnrollmentBypass) *domain.MfaEnrollmentBypass {
	if in == nil {
		return nil
	}
	out := *in
	if in.ConsumedAt != nil {
		value := *in.ConsumedAt
		out.ConsumedAt = &value
	}
	if in.RevokedAt != nil {
		value := *in.RevokedAt
		out.RevokedAt = &value
	}
	if in.ExpiredAt != nil {
		value := *in.ExpiredAt
		out.ExpiredAt = &value
	}
	return &out
}
