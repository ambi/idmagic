package wsfederation

import (
	"context"

	"github.com/ambi/idmagic/backend/wsfederation/internal/db_postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// LinkApplicationInTx は未結合の RP を、呼び出し側が作る Application へ同じ Tx で結ぶ。
func LinkApplicationInTx(ctx context.Context, tx pgx.Tx, tenantID, applicationID, wtrealm string) error {
	var id pgtype.UUID
	if err := id.Scan(applicationID); err != nil {
		return err
	}
	return db_postgres.New(tx).LinkApplication(ctx, db_postgres.LinkApplicationParams{
		ApplicationID: id, TenantID: tenantID, Wtrealm: wtrealm,
	})
}
