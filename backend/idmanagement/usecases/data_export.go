package usecases

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	jobsusecases "github.com/ambi/idmagic/backend/jobs/usecases"
	"github.com/ambi/idmagic/backend/shared/logging"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

// KindDataExport is the Jobs.JobKind for one admin CSV data export
// (spec/contexts/identity-management.yaml interfaces.StartResourceCsvExport).
// It is a caller-owned kind registered on the bulk lane:
// exports are throughput-oriented background work, not latency sensitive.
const KindDataExport jobsdomain.JobKind = "data_export"

func init() {
	jobsdomain.RegisterKind(KindDataExport, jobsdomain.LaneBulk)
}

const (
	// DataExportMaxRows caps how many rows one export may contain. It bounds
	// the CSV materialized into the Job result (stores the file there),
	// keeping worker memory and the jobs row size in check.
	DataExportMaxRows = 100_000
	// DataExportMaxBytes caps the generated CSV size for the same reason.
	DataExportMaxBytes = 8 << 20 // 8 MiB
	// DataExportTTL is how long a completed export stays downloadable. It
	// aligns with the Jobs default record retention: once the jobs
	// row is purged the export is gone, and between logical expiry and physical
	// purge the read model reports it expired and refuses download.
	DataExportTTL = 30 * 24 * time.Hour
)

var (
	// ErrInvalidExportFilter is returned when a filter key is not allowlisted
	// for the target or its value is invalid (fail-closed keeps
	// exports to defined filters, never arbitrary querying).
	ErrInvalidExportFilter = errors.New("data export: invalid filter")
	// ErrExportNotFound is returned when an export id does not exist in the
	// caller's tenant (or is not a data_export Job).
	ErrExportNotFound = errors.New("data export: not found")
	// ErrExportNotDownloadable is returned when download is attempted on an
	// export that is not succeeded, or whose file has expired.
	ErrExportNotDownloadable = errors.New("data export: not downloadable")
)

// DataExportParams is the data_export Job's params payload.
type DataExportParams struct {
	Target      string            `json:"target"`
	Columns     []string          `json:"columns"`
	Filter      map[string]string `json:"filter,omitempty"`
	ActorUserID string            `json:"actor_user_id"`
}

// DataExportResult is metadata only for User exports. CSVBase64 remains a
// compatibility field for Group exports until their follow-up work items move
// them to the same immutable artifact contract.
type DataExportResult struct {
	Filename    string `json:"filename"`
	TotalRows   int    `json:"total_rows"`
	ByteSize    int    `json:"byte_size"`
	ArtifactRef string `json:"artifact_ref,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
	CSVBase64   string `json:"csv_base64,omitempty"`
}

// DataExportView is the PII-free public projection of an export
// (spec models.DataExportJob). It never carries the CSV bytes.
type DataExportView struct {
	ID               string
	Status           idmdomain.DataExportStatus
	Target           string
	Format           string
	RequestedColumns []string
	Filename         string
	TotalRows        *int
	ByteSize         *int
	ErrorCode        string
	RequestedBy      string
	CreatedAt        time.Time
	CompletedAt      *time.Time
	ExpiresAt        *time.Time
	Downloadable     bool
}

// DataExportFile is the downloadable file (spec models.DataExportFile).
type DataExportFile struct {
	Filename    string
	ContentType string
	ByteSize    int
	Content     []byte
	Reader      io.ReadCloser
}

// DataExportDeps are the dependencies for the data export usecases.
type DataExportDeps struct {
	UserRepo         userports.UserRepository
	GroupRepo        groupports.GroupRepository
	JobRepo          jobsports.JobRepository
	UserCSVExporter  UserCSVExporter
	UserCSVArtifacts userports.UserCSVArtifactStore
	Emit             func(spec.DomainEvent) error
	// QuotaRepo enforces the tenant's active_jobs Hard Quota at enqueue
	// (wi-160). nil skips enforcement.
	QuotaRepo tenantports.QuotaRepository
	// Now returns the current time; defaults to time.Now().UTC() when nil.
	Now func() time.Time
}

type UserCSVExporter interface {
	ValidateUserCSVColumns(ctx context.Context, columns []string) error
	ExportUserCSV(ctx context.Context, columns []string, status string) (userports.UserCSVArtifact, int, error)
}

func (d DataExportDeps) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now().UTC()
}

// StartDataExport validates the request and enqueues a data_export Job
// (StartResourceCsvExport). Columns must be a subset of the target's allowlist
// and filters must be allowlisted per target; both are fail-closed. It emits
// DataExportRequested and returns the queued export's view.
func StartDataExport(ctx context.Context, deps DataExportDeps, actorUserID, target string, columns []string, filter map[string]string, now time.Time) (*DataExportView, error) {
	kind := idmdomain.DataExportTargetKind(target)
	if kind == idmdomain.ExportTargetUser && deps.UserCSVExporter != nil {
		if err := deps.UserCSVExporter.ValidateUserCSVColumns(ctx, columns); err != nil {
			if csvErr, ok := errors.AsType[*userdomain.UserCSVError](err); ok {
				return nil, fmt.Errorf("%w: %s", idmdomain.ErrInvalidExportColumns, csvErr.Code)
			}
			return nil, err
		}
	} else {
		if err := idmdomain.ValidateExportColumns(kind, columns); err != nil {
			return nil, err
		}
	}
	if err := validateExportFilter(kind, filter); err != nil {
		return nil, err
	}
	tenantID := tenancy.TenantID(ctx)
	params, err := json.Marshal(DataExportParams{Target: target, Columns: columns, Filter: filter, ActorUserID: actorUserID})
	if err != nil {
		return nil, err
	}
	// MaxAttempts=1: export generation is deterministic in the resource data,
	// so a failure won't be fixed by retrying; a single attempt keeps the
	// lifecycle to exactly one Started + one Succeeded/Failed event.
	//
	// Emit is intentionally omitted here so Enqueue does not also emit the
	// generic JobEnqueued: DataExportRequested (emitted below) is the
	// domain-level audit record for this action, and double-auditing the same
	// enqueue is noise (matches group/usecases dynamic-group reconcile).
	job, err := jobsusecases.Enqueue(ctx, jobsusecases.EnqueueDeps{Repo: deps.JobRepo, QuotaRepo: deps.QuotaRepo}, jobsports.EnqueueInput{
		TenantID: tenantID, Kind: KindDataExport, Params: params, MaxAttempts: 1,
	}, now)
	if err != nil {
		return nil, err
	}
	if err := adminEmitExport(deps.Emit, &idmdomain.DataExportRequested{At: now, TenantID: tenantID, ActorUserID: actorUserID, ExportID: job.ID, Target: target, RequestedColumns: columns}); err != nil {
		return nil, err
	}
	return exportViewFromJob(job, now), nil
}

// DataExportHandler is the data_export Job handler. It emits
// DataExportStarted, generates the CSV (paging the target's repository,
// applying the column allowlist projection and CSV formula-injection escaping),
// and on success emits DataExportSucceeded and returns the file in the
// result. On failure it emits DataExportFailed and returns the error so the
// Job terminates as failed (MaxAttempts=1).
func DataExportHandler(deps DataExportDeps) func(context.Context, *jobsdomain.Job) (json.RawMessage, error) {
	return func(ctx context.Context, job *jobsdomain.Job) (json.RawMessage, error) {
		var p DataExportParams
		if err := json.Unmarshal(job.Params, &p); err != nil {
			return nil, err
		}
		ctx = tenancy.WithTenant(ctx, &tenancydomain.Tenant{ID: job.TenantID}, "", "")
		now := deps.now()
		_ = adminEmitExport(deps.Emit, &idmdomain.DataExportStarted{At: now, TenantID: job.TenantID, ExportID: job.ID, Target: p.Target})

		result, err := generateExport(ctx, deps, job.TenantID, p)
		if err != nil {
			code := exportErrorCode(err)
			logging.Error(ctx, "data export failed", "error", err, "export_id", job.ID, "target", p.Target, "error_code", code)
			_ = adminEmitExport(deps.Emit, &idmdomain.DataExportFailed{At: deps.now(), TenantID: job.TenantID, ExportID: job.ID, Target: p.Target, ErrorCode: code})
			return nil, fmt.Errorf("%s", code)
		}
		raw, err := json.Marshal(result)
		if err != nil {
			return nil, err
		}
		_ = adminEmitExport(deps.Emit, &idmdomain.DataExportSucceeded{At: deps.now(), TenantID: job.TenantID, ExportID: job.ID, Target: p.Target, TotalRows: result.TotalRows, ByteSize: result.ByteSize})
		return raw, nil
	}
}

// ExportScope narrows read/download/cancel to one export target (and, for
// group_membership, one group). It is how the per-type HTTP endpoints
// (/users/exports, /groups/exports, /groups/{id}/members/exports) keep each
// collection isolated: a group export id must not resolve under /users/exports,
// and a member export id must not resolve under a different group.
type ExportScope struct {
	Target  idmdomain.DataExportTargetKind
	GroupID string // required and matched only when Target is group_membership
}

func (s ExportScope) matches(p DataExportParams) bool {
	if p.Target != string(s.Target) {
		return false
	}
	if s.Target == idmdomain.ExportTargetGroupMembership && p.Filter["group_id"] != s.GroupID {
		return false
	}
	return true
}

// ListDataExports returns the tenant's data exports within scope, newest first.
func ListDataExports(ctx context.Context, deps DataExportDeps, scope ExportScope) ([]*DataExportView, error) {
	tenantID := tenancy.TenantID(ctx)
	jobs, err := deps.JobRepo.ListByTenantAndKinds(ctx, tenantID, []jobsdomain.JobKind{KindDataExport}, 200)
	if err != nil {
		return nil, err
	}
	now := deps.now()
	views := make([]*DataExportView, 0, len(jobs))
	for _, j := range jobs {
		var p DataExportParams
		if err := json.Unmarshal(j.Params, &p); err != nil || !scope.matches(p) {
			continue
		}
		views = append(views, exportViewFromJob(j, now))
	}
	return views, nil
}

// GetDataExport returns one export's view, scoped to tenant and target.
func GetDataExport(ctx context.Context, deps DataExportDeps, scope ExportScope, exportID string) (*DataExportView, error) {
	job, err := loadScopedExport(ctx, deps, scope, exportID)
	if err != nil {
		return nil, err
	}
	return exportViewFromJob(job, deps.now()), nil
}

// DownloadDataExport returns the export file for a succeeded, non-expired
// export and emits DataExportDownloaded. Other states are rejected with
// ErrExportNotDownloadable (DownloadDataExportFile).
func DownloadDataExport(ctx context.Context, deps DataExportDeps, scope ExportScope, actorUserID, exportID string) (*DataExportFile, error) {
	job, err := loadScopedExport(ctx, deps, scope, exportID)
	if err != nil {
		return nil, err
	}
	now := deps.now()
	view := exportViewFromJob(job, now)
	if !view.Downloadable {
		return nil, ErrExportNotDownloadable
	}
	var result DataExportResult
	if err := json.Unmarshal(job.Result, &result); err != nil {
		return nil, err
	}
	var content []byte
	var contentReader io.ReadCloser
	if result.ArtifactRef != "" {
		if deps.UserCSVArtifacts == nil {
			return nil, ErrExportNotDownloadable
		}
		reader, artifact, err := deps.UserCSVArtifacts.OpenUserCSVArtifact(ctx, job.TenantID, result.ArtifactRef)
		if err != nil {
			return nil, ErrExportNotDownloadable
		}
		if artifact.SHA256 != result.SHA256 || artifact.ByteSize != int64(result.ByteSize) {
			_ = reader.Close()
			return nil, ErrExportNotDownloadable
		}
		contentReader = reader
	} else {
		var err error
		content, err = base64.StdEncoding.DecodeString(result.CSVBase64)
		if err != nil {
			return nil, err
		}
	}
	var p DataExportParams
	_ = json.Unmarshal(job.Params, &p)
	if err := adminEmitExport(deps.Emit, &idmdomain.DataExportDownloaded{At: now, TenantID: job.TenantID, ActorUserID: actorUserID, ExportID: job.ID, Target: p.Target}); err != nil {
		return nil, err
	}
	return &DataExportFile{Filename: result.Filename, ContentType: "text/csv; charset=utf-8", ByteSize: result.ByteSize, Content: content, Reader: contentReader}, nil
}

// CancelDataExport cancels a non-terminal export and emits
// DataExportCanceled (CancelDataExport).
func CancelDataExport(ctx context.Context, deps DataExportDeps, scope ExportScope, actorUserID, exportID string) (*DataExportView, error) {
	job, err := loadScopedExport(ctx, deps, scope, exportID)
	if err != nil {
		return nil, err
	}
	now := deps.now()
	canceled, err := deps.JobRepo.Cancel(ctx, job.ID, now)
	if err != nil {
		return nil, err
	}
	if err := adminEmitExport(deps.Emit, &idmdomain.DataExportCanceled{At: now, TenantID: job.TenantID, ActorUserID: actorUserID, ExportID: job.ID}); err != nil {
		return nil, err
	}
	return exportViewFromJob(canceled, now), nil
}

// loadScopedExport fetches a data_export Job and enforces tenant ownership and
// scope (target, and group for member exports). All mismatches collapse to
// ErrExportNotFound so one collection never leaks another's export ids.
func loadScopedExport(ctx context.Context, deps DataExportDeps, scope ExportScope, exportID string) (*jobsdomain.Job, error) {
	job, err := deps.JobRepo.Get(ctx, exportID)
	if errors.Is(err, jobsports.ErrJobNotFound) {
		return nil, ErrExportNotFound
	}
	if err != nil {
		return nil, err
	}
	if job.TenantID != tenancy.TenantID(ctx) || job.Kind != KindDataExport {
		return nil, ErrExportNotFound
	}
	var p DataExportParams
	if err := json.Unmarshal(job.Params, &p); err != nil || !scope.matches(p) {
		return nil, ErrExportNotFound
	}
	return job, nil
}

func exportViewFromJob(job *jobsdomain.Job, now time.Time) *DataExportView {
	var p DataExportParams
	_ = json.Unmarshal(job.Params, &p)
	expiresAt := job.CreatedAt.Add(DataExportTTL)
	status := mapExportStatus(job.Status, expiresAt, now)
	view := &DataExportView{
		ID:               job.ID,
		Status:           status,
		Target:           p.Target,
		Format:           "csv",
		RequestedColumns: p.Columns,
		RequestedBy:      p.ActorUserID,
		CreatedAt:        job.CreatedAt,
		ExpiresAt:        &expiresAt,
	}
	if job.Status == jobsdomain.StatusFailed && job.Error != nil {
		view.ErrorCode = *job.Error
	}
	if job.Status == jobsdomain.StatusSucceeded && len(job.Result) > 0 {
		var result DataExportResult
		if err := json.Unmarshal(job.Result, &result); err == nil {
			totalRows := result.TotalRows
			byteSize := result.ByteSize
			view.Filename = result.Filename
			view.TotalRows = &totalRows
			view.ByteSize = &byteSize
			completed := job.UpdatedAt
			view.CompletedAt = &completed
			view.Downloadable = status == idmdomain.ExportStatusSucceeded && (result.ArtifactRef != "" || result.CSVBase64 != "")
		}
	}
	return view
}

func mapExportStatus(s jobsdomain.JobStatus, expiresAt, now time.Time) idmdomain.DataExportStatus {
	switch s {
	case jobsdomain.StatusQueued:
		return idmdomain.ExportStatusQueued
	case jobsdomain.StatusRunning:
		return idmdomain.ExportStatusRunning
	case jobsdomain.StatusSucceeded:
		if !now.Before(expiresAt) {
			return idmdomain.ExportStatusExpired
		}
		return idmdomain.ExportStatusSucceeded
	case jobsdomain.StatusFailed:
		return idmdomain.ExportStatusFailed
	case jobsdomain.StatusCanceled:
		return idmdomain.ExportStatusCanceled
	}
	return idmdomain.ExportStatusFailed
}

// generateExport builds the CSV for the requested target and filter.
func generateExport(ctx context.Context, deps DataExportDeps, tenantID string, p DataExportParams) (*DataExportResult, error) {
	kind := idmdomain.DataExportTargetKind(p.Target)
	if kind == idmdomain.ExportTargetUser {
		if deps.UserCSVExporter == nil {
			return nil, errors.New("user CSV exporter is unavailable")
		}
		artifact, totalRows, err := deps.UserCSVExporter.ExportUserCSV(ctx, p.Columns, p.Filter["status"])
		if err != nil {
			return nil, err
		}
		return &DataExportResult{
			Filename:  fmt.Sprintf("%s-export-%s.csv", p.Target, deps.now().Format("20060102-150405")),
			TotalRows: totalRows, ByteSize: int(artifact.ByteSize), ArtifactRef: artifact.Ref, SHA256: artifact.SHA256,
		}, nil
	}
	header := idmdomain.LabelsForColumns(kind, p.Columns)
	var (
		rows [][]string
		err  error
	)
	switch kind {
	case idmdomain.ExportTargetGroup:
		rows, err = groupExportRows(ctx, deps, tenantID, p.Columns)
	case idmdomain.ExportTargetGroupMembership:
		rows, err = groupMembershipExportRows(ctx, deps, tenantID, p.Columns, p.Filter)
	default:
		return nil, idmdomain.ErrInvalidExportTarget
	}
	if err != nil {
		return nil, err
	}
	if len(rows) > DataExportMaxRows {
		return nil, errExportTooLarge
	}
	csvBytes, err := idmdomain.EncodeCSVRecords(header, rows)
	if err != nil {
		return nil, err
	}
	if len(csvBytes) > DataExportMaxBytes {
		return nil, errExportTooLarge
	}
	return &DataExportResult{
		Filename:  fmt.Sprintf("%s-export-%s.csv", p.Target, deps.now().Format("20060102-150405")),
		TotalRows: len(rows),
		ByteSize:  len(csvBytes),
		CSVBase64: base64.StdEncoding.EncodeToString(csvBytes),
	}, nil
}

var errExportTooLarge = errors.New("export_too_large")

func groupExportRows(ctx context.Context, deps DataExportDeps, tenantID string, columns []string) ([][]string, error) {
	groups, err := deps.GroupRepo.ListAll(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Name < groups[j].Name })
	rows := make([][]string, 0, len(groups))
	for _, g := range groups {
		row := make([]string, len(columns))
		for i, col := range columns {
			row[i] = groupColumnValue(g, col)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func groupColumnValue(g *groupdomain.Group, col string) string {
	switch col {
	case "id":
		return g.ID
	case "name":
		return g.Name
	case "description":
		return derefString(g.Description)
	case "membership_type":
		return string(g.MembershipType.Effective())
	case "roles":
		return strings.Join(g.Roles, "|")
	case "created_at":
		return g.CreatedAt.UTC().Format(time.RFC3339)
	case "updated_at":
		return g.UpdatedAt.UTC().Format(time.RFC3339)
	}
	return ""
}

func groupMembershipExportRows(ctx context.Context, deps DataExportDeps, tenantID string, columns []string, filter map[string]string) ([][]string, error) {
	groups, err := deps.GroupRepo.ListAll(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	groupIDFilter := strings.TrimSpace(filter["group_id"])
	usernameNeeded := false
	for _, col := range columns {
		if col == "preferred_username" {
			usernameNeeded = true
		}
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Name < groups[j].Name })
	var rows [][]string
	for _, g := range groups {
		if groupIDFilter != "" && g.ID != groupIDFilter {
			continue
		}
		members, err := deps.GroupRepo.ListMembersByGroup(ctx, tenantID, g.ID)
		if err != nil {
			return nil, err
		}
		sort.Slice(members, func(i, j int) bool { return members[i].UserID < members[j].UserID })
		for _, m := range members {
			username := ""
			if usernameNeeded {
				if u, err := deps.UserRepo.FindBySub(ctx, m.UserID); err == nil && u != nil {
					username = u.PreferredUsername
				}
			}
			row := make([]string, len(columns))
			for i, col := range columns {
				row[i] = membershipColumnValue(g, m, username, col)
			}
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func membershipColumnValue(g *groupdomain.Group, m *groupdomain.GroupMember, username, col string) string {
	switch col {
	case "group_id":
		return m.GroupID
	case "group_name":
		return g.Name
	case "user_id":
		return m.UserID
	case "preferred_username":
		return username
	case "source":
		return string(m.Source)
	case "created_at":
		return m.CreatedAt.UTC().Format(time.RFC3339)
	}
	return ""
}

// validateExportFilter fails closed on any filter key not allowlisted for the
// target, or an invalid value (exports use defined filters only).
// group_membership requires group_id: a member export is always scoped to one
// group (the /groups/{group_id}/members/exports path), matching how Entra /
// Okta / Google expose member CSV export per-group.
func validateExportFilter(kind idmdomain.DataExportTargetKind, filter map[string]string) error {
	if kind == idmdomain.ExportTargetGroupMembership && strings.TrimSpace(filter["group_id"]) == "" {
		return fmt.Errorf("%w: group_id is required for a group membership export", ErrInvalidExportFilter)
	}
	if len(filter) == 0 {
		return nil
	}
	allowed := map[string]struct{}{}
	switch kind {
	case idmdomain.ExportTargetUser:
		allowed["status"] = struct{}{}
	case idmdomain.ExportTargetGroupMembership:
		allowed["group_id"] = struct{}{}
	case idmdomain.ExportTargetGroup:
		// no filters
	default:
		return fmt.Errorf("%w: unknown target", ErrInvalidExportFilter)
	}
	for key, val := range filter {
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("%w: %q is not allowlisted for target %q", ErrInvalidExportFilter, key, kind)
		}
		if key == "status" {
			if !idmdomain.UserStatus(strings.ToLower(strings.TrimSpace(val))).Valid() {
				return fmt.Errorf("%w: invalid status %q", ErrInvalidExportFilter, val)
			}
		}
	}
	return nil
}

func exportErrorCode(err error) string {
	if errors.Is(err, errExportTooLarge) {
		return "export_too_large"
	}
	return "export_failed"
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// adminEmitExport emits a domain event when a sink is configured.
func adminEmitExport(sink func(spec.DomainEvent) error, event spec.DomainEvent) error {
	if sink == nil {
		return nil
	}
	return sink(event)
}
