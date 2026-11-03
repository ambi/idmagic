package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
)

// ErrInvalidToolReport は報告の形か上限が REQ-OAUTH2-086 を満たさないことを表す。
var ErrInvalidToolReport = errors.New("invalid tool report")

// ToolReportDeps は報告の保存に要る保存先である。
type ToolReportDeps struct {
	Servers ports.McpResourceServerRepository
	Reports ports.McpToolReportRepository
}

// ToolReportRequest は認証したクライアントと、フォームで受けた報告である。
type ToolReportRequest struct {
	TenantID string
	ClientID string
	Tools    string
	Complete string
}

// ReportResourceTools は、クライアントに束縛した Active の登録について最新の報告を置き換える。
// 登録そのものと操作割当は読むだけで変えない。
func ReportResourceTools(ctx context.Context, deps ToolReportDeps, in ToolReportRequest, now time.Time) error {
	server, err := deps.Servers.FindByClientID(ctx, in.TenantID, in.ClientID)
	if err != nil {
		return err
	}
	if server == nil || !server.IsActive() {
		return ErrResourceServerNotRegistered
	}
	if in.Complete != "true" && in.Complete != "false" {
		return ErrInvalidToolReport
	}
	tools, err := domain.ParseReportedTools(in.Tools)
	if err != nil {
		return ErrInvalidToolReport
	}
	return deps.Reports.Save(ctx, &domain.McpToolReport{
		TenantID: server.TenantID, ResourceServerID: server.ID, Tools: tools,
		Complete: in.Complete == "true", ReportedAt: now,
	})
}
