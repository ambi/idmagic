package db_postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/ambi/idmagic/backend/idmanagement/agent/db_postgres/sqlcgen"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	sharedpg "github.com/ambi/idmagic/backend/shared/storage/db_postgres"
)

// AgentRepository は ADR-048 の Agent 集約と OAuth2Client 束縛を PostgreSQL に永続化
// する。すべての参照はテナント境界に閉じる。agent_credential_bindings は agents への
// ON DELETE CASCADE FK を持つため、DeleteAgent の cascade は DB 側でも保証される。
// クエリは sqlc 生成 (wi-178, ADR-090); Pool は sqlcgen.DBTX を構造的に満たす。
type AgentRepository struct{ Pool sharedpg.DB }

func timestamptzOrNil(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func textOrNil(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func agentFromRow(row *sqlcgen.Agent) (*agentdomain.Agent, error) {
	a := &agentdomain.Agent{
		ID:          row.ID,
		TenantID:    row.TenantID,
		Name:        row.Name,
		Kind:        idmdomain.AgentKind(row.Kind),
		OwnerUserID: row.OwnerUserID,
		Status:      idmdomain.AgentStatus(row.Status),
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
	if row.Description.Valid {
		a.Description = &row.Description.String
	}
	if row.DisabledAt.Valid {
		disabledAt := row.DisabledAt.Time
		a.DisabledAt = &disabledAt
	}
	if row.KilledAt.Valid {
		killedAt := row.KilledAt.Time
		a.KilledAt = &killedAt
	}
	if err := json.Unmarshal(row.Roles, &a.Roles); err != nil {
		return nil, err
	}
	if a.Roles == nil {
		a.Roles = []string{}
	}
	return a, a.Validate()
}

func (r *AgentRepository) ListByTenant(ctx context.Context, tenantID string) ([]*agentdomain.Agent, error) {
	rows, err := sqlcgen.New(r.Pool).ListAgentsByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]*agentdomain.Agent, 0, len(rows))
	for _, row := range rows {
		agent, err := agentFromRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, agent)
	}
	return out, nil
}

func (r *AgentRepository) FindByID(ctx context.Context, tenantID, id string) (*agentdomain.Agent, error) {
	row, err := sqlcgen.New(r.Pool).FindAgentByID(ctx, sqlcgen.FindAgentByIDParams{TenantID: tenantID, ID: id})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return agentFromRow(row)
}

func (r *AgentRepository) Save(ctx context.Context, agent *agentdomain.Agent) error {
	roles := agent.Roles
	if roles == nil {
		roles = []string{}
	}
	rolesJSON, err := json.Marshal(roles)
	if err != nil {
		return err
	}
	return sqlcgen.New(r.Pool).SaveAgent(ctx, sqlcgen.SaveAgentParams{
		ID:          agent.ID,
		TenantID:    agent.TenantID,
		Name:        agent.Name,
		Description: textOrNil(agent.Description),
		Kind:        string(agent.Kind),
		OwnerUserID: agent.OwnerUserID,
		Status:      string(agent.Status),
		Roles:       rolesJSON,
		CreatedAt:   agent.CreatedAt,
		UpdatedAt:   agent.UpdatedAt,
		DisabledAt:  timestamptzOrNil(agent.DisabledAt),
		KilledAt:    timestamptzOrNil(agent.KilledAt),
	})
}

func (r *AgentRepository) Delete(ctx context.Context, tenantID, id string) error {
	return sqlcgen.New(r.Pool).DeleteAgent(ctx, sqlcgen.DeleteAgentParams{TenantID: tenantID, ID: id})
}

func (r *AgentRepository) ListBindings(ctx context.Context, tenantID, agentID string) ([]*agentdomain.AgentCredentialBinding, error) {
	rows, err := sqlcgen.New(r.Pool).ListAgentBindingsByAgent(ctx, sqlcgen.ListAgentBindingsByAgentParams{
		TenantID: tenantID, AgentID: agentID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*agentdomain.AgentCredentialBinding, 0, len(rows))
	for _, row := range rows {
		out = append(out, &agentdomain.AgentCredentialBinding{AgentID: row.AgentID, ClientID: row.ClientID, CreatedAt: row.CreatedAt})
	}
	return out, nil
}

func (r *AgentRepository) AddBinding(ctx context.Context, binding *agentdomain.AgentCredentialBinding) (bool, error) {
	n, err := sqlcgen.New(r.Pool).AddAgentBinding(ctx, sqlcgen.AddAgentBindingParams{
		AgentID: binding.AgentID, ClientID: binding.ClientID, CreatedAt: binding.CreatedAt,
	})
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *AgentRepository) RemoveBinding(ctx context.Context, tenantID, agentID, clientID string) (bool, error) {
	n, err := sqlcgen.New(r.Pool).RemoveAgentBinding(ctx, sqlcgen.RemoveAgentBindingParams{
		TenantID: tenantID, AgentID: agentID, ClientID: clientID,
	})
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *AgentRepository) FindByClientID(ctx context.Context, tenantID, clientID string) (*agentdomain.Agent, error) {
	row, err := sqlcgen.New(r.Pool).FindAgentByClientID(ctx, sqlcgen.FindAgentByClientIDParams{TenantID: tenantID, ClientID: clientID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return agentFromRow(row)
}
