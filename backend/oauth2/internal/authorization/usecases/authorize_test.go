package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"

	oauth2memory "github.com/ambi/idmagic/backend/oauth2/internal/db_memory"

	"github.com/ambi/idmagic/backend/oauth2/domain"

	"github.com/ambi/idmagic/backend/shared/spec"
)

func newAuthorizeDeps(requirePAR bool) AuthorizeDeps {
	repo := oauth2memory.NewClientRepository()
	repo.Seed(&domain.OAuth2Client{
		ClientID: "client", ClientType: spec.ClientPublic,
		RedirectURIs: []string{"https://client.example/cb"},
		GrantTypes:   []spec.GrantType{spec.GrantAuthorizationCode},
		ResponseTypes: []spec.ResponseType{
			spec.ResponseTypeCode,
		},
		TokenEndpointAuthMethod:            domain.AuthMethodNone,
		Scope:                              "openid profile",
		IDTokenSignedResponseAlg:           signingdomain.SigAlgPS256,
		RequirePushedAuthorizationRequests: requirePAR,
		FapiProfile:                        domain.FapiNone,
		CreatedAt:                          time.Now(),
	})
	return AuthorizeDeps{
		ClientRepo: repo, RequestStore: oauth2memory.NewAuthorizationRequestStore(),
	}
}

func validAuthorizeInput() AuthorizeRequestInput {
	return AuthorizeRequestInput{
		ClientID: "client", RedirectURI: "https://client.example/cb",
		ResponseType: "code", Scope: "openid",
		CodeChallenge: "challenge", CodeChallengeMethod: "S256",
	}
}

//spec:covers REQ-OAUTH2-005, FAPI2-PAR-PKCE: 設定の有無とクライアント種別による必須条件、任意利用時の S256 制約を検証する。
func TestAuthorizePKCEPolicy(t *testing.T) {
	for _, tt := range []struct {
		name              string
		clientType        spec.ClientType
		fapi              domain.FapiProfile
		required          bool
		unauthenticated   bool
		challenge, method string
		wantError         bool
	}{
		{name: "confidential omitted", clientType: spec.ClientConfidential},
		{name: "confidential opt in", clientType: spec.ClientConfidential, required: true, wantError: true},
		{name: "confidential opt in S256", clientType: spec.ClientConfidential, required: true, challenge: "challenge", method: "S256"},
		{name: "public cannot opt out", clientType: spec.ClientPublic, wantError: true},
		{name: "unauthenticated cannot opt out", clientType: spec.ClientConfidential, unauthenticated: true, wantError: true},
		{name: "public S256", clientType: spec.ClientPublic, challenge: "challenge", method: "S256"},
		{name: "FAPI cannot opt out", clientType: spec.ClientConfidential, fapi: domain.FapiSecurityProfileV2, wantError: true},
		{name: "FAPI S256", clientType: spec.ClientConfidential, fapi: domain.FapiSecurityProfileV2, challenge: "challenge", method: "S256"},
		{name: "optional S256", clientType: spec.ClientConfidential, challenge: "challenge", method: "S256"},
		{name: "optional plain rejected", clientType: spec.ClientConfidential, challenge: "challenge", method: "plain", wantError: true},
		{name: "optional missing method", clientType: spec.ClientConfidential, challenge: "challenge", wantError: true},
		{name: "optional missing challenge", clientType: spec.ClientConfidential, method: "S256", wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testing_tenant.Default(t.Context())
			deps := newAuthorizeDeps(false)
			client, err := deps.ClientRepo.FindByID(ctx, tenantports.TenantID(ctx), "client")
			if err != nil || client == nil {
				t.Fatalf("client=%+v err=%v", client, err)
			}
			client.ClientType, client.FapiProfile, client.RequirePKCE = tt.clientType, tt.fapi, tt.required
			if tt.clientType == spec.ClientConfidential && !tt.unauthenticated {
				client.TokenEndpointAuthMethod = domain.AuthMethodClientSecretBasic
			}
			if err := deps.ClientRepo.Save(ctx, client); err != nil {
				t.Fatal(err)
			}
			in := validAuthorizeInput()
			in.CodeChallenge, in.CodeChallengeMethod, in.ParUsed = tt.challenge, tt.method, true
			out, err := Authorize(ctx, deps, in)
			if tt.wantError {
				if oauth, ok := errors.AsType[*OAuthError](err); !ok || oauth.Code != "invalid_request" || out != nil {
					t.Fatalf("authorization=%+v err=%v", out, err)
				}
				return
			}
			if err != nil || out == nil {
				t.Fatalf("authorization=%+v err=%v", out, err)
			}
			if out.Request.CodeChallenge != tt.challenge || string(out.Request.CodeChallengeMethod) != tt.method {
				t.Fatalf("saved PKCE=%q/%q", out.Request.CodeChallenge, out.Request.CodeChallengeMethod)
			}
		})
	}
}

func TestAuthorizeRejectsUndeclaredScope(t *testing.T) {
	in := validAuthorizeInput()
	in.Scope = "openid admin"
	if _, err := Authorize(testing_tenant.Default(context.Background()), newAuthorizeDeps(false), in); err == nil {
		t.Fatal("expected invalid_scope")
	}
}

func TestAuthorizeRequiresPARWhenConfigured(t *testing.T) {
	in := validAuthorizeInput()
	if _, err := Authorize(testing_tenant.Default(context.Background()), newAuthorizeDeps(true), in); err == nil {
		t.Fatal("expected PAR requirement rejection")
	}
	in.ParUsed = true
	in.ParRequestURI = "urn:ietf:params:oauth:request_uri:test"
	if _, err := Authorize(testing_tenant.Default(context.Background()), newAuthorizeDeps(true), in); err != nil {
		t.Fatalf("PAR request rejected: %v", err)
	}
}

func TestAuthorizePersistsPromptAndMaxAge(t *testing.T) {
	in := validAuthorizeInput()
	in.Prompt = "login"
	maxAge := 30
	in.MaxAge = &maxAge
	out, err := Authorize(testing_tenant.Default(context.Background()), newAuthorizeDeps(false), in)
	if err != nil {
		t.Fatal(err)
	}
	if out.Request.Prompt == nil || *out.Request.Prompt != "login" {
		t.Fatal("prompt was not persisted")
	}
	if out.Request.MaxAge == nil || *out.Request.MaxAge != maxAge {
		t.Fatal("max_age was not persisted")
	}
}

// newFapi2AuthorizeDeps は プロファイルの選択だけが違うクライアントを 1 件置く。
// RequirePushedAuthorizationRequests は false のままなので、PAR が必須になった
// 場合その理由はプロファイルの選択しかない。
func newFapi2AuthorizeDeps(profile domain.FapiProfile) AuthorizeDeps {
	repo := oauth2memory.NewClientRepository()
	repo.Seed(&domain.OAuth2Client{
		ClientID: "client", ClientType: spec.ClientConfidential,
		RedirectURIs:                       []string{"https://client.example/cb"},
		GrantTypes:                         []spec.GrantType{spec.GrantAuthorizationCode},
		ResponseTypes:                      []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod:            domain.AuthMethodPrivateKeyJwt,
		Scope:                              "openid profile",
		IDTokenSignedResponseAlg:           signingdomain.SigAlgPS256,
		RequirePushedAuthorizationRequests: false,
		FapiProfile:                        profile,
		CreatedAt:                          time.Now(),
	})
	return AuthorizeDeps{
		ClientRepo: repo, RequestStore: oauth2memory.NewAuthorizationRequestStore(),
	}
}

// 認可リクエストを通せないことを固定する。クライアント個別の PAR 設定は false の
// ままなので、拒否の理由はプロファイルの選択だけである。
//
//spec:covers FAPI2-PAR-PKCE: プロファイルを選択しただけのクライアントが、PAR を経由しない
func TestAuthorizeRequiresPARFromFapi2Clients(t *testing.T) {
	in := validAuthorizeInput()
	if _, err := Authorize(testing_tenant.Default(context.Background()), newFapi2AuthorizeDeps(domain.FapiSecurityProfileV2), in); err == nil {
		t.Fatal("プロファイルを選んだクライアントの PAR 無しリクエストが通った")
	}

	// PAR を経由すれば同じリクエストが保存まで進む。拒否だけを読むと、プロファイル
	// ごと通さない実装と区別できない。
	in.ParUsed = true
	in.ParRequestURI = "urn:ietf:params:oauth:request_uri:test"
	if _, err := Authorize(testing_tenant.Default(context.Background()), newFapi2AuthorizeDeps(domain.FapiSecurityProfileV2), in); err != nil {
		t.Fatalf("PAR 経由の FAPI クライアントが拒否された: %v", err)
	}

	// 対照: プロファイル以外がすべて同じクライアントは PAR 無しで通る。ここが落ちる
	// なら、制約は FAPI ではなく全クライアントに掛かっている (FAPI2-PROFILE-SELECTION)。
	control := validAuthorizeInput()
	if _, err := Authorize(testing_tenant.Default(context.Background()), newFapi2AuthorizeDeps(domain.FapiNone), control); err != nil {
		t.Fatalf("プロファイルを選んでいないクライアントまで PAR を求められた: %v", err)
	}
}
