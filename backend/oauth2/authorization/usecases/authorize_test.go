package usecases

import (
	"context"
	"testing"
	"time"

	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"

	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"

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

func TestAuthorizeRejectsUndeclaredScope(t *testing.T) {
	in := validAuthorizeInput()
	in.Scope = "openid admin"
	if _, err := Authorize(context.Background(), newAuthorizeDeps(false), in); err == nil {
		t.Fatal("expected invalid_scope")
	}
}

func TestAuthorizeRequiresPARWhenConfigured(t *testing.T) {
	in := validAuthorizeInput()
	if _, err := Authorize(context.Background(), newAuthorizeDeps(true), in); err == nil {
		t.Fatal("expected PAR requirement rejection")
	}
	in.ParUsed = true
	in.ParRequestURI = "urn:ietf:params:oauth:request_uri:test"
	if _, err := Authorize(context.Background(), newAuthorizeDeps(true), in); err != nil {
		t.Fatalf("PAR request rejected: %v", err)
	}
}

func TestAuthorizePersistsPromptAndMaxAge(t *testing.T) {
	in := validAuthorizeInput()
	in.Prompt = "login"
	maxAge := 30
	in.MaxAge = &maxAge
	out, err := Authorize(context.Background(), newAuthorizeDeps(false), in)
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

// FAPI2-PAR-PKCE: プロファイルを選択しただけのクライアントが、PAR を経由しない
// 認可リクエストを通せないことを固定する。クライアント個別の PAR 設定は false の
// ままなので、拒否の理由はプロファイルの選択だけである。
func TestAuthorizeRequiresPARFromFapi2Clients(t *testing.T) {
	in := validAuthorizeInput()
	if _, err := Authorize(context.Background(), newFapi2AuthorizeDeps(domain.FapiSecurityProfileV2), in); err == nil {
		t.Fatal("プロファイルを選んだクライアントの PAR 無しリクエストが通った")
	}

	// PAR を経由すれば同じリクエストが保存まで進む。拒否だけを読むと、プロファイル
	// ごと通さない実装と区別できない。
	in.ParUsed = true
	in.ParRequestURI = "urn:ietf:params:oauth:request_uri:test"
	if _, err := Authorize(context.Background(), newFapi2AuthorizeDeps(domain.FapiSecurityProfileV2), in); err != nil {
		t.Fatalf("PAR 経由の FAPI クライアントが拒否された: %v", err)
	}

	// 対照: プロファイル以外がすべて同じクライアントは PAR 無しで通る。ここが落ちる
	// なら、制約は FAPI ではなく全クライアントに掛かっている (FAPI2-PROFILE-SELECTION)。
	control := validAuthorizeInput()
	if _, err := Authorize(context.Background(), newFapi2AuthorizeDeps(domain.FapiNone), control); err != nil {
		t.Fatalf("プロファイルを選んでいないクライアントまで PAR を求められた: %v", err)
	}
}
