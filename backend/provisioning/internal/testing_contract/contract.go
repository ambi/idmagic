// Package testing_contract defines the shared provisioning persistence contract.
package testing_contract

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	provisioningdomain "github.com/ambi/idmagic/backend/provisioning/internal/domain"
	provisioningports "github.com/ambi/idmagic/backend/provisioning/internal/ports"
)

type Fixture struct {
	Connections        provisioningports.ProvisioningConnectionRepository
	Tasks              provisioningports.ProvisioningTaskRepository
	Links              provisioningports.RemoteResourceLinkRepository
	SiblingApplication string
	OtherApplication   string
	TenantA            string
	TenantB            string
	Application        string
	CredentialID       string
	Now                time.Time
}

type NewFixture func(t *testing.T) Fixture

func Run(t *testing.T, newFixture NewFixture) {
	t.Helper()
	f := newFixture(t)
	ctx := context.Background()
	connection := &provisioningdomain.ProvisioningConnection{
		ApplicationID: f.Application, TenantID: f.TenantA, Status: provisioningdomain.ConnectionActive,
		BaseURL: "https://downstream.example/scim/v2",
		Credential: provisioningdomain.ProvisioningConnectionCredentialMetadata{
			CredentialID: f.CredentialID, AuthMethod: provisioningdomain.AuthBearerToken, CreatedAt: f.Now,
		},
		FeatureFlags: provisioningdomain.ProvisioningFeatureFlags{CreateUsers: true, UpdateUsers: true, DeactivateUsers: true},
		Scope:        provisioningdomain.ScopeAssignedOnly, Matching: provisioningdomain.MatchingRule{ConflictMatchAttribute: "userName"},
		DeprovisionPolicy: provisioningdomain.DeprovisionPolicy{
			OnUnassign: provisioningdomain.DeprovisionDeactivate, OnDelete: provisioningdomain.DeprovisionDeactivate,
		},
		RateLimitPerMinute: 60, MaxAttempts: 8, QuarantineAfterConsecutiveFailure: 10,
		Health: provisioningdomain.HealthOK, CreatedAt: f.Now, UpdatedAt: f.Now,
	}
	if err := f.Connections.Register(ctx, connection, "secret-1"); err != nil {
		t.Fatalf("Register: %v", err)
	}
	duplicate := *connection
	if err := f.Connections.Register(ctx, &duplicate, "secret-2"); !errors.Is(err, provisioningports.ErrConnectionAlreadyExists) {
		t.Fatalf("duplicate Register = %v", err)
	}
	found, err := f.Connections.Find(ctx, f.TenantA, f.Application)
	if err != nil || found == nil || found.BaseURL != connection.BaseURL {
		t.Fatalf("Find = (%+v, %v)", found, err)
	}
	if leaked, err := f.Connections.Find(ctx, f.TenantB, f.Application); err != nil || leaked != nil {
		t.Fatalf("other tenant Find = (%+v, %v)", leaked, err)
	}
	if secret, err := f.Connections.CredentialSecret(ctx, f.TenantA, f.Application); err != nil || secret != "secret-1" {
		t.Fatalf("CredentialSecret = (%q, %v)", secret, err)
	}
	connection.BaseURL += "/updated"
	if err := f.Connections.Update(ctx, connection, nil); err != nil {
		t.Fatalf("Update without secret: %v", err)
	}
	if secret, err := f.Connections.CredentialSecret(ctx, f.TenantA, f.Application); err != nil || secret != "secret-1" {
		t.Fatalf("CredentialSecret after metadata update = (%q, %v)", secret, err)
	}
	rotated := "secret-2"
	if err := f.Connections.Update(ctx, connection, &rotated); err != nil {
		t.Fatalf("Update with secret: %v", err)
	}
	if secret, err := f.Connections.CredentialSecret(ctx, f.TenantA, f.Application); err != nil || secret != rotated {
		t.Fatalf("CredentialSecret after rotation = (%q, %v)", secret, err)
	}
	listed, err := f.Connections.ListAll(ctx, f.TenantA)
	if err != nil || len(listed) != 1 || listed[0].ApplicationID != f.Application {
		t.Fatalf("ListAll = (%+v, %v)", listed, err)
	}
	tenants, err := f.Connections.ListTenantsWithActiveConnections(ctx)
	if err != nil || !slices.Contains(tenants, f.TenantA) || slices.Contains(tenants, f.TenantB) {
		t.Fatalf("ListTenantsWithActiveConnections = (%v, %v)", tenants, err)
	}
	runQuarantineTasks(t, f)
	runConnectionDeletion(t, f, connection)
	if err := f.Connections.Delete(ctx, f.TenantA, f.Application); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if deleted, err := f.Connections.Find(ctx, f.TenantA, f.Application); err != nil || deleted != nil {
		t.Fatalf("Find after Delete = (%+v, %v)", deleted, err)
	}
}
