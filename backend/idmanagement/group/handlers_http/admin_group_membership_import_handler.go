package handlers_http

// メンバーシップ CSV import の HTTP 境界。プレビューは CSV を 1 回だけ受け取り、
// 適用は成功済みプレビューの ID だけを受け取る。対象 Group はパスが決め、別 Group
// のジョブ ID は「存在しない」に潰す。結果のエラー一覧は管理一覧と同じ署名済み
// カーソル、`Link`、`Pagination-*` で返す。

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

const groupMembershipImportErrorsQuery = "GetAdminGroupMemberImport;errors"

func groupMembershipImportStartDeps(d Deps) groupusecases.GroupMembershipImportStartDeps {
	return groupusecases.GroupMembershipImportStartDeps{
		Artifacts: d.CSVArtifacts, Jobs: d.JobRepo, QuotaRepo: d.QuotaRepo, Emit: d.Emit,
		Policy: idmdomain.DefaultCSVTransferPolicy(),
	}
}

func writeGroupMembershipImportUnavailable(c *echo.Context) error {
	return support.WriteProblem(c, http.StatusServiceUnavailable,
		"group_membership_import_unavailable", "The group membership import service is unavailable.")
}

func HandleImportAdminGroupMembers(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.JobRepo == nil || d.CSVArtifacts == nil {
		return writeGroupMembershipImportUnavailable(c)
	}
	job, err := groupusecases.StartGroupMembershipImportPreview(
		c.Request().Context(), groupMembershipImportStartDeps(d), actor.ID, c.Param("group_id"),
		c.Request().Body, time.Now().UTC())
	if err != nil {
		return writeGroupMembershipImportError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusAccepted, map[string]any{
		"id": job.ID, "status": job.Status, "mode": groupusecases.GroupMembershipImportModePreview,
	})
}

func HandleApplyAdminGroupMemberImport(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.JobRepo == nil || d.CSVArtifacts == nil {
		return writeGroupMembershipImportUnavailable(c)
	}
	job, err := groupusecases.StartGroupMembershipImportApply(
		c.Request().Context(), groupMembershipImportStartDeps(d), actor.ID,
		c.Param("group_id"), c.Param("preview_job_id"), time.Now().UTC())
	if err != nil {
		return writeGroupMembershipImportError(c, err)
	}
	return support.NoStoreJSON(c, http.StatusAccepted, map[string]any{
		"id": job.ID, "status": job.Status, "mode": groupusecases.GroupMembershipImportModeApply,
	})
}

func HandleGetAdminGroupMemberImport(d Deps, c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.JobRepo == nil || d.CSVArtifacts == nil {
		return writeGroupMembershipImportUnavailable(c)
	}
	tenantID := tenancy.TenantID(c.Request().Context())
	groupID := c.Param("group_id")
	job, err := d.JobRepo.Get(c.Request().Context(), c.Param("job_id"))
	if errors.Is(err, jobsports.ErrJobNotFound) || job == nil || job.TenantID != tenantID ||
		(job.Kind != jobsdomain.KindGroupMembershipImportPreview && job.Kind != jobsdomain.KindGroupMembershipImportApply) {
		return writeGroupMembershipImportNotFound(c)
	}
	if err != nil {
		return err
	}
	var params groupusecases.GroupMembershipImportParams
	if err := json.Unmarshal(job.Params, &params); err != nil {
		return err
	}
	// 別 Group のジョブ ID は「存在しない」と同じ答えに潰す。あるグループが別の
	// グループのジョブ ID を持っているかどうかを、応答の差から知られないためである。
	if params.GroupID != groupID {
		return writeGroupMembershipImportNotFound(c)
	}
	mode := groupusecases.GroupMembershipImportModePreview
	if job.Kind == jobsdomain.KindGroupMembershipImportApply {
		mode = groupusecases.GroupMembershipImportModeApply
	}
	var result groupusecases.GroupMembershipImportResult
	if len(job.Result) > 0 {
		if err := json.Unmarshal(job.Result, &result); err != nil {
			return err
		}
	}
	query := groupMembershipImportErrorsQuery + ";group=" + groupID + ";job=" + job.ID
	page, err := support.ParsePageRequest(c, d.PaginationCodec, tenantID, query,
		groupusecases.GroupMembershipImportDefaultErrorLimit, groupusecases.GroupMembershipImportMaxErrorLimit)
	if err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", err.Error())
	}
	boundaryOrdinal, err := parseGroupMembershipImportErrorKeyset(page.AfterPrimary, page.AfterID, job.ID)
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
	errorsPage, err := groupusecases.ReadGroupMembershipImportErrorRange(
		c.Request().Context(), d.CSVArtifacts, tenantID, result, startOrdinal, page.Limit)
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
	if err := support.SetPaginationLinks(c, d.PaginationCodec, d.Issuer, tenantID, query, page,
		firstPrimary, firstID, lastPrimary, lastID, hasPrevious, hasNext, metadata.TotalPages); err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{
		"id": job.ID, "status": job.Status, "mode": mode, "result": result, "errors": errorsPage,
	})
}

func parseGroupMembershipImportErrorKeyset(primary, id, jobID string) (int, error) {
	if primary == "" && id == "" {
		return 0, nil
	}
	ordinal, err := strconv.Atoi(primary)
	if err != nil || ordinal < 1 || id != jobID {
		return 0, errors.New("invalid error ordinal cursor")
	}
	return ordinal, nil
}

func writeGroupMembershipImportNotFound(c *echo.Context) error {
	return support.WriteProblem(c, http.StatusNotFound,
		"group_membership_import_not_found", "The import does not exist for this group.")
}

func writeGroupMembershipImportError(c *echo.Context, err error) error {
	var csvErr *idmdomain.CSVError
	switch {
	case errors.Is(err, groupusecases.ErrGroupMembershipImportNotFound):
		return writeGroupMembershipImportNotFound(c)
	case errors.Is(err, groupusecases.ErrGroupMembershipImportPreviewNotReady):
		return support.WriteProblem(c, http.StatusConflict, "preview_not_ready", "The preview has not succeeded.")
	case errors.Is(err, groupusecases.ErrGroupMembershipImportDigestMismatch):
		return support.WriteProblem(c, http.StatusConflict, "preview_digest_mismatch",
			"The preview payload integrity check failed.")
	case errors.As(err, &csvErr):
		return support.WriteProblem(c, http.StatusBadRequest, string(csvErr.Code),
			"The CSV exceeds the configured transfer policy.")
	default:
		return err
	}
}
