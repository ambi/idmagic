package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	"github.com/ambi/idmagic/backend/oauth2/ports"
	"github.com/ambi/idmagic/backend/shared/spec"
)

type staticHintVerifier struct {
	claims *ports.IDTokenHintClaims
	err    error
}

func (s staticHintVerifier) VerifyIDTokenHint(context.Context, string) (*ports.IDTokenHintClaims, error) {
	return s.claims, s.err
}

func seedEndSessionClient(t *testing.T, repo *memory.OAuth2ClientRepository, clientID string, redirectURIs []string) {
	t.Helper()
	repo.Seed(&domain.OAuth2Client{
		ClientID: clientID, ClientType: spec.ClientPublic,
		RedirectURIs:            redirectURIs,
		GrantTypes:              []spec.GrantType{spec.GrantAuthorizationCode},
		ResponseTypes:           []spec.ResponseType{spec.ResponseTypeCode},
		TokenEndpointAuthMethod: domain.AuthMethodNone,
		Scope:                   "openid",
		CreatedAt:               time.Now().UTC(),
	})
}

func TestResolveEndSessionNoRedirectSkipsClientResolution(t *testing.T) {
	// レガシー互換: post_logout_redirect_uri が無ければ client_id も id_token_hint も
	// 不要で、即座に "signed-out" 相当 (Client==nil) を返す。
	target, err := ResolveEndSession(context.Background(), EndSessionDeps{}, EndSessionInput{})
	if err != nil {
		t.Fatal(err)
	}
	if target.Client != nil {
		t.Fatalf("expected no client resolution, got %#v", target.Client)
	}
}

func TestResolveEndSessionValidatesRegisteredRedirectURI(t *testing.T) {
	repo := memory.NewClientRepository()
	seedEndSessionClient(t, repo, "web-app", []string{"https://app.example.com/post-logout"})

	target, err := ResolveEndSession(context.Background(), EndSessionDeps{ClientRepo: repo}, EndSessionInput{
		ClientID: "web-app", PostLogoutRedirectURI: "https://app.example.com/post-logout",
	})
	if err != nil {
		t.Fatal(err)
	}
	if target.Client == nil || target.RedirectURI != "https://app.example.com/post-logout" {
		t.Fatalf("unexpected target: %#v", target)
	}

	_, err = ResolveEndSession(context.Background(), EndSessionDeps{ClientRepo: repo}, EndSessionInput{
		ClientID: "web-app", PostLogoutRedirectURI: "https://evil.example.com/cb",
	})
	var oerr *OAuthError
	if !errors.As(err, &oerr) || oerr.Code != "invalid_request" {
		t.Fatalf("expected invalid_request for unregistered redirect, got %v", err)
	}
}

// ADR-127: id_token_hint の sid をそのまま session 解決に使い、hint の aud を
// client_id として扱う。
func TestResolveEndSessionResolvesSidAndClientFromHint(t *testing.T) {
	repo := memory.NewClientRepository()
	seedEndSessionClient(t, repo, "web-app", []string{"https://app.example.com/post-logout"})

	target, err := ResolveEndSession(context.Background(), EndSessionDeps{
		ClientRepo:   repo,
		HintVerifier: staticHintVerifier{claims: &ports.IDTokenHintClaims{Audience: "web-app", Sid: "session-1", Subject: "alice"}},
	}, EndSessionInput{IDTokenHint: "hint-token"})
	if err != nil {
		t.Fatal(err)
	}
	if target.Sid != "session-1" {
		t.Fatalf("sid=%q, want session-1", target.Sid)
	}
	// post_logout_redirect_uri が無いので client 解決はスキップされる。
	if target.Client != nil {
		t.Fatalf("expected no client resolution without post_logout_redirect_uri, got %#v", target.Client)
	}
}

func TestResolveEndSessionRejectsClientIDMismatchWithHint(t *testing.T) {
	target, err := ResolveEndSession(context.Background(), EndSessionDeps{
		HintVerifier: staticHintVerifier{claims: &ports.IDTokenHintClaims{Audience: "web-app", Sid: "session-1"}},
	}, EndSessionInput{ClientID: "other-app", IDTokenHint: "hint-token"})
	if target != nil {
		t.Fatalf("expected no target on mismatch, got %#v", target)
	}
	var oerr *OAuthError
	if !errors.As(err, &oerr) || oerr.Code != "invalid_request" {
		t.Fatalf("expected invalid_request for client_id/hint mismatch, got %v", err)
	}
}

func TestResolveEndSessionRejectsUnverifiableHint(t *testing.T) {
	_, err := ResolveEndSession(context.Background(), EndSessionDeps{
		HintVerifier: staticHintVerifier{err: errors.New("bad signature")},
	}, EndSessionInput{IDTokenHint: "hint-token"})
	var oerr *OAuthError
	if !errors.As(err, &oerr) || oerr.Code != "invalid_request" {
		t.Fatalf("expected invalid_request for unverifiable hint, got %v", err)
	}
}

func TestResolveEndSessionWithoutVerifierRejectsHint(t *testing.T) {
	// HintVerifier が配線されていない環境で id_token_hint を渡された場合は
	// fail-closed で拒否する (cookie フォールバックへの黙示的降格をしない)。
	_, err := ResolveEndSession(context.Background(), EndSessionDeps{}, EndSessionInput{IDTokenHint: "hint-token"})
	var oerr *OAuthError
	if !errors.As(err, &oerr) || oerr.Code != "invalid_request" {
		t.Fatalf("expected invalid_request, got %v", err)
	}
}
