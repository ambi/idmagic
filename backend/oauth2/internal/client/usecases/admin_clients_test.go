package usecases

// 主要ユースケース追跡: REQ-OAUTH2-035。

import (
	"context"
	"errors"
	"testing"
	"time"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"

	clientports "github.com/ambi/idmagic/backend/oauth2/client/ports"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/internal/db_memory"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestDeleteAdminOAuth2ClientRejectsApplicationOwnedClient(t *testing.T) {
	ctx := tenantContext("acme")
	repo := oauth2memory.NewClientRepository()
	repo.Seed(&domain.OAuth2Client{
		TenantID: "acme", ClientID: "client-1", ApplicationID: "app-1",
		ClientType: spec.ClientConfidential, GrantTypes: []spec.GrantType{spec.GrantClientCredentials},
		TokenEndpointAuthMethod: domain.AuthMethodClientSecretBasic,
	})

	err := DeleteAdminOAuth2Client(ctx, AdminOAuth2ClientDeps{ClientRepo: repo, ClientDeletion: repositoryClientDeletion{repo: repo}}, "admin", "client-1", time.Now())
	if !errors.Is(err, ErrProtocolOwnedByApplication) {
		t.Fatalf("DeleteAdminOAuth2Client() error = %v, want ErrProtocolOwnedByApplication", err)
	}
}

func TestAdminOAuth2Client(t *testing.T) {
	ctx := tenantContext(tenancydomain.DefaultTenantID)
	clientRepo := oauth2memory.NewClientRepository()
	var emitted []spec.DomainEvent
	emitFunc := func(e spec.DomainEvent) {
		emitted = append(emitted, e)
	}

	deps := AdminOAuth2ClientDeps{
		ClientRepo:     clientRepo,
		ClientDeletion: repositoryClientDeletion{repo: clientRepo},
		Emit:           emitFunc,
	}

	now := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)

	t.Run("CreateAdminOAuth2Client", func(t *testing.T) {
		emitted = nil
		in := CreateAdminOAuth2ClientInput{
			ActorUserID: "admin-1",
			Now:         now,
			Registration: RegisterClientInput{
				ClientName:    "Client A",
				RedirectURIs:  []string{"https://example.com/cb"},
				GrantTypes:    []spec.GrantType{spec.GrantAuthorizationCode},
				ResponseTypes: []spec.ResponseType{spec.ResponseTypeCode},
			},
		}

		result, err := CreateAdminOAuth2Client(ctx, deps, in)
		if err != nil {
			t.Fatal(err)
		}
		if result.Client.ClientName == nil || *result.Client.ClientName != "Client A" {
			t.Errorf("expected client name 'Client A', got %v", result.Client.ClientName)
		}

		if len(emitted) != 1 {
			t.Fatalf("expected 1 event, got %d", len(emitted))
		}
		ev, ok := emitted[0].(*domain.AdminOAuth2ClientCreated)
		if !ok {
			t.Fatalf("expected AdminOAuth2ClientCreated event, got %T", emitted[0])
		}
		if ev.ActorUserID != "admin-1" || ev.ClientID != result.Client.ClientID {
			t.Errorf("unexpected event content: %+v", ev)
		}
	})

	t.Run("UpdateAdminOAuth2Client", func(t *testing.T) {
		// まず登録
		regIn := CreateAdminOAuth2ClientInput{
			ActorUserID: "admin-1",
			Now:         now,
			Registration: RegisterClientInput{
				ClientName:    "Client Update Test",
				RedirectURIs:  []string{"https://example.com/cb"},
				GrantTypes:    []spec.GrantType{spec.GrantAuthorizationCode},
				ResponseTypes: []spec.ResponseType{spec.ResponseTypeCode},
			},
		}
		res, err := CreateAdminOAuth2Client(ctx, deps, regIn)
		if err != nil {
			t.Fatalf("create failed: %v", err)
		}
		clientID := res.Client.ClientID

		// 正常系更新
		emitted = nil
		nameUpdate := "Client Updated Name"
		redirectURIs := []string{"https://example.com/callback"}
		grantTypes := []spec.GrantType{spec.GrantAuthorizationCode}
		responseTypes := []spec.ResponseType{spec.ResponseTypeCode}
		scopeUpdate := "openid profile"
		requirePAR := true
		dpopBound := true

		upIn := UpdateAdminOAuth2ClientInput{
			ActorUserID:     "admin-2",
			ClientID:        clientID,
			ClientName:      &nameUpdate,
			RedirectURIs:    &redirectURIs,
			GrantTypes:      &grantTypes,
			ResponseTypes:   &responseTypes,
			Scope:           &scopeUpdate,
			RequirePAR:      &requirePAR,
			DpopBoundTokens: &dpopBound,
			Now:             now,
		}

		updated, err := UpdateAdminOAuth2Client(ctx, deps, upIn)
		if err != nil {
			t.Fatal(err)
		}
		if *updated.ClientName != "Client Updated Name" || updated.RedirectURIs[0] != "https://example.com/callback" {
			t.Errorf("update fields mismatch: %+v", updated)
		}

		if len(emitted) != 1 {
			t.Fatalf("expected 1 event, got %d", len(emitted))
		}
		ev, ok := emitted[0].(*domain.AdminOAuth2ClientUpdated)
		if !ok {
			t.Fatalf("expected AdminOAuth2ClientUpdated event, got %T", emitted[0])
		}
		if ev.ActorUserID != "admin-2" || ev.ClientID != clientID {
			t.Errorf("unexpected event content: %+v", ev)
		}

		// 別テナントからは同じ client を更新できない。所有権の確認が抜けると、
		// テナント境界を越えた設定変更がそのまま通ってしまう。
		otherTenant := tenantports.WithTenant(ctx, &tenancydomain.Tenant{
			ID: "another-tenant", DisplayName: "another-tenant",
			Status: tenancydomain.TenantStatusActive, CreatedAt: now,
		}, "https://idp.example/realms/another-tenant", "/realms/another-tenant")
		if _, err := UpdateAdminOAuth2Client(otherTenant, deps, upIn); !errors.Is(err, ErrClientNotFound) {
			t.Fatalf("cross-tenant update err=%v, want ErrClientNotFound", err)
		}
		afterCrossTenant, err := clientRepo.FindByID(ctx, tenancydomain.DefaultTenantID, clientID)
		if err != nil {
			t.Fatal(err)
		}
		if afterCrossTenant == nil || afterCrossTenant.ClientName == nil || *afterCrossTenant.ClientName != "Client Updated Name" {
			t.Fatalf("cross-tenant refusal changed the client: %+v", afterCrossTenant)
		}
		if len(emitted) != 1 {
			t.Fatalf("cross-tenant refusal emitted events: %+v", emitted)
		}

		// 値が同じ場合（no-op）
		emitted = nil
		_, err = UpdateAdminOAuth2Client(ctx, deps, upIn)
		if err != nil {
			t.Fatal(err)
		}
		if len(emitted) != 0 {
			t.Errorf("expected 0 events for no-op, got %d", len(emitted))
		}

		// クライアントが存在しない場合
		upIn.ClientID = "non-existent"
		_, err = UpdateAdminOAuth2Client(ctx, deps, upIn)
		if !errors.Is(err, ErrClientNotFound) {
			t.Errorf("expected ErrClientNotFound, got %v", err)
		}

		// 検証エラー (Invalid Redirect URI)
		upIn.ClientID = clientID
		badRedirect := []string{"not-a-valid-uri"}
		upIn.RedirectURIs = &badRedirect
		_, err = UpdateAdminOAuth2Client(ctx, deps, upIn)
		if err == nil {
			t.Error("expected validation error, got nil")
		}
		afterRefusals, err := clientRepo.FindByID(ctx, tenancydomain.DefaultTenantID, clientID)
		if err != nil {
			t.Fatal(err)
		}
		if afterRefusals == nil || afterRefusals.ClientName == nil || *afterRefusals.ClientName != "Client Updated Name" {
			t.Fatalf("rejected updates changed the client: %+v", afterRefusals)
		}
		if len(emitted) != 0 {
			t.Fatalf("rejected updates emitted events: %+v", emitted)
		}
	})

	t.Run("DeleteAdminOAuth2Client", func(t *testing.T) {
		// まず登録
		regIn := CreateAdminOAuth2ClientInput{
			ActorUserID: "admin-1",
			Now:         now,
			Registration: RegisterClientInput{
				ClientName:    "Client Delete Test",
				RedirectURIs:  []string{"https://example.com/cb"},
				GrantTypes:    []spec.GrantType{spec.GrantAuthorizationCode},
				ResponseTypes: []spec.ResponseType{spec.ResponseTypeCode},
			},
		}
		res, err := CreateAdminOAuth2Client(ctx, deps, regIn)
		if err != nil {
			t.Fatalf("create failed: %v", err)
		}
		clientID := res.Client.ClientID

		// 正常削除
		emitted = nil
		err = DeleteAdminOAuth2Client(ctx, deps, "admin-3", clientID, now)
		if err != nil {
			t.Fatal(err)
		}

		if len(emitted) != 1 {
			t.Fatalf("expected 1 event, got %d", len(emitted))
		}
		ev, ok := emitted[0].(*domain.AdminOAuth2ClientDeleted)
		if !ok {
			t.Fatalf("expected AdminOAuth2ClientDeleted event, got %T", emitted[0])
		}
		if ev.ActorUserID != "admin-3" || ev.ClientID != clientID {
			t.Errorf("unexpected event content: %+v", ev)
		}

		// 二重削除 (ErrClientNotFound)
		err = DeleteAdminOAuth2Client(ctx, deps, "admin-3", clientID, now)
		if !errors.Is(err, ErrClientNotFound) {
			t.Errorf("expected ErrClientNotFound, got %v", err)
		}
		deleted, findErr := clientRepo.FindByID(ctx, tenancydomain.DefaultTenantID, clientID)
		if findErr != nil {
			t.Fatal(findErr)
		}
		if deleted != nil || len(emitted) != 1 {
			t.Fatalf("duplicate delete changed state: client=%+v events=%+v", deleted, emitted)
		}
	})

	t.Run("adminNow with zero time", func(t *testing.T) {
		// zero time の場合のカバー
		got := adminNow(time.Time{})
		if got.IsZero() {
			t.Error("expected non-zero time for ZeroTime input")
		}
	})
}

// repositoryClientDeletion は単独の削除の操作のテストダブルである。拒否の判定と行の削除だけを行い、
// 専用状態と使用量は扱わない。それらは oauth2 の削除の操作のテストが確かめる。
type repositoryClientDeletion struct {
	repo clientports.OAuth2ClientRepository
}

func (d repositoryClientDeletion) DeleteStandalone(ctx context.Context, tenantID, clientID string) error {
	client, err := d.repo.FindByID(ctx, tenantID, clientID)
	if err != nil {
		return err
	}
	if client == nil {
		return clientports.ErrCatalogClientNotFound
	}
	if err := clientports.StandaloneDeletionRefusal(client.FirstParty, client.ApplicationID != ""); err != nil {
		return err
	}
	return d.repo.Delete(ctx, tenantID, clientID)
}
