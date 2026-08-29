package usecases

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	idmports "github.com/ambi/idmagic/backend/idmanagement/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	jobsusecases "github.com/ambi/idmagic/backend/jobs/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

const (
	UserImportDefaultErrorLimit     = 100
	UserImportMaxErrorLimit         = 200
	UserImportErrorArtifactPageSize = 200
)

var (
	ErrUserImportNotFound        = errors.New("user import not found")
	ErrUserImportPreviewNotReady = errors.New("user import preview not ready")
	ErrUserImportDigestMismatch  = errors.New("user import digest mismatch")
)

type UserImportMode string

const (
	UserImportModePreview UserImportMode = "preview"
	UserImportModeApply   UserImportMode = "apply"
)

type UserImportParams struct {
	ArtifactRef  string `json:"artifact_ref,omitempty"`
	SourceSHA256 string `json:"source_sha256"`
	ByteSize     int64  `json:"byte_size,omitempty"`
	PreviewJobID string `json:"preview_job_id,omitempty"`
	ActorUserID  string `json:"actor_user_id"`
}

type UserImportRowError struct {
	Row    int    `json:"row"`
	Column string `json:"column,omitempty"`
	Code   string `json:"code"`
}

type UserImportResult struct {
	SourceSHA256     string `json:"source_sha256"`
	PreviewJobID     string `json:"preview_job_id,omitempty"`
	TotalRows        int    `json:"total_rows"`
	CreatedRows      int    `json:"created_rows"`
	UpdatedRows      int    `json:"updated_rows"`
	UnchangedRows    int    `json:"unchanged_rows"`
	AcceptedRows     int    `json:"accepted_rows,omitempty"`
	RejectedRows     int    `json:"rejected_rows"`
	ErrorArtifactRef string `json:"error_artifact_ref,omitempty"`
	ErrorArtifactSHA string `json:"error_artifact_sha256,omitempty"`
	ErrorTotal       int    `json:"error_total"`
}

type UserImportStartDeps struct {
	Artifacts idmports.CSVArtifactStore
	Jobs      jobsports.JobRepository
	QuotaRepo tenantports.QuotaRepository
	Emit      func(spec.DomainEvent)
	Policy    idmdomain.CSVTransferPolicy
}

func (d UserImportStartDeps) policy() idmdomain.CSVTransferPolicy {
	if d.Policy == (idmdomain.CSVTransferPolicy{}) {
		return idmdomain.DefaultCSVTransferPolicy()
	}
	return d.Policy
}

// StartUserImportPreview streams one upload into immutable storage before
// enqueueing metadata-only job params.
func StartUserImportPreview(ctx context.Context, deps UserImportStartDeps, actorUserID string, input io.Reader, now time.Time) (*jobsdomain.Job, error) {
	if deps.Artifacts == nil || deps.Jobs == nil || input == nil {
		return nil, errors.New("user import preview dependencies are incomplete")
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
	params, err := json.Marshal(UserImportParams{
		ArtifactRef: artifact.Ref, SourceSHA256: artifact.SHA256, ByteSize: artifact.ByteSize, ActorUserID: actorUserID,
	})
	if err != nil {
		return nil, err
	}
	return jobsusecases.Enqueue(ctx, jobsusecases.EnqueueDeps{Repo: deps.Jobs, QuotaRepo: deps.QuotaRepo, Emit: deps.Emit}, jobsports.EnqueueInput{
		TenantID: tenantID, Kind: jobsdomain.KindUserImportPreview, Params: params, MaxAttempts: 1,
	}, now)
}

// StartUserImportApply binds a new apply job to one succeeded same-tenant
// preview. The apply params deliberately do not repeat the artifact reference.
func StartUserImportApply(ctx context.Context, deps UserImportStartDeps, actorUserID, previewJobID string, now time.Time) (*jobsdomain.Job, error) {
	if deps.Artifacts == nil || deps.Jobs == nil {
		return nil, errors.New("user import apply dependencies are incomplete")
	}
	tenantID := tenancy.TenantID(ctx)
	params, result, err := loadBoundPreview(ctx, deps.Jobs, deps.Artifacts, tenantID, previewJobID)
	if err != nil {
		return nil, err
	}
	applyParams, err := json.Marshal(UserImportParams{
		SourceSHA256: params.SourceSHA256, PreviewJobID: previewJobID, ActorUserID: actorUserID,
	})
	if err != nil {
		return nil, err
	}
	if result.SourceSHA256 != params.SourceSHA256 {
		return nil, ErrUserImportDigestMismatch
	}
	return jobsusecases.Enqueue(ctx, jobsusecases.EnqueueDeps{Repo: deps.Jobs, QuotaRepo: deps.QuotaRepo, Emit: deps.Emit}, jobsports.EnqueueInput{
		TenantID: tenantID, Kind: jobsdomain.KindUserImportApply, Params: applyParams, MaxAttempts: 1,
	}, now)
}

type UserImportJobDeps struct {
	Artifacts idmports.CSVArtifactStore
	Jobs      jobsports.JobRepository
	Plan      UserImportPlanDeps
	Apply     UserImportApplyDeps
	Policy    idmdomain.CSVTransferPolicy
	Now       func() time.Time
}

func (d UserImportJobDeps) policy() idmdomain.CSVTransferPolicy {
	if d.Policy == (idmdomain.CSVTransferPolicy{}) {
		return idmdomain.DefaultCSVTransferPolicy()
	}
	return d.Policy
}

func (d UserImportJobDeps) now() time.Time {
	if d.Now != nil {
		return d.Now().UTC()
	}
	return time.Now().UTC()
}

func UserImportJobHandler(deps UserImportJobDeps, mode UserImportMode) func(context.Context, *jobsdomain.Job) (json.RawMessage, error) {
	return func(ctx context.Context, job *jobsdomain.Job) (json.RawMessage, error) {
		if deps.Artifacts == nil {
			return nil, errors.New("user import artifact store is unavailable")
		}
		var params UserImportParams
		if err := json.Unmarshal(job.Params, &params); err != nil {
			return nil, err
		}
		ctx = tenancy.WithTenant(ctx, &tenancydomain.Tenant{ID: job.TenantID}, "", "")
		var source UserImportParams
		if mode == UserImportModeApply {
			bound, previewResult, err := loadBoundPreview(ctx, deps.Jobs, deps.Artifacts, job.TenantID, params.PreviewJobID)
			if err != nil {
				return nil, err
			}
			if params.SourceSHA256 != bound.SourceSHA256 || previewResult.SourceSHA256 != bound.SourceSHA256 {
				return nil, ErrUserImportDigestMismatch
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
			return nil, ErrUserImportDigestMismatch
		}

		result := UserImportResult{SourceSHA256: source.SourceSHA256, PreviewJobID: params.PreviewJobID}
		var summary userdomain.UserImportPlanSummary
		errorArtifact, err := deps.Artifacts.PutCSVArtifactPages(ctx, job.TenantID, func(emitPage func([]byte) error) error {
			pendingErrors := make([]UserImportRowError, 0, UserImportErrorArtifactPageSize)
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
			emit := func(row userdomain.UserImportRowPlan) error {
				if row.Error == nil {
					return nil
				}
				result.ErrorTotal++
				pendingErrors = append(pendingErrors, UserImportRowError{Row: row.Error.Row, Column: row.Error.Column, Code: string(row.Error.Code)})
				if len(pendingErrors) == UserImportErrorArtifactPageSize {
					return flushErrors()
				}
				return nil
			}
			var runErr error
			if mode == UserImportModeApply {
				summary, runErr = ApplyUserImport(ctx, deps.Apply, reader, deps.policy(), params.ActorUserID, deps.now(), emit)
			} else {
				summary, runErr = PlanUserImport(ctx, deps.Plan, reader, deps.policy(), emit)
			}
			if runErr != nil {
				var csvErr *idmdomain.CSVError
				if !errors.As(runErr, &csvErr) {
					return runErr
				}
				summary.RejectedRows++
				result.ErrorTotal++
				pendingErrors = append(pendingErrors, UserImportRowError{Row: csvErr.Row, Column: csvErr.Column, Code: string(csvErr.Code)})
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
		result.AcceptedRows = summary.CreatedRows + summary.UpdatedRows + summary.UnchangedRows
		result.RejectedRows = summary.RejectedRows
		return json.Marshal(result)
	}
}

func loadBoundPreview(ctx context.Context, jobs jobsports.JobRepository, artifacts idmports.CSVArtifactStore, tenantID, previewJobID string) (UserImportParams, UserImportResult, error) {
	preview, err := jobs.Get(ctx, previewJobID)
	if errors.Is(err, jobsports.ErrJobNotFound) || preview == nil || preview.TenantID != tenantID || preview.Kind != jobsdomain.KindUserImportPreview {
		return UserImportParams{}, UserImportResult{}, ErrUserImportNotFound
	}
	if err != nil {
		return UserImportParams{}, UserImportResult{}, err
	}
	if preview.Status != jobsdomain.StatusSucceeded {
		return UserImportParams{}, UserImportResult{}, ErrUserImportPreviewNotReady
	}
	var params UserImportParams
	var result UserImportResult
	if json.Unmarshal(preview.Params, &params) != nil || json.Unmarshal(preview.Result, &result) != nil {
		return UserImportParams{}, UserImportResult{}, ErrUserImportDigestMismatch
	}
	reader, artifact, err := artifacts.OpenCSVArtifact(ctx, tenantID, params.ArtifactRef)
	if err != nil {
		return UserImportParams{}, UserImportResult{}, ErrUserImportDigestMismatch
	}
	_ = reader.Close()
	if artifact.SHA256 != params.SourceSHA256 || artifact.ByteSize != params.ByteSize || result.SourceSHA256 != params.SourceSHA256 {
		return UserImportParams{}, UserImportResult{}, ErrUserImportDigestMismatch
	}
	return params, result, nil
}

// ReadUserImportErrorRange maps an immutable 1-based error ordinal to bounded
// artifact pages, so cursor pagination never scans preceding errors.
func ReadUserImportErrorRange(ctx context.Context, artifacts idmports.CSVArtifactStore, tenantID string, result UserImportResult, startOrdinal, limit int) ([]UserImportRowError, error) {
	if startOrdinal < 1 || limit < 1 || limit > UserImportMaxErrorLimit {
		return nil, errors.New("invalid user import error range")
	}
	if result.ErrorTotal == 0 || startOrdinal > result.ErrorTotal {
		return []UserImportRowError{}, nil
	}
	want := min(limit, result.ErrorTotal-startOrdinal+1)
	pageNumber := (startOrdinal - 1) / UserImportErrorArtifactPageSize
	pageOffset := (startOrdinal - 1) % UserImportErrorArtifactPageSize
	out := make([]UserImportRowError, 0, want)
	for len(out) < want {
		payload, artifact, err := artifacts.ReadCSVArtifactPage(ctx, tenantID, result.ErrorArtifactRef, pageNumber)
		if err != nil {
			return nil, err
		}
		if artifact.SHA256 != result.ErrorArtifactSHA {
			return nil, ErrUserImportDigestMismatch
		}
		var page []UserImportRowError
		if err := json.Unmarshal(payload, &page); err != nil {
			return nil, err
		}
		if pageOffset > len(page) {
			return nil, ErrUserImportDigestMismatch
		}
		take := min(want-len(out), len(page)-pageOffset)
		out = append(out, page[pageOffset:pageOffset+take]...)
		pageNumber++
		pageOffset = 0
	}
	return out, nil
}

func randomImportPassword() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("A1!%x", b), nil
}
