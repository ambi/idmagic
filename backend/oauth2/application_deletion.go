package oauth2

import (
	"context"
	"errors"

	clientports "github.com/ambi/idmagic/backend/oauth2/client/ports"
	"github.com/ambi/idmagic/backend/oauth2/internal/db_postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// PrepareApplicationDeletionInTx はクライアントを固定して保護を確認し、削除を妨げる専用状態を消す。
// クライアント本体は呼び出し側の Application の削除に連鎖させる。
func PrepareApplicationDeletionInTx(ctx context.Context, tx pgx.Tx, tenantID, clientID string) (int, error) {
	q := db_postgres.New(tx)
	firstParty, err := q.LockClientForApplicationDeletion(ctx, db_postgres.LockClientForApplicationDeletionParams{TenantID: tenantID, ClientID: clientID})
	if err != nil {
		return 0, err
	}
	if firstParty {
		return 0, clientports.ErrFirstPartyClientProtected
	}
	if err := q.DeleteRefreshTokensForClient(ctx, clientID); err != nil {
		return 0, err
	}
	var clientUUID pgtype.UUID
	if err := clientUUID.Scan(clientID); err != nil {
		return 0, err
	}
	consents, err := q.DeleteConsentsForClient(ctx, clientUUID)
	return int(consents), err
}

func (m Module) deleteApplicationClientState(ctx context.Context, tenantID, clientID string) (int, error) {
	if m.ClientSessionStore != nil {
		cleaner, ok := m.ClientSessionStore.(interface {
			DeleteForClient(context.Context, string, string) error
		})
		if !ok {
			return 0, errors.New("client session store does not support application deletion")
		}
		if err := cleaner.DeleteForClient(ctx, tenantID, clientID); err != nil {
			return 0, err
		}
	}
	if m.RefreshStore != nil {
		cleaner, ok := m.RefreshStore.(interface {
			DeleteForClient(context.Context, string, string) error
		})
		if !ok {
			return 0, errors.New("refresh token store does not support application deletion")
		}
		if err := cleaner.DeleteForClient(ctx, tenantID, clientID); err != nil {
			return 0, err
		}
	}
	if m.ConsentRepo != nil {
		cleaner, ok := m.ConsentRepo.(interface {
			DeleteForClient(context.Context, string, string) (int, error)
		})
		if !ok {
			return 0, errors.New("consent repository does not support application deletion")
		}
		return cleaner.DeleteForClient(ctx, tenantID, clientID)
	}
	return 0, nil
}
