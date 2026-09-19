package bootstrap

// docs/domain/system/scenarios.feature.md の REQ-SYSTEM-006 は、DemoLoginAffordance が
// 使う資格情報を `development` プロファイルが投入することを述べる。導線の表示そのものは
// frontend が持つ (routes/-demo-login.ts)。ここで観測するのは、その導線が始める認可が
// 成立する側の条件、つまり demo クライアントと資格情報を持つ利用者が
// `development` プロファイルにだけ存在することである。
//
// 2 つのプロファイルを並べるのは、片方だけを見る検査が「どのプロファイルでも投入される」
// 実装を通すからである。その実装では、本番の bootstrap に既知のパスワードを持つ管理者が
// 残る。

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/seeding/domain"
	"github.com/ambi/idmagic/backend/shared/spec"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

func applySeedProfile(t *testing.T, profile domain.Profile) *Dependencies {
	t.Helper()
	deps, err := assembleMemory(SharedConfig{})
	if err != nil {
		t.Fatalf("assembleMemory(SharedConfig{}) error = %v", err)
	}
	request := domain.Request{
		Environment: domain.EnvironmentDevelopment,
		Profile:     profile,
		Mode:        domain.ModeApply,
	}
	if _, err := Seed(context.Background(), deps, request, ""); err != nil {
		t.Fatalf("Seed(%s) error = %v", profile, err)
	}
	return deps
}

//spec:covers EX-SYSTEM-006-01: development プロファイルが、DemoLoginAffordance の始める authorization_code の認可先となる demo クライアントと、パスワードを持つデモ利用者を投入すること。
//spec:covers EX-SYSTEM-006-03: development プロファイルを適用していないときは demo クライアントもデモ利用者も存在せず、既知のデモ資格情報では認可を始められないこと。
func TestDemoLoginCredentialsExistOnlyInTheDevelopmentProfile(t *testing.T) {
	t.Setenv("DEMO_CLIENT_SECRET", "demo-client-secret")
	t.Setenv("DEMO_USER_PASSWORD", "demo-password-1234")
	ctx := context.Background()

	development := applySeedProfile(t, domain.ProfileDevelopment)

	client, err := development.OAuth2.ClientRepo.FindByID(ctx, tenancydomain.DefaultTenantID, seedDemoClientID)
	if err != nil {
		t.Fatalf("FindByID(demo client) error = %v", err)
	}
	if client == nil {
		t.Fatal("the development profile installed no demo client; the affordance would have no client_id to authorize with")
	}
	if !slices.Contains(client.GrantTypes, spec.GrantAuthorizationCode) {
		t.Errorf("demo client grant types = %v, want the authorization_code grant the affordance starts", client.GrantTypes)
	}
	if !slices.Contains(client.ResponseTypes, spec.ResponseTypeCode) {
		t.Errorf("demo client response types = %v, want the code response type", client.ResponseTypes)
	}
	// 導線は現在の origin の /callback へ戻す。登録済みの URI にそれが無いと認可は
	// redirect_uri の不一致で止まる。
	hasCallback := slices.ContainsFunc(client.RedirectURIs, func(uri string) bool {
		return strings.HasSuffix(uri, "/callback")
	})
	if !hasCallback {
		t.Errorf("demo client redirect URIs = %v, want one ending in /callback", client.RedirectURIs)
	}

	alice, err := development.IdManagement.UserRepo.FindBySub(ctx, seedUserAliceID)
	if err != nil {
		t.Fatalf("FindBySub(demo user) error = %v", err)
	}
	if alice == nil {
		t.Fatal("the development profile installed no demo user")
	}
	if alice.PasswordHash == "" {
		t.Error("the demo user has no password; the affordance's credentials could not authenticate")
	}

	// 同じ環境で bootstrap を当てると、どちらも存在しない。
	bootstrapped := applySeedProfile(t, domain.ProfileBootstrap)

	absentClient, err := bootstrapped.OAuth2.ClientRepo.FindByID(ctx, tenancydomain.DefaultTenantID, seedDemoClientID)
	if err != nil {
		t.Fatalf("FindByID(demo client, bootstrap) error = %v", err)
	}
	if absentClient != nil {
		t.Errorf("the bootstrap profile installed the demo client %#v", absentClient)
	}
	absentUser, err := bootstrapped.IdManagement.UserRepo.FindBySub(ctx, seedUserAliceID)
	if err != nil {
		t.Fatalf("FindBySub(demo user, bootstrap) error = %v", err)
	}
	if absentUser != nil {
		t.Errorf("the bootstrap profile installed the demo user %#v", absentUser)
	}
}
