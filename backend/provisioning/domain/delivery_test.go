package domain

import (
	"testing"
	"time"
)

// allDeliveryStatuses and allDeliveryEvents enumerate the ProvisioningDeliveryLifecycle
// alphabet (spec/contexts/provisioning.yaml states.ProvisioningDeliveryLifecycle) so the
// invariant tests below can exhaustively check every (status, event) pair, mirroring
// backend/jobs/domain/job_test.go.
var (
	allDeliveryStatuses = []ProvisioningDeliveryStatus{
		DeliveryPending, DeliveryInFlight, DeliverySucceeded, DeliveryDeadLetter,
	}
	allDeliveryEvents = []ProvisioningDeliveryLifecycleEvent{
		EventProvisioningDeliveryStarted,
		EventUserProvisioned,
		EventUserDeprovisioned,
		EventGroupPushed,
		EventGroupMembershipPushed,
		EventUserProvisioningFailed,
	}
)

func TestTransitionProvisioningDeliveryLifecycle_DeclaredTransitions(t *testing.T) {
	tests := []struct {
		from  ProvisioningDeliveryStatus
		event ProvisioningDeliveryLifecycleEvent
		want  ProvisioningDeliveryStatus
	}{
		{DeliveryPending, EventProvisioningDeliveryStarted, DeliveryInFlight},
		{DeliveryInFlight, EventUserProvisioned, DeliverySucceeded},
		{DeliveryInFlight, EventUserDeprovisioned, DeliverySucceeded},
		{DeliveryInFlight, EventGroupPushed, DeliverySucceeded},
		{DeliveryInFlight, EventGroupMembershipPushed, DeliverySucceeded},
		{DeliveryInFlight, EventUserProvisioningFailed, DeliveryDeadLetter},
	}
	for _, tt := range tests {
		got, err := TransitionProvisioningDeliveryLifecycle(tt.from, tt.event)
		if err != nil {
			t.Errorf("TransitionProvisioningDeliveryLifecycle(%q, %q) unexpected error: %v", tt.from, tt.event, err)
			continue
		}
		if got != tt.want {
			t.Errorf("TransitionProvisioningDeliveryLifecycle(%q, %q) = %q, want %q", tt.from, tt.event, got, tt.want)
		}
	}
}

func TestTransitionProvisioningDeliveryLifecycle_InvariantOnlyDeclaredTransitionsSucceed(t *testing.T) {
	declared := map[[2]string]bool{}
	for _, tr := range provisioningDeliveryTransitions {
		declared[[2]string{string(tr.From), string(tr.Event)}] = true
	}
	for _, from := range allDeliveryStatuses {
		for _, event := range allDeliveryEvents {
			_, err := TransitionProvisioningDeliveryLifecycle(from, event)
			ok := declared[[2]string{string(from), string(event)}]
			if ok && err != nil {
				t.Errorf("TransitionProvisioningDeliveryLifecycle(%q, %q) should succeed (declared) but got error: %v", from, event, err)
			}
			if !ok && err == nil {
				t.Errorf("TransitionProvisioningDeliveryLifecycle(%q, %q) should fail (undeclared) but succeeded", from, event)
			}
		}
	}
}

func TestTransitionProvisioningDeliveryLifecycle_InvariantTerminalStatesHaveNoOutgoingTransitions(t *testing.T) {
	for _, tr := range provisioningDeliveryTransitions {
		if IsProvisioningDeliveryTerminal(tr.From) {
			t.Errorf("terminal status %q has outgoing transition on event %q", tr.From, tr.Event)
		}
	}
}

func TestIsProvisioningDeliveryTerminal(t *testing.T) {
	terminal := map[ProvisioningDeliveryStatus]bool{DeliverySucceeded: true, DeliveryDeadLetter: true}
	for _, s := range allDeliveryStatuses {
		if got, want := IsProvisioningDeliveryTerminal(s), terminal[s]; got != want {
			t.Errorf("IsProvisioningDeliveryTerminal(%q) = %v, want %v", s, got, want)
		}
	}
}

func TestProvisioningDelivery_IdempotencyKey_StableForSameInputs(t *testing.T) {
	d1 := ProvisioningDelivery{TenantID: "tenant-a", ConnectionID: "conn-1", SourceType: SourceTypeUser, SourceID: "user-1", SourceVersion: 3}
	d2 := ProvisioningDelivery{TenantID: "tenant-a", ConnectionID: "conn-1", SourceType: SourceTypeUser, SourceID: "user-1", SourceVersion: 3}
	if d1.IdempotencyKey() != d2.IdempotencyKey() {
		t.Errorf("IdempotencyKey() not stable for identical inputs: %q != %q", d1.IdempotencyKey(), d2.IdempotencyKey())
	}
}

func TestProvisioningDelivery_IdempotencyKey_DiffersOnAnyComponent(t *testing.T) {
	base := ProvisioningDelivery{TenantID: "tenant-a", ConnectionID: "conn-1", SourceType: SourceTypeUser, SourceID: "user-1", SourceVersion: 3}
	variants := []ProvisioningDelivery{
		{TenantID: "tenant-b", ConnectionID: "conn-1", SourceType: SourceTypeUser, SourceID: "user-1", SourceVersion: 3},
		{TenantID: "tenant-a", ConnectionID: "conn-2", SourceType: SourceTypeUser, SourceID: "user-1", SourceVersion: 3},
		{TenantID: "tenant-a", ConnectionID: "conn-1", SourceType: SourceTypeGroup, SourceID: "user-1", SourceVersion: 3},
		{TenantID: "tenant-a", ConnectionID: "conn-1", SourceType: SourceTypeUser, SourceID: "user-2", SourceVersion: 3},
		{TenantID: "tenant-a", ConnectionID: "conn-1", SourceType: SourceTypeUser, SourceID: "user-1", SourceVersion: 4},
	}
	for i, v := range variants {
		if v.IdempotencyKey() == base.IdempotencyKey() {
			t.Errorf("variant %d: IdempotencyKey() collided with base (%q)", i, base.IdempotencyKey())
		}
	}
}

func TestRemoteResourceLink_ApplySync_FirstSyncAlwaysApplies(t *testing.T) {
	now := time.Now().UTC()
	link := NewRemoteResourceLink("conn-1", "tenant-a", SourceTypeUser, "user-1")
	if err := link.ApplySync(1, "remote-1", "user-1", nil, now); err != nil {
		t.Fatalf("ApplySync() on fresh link returned error: %v", err)
	}
	if link.RemoteID != "remote-1" || link.LastSyncedVersion != 1 {
		t.Errorf("ApplySync() did not update link: %+v", link)
	}
}

func TestRemoteResourceLink_ApplySync_RejectsOutOfOrderVersion(t *testing.T) {
	now := time.Now().UTC()
	link := NewRemoteResourceLink("conn-1", "tenant-a", SourceTypeUser, "user-1")
	if err := link.ApplySync(5, "remote-1", "user-1", nil, now); err != nil {
		t.Fatalf("ApplySync(5) returned error: %v", err)
	}
	if err := link.ApplySync(3, "remote-1", "user-1", nil, now); err == nil {
		t.Error("ApplySync(3) after version 5 should reject out-of-order version, got nil error")
	}
	if link.LastSyncedVersion != 5 {
		t.Errorf("out-of-order ApplySync must not mutate link, LastSyncedVersion = %d, want 5", link.LastSyncedVersion)
	}
}

func TestRemoteResourceLink_ApplySync_RejectsRepeatedVersion(t *testing.T) {
	now := time.Now().UTC()
	link := NewRemoteResourceLink("conn-1", "tenant-a", SourceTypeUser, "user-1")
	_ = link.ApplySync(2, "remote-1", "user-1", nil, now)
	if err := link.ApplySync(2, "remote-1", "user-1", nil, now); err == nil {
		t.Error("ApplySync(2) repeated should be rejected as non-monotonic, got nil error")
	}
}

func TestRemoteResourceLink_ApplySync_AcceptsStrictlyIncreasingVersion(t *testing.T) {
	now := time.Now().UTC()
	link := NewRemoteResourceLink("conn-1", "tenant-a", SourceTypeUser, "user-1")
	_ = link.ApplySync(1, "remote-1", "user-1", nil, now)
	if err := link.ApplySync(2, "remote-1", "user-1", nil, now); err != nil {
		t.Errorf("ApplySync(2) after 1 should succeed, got error: %v", err)
	}
	if link.LastSyncedVersion != 2 {
		t.Errorf("LastSyncedVersion = %d, want 2", link.LastSyncedVersion)
	}
}

func TestProvisioningSourceType_Valid(t *testing.T) {
	for _, s := range []ProvisioningSourceType{SourceTypeUser, SourceTypeGroup} {
		if !s.Valid() {
			t.Errorf("ProvisioningSourceType(%q).Valid() = false, want true", s)
		}
	}
	if ProvisioningSourceType("bogus").Valid() {
		t.Error(`ProvisioningSourceType("bogus").Valid() = true, want false`)
	}
}

func TestProvisioningDeliveryStatus_Valid(t *testing.T) {
	for _, s := range allDeliveryStatuses {
		if !s.Valid() {
			t.Errorf("ProvisioningDeliveryStatus(%q).Valid() = false, want true", s)
		}
	}
	if ProvisioningDeliveryStatus("bogus").Valid() {
		t.Error(`ProvisioningDeliveryStatus("bogus").Valid() = true, want false`)
	}
}
