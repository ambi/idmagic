package usecases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/ambi/idmagic/backend/audit/domain"
	"github.com/ambi/idmagic/backend/audit/ports"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenantdomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

const (
	KindAuditExport jobsdomain.JobKind = "audit_export"
	ExportPageSize  int                = 500
)

func init() { jobsdomain.RegisterKind(KindAuditExport, jobsdomain.LaneBulk) }

var (
	ErrEventLimit        = errors.New("event_limit_exceeded")
	ErrByteLimit         = errors.New("byte_limit_exceeded")
	ErrExportFailed      = errors.New("audit_export_failed")
	ErrExportNotFound    = errors.New("audit_export_not_found")
	ErrExportUnavailable = errors.New("audit_export_unavailable")
)

type ExportDeps struct {
	Events    ports.AuditEventRepository
	Artifacts ports.ExportArtifactStore
	Jobs      jobsports.JobRepository
	Enqueuer  jobsports.Enqueuer
	Quotas    tenantports.QuotaRepository
	Codec     *support.CursorCodec
	Emit      func(spec.DomainEvent)
	Metrics   ports.ExportMetrics
	Now       func() time.Time
}

func (d ExportDeps) now() time.Time {
	if d.Now != nil {
		return d.Now().UTC()
	}
	return time.Now().UTC()
}

func (d ExportDeps) record(op, outcome string) {
	if d.Metrics != nil {
		d.Metrics.Record(op, outcome)
	}
}

type ExportScope struct {
	TenantID, RequestedBy string
	System                bool
}

type ExportView struct {
	ID           string     `json:"id"`
	Status       string     `json:"status"`
	Scope        string     `json:"scope"`
	Format       string     `json:"format"`
	RequestedBy  string     `json:"requested_by"`
	CreatedAt    time.Time  `json:"created_at"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	Events       *int       `json:"events,omitempty"`
	ByteSize     *int64     `json:"byte_size,omitempty"`
	ErrorCode    string     `json:"error_code,omitempty"`
	Downloadable bool       `json:"downloadable"`
}

func StartExport(ctx context.Context, d ExportDeps, p ports.ExportParams) (*ExportView, error) {
	if d.Enqueuer == nil || d.Artifacts == nil || d.Jobs == nil || d.Emit == nil {
		return nil, ErrExportFailed
	}
	if p.Policy.Validate() != nil || p.RequestedBy == "" || p.Query.TenantID == "" || p.Format != "json" {
		return nil, ErrExportFailed
	}
	p.RequestedAt = d.now()
	if p.Query.Before.IsZero() || p.Query.Before.After(p.RequestedAt) {
		p.Query.Before = p.RequestedAt
	}
	p.Query.AfterID, p.Query.BeforeID = "", ""
	p.Query.AfterOccurredAt, p.Query.BeforeOccurredAt = time.Time{}, time.Time{}
	p.Query.Limit, p.Query.FromEnd = 0, false
	raw, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	job, err := d.Enqueuer.Enqueue(ctx, jobsports.EnqueueInput{TenantID: p.Query.TenantID, Kind: KindAuditExport, Params: raw, MaxAttempts: 1}, p.RequestedAt)
	if err != nil {
		return nil, err
	}
	d.Emit(&domain.AuditExportRequested{ExportEvent: exportEvent(job, p, p.RequestedAt)})
	d.record("requested", "accepted")
	return exportView(job, p, d.now()), nil
}

// StreamExport は JSON 全体の大きさを検査し、ページ一つと一つのイベントだけを保持する。
func StreamExport(ctx context.Context, d ExportDeps, p ports.ExportParams, out io.Writer, check func() error, truncate bool) (int, error) {
	if d.Codec == nil || (d.Events == nil && !p.NoMatch) {
		return 0, ErrExportFailed
	}
	writer := &exportLimitWriter{writer: out, max: p.Policy.MaxBytes}
	if _, err := io.WriteString(writer, `{"events":[`); err != nil {
		return 0, err
	}
	query := p.Query
	query.Limit = ExportPageSize
	bind, err := json.Marshal(p)
	if err != nil {
		return 0, err
	}
	digest := sha256.Sum256(bind)
	hash := hex.EncodeToString(digest[:])
	var total int
	for !p.NoMatch {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		if check != nil {
			if err := check(); err != nil {
				return total, err
			}
		}
		page, err := d.Events.List(ctx, query)
		if err != nil {
			return total, err
		}
		for _, record := range page {
			if total == p.Policy.MaxEvents {
				if truncate {
					break
				}
				return total, ErrEventLimit
			}
			if total > 0 {
				if _, err := io.WriteString(writer, ","); err != nil {
					return total, err
				}
			}
			item := struct {
				ID         string         `json:"id"`
				TenantID   string         `json:"tenant_id"`
				Type       string         `json:"type"`
				OccurredAt time.Time      `json:"occurred_at"`
				Payload    map[string]any `json:"payload"`
			}{record.ID, record.TenantID, record.Type, record.OccurredAt, record.Payload}
			if err := json.NewEncoder(writer).Encode(item); err != nil {
				return total, err
			}
			total++
		}
		if len(page) < ExportPageSize || (truncate && total == p.Policy.MaxEvents) {
			break
		}
		last := page[len(page)-1]
		marker, err := json.Marshal(struct {
			At time.Time
			ID string
		}{last.OccurredAt, last.ID})
		if err != nil {
			return total, err
		}
		token, err := d.Codec.Encode(support.Cursor{TenantID: query.TenantID, QueryHash: hash, After: string(marker)})
		if err != nil {
			return total, err
		}
		after, err := d.Codec.DecodeForQuery(token, query.TenantID, hash)
		if err != nil {
			return total, err
		}
		var next struct {
			At time.Time
			ID string
		}
		if err := json.Unmarshal([]byte(after), &next); err != nil {
			return total, err
		}
		query.AfterOccurredAt, query.AfterID = next.At, next.ID
	}
	if check != nil {
		if err := check(); err != nil {
			return total, err
		}
	}
	_, err = io.WriteString(writer, "]}")
	return total, err
}

type exportLimitWriter struct {
	writer    io.Writer
	max, size int64
}

func (w *exportLimitWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > w.max-w.size {
		return 0, ErrByteLimit
	}
	n, err := w.writer.Write(p)
	w.size += int64(n)
	return n, err
}

func ExportHandler(d ExportDeps) jobsports.Handler {
	return func(ctx context.Context, job *jobsdomain.Job) (json.RawMessage, error) {
		var p ports.ExportParams
		if json.Unmarshal(job.Params, &p) != nil || p.Query.TenantID != job.TenantID || p.Policy.Validate() != nil || p.Format != "json" {
			return nil, ErrExportFailed
		}
		check := func() error {
			if err := ctx.Err(); err != nil {
				return jobsports.ErrJobLeaseLost
			}
			current, err := d.Jobs.Get(ctx, job.ID)
			if err != nil {
				return ErrExportFailed
			}
			if current.Status != jobsdomain.StatusRunning || current.LeaseOwner == nil || job.LeaseOwner == nil || *current.LeaseOwner != *job.LeaseOwner {
				return jobsports.ErrJobLeaseLost
			}
			return nil
		}
		if err := check(); err != nil {
			return nil, err
		}
		artifact, err := d.Artifacts.Write(ctx, job.TenantID, job.ID, func(out io.Writer) (ports.ExportCompletion, error) {
			count, err := StreamExport(ctx, d, p, out, check, false)
			return ports.ExportCompletion{Events: count, ExpiresAt: d.now().Add(p.Policy.TTL), BeforePublish: check}, err
		})
		if err != nil {
			if errors.Is(err, ErrEventLimit) || errors.Is(err, ErrByteLimit) {
				if d.Metrics != nil {
					d.Metrics.Limit(err.Error())
				}
				return nil, err
			}
			if errors.Is(err, jobsports.ErrJobLeaseLost) || ctx.Err() != nil {
				return nil, jobsports.ErrJobLeaseLost
			}
			return nil, ErrExportFailed
		}
		if err := check(); err != nil {
			return nil, err
		}
		return json.Marshal(artifact)
	}
}

func scopedExport(ctx context.Context, d ExportDeps, s ExportScope, id string) (*jobsdomain.Job, ports.ExportParams, error) {
	job, err := d.Jobs.Get(ctx, id)
	if errors.Is(err, jobsports.ErrJobNotFound) {
		return nil, ports.ExportParams{}, ErrExportNotFound
	}
	if err != nil {
		return nil, ports.ExportParams{}, err
	}
	var p ports.ExportParams
	if job.Kind != KindAuditExport || job.TenantID != s.TenantID || json.Unmarshal(job.Params, &p) != nil || p.Query.TenantID != s.TenantID || p.Query.AllTenants != s.System || p.RequestedBy != s.RequestedBy {
		return nil, p, ErrExportNotFound
	}
	return job, p, nil
}

func GetExport(ctx context.Context, d ExportDeps, s ExportScope, id string) (*ExportView, error) {
	job, p, err := scopedExport(ctx, d, s, id)
	if err != nil {
		return nil, err
	}
	return exportView(job, p, d.now()), nil
}

func ListExports(ctx context.Context, d ExportDeps, s ExportScope) ([]*ExportView, error) {
	jobs, err := d.Jobs.ListByTenantAndKinds(ctx, s.TenantID, []jobsdomain.JobKind{KindAuditExport}, 200)
	if err != nil {
		return nil, err
	}
	views := make([]*ExportView, 0, 20)
	for _, job := range jobs {
		var p ports.ExportParams
		if json.Unmarshal(job.Params, &p) != nil || p.Query.AllTenants != s.System || p.RequestedBy != s.RequestedBy {
			continue
		}
		views = append(views, exportView(job, p, d.now()))
		if len(views) == 20 {
			break
		}
	}
	return views, nil
}

func CancelExport(ctx context.Context, d ExportDeps, s ExportScope, id string) (*ExportView, error) {
	job, p, err := scopedExport(ctx, d, s, id)
	if err != nil {
		return nil, err
	}
	canceled, err := d.Jobs.Cancel(ctx, job.ID, d.now())
	if err != nil {
		return nil, err
	}
	if d.Quotas != nil {
		if err := d.Quotas.Decrement(ctx, job.TenantID, tenantdomain.ResourceActiveJobs, 1); err != nil {
			logging.Error(ctx, "quota: failed to decrement active_jobs on audit export cancellation", "error", err, "tenant_id", job.TenantID)
		}
	}
	d.Emit(&jobsdomain.JobCanceled{At: d.now(), JobID: job.ID, TenantID: job.TenantID})
	return exportView(canceled, p, d.now()), nil
}

func DownloadExport(ctx context.Context, d ExportDeps, s ExportScope, id string) (io.ReadCloser, ports.ExportArtifact, error) {
	job, p, err := scopedExport(ctx, d, s, id)
	if err != nil {
		return nil, ports.ExportArtifact{}, err
	}
	view := exportView(job, p, d.now())
	if !view.Downloadable {
		return nil, ports.ExportArtifact{}, ErrExportUnavailable
	}
	reader, artifact, err := d.Artifacts.Open(ctx, job.TenantID, job.ID)
	if errors.Is(err, ports.ErrExportArtifactNotFound) {
		return nil, artifact, ErrExportUnavailable
	}
	if err != nil {
		return nil, artifact, err
	}
	var result ports.ExportArtifact
	if json.Unmarshal(job.Result, &result) != nil || artifact.Ref != result.Ref || artifact.JobID != result.JobID || artifact.TenantID != result.TenantID || artifact.SHA256 != result.SHA256 || artifact.ByteSize != result.ByteSize || artifact.Events != result.Events || !artifact.ExpiresAt.Equal(result.ExpiresAt) || !d.now().Before(artifact.ExpiresAt) {
		_ = reader.Close()
		return nil, artifact, ErrExportUnavailable
	}
	d.Emit(&domain.AuditExportDownloaded{ExportEvent: exportEvent(job, p, d.now())})
	d.record("downloaded", "accepted")
	return reader, artifact, nil
}

func exportView(job *jobsdomain.Job, p ports.ExportParams, now time.Time) *ExportView {
	view := &ExportView{ID: job.ID, Status: string(job.Status), Scope: "tenant", Format: "json", RequestedBy: p.RequestedBy, CreatedAt: job.CreatedAt}
	if p.Query.AllTenants {
		view.Scope = "system"
	}
	if job.Status == jobsdomain.StatusFailed {
		if job.Error != nil {
			view.ErrorCode = *job.Error
		}
		if view.ErrorCode != ErrByteLimit.Error() && view.ErrorCode != ErrEventLimit.Error() {
			view.ErrorCode = ErrExportFailed.Error()
		}
	}
	if job.Status == jobsdomain.StatusSucceeded {
		var result ports.ExportArtifact
		if json.Unmarshal(job.Result, &result) != nil || result.JobID != job.ID || result.TenantID != job.TenantID {
			view.Status = "failed"
			view.ErrorCode = ErrExportFailed.Error()
			return view
		}
		view.ExpiresAt, view.Events, view.ByteSize = &result.ExpiresAt, &result.Events, &result.ByteSize
		view.Downloadable = now.Before(result.ExpiresAt)
		if !view.Downloadable {
			view.Status = "expired"
		}
	}
	return view
}

func exportEvent(job *jobsdomain.Job, p ports.ExportParams, now time.Time) domain.ExportEvent {
	scope := "tenant"
	if p.Query.AllTenants {
		scope = "system"
	}
	return domain.ExportEvent{At: now, TenantID: job.TenantID, ExportID: job.ID, ActorUserID: p.RequestedBy, Scope: scope}
}

// ObserveExportJob は Jobs の確定後にだけ業務上のライフサイクルを記録する。
func ObserveExportJob(ctx context.Context, d ExportDeps, event spec.DomainEvent) error {
	var id, operation string
	switch e := event.(type) {
	case *jobsdomain.JobStarted:
		id, operation = e.JobID, "started"
	case *jobsdomain.JobSucceeded:
		id, operation = e.JobID, "succeeded"
	case *jobsdomain.JobFailed:
		if !e.Terminal {
			return nil
		}
		id, operation = e.JobID, "failed"
	case *jobsdomain.JobCanceled:
		id, operation = e.JobID, "canceled"
	default:
		return nil
	}
	if d.Jobs == nil || d.Emit == nil {
		return nil
	}
	job, err := d.Jobs.Get(ctx, id)
	if err != nil {
		return err
	}
	if job.Kind != KindAuditExport {
		return nil
	}
	var p ports.ExportParams
	if err := json.Unmarshal(job.Params, &p); err != nil {
		return err
	}
	base := exportEvent(job, p, event.OccurredAt())
	switch operation {
	case "started":
		d.Emit(&domain.AuditExportStarted{ExportEvent: base})
	case "succeeded":
		d.Emit(&domain.AuditExportSucceeded{ExportEvent: base})
	case "failed":
		d.Emit(&domain.AuditExportFailed{ExportEvent: base})
	case "canceled":
		d.Emit(&domain.AuditExportCanceled{ExportEvent: base})
	}
	d.record(operation, operation)
	return nil
}
