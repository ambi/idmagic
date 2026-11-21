package bootstrap

import (
	"time"

	"github.com/ambi/idmagic/backend/idgovernance"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// AccessReviewDeps は worker の見直しと、状態の所有者の必須の作用を結ぶ。
func (d *Dependencies) AccessReviewDeps(emit func(spec.DomainEvent)) idgovernance.AccessReviewDeps {
	audit := func(event spec.DomainEvent) error { emit(event); return nil }
	const actor = "access-review"
	return idgovernance.AccessReviewDeps{
		Now:      time.Now,
		UserRepo: d.IdManagement.UserRepo, GroupRepo: d.IdManagement.GroupRepo, AgentRepo: d.IdManagement.AgentRepo,
		ApplicationRepo: d.Application.Repo, AssignmentRepo: d.Application.AssignmentRepo,
		GroupMembership:        d.IdManagement.GroupMembershipCommands(audit, actor),
		ApplicationAssignments: d.Application.DesiredStateAssignments(d.IdManagement.UserRepo, d.IdManagement.GroupRepo, emit, actor),
		Agents:                 d.IdManagement.AgentLifecycleCommands(audit, d.AgentRevocationReactor(emit), actor),
		Notifier:               d.Notification.Notifier, Emit: emit,
	}
}
