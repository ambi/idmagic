package usecases_test

// 管理者向け Application 操作 (Update / Delete / Icon / Binding / Assignment) の
// 未カバー分岐 (エラーパス・冪等・検証失敗) を補う (wi-129)。

import (
	"context"
	"errors"
	"testing"
	"time"

	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	"github.com/ambi/idmagic/backend/application/domain"
	"github.com/ambi/idmagic/backend/application/ports"
	appusecases "github.com/ambi/idmagic/backend/application/usecases"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func fullAppDeps() appusecases.ApplicationDeps {
	return appusecases.ApplicationDeps{
		Repo:           appmemory.NewApplicationRepository(),
		IconStore:      appmemory.NewApplicationIconStore(),
		AssignmentRepo: appmemory.NewApplicationAssignmentRepository(),
		PolicyRepo:     appmemory.NewSignInPolicyRepository(),
		QuotaRepo:      tenancymemory.NewQuotaRepository(),
	}
}

func seedApp(ctx context.Context, t *testing.T, deps appusecases.ApplicationDeps, name string) *domain.Application {
	t.Helper()
	app, err := appusecases.CreateApplication(ctx, deps, appusecases.CreateApplicationInput{
		ActorUserID: "admin", Name: name, Kind: domain.ApplicationFederated, Protocol: &domain.ApplicationProtocol{Type: domain.ApplicationProtocolOIDC, ClientID: "test-client"},
	})
	if err != nil {
		t.Fatalf("seed app: %v", err)
	}
	return app
}

// TestCreateApplication_rejectsWhenHardQuotaExceeded is a wi-160 T004.4 RED
// test for the SCL scenario "Hard Quota を超過したリソース作成は拒否される"
// (spec/contexts/tenancy.yaml), applied to the applications resource.
func TestCreateApplication_rejectsWhenHardQuotaExceeded(t *testing.T) {
	ctx := tenantContext()
	deps := fullAppDeps()
	limit := 1
	if err := deps.QuotaRepo.SetQuota(ctx, "acme", &tenancydomain.TenantQuota{Applications: &limit}); err != nil {
		t.Fatalf("SetQuota: %v", err)
	}
	seedApp(ctx, t, deps, "app-one")
	_, err := appusecases.CreateApplication(ctx, deps, appusecases.CreateApplicationInput{
		ActorUserID: "admin", Name: "app-two", Kind: domain.ApplicationFederated, Protocol: &domain.ApplicationProtocol{Type: domain.ApplicationProtocolOIDC, ClientID: "test-client"},
	})
	var qErr *tenancydomain.QuotaExceededError
	if !errors.As(err, &qErr) {
		t.Fatalf("expected *domain.QuotaExceededError, got %v", err)
	}
	if qErr.Resource != tenancydomain.ResourceApplications {
		t.Fatalf("unexpected resource: %s", qErr.Resource)
	}
}

// TestDeleteApplication_decrementsQuotaUsage is a wi-160 T004.4 RED test:
// deleting an application must free its quota slot so a subsequent create at
// the same limit succeeds.
func TestDeleteApplication_decrementsQuotaUsage(t *testing.T) {
	ctx := tenantContext()
	deps := fullAppDeps()
	limit := 1
	if err := deps.QuotaRepo.SetQuota(ctx, "acme", &tenancydomain.TenantQuota{Applications: &limit}); err != nil {
		t.Fatalf("SetQuota: %v", err)
	}
	app := seedApp(ctx, t, deps, "app-one")
	if err := appusecases.DeleteApplication(ctx, deps, "admin", app.ApplicationID, time.Time{}); err != nil {
		t.Fatal(err)
	}
	if _, err := appusecases.CreateApplication(ctx, deps, appusecases.CreateApplicationInput{
		ActorUserID: "admin", Name: "app-two", Kind: domain.ApplicationFederated, Protocol: &domain.ApplicationProtocol{Type: domain.ApplicationProtocolOIDC, ClientID: "test-client"},
	}); err != nil {
		t.Fatalf("expected create to succeed after delete freed quota, got %v", err)
	}
}

func TestUpdateApplicationChangesAndNoop(t *testing.T) {
	ctx := tenantContext()
	deps := fullAppDeps()
	app := seedApp(ctx, t, deps, "Payroll")

	// name を変更すると changed が立ち保存される。
	newName := "Payroll v2"
	updated, err := appusecases.UpdateApplication(ctx, deps, appusecases.UpdateApplicationInput{
		ActorUserID: "admin", ApplicationID: app.ApplicationID, Name: &newName,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Name != "Payroll v2" {
		t.Fatalf("name not updated: %q", updated.Name)
	}

	// 同じ値なら no-op (changed 無し) で updatedAt は据え置き。
	same := "Payroll v2"
	noop, err := appusecases.UpdateApplication(ctx, deps, appusecases.UpdateApplicationInput{
		ActorUserID: "admin", ApplicationID: app.ApplicationID, Name: &same,
	})
	if err != nil {
		t.Fatalf("noop update: %v", err)
	}
	if !noop.UpdatedAt.Equal(updated.UpdatedAt) {
		t.Fatalf("noop update should not bump updatedAt")
	}

	// status も更新できる。
	disabled := domain.ApplicationDisabled
	got, err := appusecases.UpdateApplication(ctx, deps, appusecases.UpdateApplicationInput{
		ActorUserID: "admin", ApplicationID: app.ApplicationID, Status: &disabled,
	})
	if err != nil {
		t.Fatalf("update status: %v", err)
	}
	if got.Status != domain.ApplicationDisabled {
		t.Fatalf("status not updated: %v", got.Status)
	}
}

func TestUpdateApplicationNotFound(t *testing.T) {
	ctx := tenantContext()
	deps := fullAppDeps()
	name := "x"
	if _, err := appusecases.UpdateApplication(ctx, deps, appusecases.UpdateApplicationInput{
		ActorUserID: "admin", ApplicationID: "ghost", Name: &name,
	}); !errors.Is(err, appusecases.ErrApplicationNotFound) {
		t.Fatalf("expected ErrApplicationNotFound, got %v", err)
	}
}

func TestDeleteApplicationRemovesAssignmentsAndPolicy(t *testing.T) {
	ctx := tenantContext()
	deps := fullAppDeps()
	app := seedApp(ctx, t, deps, "Payroll")

	assignDeps := appusecases.AssignmentDeps{Repo: deps.Repo, AssignmentRepo: deps.AssignmentRepo}
	if _, err := appusecases.AssignApplication(ctx, assignDeps, appusecases.AssignApplicationInput{
		ActorUserID: "admin", ApplicationID: app.ApplicationID,
		SubjectType: domain.AssignmentSubjectUser, SubjectID: "alice",
	}); err != nil {
		t.Fatalf("assign: %v", err)
	}

	if err := appusecases.DeleteApplication(ctx, deps, "admin", app.ApplicationID, time.Time{}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if got, _ := deps.Repo.FindByID(ctx, "acme", app.ApplicationID); got != nil {
		t.Fatalf("app not deleted: %+v", got)
	}
	remaining, _ := deps.AssignmentRepo.ListByApplication(ctx, "acme", app.ApplicationID)
	if len(remaining) != 0 {
		t.Fatalf("assignments not cascaded: %+v", remaining)
	}

	// 二重削除は not found。
	if err := appusecases.DeleteApplication(ctx, deps, "admin", app.ApplicationID, time.Time{}); !errors.Is(err, appusecases.ErrApplicationNotFound) {
		t.Fatalf("expected ErrApplicationNotFound on re-delete, got %v", err)
	}
}

func TestUploadAndDeleteApplicationIcon(t *testing.T) {
	ctx := tenantContext()
	deps := fullAppDeps()
	app := seedApp(ctx, t, deps, "Payroll")

	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}
	updated, err := appusecases.UploadApplicationIcon(ctx, deps, appusecases.UploadApplicationIconInput{
		ActorUserID: "admin", ApplicationID: app.ApplicationID, Data: png, IconURL: "/icons/x.png",
	})
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if updated.IconObjectKey == "" || updated.IconURL != "/icons/x.png" {
		t.Fatalf("icon fields not set: %+v", updated)
	}

	// フォーマット不正は拒否する。
	if _, err := appusecases.UploadApplicationIcon(ctx, deps, appusecases.UploadApplicationIconInput{
		ActorUserID: "admin", ApplicationID: app.ApplicationID, Data: []byte("not-an-image"),
	}); !errors.Is(err, appusecases.ErrApplicationIconFormat) {
		t.Fatalf("expected ErrApplicationIconFormat, got %v", err)
	}

	// 存在しないアプリへのアップロードは not found。
	if _, err := appusecases.UploadApplicationIcon(ctx, deps, appusecases.UploadApplicationIconInput{
		ActorUserID: "admin", ApplicationID: "ghost", Data: png,
	}); !errors.Is(err, appusecases.ErrApplicationNotFound) {
		t.Fatalf("expected ErrApplicationNotFound, got %v", err)
	}

	cleared, err := appusecases.DeleteApplicationIcon(ctx, deps, "admin", app.ApplicationID, time.Time{})
	if err != nil {
		t.Fatalf("delete icon: %v", err)
	}
	if cleared.IconObjectKey != "" || cleared.IconURL != "" {
		t.Fatalf("icon not cleared: %+v", cleared)
	}
	if _, err := appusecases.DeleteApplicationIcon(ctx, deps, "admin", "ghost", time.Time{}); !errors.Is(err, appusecases.ErrApplicationNotFound) {
		t.Fatalf("expected ErrApplicationNotFound on delete icon, got %v", err)
	}
}

func TestIconStoreNotConfigured(t *testing.T) {
	ctx := tenantContext()
	deps := appusecases.ApplicationDeps{Repo: appmemory.NewApplicationRepository()}
	if _, err := appusecases.UploadApplicationIcon(ctx, deps, appusecases.UploadApplicationIconInput{
		ApplicationID: "x", Data: []byte{0x89},
	}); err == nil {
		t.Fatal("expected error when icon store is nil (upload)")
	}
	if _, err := appusecases.DeleteApplicationIcon(ctx, deps, "admin", "x", time.Time{}); err == nil {
		t.Fatal("expected error when icon store is nil (delete)")
	}
}

func TestDetectApplicationIconContentType(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}
	jpeg := []byte{0xff, 0xd8, 0xff, 0, 0, 0}
	webp := append([]byte("RIFF0000WEBP"), 0, 0)
	gif := []byte("GIF89a\x00\x00")
	cases := []struct {
		name string
		data []byte
		want string
	}{
		{"png", png, "image/png"},
		{"jpeg", jpeg, "image/jpeg"},
		{"webp", webp, "image/webp"},
		{"gif", gif, "image/gif"},
	}
	for _, tc := range cases {
		got, err := appusecases.DetectApplicationIconContentType(tc.data)
		if err != nil || got != tc.want {
			t.Fatalf("%s: got %q err=%v, want %q", tc.name, got, err, tc.want)
		}
	}

	if _, err := appusecases.DetectApplicationIconContentType(nil); !errors.Is(err, appusecases.ErrApplicationIconRequired) {
		t.Fatalf("empty data want ErrApplicationIconRequired, got %v", err)
	}
	big := make([]byte, appusecases.MaxApplicationIconBytes+1)
	if _, err := appusecases.DetectApplicationIconContentType(big); !errors.Is(err, appusecases.ErrApplicationIconTooLarge) {
		t.Fatalf("oversize want ErrApplicationIconTooLarge, got %v", err)
	}
	if _, err := appusecases.DetectApplicationIconContentType([]byte("plain text data")); !errors.Is(err, appusecases.ErrApplicationIconFormat) {
		t.Fatalf("unknown format want ErrApplicationIconFormat, got %v", err)
	}
}

func TestAssignmentErrorPathsAndListing(t *testing.T) {
	ctx := tenantContext()
	deps := fullAppDeps()
	app := seedApp(ctx, t, deps, "Payroll")
	assignDeps := appusecases.AssignmentDeps{Repo: deps.Repo, AssignmentRepo: deps.AssignmentRepo}

	// 不明なアプリ。
	if _, err := appusecases.AssignApplication(ctx, assignDeps, appusecases.AssignApplicationInput{
		ActorUserID: "admin", ApplicationID: "ghost", SubjectType: domain.AssignmentSubjectUser, SubjectID: "alice",
	}); !errors.Is(err, appusecases.ErrApplicationNotFound) {
		t.Fatalf("expected ErrApplicationNotFound, got %v", err)
	}
	// 不正な subject type。
	if _, err := appusecases.AssignApplication(ctx, assignDeps, appusecases.AssignApplicationInput{
		ActorUserID: "admin", ApplicationID: app.ApplicationID, SubjectType: "nope", SubjectID: "alice",
	}); !errors.Is(err, appusecases.ErrInvalidSubjectType) {
		t.Fatalf("expected ErrInvalidSubjectType, got %v", err)
	}
	// 空 subject id。
	if _, err := appusecases.AssignApplication(ctx, assignDeps, appusecases.AssignApplicationInput{
		ActorUserID: "admin", ApplicationID: app.ApplicationID, SubjectType: domain.AssignmentSubjectUser, SubjectID: "  ",
	}); !errors.Is(err, appusecases.ErrSubjectRequired) {
		t.Fatalf("expected ErrSubjectRequired, got %v", err)
	}
	// 不正な visibility。
	if _, err := appusecases.AssignApplication(ctx, assignDeps, appusecases.AssignApplicationInput{
		ActorUserID: "admin", ApplicationID: app.ApplicationID, SubjectType: domain.AssignmentSubjectUser,
		SubjectID: "alice", Visibility: "weird",
	}); !errors.Is(err, appusecases.ErrInvalidVisibility) {
		t.Fatalf("expected ErrInvalidVisibility, got %v", err)
	}

	// 正常割当のあと ListAssignments で確認。
	if _, err := appusecases.AssignApplication(ctx, assignDeps, appusecases.AssignApplicationInput{
		ActorUserID: "admin", ApplicationID: app.ApplicationID, SubjectType: domain.AssignmentSubjectUser, SubjectID: "alice",
	}); err != nil {
		t.Fatalf("assign: %v", err)
	}
	list, err := appusecases.ListAssignments(ctx, assignDeps, app.ApplicationID)
	if err != nil || len(list) != 1 {
		t.Fatalf("list assignments = %d err=%v", len(list), err)
	}
	if _, err := appusecases.ListAssignments(ctx, assignDeps, "ghost"); !errors.Is(err, appusecases.ErrApplicationNotFound) {
		t.Fatalf("expected ErrApplicationNotFound, got %v", err)
	}

	// Unassign。
	if err := appusecases.UnassignApplication(ctx, assignDeps, "admin", app.ApplicationID,
		domain.AssignmentSubjectUser, "alice", time.Time{}); err != nil {
		t.Fatalf("unassign: %v", err)
	}
	list, _ = appusecases.ListAssignments(ctx, assignDeps, app.ApplicationID)
	if len(list) != 0 {
		t.Fatalf("unassign left %d assignments", len(list))
	}
}

func TestDeleteApplicationErrors(t *testing.T) {
	ctx := tenantContext()
	deps := fullAppDeps()

	// Delete non-existent app
	err := appusecases.DeleteApplication(ctx, deps, "admin", "ghost", time.Time{})
	if !errors.Is(err, appusecases.ErrApplicationNotFound) {
		t.Fatalf("expected ErrApplicationNotFound, got %v", err)
	}
}

func TestCreateApplicationWithNilEmit(t *testing.T) {
	ctx := tenantContext()
	deps := appusecases.ApplicationDeps{
		Repo: appmemory.NewApplicationRepository(),
		Emit: nil,
	}
	_, err := appusecases.CreateApplication(ctx, deps, appusecases.CreateApplicationInput{
		ActorUserID: "admin", Name: "No Emit App", Kind: domain.ApplicationWeblink, LaunchURL: "https://example.com",
	})
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
}

func TestCreateApplicationWithEmit(t *testing.T) {
	ctx := tenantContext()
	var emitted bool
	deps := appusecases.ApplicationDeps{
		Repo: appmemory.NewApplicationRepository(),
		Emit: func(event spec.DomainEvent) {
			emitted = true
		},
	}
	_, err := appusecases.CreateApplication(ctx, deps, appusecases.CreateApplicationInput{
		ActorUserID: "admin", Name: "Emit App", Kind: domain.ApplicationWeblink, LaunchURL: "https://example.com",
	})
	if err != nil {
		t.Fatalf("failed: %v", err)
	}
	if !emitted {
		t.Fatalf("expected event to be emitted")
	}
}

func TestUpdateApplicationValidationErrors(t *testing.T) {
	ctx := tenantContext()
	deps := fullAppDeps()
	app := seedApp(ctx, t, deps, "CRM")

	// Update to empty name (validation error)
	emptyName := "  "
	_, err := appusecases.UpdateApplication(ctx, deps, appusecases.UpdateApplicationInput{
		ActorUserID: "admin", ApplicationID: app.ApplicationID, Name: &emptyName,
	})
	if err == nil {
		t.Fatalf("expected validation error for empty name")
	}
}

func TestUploadApplicationIconWithEmptyObjectKey(t *testing.T) {
	ctx := tenantContext()
	deps := fullAppDeps()
	app := seedApp(ctx, t, deps, "CRM")

	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}
	updated, err := appusecases.UploadApplicationIcon(ctx, deps, appusecases.UploadApplicationIconInput{
		ActorUserID: "admin", ApplicationID: app.ApplicationID, ObjectKey: "", Data: png, IconURL: "https://example.com/icon.png",
	})
	if err != nil {
		t.Fatalf("failed to upload icon: %v", err)
	}
	if updated.IconObjectKey == "" {
		t.Fatalf("expected generated UUID for empty ObjectKey")
	}
}

func TestApplicationIconUsecasesNilStoreErrors(t *testing.T) {
	ctx := tenantContext()
	deps := fullAppDeps()
	deps.IconStore = nil // Set to nil to test the error paths
	app := seedApp(ctx, t, deps, "CRM")

	// Upload icon with nil store
	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}
	_, err := appusecases.UploadApplicationIcon(ctx, deps, appusecases.UploadApplicationIconInput{
		ActorUserID: "admin", ApplicationID: app.ApplicationID, Data: png,
	})
	if err == nil || err.Error() != "application icon store is not configured" {
		t.Fatalf("expected store not configured error, got %v", err)
	}

	// Delete icon with nil store
	_, err = appusecases.DeleteApplicationIcon(ctx, deps, "admin", app.ApplicationID, time.Time{})
	if err == nil || err.Error() != "application icon store is not configured" {
		t.Fatalf("expected store not configured error, got %v", err)
	}
}

func TestListMyApplicationsFiltering(t *testing.T) {
	ctx := tenantContext()
	deps := fullAppDeps()

	// 1. Service App (should be excluded)
	serviceApp, _ := appusecases.CreateApplication(ctx, deps, appusecases.CreateApplicationInput{
		ActorUserID: "admin", Name: "Service App", Kind: domain.ApplicationService,
		Protocol: &domain.ApplicationProtocol{Type: domain.ApplicationProtocolOIDC, ClientID: "service-client"},
	})
	_, _ = appusecases.AssignApplication(ctx, appusecases.AssignmentDeps{Repo: deps.Repo, AssignmentRepo: deps.AssignmentRepo}, appusecases.AssignApplicationInput{
		ActorUserID: "admin", ApplicationID: serviceApp.ApplicationID, SubjectType: domain.AssignmentSubjectUser, SubjectID: "alice",
	})

	// 2. Disabled App (should be excluded)
	disabledApp, _ := appusecases.CreateApplication(ctx, deps, appusecases.CreateApplicationInput{
		ActorUserID: "admin", Name: "Disabled App", Kind: domain.ApplicationFederated, Protocol: &domain.ApplicationProtocol{Type: domain.ApplicationProtocolOIDC, ClientID: "test-client"},
	})
	disabled := domain.ApplicationDisabled
	_, _ = appusecases.UpdateApplication(ctx, deps, appusecases.UpdateApplicationInput{
		ActorUserID: "admin", ApplicationID: disabledApp.ApplicationID, Status: &disabled,
	})
	_, _ = appusecases.AssignApplication(ctx, appusecases.AssignmentDeps{Repo: deps.Repo, AssignmentRepo: deps.AssignmentRepo}, appusecases.AssignApplicationInput{
		ActorUserID: "admin", ApplicationID: disabledApp.ApplicationID, SubjectType: domain.AssignmentSubjectUser, SubjectID: "alice",
	})

	// 3. Normal App with multiple assignments (should be deduped)
	normalApp, _ := appusecases.CreateApplication(ctx, deps, appusecases.CreateApplicationInput{
		ActorUserID: "admin", Name: "Normal App", Kind: domain.ApplicationFederated, Protocol: &domain.ApplicationProtocol{Type: domain.ApplicationProtocolOIDC, ClientID: "test-client"},
	})
	// assign to user "alice"
	_, _ = appusecases.AssignApplication(ctx, appusecases.AssignmentDeps{Repo: deps.Repo, AssignmentRepo: deps.AssignmentRepo}, appusecases.AssignApplicationInput{
		ActorUserID: "admin", ApplicationID: normalApp.ApplicationID, SubjectType: domain.AssignmentSubjectUser, SubjectID: "alice",
	})
	// assign to group "alice-group"
	_, _ = appusecases.AssignApplication(ctx, appusecases.AssignmentDeps{Repo: deps.Repo, AssignmentRepo: deps.AssignmentRepo}, appusecases.AssignApplicationInput{
		ActorUserID: "admin", ApplicationID: normalApp.ApplicationID, SubjectType: domain.AssignmentSubjectGroup, SubjectID: "alice-group",
	})
	subjectsWithGroup := []ports.SubjectRef{
		{Type: domain.AssignmentSubjectUser, ID: "alice"},
		{Type: domain.AssignmentSubjectGroup, ID: "alice-group"},
	}

	apps, err := appusecases.ListMyApplications(ctx, appusecases.AssignmentDeps{Repo: deps.Repo, AssignmentRepo: deps.AssignmentRepo}, subjectsWithGroup)
	if err != nil {
		t.Fatalf("ListMyApplications: %v", err)
	}

	// Should only contain normalApp (1 item)
	if len(apps) != 1 || apps[0].ApplicationID != normalApp.ApplicationID {
		t.Fatalf("expected only 1 normal app, got: %d apps", len(apps))
	}
}

type errorAppRepo struct {
	ports.ApplicationRepository
}

func (e errorAppRepo) FindByID(ctx context.Context, tenantID, id string) (*domain.Application, error) {
	return nil, errors.New("database error")
}

func (e errorAppRepo) Save(ctx context.Context, app *domain.Application) error {
	return errors.New("database error")
}

func (e errorAppRepo) Create(ctx context.Context, app *domain.Application) error {
	return errors.New("database error")
}

func (e errorAppRepo) Delete(ctx context.Context, tenantID, id string) error {
	return errors.New("database error")
}

type errorAssignmentRepo struct {
	ports.AssignmentRepository
}

func (e errorAssignmentRepo) ListBySubjects(ctx context.Context, tenantID string, subjects []ports.SubjectRef) ([]*domain.ApplicationAssignment, error) {
	return nil, errors.New("database error")
}

func (e errorAssignmentRepo) ListByApplication(ctx context.Context, tenantID, appID string) ([]*domain.ApplicationAssignment, error) {
	return nil, errors.New("database error")
}

func (e errorAssignmentRepo) DeleteByApplication(ctx context.Context, tenantID, appID string) error {
	return errors.New("database error")
}

func (e errorAssignmentRepo) Save(ctx context.Context, a *domain.ApplicationAssignment) error {
	return errors.New("database error")
}

func TestUsecaseDatabaseErrors(t *testing.T) {
	ctx := tenantContext()
	errRepo := errorAppRepo{}
	errAssignRepo := errorAssignmentRepo{}

	// 1. CreateApplication save error
	_, err := appusecases.CreateApplication(ctx, appusecases.ApplicationDeps{Repo: errRepo}, appusecases.CreateApplicationInput{
		Name: "Test App", Kind: domain.ApplicationWeblink, LaunchURL: "https://example.com",
	})
	if err == nil || err.Error() != "database error" {
		t.Fatalf("expected database error, got %v", err)
	}

	// 2. UpdateApplication find error
	name := "New Name"
	_, err = appusecases.UpdateApplication(ctx, appusecases.ApplicationDeps{Repo: errRepo}, appusecases.UpdateApplicationInput{
		ApplicationID: "app-1", Name: &name,
	})
	if err == nil || err.Error() != "database error" {
		t.Fatalf("expected database error, got %v", err)
	}

	// 3. DeleteApplication find error
	err = appusecases.DeleteApplication(ctx, appusecases.ApplicationDeps{Repo: errRepo}, "admin", "app-1", time.Time{})
	if err == nil || err.Error() != "database error" {
		t.Fatalf("expected database error, got %v", err)
	}

	// 4. AssignApplication find error
	_, err = appusecases.AssignApplication(ctx, appusecases.AssignmentDeps{Repo: errRepo, AssignmentRepo: errAssignRepo}, appusecases.AssignApplicationInput{
		ApplicationID: "app-1", SubjectType: domain.AssignmentSubjectUser, SubjectID: "alice",
	})
	if err == nil || err.Error() != "database error" {
		t.Fatalf("expected database error, got %v", err)
	}

	// 5. ListAssignments find error
	_, err = appusecases.ListAssignments(ctx, appusecases.AssignmentDeps{Repo: errRepo, AssignmentRepo: errAssignRepo}, "app-1")
	if err == nil || err.Error() != "database error" {
		t.Fatalf("expected database error, got %v", err)
	}

	// 8. IsSubjectAssigned list error
	_, err = appusecases.IsSubjectAssigned(ctx, errAssignRepo, "acme", "app-1", []ports.SubjectRef{{Type: domain.AssignmentSubjectUser, ID: "alice"}})
	if err == nil || err.Error() != "database error" {
		t.Fatalf("expected database error, got %v", err)
	}

	// 9. ListMyApplications list error
	_, err = appusecases.ListMyApplications(ctx, appusecases.AssignmentDeps{Repo: errRepo, AssignmentRepo: errAssignRepo}, []ports.SubjectRef{{Type: domain.AssignmentSubjectUser, ID: "alice"}})
	if err == nil || err.Error() != "database error" {
		t.Fatalf("expected database error, got %v", err)
	}
}
