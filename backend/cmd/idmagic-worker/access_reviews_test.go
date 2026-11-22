package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/application"
	appdomain "github.com/ambi/idmagic/backend/application/domain"
	"github.com/ambi/idmagic/backend/idgovernance"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	"github.com/ambi/idmagic/backend/shared/notification/email_memory"
	notificationports "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/notification/template"
	"github.com/ambi/idmagic/backend/shared/spec"
)

//spec:covers REQ-IDGOVERNANCE-021, REQ-IDGOVERNANCE-023, REQ-IDGOVERNANCE-024: worker と同じ組み立てから割り当てと Agent を失効し、監査、メールと失効エポックを観測し、維持する Agent は動作を保つ。
func TestAccessReviewRuntimeRevokesAssignmentsAndAgent(t *testing.T) {
	f := newRevocationFixture(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	f.deps.Application = application.NewMemoryModule()
	app := &appdomain.Application{ID: "portal", TenantID: "tenant-a", Name: "Portal", Kind: appdomain.ApplicationWeblink, Status: appdomain.ApplicationActive, CreatedAt: now, UpdatedAt: now}
	if err := f.deps.Application.Repo.Save(ctx, app); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []appdomain.AssignmentSubjectType{appdomain.AssignmentSubjectUser, appdomain.AssignmentSubjectGroup} {
		if err := f.deps.Application.AssignmentRepo.Save(ctx, &appdomain.ApplicationAssignment{TenantID: "tenant-a", ApplicationID: app.ID, SubjectType: kind, SubjectID: "alice", Visibility: appdomain.AssignmentVisible, CreatedAt: now, UpdatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.deps.IdManagement.GroupRepo.Save(ctx, &groupdomain.Group{ID: "alice", TenantID: "tenant-a", Name: "Team", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	u, err := f.deps.IdManagement.UserRepo.FindBySub(ctx, "alice")
	if err != nil {
		t.Fatal(err)
	}
	email := "alice@example.test"
	u.Email = &email
	if err := f.deps.IdManagement.UserRepo.Save(ctx, u); err != nil {
		t.Fatal(err)
	}
	sender := &email_memory.NoopEmailSender{}
	f.deps.Notification = notificationports.Module{Notifier: &template.Notifier{Sender: sender}}
	emit := func(e spec.DomainEvent) {
		if err := f.sink.Emit(ctx, e); err != nil {
			t.Error(err)
		}
	}
	runtime := f.deps.AccessReviewDeps(emit)
	var agentPlanID string
	for _, kind := range []string{"application_assignment", "agent"} {
		resources := []string{app.ID}
		decision := "revoke"
		if kind == "agent" {
			resources = f.agentIDs
			decision = "retain"
		}
		plan, err := f.deps.IdGovernance.CreateAccessReviewPlan(ctx, runtime, "tenant-a", "admin", idgovernance.AccessReviewPlanInput{Name: kind, TargetKind: kind, ResourceIDs: resources, ReviewerID: "alice", FallbackReviewerID: "alice", DefaultDecision: decision, StartsAt: now, DurationDays: 1, Enabled: true}, now)
		if err != nil {
			t.Fatal(err)
		}
		if kind == "agent" {
			agentPlanID = plan.ID
		}
	}
	if err := f.deps.IdGovernance.AdvanceAccessReviews(ctx, runtime, now); err != nil {
		t.Fatal(err)
	}
	reviews, err := f.deps.IdGovernance.AccessReviews.ListByPlan(ctx, "tenant-a", agentPlanID)
	if err != nil || len(reviews) != 1 {
		t.Fatalf("reviews=%+v err=%v", reviews, err)
	}
	for _, item := range reviews[0].Items {
		if item.ResourceID == f.agentIDs[0] {
			if _, err := f.deps.IdGovernance.DecideAccessReview(ctx, runtime, "tenant-a", reviews[0].ID, item.ID, "alice", "revoke", now); err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(sender.Sent) != 2 {
		t.Fatalf("start notifications=%d", len(sender.Sent))
	}
	if err := f.deps.IdGovernance.AdvanceAccessReviews(ctx, runtime, now.Add(12*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if len(sender.Sent) != 4 {
		t.Fatalf("reminders=%d", len(sender.Sent))
	}
	for range 2 {
		if err := f.deps.IdGovernance.AdvanceAccessReviews(ctx, runtime, now.Add(24*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	assignments, err := f.deps.Application.AssignmentRepo.ListByApplication(ctx, "tenant-a", app.ID)
	if err != nil || len(assignments) != 0 {
		t.Fatalf("assignments=%+v err=%v", assignments, err)
	}
	for i, id := range f.agentIDs {
		a, err := f.deps.IdManagement.AgentRepo.FindByID(ctx, "tenant-a", id)
		if err != nil {
			t.Fatal(err)
		}
		epoch, err := f.epochs.FindByAgent(ctx, "tenant-a", id)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			if a.Status != idmdomain.AgentStatusDisabled || epoch == nil || !epoch.Epoch.Equal(now.Add(24*time.Hour)) {
				t.Fatalf("revoked agent=%+v epoch=%+v", a, epoch)
			}
		} else if a.Status != idmdomain.AgentStatusActive || epoch != nil {
			t.Fatalf("retained agent=%+v epoch=%+v", a, epoch)
		}
	}
	if f.sink.count("AgentDisabled") != 1 || f.sink.count("ApplicationUnassigned") != 2 {
		t.Fatalf("events=%v", f.sink.types)
	}
}

type failFirstReviewRevocation struct {
	delegate userports.AccessRevocationReactor
	failed   bool
}

func (r *failFirstReviewRevocation) React(ctx context.Context, event spec.DomainEvent) error {
	if !r.failed {
		r.failed = true
		return errors.New("injected revocation failure")
	}
	return r.delegate.React(ctx, event)
}

//spec:covers REQ-IDGOVERNANCE-023: Agent の保存後に失効が失敗しても適用済みにせず、Disabled の再実行で失効エポックまで進める。
func TestAccessReviewRetriesRevocationAfterAgentStateCommit(t *testing.T) {
	f := newRevocationFixture(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	emit := func(e spec.DomainEvent) {
		if err := f.sink.Emit(ctx, e); err != nil {
			t.Error(err)
		}
	}
	runtime := f.deps.AccessReviewDeps(emit)
	flaky := &failFirstReviewRevocation{delegate: f.deps.AgentRevocationReactor(emit)}
	runtime.Agents = f.deps.IdManagement.AgentLifecycleCommands(func(e spec.DomainEvent) error { emit(e); return nil }, flaky, "access-review")
	plan, err := f.deps.IdGovernance.CreateAccessReviewPlan(ctx, runtime, "tenant-a", "admin", idgovernance.AccessReviewPlanInput{Name: "Revocation retry", TargetKind: "agent", ResourceIDs: []string{f.agentIDs[0]}, FallbackReviewerID: "alice", DefaultDecision: "revoke", StartsAt: now, DurationDays: 1, Enabled: true}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.deps.IdGovernance.AdvanceAccessReviews(ctx, runtime, now); err != nil {
		t.Fatal(err)
	}
	deadline := now.Add(24 * time.Hour)
	if err := f.deps.IdGovernance.AdvanceAccessReviews(ctx, runtime, deadline); err != nil {
		t.Fatal(err)
	}
	a, err := f.deps.IdManagement.AgentRepo.FindByID(ctx, "tenant-a", f.agentIDs[0])
	if err != nil || a.Status != idmdomain.AgentStatusDisabled {
		t.Fatalf("agent=%+v err=%v", a, err)
	}
	if epoch, err := f.epochs.FindByAgent(ctx, "tenant-a", a.ID); err != nil || epoch != nil {
		t.Fatalf("failed epoch=%+v err=%v", epoch, err)
	}
	reviews, err := f.deps.IdGovernance.AccessReviews.ListByPlan(ctx, "tenant-a", plan.ID)
	if err != nil || len(reviews) != 1 {
		t.Fatalf("reviews=%+v err=%v", reviews, err)
	}
	if reviews[0].Status != "applying" || reviews[0].Items[0].ErrorCode != "apply_failed" {
		t.Fatalf("failure result=%+v", reviews[0])
	}
	if err := f.deps.IdGovernance.AdvanceAccessReviews(ctx, runtime, deadline.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if epoch, err := f.epochs.FindByAgent(ctx, "tenant-a", a.ID); err != nil || epoch == nil || !epoch.Epoch.Equal(deadline.Add(time.Second)) {
		t.Fatalf("retried epoch=%+v err=%v", epoch, err)
	}
	reviews, err = f.deps.IdGovernance.AccessReviews.ListByPlan(ctx, "tenant-a", plan.ID)
	if err != nil || reviews[0].Status != "completed" || reviews[0].Items[0].ErrorCode != "" {
		t.Fatalf("retry result=%+v err=%v", reviews, err)
	}
}
