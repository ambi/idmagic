package db_postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/jackc/pgx/v5"
)

// McpToolReportRepository は登録ごとの最新の報告を PostgreSQL に置く。
// 登録の削除は外部キーで報告も消す。
type McpToolReportRepository struct{ Pool sharedpg.DB }

func (r *McpToolReportRepository) Find(ctx context.Context, tenantID, resourceServerID string) (*domain.McpToolReport, error) {
	row, err := New(r.Pool).GetMcpToolReport(ctx, GetMcpToolReportParams{TenantID: tenantID, ResourceServerID: resourceServerID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	report := &domain.McpToolReport{
		TenantID: row.TenantID, ResourceServerID: row.ResourceServerID, Complete: row.Complete, ReportedAt: row.ReportedAt,
	}
	if err := json.Unmarshal(row.Tools, &report.Tools); err != nil {
		return nil, err
	}
	return report, nil
}

func (r *McpToolReportRepository) Save(ctx context.Context, report *domain.McpToolReport) error {
	tools, err := json.Marshal(report.Tools)
	if err != nil {
		return err
	}
	return New(r.Pool).UpsertMcpToolReport(ctx, UpsertMcpToolReportParams{
		TenantID: report.TenantID, ResourceServerID: report.ResourceServerID, Tools: tools,
		Complete: report.Complete, ReportedAt: report.ReportedAt,
	})
}

func (r *McpToolReportRepository) Delete(ctx context.Context, tenantID, resourceServerID string) error {
	return New(r.Pool).DeleteMcpToolReport(ctx, DeleteMcpToolReportParams{TenantID: tenantID, ResourceServerID: resourceServerID})
}
