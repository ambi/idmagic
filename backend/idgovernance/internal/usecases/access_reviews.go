package usecases

import (
	"context"
	"errors"
	"slices"
	"time"

	appdomain "github.com/ambi/idmagic/backend/application/domain"
	appports "github.com/ambi/idmagic/backend/application/ports"
	d "github.com/ambi/idmagic/backend/idgovernance/internal/domain"
	igports "github.com/ambi/idmagic/backend/idgovernance/internal/ports"
	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	groupports "github.com/ambi/idmagic/backend/idmanagement/group/ports"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	notification "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type AccessReviewDeps struct {
	// Now は逐次処理中に進む時計である。未設定なら引数の時刻を固定入力として使う。
	Now                    func() time.Time
	Plans                  igports.AccessReviewPlanRepository
	Reviews                igports.AccessReviewRepository
	UserRepo               userports.UserRepository
	GroupRepo              groupports.GroupRepository
	AgentRepo              agentports.AgentRepository
	ApplicationRepo        appports.ApplicationRepository
	AssignmentRepo         appports.AssignmentRepository
	GroupMembership        igports.GroupMembership
	ApplicationAssignments igports.ReviewedApplicationAssignments
	Agents                 igports.ReviewedAgents
	Notifier               notification.Notifier
	Emit                   func(spec.DomainEvent)
}

//nolint:nilnil // 不在と読取失敗を区別し、不在だけを明示した代替担当者へ回す。
func reviewUser(ctx context.Context, deps AccessReviewDeps, tenant, id string) (*userdomain.User, error) {
	u, err := deps.UserRepo.FindBySub(ctx, id)
	if err != nil {
		return nil, err
	}
	if u == nil || u.TenantID != tenant || !u.IsActive() {
		return nil, nil
	}
	return u, nil
}

func validateReviewReferences(ctx context.Context, deps AccessReviewDeps, tenant string, input d.AccessReviewPlanInput) error {
	if deps.UserRepo == nil {
		return d.ErrAccessReviewInvalid
	}
	u, err := reviewUser(ctx, deps, tenant, input.FallbackReviewerID)
	if err != nil {
		return err
	}
	if u == nil {
		return d.ErrAccessReviewInvalid
	}
	if input.TargetKind != "agent" {
		u, err = reviewUser(ctx, deps, tenant, input.ReviewerID)
		if err != nil {
			return err
		}
		if u == nil {
			return d.ErrAccessReviewInvalid
		}
	}
	for _, id := range input.ResourceIDs {
		switch input.TargetKind {
		case "group_membership":
			if deps.GroupRepo == nil {
				return d.ErrAccessReviewInvalid
			}
			g, err := deps.GroupRepo.FindByID(ctx, tenant, id)
			if err != nil {
				return err
			}
			if g == nil || g.MembershipType.Effective() != groupdomain.GroupMembershipManual || g.UpstreamManaged {
				return d.ErrAccessReviewInvalid
			}
		case "application_assignment":
			if deps.ApplicationRepo == nil || deps.AssignmentRepo == nil {
				return d.ErrAccessReviewInvalid
			}
			a, err := deps.ApplicationRepo.FindByID(ctx, tenant, id)
			if err != nil {
				return err
			}
			if a == nil {
				return d.ErrAccessReviewInvalid
			}
		case "agent":
			if deps.AgentRepo == nil {
				return d.ErrAccessReviewInvalid
			}
			a, err := deps.AgentRepo.FindByID(ctx, tenant, id)
			if err != nil {
				return err
			}
			if a == nil {
				return d.ErrAccessReviewInvalid
			}
		}
	}
	return nil
}

func CreateAccessReviewPlan(ctx context.Context, deps AccessReviewDeps, tenant, actor string, input d.AccessReviewPlanInput, now time.Time) (*d.AccessReviewPlan, error) {
	if deps.Emit == nil {
		return nil, errors.New("access review audit is not configured")
	}
	if err := input.NormalizeAndValidate(); err != nil {
		return nil, err
	}
	if err := validateReviewReferences(ctx, deps, tenant, input); err != nil {
		return nil, err
	}
	id, err := spec.NewUUIDv4()
	if err != nil {
		return nil, err
	}
	p := &d.AccessReviewPlan{AccessReviewPlanInput: input, ID: id, TenantID: tenant, Revision: 1, NextStartAt: &input.StartsAt}
	if err := deps.Plans.CreatePlan(ctx, p); err != nil {
		return nil, err
	}
	emitReviewEvent(deps, p.TenantID, actor, p.ID, "", "", "plan_created", now)
	return p, nil
}

func UpdateAccessReviewPlan(ctx context.Context, deps AccessReviewDeps, tenant, actor, id string, revision int64, input d.AccessReviewPlanInput, now time.Time) (*d.AccessReviewPlan, error) {
	if deps.Emit == nil {
		return nil, errors.New("access review audit is not configured")
	}
	if err := input.NormalizeAndValidate(); err != nil {
		return nil, err
	}
	current, err := deps.Plans.FindPlan(ctx, tenant, id)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, d.ErrAccessReviewInvalid
	}
	if current.Revision != revision {
		return nil, d.ErrAccessReviewConflict
	}
	// 参照先が止まっていても、設定を変えずに計画だけを止められる。
	if input.Enabled || !sameReviewDefinition(input, current.AccessReviewPlanInput) {
		if err := validateReviewReferences(ctx, deps, tenant, input); err != nil {
			return nil, err
		}
	}
	var lastStart *time.Time
	if current.RecurrenceDays != input.RecurrenceDays {
		reviews, err := deps.Reviews.ListByPlan(ctx, tenant, id)
		if err != nil {
			return nil, err
		}
		if len(reviews) > 0 {
			at := reviews[0].StartedAt
			lastStart = &at
		}
	}
	p, err := deps.Plans.UpdatePlan(ctx, tenant, id, func(plan *d.AccessReviewPlan) error {
		if plan.Revision != revision || !sameReviewTime(plan.NextStartAt, current.NextStartAt) {
			return d.ErrAccessReviewConflict
		}
		// 更新後の開始時刻は以後の回だけに使い、過去の実施を作り直さない。
		if !plan.StartsAt.Equal(input.StartsAt) || (!plan.Enabled && input.Enabled) {
			plan.NextStartAt = &input.StartsAt
		} else if plan.RecurrenceDays != input.RecurrenceDays && lastStart != nil {
			plan.NextStartAt = nil
			if input.RecurrenceDays > 0 {
				at := lastStart.Add(time.Duration(input.RecurrenceDays) * 24 * time.Hour)
				plan.NextStartAt = &at
			}
		}
		plan.AccessReviewPlanInput = input
		plan.Revision++
		plan.LastError = ""
		return nil
	})
	if err == nil {
		emitReviewEvent(deps, tenant, actor, id, "", "", "plan_updated", now)
	}
	return p, err
}

func DecideAccessReview(ctx context.Context, deps AccessReviewDeps, tenant, reviewID, itemID, reviewer, decision string, now time.Time) (*d.AccessReview, error) {
	if deps.Emit == nil {
		return nil, errors.New("access review audit is not configured")
	}
	changed := false
	r, err := deps.Reviews.UpdateReview(ctx, tenant, reviewID, func(r *d.AccessReview) error {
		var err error
		changed, err = r.Decide(itemID, reviewer, decision, now)
		return err
	})
	if err != nil {
		return nil, err
	}
	if changed {
		emitReviewEvent(deps, tenant, reviewer, r.PlanID, r.ID, itemID, "decision_recorded", now)
	}
	// 本人の回答の応答にも、同じ回の他人の項目を混ぜない。
	r.Items = slices.DeleteFunc(r.Items, func(i d.AccessReviewItem) bool { return i.ReviewerID != reviewer })
	return r, nil
}

func buildAccessReview(ctx context.Context, deps AccessReviewDeps, plan *d.AccessReviewPlan, now time.Time) (*d.AccessReview, error) {
	// 開始時は停止した明示の担当者を代替へ回せるので、対象と代替だけを再検証する。
	validation := plan.AccessReviewPlanInput
	if validation.TargetKind != "agent" {
		validation.ReviewerID = validation.FallbackReviewerID
	}
	if err := validateReviewReferences(ctx, deps, plan.TenantID, validation); err != nil {
		return nil, err
	}
	id, err := spec.NewUUIDv4()
	if err != nil {
		return nil, err
	}
	r := &d.AccessReview{ID: id, TenantID: plan.TenantID, PlanID: plan.ID, Name: plan.Name, TargetKind: plan.TargetKind, DefaultDecision: plan.DefaultDecision, Status: "open", StartedAt: now, DueAt: now.Add(time.Duration(plan.DurationDays) * 24 * time.Hour), Items: []d.AccessReviewItem{}, Notified: map[string]bool{}}
	reviewer := plan.ReviewerID
	if plan.TargetKind != "agent" {
		u, err := reviewUser(ctx, deps, plan.TenantID, reviewer)
		if err != nil {
			return nil, err
		}
		if u == nil {
			reviewer = plan.FallbackReviewerID
		}
	}
	appendItem := func(resourceID, resourceName, subjectType, subjectID, subjectName, reviewerID string) error {
		if len(r.Items) >= 1000 {
			return d.ErrAccessReviewInvalid
		}
		itemID, err := spec.NewUUIDv4()
		if err != nil {
			return err
		}
		r.Items = append(r.Items, d.AccessReviewItem{ID: itemID, ResourceID: resourceID, ResourceName: resourceName, SubjectType: subjectType, SubjectID: subjectID, SubjectName: subjectName, ReviewerID: reviewerID})
		return nil
	}
	for _, resourceID := range plan.ResourceIDs {
		switch plan.TargetKind {
		case "group_membership":
			g, err := deps.GroupRepo.FindByID(ctx, plan.TenantID, resourceID)
			if err != nil {
				return nil, err
			}
			if g == nil || g.UpstreamManaged || g.MembershipType.Effective() != groupdomain.GroupMembershipManual {
				return nil, d.ErrAccessReviewInvalid
			}
			members, err := deps.GroupRepo.ListMembersByGroup(ctx, plan.TenantID, resourceID)
			if err != nil {
				return nil, err
			}
			for _, m := range members {
				u, err := deps.UserRepo.FindBySub(ctx, m.UserID)
				if err != nil {
					return nil, err
				}
				name := m.UserID
				if u != nil && u.TenantID == plan.TenantID {
					name = u.DisplayName()
				}
				if err := appendItem(g.ID, g.Name, "user", m.UserID, name, reviewer); err != nil {
					return nil, err
				}
			}
		case "application_assignment":
			a, err := deps.ApplicationRepo.FindByID(ctx, plan.TenantID, resourceID)
			if err != nil {
				return nil, err
			}
			if a == nil {
				return nil, d.ErrAccessReviewInvalid
			}
			assignments, err := deps.AssignmentRepo.ListByApplication(ctx, plan.TenantID, resourceID)
			if err != nil {
				return nil, err
			}
			for _, assignment := range assignments {
				name := assignment.SubjectID
				if assignment.SubjectType == appdomain.AssignmentSubjectUser {
					u, err := deps.UserRepo.FindBySub(ctx, assignment.SubjectID)
					if err != nil {
						return nil, err
					}
					if u != nil && u.TenantID == plan.TenantID {
						name = u.DisplayName()
					}
				} else {
					g, err := deps.GroupRepo.FindByID(ctx, plan.TenantID, assignment.SubjectID)
					if err != nil {
						return nil, err
					}
					if g != nil {
						name = g.Name
					}
				}
				if err := appendItem(a.ID, a.Name, string(assignment.SubjectType), assignment.SubjectID, name, reviewer); err != nil {
					return nil, err
				}
			}
		case "agent":
			a, err := deps.AgentRepo.FindByID(ctx, plan.TenantID, resourceID)
			if err != nil {
				return nil, err
			}
			if a == nil {
				return nil, d.ErrAccessReviewInvalid
			}
			u, err := reviewUser(ctx, deps, plan.TenantID, a.OwnerUserID)
			if err != nil {
				return nil, err
			}
			owner := plan.FallbackReviewerID
			if u != nil {
				owner = u.ID
			}
			if err := appendItem(a.ID, a.Name, "agent", a.ID, a.Name, owner); err != nil {
				return nil, err
			}
		}
	}
	return r, nil
}

func AdvanceAccessReviews(ctx context.Context, deps AccessReviewDeps, now time.Time) error {
	if deps.Emit == nil {
		return errors.New("access review audit is not configured")
	}
	plans, err := deps.Plans.ReadyPlans(ctx, now)
	if err != nil {
		return err
	}
	var failures []error
	for _, plan := range plans {
		r, err := buildAccessReview(ctx, deps, plan, now)
		if err != nil {
			_, saveErr := deps.Plans.UpdatePlan(ctx, plan.TenantID, plan.ID, func(p *d.AccessReviewPlan) error {
				if p.Revision == plan.Revision {
					p.LastError = "start_failed"
				}
				return nil
			})
			failures = append(failures, errors.Join(err, saveErr))
			continue
		}
		var next *time.Time
		if plan.RecurrenceDays > 0 {
			at := now.Add(time.Duration(plan.RecurrenceDays) * 24 * time.Hour)
			next = &at
		}
		captured, err := deps.Plans.CaptureReview(ctx, plan, r, next)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		if captured {
			emitReviewEvent(deps, plan.TenantID, "access-review", plan.ID, r.ID, "", "review_started", now)
		}
	}
	reviews, err := deps.Reviews.ActiveReviews(ctx)
	if err != nil {
		return errors.Join(append(failures, err)...)
	}
	for _, review := range reviews {
		if err := advanceAccessReview(ctx, deps, review, now); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func revokeReviewItem(ctx context.Context, deps AccessReviewDeps, r *d.AccessReview, item *d.AccessReviewItem, now time.Time) error {
	switch r.TargetKind {
	case "group_membership":
		if deps.GroupMembership == nil || deps.GroupRepo == nil {
			return errors.New("group revocation is not configured")
		}
		g, err := deps.GroupRepo.FindByID(ctx, r.TenantID, item.ResourceID)
		if err != nil || g == nil {
			return err
		}
		if g.UpstreamManaged || g.MembershipType.Effective() != groupdomain.GroupMembershipManual {
			return d.ErrAccessReviewInvalid
		}
		_, err = deps.GroupMembership.SetGroupMembership(ctx, r.TenantID, item.ResourceID, item.SubjectID, false, now)
		return err
	case "application_assignment":
		if deps.ApplicationAssignments == nil {
			return errors.New("application revocation is not configured")
		}
		return deps.ApplicationAssignments.UnassignReviewedApplication(ctx, r.TenantID, item.ResourceID, appdomain.AssignmentSubjectType(item.SubjectType), item.SubjectID, now)
	case "agent":
		if deps.Agents == nil {
			return errors.New("agent revocation is not configured")
		}
		return deps.Agents.DisableReviewedAgent(ctx, r.TenantID, item.ResourceID, now)
	}
	return d.ErrAccessReviewInvalid
}

func emitReviewEvent(deps AccessReviewDeps, tenant, actor, planID, reviewID, itemID, action string, now time.Time) {
	deps.Emit(&d.AccessReviewChanged{At: now, TenantID: tenant, ActorUserID: actor, PlanID: planID, ReviewID: reviewID, ItemID: itemID, Action: action})
}

func sameReviewDefinition(a, b d.AccessReviewPlanInput) bool {
	return a.Name == b.Name && a.TargetKind == b.TargetKind && slices.Equal(a.ResourceIDs, b.ResourceIDs) && a.ReviewerID == b.ReviewerID && a.FallbackReviewerID == b.FallbackReviewerID && a.DefaultDecision == b.DefaultDecision && a.StartsAt.Equal(b.StartsAt) && a.DurationDays == b.DurationDays && a.RecurrenceDays == b.RecurrenceDays
}

func sameReviewTime(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}
