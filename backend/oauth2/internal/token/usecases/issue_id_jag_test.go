package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/internal/db_memory"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"

	agentports "github.com/ambi/idmagic/backend/idmanagement/agent/ports"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/kernel"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

const (
	idJagAudience = "https://acme.chat.example/"
	idJagResource = "https://api.chat.example/"
	idJagUser     = "user-1"
)

type stubAssertionVerifier struct {
	assertion *ports.IDTokenAssertion
	err       error
}

func (s stubAssertionVerifier) VerifyIDTokenAssertion(context.Context, string) (*ports.IDTokenAssertion, error) {
	return s.assertion, s.err
}

type recordingIDJAGSigner struct {
	calls int
	last  ports.IDJAGInput
}

func (s *recordingIDJAGSigner) SignIDJAG(_ context.Context, in ports.IDJAGInput) (string, string, error) {
	s.calls++
	s.last = in
	return "id-jag", "jti-1", nil
}
func (s *recordingIDJAGSigner) IDJAGTTLSeconds() int { return 300 }

type idJagIssueFixture struct {
	deps   IssueIDJAGDeps
	signer *recordingIDJAGSigner
	events []spec.DomainEvent
	in     IssueIDJAGInput
}

func newIDJAGIssueFixture(t *testing.T) *idJagIssueFixture {
	t.Helper()
	ctx := testing_tenant.Default(context.Background())
	clients := oauth2memory.NewClientRepository()
	clients.Seed(&domain.OAuth2Client{ClientID: "client", ClientType: spec.ClientConfidential, Scope: "openid", CreatedAt: time.Now().UTC()})
	clients.Seed(&domain.OAuth2Client{ClientID: "other-client", ClientType: spec.ClientConfidential, Scope: "openid", CreatedAt: time.Now().UTC()})
	clients.Seed(&domain.OAuth2Client{ClientID: "public-client", ClientType: spec.ClientPublic, Scope: "openid", CreatedAt: time.Now().UTC()})
	users := idmanagement.NewMemoryModule().UserRepo
	email := "user@example.com"
	if err := users.Save(ctx, &userdomain.User{
		ID: idJagUser, TenantID: kernel.DefaultTenantID, PreferredUsername: "user", Email: &email, EmailVerified: true,
		Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
	}); err != nil {
		t.Fatal(err)
	}
	policies := oauth2memory.NewCrossAppAccessPolicyRepository()
	if err := policies.CreateIssuePolicy(ctx, &domain.CrossAppAccessIssuePolicy{
		TenantID: kernel.DefaultTenantID, ID: "p1", ClientID: "client", Audience: idJagAudience,
		RelyingPartyClientID: "chat-rp", TargetClientID: "f53f191f9311af35",
		Resources: []string{idJagResource}, Scopes: []string{"chat.read", "chat.history"},
		AuthorizationDetailTypes: []string{"chat_read"},
	}); err != nil {
		t.Fatal(err)
	}
	f := &idJagIssueFixture{signer: &recordingIDJAGSigner{}}
	f.deps = IssueIDJAGDeps{
		ClientRepo: clients, UserRepo: users, Policies: policies, Signer: f.signer,
		Verifier:          stubAssertionVerifier{assertion: &ports.IDTokenAssertion{Subject: idJagUser, Audiences: []string{"client"}, AuthTime: 100, AMR: []string{"pwd"}}},
		ApplicationAccess: func(context.Context, string, *ports.IDTokenAssertion) (bool, error) { return true, nil },
		Emit:              func(event spec.DomainEvent) { f.events = append(f.events, event) },
	}
	// 値は draft の 4.3.1 節の例から写す。製品の定数から作らない。
	f.in = IssueIDJAGInput{
		ClientID: "client", SubjectToken: "id-token", SubjectTokenType: "urn:ietf:params:oauth:token-type:id_token",
		Audiences: []string{idJagAudience}, Resources: []string{idJagResource}, Scope: "chat.read chat.history",
	}
	return f
}

func (f *idJagIssueFixture) issue() (*ExchangeTokenResult, error) {
	return IssueIDJAG(testing_tenant.Default(context.Background()), f.deps, f.in, time.Now().UTC())
}

//spec:covers REQ-OAUTH2-066, IDJAG-ISSUANCE: 許可された要求の ID-JAG のクレームと応答、部分集合のスコープ、詳細の絞り込み、監査
func TestIssueIDJAG_REQ_OAUTH2_066(t *testing.T) {
	f := newIDJAGIssueFixture(t)
	f.in.Scope = "chat.read admin"
	f.in.AuthorizationDetails = []spec.AuthorizationDetail{{Type: "chat_read"}, {Type: "secret"}}
	res, err := f.issue()
	if err != nil {
		t.Fatal(err)
	}
	if res.IssuedTokenType != "urn:ietf:params:oauth:token-type:id-jag" || res.TokenType != "N_A" || res.ExpiresIn != 300 || res.Scope != "chat.read" {
		t.Fatalf("response = %+v", res)
	}
	got := f.signer.last
	if got.Subject != idJagUser || got.Audience != idJagAudience || got.ClientID != "f53f191f9311af35" ||
		len(got.Resources) != 1 || got.Resources[0] != idJagResource || got.AuthTime != 100 || got.SenderConstraint != nil || got.Email != "" {
		t.Fatalf("ID-JAG input = %+v", got)
	}
	if len(got.AuthorizationDetails) != 1 || got.AuthorizationDetails[0].Type != "chat_read" {
		t.Fatalf("details = %+v", got.AuthorizationDetails)
	}
	issued, ok := f.events[len(f.events)-1].(*domain.IdJagIssued)
	if !ok || issued.Audience != idJagAudience || issued.ClientID != "client" {
		t.Fatalf("events = %+v", f.events)
	}
	t.Run("scope omitted grants the policy", func(t *testing.T) {
		f := newIDJAGIssueFixture(t)
		f.in.Scope = ""
		res, err := f.issue()
		if err != nil || res.Scope != "chat.read chat.history" {
			t.Fatalf("res = %+v err = %v", res, err)
		}
	})
}

//spec:covers REQ-OAUTH2-067: 許可の外の要求の誤りコード、ID-JAG を返さないこと、拒否の監査
func TestIssueIDJAGRefusals_REQ_OAUTH2_067(t *testing.T) {
	cases := map[string]struct {
		mutate func(f *idJagIssueFixture)
		code   string
	}{
		"public client":     {func(f *idJagIssueFixture) { f.in.ClientID = "public-client" }, "invalid_client"},
		"unknown client":    {func(f *idJagIssueFixture) { f.in.ClientID = "nobody" }, "invalid_client"},
		"actor token":       {func(f *idJagIssueFixture) { f.in.HasActorToken = true }, "invalid_request"},
		"access token type": {func(f *idJagIssueFixture) { f.in.SubjectTokenType = "urn:ietf:params:oauth:token-type:access_token" }, "invalid_request"},
		"two audiences":     {func(f *idJagIssueFixture) { f.in.Audiences = []string{idJagAudience, "https://b.example/"} }, "invalid_request"},
		"no audience":       {func(f *idJagIssueFixture) { f.in.Audiences = nil }, "invalid_request"},
		"not an ID Token": {func(f *idJagIssueFixture) {
			f.deps.Verifier = stubAssertionVerifier{err: errors.New("typ identifies another kind")}
		}, "invalid_grant"},
		"ID Token of another client": {func(f *idJagIssueFixture) {
			f.deps.Verifier = stubAssertionVerifier{assertion: &ports.IDTokenAssertion{Subject: idJagUser, Audiences: []string{"other-client"}}}
		}, "invalid_grant"},
		"ID Token with two audiences": {func(f *idJagIssueFixture) {
			f.deps.Verifier = stubAssertionVerifier{assertion: &ports.IDTokenAssertion{Subject: idJagUser, Audiences: []string{"client", "other-client"}}}
		}, "invalid_grant"},
		"unknown audience":    {func(f *idJagIssueFixture) { f.in.Audiences = []string{"https://unknown.example/"} }, "invalid_target"},
		"no resource":         {func(f *idJagIssueFixture) { f.in.Resources = nil }, "invalid_target"},
		"ungoverned resource": {func(f *idJagIssueFixture) { f.in.Resources = []string{idJagResource, "https://evil.example/"} }, "invalid_target"},
		"client without a policy": {func(f *idJagIssueFixture) {
			f.in.ClientID = "other-client"
			f.deps.Verifier = stubAssertionVerifier{assertion: &ports.IDTokenAssertion{Subject: idJagUser, Audiences: []string{"other-client"}}}
		}, "access_denied"},
		"no permitted scope": {func(f *idJagIssueFixture) { f.in.Scope = "admin" }, "access_denied"},
		"unknown user": {func(f *idJagIssueFixture) {
			f.deps.Verifier = stubAssertionVerifier{assertion: &ports.IDTokenAssertion{Subject: "nobody", Audiences: []string{"client"}}}
		}, "access_denied"},
		"application access denied": {func(f *idJagIssueFixture) {
			f.deps.ApplicationAccess = func(context.Context, string, *ports.IDTokenAssertion) (bool, error) { return false, nil }
		}, "access_denied"},
		"application gate not wired": {func(f *idJagIssueFixture) { f.deps.ApplicationAccess = nil }, "access_denied"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := newIDJAGIssueFixture(t)
			tc.mutate(f)
			_, err := f.issue()
			assertOAuthError(t, err, tc.code)
			if f.signer.calls != 0 {
				t.Fatal("a refused request must not sign an ID-JAG")
			}
			if len(f.events) != 1 {
				t.Fatalf("events = %+v", f.events)
			}
			rejected, ok := f.events[0].(*domain.IdJagRejected)
			if !ok || rejected.Reason != tc.code || rejected.Stage != "issue" {
				t.Fatalf("event = %+v", f.events[0])
			}
		})
	}
}

//spec:covers REQ-OAUTH2-068, IDJAG-SENDER-CONSTRAINT: 要求の証明から ID-JAG の束縛が一つだけ決まること
func TestIssueIDJAGBinding_REQ_OAUTH2_068(t *testing.T) {
	for name, tc := range map[string]struct {
		jkt, x5t  string
		wantJKT   string
		wantX5T   string
		unbounded bool
	}{
		"dpop": {jkt: "k", wantJKT: "k"},
		"mtls": {x5t: "c", wantX5T: "c"},
		"both": {jkt: "k", x5t: "c", wantJKT: "k"},
		"none": {unbounded: true},
	} {
		t.Run(name, func(t *testing.T) {
			f := newIDJAGIssueFixture(t)
			f.in.ProofJKT, f.in.ProofX5TS256 = tc.jkt, tc.x5t
			if _, err := f.issue(); err != nil {
				t.Fatal(err)
			}
			sc := f.signer.last.SenderConstraint
			if tc.unbounded {
				if sc != nil {
					t.Fatalf("constraint = %+v", sc)
				}
				return
			}
			if sc == nil || sc.JKT != tc.wantJKT || sc.X5TS256 != tc.wantX5T {
				t.Fatalf("constraint = %+v", sc)
			}
		})
	}
}

//spec:covers REQ-OAUTH2-066: 確認済みのメールアドレスだけを、ポリシーが許すときに載せること
func TestIssueIDJAGReleasesVerifiedEmailOnly_REQ_OAUTH2_066(t *testing.T) {
	ctx := testing_tenant.Default(context.Background())
	f := newIDJAGIssueFixture(t)
	policies := oauth2memory.NewCrossAppAccessPolicyRepository()
	if err := policies.CreateIssuePolicy(ctx, &domain.CrossAppAccessIssuePolicy{
		TenantID: kernel.DefaultTenantID, ID: "p1", ClientID: "client", Audience: idJagAudience, RelyingPartyClientID: "rp",
		TargetClientID: "t", Resources: []string{idJagResource}, Scopes: []string{"chat.read"}, ReleaseEmail: true,
	}); err != nil {
		t.Fatal(err)
	}
	f.deps.Policies = policies
	f.in.Scope = ""
	if _, err := f.issue(); err != nil {
		t.Fatal(err)
	}
	if f.signer.last.Email != "user@example.com" {
		t.Fatalf("email = %q", f.signer.last.Email)
	}
	user, err := f.deps.UserRepo.FindBySub(ctx, idJagUser)
	if err != nil {
		t.Fatal(err)
	}
	user.EmailVerified = false
	if err := f.deps.UserRepo.Save(ctx, user); err != nil {
		t.Fatal(err)
	}
	if _, err := f.issue(); err != nil {
		t.Fatal(err)
	}
	if f.signer.last.Email != "" {
		t.Fatalf("an unverified email was released: %q", f.signer.last.Email)
	}
}

//spec:covers REQ-OAUTH2-067, REQ-OAUTH2-070: 要求したクライアントの Agent の区分と本人の付与を、発行と償還の両方で確かめること
func TestIDJAGChecksTheRequestingAgent_REQ_OAUTH2_067(t *testing.T) {
	seedAgent := func(t *testing.T, clientID string, kind idmdomain.AgentKind) agentports.AgentRepository {
		t.Helper()
		agents := idmanagement.NewMemoryModule().AgentRepo
		seedSupervisionAgent(t, agents, "agent_1", clientID, kind)
		return agents
	}
	grant := func(t *testing.T, sub string, scopes ...string) ports.ConsentRepository {
		t.Helper()
		consents := oauth2memory.NewConsentRepository()
		if err := consents.SaveAgentDelegation(testing_tenant.Default(context.Background()), kernel.DefaultTenantID, &domain.Consent{
			UserID: sub, AgentID: "agent_1", Scopes: scopes, State: domain.ConsentGranted,
			GrantedAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Hour),
		}); err != nil {
			t.Fatal(err)
		}
		return consents
	}

	t.Run("issue refuses a supervised agent", func(t *testing.T) {
		f := newIDJAGIssueFixture(t)
		f.deps.AgentRepo = seedAgent(t, "client", idmdomain.AgentKindSupervised)
		_, err := f.issue()
		assertOAuthError(t, err, "unauthorized_client")
		if f.signer.calls != 0 {
			t.Fatal("signed for a supervised agent")
		}
	})
	t.Run("issue refuses an agent without the grant", func(t *testing.T) {
		f := newIDJAGIssueFixture(t)
		f.deps.AgentRepo = seedAgent(t, "client", idmdomain.AgentKindAutonomous)
		f.deps.Consents = oauth2memory.NewConsentRepository()
		_, err := f.issue()
		assertOAuthError(t, err, "access_denied")
	})
	t.Run("issue accepts an agent with the grant", func(t *testing.T) {
		f := newIDJAGIssueFixture(t)
		f.deps.AgentRepo = seedAgent(t, "client", idmdomain.AgentKindAutonomous)
		f.deps.Consents = grant(t, idJagUser, "anything")
		if _, err := f.issue(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("redeem refuses a grant that lacks the scopes", func(t *testing.T) {
		f := newIDJAGRedeemFixture(t)
		f.deps.AgentRepo = seedAgent(t, redeemClient, idmdomain.AgentKindAutonomous)
		f.deps.Consents = grant(t, "local-1", "chat.read")
		_, err := f.redeem()
		assertOAuthError(t, err, "access_denied")
		if f.issuer.calls != 0 {
			t.Fatal("issued for an agent without the grant")
		}
	})
	t.Run("redeem accepts a grant with the scopes", func(t *testing.T) {
		f := newIDJAGRedeemFixture(t)
		f.deps.AgentRepo = seedAgent(t, redeemClient, idmdomain.AgentKindAutonomous)
		f.deps.Consents = grant(t, "local-1", "chat.read", "chat.history")
		if _, err := f.redeem(); err != nil {
			t.Fatal(err)
		}
	})
}

type stubUserEpochs struct {
	epoch *ssdomain.UserRevocationEpoch
}

func (s stubUserEpochs) FindByUser(context.Context, string, string) (*ssdomain.UserRevocationEpoch, error) {
	return s.epoch, nil
}
func (s stubUserEpochs) Advance(context.Context, ssdomain.UserRevocationEpoch) error { return nil }

//spec:covers REQ-OAUTH2-067: 利用者を失効させた後に、それより前の ID Token で ID-JAG を得られないこと
func TestIssueIDJAGRefusesAfterUserRevocation_REQ_OAUTH2_067(t *testing.T) {
	issuedAt := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
	for name, tc := range map[string]struct {
		epoch  time.Time
		refuse bool
	}{
		"revoked after the ID Token":  {issuedAt.Add(time.Second), true},
		"revoked before the ID Token": {issuedAt.Add(-time.Second), false},
	} {
		t.Run(name, func(t *testing.T) {
			f := newIDJAGIssueFixture(t)
			f.deps.Verifier = stubAssertionVerifier{assertion: &ports.IDTokenAssertion{Subject: idJagUser, Audiences: []string{"client"}, IssuedAt: issuedAt}}
			f.deps.UserEpochs = stubUserEpochs{epoch: &ssdomain.UserRevocationEpoch{TenantID: kernel.DefaultTenantID, UserID: idJagUser, Epoch: tc.epoch}}
			_, err := f.issue()
			if tc.refuse {
				assertOAuthError(t, err, "invalid_grant")
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
