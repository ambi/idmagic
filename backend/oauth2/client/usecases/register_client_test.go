package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"

	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func TestRegisterClientHashesSecret(t *testing.T) {
	repo := oauth2memory.NewClientRepository()
	result, err := RegisterClient(context.Background(), RegisterClientDeps{ClientRepo: repo}, RegisterClientInput{
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
// (spec/contexts/tenancy.yaml), applied to the oauth2_clients resource. This
// covers both dynamic client registration and admin client creation since
// CreateAdminOAuth2Client calls RegisterClient internally.
func TestRegisterClient_rejectsWhenHardQuotaExceeded(t *testing.T) {
	ctx := context.Background()
	repo := oauth2memory.NewClientRepository()
	quotaRepo := tenancymemory.NewQuotaRepository()
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
	_, err := RegisterClient(context.Background(), RegisterClientDeps{ClientRepo: repo}, RegisterClientInput{
		ClientType:              spec.ClientConfidential,
		RedirectURIs:            []string{"https://client.example/cb"},
		TokenEndpointAuthMethod: domain.AuthMethodPrivateKeyJwt,
	}, time.Now())
	if err == nil {
		t.Fatal("expected missing jwks rejection")
	}
}
