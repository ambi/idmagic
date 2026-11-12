package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	oauth2memory "github.com/ambi/idmagic/backend/oauth2/internal/db_memory"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

func TestRegisterClientHashesSecret(t *testing.T) {
	repo := oauth2memory.NewClientRepository()
	result, err := RegisterClient(testing_tenant.Default(context.Background()), RegisterClientDeps{ClientRepo: repo}, RegisterClientInput{
		ClientType:              spec.ClientConfidential,
		RedirectURIs:            []string{"https://client.example/cb"},
		TokenEndpointAuthMethod: domain.AuthMethodClientSecretBasic,
	}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if result.ClientSecret == "" || result.Client.ClientSecretHash == nil {
		t.Fatal("client secret was not issued")
	}
	if *result.Client.ClientSecretHash == result.ClientSecret {
		t.Fatal("client secret was stored in plaintext")
	}
	if !domain.VerifyClientSecret(result.ClientSecret, *result.Client.ClientSecretHash) {
		t.Fatal("stored client secret hash does not verify")
	}
}

// TestRegisterClient_rejectsWhenHardQuotaExceeded is a wi-160 T004.5 RED test
// for the SCL scenario "Hard Quota を超過したリソース作成は拒否される"
// (docs/modules/tenancy/), applied to the oauth2_clients resource. This
// covers both dynamic client registration and admin client creation since
// CreateAdminOAuth2Client calls RegisterClient internally.
func TestRegisterClient_rejectsWhenHardQuotaExceeded(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	repo := oauth2memory.NewClientRepository()
	quotaRepo := tenancy.NewMemoryModule().QuotaRepo
	limit := 1
	if err := quotaRepo.SetQuota(ctx, tenancydomain.DefaultTenantID, &tenancydomain.TenantQuota{OAuth2Clients: &limit}); err != nil {
		t.Fatalf("SetQuota: %v", err)
	}
	deps := RegisterClientDeps{ClientRepo: repo, QuotaRepo: quotaRepo}
	in := RegisterClientInput{
		ClientType:              spec.ClientConfidential,
		RedirectURIs:            []string{"https://client.example/cb"},
		TokenEndpointAuthMethod: domain.AuthMethodClientSecretBasic,
	}
	if _, err := RegisterClient(ctx, deps, in, time.Now()); err != nil {
		t.Fatalf("first RegisterClient: %v", err)
	}
	_, err := RegisterClient(ctx, deps, in, time.Now())
	var qErr *tenancydomain.QuotaExceededError
	if !errors.As(err, &qErr) {
		t.Fatalf("expected *domain.QuotaExceededError, got %v", err)
	}
	if qErr.Resource != tenancydomain.ResourceOAuth2Clients {
		t.Fatalf("unexpected resource: %s", qErr.Resource)
	}
}

func TestRegisterPrivateKeyJWTRequiresInlineJWKS(t *testing.T) {
	repo := oauth2memory.NewClientRepository()
	_, err := RegisterClient(testing_tenant.Default(context.Background()), RegisterClientDeps{ClientRepo: repo}, RegisterClientInput{
		ClientType:              spec.ClientConfidential,
		RedirectURIs:            []string{"https://client.example/cb"},
		TokenEndpointAuthMethod: domain.AuthMethodPrivateKeyJwt,
	}, time.Now())
	if err == nil {
		t.Fatal("expected missing jwks rejection")
	}
}

// 動的登録は外部のクライアントが繰り返し送れるので、不正なメタデータの登録が使用量を占めると、
// 上限の数だけ失敗させるだけで正当な登録まで拒否される。
//
//spec:covers REQ-TENANCY-050: 検証で失敗した登録は oauth2_clients の使用量を残さず、上限いっぱいの正当な登録が成功する。
func TestRegisterClientRejectedMetadataDoesNotConsumeQuota(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	quotaRepo := tenancy.NewMemoryModule().QuotaRepo
	limit := 1
	if err := quotaRepo.SetQuota(ctx, tenancydomain.DefaultTenantID, &tenancydomain.TenantQuota{OAuth2Clients: &limit}); err != nil {
		t.Fatalf("SetQuota: %v", err)
	}
	deps := RegisterClientDeps{ClientRepo: oauth2memory.NewClientRepository(), QuotaRepo: quotaRepo}
	rejected := []RegisterClientInput{
		// authorization_code の登録に redirect_uris がない。
		{ClientType: spec.ClientConfidential, TokenEndpointAuthMethod: domain.AuthMethodClientSecretBasic},
		// private_key_jwt に鍵がない。
		{
			ClientType: spec.ClientConfidential, RedirectURIs: []string{"https://client.example/cb"},
			TokenEndpointAuthMethod: domain.AuthMethodPrivateKeyJwt,
		},
		// redirect_uri が URL でないので Validate が拒否する。
		{
			ClientType: spec.ClientConfidential, RedirectURIs: []string{"not a url"},
			TokenEndpointAuthMethod: domain.AuthMethodClientSecretBasic,
		},
	}
	for i, in := range rejected {
		var oauthErr *OAuthError
		if _, err := RegisterClient(ctx, deps, in, time.Now()); !errors.As(err, &oauthErr) {
			t.Fatalf("rejected[%d]: RegisterClient = %v, want an OAuthError", i, err)
		}
	}
	usage, err := quotaRepo.GetUsage(ctx, tenancydomain.DefaultTenantID)
	if err != nil {
		t.Fatal(err)
	}
	if usage.OAuth2Clients != 0 {
		t.Fatalf("oauth2_clients usage after rejected registrations = %d, want 0", usage.OAuth2Clients)
	}
	if _, err := RegisterClient(ctx, deps, RegisterClientInput{
		ClientType: spec.ClientConfidential, RedirectURIs: []string{"https://client.example/cb"},
		TokenEndpointAuthMethod: domain.AuthMethodClientSecretBasic,
	}, time.Now()); err != nil {
		t.Fatalf("valid registration within the limit: %v", err)
	}
}
