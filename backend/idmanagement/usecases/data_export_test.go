package usecases_test

import (
	"context"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	idmusecases "github.com/ambi/idmagic/backend/idmanagement/usecases"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	jobsmemory "github.com/ambi/idmagic/backend/jobs/db_memory"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
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
	return tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "acme"}, "", "")
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
	deps := idmusecases.DataExportDeps{
		UserRepo:  users,
		GroupRepo: groups,
		JobRepo:   jobsmemory.NewJobRepository(),
		Emit:      rec.emit,
		Now:       func() time.Time { return now },
	}
	return deps, rec
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
	if got := rec.types(); len(got) != 1 || got[0] != "DataExportRequested" {
		t.Errorf("events=%v, want [DataExportRequested]", got)
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
	if result.TotalRows != 2 || result.ByteSize == 0 || result.CSVBase64 == "" {
		t.Fatalf("unexpected result: %+v", result)
	}
	csvBytes, err := base64.StdEncoding.DecodeString(result.CSVBase64)
	if err != nil {
		t.Fatal(err)
	}
	records, err := csv.NewReader(strings.NewReader(string(csvBytes))).ReadAll()
	if err != nil {
		t.Fatalf("output not RFC4180: %v", err)
	}
	if len(records) != 3 || records[0][0] != "Preferred username" {
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
