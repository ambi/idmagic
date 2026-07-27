package usecases_test

import (
	"context"
	"errors"
	"testing"
	"time"

	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	federationmemory "github.com/ambi/idmagic/backend/authentication/federation/db_memory"
	federationdomain "github.com/ambi/idmagic/backend/authentication/federation/domain"
	federationusecases "github.com/ambi/idmagic/backend/authentication/federation/usecases"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func TestCompleteUsesExistingFederatedIdentityAndIssuesSession(t *testing.T) {
	deps, connection, users, repos := brokerFixture(t)
	now := time.Now().UTC()
	user := activeUser("user-1", "existing@example.com", now)
	users.Seed(user)
	if err := repos.Identities.Create(context.Background(), &federationdomain.FederatedIdentity{
		TenantID: tenancydomain.DefaultTenantID, ProviderID: connection.ID,
		ExternalSubject: "external", LocalUserID: user.ID, LinkedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	completion, err := federationusecases.CompleteIdentity(
		context.Background(), deps, connection,
		federationdomain.FederatedLoginAttempt{},
		federationdomain.NormalizedClaims{Subject: "external", Username: "existing@example.com"},
		now,
	)
	if err != nil {
		t.Fatalf("CompleteIdentity: %v", err)
	}
	if completion.User.ID != user.ID || completion.Authentication.SessionID == "" {
		t.Fatalf("completion=%+v", completion)
	}
}

func TestCompleteRequiresExplicitVerifiedEmailPolicyForAutoLink(t *testing.T) {
	deps, connection, users, _ := brokerFixture(t)
	now := time.Now().UTC()
	users.Seed(activeUser("user-1", "linked@example.com", now))
	claims := federationdomain.NormalizedClaims{
		Subject: "external", Username: "linked@example.com",
		Email: "linked@example.com", EmailVerified: true,
	}
	if _, err := federationusecases.CompleteIdentity(
		context.Background(), deps, connection, federationdomain.FederatedLoginAttempt{}, claims, now,
	); !errors.Is(err, federationusecases.ErrLinkingDenied) {
		t.Fatalf("policy none err=%v", err)
	}
	connection.LinkingPolicy = federationdomain.LinkingVerifiedEmail
	completion, err := federationusecases.CompleteIdentity(
		context.Background(), deps, connection, federationdomain.FederatedLoginAttempt{}, claims, now,
	)
	if err != nil {
		t.Fatalf("verified email completion: %v", err)
	}
	if completion.LinkingMethod != federationusecases.LinkingMethodVerifiedEmail {
		t.Fatalf("linking method=%q", completion.LinkingMethod)
	}
}

func TestCompleteJITRequiresPolicyAndProvisioner(t *testing.T) {
	deps, connection, _, _ := brokerFixture(t)
	now := time.Now().UTC()
	claims := federationdomain.NormalizedClaims{
		Subject: "external", Username: "new-user", Email: "new@example.com", EmailVerified: true,
	}
	if _, err := federationusecases.CompleteIdentity(
		context.Background(), deps, connection, federationdomain.FederatedLoginAttempt{}, claims, now,
	); !errors.Is(err, federationusecases.ErrLinkingDenied) {
		t.Fatalf("JIT disabled err=%v", err)
	}
	connection.JITProvisioning = true
	deps.ProvisionUser = func(_ context.Context, claims federationdomain.NormalizedClaims, now time.Time) (*userdomain.User, error) {
		return activeUser("jit-user", claims.Email, now), nil
	}
	completion, err := federationusecases.CompleteIdentity(
		context.Background(), deps, connection, federationdomain.FederatedLoginAttempt{}, claims, now,
	)
	if err != nil {
		t.Fatalf("JIT completion: %v", err)
	}
	if completion.User.ID != "jit-user" || completion.LinkingMethod != federationusecases.LinkingMethodJIT {
		t.Fatalf("completion=%+v", completion)
	}
}

func TestUnlinkRequiresRecentStepUpAndPreservesLastLoginMethod(t *testing.T) {
	deps, connection, users, repos := brokerFixture(t)
	now := time.Now().UTC()
	user := activeUser("user-1", "linked@example.com", now)
	user.PasswordHash = ""
	users.Seed(user)
	if err := repos.Identities.Create(context.Background(), &federationdomain.FederatedIdentity{
		TenantID: tenancydomain.DefaultTenantID, ProviderID: connection.ID,
		ExternalSubject: "external", LocalUserID: user.ID, LinkedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	stale := &authdomain.AuthenticationContext{UserID: user.ID, AuthTime: now.Add(-time.Hour).Unix()}
	if err := federationusecases.UnlinkIdentity(context.Background(), deps, stale, connection.ID, now); err == nil {
		t.Fatal("stale step-up must be rejected")
	}
	recent := &authdomain.AuthenticationContext{UserID: user.ID, AuthTime: now.Unix(), StepUpAt: now.Unix()}
	if err := federationusecases.UnlinkIdentity(context.Background(), deps, recent, connection.ID, now); err == nil {
		t.Fatal("last login method must not be removed")
	}
	user.PasswordHash = "hash"
	users.Seed(user)
	if err := federationusecases.UnlinkIdentity(context.Background(), deps, recent, connection.ID, now); err != nil {
		t.Fatalf("UnlinkIdentity: %v", err)
	}
}

func brokerFixture(t *testing.T) (
	federationusecases.BrokerDeps,
	federationdomain.IdentityProviderConnection,
	*usermemory.UserRepository,
	federationmemory.Repositories,
) {
	t.Helper()
	repos := federationmemory.NewRepositories()
	users := usermemory.NewUserRepository()
	sessions := sessionusecases.NewSessionManager(sessionmemory.NewSessionStore())
	connection := federationdomain.IdentityProviderConnection{
		ID: "provider", TenantID: tenancydomain.DefaultTenantID, DisplayName: "Provider",
		Protocol: federationdomain.ProtocolOIDC, Status: federationdomain.ConnectionActive,
		Issuer: "https://idp.example", ClientID: "client",
		AuthorizationEndpoint: "https://idp.example/auth",
		TokenEndpoint:         "https://idp.example/token", JWKSURI: "https://idp.example/jwks",
		ClaimMapping:  federationdomain.ClaimMapping{Subject: "sub", Username: "email"},
		LinkingPolicy: federationdomain.LinkingNone,
	}
	return federationusecases.BrokerDeps{
		Identities: repos.Identities, Users: users, Sessions: sessions,
	}, connection, users, repos
}

func activeUser(id, email string, now time.Time) *userdomain.User {
	return &userdomain.User{
		ID: id, TenantID: tenancydomain.DefaultTenantID, PreferredUsername: email,
		PasswordHash: "hash", Email: &email, EmailVerified: true, Roles: []string{},
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
		CreatedAt: now, UpdatedAt: now,
	}
}
