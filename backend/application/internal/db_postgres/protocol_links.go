package db_postgres

import (
	"context"

	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/jackc/pgx/v5"
)

// ProtocolLinkers は所有者の更新を Application の作成と削除の Tx へ結ぶ。
type ProtocolLinkers struct {
	OIDC  func(ctx context.Context, tx pgx.Tx, tenantID, applicationID, clientID string) error
	SAML  func(ctx context.Context, tx pgx.Tx, tenantID, applicationID, entityID string) error
	WsFed func(ctx context.Context, tx pgx.Tx, tenantID, applicationID, wtrealm string) error
	// DeleteOIDC はクライアントを固定し、保護を確認して専用の参照状態を削除する。
	DeleteOIDC func(ctx context.Context, tx pgx.Tx, tenantID, clientID string) (int, error)
	QuotaInTx  func(pgx.Tx) tenantports.QuotaRepository
}

func NewApplicationRepository(pool sharedpg.DB, linkers ProtocolLinkers) *ApplicationRepository {
	if linkers.OIDC == nil || linkers.SAML == nil || linkers.WsFed == nil {
		panic("application: protocol linkers are required")
	}
	return &ApplicationRepository{Pool: pool, ProtocolLinkers: linkers}
}
