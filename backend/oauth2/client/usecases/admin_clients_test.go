package usecases

import (
	"errors"
	"testing"
	"time"

	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

// TestDeleteAdminOAuth2Client_decrementsQuotaUsage is a wi-160 T004.5 RED
// test: deleting a client must free its oauth2_clients quota slot so a
// subsequent registration at the same limit succeeds.
func TestDeleteAdminOAuth2Client_decrementsQuotaUsage(t *testing.T) {
	ctx := tenantContext(tenancydomain.DefaultTenantID)
	clientRepo := oauth2memory.NewClientRepository()
	quotaRepo := tenancymemory.NewQuotaRepository()
	limit := 1
	if err := quotaRepo.SetQuota(ctx, tenancydomain.DefaultTenantID, &tenancydomain.TenantQuota{OAuth2Clients: &limit}); err != nil {
		t.Fatalf("SetQuota: %v", err)
	}
	deps := AdminOAuth2ClientDeps{ClientRepo: clientRepo, QuotaRepo: quotaRepo}
	now := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	regIn := CreateAdminOAuth2ClientInput{
		ActorUserID: "admin-1", Now: now,
		Registration: RegisterClientInput{
			ClientName: "Client A", RedirectURIs: []string{"https://example.com/cb"},
			GrantTypes: []spec.GrantType{spec.GrantAuthorizationCode}, ResponseTypes: []spec.ResponseType{spec.ResponseTypeCode},
		},
	}
	res, err := CreateAdminOAuth2Client(ctx, deps, regIn)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := DeleteAdminOAuth2Client(ctx, deps, "admin-1", res.Client.ClientID, now); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := CreateAdminOAuth2Client(ctx, deps, regIn); err != nil {
		t.Fatalf("expected create to succeed after delete freed quota, got %v", err)
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
		ClientRepo: clientRepo,
		Emit:       emitFunc,
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
	})

	t.Run("adminNow with zero time", func(t *testing.T) {
		// zero time の場合のカバー
		got := adminNow(time.Time{})
		if got.IsZero() {
			t.Error("expected non-zero time for ZeroTime input")
		}
	})
}
