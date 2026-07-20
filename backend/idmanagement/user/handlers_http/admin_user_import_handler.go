package handlers_http

import (
	"encoding/json"
	"net/http"
	"time"

	userusecases "github.com/ambi/idmagic/backend/idmanagement/user/usecases"
	"github.com/ambi/idmagic/backend/jobs/domain"
	jobports "github.com/ambi/idmagic/backend/jobs/ports"
	jobusecases "github.com/ambi/idmagic/backend/jobs/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/tenancy"
	"github.com/labstack/echo/v5"
)

type userImportRequest struct {
	CSV  string `json:"csv"`
	Mode string `json:"mode"`
}

func HandleImportAdminUsers(d Deps, c *echo.Context) error {
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.JobRepo == nil {
		return support.WriteBrowserError(c, http.StatusServiceUnavailable, "jobs_unavailable", "ジョブ基盤を利用できません")
	}
	var in userImportRequest
	if err := support.DecodeJSON(c.Request(), &in); err != nil {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_csv", "CSVリクエストが不正です")
	}
	if in.Mode != "dry_run" && in.Mode != "apply" {
		return support.WriteBrowserError(c, http.StatusBadRequest, "invalid_request", "mode は dry_run または apply です")
	}
	_, result := userusecases.ParseUserImportCSV(in.CSV)
	if len(result.Errors) > 0 && result.TotalRows == 0 {
		return support.NoStoreJSON(c, http.StatusBadRequest, map[string]any{"error": "invalid_csv", "errors": result.Errors})
	}
	params, err := json.Marshal(userusecases.UserImportParams{CSV: in.CSV, ActorUserID: actor.ID})
	if err != nil {
		return err
	}
	kind := domain.KindUserImportPreview
	if in.Mode == "apply" {
		kind = domain.KindUserImportApply
	}
	now := time.Now().UTC()
	job, err := jobusecases.Enqueue(c.Request().Context(), jobusecases.EnqueueDeps{Repo: d.JobRepo, Emit: d.Emit}, jobports.EnqueueInput{TenantID: tenancy.TenantID(c.Request().Context()), Kind: kind, Params: params}, now)
	if err != nil {
		return err
	}
	return support.NoStoreJSON(c, http.StatusAccepted, map[string]any{"id": job.ID, "status": job.Status, "mode": in.Mode})
}

func HandleGetAdminUserImport(d Deps, c *echo.Context) error {
	if _, err := d.RequireAdmin(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	if d.JobRepo == nil {
		return support.WriteBrowserError(c, http.StatusServiceUnavailable, "jobs_unavailable", "ジョブ基盤を利用できません")
	}
	job, err := d.JobRepo.Get(c.Request().Context(), c.Param("job_id"))
	if err != nil {
		return err
	}
	if job == nil || job.TenantID != tenancy.TenantID(c.Request().Context()) || (job.Kind != domain.KindUserImportPreview && job.Kind != domain.KindUserImportApply) {
		return support.WriteBrowserError(c, http.StatusNotFound, "user_import_not_found", "インポートが存在しません")
	}
	var result any
	_ = json.Unmarshal(job.Result, &result)
	return support.NoStoreJSON(c, http.StatusOK, map[string]any{"id": job.ID, "status": job.Status, "result": result})
}
