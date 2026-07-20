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
	var currentQuota int

	switch resource {
	case "users":
		currentUsage = usage.Users
		currentQuota = 10000
		if quota.Users != nil {
			currentQuota = *quota.Users
		}
	case "groups":
		currentUsage = usage.Groups
		currentQuota = 1000
		if quota.Groups != nil {
			currentQuota = *quota.Groups
		}
	case "agents":
		currentUsage = usage.Agents
		currentQuota = 100
		if quota.Agents != nil {
			currentQuota = *quota.Agents
		}
	case "applications":
		currentUsage = usage.Applications
		currentQuota = 50
		if quota.Applications != nil {
			currentQuota = *quota.Applications
		}
	case "oauth2_clients":
		currentUsage = usage.OAuth2Clients
		currentQuota = 100
		if quota.OAuth2Clients != nil {
			currentQuota = *quota.OAuth2Clients
		}
	case "active_sessions":
		currentUsage = usage.ActiveSessions
		currentQuota = 50000
		if quota.ActiveSessions != nil {
			currentQuota = *quota.ActiveSessions
		}
	case "consents":
		currentUsage = usage.Consents
		currentQuota = 10000
		if quota.Consents != nil {
			currentQuota = *quota.Consents
		}
	case "active_jobs":
		currentUsage = usage.ActiveJobs
		currentQuota = 10
		if quota.ActiveJobs != nil {
			currentQuota = *quota.ActiveJobs
		}
	default:
		return fmt.Errorf("unknown resource: %s", resource)
	}

	if currentUsage+delta > currentQuota {
		return &domain.QuotaExceededError{TenantID: tenantID, Resource: resource}
	}

	switch resource {
	case "users":
		usage.Users += delta
	case "groups":
		usage.Groups += delta
	case "agents":
		usage.Agents += delta
	case "applications":
		usage.Applications += delta
	case "oauth2_clients":
		usage.OAuth2Clients += delta
	case "active_sessions":
		usage.ActiveSessions += delta
	case "consents":
		usage.Consents += delta
	case "active_jobs":
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
	case "users":
		decrementValue(&usage.Users)
	case "groups":
		decrementValue(&usage.Groups)
	case "agents":
		decrementValue(&usage.Agents)
	case "applications":
		decrementValue(&usage.Applications)
	case "oauth2_clients":
		decrementValue(&usage.OAuth2Clients)
	case "active_sessions":
		decrementValue(&usage.ActiveSessions)
	case "consents":
		decrementValue(&usage.Consents)
	case "active_jobs":
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
