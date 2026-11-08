package db_postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	"github.com/jackc/pgx/v5"
)

// McpToolCatalogRepository は登録ごとの最新のカタログを PostgreSQL に置く。
// 登録の削除は外部キーでカタログも消す。
type McpToolCatalogRepository struct{ Pool sharedpg.DB }

func (r *McpToolCatalogRepository) Find(ctx context.Context, tenantID, resourceServerID string) (*domain.McpToolCatalog, error) {
	row, err := New(r.Pool).GetMcpToolCatalog(ctx, GetMcpToolCatalogParams{TenantID: tenantID, ResourceServerID: resourceServerID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	catalog := &domain.McpToolCatalog{
		TenantID: row.TenantID, ResourceServerID: row.ResourceServerID, Complete: row.Complete, ReceivedAt: row.ReceivedAt,
	}
	if err := json.Unmarshal(row.Tools, &catalog.Tools); err != nil {
		return nil, err
	}
	if row.PreviousTools != nil {
		catalog.Previous = &domain.CatalogToolList{Complete: row.PreviousComplete.Bool}
		if err := json.Unmarshal(row.PreviousTools, &catalog.Previous.Tools); err != nil {
			return nil, err
		}
		catalog.ChangedAt = row.ChangedAt.Time
	}
	return catalog, nil
}

// Save はカタログを置き換える。変更前の一覧と変更の時刻は、保存済みの行の指紋との比較で一つの文が決める。
func (r *McpToolCatalogRepository) Save(ctx context.Context, catalog *domain.McpToolCatalog) error {
	tools, err := json.Marshal(catalog.Tools)
	if err != nil {
		return err
	}
	return New(r.Pool).UpsertMcpToolCatalog(ctx, UpsertMcpToolCatalogParams{
		TenantID: catalog.TenantID, ResourceServerID: catalog.ResourceServerID, Tools: tools,
		Complete: catalog.Complete, ReceivedAt: catalog.ReceivedAt, Fingerprint: catalog.Fingerprint(),
	})
}

func (r *McpToolCatalogRepository) Delete(ctx context.Context, tenantID, resourceServerID string) error {
	return New(r.Pool).DeleteMcpToolCatalog(ctx, DeleteMcpToolCatalogParams{TenantID: tenantID, ResourceServerID: resourceServerID})
}
