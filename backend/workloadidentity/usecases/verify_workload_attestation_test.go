package usecases_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	agentmemory "github.com/ambi/idmagic/backend/idmanagement/agent/db_memory"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"

	"github.com/ambi/idmagic/backend/shared/spec"
	workloadmemory "github.com/ambi/idmagic/backend/workloadidentity/db_memory"
	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
	"github.com/ambi/idmagic/backend/workloadidentity/usecases"
	"github.com/ambi/idmagic/backend/workloadidentity/verification_jose"
)

const (
	testTenant   = "tenant-a"
	testIssuer   = "https://issuer.example"
	testAudience = "https://idmagic.example/token"
)

func bigIntBytes(e int) []byte {
	return new(big.Int).SetInt64(int64(e)).Bytes()
}

const testSubject = "spiffe://example.org/ns/prod/sa/worker-1"

func signSVID(t *testing.T, key *rsa.PrivateKey, kid, iss string, iat, exp time.Time) string {
	t.Helper()
	header, err := json.Marshal(map[string]any{"alg": "RS256", "kid": kid})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"iss": iss, "sub": testSubject, "aud": testAudience, "iat": iat.Unix(), "exp": exp.Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(sig)
}

type fixture struct {
	deps    usecases.VerifyWorkloadAttestationDeps
	key     *rsa.PrivateKey
	kid     string
	now     time.Time
	rejects []workloaddomain.WorkloadAttestationRejected
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	kid := "workload-key"
	jwk := map[string]any{
		"kty": "RSA",
		"kid": kid,
		"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(bigIntBytes(key.E)),
	}

	bundleRepo := workloadmemory.NewWorkloadTrustBundleRepository()
	bindingRepo := workloadmemory.NewAgentWorkloadBindingRepository()
	agentRepo := agentmemory.NewAgentRepository()

	f := &fixture{key: key, kid: kid, now: time.Now().UTC()}
	f.deps = usecases.VerifyWorkloadAttestationDeps{
		TrustBundleRepo: bundleRepo,
		BindingRepo:     bindingRepo,
		AgentRepo:       agentRepo,
		SVIDVerifier:    verification_jose.NewVerifier(),
		FetchJWKS: func(context.Context, *workloaddomain.WorkloadTrustBundle) ([]map[string]any, error) {
			return []map[string]any{jwk}, nil
		},
		Emit: func(e spec.DomainEvent) {
			if rej, ok := e.(*workloaddomain.WorkloadAttestationRejected); ok {
				f.rejects = append(f.rejects, *rej)
			}
		},
	}
	return f
}

func (f *fixture) registerBundle(t *testing.T, tenantID string, mutate func(*workloaddomain.WorkloadTrustBundle)) *workloaddomain.WorkloadTrustBundle {
	t.Helper()
	id, err := workloaddomain.NewWorkloadTrustBundleID()
	if err != nil {
		t.Fatal(err)
	}
	b := &workloaddomain.WorkloadTrustBundle{
		ID: id, TenantID: tenantID, Name: "prod-cluster", TrustDomain: "example.org",
		Issuer: testIssuer, AcceptedAudiences: []string{testAudience},
		MaxSubjectTokenTTLSeconds: 3600, Status: workloaddomain.WorkloadTrustBundleStatusEnabled,
		CreatedAt: f.now,
	}
	if mutate != nil {
		mutate(b)
	}
	if err := f.deps.TrustBundleRepo.Save(context.Background(), b); err != nil {
		t.Fatal(err)
	}
	return b
}

func (f *fixture) registerBinding(t *testing.T, bundleID, pattern, agentID string) {
	t.Helper()
	id, err := workloaddomain.NewAgentWorkloadBindingID()
	if err != nil {
		t.Fatal(err)
	}
	b := &workloaddomain.AgentWorkloadBinding{
		ID: id, TenantID: testTenant, TrustBundleID: bundleID, SubjectPattern: pattern,
		AgentID: agentID, Status: workloaddomain.AgentWorkloadBindingStatusEnabled, CreatedAt: f.now,
	}
	if err := f.deps.BindingRepo.Save(context.Background(), b); err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) registerAgent(t *testing.T, id string, active bool) {
	t.Helper()
	status := idmdomain.AgentStatusActive
	if !active {
		status = idmdomain.AgentStatusKilled
	}
	agent := &agentdomain.Agent{
		ID: id, TenantID: testTenant, Name: id, Kind: idmdomain.AgentKindAutonomous,
		OwnerUserID: "user_1", Status: status, CreatedAt: f.now, UpdatedAt: f.now,
	}
	if !active {
		killedAt := f.now
		agent.KilledAt = &killedAt
	}
	if err := f.deps.AgentRepo.Save(context.Background(), agent); err != nil {
		t.Fatal(err)
	}
	if _, err := f.deps.AgentRepo.AddBinding(context.Background(), &agentdomain.AgentCredentialBinding{
		AgentID: id, ClientID: id + "-client", CreatedAt: f.now,
	}); err != nil {
		t.Fatal(err)
	}
}

// TestVerifyWorkloadAttestation_Success — scenario
// `登録済みtrustbundle経由でワークロードトークンをAgent資格情報に交換できる`。
func TestVerifyWorkloadAttestation_Success(t *testing.T) {
	f := newFixture(t)
	bundle := f.registerBundle(t, testTenant, nil)
	f.registerAgent(t, "agent_1", true)
	f.registerBinding(t, bundle.ID, "spiffe://example.org/ns/prod/sa/*", "agent_1")

	token := signSVID(t, f.key, f.kid, testIssuer, f.now, f.now.Add(10*time.Minute))
	grant, err := usecases.VerifyWorkloadAttestation(context.Background(), f.deps, testTenant, usecases.VerifyWorkloadAttestationInput{
		SubjectToken: token,
	}, f.now)
	if err != nil {
		t.Fatalf("VerifyWorkloadAttestation: %v", err)
	}
	if grant.AgentID != "agent_1" || grant.ClientID != "agent_1-client" {
		t.Fatalf("grant = %+v", grant)
	}
	if len(f.rejects) != 0 {
		t.Fatalf("unexpected rejects: %+v", f.rejects)
	}
}

// TestVerifyWorkloadAttestation_UnregisteredIssuer — scenario `未登録issuerは拒否される`。
func TestVerifyWorkloadAttestation_UnregisteredIssuer(t *testing.T) {
	f := newFixture(t)
	token := signSVID(t, f.key, f.kid, "https://unknown-issuer.example", f.now, f.now.Add(10*time.Minute))
	_, err := usecases.VerifyWorkloadAttestation(context.Background(), f.deps, testTenant, usecases.VerifyWorkloadAttestationInput{SubjectToken: token}, f.now)
	if err == nil {
		t.Fatal("expected rejection")
	}
	if len(f.rejects) != 1 || f.rejects[0].Reason != "unregistered_issuer" {
		t.Fatalf("rejects = %+v", f.rejects)
	}
}

// TestVerifyWorkloadAttestation_SpoofedSignature — scenario `署名が不正なattestationは拒否される`。
func TestVerifyWorkloadAttestation_SpoofedSignature(t *testing.T) {
	f := newFixture(t)
	bundle := f.registerBundle(t, testTenant, nil)
	f.registerAgent(t, "agent_1", true)
	f.registerBinding(t, bundle.ID, "spiffe://example.org/ns/prod/sa/*", "agent_1")

	attacker, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	token := signSVID(t, attacker, f.kid, testIssuer, f.now, f.now.Add(10*time.Minute))
	_, err = usecases.VerifyWorkloadAttestation(context.Background(), f.deps, testTenant, usecases.VerifyWorkloadAttestationInput{SubjectToken: token}, f.now)
	if err == nil {
		t.Fatal("expected rejection")
	}
	if len(f.rejects) != 1 || f.rejects[0].Reason != "invalid_signature" {
		t.Fatalf("rejects = %+v", f.rejects)
	}
}

// TestVerifyWorkloadAttestation_Expired — scenario `期限切れのattestationは拒否される`。
func TestVerifyWorkloadAttestation_Expired(t *testing.T) {
	f := newFixture(t)
	bundle := f.registerBundle(t, testTenant, nil)
	f.registerAgent(t, "agent_1", true)
	f.registerBinding(t, bundle.ID, "spiffe://example.org/ns/prod/sa/*", "agent_1")

	token := signSVID(t, f.key, f.kid, testIssuer, f.now.Add(-2*time.Hour), f.now.Add(-time.Hour))
	_, err := usecases.VerifyWorkloadAttestation(context.Background(), f.deps, testTenant, usecases.VerifyWorkloadAttestationInput{SubjectToken: token}, f.now)
	if err == nil {
		t.Fatal("expected rejection")
	}
	if len(f.rejects) != 1 || f.rejects[0].Reason != "expired" {
		t.Fatalf("rejects = %+v", f.rejects)
	}
}

// TestVerifyWorkloadAttestation_AmbiguousMatch — scenario
// `複数bindingに曖昧にマッチするsubjectは拒否される` (binding collision)。
func TestVerifyWorkloadAttestation_AmbiguousMatch(t *testing.T) {
	f := newFixture(t)
	bundle := f.registerBundle(t, testTenant, nil)
	f.registerAgent(t, "agent_a", true)
	f.registerAgent(t, "agent_b", true)
	f.registerBinding(t, bundle.ID, "spiffe://example.org/ns/prod/sa/*", "agent_a")
	f.registerBinding(t, bundle.ID, "spiffe://example.org/ns/prod/sa/worker-*", "agent_b")

	token := signSVID(t, f.key, f.kid, testIssuer, f.now, f.now.Add(10*time.Minute))
	_, err := usecases.VerifyWorkloadAttestation(context.Background(), f.deps, testTenant, usecases.VerifyWorkloadAttestationInput{SubjectToken: token}, f.now)
	if err == nil {
		t.Fatal("expected rejection")
	}
	if len(f.rejects) != 1 || f.rejects[0].Reason != "ambiguous_match" {
		t.Fatalf("rejects = %+v", f.rejects)
	}
}

// TestVerifyWorkloadAttestation_KilledAgent — scenario `束縛先AgentがKilled後は拒否される`。
func TestVerifyWorkloadAttestation_KilledAgent(t *testing.T) {
	f := newFixture(t)
	bundle := f.registerBundle(t, testTenant, nil)
	f.registerAgent(t, "agent_1", false)
	f.registerBinding(t, bundle.ID, "spiffe://example.org/ns/prod/sa/*", "agent_1")

	token := signSVID(t, f.key, f.kid, testIssuer, f.now, f.now.Add(10*time.Minute))
	_, err := usecases.VerifyWorkloadAttestation(context.Background(), f.deps, testTenant, usecases.VerifyWorkloadAttestationInput{SubjectToken: token}, f.now)
	if err == nil {
		t.Fatal("expected rejection")
	}
	if len(f.rejects) != 1 || f.rejects[0].Reason != "agent_not_active" {
		t.Fatalf("rejects = %+v", f.rejects)
	}
}

// TestVerifyWorkloadAttestation_CrossTenant — scenario `他テナントのtrustbundleは利用できない`。
func TestVerifyWorkloadAttestation_CrossTenant(t *testing.T) {
	f := newFixture(t)
	// tenant-b に登録した bundle は tenant-a のコンテキストからは見えない。
	f.registerBundle(t, "tenant-b", nil)

	token := signSVID(t, f.key, f.kid, testIssuer, f.now, f.now.Add(10*time.Minute))
	_, err := usecases.VerifyWorkloadAttestation(context.Background(), f.deps, testTenant, usecases.VerifyWorkloadAttestationInput{SubjectToken: token}, f.now)
	if err == nil {
		t.Fatal("expected rejection")
	}
	if len(f.rejects) != 1 || f.rejects[0].Reason != "unregistered_issuer" {
		t.Fatalf("rejects = %+v", f.rejects)
	}
}

// TestVerifyWorkloadAttestation_DisabledTrustBundle — 管理者が無効化した bundle は
// 以後の交換に使えない (fail-closed)。
func TestVerifyWorkloadAttestation_DisabledTrustBundle(t *testing.T) {
	f := newFixture(t)
	bundle := f.registerBundle(t, testTenant, func(b *workloaddomain.WorkloadTrustBundle) {
		b.Status = workloaddomain.WorkloadTrustBundleStatusDisabled
	})
	f.registerAgent(t, "agent_1", true)
	f.registerBinding(t, bundle.ID, "spiffe://example.org/ns/prod/sa/*", "agent_1")

	token := signSVID(t, f.key, f.kid, testIssuer, f.now, f.now.Add(10*time.Minute))
	_, err := usecases.VerifyWorkloadAttestation(context.Background(), f.deps, testTenant, usecases.VerifyWorkloadAttestationInput{SubjectToken: token}, f.now)
	if err == nil {
		t.Fatal("expected rejection")
	}
	if len(f.rejects) != 1 || f.rejects[0].Reason != "trust_bundle_disabled" {
		t.Fatalf("rejects = %+v", f.rejects)
	}
}
