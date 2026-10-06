package usecases

// 主要ユースケース追跡: REQ-OAUTH2-027。

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type deviceFixture struct {
	deps        ExchangeDeviceCodeDeps
	requestDeps DeviceAuthorizationDeps
	verifyDeps  VerifyUserCodeDeps
}

func newDeviceFixture() deviceFixture {
	clientRepo := oauth2memory.NewClientRepository()
	userRepo := usermemory.NewUserRepository()
	deviceStore := oauth2memory.NewDeviceCodeStore()
	refreshStore := oauth2memory.NewRefreshTokenStore()
	now := time.Now().UTC()
	clientRepo.Seed(&domain.OAuth2Client{
		ClientID: "device-client", ClientType: spec.ClientPublic,
		RedirectURIs: []string{"https://device.example/cb"},
		GrantTypes:   []spec.GrantType{spec.GrantDeviceCode, spec.GrantRefreshToken},
		ResponseTypes: []spec.ResponseType{
			spec.ResponseTypeCode,
		},
		TokenEndpointAuthMethod:  domain.AuthMethodNone,
		Scope:                    "openid profile offline_access",
		IDTokenSignedResponseAlg: signingdomain.SigAlgPS256,
		FapiProfile:              domain.FapiNone,
		CreatedAt:                now,
	})
	userRepo.Seed(&userdomain.User{
		ID: "user", PreferredUsername: "alice", PasswordHash: "hash",
		CreatedAt: now, UpdatedAt: now,
	})
	return deviceFixture{
		requestDeps: DeviceAuthorizationDeps{
			ClientRepo: clientRepo, DeviceCodeStore: deviceStore,
			BaseVerification: "https://idp.example/device",
		},
		verifyDeps: VerifyUserCodeDeps{DeviceCodeStore: deviceStore},
		deps: ExchangeDeviceCodeDeps{
			ClientRepo: clientRepo, UserRepo: userRepo, DeviceCodeStore: deviceStore,
			RefreshStore: refreshStore, TokenIssuer: &fakeTokenIssuer{},
		},
	}
}

func TestDeviceFlowPollingAndReplay(t *testing.T) {
	f := newDeviceFixture()
	t0 := time.Now().UTC()
	auth, err := RequestDeviceAuthorization(
		context.Background(),
		f.requestDeps,
		DeviceAuthorizationInput{ClientID: "device-client", Scope: "openid"},
		t0,
	)
	if err != nil {
		t.Fatal(err)
	}
	input := ExchangeDeviceCodeInput{ClientID: "device-client", DeviceCode: auth.DeviceCode}
	if _, err := ExchangeDeviceCode(context.Background(), f.deps, input, t0); oauthErrorCode(err) != "authorization_pending" {
		t.Fatalf("first poll: %v", err)
	}
	if _, err := ExchangeDeviceCode(context.Background(), f.deps, input, t0.Add(time.Second)); oauthErrorCode(err) != "slow_down" {
		t.Fatalf("fast poll: %v", err)
	}
	if err := ApproveUserCode(
		context.Background(), f.verifyDeps, auth.UserCode, "user", t0.Add(2*time.Second),
	); err != nil {
		t.Fatal(err)
	}
	out, err := ExchangeDeviceCode(context.Background(), f.deps, input, t0.Add(11*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if out.AccessToken == "" || out.IDToken == "" || out.RefreshToken == "" {
		t.Fatal("device exchange did not issue tokens")
	}
	if _, err := ExchangeDeviceCode(context.Background(), f.deps, input, t0.Add(20*time.Second)); oauthErrorCode(err) != "invalid_grant" {
		t.Fatalf("replay: %v", err)
	}
}

func TestDeviceAuthorizationRejectsUndeclaredScope(t *testing.T) {
	f := newDeviceFixture()
	_, err := RequestDeviceAuthorization(
		context.Background(),
		f.requestDeps,
		DeviceAuthorizationInput{ClientID: "device-client", Scope: "openid admin"},
		time.Now(),
	)
	if oauthErrorCode(err) != "invalid_scope" {
		t.Fatalf("got %v", err)
	}
}

func TestDeviceFlowDeny(t *testing.T) {
	f := newDeviceFixture()
	t0 := time.Now().UTC()
	ctx := tenantContext(tenancydomain.DefaultTenantID)

	auth, err := RequestDeviceAuthorization(
		ctx,
		f.requestDeps,
		DeviceAuthorizationInput{ClientID: "device-client", Scope: "openid"},
		t0,
	)
	if err != nil {
		t.Fatal(err)
	}

	// 正常に拒否
	var emitted []spec.DomainEvent
	f.verifyDeps.Emit = func(e spec.DomainEvent) {
		emitted = append(emitted, e)
	}
	err = DenyUserCode(ctx, f.verifyDeps, auth.UserCode, "user", t0)
	if err != nil {
		t.Fatal(err)
	}

	// 状態が Denied であることの確認
	rec, _ := f.verifyDeps.DeviceCodeStore.FindByUserCode(ctx, domain.NormalizeUserCode(auth.UserCode))
	if rec.State != spec.DeviceFlowDenied {
		t.Errorf("expected state DeviceFlowDenied, got %v", rec.State)
	}
	if len(emitted) != 1 {
		t.Fatalf("expected 1 event, got %d", len(emitted))
	}

	// 存在しないUserCodeの拒否
	err = DenyUserCode(ctx, f.verifyDeps, "ABCD-EFGH", "user", t0)
	if oauthErrorCode(err) != "invalid_request" {
		t.Errorf("expected invalid_request error, got %v", err)
	}

	// すでに拒否済みのものを再度拒否しようとすると遷移エラー
	err = DenyUserCode(ctx, f.verifyDeps, auth.UserCode, "user", t0)
	if oauthErrorCode(err) != "invalid_request" {
		t.Errorf("expected invalid_request error, got %v", err)
	}

	// ZeroTime での呼び出しパス
	err = DenyUserCode(ctx, f.verifyDeps, "ABCD-EFGH", "user", time.Time{})
	if err == nil {
		t.Error("expected error for non-existent code, got nil")
	}
}

// approveDeviceCode は scope でデバイス認可を要求し、時刻 at に承認した device_code を返す。
func approveDeviceCode(t *testing.T, f deviceFixture, scope string, at time.Time) string {
	t.Helper()
	auth, err := RequestDeviceAuthorization(context.Background(), f.requestDeps, DeviceAuthorizationInput{ClientID: "device-client", Scope: scope}, at)
	if err != nil {
		t.Fatal(err)
	}
	if err := ApproveUserCode(context.Background(), f.verifyDeps, auth.UserCode, "user", at); err != nil {
		t.Fatal(err)
	}
	return auth.DeviceCode
}

func TestCharacterizeExchangeDeviceCodeRefreshToken(t *testing.T) {
	for _, scope := range []string{"openid", "openid offline_access"} {
		t.Run(scope, func(t *testing.T) {
			f := newDeviceFixture()
			var emitted []spec.DomainEvent
			f.deps.Emit = func(e spec.DomainEvent) { emitted = append(emitted, e) }
			now := time.Now().UTC()
			deviceCode := approveDeviceCode(t, f, scope, now)

			out, err := ExchangeDeviceCode(context.Background(), f.deps, ExchangeDeviceCodeInput{ClientID: "device-client", DeviceCode: deviceCode}, now)
			if err != nil {
				t.Fatal(err)
			}
			if out.RefreshToken == "" {
				t.Fatal("refresh token missing")
			}
			stored, err := f.deps.RefreshStore.FindByHash(context.Background(), domain.HashRefreshToken(out.RefreshToken))
			if err != nil || stored == nil {
				t.Fatalf("refresh record: %v %v", stored, err)
			}
			if stored.ClientID != "device-client" || stored.UserID != "user" || strings.Join(stored.Scopes, " ") != scope {
				t.Fatalf("refresh record = %+v", stored)
			}
			var types []string
			for _, e := range emitted {
				types = append(types, e.EventType())
				if issued, ok := e.(*domain.RefreshTokenIssued); ok && (issued.FamilyID != stored.FamilyID || issued.TokenID != stored.ID) {
					t.Fatalf("RefreshTokenIssued = %+v, record = %+v", issued, stored)
				}
			}
			if strings.Join(types, ",") != "AccessTokenIssued,RefreshTokenIssued" {
				t.Fatalf("events = %v", types)
			}
			rec, _ := f.deps.DeviceCodeStore.FindByDeviceCodeHash(context.Background(), domain.HashDeviceCode(deviceCode))
			if rec.State != spec.DeviceFlowExchanged || rec.IssuedFamilyID == nil || *rec.IssuedFamilyID != stored.FamilyID {
				t.Fatalf("device record = %+v", rec)
			}
		})
	}
}

func TestCharacterizeDenyUserCode(t *testing.T) {
	cases := []struct {
		name  string
		after time.Duration
	}{
		{"within the lifetime", time.Second},
		{"after the lifetime", domain.DeviceCodeTTL + time.Second},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newDeviceFixture()
			ctx := tenantContext(tenancydomain.DefaultTenantID)
			t0 := time.Now().UTC()
			auth, err := RequestDeviceAuthorization(ctx, f.requestDeps, DeviceAuthorizationInput{ClientID: "device-client", Scope: "openid"}, t0)
			if err != nil {
				t.Fatal(err)
			}
			var emitted []spec.DomainEvent
			f.verifyDeps.Emit = func(e spec.DomainEvent) { emitted = append(emitted, e) }

			err = DenyUserCode(ctx, f.verifyDeps, auth.UserCode, "user", t0.Add(tc.after))
			if err != nil {
				t.Fatal(err)
			}
			rec, _ := f.verifyDeps.DeviceCodeStore.FindByUserCode(ctx, domain.NormalizeUserCode(auth.UserCode))
			if rec.State != spec.DeviceFlowDenied {
				t.Fatalf("state = %v", rec.State)
			}
			if len(emitted) != 1 || emitted[0].EventType() != "DeviceAuthorizationDenied" {
				t.Fatalf("events = %v", emitted)
			}
		})
	}
}

func oauthErrorCode(err error) string {
	if oauthErr, ok := errors.AsType[*OAuthError](err); ok {
		return oauthErr.Code
	}
	return ""
}
