package usecases_test

// 主要ユースケース追跡: REQ-AUTHENTICATION-001。

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

//spec:covers REQ-AUTHENTICATION-002, EX-AUTHENTICATION-002-01: 明示した VerifiedEmail ポリシーと検証済みの一意なメールアドレスの一致だけが、既存の User に対する FederatedIdentity を作ることを固定する。
func TestCompleteRequiresExplicitVerifiedEmailPolicyForAutoLink(t *testing.T) {
	deps, connection, users, repos := brokerFixture(t)
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
	// 戻り値だけでは、関連付けを保存せずに「リンクした」と名乗る実装を通してしまう。
	linked, err := repos.Identities.FindBySubject(
		context.Background(), tenancydomain.DefaultTenantID, connection.ID, "external",
	)
	if err != nil {
		t.Fatal(err)
	}
	if linked == nil || linked.LocalUserID != "user-1" {
		t.Fatalf("既存の User に対する関連付けが作られていない: %+v", linked)
	}
}

//spec:covers REQ-AUTHENTICATION-001, EX-AUTHENTICATION-001-01: 初回は、明示した JIT ポリシーとクレームの対応付けに従ってローカルの User と FederatedIdentity が作られることを固定する。
func TestCompleteJITRequiresPolicyAndProvisioner(t *testing.T) {
	deps, connection, _, repos := brokerFixture(t)
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
	linked, err := repos.Identities.FindBySubject(
		context.Background(), tenancydomain.DefaultTenantID, connection.ID, "external",
	)
	if err != nil {
		t.Fatal(err)
	}
	if linked == nil || linked.LocalUserID != "jit-user" {
		t.Fatalf("JIT で作った User に対する関連付けが残っていない: %+v", linked)
	}
}

// 明示的なリンクは、上流の callback が「この利用者へ結び付ける」試行として戻ったときに
// 成立する。未使用の外部 subject だけが対象で、他人が使っている subject は横取りできない。
//
//spec:covers REQ-AUTHENTICATION-003, EX-AUTHENTICATION-003-01: 未使用の外部 subject が要求元の User 自身へリンクされ、既に他人が使っている subject は拒否されることを固定する (解除の側は TestUnlinkRequiresRecentStepUpAndPreservesLastLoginMethod が持つ)。
func TestCompleteLinksAnUnusedSubjectToTheRequestingUser(t *testing.T) {
	deps, connection, users, repos := brokerFixture(t)
	now := time.Now().UTC()
	self := activeUser("user-self", "self@example.com", now)
	users.Seed(self)
	attempt := federationdomain.FederatedLoginAttempt{LinkUserID: self.ID}
	claims := federationdomain.NormalizedClaims{Subject: "unused-external", Username: "self@example.com"}

	completion, err := federationusecases.CompleteIdentity(
		context.Background(), deps, connection, attempt, claims, now,
	)
	if err != nil {
		t.Fatalf("CompleteIdentity: %v", err)
	}
	if completion.LinkingMethod != federationusecases.LinkingMethodExplicit {
		t.Fatalf("linking method=%q", completion.LinkingMethod)
	}
	linked, err := repos.Identities.FindBySubject(
		context.Background(), tenancydomain.DefaultTenantID, connection.ID, "unused-external",
	)
	if err != nil {
		t.Fatal(err)
	}
	if linked == nil || linked.LocalUserID != self.ID {
		t.Fatalf("自身へのリンクが残っていない: %+v", linked)
	}

	// 未使用であることが条件である。他人が使っている subject は横取りできない。
	other := activeUser("user-other", "other@example.com", now)
	users.Seed(other)
	if err := repos.Identities.Create(context.Background(), &federationdomain.FederatedIdentity{
		TenantID: tenancydomain.DefaultTenantID, ProviderID: connection.ID,
		ExternalSubject: "taken-external", LocalUserID: other.ID, LinkedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := federationusecases.CompleteIdentity(
		context.Background(), deps, connection, attempt,
		federationdomain.NormalizedClaims{Subject: "taken-external", Username: "self@example.com"}, now,
	); !errors.Is(err, federationusecases.ErrLinkingDenied) {
		t.Fatalf("他人が使っている subject のリンク err=%v", err)
	}
	taken, err := repos.Identities.FindBySubject(
		context.Background(), tenancydomain.DefaultTenantID, connection.ID, "taken-external",
	)
	if err != nil {
		t.Fatal(err)
	}
	if taken == nil || taken.LocalUserID != other.ID {
		t.Fatalf("拒否したのに既存の関連付けが動いた: %+v", taken)
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
