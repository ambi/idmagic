package domain

import (
	"encoding/json"
	"maps"
	"slices"
	"testing"
	"time"
)

// TestProvisioningEvents_ImplementDomainEvent is a structural check: each event
// exposes EventType()/OccurredAt() with the signature backend/shared/spec.DomainEvent
// requires, without importing that package from domain (mirrors
// backend/jobs/domain/job_test.go TestJobEvents_ImplementDomainEvent).
func TestProvisioningEvents_ImplementDomainEvent(t *testing.T) {
	type domainEvent interface {
		EventType() string
		OccurredAt() time.Time
	}
	now := time.Now()
	events := []domainEvent{
		&ProvisioningConnectionRegistered{At: now, TenantID: "tenant-a", ApplicationID: "app-1"},
		&ProvisioningConnectionUpdated{At: now, TenantID: "tenant-a", ApplicationID: "app-1"},
		&ProvisioningConnectionDisabled{At: now, TenantID: "tenant-a", ApplicationID: "app-1"},
		&ProvisioningConnectionDeleted{At: now, TenantID: "tenant-a", ApplicationID: "app-1"},
		&ProvisioningCredentialRotated{At: now, TenantID: "tenant-a", ApplicationID: "app-1", CredentialID: "cred-1"},
		&ProvisioningDeliveryStarted{At: now, TenantID: "tenant-a", ConnectionID: "app-1", DeliveryID: "delivery-1", JobID: "job-1"},
		&UserProvisioned{At: now, TenantID: "tenant-a", ConnectionID: "app-1", DeliveryID: "delivery-1", UserID: "user-1", RemoteID: "remote-1"},
		&UserDeprovisioned{At: now, TenantID: "tenant-a", ConnectionID: "app-1", DeliveryID: "delivery-1", UserID: "user-1", Action: DeprovisionDeactivate},
		&UserProvisioningFailed{At: now, TenantID: "tenant-a", ConnectionID: "app-1", DeliveryID: "delivery-1", SourceType: SourceTypeUser, SourceID: "user-1", Error: "boom"},
		&GroupPushed{At: now, TenantID: "tenant-a", ConnectionID: "app-1", DeliveryID: "delivery-1", GroupID: "group-1", RemoteID: "remote-1"},
		&GroupMembershipPushed{At: now, TenantID: "tenant-a", ConnectionID: "app-1", DeliveryID: "delivery-1", GroupID: "group-1"},
		&ConnectionQuarantined{At: now, TenantID: "tenant-a", ApplicationID: "app-1", Reason: "too many failures", ConsecutiveFailures: 10},
		&ProvisioningConnectionQuarantineCleared{At: now, TenantID: "tenant-a", ApplicationID: "app-1"},
		&FullResyncCompleted{At: now, TenantID: "tenant-a", ApplicationID: "app-1", TotalSubjects: 10, SucceededCount: 9, FailedCount: 1},
	}
	wantTypes := []string{
		"ProvisioningConnectionRegistered", "ProvisioningConnectionUpdated", "ProvisioningConnectionDisabled",
		"ProvisioningConnectionDeleted", "ProvisioningCredentialRotated", "ProvisioningDeliveryStarted",
		"UserProvisioned", "UserDeprovisioned", "UserProvisioningFailed", "GroupPushed", "GroupMembershipPushed",
		"ConnectionQuarantined", "ProvisioningConnectionQuarantineCleared", "FullResyncCompleted",
	}
	if len(events) != len(wantTypes) {
		t.Fatalf("test setup mismatch: %d events, %d wantTypes", len(events), len(wantTypes))
	}
	for i, e := range events {
		if got := e.EventType(); got != wantTypes[i] {
			t.Errorf("events[%d].EventType() = %q, want %q", i, got, wantTypes[i])
		}
		if !e.OccurredAt().Equal(now) {
			t.Errorf("events[%d].OccurredAt() = %v, want %v", i, e.OccurredAt(), now)
		}
	}
}

// 監査はイベントを JSON にして保存し、テナントを `tenantId` から取り出す。項目名が
// TypeSpec の宣言と異なると、購読側が値を読めず、監査のテナント絞り込みからも漏れる。
// `occurredAt` は spec.MarshalDomainEvent が加えるので、構造体からは出さない。
func TestProvisioningEvents_MarshalWithTheContractFieldNames(t *testing.T) {
	now := time.Now()
	cases := []struct {
		event any
		want  []string
	}{
		{&ProvisioningConnectionRegistered{At: now}, []string{"tenantId", "applicationId"}},
		{&ProvisioningConnectionUpdated{At: now}, []string{"tenantId", "applicationId"}},
		{&ProvisioningConnectionDisabled{At: now}, []string{"tenantId", "applicationId"}},
		{&ProvisioningConnectionDeleted{At: now}, []string{"tenantId", "applicationId"}},
		{&ProvisioningCredentialRotated{At: now}, []string{"tenantId", "applicationId", "credentialId"}},
		{&ProvisioningDeliveryStarted{At: now}, []string{"tenantId", "connectionId", "deliveryId", "jobId"}},
		{&UserProvisioned{At: now}, []string{"tenantId", "connectionId", "deliveryId", "userId", "remoteId"}},
		{&UserDeprovisioned{At: now}, []string{"tenantId", "connectionId", "deliveryId", "userId", "action"}},
		{&UserProvisioningFailed{At: now}, []string{"tenantId", "connectionId", "deliveryId", "sourceType", "sourceId", "error"}},
		{&GroupPushed{At: now}, []string{"tenantId", "connectionId", "deliveryId", "groupId", "remoteId"}},
		{&GroupMembershipPushed{At: now}, []string{"tenantId", "connectionId", "deliveryId", "groupId"}},
		{&ConnectionQuarantined{At: now}, []string{"tenantId", "applicationId", "reason", "consecutiveFailures"}},
		{&ProvisioningConnectionQuarantineCleared{At: now}, []string{"tenantId", "applicationId"}},
		{&FullResyncCompleted{At: now}, []string{"tenantId", "applicationId", "totalSubjects", "succeededCount", "failedCount"}},
	}
	for _, c := range cases {
		wire, err := json.Marshal(c.event)
		if err != nil {
			t.Fatalf("json.Marshal(%T) error = %v", c.event, err)
		}
		var fields map[string]any
		if err := json.Unmarshal(wire, &fields); err != nil {
			t.Fatalf("json.Unmarshal(%T) error = %v", c.event, err)
		}
		got := slices.Sorted(maps.Keys(fields))
		want := slices.Sorted(slices.Values(c.want))
		if !slices.Equal(got, want) {
			t.Errorf("%T fields = %v, want %v", c.event, got, want)
		}
	}
}
