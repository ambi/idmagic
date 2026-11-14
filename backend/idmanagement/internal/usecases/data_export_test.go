package usecases_test

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/tenancy"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	idmmemory "github.com/ambi/idmagic/backend/idmanagement/internal/db_memory"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/internal/group/db_memory"
	groupusecases "github.com/ambi/idmagic/backend/idmanagement/internal/group/usecases"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/internal/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/internal/user/db_memory"
	userusecases "github.com/ambi/idmagic/backend/idmanagement/internal/user/usecases"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	jobsmodule "github.com/ambi/idmagic/backend/jobs"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

type eventRecorder struct {
	mu     sync.Mutex
	events []spec.DomainEvent
}

func (r *eventRecorder) emit(e spec.DomainEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
	return nil
}

func (r *eventRecorder) types() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.events))
	for i, e := range r.events {
		out[i] = e.EventType()
	}
	return out
}

func exportTestCtx() context.Context {
	return tenantports.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "acme"}, "", "")
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func seededExportDeps(t *testing.T) (idmusecases.DataExportDeps, *eventRecorder) {
	t.Helper()
	deps, rec, _ := seededExportDepsWithQuotas(t)
	return deps, rec
}

// seededExportDepsWithQuotas は、投入器と取り消しが共有するテナントの使用量も返す。
func seededExportDepsWithQuotas(t *testing.T) (idmusecases.DataExportDeps, *eventRecorder, tenantports.QuotaRepository) {
	t.Helper()
	ctx := exportTestCtx()
	users := usermemory.NewUserRepository()
	groups := groupmemory.NewGroupRepository()
	now := time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC)
	email := "alice@example.com"
	name := "Alice"
	if err := users.Save(ctx, &userdomain.User{ID: "u1", TenantID: "acme", PreferredUsername: "alice", PasswordHash: "secret-hash", Email: &email, Name: &name, Roles: []string{"admin"}, Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	// A username crafted to trigger CSV formula injection.
	if err := users.Save(ctx, &userdomain.User{ID: "u2", TenantID: "acme", PreferredUsername: "=cmd|calc", PasswordHash: "secret-hash", Roles: []string{}, Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusDisabled}, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := groups.Save(ctx, &groupdomain.Group{ID: "g1", TenantID: "acme", Name: "engineering", Roles: []string{"dev"}, MembershipType: groupdomain.GroupMembershipManual, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	rec := &eventRecorder{}
	artifacts := idmmemory.NewCSVArtifactStore()
	exporter := userusecases.UserCSVExporter{
		Deps: userusecases.UserCSVExportDeps{
			UserRepo: users, SchemaReader: userusecases.TenantUserCSVSchemaReader{}, Artifacts: artifacts,
		},
		Policy: idmdomain.DefaultCSVTransferPolicy(),
	}
	groupExporter := groupusecases.GroupCSVExporter{
		Deps:   groupusecases.GroupCSVExportDeps{GroupRepo: groups, Artifacts: artifacts},
		Policy: idmdomain.DefaultCSVTransferPolicy(),
	}
	jobRepo := jobsmodule.NewJobRepository()
	// 一覧の窓の例は 200 件を超えるエクスポートを作るので、active_jobs の上限を既定より広げる。
	quotas := tenancy.NewMemoryModule().QuotaRepo
	activeJobs := 1000
	if err := quotas.SetQuota(ctx, "acme", &tenancydomain.TenantQuota{ActiveJobs: &activeJobs}); err != nil {
		t.Fatal(err)
	}
	deps := idmusecases.DataExportDeps{
		UserRepo: users, GroupRepo: groups, JobRepo: jobRepo,
		UserCSVExporter: exporter, GroupCSVExporter: groupExporter, CSVArtifacts: artifacts,
		Emit: rec.emit, Now: func() time.Time { return now },
		Enqueuer:    jobsmodule.NewEnqueuer(jobRepo, quotas, func(event spec.DomainEvent) { _ = rec.emit(event) }),
		JobCanceler: jobsmodule.NewCanceler(jobRepo, quotas, func(event spec.DomainEvent) { _ = rec.emit(event) }),
	}
	return deps, rec, quotas
}

// 取り消しが保存先の取り消しを直接呼ぶと、Job は終わっても active_jobs の枠が戻らず、
// エクスポートを取り消すたびにテナントの投入できる数が減っていく。
//
//spec:covers REQ-IDMANAGEMENT-041: 取り消しは active_jobs を一つ減らして JobCanceled と DataExportCanceled を発行し、終了したエクスポートの取り消しはどちらも変えない。
func TestCancelDataExport_ReleasesTheActiveJobsQuota(t *testing.T) {
	deps, rec, quotas := seededExportDepsWithQuotas(t)
	ctx := exportTestCtx()
	userScope := idmusecases.ExportScope{Target: idmdomain.ExportTargetUser}
	view, err := idmusecases.StartDataExport(ctx, deps, "admin", "user", []string{"preferred_username"}, nil, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	assertActiveJobs(t, quotas, 1)
	rec.events = nil

	canceled, err := idmusecases.CancelDataExport(ctx, deps, userScope, "admin", view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if canceled.Status != idmdomain.ExportStatusCanceled {
		t.Fatalf("status=%q, want canceled", canceled.Status)
	}
	assertActiveJobs(t, quotas, 0)
	if got := rec.types(); !slices.Equal(got, []string{"JobCanceled", "DataExportCanceled"}) {
		t.Fatalf("events=%v, want [JobCanceled DataExportCanceled]", got)
	}

	// 拒否が使用量を変えないことは、0 で止まる減算と区別できるよう、使用量を残して確かめる。
	if err := quotas.CheckAndIncrement(ctx, "acme", tenancydomain.ResourceActiveJobs, 1); err != nil {
		t.Fatal(err)
	}
	rec.events = nil
	if _, err := idmusecases.CancelDataExport(ctx, deps, userScope, "admin", view.ID); !errors.Is(err, jobsports.ErrJobAlreadyTerminal) {
		t.Fatalf("second cancel err=%v, want ErrJobAlreadyTerminal", err)
	}
	assertActiveJobs(t, quotas, 1)
	if got := rec.types(); len(got) != 0 {
		t.Fatalf("refused cancel emitted %v", got)
	}
}

func assertActiveJobs(t *testing.T, quotas tenantports.QuotaRepository, want int) {
	t.Helper()
	usage, err := quotas.GetUsage(exportTestCtx(), "acme")
	if err != nil {
		t.Fatal(err)
	}
	if usage.ActiveJobs != want {
		t.Fatalf("active_jobs usage = %d, want %d", usage.ActiveJobs, want)
	}
}

// TestStartDataExport_RejectsInvalidColumns: scenario extension
// "allowlist 外の key" → InvalidRequestError 相当 (ErrInvalidExportColumns)。
func TestStartDataExport_RejectsInvalidColumns(t *testing.T) {
	deps, _ := seededExportDeps(t)
	_, err := idmusecases.StartDataExport(exportTestCtx(), deps, "admin", "user", []string{"preferred_username", "password_hash"}, nil, time.Now().UTC())
	if !errors.Is(err, idmdomain.ErrInvalidExportColumns) {
		t.Fatalf("got %v, want ErrInvalidExportColumns", err)
	}
}

// TestStartDataExport_EnqueuesQueuedJob: scenario main_success step 1-2。
func TestStartDataExport_EnqueuesQueuedJob(t *testing.T) {
	deps, rec := seededExportDeps(t)
	view, err := idmusecases.StartDataExport(exportTestCtx(), deps, "admin", "user", []string{"preferred_username", "email"}, nil, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != idmdomain.ExportStatusQueued {
		t.Errorf("status=%q, want queued", view.Status)
	}
	if view.ID == "" || view.RequestedBy != "admin" || view.Target != "user" {
		t.Errorf("unexpected view: %+v", view)
	}
	if got := rec.types(); !slices.Equal(got, []string{"JobEnqueued", "DataExportRequested"}) {
		t.Errorf("events=%v, want [JobEnqueued DataExportRequested]", got)
	}
}

// TestDataExportHandler_User_GeneratesInjectionSafeCSV: scenario main_success
// (succeeded + downloadable + counts) と formula injection extension。
func TestDataExportHandler_User_GeneratesInjectionSafeCSV(t *testing.T) {
	deps, rec := seededExportDeps(t)
	params := mustMarshal(t, idmusecases.DataExportParams{Target: "user", Columns: []string{"preferred_username", "email", "status"}, ActorUserID: "admin"})
	job := &jobsdomain.Job{ID: "exp-1", TenantID: "acme", Kind: idmusecases.KindDataExport, Params: params}
	raw, err := idmusecases.DataExportHandler(deps)(context.Background(), job)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	var result idmusecases.DataExportResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if result.TotalRows != 2 || result.ByteSize == 0 || result.ArtifactRef == "" || result.SHA256 == "" {
		t.Fatalf("unexpected result: %+v", result)
	}
	reader, _, err := deps.CSVArtifacts.OpenCSVArtifact(context.Background(), "acme", result.ArtifactRef)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	csvBytes, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	records, err := csv.NewReader(strings.NewReader(string(csvBytes))).ReadAll()
	if err != nil {
		t.Fatalf("output not RFC4180: %v", err)
	}
	if len(records) != 3 || records[0][0] != "preferred_username" {
		t.Fatalf("unexpected records: %+v", records)
	}
	// The =cmd|calc username must be neutralized after round-trip.
	for _, rec := range records[1:] {
		if rec[0] != "" && (rec[0][0] == '=' || rec[0][0] == '+' || rec[0][0] == '-' || rec[0][0] == '@') {
			t.Errorf("formula injection leaked: %q", rec[0])
		}
	}
	// No password_hash value should appear anywhere in the output.
	if strings.Contains(string(csvBytes), "secret-hash") {
		t.Error("sensitive value leaked into export")
	}
	if types := rec.types(); len(types) < 2 || types[0] != "DataExportStarted" || types[len(types)-1] != "DataExportSucceeded" {
		t.Errorf("events=%v, want DataExportStarted..DataExportSucceeded", types)
	}
}

// TestDataExportHandler_User_StatusFilter: scenario "現在のフィルタで
// エクスポート" — status フィルタで対象行を絞る。
func TestDataExportHandler_User_StatusFilter(t *testing.T) {
	deps, _ := seededExportDeps(t)
	params := mustMarshal(t, idmusecases.DataExportParams{Target: "user", Columns: []string{"preferred_username"}, Filter: map[string]string{"status": "Disabled"}, ActorUserID: "admin"})
	raw, err := idmusecases.DataExportHandler(deps)(context.Background(), &jobsdomain.Job{ID: "exp-2", TenantID: "acme", Kind: idmusecases.KindDataExport, Params: params})
	if err != nil {
		t.Fatal(err)
	}
	var result idmusecases.DataExportResult
	_ = json.Unmarshal(raw, &result)
	if result.TotalRows != 1 {
		t.Errorf("status filter: total_rows=%d, want 1", result.TotalRows)
	}
}

// TestStartDataExport_RejectsUnknownFilter: 未定義フィルタキーは fail-closed。
func TestStartDataExport_RejectsUnknownFilter(t *testing.T) {
	deps, _ := seededExportDeps(t)
	_, err := idmusecases.StartDataExport(exportTestCtx(), deps, "admin", "user", []string{"preferred_username"}, map[string]string{"password": "x"}, time.Now().UTC())
	if !errors.Is(err, idmusecases.ErrInvalidExportFilter) {
		t.Fatalf("got %v, want ErrInvalidExportFilter", err)
	}
}

// TestListAndGetDataExport: List は tenant の data_export のみを返し、
// Get はダウンロード可否を含むビューを返す。
func TestListAndGetDataExport(t *testing.T) {
	deps, _ := seededExportDeps(t)
	ctx := exportTestCtx()
	groupScope := idmusecases.ExportScope{Target: idmdomain.ExportTargetGroup}
	view, err := idmusecases.StartDataExport(ctx, deps, "admin", "group", []string{"name", "roles"}, nil, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	list, err := idmusecases.ListDataExports(ctx, deps, groupScope)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != view.ID {
		t.Fatalf("list=%+v", list)
	}
	got, err := idmusecases.GetDataExport(ctx, deps, groupScope, view.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != idmdomain.ExportStatusQueued || got.Downloadable {
		t.Errorf("queued export must not be downloadable: %+v", got)
	}
	// A group export must not resolve under the user scope (per-type isolation).
	if _, err := idmusecases.GetDataExport(ctx, deps, idmusecases.ExportScope{Target: idmdomain.ExportTargetUser}, view.ID); !errors.Is(err, idmusecases.ErrExportNotFound) {
		t.Errorf("cross-target get: got %v, want ErrExportNotFound", err)
	}
	// The user-scoped list must not include the group export.
	userList, err := idmusecases.ListDataExports(ctx, deps, idmusecases.ExportScope{Target: idmdomain.ExportTargetUser})
	if err != nil {
		t.Fatal(err)
	}
	if len(userList) != 0 {
		t.Errorf("user-scoped list leaked a group export: %+v", userList)
	}
}

// TestStartDataExport_MemberRequiresGroup: メンバーエクスポートは group_id 必須
// (per-group、Entra/Okta/Google と同様)。
func TestStartDataExport_MemberRequiresGroup(t *testing.T) {
	deps, _ := seededExportDeps(t)
	_, err := idmusecases.StartDataExport(exportTestCtx(), deps, "admin", "group_membership", []string{"user_id"}, nil, time.Now().UTC())
	if !errors.Is(err, idmusecases.ErrInvalidExportFilter) {
		t.Fatalf("member export without group_id: got %v, want ErrInvalidExportFilter", err)
	}
	// With a group_id it succeeds and is scoped to that group.
	view, err := idmusecases.StartDataExport(exportTestCtx(), deps, "admin", "group_membership", []string{"user_id"}, map[string]string{"group_id": "g1"}, time.Now().UTC())
	if err != nil {
		t.Fatalf("member export with group_id: %v", err)
	}
	// A different group's member scope must not resolve this export.
	if _, err := idmusecases.GetDataExport(exportTestCtx(), deps, idmusecases.ExportScope{Target: idmdomain.ExportTargetGroupMembership, GroupID: "g2"}, view.ID); !errors.Is(err, idmusecases.ErrExportNotFound) {
		t.Errorf("cross-group member get: got %v, want ErrExportNotFound", err)
	}
}

// TestDownloadDataExport_OnlySucceeded: succeeded かつ未期限のジョブのみ
// ダウンロードでき、それ以外は拒否される (DataExportDownloaded を発行)。
func TestDownloadDataExport_OnlySucceeded(t *testing.T) {
	deps, rec := seededExportDeps(t)
	ctx := exportTestCtx()
	userScope := idmusecases.ExportScope{Target: idmdomain.ExportTargetUser}
	view, err := idmusecases.StartDataExport(ctx, deps, "admin", "user", []string{"preferred_username"}, nil, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	// queued export: download must be rejected.
	if _, err := idmusecases.DownloadDataExport(ctx, deps, userScope, "admin", view.ID); !errors.Is(err, idmusecases.ErrExportNotDownloadable) {
		t.Fatalf("queued download: got %v, want ErrExportNotDownloadable", err)
	}
	// Drive the job to succeeded, then download.
	job, err := deps.JobRepo.Get(ctx, view.ID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := idmusecases.DataExportHandler(deps)(context.Background(), job)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := deps.JobRepo.ClaimBatch(ctx, "w1", jobsdomain.LaneBulk, 10, time.Minute, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, err := deps.JobRepo.Complete(ctx, view.ID, "w1", raw, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	rec.mu.Lock()
	rec.events = nil
	rec.mu.Unlock()
	file, err := idmusecases.DownloadDataExport(ctx, deps, userScope, "admin", view.ID)
	if err != nil {
		t.Fatalf("succeeded download: %v", err)
	}
	if file.ContentType != "text/csv; charset=utf-8" || file.ByteSize == 0 || !strings.HasSuffix(file.Filename, ".csv") {
		t.Errorf("unexpected file: %+v", file)
	}
	if got := rec.types(); len(got) != 1 || got[0] != "DataExportDownloaded" {
		t.Errorf("events=%v, want [DataExportDownloaded]", got)
	}
}

// scenario REQ-IDMANAGEMENT-027: Group export は User と同じ不変成果物へ書き出し、
// ジョブ結果に CSV 本文も base64 も残さない。`lifecycle_action` は全行で空である。
func TestDataExportHandler_GroupExportWritesAnImmutableArtifact(t *testing.T) {
	deps, _ := seededExportDeps(t)
	ctx := exportTestCtx()
	schema, err := groupdomain.NewGroupCSVSchema(nil)
	if err != nil {
		t.Fatal(err)
	}
	columns := schema.ColumnKeys()
	view, err := idmusecases.StartDataExport(ctx, deps, "admin", "group", columns, nil, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	job, err := deps.JobRepo.Get(ctx, view.ID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := idmusecases.DataExportHandler(deps)(ctx, job)
	if err != nil {
		t.Fatal(err)
	}
	var result idmusecases.DataExportResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if result.ArtifactRef == "" || result.SHA256 == "" || result.TotalRows != 1 {
		t.Fatalf("result = %+v, want an artifact reference, a digest, and the row count", result)
	}

	reader, artifact, err := deps.CSVArtifacts.OpenCSVArtifact(ctx, "acme", result.ArtifactRef)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	if artifact.SHA256 != result.SHA256 {
		t.Fatalf("digest = %q, want %q", artifact.SHA256, result.SHA256)
	}
	payload, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	document := string(payload)
	header, rows, _ := strings.Cut(document, "\n")
	if !strings.Contains(header, "lifecycle_action") {
		t.Fatalf("header = %q, want the import-compatible columns", header)
	}
	if strings.Contains(rows, "delete") {
		t.Fatalf("an export row carries an intent to delete: %q", rows)
	}
}

// scenario REQ-IDMANAGEMENT-027: Group の許可一覧は import 互換列とテナントが
// 宣言した `custom:<key>` に閉じる。宣言していないキーは開始時点で拒否する。
func TestStartDataExport_RejectsColumnsOutsideTheGroupAllowList(t *testing.T) {
	deps, _ := seededExportDeps(t)
	// テナントが属性を宣言していないので、custom 列はどれも許可されない。
	for _, column := range []string{"attributes", "attr:cost_center", "custom:cost_center", "password_hash"} {
		_, err := idmusecases.StartDataExport(exportTestCtx(), deps, "admin", "group", []string{"name", column}, nil, time.Now().UTC())
		if !errors.Is(err, idmdomain.ErrInvalidExportColumns) {
			t.Fatalf("column %q: got %v, want ErrInvalidExportColumns", column, err)
		}
	}
	// `email` は import 互換列なので受理する。
	if _, err := idmusecases.StartDataExport(exportTestCtx(), deps, "admin", "group", []string{"name", "email"}, nil, time.Now().UTC()); err != nil {
		t.Fatalf("email is an import-compatible column: %v", err)
	}
}

// transferLimitExportDeps は、各対象が 2 行を出し、行の一つが longValue を持つ世界を、
// policy で組み立てたエクスポーターで用意する。
func transferLimitExportDeps(t *testing.T, policy idmdomain.CSVTransferPolicy, longValue string) (idmusecases.DataExportDeps, *eventRecorder) {
	t.Helper()
	ctx := exportTestCtx()
	now := time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC)
	users := usermemory.NewUserRepository()
	groups := groupmemory.NewGroupRepository()
	for _, user := range []*userdomain.User{
		{ID: "u1", TenantID: "acme", PreferredUsername: "alice", Roles: []string{}, Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now},
		{ID: "u2", TenantID: "acme", PreferredUsername: longValue, Roles: []string{}, Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now},
	} {
		if err := users.Save(ctx, user); err != nil {
			t.Fatal(err)
		}
	}
	for _, group := range []*groupdomain.Group{
		{ID: "g1", TenantID: "acme", Name: "engineering", Roles: []string{}, MembershipType: groupdomain.GroupMembershipManual, CreatedAt: now, UpdatedAt: now},
		{ID: "g2", TenantID: "acme", Name: longValue, Roles: []string{}, MembershipType: groupdomain.GroupMembershipManual, CreatedAt: now, UpdatedAt: now},
	} {
		if err := groups.Save(ctx, group); err != nil {
			t.Fatal(err)
		}
	}
	for _, userID := range []string{"u1", "u2"} {
		if _, err := groups.AddMember(ctx, &groupdomain.GroupMember{GroupID: "g1", UserID: userID, Source: groupdomain.MembershipSourceManual, CreatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	artifacts := idmmemory.NewCSVArtifactStore()
	rec := &eventRecorder{}
	return idmusecases.DataExportDeps{
		UserRepo: users, GroupRepo: groups, CSVArtifacts: artifacts,
		UserCSVExporter: userusecases.UserCSVExporter{
			Deps:   userusecases.UserCSVExportDeps{UserRepo: users, SchemaReader: userusecases.TenantUserCSVSchemaReader{}, Artifacts: artifacts},
			Policy: policy,
		},
		GroupCSVExporter: groupusecases.GroupCSVExporter{
			Deps:   groupusecases.GroupCSVExportDeps{GroupRepo: groups, Artifacts: artifacts},
			Policy: policy,
		},
		GroupMembershipCSVExporter: groupusecases.GroupMembershipCSVExporter{
			Deps:   groupusecases.GroupMembershipCSVExportDeps{GroupRepo: groups, UserRepo: users, Artifacts: artifacts},
			Policy: policy,
		},
		Emit: rec.emit, Now: func() time.Time { return now },
	}, rec
}

//spec:covers REQ-IDMANAGEMENT-007, REQ-IDMANAGEMENT-027, REQ-IDMANAGEMENT-030, EX-IDMANAGEMENT-007-03, EX-IDMANAGEMENT-027-03, EX-IDMANAGEMENT-030-03: 行数、成果物の大きさ、項目の大きさのどの上限を超えても、User、Group、メンバーシップのエクスポートのジョブが csv_transfer_limit_exceeded で失敗し、DataExportFailed が同じコードを運び、結果を返さないこと。
func TestDataExportHandler_ReportsEveryTransferLimitAsCSVTransferLimitExceeded(t *testing.T) {
	// 見出しの最長のキー（dynamic_rule_expression の 23 バイト）は項目の上限に収め、値だけが超えるようにする。
	const maxFieldBytes = 32
	longValue := strings.Repeat("x", maxFieldBytes+1)
	unlimited := idmdomain.DefaultCSVTransferPolicy()
	groupSchema, err := groupdomain.NewGroupCSVSchema(nil)
	if err != nil {
		t.Fatal(err)
	}
	targets := []struct {
		target  string
		columns []string
		filter  map[string]string
	}{
		{target: "user", columns: []string{"preferred_username"}},
		{target: "group", columns: groupSchema.ColumnKeys()},
		{target: "group_membership", columns: groupdomain.NewGroupMembershipCSVSchema().ColumnKeys(), filter: map[string]string{"group_id": "g1"}},
	}
	limits := []struct {
		name   string
		policy idmdomain.CSVTransferPolicy
	}{
		{name: "rows", policy: idmdomain.CSVTransferPolicy{MaxRows: 1, MaxBytes: unlimited.MaxBytes, MaxFieldBytes: unlimited.MaxFieldBytes}},
		{name: "bytes", policy: idmdomain.CSVTransferPolicy{MaxRows: unlimited.MaxRows, MaxBytes: 16, MaxFieldBytes: unlimited.MaxFieldBytes}},
		{name: "field", policy: idmdomain.CSVTransferPolicy{MaxRows: unlimited.MaxRows, MaxBytes: unlimited.MaxBytes, MaxFieldBytes: maxFieldBytes}},
	}
	for _, target := range targets {
		for _, limit := range limits {
			t.Run(target.target+"/"+limit.name, func(t *testing.T) {
				deps, rec := transferLimitExportDeps(t, limit.policy, longValue)
				params := mustMarshal(t, idmusecases.DataExportParams{Target: target.target, Columns: target.columns, Filter: target.filter, ActorUserID: "admin"})
				job := &jobsdomain.Job{ID: "exp-limit", TenantID: "acme", Kind: idmusecases.KindDataExport, Params: params}

				raw, err := idmusecases.DataExportHandler(deps)(exportTestCtx(), job)

				if err == nil || err.Error() != "csv_transfer_limit_exceeded" {
					t.Fatalf("job error = %v, want csv_transfer_limit_exceeded", err)
				}
				if raw != nil {
					t.Fatalf("result = %s, want none", raw)
				}
				failed, ok := rec.events[len(rec.events)-1].(*idmdomain.DataExportFailed)
				if !ok || failed.ErrorCode != "csv_transfer_limit_exceeded" {
					t.Fatalf("last event = %#v, want DataExportFailed with csv_transfer_limit_exceeded", rec.events[len(rec.events)-1])
				}
			})
		}
	}
}

// 対照：同じ世界を既定のポリシーで生成すると、どの対象も成功する。上の失敗が上限だけによることを確かめる。
func TestDataExportHandler_TransferLimitWorldSucceedsUnderTheDefaultPolicy(t *testing.T) {
	longValue := strings.Repeat("x", 33)
	groupSchema, err := groupdomain.NewGroupCSVSchema(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, params := range []idmusecases.DataExportParams{
		{Target: "user", Columns: []string{"preferred_username"}, ActorUserID: "admin"},
		{Target: "group", Columns: groupSchema.ColumnKeys(), ActorUserID: "admin"},
		{Target: "group_membership", Columns: groupdomain.NewGroupMembershipCSVSchema().ColumnKeys(), Filter: map[string]string{"group_id": "g1"}, ActorUserID: "admin"},
	} {
		deps, _ := transferLimitExportDeps(t, idmdomain.DefaultCSVTransferPolicy(), longValue)
		job := &jobsdomain.Job{ID: "exp-control", TenantID: "acme", Kind: idmusecases.KindDataExport, Params: mustMarshal(t, params)}
		raw, err := idmusecases.DataExportHandler(deps)(exportTestCtx(), job)
		if err != nil {
			t.Fatalf("%s: %v", params.Target, err)
		}
		var result idmusecases.DataExportResult
		if err := json.Unmarshal(raw, &result); err != nil {
			t.Fatal(err)
		}
		if result.TotalRows != 2 {
			t.Fatalf("%s: rows = %d, want 2", params.Target, result.TotalRows)
		}
	}
}
