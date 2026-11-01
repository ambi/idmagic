package bootstrap

import (
	"context"

	"github.com/ambi/idmagic/backend/idmanagement"
	"github.com/ambi/idmagic/backend/oauth2"
	"github.com/jackc/pgx/v5"
)

func deleteApplicationClientInTx(ctx context.Context, tx pgx.Tx, tenantID, clientID string) (int, error) {
	count, err := oauth2.PrepareApplicationDeletionInTx(ctx, tx, tenantID, clientID)
	if err != nil {
		return 0, err
	}
	return count, idmanagement.DeleteClientBindingsInTx(ctx, tx, tenantID, clientID)
}
