package handlers_http_test

import (
	"context"
	cryptostd "crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	agentmemory "github.com/ambi/idmagic/backend/idmanagement/agent/db_memory"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauth2http "github.com/ambi/idmagic/backend/oauth2/handlers_http"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	tokenusecases "github.com/ambi/idmagic/backend/oauth2/token/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tokensjose "github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	ssmemory "github.com/ambi/idmagic/backend/sharedsignals/db_memory"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"

	"github.com/labstack/echo/v5"
)

// resourceTestRealmAPI は、テストのリクエスト先レルムの IdMagic API (レルムの発行者識別子) である。
const resourceTestRealmAPI = "https://idp.test/realms/acme"

type resourceTestIntrospector struct {
	result *oauthports.IntrospectionResult
}

func (f resourceTestIntrospector) IntrospectAccessToken(context.Context, string) (*oauthports.IntrospectionResult, error) {
	return f.result, nil
}

// resourceTestDPoPProof signs a DPoP proof JWT with the given htm / htu / ath.
func resourceTestDPoPProof(t *testing.T, key *rsa.PrivateKey, jwk map[string]any, htm, htu, jti, ath string, now time.Time) string {
	t.Helper()
	header, err := json.Marshal(map[string]any{"typ": "dpop+jwt", "alg": "PS256", "jwk": jwk})
	if err != nil {
		t.Fatal(err)
	}
	claims := map[string]any{"htm": htm, "htu": htu, "jti": jti, "iat": now.Unix()}
	if ath != "" {
		claims["ath"] = ath
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	sig, err := rsa.SignPSS(rand.Reader, key, cryptostd.SHA256, digest[:], &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func resourceTestJWK(pub *rsa.PublicKey) map[string]any {
	return map[string]any{
		"kty": "RSA",
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(new(big.Int).SetInt64(int64(pub.E)).Bytes()),
	}
}

func resourceTestJKT(t *testing.T, jwk map[string]any) string {
	t.Helper()
	canonical, err := json.Marshal(map[string]any{"e": jwk["e"], "kty": jwk["kty"], "n": jwk["n"]})
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(canonical)
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// token through ath. A proof that only demonstrates key possession (no ath), or one made
// for another token, is rejected.
//
//spec:covers REQ-OAUTH2-045: a DPoP proof at a protected resource is bound to the presented access
func TestResourceDPoPProofBindsToPresentedAccessToken(t *testing.T) {
	now := time.Now().UTC()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwk := resourceTestJWK(&key.PublicKey)
	const (
		path        = "/realms/acme/api/account/v1/profile"
		accessToken = "AT1"
		otherToken  = "AT2"
	)

	for _, tc := range []struct {
		name          string
		ath           string
		authenticated bool
	}{
		{name: "ath of the presented token", ath: tokensjose.AccessTokenHash(accessToken), authenticated: true},
		{name: "no ath", ath: ""},
		{name: "ath of another access token", ath: tokensjose.AccessTokenHash(otherToken)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, path, http.NoBody)
			req.Header.Set("Authorization", "DPoP "+accessToken)
			req.Header.Set("DPoP", resourceTestDPoPProof(t, key, jwk, http.MethodGet, "https://idp.test"+path, tc.name, tc.ath, now))
			req = req.WithContext(tenancy.WithTenant(req.Context(), &tenancydomain.Tenant{ID: "acme"}, resourceTestRealmAPI, "/realms/acme"))
			c := e.NewContext(req, httptest.NewRecorder())
			a := support.Authenticator{AccessTokens: oauth2http.ResourceAccessTokens{
				Introspector: resourceTestIntrospector{result: &oauthports.IntrospectionResult{
					Active: true, Sub: "user-1", Scope: "account:read", Aud: []string{resourceTestRealmAPI},
					SenderConstraint: &oauthdomain.SenderConstraint{
						Type: spec.SenderConstraintDPoP, JKT: resourceTestJKT(t, jwk),
					},
				}},
				DpopReplayStore: oauth2memory.NewDpopReplayStore(),
			}}

			got, err := a.Authenticate(c)
			if tc.authenticated {
				if err != nil {
					t.Fatal(err)
				}
				if got == nil || got.Subject() != "user-1" {
					t.Fatalf("authn=%+v", got)
				}
				return
			}
			if _, ok := errors.AsType[*support.InvalidTokenError](err); !ok {
				t.Fatalf("err=%v authn=%+v; want InvalidTokenError", err, got)
			}
		})
	}
}

// admin / account portal の Bearer 認証は、/introspect と同じ失効判定
// (AgentRevocationEpoch と AccessTokenDenylist) を通る。
//
// 境界の両側と失効リストを 1 つの表で踏むのは、片方の判定だけを実装した経路を見分けるためである。
//
//spec:covers EX-OAUTH2-047-01, EX-OAUTH2-047-02, EX-OAUTH2-047-03: revocation epoch より前に発行された token と jti が失効リストに載った token は invalid_token で拒否され、epoch より後に発行された token は認証が成立する。
func TestResolveAuthnContextAppliesRevocation(t *testing.T) {
	now := time.Now().UTC()
	const clientID = "agent_client"

	agentRepo := agentmemory.NewAgentRepository()
	if err := agentRepo.Save(testing_tenant.Default(context.Background()), &agentdomain.Agent{
		ID: "agent_1", TenantID: tenancydomain.DefaultTenantID, Name: "agent_1",
		Kind: idmdomain.AgentKindAutonomous, OwnerUserID: "owner_1", Status: idmdomain.AgentStatusKilled,
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed agent: %v", err)
	}
	if _, err := agentRepo.AddBinding(testing_tenant.Default(context.Background()), &agentdomain.AgentCredentialBinding{
		AgentID: "agent_1", ClientID: clientID, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed binding: %v", err)
	}

	epochRepo := ssmemory.NewAgentRevocationEpochRepository()
	if err := epochRepo.Advance(testing_tenant.Default(context.Background()), ssdomain.AgentRevocationEpoch{
		AgentID: "agent_1", TenantID: tenancydomain.DefaultTenantID, Epoch: now,
		Reason: ssdomain.RevocationReasonAgentKilled, AdvancedAt: now,
	}); err != nil {
		t.Fatalf("seed epoch: %v", err)
	}

	denylist := oauth2memory.NewAccessTokenDenylist()
	if err := denylist.Add(testing_tenant.Default(context.Background()), "jti-denied", now.Add(time.Hour)); err != nil {
		t.Fatalf("seed denylist: %v", err)
	}

	for _, tc := range []struct {
		name          string
		result        *oauthports.IntrospectionResult
		authenticated bool
	}{
		{
			name: "issued before the agent revocation epoch",
			result: &oauthports.IntrospectionResult{
				Active: true, Sub: "user-1", Scope: "idmagic.admin", JTI: "jti-1",
				ClientID: clientID, Iat: now.Add(-time.Minute).Unix(),
			},
		},
		{
			name: "jti on the access token denylist",
			result: &oauthports.IntrospectionResult{
				Active: true, Sub: "user-1", Scope: "idmagic.admin", JTI: "jti-denied",
				ClientID: clientID, Iat: now.Add(time.Minute).Unix(),
			},
		},
		{
			name: "issued after the agent revocation epoch",
			result: &oauthports.IntrospectionResult{
				Active: true, Sub: "user-1", Scope: "idmagic.admin", JTI: "jti-2",
				ClientID: clientID, Iat: now.Add(time.Minute).Unix(),
			},
			authenticated: true,
		},
		{
			name: "client bound to no agent",
			result: &oauthports.IntrospectionResult{
				Active: true, Sub: "user-1", Scope: "idmagic.admin", JTI: "jti-3",
				ClientID: "plain_client", Iat: now.Add(-time.Hour).Unix(),
			},
			authenticated: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/realms/default/api/admin/v1/users", http.NoBody)
			req.Header.Set("Authorization", "Bearer jwt")
			req = req.WithContext(testing_tenant.Default(req.Context()))
			c := e.NewContext(req, httptest.NewRecorder())
			a := support.Authenticator{AccessTokens: oauth2http.ResourceAccessTokens{
				Introspector: resourceTestIntrospector{result: tc.result},
				Revocation: tokenusecases.IntrospectDeps{
					AgentRepo: agentRepo, RevocationEpochRepo: epochRepo, AccessTokenDenylist: denylist,
				},
			}}

			got, err := a.Authenticate(c)
			if tc.authenticated {
				if err != nil {
					t.Fatal(err)
				}
				if got == nil || got.Subject() != "user-1" {
					t.Fatalf("authn=%+v", got)
				}
				return
			}
			if _, ok := errors.AsType[*support.InvalidTokenError](err); !ok {
				t.Fatalf("err=%v authn=%+v; want InvalidTokenError", err, got)
			}
		})
	}
}
