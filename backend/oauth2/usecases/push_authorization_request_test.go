package usecases

import (
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2/domain"

	"github.com/ambi/idmagic/backend/shared/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestPushAuthorizationRequest(t *testing.T) {
	ctx := tenantContext(spec.DefaultTenantID)
	clientRepo := memory.NewClientRepository()
	parStore := memory.NewPARStore()
	authzDetailRepo := memory.NewAuthorizationDetailTypeRepository()

	var emitted []spec.DomainEvent
	emitFunc := func(e spec.DomainEvent) {
		emitted = append(emitted, e)
	}

	deps := PARDeps{
		ClientRepo:          clientRepo,
		Store:               parStore,
		AuthzDetailTypeRepo: authzDetailRepo,
		Emit:                emitFunc,
	}

	now := time.Now().UTC()

	// テスト用クライアントを登録
	client := &domain.OAuth2Client{
		TenantID:      spec.DefaultTenantID,
		ClientID:      "client-1",
		RedirectURIs:  []string{"https://example.com/cb"},
		GrantTypes:    []spec.GrantType{spec.GrantAuthorizationCode},
		ResponseTypes: []spec.ResponseType{spec.ResponseTypeCode},
	}
	clientRepo.Seed(client)

	t.Run("Succeeds", func(t *testing.T) {
		emitted = nil
		in := PARInput{
			ClientID: "client-1",
			Parameters: map[string]string{
				"response_type": "code",
				"scope":         "openid",
			},
		}

		res, err := PushAuthorizationRequest(ctx, deps, in, now)
		if err != nil {
			t.Fatal(err)
		}
		if res.ExpiresIn != 90 || res.RequestURI == "" {
			t.Errorf("unexpected result: %+v", res)
		}

		if len(emitted) != 1 {
			t.Fatalf("expected 1 event, got %d", len(emitted))
		}
		ev, ok := emitted[0].(*spec.PARStored)
		if !ok {
			t.Fatalf("expected PARStored, got %T", emitted[0])
		}
		if ev.ClientID != "client-1" || ev.RequestURI != res.RequestURI {
			t.Errorf("mismatched event content: %+v", ev)
		}

		// 保存されていることの検証
		saved, err := parStore.Consume(ctx, res.RequestURI)
		if err != nil {
			t.Fatal(err)
		}
		if saved.ClientID != "client-1" {
			t.Errorf("expected ClientID 'client-1', got %q", saved.ClientID)
		}
	})

	t.Run("ClientNotFound", func(t *testing.T) {
		in := PARInput{
			ClientID: "client-none",
		}
		_, err := PushAuthorizationRequest(ctx, deps, in, now)
		var oerr *OAuthError
		if !errors.As(err, &oerr) || oerr.Code != "invalid_client" {
			t.Errorf("expected invalid_client error, got %v", err)
		}
	})

	t.Run("AuthorizationDetailsParseError", func(t *testing.T) {
		in := PARInput{
			ClientID: "client-1",
			Parameters: map[string]string{
				"authorization_details": "not-valid-json",
			},
		}
		_, err := PushAuthorizationRequest(ctx, deps, in, now)
		if err == nil {
			t.Error("expected error for invalid JSON details, got nil")
		}
	})

	t.Run("AuthorizationDetailsValidationError", func(t *testing.T) {
		// タイプ登録なしで details を検証させると失敗する
		in := PARInput{
			ClientID: "client-1",
			Parameters: map[string]string{
				"authorization_details": `[{"type":"payment","actions":["read"]}]`,
			},
		}
		_, err := PushAuthorizationRequest(ctx, deps, in, now)
		if err == nil {
			t.Error("expected validation error for unregistered type, got nil")
		}
	})

	t.Run("ZeroTimeHandling", func(t *testing.T) {
		in := PARInput{
			ClientID: "client-1",
		}
		_, err := PushAuthorizationRequest(ctx, deps, in, time.Time{})
		if err != nil {
			t.Fatal(err)
		}
	})
}
