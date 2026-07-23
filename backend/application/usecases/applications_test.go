package usecases_test

import (
	"context"
	"testing"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	appmemory "github.com/ambi/idmagic/backend/application/db_memory"
	"github.com/ambi/idmagic/backend/application/domain"
	"github.com/ambi/idmagic/backend/application/ports"
	appusecases "github.com/ambi/idmagic/backend/application/usecases"
	"github.com/ambi/idmagic/backend/tenancy"
)

func tenantContext() context.Context {
	return tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: "acme"}, "https://idp.example", "")
}

func newDeps() (appusecases.ApplicationDeps, appusecases.AssignmentDeps) {
	apps := appmemory.NewApplicationRepository()
	assignments := appmemory.NewApplicationAssignmentRepository()
	appDeps := appusecases.ApplicationDeps{Repo: apps, AssignmentRepo: assignments}
	assignDeps := appusecases.AssignmentDeps{Repo: apps, AssignmentRepo: assignments}
	return appDeps, assignDeps
}

func TestCreateAndListMyApplicationsRespectsAssignmentAndVisibility(t *testing.T) {
	ctx := tenantContext()
	appDeps, assignDeps := newDeps()

	app, err := appusecases.CreateApplication(ctx, appDeps, appusecases.CreateApplicationInput{
		ActorUserID: "admin", Name: "Payroll", Kind: domain.ApplicationFederated, Protocol: &domain.ApplicationProtocol{Type: domain.ApplicationProtocolOIDC, ClientID: "test-client"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	userSubjects := []ports.SubjectRef{{Type: domain.AssignmentSubjectUser, ID: "alice"}}

	// 未割当はポータルに出ず、割当ゲートも閉じる。
	mine, err := appusecases.ListMyApplications(ctx, assignDeps, userSubjects)
	if err != nil {
		t.Fatalf("list mine (unassigned): %v", err)
	}
	if len(mine) != 0 {
		t.Fatalf("unassigned user should see no apps, got %d", len(mine))
	}
	assigned, err := appusecases.IsSubjectAssigned(ctx, assignDeps.AssignmentRepo, "acme", app.ApplicationID, userSubjects)
	if err != nil || assigned {
		t.Fatalf("unassigned subject must fail the gate: assigned=%v err=%v", assigned, err)
	}

	// 割当後はポータルに出て、ゲートが開く。
	if _, err := appusecases.AssignApplication(ctx, assignDeps, appusecases.AssignApplicationInput{
		ActorUserID: "admin", ApplicationID: app.ApplicationID, SubjectType: domain.AssignmentSubjectUser, SubjectID: "alice",
	}); err != nil {
		t.Fatalf("assign: %v", err)
	}
	mine, err = appusecases.ListMyApplications(ctx, assignDeps, userSubjects)
	if err != nil || len(mine) != 1 {
		t.Fatalf("assigned user should see 1 app, got %d err=%v", len(mine), err)
	}
	assigned, err = appusecases.IsSubjectAssigned(ctx, assignDeps.AssignmentRepo, "acme", app.ApplicationID, userSubjects)
	if err != nil || !assigned {
		t.Fatalf("assigned subject must pass the gate: assigned=%v err=%v", assigned, err)
	}

	// hidden 割当はポータルから消えるが、ゲートは開いたまま。
	if _, err := appusecases.AssignApplication(ctx, assignDeps, appusecases.AssignApplicationInput{
		ActorUserID: "admin", ApplicationID: app.ApplicationID, SubjectType: domain.AssignmentSubjectUser, SubjectID: "alice",
		Visibility: domain.AssignmentHidden,
	}); err != nil {
		t.Fatalf("assign hidden: %v", err)
	}
	mine, err = appusecases.ListMyApplications(ctx, assignDeps, userSubjects)
	if err != nil || len(mine) != 0 {
		t.Fatalf("hidden assignment should hide app from portal, got %d err=%v", len(mine), err)
	}
	assigned, err = appusecases.IsSubjectAssigned(ctx, assignDeps.AssignmentRepo, "acme", app.ApplicationID, userSubjects)
	if err != nil || !assigned {
		t.Fatalf("hidden assignment must still pass the gate: assigned=%v err=%v", assigned, err)
	}
}

func TestWeblinkRequiresLaunchURLAndRejectsProtocol(t *testing.T) {
	ctx := tenantContext()
	appDeps, _ := newDeps()

	if _, err := appusecases.CreateApplication(ctx, appDeps, appusecases.CreateApplicationInput{
		ActorUserID: "admin", Name: "Wiki", Kind: domain.ApplicationWeblink,
	}); err == nil {
		t.Fatal("weblink without launch_url must be rejected")
	}

	if _, err := appusecases.CreateApplication(ctx, appDeps, appusecases.CreateApplicationInput{
		ActorUserID: "admin", Name: "Wiki", Kind: domain.ApplicationWeblink,
		LaunchURL: "https://wiki.example",
		Protocol:  &domain.ApplicationProtocol{Type: domain.ApplicationProtocolOIDC, ClientID: "c1"},
	}); err == nil {
		t.Fatal("weblink must not accept protocol")
	}
}
