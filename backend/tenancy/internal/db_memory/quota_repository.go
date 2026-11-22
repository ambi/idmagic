package db_memory

import (
	"context"
	"fmt"
	"math"
	"sync"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

type QuotaRepository struct {
	mu     sync.Mutex
	quotas map[string]*tenancydomain.TenantQuota
	usages map[string]*tenancydomain.TenantUsage
}

var _ tenantports.QuotaRepository = (*QuotaRepository)(nil)

func NewQuotaRepository() *QuotaRepository {
	return &QuotaRepository{
		quotas: make(map[string]*tenancydomain.TenantQuota),
		usages: make(map[string]*tenancydomain.TenantUsage),
	}
}

func (r *QuotaRepository) CheckAndIncrement(ctx context.Context, tenantID, resource string, delta int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if tenancydomain.IsSoftQuota(resource) && delta < 0 {
		return fmt.Errorf("soft quota: negative increment")
	}

	usage := r.usages[tenantID]
	if usage == nil {
		usage = &tenancydomain.TenantUsage{}
		r.usages[tenantID] = usage
	}

	quota := r.quotas[tenantID]
	if quota == nil {
		quota = &tenancydomain.TenantQuota{}
	}

	var currentUsage int

	switch resource {
	case tenancydomain.ResourceUsers:
		currentUsage = usage.Users
	case tenancydomain.ResourceGroups:
		currentUsage = usage.Groups
	case tenancydomain.ResourceAgents:
		currentUsage = usage.Agents
	case tenancydomain.ResourceApplications:
		currentUsage = usage.Applications
	case tenancydomain.ResourceOAuth2Clients:
		currentUsage = usage.OAuth2Clients
	case tenancydomain.ResourceActiveSessions:
		currentUsage = usage.ActiveSessions
	case tenancydomain.ResourceConsents:
		currentUsage = usage.Consents
	case tenancydomain.ResourceActiveJobs:
		currentUsage = usage.ActiveJobs
	case tenancydomain.ResourceSsfStreams:
		currentUsage = usage.SsfStreams
	case tenancydomain.ResourceAuditEventsRetained:
		currentUsage = usage.AuditEventsRetained
	case tenancydomain.ResourceExportArtifactsBytes:
		currentUsage = usage.ExportArtifactsBytes
	default:
		return fmt.Errorf("unknown resource: %s", resource)
	}
	currentQuota := quota.EffectiveLimit(resource)

	if tenancydomain.IsSoftQuota(resource) && delta > math.MaxInt-currentUsage {
		return fmt.Errorf("soft quota: usage overflow")
	}
	if !tenancydomain.IsSoftQuota(resource) && currentUsage+delta > currentQuota {
		return &tenancydomain.QuotaExceededError{TenantID: tenantID, Resource: resource}
	}

	switch resource {
	case tenancydomain.ResourceUsers:
		usage.Users += delta
	case tenancydomain.ResourceGroups:
		usage.Groups += delta
	case tenancydomain.ResourceAgents:
		usage.Agents += delta
	case tenancydomain.ResourceApplications:
		usage.Applications += delta
	case tenancydomain.ResourceOAuth2Clients:
		usage.OAuth2Clients += delta
	case tenancydomain.ResourceActiveSessions:
		usage.ActiveSessions += delta
	case tenancydomain.ResourceConsents:
		usage.Consents += delta
	case tenancydomain.ResourceActiveJobs:
		usage.ActiveJobs += delta
	case tenancydomain.ResourceSsfStreams:
		usage.SsfStreams += delta
	case tenancydomain.ResourceAuditEventsRetained:
		usage.AuditEventsRetained += delta
	case tenancydomain.ResourceExportArtifactsBytes:
		usage.ExportArtifactsBytes += delta
	}

	return nil
}

func (r *QuotaRepository) Decrement(ctx context.Context, tenantID, resource string, delta int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if tenancydomain.IsSoftQuota(resource) && delta < 0 {
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
	case tenancydomain.ResourceUsers:
		decrementValue(&usage.Users)
	case tenancydomain.ResourceGroups:
		decrementValue(&usage.Groups)
	case tenancydomain.ResourceAgents:
		decrementValue(&usage.Agents)
	case tenancydomain.ResourceApplications:
		decrementValue(&usage.Applications)
	case tenancydomain.ResourceOAuth2Clients:
		decrementValue(&usage.OAuth2Clients)
	case tenancydomain.ResourceActiveSessions:
		decrementValue(&usage.ActiveSessions)
	case tenancydomain.ResourceConsents:
		decrementValue(&usage.Consents)
	case tenancydomain.ResourceActiveJobs:
		decrementValue(&usage.ActiveJobs)
	case tenancydomain.ResourceSsfStreams:
		decrementValue(&usage.SsfStreams)
	case tenancydomain.ResourceAuditEventsRetained:
		decrementValue(&usage.AuditEventsRetained)
	case tenancydomain.ResourceExportArtifactsBytes:
		decrementValue(&usage.ExportArtifactsBytes)
	default:
		return fmt.Errorf("unknown resource: %s", resource)
	}
	return nil
}

func (r *QuotaRepository) SetQuota(ctx context.Context, tenantID string, quota *tenancydomain.TenantQuota) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.quotas[tenantID] = quota
	return nil
}

func (r *QuotaRepository) SetSoftUsage(_ context.Context, tenantID string, measured tenancydomain.SoftQuotaUsage) error {
	if err := measured.Validate(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	usage := r.usages[tenantID]
	if usage == nil {
		usage = &tenancydomain.TenantUsage{}
		r.usages[tenantID] = usage
	}
	usage.AuditEventsRetained = measured.AuditEventsRetained
	usage.ExportArtifactsBytes = measured.ExportArtifactsBytes
	return nil
}

func (r *QuotaRepository) RecountUsage(ctx context.Context, tenantID, resource string, count func(context.Context) (int, error)) (int, error) {
	field := recountFields[resource]
	if field == nil {
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
		usage = &tenancydomain.TenantUsage{}
		r.usages[tenantID] = usage
	}
	previous := *field(usage)
	*field(usage) = counted
	return previous, nil
}

// recountFields は、実データの件数で使用量を置き換えるリソースの列である。
var recountFields = map[string]func(*tenancydomain.TenantUsage) *int{
	tenancydomain.ResourceUsers:          func(u *tenancydomain.TenantUsage) *int { return &u.Users },
	tenancydomain.ResourceActiveSessions: func(u *tenancydomain.TenantUsage) *int { return &u.ActiveSessions },
}

func (r *QuotaRepository) GetQuota(ctx context.Context, tenantID string) (*tenancydomain.TenantQuota, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	quota, ok := r.quotas[tenantID]
	if !ok {
		return &tenancydomain.TenantQuota{}, nil
	}
	return quota, nil
}

func (r *QuotaRepository) GetUsage(ctx context.Context, tenantID string) (*tenancydomain.TenantUsage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	usage, ok := r.usages[tenantID]
	if !ok {
		return &tenancydomain.TenantUsage{}, nil
	}
	snapshot := *usage
	return &snapshot, nil
}
