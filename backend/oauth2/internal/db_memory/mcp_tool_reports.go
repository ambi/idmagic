package db_memory

import (
	"context"
	"slices"
	"sync"
	"time"

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
	cloned.Tools = cloneReportedTools(report.Tools)
	if report.Previous != nil {
		cloned.Previous = &domain.ReportedToolList{Tools: cloneReportedTools(report.Previous.Tools), Complete: report.Previous.Complete}
	}
	return &cloned
}

func cloneReportedTools(tools []domain.ReportedTool) []domain.ReportedTool {
	cloned := slices.Clone(tools)
	for index := range cloned {
		cloned[index].Arguments = slices.Clone(cloned[index].Arguments)
	}
	return cloned
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

// Save は報告を置き換え、内容が保存済みの報告と異なるときだけ保存済みの一覧を変更前として残す。
// 比較と置き換えを同じロックの中で行い、並行する報告が変更前の一覧を失わないようにする。
func (r *McpToolReportRepository) Save(_ context.Context, report *domain.McpToolReport) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := reportKey(report.TenantID, report.ResourceServerID)
	saved := cloneMcpToolReport(report)
	saved.Previous, saved.ChangedAt = nil, time.Time{}
	if stored, ok := r.reports[key]; ok {
		if stored.Fingerprint() == report.Fingerprint() {
			saved.Previous, saved.ChangedAt = stored.Previous, stored.ChangedAt
		} else {
			saved.Previous = &domain.ReportedToolList{Tools: stored.Tools, Complete: stored.Complete}
			saved.ChangedAt = report.ReportedAt
		}
	}
	r.reports[key] = cloneMcpToolReport(saved)
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
