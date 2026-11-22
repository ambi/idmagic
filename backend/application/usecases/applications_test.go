package usecases_test

// 主要ユースケース追跡: REQ-APPLICATION-012。

import (
	"context"
	"testing"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	appmemory "github.com/ambi/idmagic/backend/application/internal/db_memory"
	appports "github.com/ambi/idmagic/backend/application/ports"
	appusecases "github.com/ambi/idmagic/backend/application/usecases"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

func tenantContext() context.Context {
	return tenantports.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "acme"}, "https://idp.example", "")
}

func newDeps() (appusecases.ApplicationDeps, appusecases.AssignmentDeps) {
	apps := appmemory.NewApplicationRepository()
	assignments := appmemory.NewApplicationAssignmentRepository()
	appDeps := appusecases.ApplicationDeps{Repo: apps, AssignmentRepo: assignments}
	assignDeps := appusecases.AssignmentDeps{
		Repo: apps, AssignmentRepo: assignments, SubjectDirectory: existingSubjectDirectoryFake{},
	}
	return appDeps, assignDeps
}

func TestCreateAndListMyApplicationsRespectsAssignmentAndVisibility(t *testing.T) {
	ctx := tenantContext()
	appDeps, assignDeps := newDeps()

	app, err := appusecases.CreateApplication(ctx, appDeps, appusecases.CreateApplicationInput{
		ActorUserID: "admin", Name: "Payroll", Kind: appdomain.ApplicationFederated, Protocol: &appdomain.ApplicationProtocol{Type: appdomain.ApplicationProtocolOIDC, ClientID: "test-client"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	userSubjects := []appports.SubjectRef{{Type: appdomain.AssignmentSubjectUser, ID: "alice"}}

	// 未割当はポータルに出ず、割当ゲートも閉じる。
	mine, err := appusecases.ListMyApplications(ctx, assignDeps, userSubjects)
	if err != nil {
		t.Fatalf("list mine (unassigned): %v", err)
	}
	if len(mine) != 0 {
		t.Fatalf("unassigned user should see no apps, got %d", len(mine))
	}
	assigned, err := appusecases.IsSubjectAssigned(ctx, assignDeps.AssignmentRepo, "acme", app.ID, userSubjects)
	if err != nil || assigned {
		t.Fatalf("unassigned subject must fail the gate: assigned=%v err=%v", assigned, err)
	}

	// 割当後はポータルに出て、ゲートが開く。
	if _, err := appusecases.AssignApplication(ctx, assignDeps, appusecases.AssignApplicationInput{
		ActorUserID: "admin", ApplicationID: app.ID, SubjectType: appdomain.AssignmentSubjectUser, SubjectID: "alice",
	}); err != nil {
		t.Fatalf("assign: %v", err)
	}
	mine, err = appusecases.ListMyApplications(ctx, assignDeps, userSubjects)
	if err != nil || len(mine) != 1 {
		t.Fatalf("assigned user should see 1 app, got %d err=%v", len(mine), err)
	}
	assigned, err = appusecases.IsSubjectAssigned(ctx, assignDeps.AssignmentRepo, "acme", app.ID, userSubjects)
	if err != nil || !assigned {
		t.Fatalf("assigned subject must pass the gate: assigned=%v err=%v", assigned, err)
	}

	// hidden 割当はポータルから消えるが、ゲートは開いたまま。
	if _, err := appusecases.AssignApplication(ctx, assignDeps, appusecases.AssignApplicationInput{
		ActorUserID: "admin", ApplicationID: app.ID, SubjectType: appdomain.AssignmentSubjectUser, SubjectID: "alice",
		Visibility: appdomain.AssignmentHidden,
	}); err != nil {
		t.Fatalf("assign hidden: %v", err)
	}
	mine, err = appusecases.ListMyApplications(ctx, assignDeps, userSubjects)
	if err != nil || len(mine) != 0 {
		t.Fatalf("hidden assignment should hide app from portal, got %d err=%v", len(mine), err)
	}
	assigned, err = appusecases.IsSubjectAssigned(ctx, assignDeps.AssignmentRepo, "acme", app.ID, userSubjects)
	if err != nil || !assigned {
		t.Fatalf("hidden assignment must still pass the gate: assigned=%v err=%v", assigned, err)
	}
}

func TestWeblinkRequiresLaunchURLAndRejectsProtocol(t *testing.T) {
	ctx := tenantContext()
	appDeps, _ := newDeps()

	if _, err := appusecases.CreateApplication(ctx, appDeps, appusecases.CreateApplicationInput{
		ActorUserID: "admin", Name: "Wiki", Kind: appdomain.ApplicationWeblink,
	}); err == nil {
		t.Fatal("weblink without launch_url must be rejected")
	}

	if _, err := appusecases.CreateApplication(ctx, appDeps, appusecases.CreateApplicationInput{
		ActorUserID: "admin", Name: "Wiki", Kind: appdomain.ApplicationWeblink,
		LaunchURL: "https://wiki.example",
		Protocol:  &appdomain.ApplicationProtocol{Type: appdomain.ApplicationProtocolOIDC, ClientID: "c1"},
	}); err == nil {
		t.Fatal("weblink must not accept protocol")
	}
}
