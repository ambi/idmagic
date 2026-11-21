package idgovernance_test

import (
	"context"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idgovernance"
	"github.com/ambi/idmagic/backend/idmanagement"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	notification "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

//spec:covers REQ-IDGOVERNANCE-020, REQ-IDGOVERNANCE-021, REQ-IDGOVERNANCE-022, REQ-IDGOVERNANCE-023: 維持、却下、未回答を同じ回に置き、期限後に失効だけが所属を外し、再実行しても戻さない。
func TestAccessReviewAppliesDecisionsAndDefaults(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
	idm := idmanagement.NewMemoryModule()
	for _, id := range []string{"reviewer", "fallback", "retain", "revoke", "unanswered"} {
		if err := idm.UserRepo.Save(ctx, &userdomain.User{ID: id, TenantID: "tenant-a", PreferredUsername: id, CreatedAt: now, UpdatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	if err := idm.GroupRepo.Save(ctx, &groupdomain.Group{ID: "group", TenantID: "tenant-a", Name: "Engineering", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"retain", "revoke", "unanswered"} {
		if _, err := idm.GroupRepo.AddMember(ctx, &groupdomain.GroupMember{GroupID: "group", UserID: id, CreatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	m := idgovernance.NewMemoryModule(idm.UserRepo)
	d := idgovernance.AccessReviewDeps{UserRepo: idm.UserRepo, GroupRepo: idm.GroupRepo, GroupMembership: idm.GroupMembershipCommands(func(spec.DomainEvent) error { return nil }, "access-review"), Emit: func(spec.DomainEvent) {}}
	p, err := m.CreateAccessReviewPlan(ctx, d, "tenant-a", "admin", idgovernance.AccessReviewPlanInput{Name: "Quarterly", TargetKind: "group_membership", ResourceIDs: []string{"group"}, ReviewerID: "reviewer", FallbackReviewerID: "fallback", DefaultDecision: "revoke", StartsAt: now, DurationDays: 1, Enabled: true}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.AdvanceAccessReviews(ctx, d, now); err != nil {
		t.Fatal(err)
	}
	reviews, err := m.AccessReviews.ListByPlan(ctx, "tenant-a", p.ID)
	if err != nil || len(reviews) != 1 || len(reviews[0].Items) != 3 {
		t.Fatalf("reviews=%+v error=%v", reviews, err)
	}
	r := reviews[0]
	for _, item := range r.Items {
		decision := map[string]string{"retain": "retain", "revoke": "revoke"}[item.SubjectID]
		if decision != "" {
			if _, err := m.DecideAccessReview(ctx, d, "tenant-a", r.ID, item.ID, "reviewer", decision, now); err != nil {
				t.Fatal(err)
			}
		}
	}
	for range 2 {
		if err := m.AdvanceAccessReviews(ctx, d, now.Add(24*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	members, err := idm.GroupRepo.ListMembersByGroup(ctx, "tenant-a", "group")
	if err != nil || len(members) != 1 || members[0].UserID != "retain" {
		t.Fatalf("members=%+v error=%v", members, err)
	}
	reviews, err = m.AccessReviews.ListByPlan(ctx, "tenant-a", p.ID)
	if err != nil || len(reviews) != 1 || reviews[0].Status != "completed" {
		t.Fatalf("result=%+v error=%v", reviews, err)
	}
	for _, item := range reviews[0].Items {
		if item.Outcome == "" || (item.SubjectID == "unanswered" && (!item.Defaulted || item.DecidedBy != "")) {
			t.Fatalf("result item=%+v", item)
		}
	}
}

//spec:covers REQ-IDGOVERNANCE-020, REQ-IDGOVERNANCE-025: 動的、上流管理、別テナントの対象と無効な担当者を API 側でも拒否し、計画と監査を残さない。
func TestAccessReviewRejectsNonlocalAndAutomaticTargets(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
	idm := idmanagement.NewMemoryModule()
	for _, fixture := range []struct {
		id, tenant string
		disabled   bool
	}{{"reviewer", "tenant-a", false}, {"inactive", "tenant-a", true}, {"foreign", "tenant-b", false}} {
		u := &userdomain.User{ID: fixture.id, TenantID: fixture.tenant, PreferredUsername: fixture.id, CreatedAt: now, UpdatedAt: now}
		if fixture.disabled {
			u.Lifecycle.Status = idmdomain.UserStatusDisabled
		}
		if err := idm.UserRepo.Save(ctx, u); err != nil {
			t.Fatal(err)
		}
	}
	for _, fixture := range []struct {
		id, tenant        string
		dynamic, upstream bool
	}{{"manual", "tenant-a", false, false}, {"dynamic", "tenant-a", true, false}, {"upstream", "tenant-a", false, true}, {"foreign", "tenant-b", false, false}} {
		g := &groupdomain.Group{ID: fixture.id, TenantID: fixture.tenant, Name: fixture.id, UpstreamManaged: fixture.upstream, CreatedAt: now, UpdatedAt: now}
		if fixture.dynamic {
			g.MembershipType = groupdomain.GroupMembershipDynamic
		}
		if err := idm.GroupRepo.Save(ctx, g); err != nil {
			t.Fatal(err)
		}
	}
	m := idgovernance.NewMemoryModule(idm.UserRepo)
	events := 0
	deps := idgovernance.AccessReviewDeps{UserRepo: idm.UserRepo, GroupRepo: idm.GroupRepo, Emit: func(spec.DomainEvent) { events++ }}
	base := idgovernance.AccessReviewPlanInput{Name: "Manual membership", TargetKind: "group_membership", ResourceIDs: []string{"manual"}, ReviewerID: "reviewer", FallbackReviewerID: "reviewer", DefaultDecision: "retain", StartsAt: now, DurationDays: 1, Enabled: true}
	for _, fixture := range []struct{ resource, reviewer, fallback string }{{"dynamic", "reviewer", "reviewer"}, {"upstream", "reviewer", "reviewer"}, {"foreign", "reviewer", "reviewer"}, {"manual", "inactive", "reviewer"}, {"manual", "reviewer", "foreign"}} {
		input := base
		input.ResourceIDs = []string{fixture.resource}
		input.ReviewerID = fixture.reviewer
		input.FallbackReviewerID = fixture.fallback
		if _, err := m.CreateAccessReviewPlan(ctx, deps, "tenant-a", "admin", input, now); err == nil {
			t.Errorf("accepted invalid references: %+v", fixture)
		}
	}
	if plans, err := m.AccessReviewPlans.ListPlans(ctx, "tenant-a"); err != nil || len(plans) != 0 || events != 0 {
		t.Fatalf("refusal changed plans=%+v events=%d err=%v", plans, events, err)
	}
	if _, err := m.CreateAccessReviewPlan(ctx, deps, "tenant-a", "admin", base, now); err != nil {
		t.Fatal(err)
	}
	if events != 1 {
		t.Fatalf("control event count=%d", events)
	}
}

type accessReviewNotifier struct {
	notify func(notification.Notification) bool
}

func (n accessReviewNotifier) Notify(_ context.Context, input notification.Notification) bool {
	return n.notify(input)
}

//spec:covers REQ-IDGOVERNANCE-024: 前の通知で時計が進んでも、次の通知は取得した瞬間から有効な占有を持つ。
func TestAccessReviewNotificationLeaseUsesCurrentClock(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
	clock := now
	idm := idmanagement.NewMemoryModule()
	for _, id := range []string{"one", "two"} {
		email := id + "@example.test"
		if err := idm.UserRepo.Save(ctx, &userdomain.User{ID: id, TenantID: "tenant-a", PreferredUsername: id, Email: &email, CreatedAt: now, UpdatedAt: now}); err != nil {
			t.Fatal(err)
		}
		if err := idm.AgentRepo.Save(ctx, &agentdomain.Agent{ID: id, TenantID: "tenant-a", Name: id, Kind: idmdomain.AgentKindAutonomous, OwnerUserID: id, Status: idmdomain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	m := idgovernance.NewMemoryModule(idm.UserRepo)
	sent := 0
	deps := idgovernance.AccessReviewDeps{UserRepo: idm.UserRepo, AgentRepo: idm.AgentRepo, Now: func() time.Time { return clock }, Emit: func(spec.DomainEvent) {}}
	deps.Notifier = accessReviewNotifier{notify: func(notification.Notification) bool {
		reviews, err := m.AccessReviews.ActiveReviews(ctx)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, review := range reviews {
			for _, lease := range review.WorkLeases {
				found = true
				if !lease.Until.After(clock) {
					t.Errorf("new lease expired: until=%v now=%v", lease.Until, clock)
				}
			}
		}
		if !found {
			t.Error("no notification lease")
		}
		sent++
		clock = clock.Add(3 * time.Minute)
		return true
	}}
	_, err := m.CreateAccessReviewPlan(ctx, deps, "tenant-a", "admin", idgovernance.AccessReviewPlanInput{Name: "Lease timing", TargetKind: "agent", ResourceIDs: []string{"one", "two"}, FallbackReviewerID: "one", DefaultDecision: "retain", StartsAt: now, DurationDays: 1, Enabled: true}, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.AdvanceAccessReviews(ctx, deps, now); err != nil {
		t.Fatal(err)
	}
	if sent != 2 {
		t.Fatalf("sent=%d want 2", sent)
	}
}

//spec:covers REQ-IDGOVERNANCE-020, REQ-IDGOVERNANCE-021, REQ-IDGOVERNANCE-024: 所有者不在は明示した代替を使い、変更は次回に反映する。通知の失敗は回答後も残り、停止は参照先の停止に妨げられない。
func TestAccessReviewFallbackScheduleAndNotificationFailure(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
	idm := idmanagement.NewMemoryModule()
	for _, id := range []string{"fallback", "owner"} {
		email := id + "@example.test"
		if err := idm.UserRepo.Save(ctx, &userdomain.User{ID: id, TenantID: "tenant-a", PreferredUsername: id, Email: &email, CreatedAt: now, UpdatedAt: now}); err != nil {
			t.Fatal(err)
		}
	}
	if err := idm.AgentRepo.Save(ctx, &agentdomain.Agent{ID: "bot", TenantID: "tenant-a", Name: "Bot", Kind: idmdomain.AgentKindAutonomous, OwnerUserID: "missing", Status: idmdomain.AgentStatusActive, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	m := idgovernance.NewMemoryModule(idm.UserRepo)
	d := idgovernance.AccessReviewDeps{UserRepo: idm.UserRepo, AgentRepo: idm.AgentRepo, Emit: func(spec.DomainEvent) {}}
	d.Notifier = accessReviewNotifier{notify: func(n notification.Notification) bool {
		// 外向きの送信中も、別の利用者の参照が保存先のロックに阻まれない。
		if _, err := m.AccessReviewPlans.ListPlans(ctx, "tenant-a"); err != nil {
			t.Error(err)
		}
		if n.To != "fallback@example.test" && n.To != "owner@example.test" {
			t.Errorf("recipient=%s", n.To)
		}
		return false
	}}
	input := idgovernance.AccessReviewPlanInput{Name: "Owner review", TargetKind: "agent", ResourceIDs: []string{"bot"}, FallbackReviewerID: "fallback", DefaultDecision: "retain", StartsAt: now, DurationDays: 1, Enabled: true}
	plan, err := m.CreateAccessReviewPlan(ctx, d, "tenant-a", "admin", input, now)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- m.AdvanceAccessReviews(ctx, d, now) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("notification held the repository lock")
	}
	reviews, err := m.AccessReviews.ListByPlan(ctx, "tenant-a", plan.ID)
	if err != nil || len(reviews) != 1 {
		t.Fatalf("reviews=%+v err=%v", reviews, err)
	}
	if reviews[0].Items[0].ReviewerID != "fallback" || !reviews[0].NotificationFailed {
		t.Fatalf("fallback review=%+v", reviews[0])
	}
	if _, err := m.DecideAccessReview(ctx, d, "tenant-a", reviews[0].ID, reviews[0].Items[0].ID, "fallback", "retain", now); err != nil {
		t.Fatal(err)
	}
	if _, err := idm.AgentRepo.Update(ctx, "tenant-a", "bot", func(a *agentdomain.Agent) (bool, error) { a.OwnerUserID = "owner"; return true, nil }); err != nil {
		t.Fatal(err)
	}
	if err := m.AdvanceAccessReviews(ctx, d, now.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	reviews, err = m.AccessReviews.ListByPlan(ctx, "tenant-a", plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reviews[0].Items[0].ReviewerID != "fallback" || !reviews[0].NotificationFailed || reviews[0].Status != "completed" {
		t.Fatalf("completed review=%+v", reviews[0])
	}
	input.RecurrenceDays = 30
	plan, err = m.UpdateAccessReviewPlan(ctx, d, "tenant-a", "admin", plan.ID, plan.Revision, input, now.Add(24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	want := now.Add(30 * 24 * time.Hour)
	if plan.NextStartAt == nil || !plan.NextStartAt.Equal(want) {
		t.Fatalf("next=%v want=%v", plan.NextStartAt, want)
	}
	if err := m.AdvanceAccessReviews(ctx, d, want); err != nil {
		t.Fatal(err)
	}
	reviews, err = m.AccessReviews.ListByPlan(ctx, "tenant-a", plan.ID)
	if err != nil || len(reviews) != 2 || reviews[0].Items[0].ReviewerID != "owner" {
		t.Fatalf("next review=%+v err=%v", reviews, err)
	}
	u, err := idm.UserRepo.FindBySub(ctx, "fallback")
	if err != nil {
		t.Fatal(err)
	}
	u.Lifecycle.Status = idmdomain.UserStatusDisabled
	if err := idm.UserRepo.Save(ctx, u); err != nil {
		t.Fatal(err)
	}
	input.Enabled = false
	plan, err = m.UpdateAccessReviewPlan(ctx, d, "tenant-a", "admin", plan.ID, plan.Revision, input, want)
	if err != nil || plan.Enabled {
		t.Fatalf("disable plan=%+v err=%v", plan, err)
	}
}
