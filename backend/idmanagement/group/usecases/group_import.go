package usecases

// Group CSV の preview/apply ジョブ境界。プレビューが CSV を受け取るのは 1 回だけで、
// 適用は成功済みプレビューの ID と server-computed SHA-256 だけを参照する。
// 行エラーは同じ不変ストアの固定件数ページへ直列化し、CSV 種別ごとのエラーテーブルを
// 作らない (docs/contexts/identity-management/internals.md)。

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	jobsusecases "github.com/ambi/idmagic/backend/jobs/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

const (
	GroupImportDefaultErrorLimit     = 100
	GroupImportMaxErrorLimit         = 200
	GroupImportErrorArtifactPageSize = 200
)

var (
	ErrGroupImportNotFound        = errors.New("group import not found")
	ErrGroupImportPreviewNotReady = errors.New("group import preview not ready")
	ErrGroupImportDigestMismatch  = errors.New("group import digest mismatch")
)

type GroupImportMode string

const (
	GroupImportModePreview GroupImportMode = "preview"
	GroupImportModeApply   GroupImportMode = "apply"
)

type GroupImportParams struct {
	ArtifactRef  string `json:"artifact_ref,omitempty"`
	SourceSHA256 string `json:"source_sha256"`
	ByteSize     int64  `json:"byte_size,omitempty"`
	PreviewJobID string `json:"preview_job_id,omitempty"`
	ActorUserID  string `json:"actor_user_id"`
}

type GroupImportRowError struct {
	Row    int    `json:"row"`
	Column string `json:"column,omitempty"`
	Code   string `json:"code"`
}

// GroupImportResult は件数とダイジェストだけを持つ。削除は不可逆で cascade するため、
// 削除件数と巻き込まれる membership 件数を他の操作と分けて返す。
type GroupImportResult struct {
	SourceSHA256       string `json:"source_sha256"`
	PreviewJobID       string `json:"preview_job_id,omitempty"`
	TotalRows          int    `json:"total_rows"`
	CreatedRows        int    `json:"created_rows"`
	UpdatedRows        int    `json:"updated_rows"`
	UnchangedRows      int    `json:"unchanged_rows"`
	DeletedRows        int    `json:"deleted_rows"`
	DeletedMemberships int    `json:"deleted_memberships"`
	RejectedRows       int    `json:"rejected_rows"`
	ErrorArtifactRef   string `json:"error_artifact_ref,omitempty"`
	ErrorArtifactSHA   string `json:"error_artifact_sha256,omitempty"`
	ErrorTotal         int    `json:"error_total"`
}

type GroupImportStartDeps struct {
	Artifacts idmports.CSVArtifactStore
	Jobs      jobsports.JobRepository
	QuotaRepo tenantports.QuotaRepository
	Emit      func(spec.DomainEvent)
	Policy    idmdomain.CSVTransferPolicy
}

func (d GroupImportStartDeps) policy() idmdomain.CSVTransferPolicy {
	if d.Policy == (idmdomain.CSVTransferPolicy{}) {
		return idmdomain.DefaultCSVTransferPolicy()
	}
	return d.Policy
}

// StartGroupImportPreview は 1 回の upload を不変ストアへ流し込んでから、
// メタデータだけのジョブパラメーターを投入する。
func StartGroupImportPreview(ctx context.Context, deps GroupImportStartDeps, actorUserID string, input io.Reader, now time.Time) (*jobsdomain.Job, error) {
	if deps.Artifacts == nil || deps.Jobs == nil || input == nil {
		return nil, errors.New("group import preview dependencies are incomplete")
	}
	policy := deps.policy()
	if err := policy.Validate(); err != nil {
		return nil, err
	}
	tenantID := tenancy.TenantID(ctx)
	artifact, err := deps.Artifacts.PutCSVArtifact(ctx, tenantID, func(output io.Writer) error {
		limited := &io.LimitedReader{R: input, N: int64(policy.MaxBytes) + 1}
		written, err := io.Copy(output, limited)
		if err != nil {
			return err
		}
		if written > int64(policy.MaxBytes) {
			return &idmdomain.CSVError{Code: idmdomain.CSVErrorCSVTooLarge}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	params, err := json.Marshal(GroupImportParams{
		ArtifactRef: artifact.Ref, SourceSHA256: artifact.SHA256, ByteSize: artifact.ByteSize, ActorUserID: actorUserID,
	})
	if err != nil {
		return nil, err
	}
	return jobsusecases.Enqueue(ctx, jobsusecases.EnqueueDeps{Repo: deps.Jobs, QuotaRepo: deps.QuotaRepo, Emit: deps.Emit}, jobsports.EnqueueInput{
		TenantID: tenantID, Kind: jobsdomain.KindGroupImportPreview, Params: params, MaxAttempts: 1,
	}, now)
}

// StartGroupImportApply は同一テナントの成功済みプレビュー 1 件へ適用ジョブを結び付ける。
// 適用のパラメーターは成果物参照を意図的に繰り返さない。
func StartGroupImportApply(ctx context.Context, deps GroupImportStartDeps, actorUserID, previewJobID string, now time.Time) (*jobsdomain.Job, error) {
	if deps.Artifacts == nil || deps.Jobs == nil {
		return nil, errors.New("group import apply dependencies are incomplete")
	}
	tenantID := tenancy.TenantID(ctx)
	params, result, err := loadBoundGroupPreview(ctx, deps.Jobs, deps.Artifacts, tenantID, previewJobID)
	if err != nil {
		return nil, err
	}
	if result.SourceSHA256 != params.SourceSHA256 {
		return nil, ErrGroupImportDigestMismatch
	}
	applyParams, err := json.Marshal(GroupImportParams{
		SourceSHA256: params.SourceSHA256, PreviewJobID: previewJobID, ActorUserID: actorUserID,
	})
	if err != nil {
		return nil, err
	}
	return jobsusecases.Enqueue(ctx, jobsusecases.EnqueueDeps{Repo: deps.Jobs, QuotaRepo: deps.QuotaRepo, Emit: deps.Emit}, jobsports.EnqueueInput{
		TenantID: tenantID, Kind: jobsdomain.KindGroupImportApply, Params: applyParams, MaxAttempts: 1,
	}, now)
}

type GroupImportJobDeps struct {
	Artifacts idmports.CSVArtifactStore
	Jobs      jobsports.JobRepository
	Plan      GroupImportPlanDeps
	Apply     GroupImportApplyDeps
	Policy    idmdomain.CSVTransferPolicy
	Now       func() time.Time
}

func (d GroupImportJobDeps) policy() idmdomain.CSVTransferPolicy {
	if d.Policy == (idmdomain.CSVTransferPolicy{}) {
		return idmdomain.DefaultCSVTransferPolicy()
	}
	return d.Policy
}

func (d GroupImportJobDeps) now() time.Time {
	if d.Now != nil {
		return d.Now().UTC()
	}
	return time.Now().UTC()
}

func GroupImportJobHandler(deps GroupImportJobDeps, mode GroupImportMode) func(context.Context, *jobsdomain.Job) (json.RawMessage, error) {
	return func(ctx context.Context, job *jobsdomain.Job) (json.RawMessage, error) {
		if deps.Artifacts == nil {
			return nil, errors.New("group import artifact store is unavailable")
		}
		var params GroupImportParams
		if err := json.Unmarshal(job.Params, &params); err != nil {
			return nil, err
		}
		ctx = tenancy.WithTenant(ctx, &tenancydomain.Tenant{ID: job.TenantID}, "", "")
		var source GroupImportParams
		if mode == GroupImportModeApply {
			bound, previewResult, err := loadBoundGroupPreview(ctx, deps.Jobs, deps.Artifacts, job.TenantID, params.PreviewJobID)
			if err != nil {
				return nil, err
			}
			if params.SourceSHA256 != bound.SourceSHA256 || previewResult.SourceSHA256 != bound.SourceSHA256 {
				return nil, ErrGroupImportDigestMismatch
			}
			source = bound
		} else {
			source = params
		}
		reader, artifact, err := deps.Artifacts.OpenCSVArtifact(ctx, job.TenantID, source.ArtifactRef)
		if err != nil {
			return nil, err
		}
		defer func() { _ = reader.Close() }()
		if artifact.SHA256 != source.SourceSHA256 || artifact.ByteSize != source.ByteSize {
			return nil, ErrGroupImportDigestMismatch
		}

		result := GroupImportResult{SourceSHA256: source.SourceSHA256, PreviewJobID: params.PreviewJobID}
		var summary GroupImportPlanSummary
		errorArtifact, err := deps.Artifacts.PutCSVArtifactPages(ctx, job.TenantID, func(emitPage func([]byte) error) error {
			pendingErrors := make([]GroupImportRowError, 0, GroupImportErrorArtifactPageSize)
			flushErrors := func() error {
				if len(pendingErrors) == 0 {
					return nil
				}
				payload, err := json.Marshal(pendingErrors)
				if err != nil {
					return err
				}
				if err := emitPage(payload); err != nil {
					return err
				}
				pendingErrors = pendingErrors[:0]
				return nil
			}
			emit := func(row groupdomain.GroupImportRowPlan) error {
				if row.Error == nil {
					return nil
				}
				result.ErrorTotal++
				pendingErrors = append(pendingErrors, GroupImportRowError{Row: row.Error.Row, Column: row.Error.Column, Code: string(row.Error.Code)})
				if len(pendingErrors) == GroupImportErrorArtifactPageSize {
					return flushErrors()
				}
				return nil
			}
			var runErr error
			if mode == GroupImportModeApply {
				summary, runErr = ApplyGroupImport(ctx, deps.Apply, reader, deps.policy(), params.ActorUserID, deps.now(), emit)
			} else {
				summary, runErr = PlanGroupImport(ctx, deps.Plan, reader, deps.policy(), emit)
			}
			if runErr != nil {
				var csvErr *idmdomain.CSVError
				if !errors.As(runErr, &csvErr) {
					return runErr
				}
				summary.RejectedRows++
				result.ErrorTotal++
				pendingErrors = append(pendingErrors, GroupImportRowError{Row: csvErr.Row, Column: csvErr.Column, Code: string(csvErr.Code)})
			}
			return flushErrors()
		})
		if err != nil {
			return nil, err
		}
		result.ErrorArtifactRef = errorArtifact.Ref
		result.ErrorArtifactSHA = errorArtifact.SHA256
		result.TotalRows = summary.TotalRows
		result.CreatedRows = summary.CreatedRows
		result.UpdatedRows = summary.UpdatedRows
		result.UnchangedRows = summary.UnchangedRows
		result.DeletedRows = summary.DeletedRows
		result.DeletedMemberships = summary.DeletedMemberships
		result.RejectedRows = summary.RejectedRows
		return json.Marshal(result)
	}
}

func loadBoundGroupPreview(ctx context.Context, jobs jobsports.JobRepository, artifacts idmports.CSVArtifactStore, tenantID, previewJobID string) (GroupImportParams, GroupImportResult, error) {
	preview, err := jobs.Get(ctx, previewJobID)
	if errors.Is(err, jobsports.ErrJobNotFound) || preview == nil || preview.TenantID != tenantID || preview.Kind != jobsdomain.KindGroupImportPreview {
		return GroupImportParams{}, GroupImportResult{}, ErrGroupImportNotFound
	}
	if err != nil {
		return GroupImportParams{}, GroupImportResult{}, err
	}
	if preview.Status != jobsdomain.StatusSucceeded {
		return GroupImportParams{}, GroupImportResult{}, ErrGroupImportPreviewNotReady
	}
	var params GroupImportParams
	var result GroupImportResult
	if json.Unmarshal(preview.Params, &params) != nil || json.Unmarshal(preview.Result, &result) != nil {
		return GroupImportParams{}, GroupImportResult{}, ErrGroupImportDigestMismatch
	}
	reader, artifact, err := artifacts.OpenCSVArtifact(ctx, tenantID, params.ArtifactRef)
	if err != nil {
		return GroupImportParams{}, GroupImportResult{}, ErrGroupImportDigestMismatch
	}
	_ = reader.Close()
	if artifact.SHA256 != params.SourceSHA256 || artifact.ByteSize != params.ByteSize || result.SourceSHA256 != params.SourceSHA256 {
		return GroupImportParams{}, GroupImportResult{}, ErrGroupImportDigestMismatch
	}
	return params, result, nil
}

// ReadGroupImportErrorRange は 1 始まりの不変なエラー通し番号を、上限付きの
// 成果物ページへ直接対応させる。深い位置を読む場合も先行するエラーを走査しない。
func ReadGroupImportErrorRange(ctx context.Context, artifacts idmports.CSVArtifactStore, tenantID string, result GroupImportResult, startOrdinal, limit int) ([]GroupImportRowError, error) {
	if startOrdinal < 1 || limit < 1 || limit > GroupImportMaxErrorLimit {
		return nil, errors.New("invalid group import error range")
	}
	if result.ErrorTotal == 0 || startOrdinal > result.ErrorTotal {
		return []GroupImportRowError{}, nil
	}
	want := min(limit, result.ErrorTotal-startOrdinal+1)
	pageNumber := (startOrdinal - 1) / GroupImportErrorArtifactPageSize
	pageOffset := (startOrdinal - 1) % GroupImportErrorArtifactPageSize
	out := make([]GroupImportRowError, 0, want)
	for len(out) < want {
		payload, artifact, err := artifacts.ReadCSVArtifactPage(ctx, tenantID, result.ErrorArtifactRef, pageNumber)
		if err != nil {
			return nil, err
		}
		if artifact.SHA256 != result.ErrorArtifactSHA {
			return nil, ErrGroupImportDigestMismatch
		}
		var page []GroupImportRowError
		if err := json.Unmarshal(payload, &page); err != nil {
			return nil, err
		}
		if pageOffset > len(page) {
			return nil, ErrGroupImportDigestMismatch
		}
		take := min(want-len(out), len(page)-pageOffset)
		out = append(out, page[pageOffset:pageOffset+take]...)
		pageNumber++
		pageOffset = 0
	}
	return out, nil
}
