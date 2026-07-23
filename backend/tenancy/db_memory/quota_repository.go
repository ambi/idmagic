package db_memory

import (
	"context"
	"fmt"
	"sync"

	"github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

type QuotaRepository struct {
	mu     sync.Mutex
	quotas map[string]*domain.TenantQuota
	usages map[string]*domain.TenantUsage
}

var _ tenantports.QuotaRepository = (*QuotaRepository)(nil)

func NewQuotaRepository() *QuotaRepository {
	return &QuotaRepository{
		quotas: make(map[string]*domain.TenantQuota),
		usages: make(map[string]*domain.TenantUsage),
	}
}

func (r *QuotaRepository) CheckAndIncrement(ctx context.Context, tenantID, resource string, delta int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	usage := r.usages[tenantID]
	if usage == nil {
		usage = &domain.TenantUsage{}
		r.usages[tenantID] = usage
	}

	quota := r.quotas[tenantID]
	if quota == nil {
		quota = &domain.TenantQuota{}
	}

	var currentUsage int

	switch resource {
	case domain.ResourceUsers:
		currentUsage = usage.Users
	case domain.ResourceGroups:
		currentUsage = usage.Groups
	case domain.ResourceAgents:
		currentUsage = usage.Agents
	case domain.ResourceApplications:
		currentUsage = usage.Applications
	case domain.ResourceOAuth2Clients:
		currentUsage = usage.OAuth2Clients
	case domain.ResourceActiveSessions:
		currentUsage = usage.ActiveSessions
	case domain.ResourceConsents:
		currentUsage = usage.Consents
	case domain.ResourceActiveJobs:
		currentUsage = usage.ActiveJobs
	default:
		return fmt.Errorf("unknown resource: %s", resource)
	}
	currentQuota := quota.EffectiveLimit(resource)

	if currentUsage+delta > currentQuota {
		return &domain.QuotaExceededError{TenantID: tenantID, Resource: resource}
	}

	switch resource {
	case domain.ResourceUsers:
		usage.Users += delta
	case domain.ResourceGroups:
		usage.Groups += delta
	case domain.ResourceAgents:
		usage.Agents += delta
	case domain.ResourceApplications:
		usage.Applications += delta
	case domain.ResourceOAuth2Clients:
		usage.OAuth2Clients += delta
	case domain.ResourceActiveSessions:
		usage.ActiveSessions += delta
	case domain.ResourceConsents:
		usage.Consents += delta
	case domain.ResourceActiveJobs:
		usage.ActiveJobs += delta
	}

	return nil
}

func (r *QuotaRepository) Decrement(ctx context.Context, tenantID, resource string, delta int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	usage := r.usages[tenantID]
	if usage == nil {
		return nil
	}

	decrementValue := func(val *int) {
		*val -= delta
		if *val < 0 {
			*val = 0
		}
	}

	switch resource {
	case domain.ResourceUsers:
		decrementValue(&usage.Users)
	case domain.ResourceGroups:
		decrementValue(&usage.Groups)
	case domain.ResourceAgents:
		decrementValue(&usage.Agents)
	case domain.ResourceApplications:
		decrementValue(&usage.Applications)
	case domain.ResourceOAuth2Clients:
		decrementValue(&usage.OAuth2Clients)
	case domain.ResourceActiveSessions:
		decrementValue(&usage.ActiveSessions)
	case domain.ResourceConsents:
		decrementValue(&usage.Consents)
	case domain.ResourceActiveJobs:
		decrementValue(&usage.ActiveJobs)
	default:
		return fmt.Errorf("unknown resource: %s", resource)
	}
	return nil
}

func (r *QuotaRepository) SetQuota(ctx context.Context, tenantID string, quota *domain.TenantQuota) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.quotas[tenantID] = quota
	return nil
}

func (r *QuotaRepository) GetQuota(ctx context.Context, tenantID string) (*domain.TenantQuota, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	quota, ok := r.quotas[tenantID]
	if !ok {
		return &domain.TenantQuota{}, nil
	}
	return quota, nil
}

func (r *QuotaRepository) GetUsage(ctx context.Context, tenantID string) (*domain.TenantUsage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	usage, ok := r.usages[tenantID]
	if !ok {
		return &domain.TenantUsage{}, nil
	}
	return usage, nil
}
