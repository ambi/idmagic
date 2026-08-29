package handlers_http

// Group CSV import の HTTP 境界。プレビューは CSV を 1 回だけ受け取り、適用は
// 成功済みプレビューの ID だけを受け取る。結果のエラー一覧は管理一覧と同じ
// 署名済みカーソル、`Link`、`Pagination-*` で返す。

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupusecases "github.com/ambi/idmagic/backend/idmanagement/group/usecases"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/tenancy"

	"github.com/labstack/echo/v5"
)

const groupImportErrorsQuery = "GetAdminGroupImport;errors"

func groupImportStartDeps(d Deps) groupusecases.GroupImportStartDeps {
	return groupusecases.GroupImportStartDeps{
		Artifacts: d.CSVArtifacts, Jobs: d.JobRepo, QuotaRepo: d.QuotaRepo, Emit: d.Emit,
		Policy: idmdomain.DefaultCSVTransferPolicy(),
	}
}

func HandleImportAdminGroups(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.JobRepo == nil || d.CSVArtifacts == nil {
		return support.WriteProblem(c, http.StatusServiceUnavailable, "group_import_unavailable", "The group import service is unavailable.")
	}
	job, err := groupusecases.StartGroupImportPreview(c.Request().Context(), groupImportStartDeps(d), actor.ID, c.Request().Body, time.Now().UTC())
	if err != nil {
		return writeGroupImportError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusAccepted, map[string]any{"id": job.ID, "status": job.Status, "mode": groupusecases.GroupImportModePreview})
}

func HandleApplyAdminGroupImport(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.JobRepo == nil || d.CSVArtifacts == nil {
		return support.WriteProblem(c, http.StatusServiceUnavailable, "group_import_unavailable", "The group import service is unavailable.")
	}
	job, err := groupusecases.StartGroupImportApply(c.Request().Context(), groupImportStartDeps(d), actor.ID, c.Param("preview_job_id"), time.Now().UTC())
	if err != nil {
		return writeGroupImportError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusAccepted, map[string]any{"id": job.ID, "status": job.Status, "mode": groupusecases.GroupImportModeApply})
}

func HandleGetAdminGroupImport(d Deps, c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.JobRepo == nil || d.CSVArtifacts == nil {
		return support.WriteProblem(c, http.StatusServiceUnavailable, "group_import_unavailable", "The group import service is unavailable.")
	}
	tenantID := tenancy.TenantID(c.Request().Context())
	job, err := d.JobRepo.Get(c.Request().Context(), c.Param("job_id"))
	if errors.Is(err, jobsports.ErrJobNotFound) || job == nil || job.TenantID != tenantID ||
		(job.Kind != jobsdomain.KindGroupImportPreview && job.Kind != jobsdomain.KindGroupImportApply) {
		return support.WriteProblem(c, http.StatusNotFound, "group_import_not_found", "The import does not exist.")
	}
	if err != nil {
		return err
	}
	mode := groupusecases.GroupImportModePreview
	if job.Kind == jobsdomain.KindGroupImportApply {
		mode = groupusecases.GroupImportModeApply
	}
	var result groupusecases.GroupImportResult
	if len(job.Result) > 0 {
		if err := json.Unmarshal(job.Result, &result); err != nil {
			return err
		}
	}
	page, err := support.ParsePageRequest(c, d.PaginationCodec, tenantID, groupImportErrorsQuery+";job="+job.ID,
		groupusecases.GroupImportDefaultErrorLimit, groupusecases.GroupImportMaxErrorLimit)
	if err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", err.Error())
	}
	boundaryOrdinal, err := parseGroupImportErrorKeyset(page.AfterPrimary, page.AfterID, job.ID)
	if err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", "The pagination cursor is invalid.")
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
	errorsPage, err := groupusecases.ReadGroupImportErrorRange(c.Request().Context(), d.CSVArtifacts, tenantID, result, startOrdinal, page.Limit)
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
	if err := support.SetPaginationLinks(c, d.PaginationCodec, d.Issuer, tenantID, groupImportErrorsQuery+";job="+job.ID, page,
		firstPrimary, firstID, lastPrimary, lastID, hasPrevious, hasNext, metadata.TotalPages); err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{
		"id": job.ID, "status": job.Status, "mode": mode, "result": result, "errors": errorsPage,
	})
}

func parseGroupImportErrorKeyset(primary, id, jobID string) (int, error) {
	if primary == "" && id == "" {
		return 0, nil
	}
	ordinal, err := strconv.Atoi(primary)
	if err != nil || ordinal < 1 || id != jobID {
		return 0, errors.New("invalid error ordinal cursor")
	}
	return ordinal, nil
}

func writeGroupImportError(c *echo.Context, err error) error {
	var csvErr *idmdomain.CSVError
	switch {
	case errors.Is(err, groupusecases.ErrGroupImportNotFound):
		return support.WriteProblem(c, http.StatusNotFound, "group_import_not_found", "The preview does not exist.")
	case errors.Is(err, groupusecases.ErrGroupImportPreviewNotReady):
		return support.WriteProblem(c, http.StatusConflict, "preview_not_ready", "The preview has not succeeded.")
	case errors.Is(err, groupusecases.ErrGroupImportDigestMismatch):
		return support.WriteProblem(c, http.StatusConflict, "preview_digest_mismatch", "The preview payload integrity check failed.")
	case errors.As(err, &csvErr):
		return support.WriteProblem(c, http.StatusBadRequest, string(csvErr.Code), "The CSV exceeds the configured transfer policy.")
	default:
		return err
	}
}
