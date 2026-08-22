package usecases_test

// REQ-OAUTH2-050 の承認側 (wi-376 T003)。Supervised な Agent が唯一通れる経路は CIBA
// であり、その承認は「どの Agent を誰が許可したか」を監査へ残す。拒否側は
// backend/oauth2/usecases と backend/oauth2/token/usecases のテストが持つ。

import (
	"testing"
	"time"

	approvalusecases "github.com/ambi/idmagic/backend/oauth2/approval/usecases"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestApprovalRecordsTheAgentItPermitted(t *testing.T) {
	f := newApprovalFixture(t)
	t0 := time.Now().UTC()

	if _, err := approvalusecases.StartApproval(f.ctx, f.startDeps, approvalusecases.StartApprovalInput{
		ClientID: "agent-app", LoginHint: "alice", Scope: "openid",
	}, t0); err != nil {
		t.Fatal(err)
	}
	records, err := f.store.ListPendingForUser(f.ctx, "alice-id")
	if err != nil || len(records) != 1 {
		t.Fatalf("pending records = %d, err = %v", len(records), err)
	}

	var events []spec.DomainEvent
	emit := func(e spec.DomainEvent) { events = append(events, e) }
	if err := approvalusecases.DecideApproval(f.ctx, f.store, emit, "alice-id", records[0].ID, true, t0.Add(time.Second)); err != nil {
		t.Fatal(err)
	}

	for _, e := range events {
		approved, ok := e.(*oauthdomain.BackchannelAuthApproved)
		if !ok {
			continue
		}
		if approved.AgentID != "agent-1" {
			t.Fatalf("agentId = %q, want agent-1 (the approval must name the agent it permitted)", approved.AgentID)
		}
		return
	}
	t.Fatalf("expected BackchannelAuthApproved, got %v", events)
}
