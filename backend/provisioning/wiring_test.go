package provisioning_test

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"

	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	"github.com/ambi/idmagic/backend/provisioning"
	provisioningdomain "github.com/ambi/idmagic/backend/provisioning/internal/domain"
	provisioningports "github.com/ambi/idmagic/backend/provisioning/internal/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type groupPushTarget struct {
	provisioningports.ProvisioningTargetClient
	attributes map[string]any
	groupID    string
	operation  string
	members    []string
}

func (c *groupPushTarget) CreateGroup(_ context.Context, _ []provisioningdomain.AttributeMappingRule, attributes map[string]any) (string, *string, error) {
	c.attributes = attributes
	return "remote-group", nil, nil
}

func (c *groupPushTarget) PatchGroupMembers(_ context.Context, groupID, operation string, members []string) error {
	c.groupID, c.operation, c.members = groupID, operation, members
	return nil
}

//spec:covers RFC7643-OUT-GROUP-RESOURCES: worker の組み立ては Group の表示名と、対応関係のある直接メンバーの増分 add を供給し、配送後の対応関係と成功状態を保存する。
func TestTaskHandlerWiresGroupAttributesAndMembers(t *testing.T) {
	ctx := t.Context()
	now := time.Now().UTC()
	m := provisioning.NewMemoryModule()
	groups := idmanagement.NewMemoryModule().GroupRepo
	if err := groups.Save(ctx, &groupdomain.Group{ID: "group", TenantID: "tenant", Name: "Staff", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := groups.AddMember(ctx, &groupdomain.GroupMember{GroupID: "group", UserID: "user", CreatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if err := m.ConnectionRepo.Register(ctx, &provisioningdomain.ProvisioningConnection{
		ApplicationID: "application", TenantID: "tenant", Status: provisioningdomain.ConnectionActive,
		BaseURL: "https://downstream.example/scim", Credential: provisioningdomain.ProvisioningConnectionCredentialMetadata{CredentialID: "credential", AuthMethod: provisioningdomain.AuthBearerToken, CreatedAt: now},
		Scope: provisioningdomain.ScopeAssignedOnly, Matching: provisioningdomain.MatchingRule{ConflictMatchAttribute: "userName"},
		DeprovisionPolicy:  provisioningdomain.DeprovisionPolicy{OnUnassign: provisioningdomain.DeprovisionDeactivate, OnDelete: provisioningdomain.DeprovisionDeactivate},
		RateLimitPerMinute: 60, MaxAttempts: 8, QuarantineAfterConsecutiveFailure: 10, Health: provisioningdomain.HealthOK,
		FeatureFlags: provisioningdomain.ProvisioningFeatureFlags{PushGroups: true}, CreatedAt: now, UpdatedAt: now,
	}, "secret"); err != nil {
		t.Fatal(err)
	}
	if err := m.RemoteLinkRepo.Upsert(ctx, &provisioningdomain.RemoteResourceLink{ConnectionID: "application", TenantID: "tenant", SourceType: provisioningdomain.SourceTypeUser, SourceID: "user", RemoteID: "remote-user", LastSyncedVersion: 1, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.TaskRepo.Save(ctx, &provisioningdomain.ProvisioningTask{ID: "task", TenantID: "tenant", ConnectionID: "application", SourceType: provisioningdomain.SourceTypeGroup, SourceID: "group", SourceVersion: 1, Operation: provisioningdomain.OperationCreate, Status: provisioningdomain.TaskInFlight, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	client := &groupPushTarget{}
	newClient := func(connection *provisioningdomain.ProvisioningConnection, secret string) (provisioningports.ProvisioningTargetClient, error) {
		if connection.ApplicationID != "application" || secret != "secret" {
			t.Fatalf("client inputs = %s, %q", connection.ApplicationID, secret)
		}
		return client, nil
	}
	handler := m.TaskHandler(idmanagement.NewMemoryModule().UserRepo, groups, newClient, func(spec.DomainEvent) {}, nil)
	if _, err := handler(ctx, &jobsdomain.Job{ID: "job", TenantID: "tenant", Params: json.RawMessage(`{"task_id":"task"}`), Attempts: 1, MaxAttempts: 8}); err != nil {
		t.Fatal(err)
	}
	if client.attributes["display_name"] != "Staff" || client.groupID != "remote-group" || client.operation != "add" || !slices.Equal(client.members, []string{"remote-user"}) {
		t.Fatalf("group push = %+v, want Staff and add remote-user to remote-group", client)
	}
	link, err := m.RemoteLinkRepo.Find(ctx, "application", provisioningdomain.SourceTypeGroup, "group")
	if err != nil || link == nil || link.RemoteID != "remote-group" {
		t.Fatalf("stored link = %+v, %v", link, err)
	}
	task, err := m.TaskRepo.Find(ctx, "tenant", "task")
	if err != nil || task == nil || task.Status != provisioningdomain.TaskSucceeded {
		t.Fatalf("stored task = %+v, %v", task, err)
	}
}
