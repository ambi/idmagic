package usecases_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
	"github.com/ambi/idmagic/backend/workloadidentity/internal/usecases"
)

const clientAssertionAudience = "https://idmagic.example/realms/default"

// signClientSVID は SPIRE の形の JWT-SVID (iss なし) に mutate を適用して署名する。
func (f *fixture) signClientSVID(t *testing.T, mutate func(map[string]any)) string {
	t.Helper()
	header, err := json.Marshal(map[string]any{"alg": "RS256", "kid": f.kid, "typ": "JWT"})
	if err != nil {
		t.Fatal(err)
	}
	claims := map[string]any{
		"sub": testSubject, "aud": []string{clientAssertionAudience}, "jti": "svid-1",
		"iat": f.now.Unix(), "exp": f.now.Add(5 * time.Minute).Unix(),
	}
	if mutate != nil {
		mutate(claims)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	sig, err := rsa.SignPKCS1v15(rand.Reader, f.key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func (f *fixture) authenticate(t *testing.T, clientID, assertion string) (*workloaddomain.WorkloadClientAuthentication, error) {
	t.Helper()
	return usecases.VerifyWorkloadClientAssertion(context.Background(), f.deps, testTenant, usecases.VerifyWorkloadClientAssertionInput{
		ClientID: clientID, Assertion: assertion, ExpectedAudience: clientAssertionAudience,
	}, f.now)
}

// clientAssertionFixture は、発行者を登録した spiffe の信頼設定と、agent_1 (agent_1-client を束縛)
// への関連付けを置く。発行者を登録しておくのは、`iss` で信頼設定を選ばないことを確かめるため。
func clientAssertionFixture(t *testing.T) (*fixture, *workloaddomain.WorkloadTrustBundle) {
	t.Helper()
	f := newFixture(t)
	bundle := f.registerBundle(t, testTenant, spiffeBundle(""))
	f.registerAgent(t, "agent_1", true)
	f.registerAgent(t, "agent_2", true)
	f.registerBinding(t, bundle.ID, "spiffe://example.org/ns/prod/sa/*", "agent_1")
	return f, bundle
}

//spec:covers REQ-WORKLOADIDENTITY-017: 束縛したクライアントの JWT-SVID は、関連付けの Agent、信頼設定、SPIFFE ID、jti、exp を返し、拒否イベントを出さない。
func TestVerifyWorkloadClientAssertion_Success(t *testing.T) {
	f, bundle := clientAssertionFixture(t)
	got, err := f.authenticate(t, "agent_1-client", f.signClientSVID(t, nil))
	if err != nil {
		t.Fatalf("VerifyWorkloadClientAssertion: %v (rejects %+v)", err, f.rejects)
	}
	want := workloaddomain.WorkloadClientAuthentication{
		AgentID: "agent_1", ClientID: "agent_1-client", TrustBundleID: bundle.ID, BindingID: got.BindingID,
		SpiffeID: testSubject, JTI: "svid-1", ExpiresAt: time.Unix(f.now.Add(5*time.Minute).Unix(), 0).UTC(),
	}
	if got.BindingID == "" || *got != want {
		t.Fatalf("result = %+v, want %+v", *got, want)
	}
	if len(f.rejects) != 0 {
		t.Fatalf("unexpected rejects: %+v", f.rejects)
	}
}

//spec:covers REQ-WORKLOADIDENTITY-017: aud が発行者だけでない、別の Agent のクライアント、iss だけが登録された発行者を指す、などの拒否は理由と用途 client_authentication を記録する。
func TestVerifyWorkloadClientAssertion_Rejections(t *testing.T) {
	cases := map[string]struct {
		clientID string
		mutate   func(map[string]any)
		bundle   func(*workloaddomain.WorkloadTrustBundle)
		reason   string
	}{
		"信頼設定の audience だけ":    {"agent_1-client", func(c map[string]any) { c["aud"] = testAudience }, nil, "audience_mismatch"},
		"発行者とほかの audience":     {"agent_1-client", func(c map[string]any) { c["aud"] = []string{clientAssertionAudience, testAudience} }, nil, "audience_mismatch"},
		"別の Agent のクライアント":     {"agent_2-client", nil, nil, "client_not_bound"},
		"Agent に束縛していないクライアント": {"unknown-client", nil, nil, "client_not_bound"},
		"SPIFFE ID でない sub":    {"agent_1-client", func(c map[string]any) { c["sub"] = "worker" }, nil, "invalid_spiffe_id"},
		"未登録のトラストドメイン":         {"agent_1-client", func(c map[string]any) { c["sub"] = "spiffe://other.org/ns/prod/sa/w" }, nil, "unregistered_issuer"},
		"関連付けに一致しない":           {"agent_1-client", func(c map[string]any) { c["sub"] = "spiffe://example.org/ns/dev/sa/w" }, nil, "no_binding_match"},
		"期限切れ":                 {"agent_1-client", func(c map[string]any) { c["exp"] = time.Now().Add(-10 * time.Minute).Unix() }, nil, "expired"},
		"無効の信頼設定": {"agent_1-client", nil, func(b *workloaddomain.WorkloadTrustBundle) {
			b.Status = workloaddomain.WorkloadTrustBundleStatusDisabled
		}, "trust_bundle_disabled"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			bundle := f.registerBundle(t, testTenant, func(b *workloaddomain.WorkloadTrustBundle) {
				spiffeBundle("")(b)
				if tc.bundle != nil {
					tc.bundle(b)
				}
			})
			f.registerAgent(t, "agent_1", true)
			f.registerAgent(t, "agent_2", true)
			f.registerBinding(t, bundle.ID, "spiffe://example.org/ns/prod/sa/*", "agent_1")
			got, err := f.authenticate(t, tc.clientID, f.signClientSVID(t, tc.mutate))
			if err == nil || got != nil {
				t.Fatalf("result=%+v err=%v, want rejection", got, err)
			}
			if len(f.rejects) != 1 || f.rejects[0].Reason != tc.reason ||
				f.rejects[0].Usage != workloaddomain.AttestationUsageClientAuthentication {
				t.Fatalf("rejects = %+v, want one %s for client_authentication", f.rejects, tc.reason)
			}
		})
	}
}

// 発行者で登録した oidc の信頼設定は、`iss` が一致しても選ばない (ドラフトの 8.3 節)。
//
//spec:covers REQ-WORKLOADIDENTITY-017: iss に一致する信頼設定があっても、トラストドメインの spiffe の信頼設定がなければ unregistered_issuer で拒否する。
func TestVerifyWorkloadClientAssertion_DoesNotSelectByIssuer(t *testing.T) {
	f := newFixture(t)
	bundle := f.registerBundle(t, testTenant, nil) // oidc、発行者は testIssuer
	f.registerAgent(t, "agent_1", true)
	f.registerBinding(t, bundle.ID, "spiffe://example.org/ns/prod/sa/*", "agent_1")
	got, err := f.authenticate(t, "agent_1-client", f.signClientSVID(t, func(c map[string]any) { c["iss"] = testIssuer }))
	if err == nil || got != nil || len(f.rejects) != 1 || f.rejects[0].Reason != "unregistered_issuer" {
		t.Fatalf("result=%+v err=%v rejects=%+v, want unregistered_issuer", got, err, f.rejects)
	}
}

// Token Exchange の拒否は用途 token_exchange を記録する。
//
//spec:covers REQ-WORKLOADIDENTITY-001
func TestVerifyWorkloadAttestation_RejectionRecordsTokenExchangeUsage(t *testing.T) {
	f := newFixture(t)
	if _, err := f.exchange(t, f.signClaims(t, "https://unknown.example", testSubject)); err == nil {
		t.Fatal("expected rejection")
	}
	if len(f.rejects) != 1 || f.rejects[0].Usage != workloaddomain.AttestationUsageTokenExchange {
		t.Fatalf("rejects = %+v, want token_exchange usage", f.rejects)
	}
}
