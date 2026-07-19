package domain

import (
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
