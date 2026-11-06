package usecases_test

// 主要ユースケース追跡: REQ-WORKLOADIDENTITY-011。

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

const otherTenant = "tenant-b"

// signClaims は iss を省けるように、クレームをそのまま RS256 の JWT にする。
func (f *fixture) signClaims(t *testing.T, iss, sub string) string {
	t.Helper()
	header, err := json.Marshal(map[string]any{"alg": "RS256", "kid": f.kid, "typ": "JWT"})
	if err != nil {
		t.Fatal(err)
	}
	claims := map[string]any{
		"sub": sub, "aud": testAudience, "iat": f.now.Unix(), "exp": f.now.Add(5 * time.Minute).Unix(),
	}
	if iss != "" {
		claims["iss"] = iss
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

func spiffeBundle(issuer string) func(*workloaddomain.WorkloadTrustBundle) {
	return func(b *workloaddomain.WorkloadTrustBundle) {
		b.Profile = workloaddomain.WorkloadTrustBundleProfileSPIFFE
		b.Issuer = issuer
		b.TrustDomain = "example.org"
	}
}

func (f *fixture) exchange(t *testing.T, token string) (*workloaddomain.WorkloadIdentityGrant, error) {
	t.Helper()
	return usecases.VerifyWorkloadAttestation(context.Background(), f.deps, testTenant,
		usecases.VerifyWorkloadAttestationInput{SubjectToken: token}, f.now)
}

// 既定の SPIRE が発行する、`iss` のない JWT-SVID を、`sub` のトラストドメインに一致する
// `spiffe` の信頼設定で検証し、関連付け先 Agent の資格情報を返す。
//
//spec:covers REQ-WORKLOADIDENTITY-011
//spec:covers REQ-WORKLOADIDENTITY-001
func TestVerifyWorkloadAttestation_SPIFFEProfileAcceptsSVIDWithoutIssuer(t *testing.T) {
	f := newFixture(t)
	bundle := f.registerBundle(t, testTenant, spiffeBundle(""))
	f.registerAgent(t, "agent_1", true)
	f.registerBinding(t, bundle.ID, "spiffe://example.org/ns/prod/sa/*", "agent_1")

	grant, err := f.exchange(t, f.signClaims(t, "", testSubject))
	if err != nil {
		t.Fatalf("VerifyWorkloadAttestation: %v (rejects %+v)", err, f.rejects)
	}
	if grant.AgentID != "agent_1" || grant.ClientID != "agent_1-client" || grant.TrustBundleID != bundle.ID {
		t.Fatalf("grant = %+v", grant)
	}
	if len(f.rejects) != 0 {
		t.Fatalf("unexpected rejects: %+v", f.rejects)
	}
}

// 発行者で一致した `spiffe` の信頼設定でも、`sub` が別のトラストドメインの主体なら、
// `iss` と鍵が一致していても拒否する。関連付けのパターンがその主体に一致していても通さない。
//
//spec:covers REQ-WORKLOADIDENTITY-011
func TestVerifyWorkloadAttestation_SPIFFEProfileRejectsForeignTrustDomain(t *testing.T) {
	f := newFixture(t)
	bundle := f.registerBundle(t, testTenant, spiffeBundle(testIssuer))
	f.registerAgent(t, "agent_1", true)
	f.registerBinding(t, bundle.ID, "spiffe://other.org/ns/prod/sa/*", "agent_1")

	grant, err := f.exchange(t, f.signClaims(t, testIssuer, "spiffe://other.org/ns/prod/sa/worker-1"))
	assertRefusedWithoutGrant(t, f, grant, err, "trust_domain_mismatch")
	if f.rejects[0].TrustBundleID != bundle.ID {
		t.Fatalf("reject trust bundle = %q, want %q", f.rejects[0].TrustBundleID, bundle.ID)
	}
}

//spec:covers REQ-WORKLOADIDENTITY-011
func TestVerifyWorkloadAttestation_SPIFFEProfileRejectsInvalidSpiffeID(t *testing.T) {
	for _, sub := range []string{"system:serviceaccount:prod:worker", "spiffe://Example.org/ns/prod", "spiffe://example.org/ns/../sa"} {
		t.Run(sub, func(t *testing.T) {
			f := newFixture(t)
			bundle := f.registerBundle(t, testTenant, spiffeBundle(testIssuer))
			f.registerAgent(t, "agent_1", true)
			f.registerBinding(t, bundle.ID, "*", "agent_1")

			grant, err := f.exchange(t, f.signClaims(t, testIssuer, sub))
			assertRefusedWithoutGrant(t, f, grant, err, "invalid_spiffe_id")
		})
	}
}

// `spiffe` の信頼設定に発行者を登録していなければ、`iss` が付いていても確かめず、
// トラストドメインで信頼設定を探す (SPIRE で `jwt_issuer` を設定した場合)。
//
//spec:covers REQ-WORKLOADIDENTITY-011
func TestVerifyWorkloadAttestation_SPIFFEProfileIgnoresIssWithoutRegisteredIssuer(t *testing.T) {
	f := newFixture(t)
	bundle := f.registerBundle(t, testTenant, spiffeBundle(""))
	f.registerAgent(t, "agent_1", true)
	f.registerBinding(t, bundle.ID, "spiffe://example.org/ns/prod/sa/*", "agent_1")

	if _, err := f.exchange(t, f.signClaims(t, "https://spire.example", testSubject)); err != nil {
		t.Fatalf("VerifyWorkloadAttestation: %v (rejects %+v)", err, f.rejects)
	}
}

// `oidc` の信頼設定はトラストドメインで探されず、`sub` を SPIFFE ID として検証しない。
//
//spec:covers REQ-WORKLOADIDENTITY-011
//spec:covers REQ-WORKLOADIDENTITY-002
func TestVerifyWorkloadAttestation_OIDCProfileIsNotSPIFFE(t *testing.T) {
	t.Run("not found by trust domain", func(t *testing.T) {
		f := newFixture(t)
		f.registerBundle(t, testTenant, nil) // oidc、トラストドメインのラベルは example.org
		grant, err := f.exchange(t, f.signClaims(t, "", testSubject))
		assertRefusedWithoutGrant(t, f, grant, err, "unregistered_issuer")
	})
	t.Run("accepts a non-SPIFFE subject", func(t *testing.T) {
		f := newFixture(t)
		bundle := f.registerBundle(t, testTenant, nil)
		f.registerAgent(t, "agent_1", true)
		f.registerBinding(t, bundle.ID, "system:serviceaccount:prod:*", "agent_1")
		if _, err := f.exchange(t, f.signClaims(t, testIssuer, "system:serviceaccount:prod:worker")); err != nil {
			t.Fatalf("VerifyWorkloadAttestation: %v (rejects %+v)", err, f.rejects)
		}
	})
}

// トラストドメインによる探索も、要求を受けたテナントの信頼設定に閉じる。
//
//spec:covers REQ-WORKLOADIDENTITY-007
func TestVerifyWorkloadAttestation_SPIFFELookupStaysInTenant(t *testing.T) {
	f := newFixture(t)
	f.registerBundle(t, otherTenant, spiffeBundle(""))
	grant, err := f.exchange(t, f.signClaims(t, "", testSubject))
	assertRefusedWithoutGrant(t, f, grant, err, "unregistered_issuer")
}

//spec:covers REQ-WORKLOADIDENTITY-001
func TestVerifyWorkloadAttestation_SPIFFELookupHonorsDisabledBundle(t *testing.T) {
	f := newFixture(t)
	bundle := f.registerBundle(t, testTenant, func(b *workloaddomain.WorkloadTrustBundle) {
		spiffeBundle("")(b)
		b.Status = workloaddomain.WorkloadTrustBundleStatusDisabled
	})
	grant, err := f.exchange(t, f.signClaims(t, "", testSubject))
	assertRefusedWithoutGrant(t, f, grant, err, "trust_bundle_disabled")
	if f.rejects[0].TrustBundleID != bundle.ID {
		t.Fatalf("reject trust bundle = %q", f.rejects[0].TrustBundleID)
	}
}
