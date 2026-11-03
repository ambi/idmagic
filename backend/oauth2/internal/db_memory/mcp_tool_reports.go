package db_memory

import (
	"context"
	"slices"
	"sync"

	"github.com/ambi/idmagic/backend/shared/kernel"
	sharedmem "github.com/ambi/idmagic/backend/shared/storage/db_memory"

	"github.com/ambi/idmagic/backend/oauth2/domain"
)

// McpToolReportRepository は登録ごとの最新の報告をメモリに置く。
type McpToolReportRepository struct {
	mu      sync.RWMutex
	reports map[string]*domain.McpToolReport // key: reportKey(tenant_id, resource_server_id)
}

func NewMcpToolReportRepository() *McpToolReportRepository {
	return &McpToolReportRepository{reports: map[string]*domain.McpToolReport{}}
}

func cloneMcpToolReport(report *domain.McpToolReport) *domain.McpToolReport {
	cloned := *report
	cloned.Tools = slices.Clone(report.Tools)
	for index := range cloned.Tools {
		cloned.Tools[index].Arguments = slices.Clone(cloned.Tools[index].Arguments)
	}
	return &cloned
}

func (r *McpToolReportRepository) Find(_ context.Context, tenantID, resourceServerID string) (*domain.McpToolReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	report, ok := r.reports[reportKey(tenantID, resourceServerID)]
	if !ok {
		return nil, nil
	}
	return cloneMcpToolReport(report), nil
}

func (r *McpToolReportRepository) Save(_ context.Context, report *domain.McpToolReport) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reports[reportKey(report.TenantID, report.ResourceServerID)] = cloneMcpToolReport(report)
	return nil
}

func (r *McpToolReportRepository) Delete(_ context.Context, tenantID, resourceServerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.reports, reportKey(tenantID, resourceServerID))
	return nil
}

func reportKey(tenantID, resourceServerID string) string {
	return sharedmem.ScopedKey(tenantID, resourceServerID, kernel.DefaultTenantID)
}
