package oauth2

import (
	"context"

	"github.com/ambi/idmagic/backend/oauth2/internal/db_postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// LinkApplicationInTx は未結合のクライアントを、呼び出し側が作る Application へ同じ Tx で結ぶ。
func LinkApplicationInTx(ctx context.Context, tx pgx.Tx, tenantID, applicationID, clientID string) error {
	var id pgtype.UUID
	if err := id.Scan(applicationID); err != nil {
		return err
	}
	return db_postgres.New(tx).LinkApplication(ctx, db_postgres.LinkApplicationParams{
		ApplicationID: id, TenantID: tenantID, ClientID: clientID,
	})
}
