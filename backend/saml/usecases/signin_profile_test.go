package usecases

import (
	"context"
	"testing"

	samldomain "github.com/ambi/idmagic/backend/saml/domain"
)

type profileTestSPRepository struct {
	sp *samldomain.SamlServiceProvider
}

func (r profileTestSPRepository) FindByEntityID(context.Context, string, string) (*samldomain.SamlServiceProvider, error) {
	return r.sp, nil
}

func (profileTestSPRepository) ListByTenant(context.Context, string) ([]*samldomain.SamlServiceProvider, error) {
	return nil, nil
}

func (profileTestSPRepository) Save(context.Context, *samldomain.SamlServiceProvider) error {
	return nil
}

func (profileTestSPRepository) Delete(context.Context, string, string) error {
	return nil
}

func TestSignInRejectsServiceProviderBoundToAnotherIDPProfile(t *testing.T) {
	t.Parallel()

	service := SignInService{SPRepo: profileTestSPRepository{sp: &samldomain.SamlServiceProvider{
		TenantID: "tenant-a", EntityID: "urn:sp", IDPProfileID: "profile-a",
	}}}
	outcome, err := service.Issue(context.Background(), SignInInput{
		TenantID:  "tenant-a",
		ProfileID: "profile-b",
		Request:   samldomain.AuthnRequest{Issuer: "urn:sp"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Kind != SignInRejected {
		t.Fatalf("outcome kind = %v, want rejected", outcome.Kind)
	}
	if outcome.Message != "service provider is not assigned to this identity provider profile" {
		t.Fatalf("message = %q", outcome.Message)
	}
}
