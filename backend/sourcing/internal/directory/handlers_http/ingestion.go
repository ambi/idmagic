package handlers_http

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/usecases"
)

type runResponse struct {
	ID          string     `json:"id"`
	Mode        string     `json:"mode"`
	Status      string     `json:"status"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type countsResponse struct {
	Batches   int `json:"batches"`
	Items     int `json:"items"`
	Created   int `json:"created"`
	Updated   int `json:"updated"`
	Unchanged int `json:"unchanged"`
	Deleted   int `json:"deleted"`
	Failed    int `json:"failed"`
	Disabled  int `json:"disabled"`

	DisabledComputers int `json:"disabled_computers"`
}

type runSummaryResponse struct {
	runResponse
	Counts countsResponse `json:"counts"`
}

type failureResponse struct {
	BatchID    string `json:"batch_id"`
	ExternalID string `json:"external_id"`
	Error      string `json:"error"`
}

func runJSON(run domain.IngestionRun) runResponse {
	return runResponse{ID: run.ID, Mode: string(run.Mode), Status: string(run.Status), StartedAt: run.StartedAt, CompletedAt: run.CompletedAt}
}

func runSummaryJSON(run domain.IngestionRun) runSummaryResponse {
	c := run.Counts
	return runSummaryResponse{runResponse: runJSON(run), Counts: countsResponse{
		Batches: c.Batches, Items: c.Items, Created: c.Created, Updated: c.Updated,
		Unchanged: c.Unchanged, Deleted: c.Deleted, Failed: c.Failed, Disabled: c.Disabled,
		DisabledComputers: c.DisabledComputers,
	}}
}

// userItemRequest は項目の本文である。enabled を省いた項目は有効として扱う。
type userItemRequest struct {
	domain.UserItem
	Enabled *bool `json:"enabled"`
}

func (h handler) startIngestionRun(c *echo.Context) error {
	connector, ok, err := h.authenticateConnector(c)
	if !ok {
		return err
	}
	var request struct {
		Mode string `json:"mode"`
	}
	if err := support.DecodeJSON(c.Request(), &request); err != nil {
		return writeInvalidIngestionRequest(c, "invalid ingestion run request")
	}
	run, err := h.deps.Ingestion.StartRun(c.Request().Context(), support.RequestTenantID(c), connector.ID, request.Mode)
	if errors.Is(err, domain.ErrInvalidMode) {
		return writeInvalidIngestionRequest(c, "mode must be full or incremental")
	}
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusCreated, runJSON(run))
}

func (h handler) sendUserBatch(c *echo.Context) error {
	connector, ok, err := h.authenticateConnector(c)
	if !ok {
		return err
	}
	var request struct {
		BatchID string            `json:"batch_id"`
		Items   []userItemRequest `json:"items"`
	}
	if err := support.DecodeJSON(c.Request(), &request); err != nil {
		return writeInvalidIngestionRequest(c, "invalid user batch")
	}
	items := make([]domain.UserItem, len(request.Items))
	for n, item := range request.Items {
		items[n] = item.UserItem
		items[n].Enabled = item.Enabled == nil || *item.Enabled
	}
	results, err := h.deps.Ingestion.ApplyUserBatch(c.Request().Context(), support.RequestTenantID(c), connector.ID, c.Param("run_id"), request.BatchID, items)
	if err != nil {
		return writeIngestionError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"batch_id": request.BatchID, "results": results})
}

// computerItemRequest はコンピューターの項目の本文である。enabled を省いた項目は有効として扱う。
type computerItemRequest struct {
	domain.ComputerItem
	Enabled *bool `json:"enabled"`
}

func (h handler) sendComputerBatch(c *echo.Context) error {
	connector, ok, err := h.authenticateConnector(c)
	if !ok {
		return err
	}
	var request struct {
		BatchID string                `json:"batch_id"`
		Items   []computerItemRequest `json:"items"`
	}
	if err := support.DecodeJSON(c.Request(), &request); err != nil {
		return writeInvalidIngestionRequest(c, "invalid computer batch")
	}
	items := make([]domain.ComputerItem, len(request.Items))
	for n, item := range request.Items {
		items[n] = item.ComputerItem
		items[n].Enabled = item.Enabled == nil || *item.Enabled
	}
	results, err := h.deps.Ingestion.ApplyComputerBatch(c.Request().Context(), support.RequestTenantID(c), connector.ID, c.Param("run_id"), request.BatchID, items)
	if err != nil {
		return writeIngestionError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"batch_id": request.BatchID, "results": results})
}

func (h handler) completeIngestionRun(c *echo.Context) error {
	connector, ok, err := h.authenticateConnector(c)
	if !ok {
		return err
	}
	var request struct {
		ItemCount *int `json:"item_count"`
	}
	if err := support.DecodeJSON(c.Request(), &request); err != nil || request.ItemCount == nil || *request.ItemCount < 0 {
		return writeInvalidIngestionRequest(c, "item_count must be a non-negative integer")
	}
	run, err := h.deps.Ingestion.CompleteRun(c.Request().Context(), support.RequestTenantID(c), connector.ID, c.Param("run_id"), *request.ItemCount)
	if err != nil {
		return writeIngestionError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, runJSON(run))
}

func (h handler) listIngestionRuns(c *echo.Context) error {
	connectorID, ok, err := h.administeredConnector(c)
	if !ok {
		return err
	}
	runs, err := h.deps.Ingestion.ListRuns(c.Request().Context(), support.RequestTenantID(c), connectorID)
	if err != nil {
		return err
	}
	result := make([]runSummaryResponse, len(runs))
	for n, run := range runs {
		result[n] = runSummaryJSON(run)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"runs": result})
}

func (h handler) getIngestionRun(c *echo.Context) error {
	connectorID, ok, err := h.administeredConnector(c)
	if !ok {
		return err
	}
	run, failures, err := h.deps.Ingestion.GetRun(c.Request().Context(), support.RequestTenantID(c), connectorID, c.Param("run_id"))
	if err != nil {
		return writeIngestionError(c, err)
	}
	listed := make([]failureResponse, len(failures))
	for n, failure := range failures {
		listed[n] = failureResponse{BatchID: failure.BatchID, ExternalID: failure.ExternalID, Error: failure.Error}
	}
	return support.NoStoreJSON(c, http.StatusOK, struct {
		runSummaryResponse
		Failures []failureResponse `json:"failures"`
	}{runSummaryJSON(run), listed})
}

// administeredConnector は管理者を認可し、要求したテナントの Connector の ID を返す。
func (h handler) administeredConnector(c *echo.Context) (string, bool, error) {
	if err := h.requireAdministrator(c); err != nil {
		return "", false, err
	}
	connector, err := h.deps.Service.Get(c.Request().Context(), support.RequestTenantID(c), c.Param("connector_id"))
	if err != nil {
		return "", false, writeAdminError(c, err)
	}
	return connector.ID, true, nil
}

func writeIngestionError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidBatch):
		return writeInvalidIngestionRequest(c, err.Error())
	case errors.Is(err, usecases.ErrRunNotFound):
		return support.WriteProblem(c, http.StatusNotFound, "ingestion_run_not_found", "ingestion run not found")
	case errors.Is(err, usecases.ErrRunClosed):
		return support.WriteProblem(c, http.StatusConflict, "ingestion_run_closed", "the ingestion run no longer accepts this request")
	case errors.Is(err, usecases.ErrRunIncomplete):
		return support.WriteProblem(c, http.StatusConflict, "ingestion_run_incomplete", err.Error())
	case errors.Is(err, usecases.ErrBatchConflict):
		return support.WriteProblem(c, http.StatusConflict, "batch_conflict", err.Error())
	default:
		return err
	}
}

func writeInvalidIngestionRequest(c *echo.Context, detail string) error {
	return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", detail)
}
