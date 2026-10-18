package applicationdeps

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	catalogports "github.com/ambi/idmagic/backend/applicationcatalog/ports"
	claimdomain "github.com/ambi/idmagic/backend/claimmapping/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	clientports "github.com/ambi/idmagic/backend/oauth2/client/ports"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

//spec:covers REQ-APPLICATION-001, REQ-APPLICATION-006: 管理用の射影は秘密を含めず、保存した設定とクレームだけを表示する。
func TestOIDCConfigurationPreservesManagedValues(t *testing.T) {
	ctx := context.Background()
	repo := oauth2.NewMemoryModule().ClientRepo
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	client := &oauthdomain.OAuth2Client{
		TenantID: "tenant", ClientID: "client", ClientType: spec.ClientConfidential,
		RedirectURIs: []string{"https://rp.example/callback"}, GrantTypes: []spec.GrantType{spec.GrantAuthorizationCode},
		ResponseTypes: []spec.ResponseType{spec.ResponseTypeCode}, Scope: "openid custom",
		TokenEndpointAuthMethod:            oauthdomain.AuthMethodClientSecretPost,
		RequirePushedAuthorizationRequests: true, DpopBoundAccessTokens: true, FapiProfile: oauthdomain.FapiSecurityProfileV2,
		ClaimPolicy: &claimdomain.ClaimMappingPolicy{},
	}
	client.ClaimPolicy.NameID.SourceAttribute = "employee_id"
	if err := repo.Save(ctx, client); err != nil {
		t.Fatal(err)
	}
	want := &catalogports.OIDCConfiguration{
		ClientID: "client", ClientType: spec.ClientConfidential,
		RedirectURIs: []string{"https://rp.example/callback"}, GrantTypes: []spec.GrantType{spec.GrantAuthorizationCode},
		ResponseTypes: []spec.ResponseType{spec.ResponseTypeCode}, Scope: "openid custom",
		TokenEndpointAuthMethod: "client_secret_post", RequirePAR: true, DpopBoundAccessTokens: true,
		FapiProfile: "fapi_2_security_profile", ClientSecretRotatable: true,
		SecretCredentials: []catalogports.ClientSecretMetadata{}, SubSourceAttribute: "employee_id", Rules: []claimdomain.ClaimMappingRule{},
	}
	got, err := NewOIDCClients(repo, nil).Configuration(ctx, "tenant", "client", now)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("configuration=%+v err=%v, want %+v", got, err, want)
	}
}

//spec:covers REQ-APPLICATION-002, REQ-APPLICATION-007: カタログの管理操作は同じ所有者の状態と、一度だけ返すシークレットを使う。
func TestOIDCAdministrationPreservesInputsAndOwnerState(t *testing.T) {
	ctx := tenantports.WithTenantID(context.Background(), "tenant")
	module := oauth2.NewMemoryModule()
	var events []spec.DomainEvent
	admin := NewOIDCClients(module.ClientRepo, module.ClientAdministration(func(e spec.DomainEvent) { events = append(events, e) }, nil))
	now := time.Now().UTC()
	created, err := admin.Create(ctx, catalogports.CreateOIDCClientInput{ActorUserID: "actor", Now: now, Registration: catalogports.OIDCClientRegistration{
		ClientName: "Payroll", ClientType: spec.ClientConfidential, RedirectURIs: []string{"https://rp.example/callback"},
		GrantTypes: []spec.GrantType{spec.GrantAuthorizationCode, spec.GrantRefreshToken}, ResponseTypes: []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod: "client_secret_basic", Scope: "openid profile",
	}})
	if err != nil || created == nil || created.ClientID == "" || created.ClientSecret == "" {
		t.Fatalf("create=%+v err=%v", created, err)
	}
	client, err := module.ClientRepo.FindByID(ctx, "tenant", created.ClientID)
	if err != nil || client.ClientName == nil || *client.ClientName != "Payroll" || client.CreatedAt != now || len(events) != 1 {
		t.Fatalf("client=%+v events=%v err=%v", client, events, err)
	}
	if err := admin.BindApplication(ctx, "tenant", created.ClientID, "application"); err != nil {
		t.Fatal(err)
	}
	scope := "openid email"
	requirePAR, dpop := true, true
	redirects := []string{"https://rp.example/new"}
	grants := []spec.GrantType{spec.GrantAuthorizationCode}
	responses := []spec.ResponseType{spec.ResponseTypeCode}
	if err := admin.Update(ctx, catalogports.UpdateOIDCClientInput{ActorUserID: "actor", ClientID: created.ClientID, RedirectURIs: &redirects, GrantTypes: &grants, ResponseTypes: &responses, Scope: &scope, RequirePAR: &requirePAR, DpopBoundTokens: &dpop, Now: now}); err != nil {
		t.Fatal(err)
	}
	config, err := admin.Configuration(ctx, "tenant", created.ClientID, now)
	if err != nil || config.Scope != scope || !config.RequirePAR || !config.DpopBoundAccessTokens || !reflect.DeepEqual(config.RedirectURIs, redirects) || !reflect.DeepEqual(config.GrantTypes, grants) || !reflect.DeepEqual(config.ResponseTypes, responses) {
		t.Fatalf("config=%+v err=%v", config, err)
	}
	sub := " employee_id "
	rules := []claimdomain.ClaimMappingRule{{ClaimType: "department", Source: claimdomain.ClaimSourceFixed, FixedValue: "eng"}}
	if err := admin.UpdateClaims(ctx, "tenant", created.ClientID, catalogports.OIDCClaimUpdate{SubSourceAttribute: &sub, Rules: &rules, Now: now}); err != nil {
		t.Fatal(err)
	}
	client, err = module.ClientRepo.FindByID(ctx, "tenant", created.ClientID)
	if err != nil || client.ApplicationID != "application" || client.UpdatedAt != now || client.ClaimPolicy.NameID.SourceAttribute != "employee_id" || !reflect.DeepEqual(client.ClaimPolicy.Rules, rules) {
		t.Fatalf("claim client=%+v err=%v", client, err)
	}
	issued, err := admin.IssueSecret(ctx, catalogports.IssueClientSecretInput{ActorUserID: "actor", ClientID: created.ClientID, ExpiresInDays: 30, Now: now})
	if err != nil || issued.ClientSecret == "" || issued.Credential.Status != "Active" || len(issued.Credentials) != 2 {
		t.Fatalf("issue=%+v err=%v", issued, err)
	}
	_, err = admin.IssueSecret(ctx, catalogports.IssueClientSecretInput{ActorUserID: "actor", ClientID: created.ClientID, ExpiresInDays: 30, Now: now})
	if !errors.Is(err, catalogports.ErrClientSecretLimitExceeded) {
		t.Fatalf("limit error=%v", err)
	}
	revoked, err := admin.RevokeSecret(ctx, catalogports.RevokeClientSecretInput{ActorUserID: "actor", ClientID: created.ClientID, CredentialID: issued.Credential.CredentialID, Now: now})
	if err != nil || len(revoked.Credentials) != 2 {
		t.Fatalf("revoke=%+v err=%v", revoked, err)
	}
	for _, credential := range revoked.Credentials {
		if credential.CredentialID == issued.Credential.CredentialID && credential.Status != "Revoked" {
			t.Fatalf("credential=%+v", credential)
		}
	}
	rotated, err := admin.RotateSecret(ctx, catalogports.RotateClientSecretInput{ActorUserID: "actor", ClientID: created.ClientID, GraceDays: 1, Now: now})
	if err != nil || rotated.ClientSecret == "" || rotated.GraceUntil == nil || !rotated.GraceUntil.Equal(now.Add(24*time.Hour)) {
		t.Fatalf("rotate=%+v err=%v", rotated, err)
	}
	if err := admin.Delete(ctx, "tenant", created.ClientID); err != nil {
		t.Fatal(err)
	}
	if client, err := module.ClientRepo.FindByID(ctx, "tenant", created.ClientID); err != nil || client != nil {
		t.Fatalf("deleted client=%+v err=%v", client, err)
	}
}

func TestOIDCRegistrationKeepsPublicKeySettings(t *testing.T) {
	ctx := tenantports.WithTenantID(context.Background(), "tenant")
	module := oauth2.NewMemoryModule()
	admin := NewOIDCClients(module.ClientRepo, module.ClientAdministration(func(spec.DomainEvent) {}, nil))
	uri, dn := "https://rp.example/jwks", "CN=client"
	created, err := admin.Create(ctx, catalogports.CreateOIDCClientInput{ActorUserID: "actor", Now: time.Now().UTC(), Registration: catalogports.OIDCClientRegistration{
		ClientName: "Key client", ClientType: spec.ClientConfidential, GrantTypes: []spec.GrantType{spec.GrantClientCredentials}, TokenEndpointAuthMethod: "private_key_jwt", Scope: "openid", JwksURI: &uri, TlsClientAuthSubjectDN: &dn,
	}})
	if err != nil || created == nil || created.ClientSecret != "" {
		t.Fatalf("create=%+v err=%v", created, err)
	}
	client, err := module.ClientRepo.FindByID(ctx, "tenant", created.ClientID)
	if err != nil || client.JwksURI == nil || *client.JwksURI != uri || client.TlsClientAuthSubjectDN == nil || *client.TlsClientAuthSubjectDN != dn {
		t.Fatalf("client=%+v err=%v", client, err)
	}
}

type faultyClients struct {
	clientports.OAuth2ClientRepository
	findErr, listErr error
}

func (r faultyClients) FindByID(ctx context.Context, tenantID, clientID string) (*oauthdomain.OAuth2Client, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	return r.OAuth2ClientRepository.FindByID(ctx, tenantID, clientID)
}

func (r faultyClients) ListClientSecretCredentials(ctx context.Context, clientID string) ([]oauthdomain.ClientSecretCredential, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.OAuth2ClientRepository.ListClientSecretCredentials(ctx, clientID)
}

func TestOIDCQueriesPreserveMissingValuesAndErrors(t *testing.T) {
	ctx := context.Background()
	repo := oauth2.NewMemoryModule().ClientRepo
	now := time.Now().UTC()
	if NewClientNames(nil) != nil {
		t.Fatal("nil repository must leave the optional name query unwired")
	}
	for _, repository := range []clientports.OAuth2ClientRepository{nil, repo} {
		if got, err := NewOIDCClients(repository, nil).Configuration(ctx, "tenant", "missing", now); got != nil || err != nil {
			t.Fatalf("missing config=%+v err=%v", got, err)
		}
	}
	if name, err := NewClientNames(repo).FindClientName(ctx, "tenant", "missing"); name != nil || err != nil {
		t.Fatalf("missing name=%v err=%v", name, err)
	}
	failure := errors.New("read failed")
	faulty := faultyClients{OAuth2ClientRepository: repo, findErr: failure}
	if _, err := NewClientNames(faulty).FindClientName(ctx, "tenant", "client"); !errors.Is(err, failure) {
		t.Fatalf("name error=%v", err)
	}
	if _, err := NewOIDCClients(faulty, nil).Configuration(ctx, "tenant", "client", now); !errors.Is(err, failure) {
		t.Fatalf("config error=%v", err)
	}
	name := "Payroll"
	if err := repo.Save(ctx, &oauthdomain.OAuth2Client{TenantID: "tenant", ClientID: "client", ClientName: &name}); err != nil {
		t.Fatal(err)
	}
	gotName, err := NewClientNames(repo).FindClientName(ctx, "tenant", "client")
	if err != nil || gotName == nil || *gotName != name {
		t.Fatalf("name=%v err=%v", gotName, err)
	}
	*gotName = "changed"
	stored, _ := repo.FindByID(ctx, "tenant", "client")
	if *stored.ClientName != name {
		t.Fatal("query result aliased the stored name")
	}
	faulty = faultyClients{OAuth2ClientRepository: repo, listErr: failure}
	config, err := NewOIDCClients(faulty, nil).Configuration(ctx, "tenant", "client", now)
	if err != nil || config == nil || config.SubSourceAttribute != "user_id" || config.SecretCredentials == nil || len(config.Rules) != 0 {
		t.Fatalf("config=%+v err=%v", config, err)
	}
}

func TestOIDCAdministrationPropagatesRefusals(t *testing.T) {
	ctx := tenantports.WithTenantID(context.Background(), "tenant")
	module := oauth2.NewMemoryModule()
	admin := NewOIDCClients(module.ClientRepo, module.ClientAdministration(func(spec.DomainEvent) {}, tenancy.NewMemoryModule().QuotaRepo))
	if _, err := admin.Create(ctx, catalogports.CreateOIDCClientInput{Registration: catalogports.OIDCClientRegistration{TokenEndpointAuthMethod: "unknown"}}); err == nil {
		t.Fatal("invalid registration accepted")
	}
	if err := admin.Update(ctx, catalogports.UpdateOIDCClientInput{ClientID: "missing"}); err == nil {
		t.Fatal("missing client updated")
	}
	if err := admin.UpdateClaims(ctx, "tenant", "missing", catalogports.OIDCClaimUpdate{}); !errors.Is(err, catalogports.ErrOIDCClientNotFound) {
		t.Fatalf("claim error=%v", err)
	}
	if _, err := admin.IssueSecret(ctx, catalogports.IssueClientSecretInput{ClientID: "missing"}); err == nil {
		t.Fatal("missing client received a secret")
	}
	if _, err := admin.RotateSecret(ctx, catalogports.RotateClientSecretInput{ClientID: "missing"}); err == nil {
		t.Fatal("missing client rotated")
	}
	if _, err := admin.RevokeSecret(ctx, catalogports.RevokeClientSecretInput{ClientID: "missing"}); err == nil {
		t.Fatal("missing client revoked")
	}
}
