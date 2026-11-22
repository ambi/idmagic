package usecases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	idmports "github.com/ambi/idmagic/backend/idmanagement/internal/ports"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
)

// CancelExpiredCSVImports は入力の期限を過ぎた非終端ジョブを取り消す。
// 適用は元のプレビューの入力を使うため、適用ジョブ自身の作成時刻では判定しない。
func CancelExpiredCSVImports(ctx context.Context, artifacts idmports.CSVArtifactStore, jobs jobsports.JobRepository, canceler jobsports.Canceler, now time.Time) error {
	if jobs == nil || canceler == nil {
		return errors.New("CSV import retention requires jobs and a canceler")
	}
	filter := jobsports.AdminJobFilter{
		AllTenants: true,
		Statuses:   []jobsdomain.JobStatus{jobsdomain.StatusQueued, jobsdomain.StatusRunning},
		Kinds: []jobsdomain.JobKind{
			jobsdomain.KindUserImportPreview, jobsdomain.KindUserImportApply,
			jobsdomain.KindGroupImportPreview, jobsdomain.KindGroupImportApply,
			jobsdomain.KindGroupMembershipImportPreview, jobsdomain.KindGroupMembershipImportApply,
		},
		Limit: 200,
	}
	for {
		page, err := jobs.ListForAdmin(ctx, filter)
		if err != nil {
			return err
		}
		for _, job := range page {
			expired, err := csvImportInputExpired(ctx, artifacts, jobs, job, now)
			if err != nil {
				return fmt.Errorf("CSV import %s retention: %w", job.ID, err)
			}
			if !expired {
				continue
			}
			if _, err := canceler.Cancel(ctx, job.ID, now); err != nil && !errors.Is(err, jobsports.ErrJobAlreadyTerminal) {
				return err
			}
		}
		if len(page) < filter.Limit {
			return nil
		}
		last := page[len(page)-1]
		filter.BeforeCreatedAt, filter.BeforeID = last.CreatedAt, last.ID
	}
}

func csvImportInputExpired(ctx context.Context, artifacts idmports.CSVArtifactStore, jobs jobsports.JobRepository, job *jobsdomain.Job, now time.Time) (bool, error) {
	type sourceParams struct {
		ArtifactRef  string `json:"artifact_ref"`
		PreviewJobID string `json:"preview_job_id"`
	}
	var params sourceParams
	if err := json.Unmarshal(job.Params, &params); err != nil {
		return false, err
	}
	var previewKind jobsdomain.JobKind
	switch job.Kind {
	case jobsdomain.KindUserImportApply:
		previewKind = jobsdomain.KindUserImportPreview
	case jobsdomain.KindGroupImportApply:
		previewKind = jobsdomain.KindGroupImportPreview
	case jobsdomain.KindGroupMembershipImportApply:
		previewKind = jobsdomain.KindGroupMembershipImportPreview
	}
	if previewKind != "" {
		preview, err := jobs.Get(ctx, params.PreviewJobID)
		if err != nil {
			return false, err
		}
		if preview == nil || preview.TenantID != job.TenantID || preview.Kind != previewKind {
			return false, errors.New("CSV import preview binding is invalid")
		}
		var bound sourceParams
		if err := json.Unmarshal(preview.Params, &bound); err != nil {
			return false, err
		}
		params = bound
	}
	if params.ArtifactRef == "" {
		return false, errors.New("CSV import artifact reference is missing")
	}
	reader, artifact, err := artifacts.OpenCSVArtifact(ctx, job.TenantID, params.ArtifactRef)
	if errors.Is(err, idmports.ErrCSVArtifactNotFound) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if err := reader.Close(); err != nil {
		return false, err
	}
	if artifact.CreatedAt.IsZero() {
		return false, errors.New("CSV artifact creation time is missing")
	}
	return artifact.CreatedAt.Before(now.Add(-CSVArtifactRetention)), nil
}
