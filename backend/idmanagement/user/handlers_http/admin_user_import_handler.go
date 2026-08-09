package handlers_http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/labstack/echo/v5"
)

const userImportErrorsQuery = "GetAdminUserImport;errors"

func userImportStartDeps(d Deps) userusecases.UserImportStartDeps {
	return userusecases.UserImportStartDeps{
		Artifacts: d.UserCSVArtifacts, Jobs: d.JobRepo, QuotaRepo: d.QuotaRepo, Emit: d.Emit,
		Policy: userdomain.DefaultUserCSVTransferPolicy(),
	}
}

func HandleImportAdminUsers(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.JobRepo == nil || d.UserCSVArtifacts == nil {
		return support.WriteBrowserError(c, http.StatusServiceUnavailable, "user_import_unavailable", "The user import service is unavailable.")
	}
	job, err := userusecases.StartUserImportPreview(c.Request().Context(), userImportStartDeps(d), actor.ID, c.Request().Body, time.Now().UTC())
	if err != nil {
		return writeUserImportError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusAccepted, map[string]any{"id": job.ID, "status": job.Status, "mode": userusecases.UserImportModePreview})
}

func HandleApplyAdminUserImport(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.JobRepo == nil || d.UserCSVArtifacts == nil {
		return support.WriteBrowserError(c, http.StatusServiceUnavailable, "user_import_unavailable", "The user import service is unavailable.")
	}
	job, err := userusecases.StartUserImportApply(c.Request().Context(), userImportStartDeps(d), actor.ID, c.Param("preview_job_id"), time.Now().UTC())
	if err != nil {
		return writeUserImportError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusAccepted, map[string]any{"id": job.ID, "status": job.Status, "mode": userusecases.UserImportModeApply})
}

func HandleGetAdminUserImport(d Deps, c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.JobRepo == nil || d.UserCSVArtifacts == nil {
		return support.WriteBrowserError(c, http.StatusServiceUnavailable, "user_import_unavailable", "The user import service is unavailable.")
	}
	tenantID := tenancy.TenantID(c.Request().Context())
	job, err := d.JobRepo.Get(c.Request().Context(), c.Param("job_id"))
	if errors.Is(err, jobsports.ErrJobNotFound) || job == nil || job.TenantID != tenantID || (job.Kind != jobsdomain.KindUserImportPreview && job.Kind != jobsdomain.KindUserImportApply) {
		return support.WriteBrowserError(c, http.StatusNotFound, "user_import_not_found", "The import does not exist.")
	}
	if err != nil {
		return err
	}
	mode := userusecases.UserImportModePreview
	if job.Kind == jobsdomain.KindUserImportApply {
		mode = userusecases.UserImportModeApply
	}
	var result userusecases.UserImportResult
	if len(job.Result) > 0 {
		if err := json.Unmarshal(job.Result, &result); err != nil {
			return err
		}
	}
	page, err := support.ParsePageRequest(c, d.PaginationCodec, tenantID, userImportErrorsQuery+";job="+job.ID,
		userusecases.UserImportDefaultErrorLimit, userusecases.UserImportMaxErrorLimit)
	if err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", err.Error())
	}
	boundaryOrdinal, err := parseUserImportErrorKeyset(page.AfterPrimary, page.AfterID, job.ID)
	if err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "The pagination cursor is invalid.")
	}
	startOrdinal := 1
	switch {
	case page.Anchor == support.PageAnchorEnd:
		startOrdinal = max(1, result.ErrorTotal-page.Limit+1)
	case page.Direction == support.PageBackward:
		startOrdinal = max(1, boundaryOrdinal-page.Limit)
	case boundaryOrdinal > 0:
		startOrdinal = boundaryOrdinal + 1
	}
	errorsPage, err := userusecases.ReadUserImportErrorRange(c.Request().Context(), d.UserCSVArtifacts, tenantID, result, startOrdinal, page.Limit)
	if err != nil {
		return err
	}
	hasPrevious := len(errorsPage) > 0 && startOrdinal > 1
	hasNext := len(errorsPage) > 0 && startOrdinal+len(errorsPage) <= result.ErrorTotal
	metadata := support.CalculatePaginationMetadata(int64(result.ErrorTotal), page)
	support.SetPaginationHeaders(c, metadata)
	firstPrimary, firstID, lastPrimary, lastID := "", "", "", ""
	if len(errorsPage) > 0 {
		firstPrimary, firstID = strconv.Itoa(startOrdinal), job.ID
		lastPrimary, lastID = strconv.Itoa(startOrdinal+len(errorsPage)-1), job.ID
	}
	if err := support.SetPaginationLinks(c, d.PaginationCodec, d.Issuer, tenantID, userImportErrorsQuery+";job="+job.ID, page,
		firstPrimary, firstID, lastPrimary, lastID, hasPrevious, hasNext, metadata.TotalPages); err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{
		"id": job.ID, "status": job.Status, "mode": mode, "result": result, "errors": errorsPage,
	})
}

func parseUserImportErrorKeyset(primary, id, jobID string) (int, error) {
	if primary == "" && id == "" {
		return 0, nil
	}
	ordinal, err := strconv.Atoi(primary)
	if err != nil || ordinal < 1 || id != jobID {
		return 0, errors.New("invalid error ordinal cursor")
	}
	return ordinal, nil
}

func writeUserImportError(c *echo.Context, err error) error {
	var csvErr *userdomain.UserCSVError
	switch {
	case errors.Is(err, userusecases.ErrUserImportNotFound):
		return support.WriteBrowserError(c, http.StatusNotFound, "user_import_not_found", "The preview does not exist.")
	case errors.Is(err, userusecases.ErrUserImportPreviewNotReady):
		return support.WriteBrowserError(c, http.StatusConflict, "preview_not_ready", "The preview has not succeeded.")
	case errors.Is(err, userusecases.ErrUserImportDigestMismatch):
		return support.WriteBrowserError(c, http.StatusConflict, "preview_digest_mismatch", "The preview payload integrity check failed.")
	case errors.As(err, &csvErr):
		return support.WriteBrowserError(c, http.StatusBadRequest, string(csvErr.Code), "The CSV exceeds the configured transfer policy.")
	default:
		return err
	}
}
