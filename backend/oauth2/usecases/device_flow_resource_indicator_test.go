package usecases

// ADR-055/wi-264: RFC 8707 resource indicator を device_code グラントへ拡張する。
// M2M/CLI パターンでも 1 トークン = 1 McpResourceServer の audience 限定を
// fail-closed で強制する。既存の resource 未指定クライアントの挙動は無変更。

import (
	"context"
	"testing"
	"time"

	oauth2memory "github.com/ambi/idmagic/backend/oauth2/adapters/persistence/memory"
	"github.com/ambi/idmagic/backend/oauth2/domain"
)

// newApprovedDeviceFixture は device_flow_test.go の newDeviceFixture と同様の
// client/user を持ち、user_code 承認済みの device_code をあらかじめ用意する。
func newApprovedDeviceFixture(t *testing.T, mcpResourceServerRepo *oauth2memory.McpResourceServerRepository) (deviceFixture, string) {
	t.Helper()
	f := newDeviceFixture()
	f.deps.McpResourceServerRepo = mcpResourceServerRepo

	now := time.Now().UTC()
	out, err := RequestDeviceAuthorization(context.Background(), f.requestDeps, DeviceAuthorizationInput{
		ClientID: "device-client", Scope: "openid",
	}, now)
	if err != nil {
		t.Fatalf("RequestDeviceAuthorization: %v", err)
	}
	if err := ApproveUserCode(context.Background(), f.verifyDeps, out.UserCode, "user", now); err != nil {
		t.Fatalf("ApproveUserCode: %v", err)
	}
	return f, out.DeviceCode
}

func TestExchangeDeviceCode_unregisteredResource_rejectedAsInvalidTarget(t *testing.T) {
	f, deviceCode := newApprovedDeviceFixture(t, oauth2memory.NewMcpResourceServerRepository())
	_, err := ExchangeDeviceCode(context.Background(), f.deps, ExchangeDeviceCodeInput{
		ClientID: "device-client", DeviceCode: deviceCode,
		Resource: []string{"https://mcp.example.com/unknown"},
	}, time.Now().UTC())
	assertOAuthError(t, err, "invalid_target")
}

func TestExchangeDeviceCode_registeredResource_boundAudienceAndRefreshRecord(t *testing.T) {
	repo := oauth2memory.NewMcpResourceServerRepository()
	repo.Seed(&domain.McpResourceServer{
		ResourceServerID: "rs-1", Resource: "https://mcp.example.com/tools",
		Name: "Tools", Scopes: []string{"openid"}, State: domain.McpResourceServerActive,
	})
	f, deviceCode := newApprovedDeviceFixture(t, repo)

	out, err := ExchangeDeviceCode(context.Background(), f.deps, ExchangeDeviceCodeInput{
		ClientID: "device-client", DeviceCode: deviceCode,
		Resource: []string{"https://mcp.example.com/tools"},
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	issuer := f.deps.TokenIssuer.(*fakeTokenIssuer)
	if len(issuer.lastAccessTokenInput.Audiences) != 1 || issuer.lastAccessTokenInput.Audiences[0] != "https://mcp.example.com/tools" {
		t.Fatalf("expected audience to be bound to resource, got %v", issuer.lastAccessTokenInput.Audiences)
	}

	// 発行された refresh token record にも resource が伝播しているはず (rotation で保持するため)。
	hash := domain.HashRefreshToken(out.RefreshToken)
	rec, err := f.deps.RefreshStore.FindByHash(context.Background(), hash)
	if err != nil || rec == nil {
		t.Fatalf("expected refresh token record to be findable: %v", err)
	}
	if rec.Resource == nil || *rec.Resource != "https://mcp.example.com/tools" {
		t.Fatalf("expected refresh token record to carry resource, got %v", rec.Resource)
	}
}

func TestExchangeDeviceCode_noResource_unaffected(t *testing.T) {
	f, deviceCode := newApprovedDeviceFixture(t, oauth2memory.NewMcpResourceServerRepository())
	out, err := ExchangeDeviceCode(context.Background(), f.deps, ExchangeDeviceCodeInput{
		ClientID: "device-client", DeviceCode: deviceCode,
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	issuer := f.deps.TokenIssuer.(*fakeTokenIssuer)
	if len(issuer.lastAccessTokenInput.Audiences) != 0 {
		t.Fatalf("expected no explicit audience binding, got %v", issuer.lastAccessTokenInput.Audiences)
	}
	if out.AccessToken == "" {
		t.Fatal("access token missing")
	}
}
