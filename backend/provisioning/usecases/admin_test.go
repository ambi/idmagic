package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/provisioning/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/provisioning/domain"
	"github.com/ambi/idmagic/backend/provisioning/ports"
	"github.com/ambi/idmagic/backend/provisioning/usecases"
)

func newAdminDeps() (usecases.AdminDeps, *memory.ProvisioningConnectionRepository, *memory.ProvisioningDeliveryRepository) {
	connRepo := memory.NewProvisioningConnectionRepository()
	deliveryRepo := memory.NewProvisioningDeliveryRepository()
	return usecases.AdminDeps{ConnectionRepo: connRepo, DeliveryRepo: deliveryRepo}, connRepo, deliveryRepo
}

func TestRegisterConnection_SeedsDefaultsAndRejectsUnsafeURL(t *testing.T) {
	deps, _, _ := newAdminDeps()
	_, err := usecases.RegisterConnection(context.Background(), deps, usecases.RegisterConnectionInput{
		TenantID: "tenant-a", ApplicationID: "app-1", BaseURL: "http://insecure.example.com",
		Credential: domain.ProvisioningCredentialInput{AuthMethod: domain.AuthBearerToken, BearerToken: "tok"},
		Now:        time.Now(),
	})
	if err == nil {
		t.Fatal("RegisterConnection() with a non-https base_url should fail")
	}
}

func TestRegisterConnection_SeedsDefaultAttributeMappingAndRejectsDuplicate(t *testing.T) {
	deps, _, _ := newAdminDeps()
	now := time.Now()
	conn, err := usecases.RegisterConnection(context.Background(), deps, usecases.RegisterConnectionInput{
		TenantID: "tenant-a", ApplicationID: "app-1", BaseURL: "https://downstream.example.com/scim/v2",
		Credential: domain.ProvisioningCredentialInput{AuthMethod: domain.AuthBearerToken, BearerToken: "tok"},
		Now:        now,
	})
	if err != nil {
		t.Fatalf("RegisterConnection() error = %v", err)
	}
	if len(conn.AttributeMappings) == 0 {
		t.Error("RegisterConnection() should seed default attribute mappings")
	}
	if conn.Scope != domain.ScopeAssignedOnly {
		t.Errorf("RegisterConnection() default scope = %v, want assigned_only", conn.Scope)
	}
	_, err = usecases.RegisterConnection(context.Background(), deps, usecases.RegisterConnectionInput{
		TenantID: "tenant-a", ApplicationID: "app-1", BaseURL: "https://downstream.example.com/scim/v2",
		Credential: domain.ProvisioningCredentialInput{AuthMethod: domain.AuthBearerToken, BearerToken: "tok2"},
		Now:        now,
	})
	if !errors.Is(err, ports.ErrConnectionAlreadyExists) {
		t.Errorf("RegisterConnection() duplicate error = %v, want ErrConnectionAlreadyExists", err)
	}
}

func TestUpdateConnection_PartialUpdateOnlyChangesGivenFields(t *testing.T) {
	deps, connRepo, _ := newAdminDeps()
	ctx := context.Background()
	now := time.Now()
	_, err := usecases.RegisterConnection(ctx, deps, usecases.RegisterConnectionInput{
		TenantID: "tenant-a", ApplicationID: "app-1", BaseURL: "https://downstream.example.com/scim/v2",
		Credential: domain.ProvisioningCredentialInput{AuthMethod: domain.AuthBearerToken, BearerToken: "tok"},
		Now:        now,
	})
	if err != nil {
		t.Fatalf("RegisterConnection() error = %v", err)
	}
	newURL := "https://downstream.example.com/scim/v2/updated"
	updated, err := usecases.UpdateConnection(ctx, deps, usecases.UpdateConnectionInput{
		TenantID: "tenant-a", ApplicationID: "app-1", BaseURL: &newURL, Now: now,
	})
	if err != nil {
		t.Fatalf("UpdateConnection() error = %v", err)
	}
	if updated.BaseURL != newURL {
		t.Errorf("UpdateConnection() BaseURL = %q, want %q", updated.BaseURL, newURL)
	}
	if updated.Scope != domain.ScopeAssignedOnly {
		t.Errorf("UpdateConnection() unexpectedly changed Scope to %v", updated.Scope)
	}
	secret, err := connRepo.CredentialSecret(ctx, "tenant-a", "app-1")
	if err != nil || secret != "tok" {
		t.Errorf("CredentialSecret() after non-rotating update = (%q, %v), want (tok, nil)", secret, err)
	}
}

func TestUpdateConnection_RotatesCredentialWhenProvided(t *testing.T) {
	deps, connRepo, _ := newAdminDeps()
	ctx := context.Background()
	now := time.Now()
	_, _ = usecases.RegisterConnection(ctx, deps, usecases.RegisterConnectionInput{
		TenantID: "tenant-a", ApplicationID: "app-1", BaseURL: "https://downstream.example.com/scim/v2",
		Credential: domain.ProvisioningCredentialInput{AuthMethod: domain.AuthBearerToken, BearerToken: "tok"},
		Now:        now,
	})
	newCred := domain.ProvisioningCredentialInput{AuthMethod: domain.AuthBearerToken, BearerToken: "tok-rotated"}
	if _, err := usecases.UpdateConnection(ctx, deps, usecases.UpdateConnectionInput{
		TenantID: "tenant-a", ApplicationID: "app-1", Credential: &newCred, Now: now,
	}); err != nil {
		t.Fatalf("UpdateConnection() error = %v", err)
	}
	secret, err := connRepo.CredentialSecret(ctx, "tenant-a", "app-1")
	if err != nil || secret != "tok-rotated" {
		t.Errorf("CredentialSecret() after rotation = (%q, %v), want (tok-rotated, nil)", secret, err)
	}
}

func TestResumeConnection_RequiresQuarantined(t *testing.T) {
	deps, _, _ := newAdminDeps()
	ctx := context.Background()
	now := time.Now()
	_, _ = usecases.RegisterConnection(ctx, deps, usecases.RegisterConnectionInput{
		TenantID: "tenant-a", ApplicationID: "app-1", BaseURL: "https://downstream.example.com/scim/v2",
		Credential: domain.ProvisioningCredentialInput{AuthMethod: domain.AuthBearerToken, BearerToken: "tok"},
		Now:        now,
	})
	if _, err := usecases.ResumeConnection(ctx, deps, "tenant-a", "app-1", now); !errors.Is(err, domain.ErrConnectionNotQuarantined) {
		t.Errorf("ResumeConnection() on a non-quarantined connection error = %v, want ErrConnectionNotQuarantined", err)
	}
}

// TestResumeConnection_ClearsQuarantineAndAllowsNextDelivery covers the
// positive path TestResumeConnection_RequiresQuarantined leaves untested
// (wi-45 T008): a quarantined connection is resumed (health back to ok,
// consecutive failure count reset), and capture immediately resumes creating
// deliveries for it again (a still-quarantined connection is skipped by
// CaptureLifecycleEvent, per TestCaptureLifecycleEvent_SkipsDisabledAndQuarantinedConnections).
func TestResumeConnection_ClearsQuarantineAndAllowsNextDelivery(t *testing.T) {
	deps, connRepo, deliveryRepo := newAdminDeps()
	ctx := context.Background()
	now := time.Now().UTC()
	conn, err := usecases.RegisterConnection(ctx, deps, usecases.RegisterConnectionInput{
		TenantID: "tenant-a", ApplicationID: "app-1", BaseURL: "https://downstream.example.com/scim/v2",
		Credential: domain.ProvisioningCredentialInput{AuthMethod: domain.AuthBearerToken, BearerToken: "tok"},
		Now:        now,
	})
	if err != nil {
		t.Fatalf("RegisterConnection() error = %v", err)
	}
	conn.Scope = domain.ScopeAllUsers // avoid needing an AssignmentRepo just to prove capture resumes
	conn.ConsecutiveFailureCount = 7
	if err := conn.Quarantine("too many failures", now); err != nil {
		t.Fatalf("Quarantine() error = %v", err)
	}
	if err := connRepo.Update(ctx, conn, nil); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	resumed, err := usecases.ResumeConnection(ctx, deps, "tenant-a", "app-1", now)
	if err != nil {
		t.Fatalf("ResumeConnection() error = %v", err)
	}
	if resumed.Health != domain.HealthOK {
		t.Errorf("resumed.Health = %v, want ok", resumed.Health)
	}
	if resumed.ConsecutiveFailureCount != 0 {
		t.Errorf("resumed.ConsecutiveFailureCount = %d, want 0", resumed.ConsecutiveFailureCount)
	}

	captureDeps := usecases.CaptureDeps{ConnectionRepo: connRepo, DeliveryRepo: deliveryRepo}
	if err := usecases.CaptureLifecycleEvent(ctx, captureDeps, "tenant-a", domain.SourceTypeUser, "user-1", ports.TriggerUserCreated, "", now); err != nil {
		t.Fatalf("CaptureLifecycleEvent() error = %v", err)
	}
	deliveries, err := deliveryRepo.ListByConnection(ctx, "tenant-a", "app-1", nil, 10)
	if err != nil || len(deliveries) != 1 {
		t.Fatalf("ListByConnection() = %+v, err=%v, want 1 delivery after resume", deliveries, err)
	}
}

func TestRetryDelivery_RequiresDeadLetter(t *testing.T) {
	deps, connRepo, deliveryRepo := newAdminDeps()
	ctx := context.Background()
	now := time.Now()
	_, _ = usecases.RegisterConnection(ctx, deps, usecases.RegisterConnectionInput{
		TenantID: "tenant-a", ApplicationID: "app-1", BaseURL: "https://downstream.example.com/scim/v2",
		Credential: domain.ProvisioningCredentialInput{AuthMethod: domain.AuthBearerToken, BearerToken: "tok"},
		Now:        now,
	})
	_ = connRepo
	d := &domain.ProvisioningDelivery{
		ID: "delivery-1", TenantID: "tenant-a", ConnectionID: "app-1", SourceType: domain.SourceTypeUser, SourceID: "user-1",
		SourceVersion: 1, Operation: domain.OperationCreate, Status: domain.DeliveryPending, CreatedAt: now, UpdatedAt: now,
	}
	if _, err := deliveryRepo.Save(ctx, d); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if _, err := usecases.RetryDelivery(ctx, deps, "tenant-a", "app-1", "delivery-1"); !errors.Is(err, usecases.ErrDeliveryNotRetryable) {
		t.Errorf("RetryDelivery() on a pending delivery error = %v, want ErrDeliveryNotRetryable", err)
	}
}

func TestProvisionOnDemand_RejectsSubjectOutOfScope(t *testing.T) {
	deps, _, _ := newAdminDeps()
	ctx := context.Background()
	now := time.Now()
	_, _ = usecases.RegisterConnection(ctx, deps, usecases.RegisterConnectionInput{
		TenantID: "tenant-a", ApplicationID: "app-1", BaseURL: "https://downstream.example.com/scim/v2",
		Credential: domain.ProvisioningCredentialInput{AuthMethod: domain.AuthBearerToken, BearerToken: "tok"},
		Now:        now,
	})
	_, err := usecases.ProvisionOnDemand(ctx, deps, "tenant-a", "app-1", domain.SourceTypeUser, "user-1", now)
	if !errors.Is(err, usecases.ErrSubjectNotInScope) {
		t.Errorf("ProvisionOnDemand() for an unassigned user (scope=assigned_only) error = %v, want ErrSubjectNotInScope", err)
	}
}
