package http

import (
	"errors"
	"net/http"
	"slices"
	"time"

	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	agentusecases "github.com/ambi/idmagic/backend/idmanagement/agent/usecases"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	"github.com/ambi/idmagic/backend/shared/adapters/http/support"

	"github.com/labstack/echo/v5"
)

type agentRegisterRequest struct {
	Name        string               `json:"name"`
	Description *string              `json:"description"`
	Kind        *idmdomain.AgentKind `json:"kind"`
	OwnerUserID *string              `json:"owner_user_id"`
	Roles       []string             `json:"roles"`
}

type agentUpdateRequest struct {
	Name        *string              `json:"name"`
	Description *string              `json:"description"`
	Kind        *idmdomain.AgentKind `json:"kind"`
	OwnerUserID *string              `json:"owner_user_id"`
	Roles       *[]string            `json:"roles"`
}

type agentCredentialBindRequest struct {
	ClientID string `json:"client_id"`
}

type agentSummaryResponse struct {
	ID          string                `json:"id"`
	TenantID    string                `json:"tenant_id"`
	Name        string                `json:"name"`
	Description *string               `json:"description,omitempty"`
	Kind        idmdomain.AgentKind   `json:"kind"`
	OwnerUserID string                `json:"owner_user_id"`
	Status      idmdomain.AgentStatus `json:"status"`
	Roles       []string              `json:"roles"`
	ClientIDs   []string              `json:"client_ids"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
	DisabledAt  *time.Time            `json:"disabled_at,omitempty"`
	KilledAt    *time.Time            `json:"killed_at,omitempty"`
}

func HandleListAgents(d Deps, c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	views, err := agentusecases.ListAgents(c.Request().Context(), adminAgentDeps(d))
	if err != nil {
		return err
	}
	agents := make([]agentSummaryResponse, len(views))
	for i, view := range views {
		agents[i] = toAgentSummaryResponse(view.Agent, view.ClientIDs)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"agents": agents})
}

func HandleGetAgent(d Deps, c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	view, err := agentusecases.GetAgent(c.Request().Context(), adminAgentDeps(d), c.Param("agent_id"))
	if err != nil {
		return writeAdminAgentError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, toAgentSummaryResponse(view.Agent, view.ClientIDs))
}

func HandleRegisterAgent(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input agentRegisterRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	ownerUserID := ""
	if input.OwnerUserID != nil {
		ownerUserID = *input.OwnerUserID
	}
	agent, err := agentusecases.RegisterAgent(c.Request().Context(), adminAgentDeps(d), agentusecases.RegisterAgentInput{
		ActorUserID: actor.ID, Name: input.Name, Description: input.Description,
		Kind: input.Kind, OwnerUserID: ownerUserID, Roles: input.Roles, Now: time.Now().UTC(),
	})
	if err != nil {
		return writeAdminAgentError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusCreated, toAgentSummaryResponse(agent, []string{}))
}

func HandleUpdateAgent(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input agentUpdateRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	agentID := c.Param("agent_id")
	if _, err := agentusecases.UpdateAgent(c.Request().Context(), adminAgentDeps(d), agentusecases.UpdateAgentInput{
		ActorUserID: actor.ID, ID: agentID,
		Name: input.Name, Description: input.Description, Kind: input.Kind,
		OwnerUserID: input.OwnerUserID, Roles: input.Roles, Now: time.Now().UTC(),
	}); err != nil {
		return writeAdminAgentError(c, err)
	}
	view, err := agentusecases.GetAgent(c.Request().Context(), adminAgentDeps(d), agentID)
	if err != nil {
		return writeAdminAgentError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, toAgentSummaryResponse(view.Agent, view.ClientIDs))
}

func HandleDisableAgent(d Deps, c *echo.Context) error {
	return changeAgentStatus(d, c, func(actorUserID, id string) error {
		_, err := agentusecases.SetAgentDisabled(c.Request().Context(), adminAgentDeps(d), actorUserID, id, true, time.Now().UTC())
		return err
	})
}

func HandleEnableAgent(d Deps, c *echo.Context) error {
	return changeAgentStatus(d, c, func(actorUserID, id string) error {
		_, err := agentusecases.SetAgentDisabled(c.Request().Context(), adminAgentDeps(d), actorUserID, id, false, time.Now().UTC())
		return err
	})
}

func HandleKillAgent(d Deps, c *echo.Context) error {
	return changeAgentStatus(d, c, func(actorUserID, id string) error {
		_, err := agentusecases.KillAgent(c.Request().Context(), adminAgentDeps(d), actorUserID, id, time.Now().UTC())
		return err
	})
}

func HandleDeleteAgent(d Deps, c *echo.Context) error {
	return changeAgentStatus(d, c, func(actorUserID, id string) error {
		return agentusecases.DeleteAgent(c.Request().Context(), adminAgentDeps(d), actorUserID, id, time.Now().UTC())
	})
}

func HandleBindAgentCredential(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input agentCredentialBindRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "JSONリクエストが不正です")
	}
	if err := agentusecases.BindCredential(c.Request().Context(), adminAgentDeps(d), actor.ID, c.Param("agent_id"), input.ClientID, time.Now().UTC()); err != nil {
		return writeAdminAgentError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func HandleUnbindAgentCredential(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if err := agentusecases.UnbindCredential(c.Request().Context(), adminAgentDeps(d), actor.ID, c.Param("agent_id"), c.Param("client_id"), time.Now().UTC()); err != nil {
		return writeAdminAgentError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

// changeAgentStatus は disable / enable / kill / delete の共通処理 (verify + admin gate
// + 204)。
func changeAgentStatus(d Deps, c *echo.Context, action func(actorUserID, id string) error) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if err := action(actor.ID, c.Param("agent_id")); err != nil {
		return writeAdminAgentError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.NoContent(http.StatusNoContent)
}

func adminAgentDeps(d Deps) agentusecases.AdminAgentDeps {
	return agentusecases.AdminAgentDeps{AgentRepo: d.AgentRepo, ClientRepo: d.ClientRepo, UserRepo: d.UserRepo, Emit: d.LegacyEmit()}
}

func toAgentSummaryResponse(agent *agentdomain.Agent, clientIDs []string) agentSummaryResponse {
	if clientIDs == nil {
		clientIDs = []string{}
	}
	return agentSummaryResponse{
		ID: agent.ID, TenantID: agent.TenantID, Name: agent.Name, Description: agent.Description,
		Kind: agent.Kind, OwnerUserID: agent.OwnerUserID, Status: agent.Status,
		Roles: slices.Clone(agent.Roles), ClientIDs: slices.Clone(clientIDs),
		CreatedAt: agent.CreatedAt, UpdatedAt: agent.UpdatedAt,
		DisabledAt: agent.DisabledAt, KilledAt: agent.KilledAt,
	}
}

func writeAdminAgentError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, agentusecases.ErrAgentNotFound):
		return support.WriteBrowserError(c, http.StatusNotFound, "agent_not_found", "エージェントが存在しません")
	case errors.Is(err, agentusecases.ErrAgentClientNotFound):
		return support.WriteBrowserError(c, http.StatusNotFound, "client_not_found", "クライアントが存在しません")
	case errors.Is(err, agentusecases.ErrAgentNameConflict):
		return support.WriteBrowserError(c, http.StatusConflict, "agent_name_conflict", "エージェント名は既に使用されています")
	case errors.Is(err, agentusecases.ErrAgentNameEmpty):
		return support.WriteBrowserError(c, http.StatusBadRequest, "agent_name_required", "エージェント名は必須です")
	case errors.Is(err, agentusecases.ErrAgentOwnerRequired):
		return support.WriteBrowserError(c, http.StatusBadRequest, "agent_owner_required", "所有者は必須です")
	case errors.Is(err, agentusecases.ErrAgentOwnerNotFound):
		return support.WriteBrowserError(c, http.StatusBadRequest, "agent_owner_not_found", "所有者ユーザーが存在しません")
	case errors.Is(err, agentusecases.ErrAgentKilled):
		return support.WriteBrowserError(c, http.StatusConflict, "agent_killed", "緊急停止済みのエージェントは変更できません")
	case errors.Is(err, agentusecases.ErrAgentClientBound):
		return support.WriteBrowserError(c, http.StatusConflict, "agent_client_already_bound", "クライアントは別のエージェントに束縛済みです")
	case errors.Is(err, idmusecases.ErrInvalidRole):
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_role", "roleが不正です")
	default:
		return err
	}
}
