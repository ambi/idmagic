package handlers_http

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	agentusecases "github.com/ambi/idmagic/backend/idmanagement/internal/agent/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

type agentInventoryEntryResponse struct {
	Agent             agentSummaryResponse           `json:"agent"`
	OwnerStatus       agentdomain.OwnerStatus        `json:"owner_status"`
	LastTokenIssuedAt *time.Time                     `json:"last_token_issued_at,omitempty"`
	MissingClientIDs  []string                       `json:"missing_client_ids"`
	Findings          []agentdomain.InventoryFinding `json:"findings"`
}

type agentInventoryResponse struct {
	DormantDays int                           `json:"dormant_days"`
	Agents      []agentInventoryEntryResponse `json:"agents"`
}

// HandleListAgentInventory は Agent の棚卸しを返す (REQ-IDMANAGEMENT-094)。
func HandleListAgentInventory(d Deps, c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	filter := agentusecases.InventoryFilter{
		Finding: agentdomain.InventoryFinding(c.QueryParam("finding")), DormantDays: agentdomain.DefaultDormantDays,
	}
	if raw := c.QueryParam("dormant_days"); raw != "" {
		days, err := strconv.Atoi(raw)
		if err != nil || days < 1 {
			return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", agentusecases.ErrInvalidDormantDays.Error())
		}
		filter.DormantDays = days
	}
	entries, err := agentusecases.ListAgentInventory(c.Request().Context(), adminAgentDeps(d), filter, time.Now().UTC())
	if errors.Is(err, agentusecases.ErrInvalidDormantDays) || errors.Is(err, agentusecases.ErrInvalidInventoryFinding) {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", err.Error())
	}
	if err != nil {
		return err
	}
	agents := make([]agentInventoryEntryResponse, len(entries))
	for i, entry := range entries {
		agents[i] = agentInventoryEntryResponse{
			Agent:             toAgentSummaryResponse(entry.Facts.Agent, entry.Facts.ClientIDs),
			OwnerStatus:       entry.Facts.OwnerStatus,
			LastTokenIssuedAt: entry.Facts.LastTokenIssuedAt,
			MissingClientIDs:  entry.Facts.MissingClientIDs,
			Findings:          entry.Findings,
		}
	}
	return c.JSON(http.StatusOK, agentInventoryResponse{DormantDays: filter.DormantDays, Agents: agents})
}
