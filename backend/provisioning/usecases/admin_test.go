package usecases_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	memory "github.com/ambi/idmagic/backend/provisioning/db_memory"
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

// RFC7643-OUT-EXTERNAL-ID: 既定の属性対応付けは、IdMagic 側の識別子を作成時だけ
// `externalId` として送る。以後の相関は保存した対応関係が担うので、更新では送り直さない。
//
// RFC7643-OUT-SCHEMA-EXTENSIONS: 既定の対応付けは拡張スキーマの属性を 1 件も含まない。
func TestRegisterConnection_DefaultMappingSendsExternalIdOnCreateOnly(t *testing.T) {
	deps, _, _ := newAdminDeps()
	conn, err := usecases.RegisterConnection(context.Background(), deps, usecases.RegisterConnectionInput{
		TenantID: "tenant-a", ApplicationID: "app-1", BaseURL: "https://downstream.example.com/scim/v2",
		Credential: domain.ProvisioningCredentialInput{AuthMethod: domain.AuthBearerToken, BearerToken: "tok"},
		Now:        time.Now(),
	})
	if err != nil {
		t.Fatalf("RegisterConnection() error = %v", err)
	}

	var externalID *domain.AttributeMappingRule
	for i, rule := range conn.AttributeMappings {
		if rule.TargetPath == "externalId" {
			externalID = &conn.AttributeMappings[i]
		}
		if strings.Contains(rule.TargetPath, "urn:ietf:params:scim:schemas:extension:") {
			t.Errorf("既定の対応付けが拡張スキーマの属性 %q を含んでいる", rule.TargetPath)
		}
	}
	if externalID == nil {
		t.Fatal("既定の対応付けに externalId が無い")
	}
	if externalID.SourceKind != domain.SourceKindAttribute || externalID.SourceKey != "id" {
		t.Errorf("externalId の供給元 = %v/%q, want attribute/id", externalID.SourceKind, externalID.SourceKey)
	}
	if externalID.ApplyOn != domain.ApplyCreateOnly {
		t.Errorf("externalId の apply_on = %v, want create_only", externalID.ApplyOn)
	}
	if !externalID.Required {
		t.Error("externalId は required でなければならない。相関の起点が欠けたまま作成が通る")
	}
}

// wi-441: 全体再同期は push_groups が対象にする Group も含める。User だけを
// 再同期する実装では、能力を有効にした接続の Group が下流と乖離したまま残る。
func TestStartFullResync_CoversTheGroupsPushGroupsTargets(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()

	// 明示指定は一覧の引き当てを要さないので、GroupRepo が無くても再同期できる。
	setup := func(t *testing.T, flags domain.ProvisioningFeatureFlags, push *domain.GroupPushConfig) (usecases.AdminDeps, *memory.ProvisioningDeliveryRepository) {
		t.Helper()
		deps, connRepo, deliveryRepo := newAdminDeps()
		conn := &domain.ProvisioningConnection{
			ApplicationID: "app-1", TenantID: "tenant-a", Status: domain.ConnectionActive,
			BaseURL:      "https://downstream.example.com/scim/v2",
			Credential:   domain.ProvisioningConnectionCredentialMetadata{CredentialID: "cred", AuthMethod: domain.AuthBearerToken, CreatedAt: now},
			FeatureFlags: flags, GroupPush: push,
			Scope:              domain.ScopeAssignedOnly,
			Matching:           domain.MatchingRule{ConflictMatchAttribute: "userName"},
			DeprovisionPolicy:  domain.DeprovisionPolicy{OnUnassign: domain.DeprovisionDeactivate, OnDelete: domain.DeprovisionDeactivate},
			RateLimitPerMinute: 60, MaxAttempts: 8, QuarantineAfterConsecutiveFailure: 10,
			Health: domain.HealthOK, CreatedAt: now, UpdatedAt: now,
		}
		if err := connRepo.Register(ctx, conn, "tok"); err != nil {
			t.Fatal(err)
		}
		return deps, deliveryRepo
	}

	groupDeliveries := func(t *testing.T, repo *memory.ProvisioningDeliveryRepository) []string {
		t.Helper()
		deliveries, err := repo.ListByConnection(ctx, "tenant-a", "app-1", nil, 50)
		if err != nil {
			t.Fatal(err)
		}
		var ids []string
		for _, d := range deliveries {
			if d.SourceType == domain.SourceTypeGroup {
				ids = append(ids, d.SourceID)
			}
		}
		return ids
	}

	t.Run("明示指定の Group を再同期する", func(t *testing.T) {
		deps, deliveryRepo := setup(t, domain.ProvisioningFeatureFlags{PushGroups: true},
			&domain.GroupPushConfig{Selection: domain.GroupSelectionExplicit, ExplicitGroupIDs: []string{"g1", "g2"}})
		if _, err := usecases.StartFullResync(ctx, deps, "tenant-a", "app-1", now); err != nil {
			t.Fatalf("StartFullResync() error = %v", err)
		}
		if got := groupDeliveries(t, deliveryRepo); len(got) != 2 {
			t.Fatalf("Group の配信 = %v, want g1 と g2", got)
		}
	})

	t.Run("push_groups が無効なら Group は再同期しない", func(t *testing.T) {
		// 対象の設定は残したまま機能フラグだけを落とす。2 つの番人を分ける。
		deps, deliveryRepo := setup(t, domain.ProvisioningFeatureFlags{PushGroups: false},
			&domain.GroupPushConfig{Selection: domain.GroupSelectionExplicit, ExplicitGroupIDs: []string{"g1"}})
		if _, err := usecases.StartFullResync(ctx, deps, "tenant-a", "app-1", now); err != nil {
			t.Fatalf("StartFullResync() error = %v", err)
		}
		if got := groupDeliveries(t, deliveryRepo); len(got) != 0 {
			t.Fatalf("Group の配信 = %v, want none", got)
		}
	})

	t.Run("対象の設定が無いなら Group は再同期しない", func(t *testing.T) {
		// 機能フラグは有効のまま、対象の設定だけを落とす。fail-closed。
		deps, deliveryRepo := setup(t, domain.ProvisioningFeatureFlags{PushGroups: true}, nil)
		if _, err := usecases.StartFullResync(ctx, deps, "tenant-a", "app-1", now); err != nil {
			t.Fatalf("StartFullResync() error = %v", err)
		}
		if got := groupDeliveries(t, deliveryRepo); len(got) != 0 {
			t.Fatalf("Group の配信 = %v, want none", got)
		}
	})
}
