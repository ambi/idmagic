package usecases_test

// 管理者向け Application 操作 (Update / Delete / Icon / Binding / Assignment) の
// 未カバー分岐 (エラーパス・冪等・検証失敗) を補う (wi-129)。

import (
	"context"
	"errors"
	"testing"
	"time"

	appusecases "github.com/ambi/idmagic/internal/application/usecases"
	"github.com/ambi/idmagic/internal/shared/adapters/persistence/memory"
	"github.com/ambi/idmagic/internal/shared/spec"
)

func fullAppDeps() appusecases.ApplicationDeps {
	return appusecases.ApplicationDeps{
		Repo:           memory.NewApplicationRepository(),
		IconStore:      memory.NewApplicationIconStore(),
		AssignmentRepo: memory.NewApplicationAssignmentRepository(),
		PolicyRepo:     memory.NewSignInPolicyRepository(),
	}
}

func seedApp(ctx context.Context, t *testing.T, deps appusecases.ApplicationDeps, name string) *spec.Application {
	t.Helper()
	app, err := appusecases.CreateApplication(ctx, deps, appusecases.CreateApplicationInput{
		ActorUserID: "admin", Name: name, Kind: spec.ApplicationFederated,
	})
	if err != nil {
		t.Fatalf("seed app: %v", err)
	}
	return app
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
	disabled := spec.ApplicationDisabled
	got, err := appusecases.UpdateApplication(ctx, deps, appusecases.UpdateApplicationInput{
		ActorUserID: "admin", ApplicationID: app.ApplicationID, Status: &disabled,
	})
	if err != nil {
		t.Fatalf("update status: %v", err)
	}
	if got.Status != spec.ApplicationDisabled {
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
		SubjectType: spec.AssignmentSubjectUser, SubjectID: "alice",
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
	deps := appusecases.ApplicationDeps{Repo: memory.NewApplicationRepository()}
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

func TestDetachBinding(t *testing.T) {
	ctx := tenantContext()
	deps := fullAppDeps()
	app := seedApp(ctx, t, deps, "CRM")

	if _, err := appusecases.AttachBinding(ctx, deps, appusecases.AttachBindingInput{
		ActorUserID: "admin", ApplicationID: app.ApplicationID,
		Binding: spec.ProtocolBinding{Type: spec.ProtocolBindingOIDC, ClientID: "c1"},
	}); err != nil {
		t.Fatalf("attach: %v", err)
	}
	if err := appusecases.DetachBinding(ctx, deps, "admin", app.ApplicationID, spec.ProtocolBindingOIDC, time.Time{}); err != nil {
		t.Fatalf("detach: %v", err)
	}
	got, _ := deps.Repo.FindByID(ctx, "acme", app.ApplicationID)
	if len(got.Bindings) != 0 {
		t.Fatalf("binding not detached: %+v", got.Bindings)
	}
	if err := appusecases.DetachBinding(ctx, deps, "admin", "ghost", spec.ProtocolBindingOIDC, time.Time{}); !errors.Is(err, appusecases.ErrApplicationNotFound) {
		t.Fatalf("expected ErrApplicationNotFound, got %v", err)
	}
}

func TestAssignmentErrorPathsAndListing(t *testing.T) {
	ctx := tenantContext()
	deps := fullAppDeps()
	app := seedApp(ctx, t, deps, "Payroll")
	assignDeps := appusecases.AssignmentDeps{Repo: deps.Repo, AssignmentRepo: deps.AssignmentRepo}

	// 不明なアプリ。
	if _, err := appusecases.AssignApplication(ctx, assignDeps, appusecases.AssignApplicationInput{
		ActorUserID: "admin", ApplicationID: "ghost", SubjectType: spec.AssignmentSubjectUser, SubjectID: "alice",
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
		ActorUserID: "admin", ApplicationID: app.ApplicationID, SubjectType: spec.AssignmentSubjectUser, SubjectID: "  ",
	}); !errors.Is(err, appusecases.ErrSubjectRequired) {
		t.Fatalf("expected ErrSubjectRequired, got %v", err)
	}
	// 不正な visibility。
	if _, err := appusecases.AssignApplication(ctx, assignDeps, appusecases.AssignApplicationInput{
		ActorUserID: "admin", ApplicationID: app.ApplicationID, SubjectType: spec.AssignmentSubjectUser,
		SubjectID: "alice", Visibility: "weird",
	}); !errors.Is(err, appusecases.ErrInvalidVisibility) {
		t.Fatalf("expected ErrInvalidVisibility, got %v", err)
	}

	// 正常割当のあと ListAssignments で確認。
	if _, err := appusecases.AssignApplication(ctx, assignDeps, appusecases.AssignApplicationInput{
		ActorUserID: "admin", ApplicationID: app.ApplicationID, SubjectType: spec.AssignmentSubjectUser, SubjectID: "alice",
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
		spec.AssignmentSubjectUser, "alice", time.Time{}); err != nil {
		t.Fatalf("unassign: %v", err)
	}
	list, _ = appusecases.ListAssignments(ctx, assignDeps, app.ApplicationID)
	if len(list) != 0 {
		t.Fatalf("unassign left %d assignments", len(list))
	}
}
