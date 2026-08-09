// admin_data_export_handler.go: admin CSV data export endpoints (wi-148,
// ADR-140). Export is exposed per resource type — /users/exports,
// /groups/exports, and /groups/{group_id}/members/exports — mirroring how
// Entra / Okta / Google surface CSV export (and symmetric with the per-type
// /users/imports). The internal job pipeline, CSV generation, storage, and
// download are shared (idmanagement/usecases), so these handlers are thin
// per-type wrappers that fix the target (and, for members, the group) and
// delegate. The cross-type "all exports / all jobs" listing is wi-157.
package handlers_http

import (
	"errors"
	"net/http"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

func exportUsecaseDeps(d Deps) idmusecases.DataExportDeps {
	exporter := userusecases.UserCSVExporter{
		Deps: userusecases.UserCSVExportDeps{
			UserRepo: d.UserRepo, SchemaReader: userusecases.TenantUserCSVSchemaReader{Repository: d.AttrSchemaRepo}, Artifacts: d.UserCSVArtifacts,
		},
		Policy: userdomain.DefaultUserCSVTransferPolicy(),
	}
	return idmusecases.DataExportDeps{
		UserRepo: d.UserRepo, GroupRepo: d.GroupRepo, JobRepo: d.JobRepo,
		UserCSVExporter: exporter, UserCSVArtifacts: d.UserCSVArtifacts,
		Emit: d.LegacyEmit(), QuotaRepo: d.QuotaRepo,
	}
}

// scopeFor builds the read/download/cancel scope for target from the route. For
// group_membership the group is taken from the :group_id path parameter.
func scopeFor(c *echo.Context, target idmdomain.DataExportTargetKind) idmusecases.ExportScope {
	scope := idmusecases.ExportScope{Target: target}
	if target == idmdomain.ExportTargetGroupMembership {
		scope.GroupID = c.Param("group_id")
	}
	return scope
}

type dataExportResponse struct {
	ID               string   `json:"id"`
	Status           string   `json:"status"`
	Target           string   `json:"target"`
	Format           string   `json:"format"`
	RequestedColumns []string `json:"requested_columns"`
	Filename         string   `json:"filename,omitempty"`
	TotalRows        *int     `json:"total_rows,omitempty"`
	ByteSize         *int     `json:"byte_size,omitempty"`
	ErrorCode        string   `json:"error_code,omitempty"`
	RequestedBy      string   `json:"requested_by"`
	CreatedAt        string   `json:"created_at"`
	CompletedAt      string   `json:"completed_at,omitempty"`
	ExpiresAt        string   `json:"expires_at,omitempty"`
	Downloadable     bool     `json:"downloadable"`
}

func toDataExportResponse(v *idmusecases.DataExportView) dataExportResponse {
	resp := dataExportResponse{
		ID:               v.ID,
		Status:           string(v.Status),
		Target:           v.Target,
		Format:           v.Format,
		RequestedColumns: v.RequestedColumns,
		Filename:         v.Filename,
		TotalRows:        v.TotalRows,
		ByteSize:         v.ByteSize,
		ErrorCode:        v.ErrorCode,
		RequestedBy:      v.RequestedBy,
		CreatedAt:        v.CreatedAt.UTC().Format(time.RFC3339),
		Downloadable:     v.Downloadable,
	}
	if v.CompletedAt != nil {
		resp.CompletedAt = v.CompletedAt.UTC().Format(time.RFC3339)
	}
	if v.ExpiresAt != nil {
		resp.ExpiresAt = v.ExpiresAt.UTC().Format(time.RFC3339)
	}
	return resp
}

type startDataExportRequest struct {
	Columns []string          `json:"columns"`
	Filter  map[string]string `json:"filter,omitempty"`
}

// handleStartExport is the shared body of the per-type start handlers. For
// group_membership the group is fixed from the path (the request body's filter
// cannot widen the scope).
func handleStartExport(d Deps, c *echo.Context, target idmdomain.DataExportTargetKind) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.JobRepo == nil {
		return support.WriteBrowserError(c, http.StatusServiceUnavailable, "jobs_unavailable", "The job service is unavailable.")
	}
	var in startDataExportRequest
	if err := support.DecodeJSON(c.Request(), &in); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The export request is invalid.")
	}
	filter := in.Filter
	if target == idmdomain.ExportTargetGroupMembership {
		filter = map[string]string{"group_id": c.Param("group_id")}
	}
	view, err := idmusecases.StartDataExport(c.Request().Context(), exportUsecaseDeps(d), actor.ID, string(target), in.Columns, filter, time.Now().UTC())
	if err != nil {
		return writeDataExportError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusAccepted, toDataExportResponse(view))
}

func handleListExports(d Deps, c *echo.Context, target idmdomain.DataExportTargetKind) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.JobRepo == nil {
		return support.WriteBrowserError(c, http.StatusServiceUnavailable, "jobs_unavailable", "The job service is unavailable.")
	}
	views, err := idmusecases.ListDataExports(c.Request().Context(), exportUsecaseDeps(d), scopeFor(c, target))
	if err != nil {
		return writeDataExportError(c, err)
	}
	out := make([]dataExportResponse, len(views))
	for i, v := range views {
		out[i] = toDataExportResponse(v)
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"exports": out})
}

func handleGetExport(d Deps, c *echo.Context, target idmdomain.DataExportTargetKind) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.JobRepo == nil {
		return support.WriteBrowserError(c, http.StatusServiceUnavailable, "jobs_unavailable", "The job service is unavailable.")
	}
	view, err := idmusecases.GetDataExport(c.Request().Context(), exportUsecaseDeps(d), scopeFor(c, target), c.Param("export_id"))
	if err != nil {
		return writeDataExportError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, toDataExportResponse(view))
}

func handleDownloadExport(d Deps, c *echo.Context, target idmdomain.DataExportTargetKind) error {
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.JobRepo == nil {
		return support.WriteBrowserError(c, http.StatusServiceUnavailable, "jobs_unavailable", "The job service is unavailable.")
	}
	file, err := idmusecases.DownloadDataExport(c.Request().Context(), exportUsecaseDeps(d), scopeFor(c, target), actor.ID, c.Param("export_id"))
	if err != nil {
		return writeDataExportError(c, err)
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename=\""+file.Filename+"\"")
	if file.Reader != nil {
		defer func() { _ = file.Reader.Close() }()
		return c.Stream(http.StatusOK, file.ContentType, file.Reader)
	}
	return c.Blob(http.StatusOK, file.ContentType, file.Content)
}

func handleCancelExport(d Deps, c *echo.Context, target idmdomain.DataExportTargetKind) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.JobRepo == nil {
		return support.WriteBrowserError(c, http.StatusServiceUnavailable, "jobs_unavailable", "The job service is unavailable.")
	}
	view, err := idmusecases.CancelDataExport(c.Request().Context(), exportUsecaseDeps(d), scopeFor(c, target), actor.ID, c.Param("export_id"))
	if err != nil {
		return writeDataExportError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusOK, toDataExportResponse(view))
}

// Per-type user export handlers.
func HandleStartUserExport(d Deps, c *echo.Context) error {
	return handleStartExport(d, c, idmdomain.ExportTargetUser)
}

func HandleListUserExports(d Deps, c *echo.Context) error {
	return handleListExports(d, c, idmdomain.ExportTargetUser)
}

func HandleGetUserExport(d Deps, c *echo.Context) error {
	return handleGetExport(d, c, idmdomain.ExportTargetUser)
}

func HandleDownloadUserExportFile(d Deps, c *echo.Context) error {
	return handleDownloadExport(d, c, idmdomain.ExportTargetUser)
}

func HandleCancelUserExport(d Deps, c *echo.Context) error {
	return handleCancelExport(d, c, idmdomain.ExportTargetUser)
}

// Per-type group export handlers.
func HandleStartGroupExport(d Deps, c *echo.Context) error {
	return handleStartExport(d, c, idmdomain.ExportTargetGroup)
}

func HandleListGroupExports(d Deps, c *echo.Context) error {
	return handleListExports(d, c, idmdomain.ExportTargetGroup)
}

func HandleGetGroupExport(d Deps, c *echo.Context) error {
	return handleGetExport(d, c, idmdomain.ExportTargetGroup)
}

func HandleDownloadGroupExportFile(d Deps, c *echo.Context) error {
	return handleDownloadExport(d, c, idmdomain.ExportTargetGroup)
}

func HandleCancelGroupExport(d Deps, c *echo.Context) error {
	return handleCancelExport(d, c, idmdomain.ExportTargetGroup)
}

// Per-group member export handlers (nested under a specific group).
func HandleStartGroupMemberExport(d Deps, c *echo.Context) error {
	return handleStartExport(d, c, idmdomain.ExportTargetGroupMembership)
}

func HandleListGroupMemberExports(d Deps, c *echo.Context) error {
	return handleListExports(d, c, idmdomain.ExportTargetGroupMembership)
}

func HandleGetGroupMemberExport(d Deps, c *echo.Context) error {
	return handleGetExport(d, c, idmdomain.ExportTargetGroupMembership)
}

func HandleDownloadGroupMemberExportFile(d Deps, c *echo.Context) error {
	return handleDownloadExport(d, c, idmdomain.ExportTargetGroupMembership)
}

func HandleCancelGroupMemberExport(d Deps, c *echo.Context) error {
	return handleCancelExport(d, c, idmdomain.ExportTargetGroupMembership)
}

func writeDataExportError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, idmdomain.ErrInvalidExportColumns):
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_columns", "The selected columns are not allowed for this target.")
	case errors.Is(err, idmdomain.ErrInvalidExportTarget):
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_target", "The export target is not supported.")
	case errors.Is(err, idmusecases.ErrInvalidExportFilter):
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_filter", "The export filter is not allowed.")
	case errors.Is(err, idmusecases.ErrExportNotFound):
		return support.WriteBrowserError(c, http.StatusNotFound, "data_export_not_found", "The export does not exist.")
	case errors.Is(err, idmusecases.ErrExportNotDownloadable):
		return support.WriteBrowserError(c, http.StatusConflict, "data_export_not_downloadable", "The export is not available for download.")
	case errors.Is(err, jobsports.ErrJobAlreadyTerminal):
		return support.WriteBrowserError(c, http.StatusConflict, "data_export_not_cancelable", "The export has already finished.")
	}
	var quotaErr *tenancydomain.QuotaExceededError
	if errors.As(err, &quotaErr) {
		return support.WriteBrowserError(c, http.StatusTooManyRequests, "quota_exceeded", "The active job quota has been exceeded.")
	}
	return support.WriteServerError(c, err)
}
