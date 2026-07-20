package usecases

import (
	"testing"
	"time"

	oauthmemory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
)

func TestRotateClientSecretReplacesCurrentCredentialWithOverlap(t *testing.T) {
	ctx := tenantContext("default")
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	repo := oauthmemory.NewClientRepository()
	oldHash := oauthdomain.HashClientSecret("old")
	client := &oauthdomain.OAuth2Client{TenantID: "default", ClientID: "client", ClientSecretHash: &oldHash, ClientType: spec.ClientConfidential, GrantTypes: []spec.GrantType{spec.GrantClientCredentials}, TokenEndpointAuthMethod: oauthdomain.AuthMethodClientSecretBasic, IDTokenSignedResponseAlg: "PS256", FapiProfile: oauthdomain.FapiNone, CreatedAt: now, UpdatedAt: now}
	if err := repo.Save(ctx, client); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveClientSecretCredential(ctx, oauthdomain.ClientSecretCredential{CredentialID: "old", ClientID: client.ClientID, SecretHash: oldHash, CreatedAt: now}); err != nil {
		t.Fatal(err)
	}

	result, err := RotateClientSecret(ctx, AdminOAuth2ClientDeps{ClientRepo: repo}, RotateClientSecretInput{ActorUserID: "admin", ClientID: client.ClientID, GraceDays: 7, Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if result.ClientSecret == "" || result.GraceUntil == nil || !result.GraceUntil.Equal(now.AddDate(0, 0, 7)) {
		t.Fatalf("unexpected rotation result: %#v", result)
	}
	credentials, err := repo.ListClientSecretCredentials(ctx, client.ClientID)
	if err != nil {
		t.Fatal(err)
	}
	if len(credentials) != 2 || credentials[0].ExpiresAt == nil || !credentials[0].ExpiresAt.Equal(*result.GraceUntil) {
		t.Fatalf("unexpected credentials: %#v", credentials)
	}
	if !oauthdomain.VerifyClientSecret(result.ClientSecret, credentials[1].SecretHash) {
		t.Fatal("new secret was not stored as a hash")
	}
}
