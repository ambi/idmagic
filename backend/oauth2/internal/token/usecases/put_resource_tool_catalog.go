package usecases

import (
	"context"
	"errors"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
)

// ErrInvalidToolCatalog はカタログの形か上限が REQ-OAUTH2-086 を満たさないことを表す。
var ErrInvalidToolCatalog = errors.New("invalid tool catalog")

// ToolCatalogDeps はカタログの保存に要る保存先である。
type ToolCatalogDeps struct {
	Servers  ports.McpResourceServerRepository
	Catalogs ports.McpToolCatalogRepository
}

// ToolCatalogRequest は認証したクライアントと、フォームで受けたカタログである。
type ToolCatalogRequest struct {
	TenantID string
	ClientID string
	Tools    string
	Complete string
}

// PutResourceToolCatalog は、クライアントに束縛した Active の登録について最新のカタログを置き換える。
// 登録そのものと操作割当は読むだけで変えない。
func PutResourceToolCatalog(ctx context.Context, deps ToolCatalogDeps, in ToolCatalogRequest, now time.Time) error {
	server, err := deps.Servers.FindByClientID(ctx, in.TenantID, in.ClientID)
	if err != nil {
		return err
	}
	if server == nil || !server.IsActive() {
		return ErrResourceServerNotRegistered
	}
	if in.Complete != "true" && in.Complete != "false" {
		return ErrInvalidToolCatalog
	}
	tools, err := domain.ParseCatalogTools(in.Tools)
	if err != nil {
		return ErrInvalidToolCatalog
	}
	return deps.Catalogs.Save(ctx, &domain.McpToolCatalog{
		TenantID: server.TenantID, ResourceServerID: server.ID, Tools: tools,
		Complete: in.Complete == "true", ReceivedAt: now,
	})
}
