package handlers_http

import (
	"errors"
	"net/http"
	"time"

	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	agentusecases "github.com/ambi/idmagic/backend/idmanagement/internal/agent/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"

	"github.com/labstack/echo/v5"
)

const listAgentObservationsQuery = "ListAgentObservations"

type agentObservationInput struct {
	Issuer                 string     `json:"issuer"`
	Subject                string     `json:"subject"`
	ExternalRegistrationID *string    `json:"external_registration_id"`
	DisplayName            *string    `json:"display_name"`
	Resources              []string   `json:"resources"`
	ResponsibleCandidate   *string    `json:"responsible_candidate"`
	ObservedAt             *time.Time `json:"observed_at"`
}

type importAgentObservationsRequest struct {
	Observations []agentObservationInput `json:"observations"`
}

type agentObservationCandidateResponse struct {
	AgentID   string                    `json:"agent_id"`
	AgentName string                    `json:"agent_name"`
	Reasons   []agentdomain.MatchReason `json:"reasons"`
}

type agentObservationResponse struct {
	ID                     string                              `json:"id"`
	Issuer                 string                              `json:"issuer"`
	Subject                string                              `json:"subject"`
	ExternalRegistrationID *string                             `json:"external_registration_id,omitempty"`
	DisplayName            *string                             `json:"display_name,omitempty"`
	Resources              []string                            `json:"resources"`
	ResponsibleCandidate   *string                             `json:"responsible_candidate,omitempty"`
	ObservedAt             time.Time                           `json:"observed_at"`
	SourceTokenID          string                              `json:"source_token_id"`
	State                  agentdomain.ObservationState        `json:"state"`
	Stale                  bool                                `json:"stale"`
	Candidates             []agentObservationCandidateResponse `json:"candidates"`
	ConfirmedAgentID       *string                             `json:"confirmed_agent_id,omitempty"`
}

func toAgentObservationResponse(view agentusecases.ObservationView) agentObservationResponse {
	candidates := make([]agentObservationCandidateResponse, len(view.Match.Candidates))
	for i, candidate := range view.Match.Candidates {
		candidates[i] = agentObservationCandidateResponse{
			AgentID: candidate.AgentID, AgentName: view.AgentNames[candidate.AgentID], Reasons: candidate.Reasons,
		}
	}
	o := view.Observation
	return agentObservationResponse{
		ID: o.ID, Issuer: o.Issuer, Subject: o.Subject, ExternalRegistrationID: o.ExternalRegistrationID,
		DisplayName: o.DisplayName, Resources: o.Resources, ResponsibleCandidate: o.ResponsibleCandidate,
		ObservedAt: o.ObservedAt, SourceTokenID: o.SourceTokenID, State: view.Match.State, Stale: view.Stale,
		Candidates: candidates, ConfirmedAgentID: o.ConfirmedAgentID,
	}
}

// HandleImportAgentObservations は外部の発見器のバッチを取り込む (REQ-IDMANAGEMENT-095)。
// スコープの判定は認証の部品が契約から行い、ここはセッションからの取り込みを拒否する。
func HandleImportAgentObservations(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, tokenID, err := d.RequireAdminApiToken(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input importAgentObservationsRequest
	if err := support.DecodeJSON(c.Request(), &input); err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	batch := make([]*agentdomain.AgentObservation, len(input.Observations))
	for i, item := range input.Observations {
		batch[i] = &agentdomain.AgentObservation{
			Issuer: item.Issuer, Subject: item.Subject, ExternalRegistrationID: item.ExternalRegistrationID,
			DisplayName: item.DisplayName, Resources: item.Resources, ResponsibleCandidate: item.ResponsibleCandidate,
		}
		if item.ObservedAt != nil {
			batch[i].ObservedAt = *item.ObservedAt
		}
	}
	counts, err := agentusecases.ImportObservations(c.Request().Context(), adminAgentDeps(d), actor.ID, tokenID, batch, time.Now().UTC())
	if err != nil {
		return writeAgentObservationError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]int{
		"created": counts.Created, "updated": counts.Updated, "unchanged": counts.Unchanged,
	})
}

// HandleListAgentObservations は観測を候補とともに返す (REQ-IDMANAGEMENT-098)。
func HandleListAgentObservations(d Deps, c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	decision, err := agentusecases.ParseObservationDecision(c.QueryParam("decision"))
	if err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", err.Error())
	}
	tenantID := support.RequestTenantID(c)
	queryHash := listAgentObservationsQuery + ":" + string(decision)
	page, err := support.ParsePageRequest(c, d.PaginationCodec, tenantID, queryHash, support.DefaultPageLimit, support.MaxPageLimit)
	if err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", err.Error())
	}
	ctx := c.Request().Context()
	filter := agentports.ObservationFilter{Decision: decision}
	var rows []*agentdomain.AgentObservation
	if page.Direction == support.PageBackward {
		rows, err = d.AgentObservationRepo.ListPageBefore(ctx, tenantID, filter, page.AfterPrimary, page.AfterID, page.Limit+1)
	} else {
		rows, err = d.AgentObservationRepo.ListPage(ctx, tenantID, filter, page.AfterPrimary, page.AfterID, page.Limit+1)
	}
	if err != nil {
		return err
	}
	totalItems, err := d.AgentObservationRepo.Count(ctx, tenantID, filter)
	if err != nil {
		return err
	}
	rows, hasPrevious, hasNext := support.TrimPage(rows, page)
	if page.Anchor == support.PageAnchorEnd {
		rows = support.TrimEndPage(rows, totalItems, page.Limit)
	}
	views, err := agentusecases.ViewObservations(ctx, adminAgentDeps(d), rows, time.Now().UTC())
	if err != nil {
		return err
	}
	metadata := support.CalculatePaginationMetadata(totalItems, page)
	support.SetPaginationHeaders(c, metadata)
	var firstPrimary, firstID, lastPrimary, lastID string
	if len(rows) > 0 {
		firstPrimary, firstID = rows[0].Subject, rows[0].ID
		lastPrimary, lastID = rows[len(rows)-1].Subject, rows[len(rows)-1].ID
	}
	if err := support.SetPaginationLinks(c, d.PaginationCodec, d.Issuer, tenantID, queryHash, page,
		firstPrimary, firstID, lastPrimary, lastID, hasPrevious, hasNext, metadata.TotalPages); err != nil {
		return err
	}
	observations := make([]agentObservationResponse, len(views))
	for i, view := range views {
		observations[i] = toAgentObservationResponse(view)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"observations": observations})
}

// HandleConfirmAgentObservation は観測を Agent に結ぶ (REQ-IDMANAGEMENT-099)。
func HandleConfirmAgentObservation(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	var input struct {
		AgentID string `json:"agent_id"`
	}
	if err := support.DecodeJSON(c.Request(), &input); err != nil || input.AgentID == "" {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The JSON request body is invalid.")
	}
	view, err := agentusecases.ConfirmAgentMatch(c.Request().Context(), adminAgentDeps(d), actor.ID, c.Param("observation_id"), input.AgentID, time.Now().UTC())
	if err != nil {
		return writeAgentObservationError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, toAgentObservationResponse(view))
}

// HandleExcludeAgentObservation は観測を除外する (REQ-IDMANAGEMENT-100)。
func HandleExcludeAgentObservation(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	view, err := agentusecases.ExcludeObservation(c.Request().Context(), adminAgentDeps(d), actor.ID, c.Param("observation_id"), time.Now().UTC())
	if err != nil {
		return writeAgentObservationError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, toAgentObservationResponse(view))
}

// HandleReopenAgentObservation は観測を未決定へ戻す (REQ-IDMANAGEMENT-100)。
func HandleReopenAgentObservation(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	view, err := agentusecases.ReopenObservation(c.Request().Context(), adminAgentDeps(d), actor.ID, c.Param("observation_id"), time.Now().UTC())
	if err != nil {
		return writeAgentObservationError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, toAgentObservationResponse(view))
}

func writeAgentObservationError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, agentusecases.ErrObservationBatchInvalid):
		return support.WriteProblem(c, http.StatusUnprocessableEntity, "agent_observation_batch_invalid", err.Error())
	case errors.Is(err, agentdomain.ErrInvalidObservation):
		return support.WriteProblem(c, http.StatusUnprocessableEntity, "agent_observation_invalid", err.Error())
	case errors.Is(err, agentports.ErrObservationLimitExceeded):
		return support.WriteProblem(c, http.StatusConflict, "agent_observation_limit_exceeded", err.Error())
	case errors.Is(err, agentusecases.ErrObservationNotFound):
		return support.WriteProblem(c, http.StatusNotFound, "agent_observation_not_found", err.Error())
	case errors.Is(err, agentusecases.ErrObservationAlreadyDecided):
		return support.WriteProblem(c, http.StatusConflict, "agent_observation_already_decided", err.Error())
	case errors.Is(err, agentusecases.ErrAgentNotFound):
		return support.WriteProblem(c, http.StatusNotFound, "agent_not_found", "The agent does not exist.")
	}
	return err
}
