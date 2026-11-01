package idmanagement

import (
	"context"

	"github.com/ambi/idmagic/backend/idmanagement/internal/agent/db_postgres"
	"github.com/jackc/pgx/v5"
)

// DeleteClientBindingsInTx はクライアントの削除と同じ確定で資格情報の束縛だけを消し、Agent 本体は維持する。
func DeleteClientBindingsInTx(ctx context.Context, tx pgx.Tx, tenantID, clientID string) error {
	return db_postgres.New(tx).DeleteClientCredentialBindings(ctx, db_postgres.DeleteClientCredentialBindingsParams{TenantID: tenantID, ClientID: clientID})
}

func (m Module) DeleteClientBindings(ctx context.Context, tenantID, clientID string) error {
	if m.AgentRepo == nil {
		return nil
	}
	agent, err := m.AgentRepo.FindByClientID(ctx, tenantID, clientID)
	if err != nil || agent == nil {
		return err
	}
	_, err = m.AgentRepo.RemoveBinding(ctx, tenantID, agent.ID, clientID)
	return err
}
