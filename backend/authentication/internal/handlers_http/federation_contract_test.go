package handlers_http_test

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	federationdomain "github.com/ambi/idmagic/backend/authentication/internal/federation/domain"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	"github.com/ambi/idmagic/backend/tenancy/testing_tenant"
)

//verify:operation ListLinkedExternalIdentities: 外部 identity 一覧と関連付け解除の HTTP 応答および保存結果
//verify:operation UnlinkExternalIdentity: 外部 identity 一覧と関連付け解除の HTTP 応答および保存結果
func TestLinkedIdentityHTTPContracts(t *testing.T) {
	f := newAuthRefusalServer(t)
	f.seedFederatedLink(t, tenancydomain.DefaultTenantID, "contract-idp", authRefusalAlice, "private-external-subject")
	f.seedFederatedLink(t, authRefusalOtherTenant, "foreign-idp", authRefusalAlice, "foreign-subject")
	session := f.seedSession(t, "link-contract", tenancydomain.DefaultTenantID, authRefusalAlice, withFreshStepUp)
	body := authContractJSON(t, f.send(t, authRefusalRequest{method: http.MethodGet, path: "/api/account/v1/linked-identities", sessionID: session}), http.StatusOK)
	identities := body["identities"].([]any)
	if len(identities) != 1 {
		t.Fatalf("identities=%v", identities)
	}
	identity := identities[0].(map[string]any)
	if identity["tenant_id"] != tenancydomain.DefaultTenantID || identity["provider_id"] != "contract-idp" || identity["local_user_id"] != authRefusalAlice {
		t.Fatalf("identity=%v", identity)
	}
	authContractTime(t, identity, "linked_at")
	if _, present := identity["external_subject"]; present {
		t.Fatal("external subject was disclosed")
	}
	authContractEmpty(t, f.send(t, authRefusalRequest{method: http.MethodDelete, path: "/api/account/v1/linked-identities/contract-idp", sessionID: session, csrf: authRefusalCSRF}))
	if links := f.linkedProviders(t, authRefusalAlice); len(links) != 0 {
		t.Fatalf("links after unlink=%v", links)
	}
}

//verify:operation ListLinkedExternalIdentities: 未認証と CSRF への Problem Details
//verify:operation UnlinkExternalIdentity: 未認証と CSRF への Problem Details
func TestLinkedIdentityHTTPContractsRefuseAuthenticationAndCSRF(t *testing.T) {
	f := newAuthRefusalServer(t)
	f.seedFederatedLink(t, tenancydomain.DefaultTenantID, "contract-idp", authRefusalAlice, "private-external-subject")
	session := f.seedSession(t, "link-refusal", tenancydomain.DefaultTenantID, authRefusalAlice, withFreshStepUp)
	authContractProblem(t, f.send(t, authRefusalRequest{method: http.MethodGet, path: "/api/account/v1/linked-identities"}), http.StatusUnauthorized, "authentication_required")
	authContractProblem(t, f.send(t, authRefusalRequest{method: http.MethodDelete, path: "/api/account/v1/linked-identities/contract-idp", sessionID: session}), http.StatusForbidden, "csrf_failed")
	if links := f.linkedProviders(t, authRefusalAlice); len(links) != 1 || len(*f.events) != 0 {
		t.Fatalf("refusal changed links=%v", links)
	}
}

//verify:operation LinkExternalIdentity: 303 Location、本人へ結び付けた試行と PKCE、および CSRF の 403
func TestExternalIdentityLinkHTTPContractStartsBoundAttemptAndRefusesCSRF(t *testing.T) {
	f := newAuthRefusalServer(t)
	f.seedConnection(t, authRefusalProviderID, "Contract provider")
	ctx := testing_tenant.Default(context.Background())
	connection, err := f.federation.Connections.Find(ctx, tenancydomain.DefaultTenantID, authRefusalProviderID)
	if err != nil || connection == nil {
		t.Fatalf("connection=%v err=%v", connection, err)
	}
	connection.Status = federationdomain.ConnectionActive
	if err := f.federation.Connections.Save(ctx, connection); err != nil {
		t.Fatal(err)
	}
	session := f.seedSession(t, "link-start-contract", tenancydomain.DefaultTenantID, authRefusalAlice, withFreshStepUp)
	path := "/api/account/v1/linked-identities/" + authRefusalProviderID
	refused := f.send(t, authRefusalRequest{method: http.MethodPost, path: path, sessionID: session, body: map[string]any{}})
	authContractProblem(t, refused, http.StatusForbidden, "csrf_failed")
	if refused.Header().Get("Location") != "" {
		t.Fatal("refused link redirected")
	}
	started := f.send(t, authRefusalRequest{method: http.MethodPost, path: path, sessionID: session, csrf: authRefusalCSRF, body: map[string]any{}})
	location, err := url.Parse(started.Header().Get("Location"))
	if err != nil || started.Code != http.StatusSeeOther || started.Body.Len() != 0 || location.Scheme+"://"+location.Host+location.Path != "https://upstream.example/auth" {
		t.Fatalf("link status=%d location=%v err=%v body=%s", started.Code, location, err, started.Body.String())
	}
	query := location.Query()
	if query.Get("state") == "" || query.Get("nonce") == "" || query.Get("code_challenge") == "" || query.Get("code_challenge_method") != "S256" || query.Get("client_id") != "upstream-client" || query.Get("response_type") != "code" {
		t.Fatalf("upstream parameters=%v", query)
	}
	attempt, err := f.federation.Attempts.Consume(ctx, tenancydomain.DefaultTenantID, query.Get("state"), time.Now().UTC())
	if err != nil || attempt == nil || attempt.LinkUserID != authRefusalAlice || attempt.ProviderID != authRefusalProviderID || attempt.TenantID != tenancydomain.DefaultTenantID {
		t.Fatalf("bound attempt=%+v err=%v", attempt, err)
	}
	if links := f.linkedProviders(t, authRefusalAlice); len(links) != 0 {
		t.Fatalf("start linked without callback: %v", links)
	}
}
