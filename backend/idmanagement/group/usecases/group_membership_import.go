package usecases

// メンバーシップ CSV の preview/apply ジョブ境界。プレビューが CSV を受け取るのは
// 1 回だけで、適用は成功済みプレビューの ID と server-computed SHA-256 だけを参照する。
// 結合の条件にはテナントに加えて Group も入れる。行エラーは同じ不変ストアの固定件数
// ページへ直列化し、CSV 種別ごとのエラーテーブルを作らない
// (docs/contexts/identity-management/internals.md)。

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
	GroupMembershipImportDefaultErrorLimit     = 100
	GroupMembershipImportMaxErrorLimit         = 200
	GroupMembershipImportErrorArtifactPageSize = 200
)

var (
	ErrGroupMembershipImportNotFound        = errors.New("group membership import not found")
	ErrGroupMembershipImportPreviewNotReady = errors.New("group membership import preview not ready")
	ErrGroupMembershipImportDigestMismatch  = errors.New("group membership import digest mismatch")
)

type GroupMembershipImportMode string

const (
	GroupMembershipImportModePreview GroupMembershipImportMode = "preview"
	GroupMembershipImportModeApply   GroupMembershipImportMode = "apply"
)

// GroupMembershipImportParams は GroupID を持つ。対象 Group はジョブのパラメーターに
// 焼き付け、適用がプレビューへ結び付くときにも同じ Group であることを条件にする。
type GroupMembershipImportParams struct {
	GroupID      string `json:"group_id"`
	ArtifactRef  string `json:"artifact_ref,omitempty"`
	SourceSHA256 string `json:"source_sha256"`
	ByteSize     int64  `json:"byte_size,omitempty"`
	PreviewJobID string `json:"preview_job_id,omitempty"`
	ActorUserID  string `json:"actor_user_id"`
}

type GroupMembershipImportRowError struct {
	Row    int    `json:"row"`
	Column string `json:"column,omitempty"`
	Code   string `json:"code"`
}

// GroupMembershipImportResult は件数とダイジェストだけを持つ。解除は所属していた
// User の実効ロールを変えるため、件数を他の操作と分けて返す。
type GroupMembershipImportResult struct {
	SourceSHA256     string `json:"source_sha256"`
	PreviewJobID     string `json:"preview_job_id,omitempty"`
	GroupID          string `json:"group_id"`
	TotalRows        int    `json:"total_rows"`
	AddedRows        int    `json:"added_rows"`
	RemovedRows      int    `json:"removed_rows"`
	UnchangedRows    int    `json:"unchanged_rows"`
	RejectedRows     int    `json:"rejected_rows"`
	ErrorArtifactRef string `json:"error_artifact_ref,omitempty"`
	ErrorArtifactSHA string `json:"error_artifact_sha256,omitempty"`
	ErrorTotal       int    `json:"error_total"`
}

type GroupMembershipImportStartDeps struct {
	Artifacts idmports.CSVArtifactStore
	Jobs      jobsports.JobRepository
	QuotaRepo tenantports.QuotaRepository
	Emit      func(spec.DomainEvent)
	Policy    idmdomain.CSVTransferPolicy
}

func (d GroupMembershipImportStartDeps) policy() idmdomain.CSVTransferPolicy {
	if d.Policy == (idmdomain.CSVTransferPolicy{}) {
		return idmdomain.DefaultCSVTransferPolicy()
	}
	return d.Policy
}

// StartGroupMembershipImportPreview は 1 回の upload を不変ストアへ流し込んでから、
// メタデータだけのジョブパラメーターを投入する。
func StartGroupMembershipImportPreview(
	ctx context.Context,
	deps GroupMembershipImportStartDeps,
	actorUserID, groupID string,
	input io.Reader,
	now time.Time,
) (*jobsdomain.Job, error) {
	if deps.Artifacts == nil || deps.Jobs == nil || input == nil {
		return nil, errors.New("group membership import preview dependencies are incomplete")
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
	params, err := json.Marshal(GroupMembershipImportParams{
		GroupID: groupID, ArtifactRef: artifact.Ref, SourceSHA256: artifact.SHA256,
		ByteSize: artifact.ByteSize, ActorUserID: actorUserID,
	})
	if err != nil {
		return nil, err
	}
	return jobsusecases.Enqueue(ctx,
		jobsusecases.EnqueueDeps{Repo: deps.Jobs, QuotaRepo: deps.QuotaRepo, Emit: deps.Emit},
		jobsports.EnqueueInput{
			TenantID: tenantID, Kind: jobsdomain.KindGroupMembershipImportPreview, Params: params, MaxAttempts: 1,
		}, now)
}

// StartGroupMembershipImportApply は同一テナントかつ同一 Group の成功済みプレビュー
// 1 件へ適用ジョブを結び付ける。適用のパラメーターは成果物参照を意図的に繰り返さない。
func StartGroupMembershipImportApply(
	ctx context.Context,
	deps GroupMembershipImportStartDeps,
	actorUserID, groupID, previewJobID string,
	now time.Time,
) (*jobsdomain.Job, error) {
	if deps.Artifacts == nil || deps.Jobs == nil {
		return nil, errors.New("group membership import apply dependencies are incomplete")
	}
	tenantID := tenancy.TenantID(ctx)
	params, result, err := loadBoundGroupMembershipPreview(ctx, deps.Jobs, deps.Artifacts, tenantID, groupID, previewJobID)
	if err != nil {
		return nil, err
	}
	if result.SourceSHA256 != params.SourceSHA256 {
		return nil, ErrGroupMembershipImportDigestMismatch
	}
	applyParams, err := json.Marshal(GroupMembershipImportParams{
		GroupID: groupID, SourceSHA256: params.SourceSHA256, PreviewJobID: previewJobID, ActorUserID: actorUserID,
	})
	if err != nil {
		return nil, err
	}
	return jobsusecases.Enqueue(ctx,
		jobsusecases.EnqueueDeps{Repo: deps.Jobs, QuotaRepo: deps.QuotaRepo, Emit: deps.Emit},
		jobsports.EnqueueInput{
			TenantID: tenantID, Kind: jobsdomain.KindGroupMembershipImportApply, Params: applyParams, MaxAttempts: 1,
		}, now)
}

type GroupMembershipImportJobDeps struct {
	Artifacts idmports.CSVArtifactStore
	Jobs      jobsports.JobRepository
	Plan      GroupMembershipImportPlanDeps
	Apply     GroupMembershipImportApplyDeps
	Policy    idmdomain.CSVTransferPolicy
	Now       func() time.Time
}

func (d GroupMembershipImportJobDeps) policy() idmdomain.CSVTransferPolicy {
	if d.Policy == (idmdomain.CSVTransferPolicy{}) {
		return idmdomain.DefaultCSVTransferPolicy()
	}
	return d.Policy
}

func (d GroupMembershipImportJobDeps) now() time.Time {
	if d.Now != nil {
		return d.Now().UTC()
	}
	return time.Now().UTC()
}

func GroupMembershipImportJobHandler(
	deps GroupMembershipImportJobDeps,
	mode GroupMembershipImportMode,
) func(context.Context, *jobsdomain.Job) (json.RawMessage, error) {
	return func(ctx context.Context, job *jobsdomain.Job) (json.RawMessage, error) {
		if deps.Artifacts == nil {
			return nil, errors.New("group membership import artifact store is unavailable")
		}
		var params GroupMembershipImportParams
		if err := json.Unmarshal(job.Params, &params); err != nil {
			return nil, err
		}
		ctx = tenancy.WithTenant(ctx, &tenancydomain.Tenant{ID: job.TenantID}, "", "")
		var source GroupMembershipImportParams
		if mode == GroupMembershipImportModeApply {
			bound, previewResult, err := loadBoundGroupMembershipPreview(
				ctx, deps.Jobs, deps.Artifacts, job.TenantID, params.GroupID, params.PreviewJobID)
			if err != nil {
				return nil, err
			}
			if params.SourceSHA256 != bound.SourceSHA256 || previewResult.SourceSHA256 != bound.SourceSHA256 {
				return nil, ErrGroupMembershipImportDigestMismatch
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
			return nil, ErrGroupMembershipImportDigestMismatch
		}

		result := GroupMembershipImportResult{
			SourceSHA256: source.SourceSHA256, PreviewJobID: params.PreviewJobID, GroupID: params.GroupID,
		}
		var summary GroupMembershipImportPlanSummary
		errorArtifact, err := deps.Artifacts.PutCSVArtifactPages(ctx, job.TenantID, func(emitPage func([]byte) error) error {
			pendingErrors := make([]GroupMembershipImportRowError, 0, GroupMembershipImportErrorArtifactPageSize)
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
			emit := func(row groupdomain.GroupMembershipImportRowPlan) error {
				if row.Error == nil {
					return nil
				}
				result.ErrorTotal++
				pendingErrors = append(pendingErrors, GroupMembershipImportRowError{
					Row: row.Error.Row, Column: row.Error.Column, Code: string(row.Error.Code),
				})
				if len(pendingErrors) == GroupMembershipImportErrorArtifactPageSize {
					return flushErrors()
				}
				return nil
			}
			var runErr error
			if mode == GroupMembershipImportModeApply {
				summary, runErr = ApplyGroupMembershipImport(
					ctx, deps.Apply, params.GroupID, reader, deps.policy(), params.ActorUserID, deps.now(), emit)
			} else {
				summary, runErr = PlanGroupMembershipImport(ctx, deps.Plan, params.GroupID, reader, deps.policy(), emit)
			}
			if runErr != nil {
				var csvErr *idmdomain.CSVError
				if !errors.As(runErr, &csvErr) {
					return runErr
				}
				// ファイル全体の拒否も 1 件のエラーとして記録する。動的グループや
				// 外部所有のように行に紐付かない拒否は Row=0 で現れる。
				summary.RejectedRows++
				result.ErrorTotal++
				pendingErrors = append(pendingErrors, GroupMembershipImportRowError{
					Row: csvErr.Row, Column: csvErr.Column, Code: string(csvErr.Code),
				})
			}
			return flushErrors()
		})
		if err != nil {
			return nil, err
		}
		result.ErrorArtifactRef = errorArtifact.Ref
		result.ErrorArtifactSHA = errorArtifact.SHA256
		result.TotalRows = summary.TotalRows
		result.AddedRows = summary.AddedRows
		result.RemovedRows = summary.RemovedRows
		result.UnchangedRows = summary.UnchangedRows
		result.RejectedRows = summary.RejectedRows
		return json.Marshal(result)
	}
}

// loadBoundGroupMembershipPreview は同一テナントかつ同一 Group の成功済みプレビュー
// だけを返す。別 Group のジョブ ID は「存在しない」に潰し、あるグループが別の
// グループのジョブ ID を漏らさないようにする。
func loadBoundGroupMembershipPreview(
	ctx context.Context,
	jobs jobsports.JobRepository,
	artifacts idmports.CSVArtifactStore,
	tenantID, groupID, previewJobID string,
) (GroupMembershipImportParams, GroupMembershipImportResult, error) {
	preview, err := jobs.Get(ctx, previewJobID)
	if errors.Is(err, jobsports.ErrJobNotFound) || preview == nil || preview.TenantID != tenantID ||
		preview.Kind != jobsdomain.KindGroupMembershipImportPreview {
		return GroupMembershipImportParams{}, GroupMembershipImportResult{}, ErrGroupMembershipImportNotFound
	}
	if err != nil {
		return GroupMembershipImportParams{}, GroupMembershipImportResult{}, err
	}
	var params GroupMembershipImportParams
	if json.Unmarshal(preview.Params, &params) != nil {
		return GroupMembershipImportParams{}, GroupMembershipImportResult{}, ErrGroupMembershipImportDigestMismatch
	}
	if params.GroupID != groupID {
		return GroupMembershipImportParams{}, GroupMembershipImportResult{}, ErrGroupMembershipImportNotFound
	}
	if preview.Status != jobsdomain.StatusSucceeded {
		return GroupMembershipImportParams{}, GroupMembershipImportResult{}, ErrGroupMembershipImportPreviewNotReady
	}
	var result GroupMembershipImportResult
	if json.Unmarshal(preview.Result, &result) != nil {
		return GroupMembershipImportParams{}, GroupMembershipImportResult{}, ErrGroupMembershipImportDigestMismatch
	}
	reader, artifact, err := artifacts.OpenCSVArtifact(ctx, tenantID, params.ArtifactRef)
	if err != nil {
		return GroupMembershipImportParams{}, GroupMembershipImportResult{}, ErrGroupMembershipImportDigestMismatch
	}
	_ = reader.Close()
	if artifact.SHA256 != params.SourceSHA256 || artifact.ByteSize != params.ByteSize ||
		result.SourceSHA256 != params.SourceSHA256 {
		return GroupMembershipImportParams{}, GroupMembershipImportResult{}, ErrGroupMembershipImportDigestMismatch
	}
	return params, result, nil
}

// ReadGroupMembershipImportErrorRange は 1 始まりの不変なエラー通し番号を、上限付きの
// 成果物ページへ直接対応させる。深い位置を読む場合も先行するエラーを走査しない。
func ReadGroupMembershipImportErrorRange(
	ctx context.Context,
	artifacts idmports.CSVArtifactStore,
	tenantID string,
	result GroupMembershipImportResult,
	startOrdinal, limit int,
) ([]GroupMembershipImportRowError, error) {
	if startOrdinal < 1 || limit < 1 || limit > GroupMembershipImportMaxErrorLimit {
		return nil, errors.New("invalid group membership import error range")
	}
	if result.ErrorTotal == 0 || startOrdinal > result.ErrorTotal {
		return []GroupMembershipImportRowError{}, nil
	}
	want := min(limit, result.ErrorTotal-startOrdinal+1)
	pageNumber := (startOrdinal - 1) / GroupMembershipImportErrorArtifactPageSize
	pageOffset := (startOrdinal - 1) % GroupMembershipImportErrorArtifactPageSize
	out := make([]GroupMembershipImportRowError, 0, want)
	for len(out) < want {
		payload, artifact, err := artifacts.ReadCSVArtifactPage(ctx, tenantID, result.ErrorArtifactRef, pageNumber)
		if err != nil {
			return nil, err
		}
		if artifact.SHA256 != result.ErrorArtifactSHA {
			return nil, ErrGroupMembershipImportDigestMismatch
		}
		var page []GroupMembershipImportRowError
		if err := json.Unmarshal(payload, &page); err != nil {
			return nil, err
		}
		if pageOffset > len(page) {
			return nil, ErrGroupMembershipImportDigestMismatch
		}
		take := min(want-len(out), len(page)-pageOffset)
		out = append(out, page[pageOffset:pageOffset+take]...)
		pageNumber++
		pageOffset = 0
	}
	return out, nil
}
