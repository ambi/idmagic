package db_memory

import (
	"context"
	"fmt"
	"math"
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
	if domain.IsSoftQuota(resource) && delta < 0 {
		return fmt.Errorf("soft quota: negative increment")
	}

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
	case domain.ResourceSsfStreams:
		currentUsage = usage.SsfStreams
	case domain.ResourceAuditEventsRetained:
		currentUsage = usage.AuditEventsRetained
	case domain.ResourceExportArtifactsBytes:
		currentUsage = usage.ExportArtifactsBytes
	default:
		return fmt.Errorf("unknown resource: %s", resource)
	}
	currentQuota := quota.EffectiveLimit(resource)

	if domain.IsSoftQuota(resource) && delta > math.MaxInt-currentUsage {
		return fmt.Errorf("soft quota: usage overflow")
	}
	if !domain.IsSoftQuota(resource) && currentUsage+delta > currentQuota {
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
	case domain.ResourceSsfStreams:
		usage.SsfStreams += delta
	case domain.ResourceAuditEventsRetained:
		usage.AuditEventsRetained += delta
	case domain.ResourceExportArtifactsBytes:
		usage.ExportArtifactsBytes += delta
	}

	return nil
}

func (r *QuotaRepository) Decrement(ctx context.Context, tenantID, resource string, delta int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if domain.IsSoftQuota(resource) && delta < 0 {
		return fmt.Errorf("soft quota: negative decrement")
	}

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
	case domain.ResourceSsfStreams:
		decrementValue(&usage.SsfStreams)
	case domain.ResourceAuditEventsRetained:
		decrementValue(&usage.AuditEventsRetained)
	case domain.ResourceExportArtifactsBytes:
		decrementValue(&usage.ExportArtifactsBytes)
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

func (r *QuotaRepository) SetSoftUsage(_ context.Context, tenantID string, measured domain.SoftQuotaUsage) error {
	if err := measured.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	usage := r.usages[tenantID]
	if usage == nil {
		usage = &domain.TenantUsage{}
		r.usages[tenantID] = usage
	}
	usage.AuditEventsRetained = measured.AuditEventsRetained
	usage.ExportArtifactsBytes = measured.ExportArtifactsBytes
	return nil
}

func (r *QuotaRepository) RecountUsage(ctx context.Context, tenantID, resource string, count func(context.Context) (int, error)) (int, error) {
	if resource != domain.ResourceUsers {
		return 0, fmt.Errorf("usage recount is not supported for %s", resource)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	counted, err := count(ctx)
	if err != nil {
		return 0, err
	}
	if counted < 0 {
		return 0, fmt.Errorf("usage recount: negative count %d", counted)
	}
	usage := r.usages[tenantID]
	if usage == nil {
		usage = &domain.TenantUsage{}
		r.usages[tenantID] = usage
	}
	previous := usage.Users
	usage.Users = counted
	return previous, nil
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
	snapshot := *usage
	return &snapshot, nil
}
