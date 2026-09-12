package handlers_http

// SCL interfaces: ListJobs / GetJob / CancelJob と、その制御面の双子
// ListSystemJobs / GetSystemJob / CancelSystemJob (bounded_context: Jobs)。
// REQ-JOBS-012 / REQ-JOBS-013 / REQ-JOBS-014 / REQ-JOBS-015。

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/ambi/idmagic/backend/jobs/domain"
	jobports "github.com/ambi/idmagic/backend/jobs/ports"
	jobusecases "github.com/ambi/idmagic/backend/jobs/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/labstack/echo/v5"
)

// adminJobResponse は SCL AdminJobResponse の双子。
//
// params / result / dedup_key は載せない。いずれも投入した Context が意味を決める
// 不透明な値であり、Jobs はその中身を検証しない。検証していない値を管理画面へ流すと、
// 個人情報が混ざっていないことを Jobs の側から主張できないまま公開することになる
// (REQ-JOBS-014)。error は載せる。理由の分からない失敗の一覧に意味はないからである。
type adminJobResponse struct {
	ID             string               `json:"id"`
	TenantID       string               `json:"tenant_id"`
	Kind           string               `json:"kind"`
	Lane           string               `json:"lane"`
	Status         string               `json:"status"`
	Attempts       int                  `json:"attempts"`
	MaxAttempts    int                  `json:"max_attempts"`
	Error          *string              `json:"error,omitempty"`
	Progress       *adminJobProgressDTO `json:"progress,omitempty"`
	LeaseOwner     *string              `json:"lease_owner,omitempty"`
	LeaseExpiresAt *time.Time           `json:"lease_expires_at,omitempty"`
	RunAt          time.Time            `json:"run_at"`
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
}

// adminJobProgressDTO は SCL JobProgress の双子。ハンドラーが進捗を報告しない限り不在。
type adminJobProgressDTO struct {
	Percent   *int      `json:"percent,omitempty"`
	Message   *string   `json:"message,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toAdminJobResponse(job *domain.Job) adminJobResponse {
	out := adminJobResponse{
		ID: job.ID, TenantID: job.TenantID, Kind: string(job.Kind), Lane: string(job.Lane),
		Status: string(job.Status), Attempts: job.Attempts, MaxAttempts: job.MaxAttempts,
		Error: job.Error, LeaseOwner: job.LeaseOwner, LeaseExpiresAt: job.LeaseExpiresAt,
		RunAt: job.RunAt, CreatedAt: job.CreatedAt, UpdatedAt: job.UpdatedAt,
	}
	if job.Progress != nil {
		out.Progress = &adminJobProgressDTO{
			Percent: job.Progress.Percent, Message: job.Progress.Message, UpdatedAt: job.Progress.UpdatedAt,
		}
	}
	return out
}

type adminJobListResponse struct {
	Jobs       []adminJobResponse `json:"jobs"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

func (d Deps) handleListJobs(c *echo.Context) error {
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	return d.listJobs(c, jobusecases.TenantScope{TenantID: actor.TenantID})
}

func (d Deps) handleListSystemJobs(c *echo.Context) error {
	if _, err := d.RequireControlPlaneUser(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	return d.listJobs(c, jobusecases.TenantScope{AllTenants: true})
}

func (d Deps) listJobs(c *echo.Context, scope jobusecases.TenantScope) error {
	if d.Repo == nil {
		return support.NoStoreJSON(c, http.StatusOK, adminJobListResponse{Jobs: []adminJobResponse{}})
	}
	in, err := parseListJobsQuery(c, scope)
	if err != nil {
		return support.WriteProblem(c, http.StatusBadRequest, "invalid_request", err.Error())
	}
	page, err := jobusecases.ListJobsForAdmin(c.Request().Context(), d.adminDeps(), in)
	if err != nil {
		if errors.Is(err, jobusecases.ErrJobCursorMismatch) {
			return support.WriteProblem(c, http.StatusBadRequest, "invalid_request",
				"The cursor was issued for a different tenant or filter. Start again from the first page.")
		}
		return err
	}
	out := make([]adminJobResponse, len(page.Jobs))
	for i, job := range page.Jobs {
		out[i] = toAdminJobResponse(job)
	}
	return support.NoStoreJSON(c, http.StatusOK, adminJobListResponse{Jobs: out, NextCursor: page.NextCursor})
}

func (d Deps) handleGetJob(c *echo.Context) error {
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	return d.getJob(c, jobusecases.TenantScope{TenantID: actor.TenantID})
}

func (d Deps) handleGetSystemJob(c *echo.Context) error {
	if _, err := d.RequireControlPlaneUser(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	return d.getJob(c, jobusecases.TenantScope{AllTenants: true})
}

func (d Deps) getJob(c *echo.Context, scope jobusecases.TenantScope) error {
	if d.Repo == nil {
		return writeJobNotFound(c)
	}
	job, err := jobusecases.GetJobForAdmin(c.Request().Context(), d.adminDeps(), c.Param("job_id"), scope)
	if err != nil {
		if errors.Is(err, jobports.ErrJobNotFound) {
			return writeJobNotFound(c)
		}
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, toAdminJobResponse(job))
}

func (d Deps) handleCancelJob(c *echo.Context) error {
	// 状態を変えるブラウザー経路なので、セッションに加えて CSRF と Origin を検証する。
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	actor, err := d.RequireAdmin(c)
	if err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	return d.cancelJob(c, jobusecases.TenantScope{TenantID: actor.TenantID})
}

func (d Deps) handleCancelSystemJob(c *echo.Context) error {
	// システム経路でも同じ順で検証する。制御面の取消しは影響範囲が他テナントへ及ぶので、
	// ブラウザー経路の証明を省く理由はいっそう無い。
	if err := d.VerifyBrowserRequest(c); err != nil {
		return err
	}
	if _, err := d.RequireControlPlaneUser(c); err != nil {
		return d.WriteAdminAccessError(c, err)
	}
	return d.cancelJob(c, jobusecases.TenantScope{AllTenants: true})
}

func (d Deps) cancelJob(c *echo.Context, scope jobusecases.TenantScope) error {
	if d.Repo == nil {
		return writeJobNotFound(c)
	}
	job, err := jobusecases.CancelJobForAdmin(c.Request().Context(), d.adminDeps(),
		c.Param("job_id"), scope, time.Now().UTC())
	switch {
	case errors.Is(err, jobports.ErrJobNotFound):
		return writeJobNotFound(c)
	case errors.Is(err, jobports.ErrJobAlreadyTerminal):
		// 成功として黙認しない。止めるよう頼んだ運用者にとって、すでに終わっていたのか
		// 止まったのかは別の事実である。
		return support.WriteProblem(c, http.StatusConflict, "job_not_cancelable",
			"The job has already reached a terminal state and cannot be canceled.")
	case err != nil:
		return err
	}
	return support.NoStoreJSON(c, http.StatusOK, toAdminJobResponse(job))
}

func (d Deps) adminDeps() jobusecases.AdminJobDeps {
	return jobusecases.AdminJobDeps{Repo: d.Repo, Emit: d.Emit}
}

// writeJobNotFound は「他テナントの Job」と「存在しない Job」を同じ応答にする。
// 区別できる応答を返すと、id を総当たりするだけで他テナントに Job があることが分かる。
func writeJobNotFound(c *echo.Context) error {
	return support.WriteProblem(c, http.StatusNotFound, "job_not_found", "The job does not exist.")
}

// parseListJobsQuery は query string を一覧の入力へ変換する。テナントの範囲は呼んだ経路が
// 決めて引数で渡り、クエリが動かせるのは絞り込みだけである。未知の状態・種別・レーンは
// 無視せず拒否する。黙って無視すると、運用者は絞り込んだつもりの一覧を絞り込みなしで読む。
func parseListJobsQuery(c *echo.Context, scope jobusecases.TenantScope) (jobusecases.ListJobsInput, error) {
	in := jobusecases.ListJobsInput{Scope: scope}
	query := c.Request().URL.Query()
	for _, raw := range query["status"] {
		status := domain.JobStatus(raw)
		if !status.Valid() {
			return jobusecases.ListJobsInput{}, errors.New("status is invalid")
		}
		in.Statuses = append(in.Statuses, status)
	}
	for _, raw := range query["kind"] {
		kind := domain.JobKind(raw)
		if !kind.Valid() {
			return jobusecases.ListJobsInput{}, errors.New("kind is invalid")
		}
		in.Kinds = append(in.Kinds, kind)
	}
	if raw := c.QueryParam("lane"); raw != "" {
		lane := domain.ExecutionLane(raw)
		if !lane.Valid() {
			return jobusecases.ListJobsInput{}, errors.New("lane is invalid")
		}
		in.Lane = lane
	}
	if raw := c.QueryParam("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 0 {
			return jobusecases.ListJobsInput{}, errors.New("limit must be a non-negative integer")
		}
		in.Limit = limit
	}
	in.Cursor = c.QueryParam("cursor")
	return in, nil
}
