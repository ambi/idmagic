package usecases_test

// 主要ユースケース追跡: REQ-WORKLOADIDENTITY-001。

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

// assertRefusedWithoutGrant は拒否の具体例が要求する 2 つの観測を 1 か所に置く。呼び出し元
// が受け取る拒否応答と、拒否が防いだ効果 — Agent 資格情報の元になる WorkloadIdentityGrant
// が返らないこと — の双方である。
//
// 仕様の WorkloadAttestationRejectedError は本体を持たない領域条件なので、Go 側に対応する
// 型は無い。具体例が名指しする `reason` は WorkloadAttestationRejected イベントに載るため、
// 理由の区別はイベント側で観測する。
func assertRefusedWithoutGrant(
	t *testing.T, f *fixture, grant *workloaddomain.WorkloadIdentityGrant, err error, reason string,
) {
	t.Helper()
	if err == nil {
		t.Fatal("expected rejection")
	}
	if grant != nil {
		t.Fatalf("拒否されたのに WorkloadIdentityGrant が返った: %+v", grant)
	}
	if len(f.rejects) != 1 || f.rejects[0].Reason != reason {
		t.Fatalf("rejects = %+v, want exactly one with reason %q", f.rejects, reason)
	}
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

// EX-WORKLOADIDENTITY-001-01: `Enabled` の信頼設定と、主体パターンが `sub` に一致する
// `Enabled` の関連付けが揃っているとき、VerifyWorkloadAttestation は関連付け先 Agent の
// `client_id` を持つ WorkloadIdentityGrant を返し、拒否イベントを 1 件も出さない。
// 固定しているのは、返る資格情報が「パターンに一致した関連付けの先」であることである。
//
// 具体例の 2 つ目の Then（短命なアクセストークンの発行）は HTTP 入口が持つので、
// backend/oauth2/handlers_http の TestTokenExchangeIssuesWorkloadCredential が観測する。
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

// EX-WORKLOADIDENTITY-002-01: `iss` に対応する WorkloadTrustBundle がテナントに無ければ、
// 署名を検べる前に `reason=unregistered_issuer` で拒否し、資格情報を返さない。
// 固定しているのは、発行者の登録が交換の前提条件であることである。
//
// 素通りすれば、誰でも自分の発行者を名乗る JWT を持ち込むだけで Agent の資格情報を得る。
// 信頼設定の登録は、その発行者を信じると管理者が宣言した唯一の記録である。
func TestVerifyWorkloadAttestation_UnregisteredIssuer(t *testing.T) {
	f := newFixture(t)
	token := signSVID(t, f.key, f.kid, "https://unknown-issuer.example", f.now, f.now.Add(10*time.Minute))
	grant, err := usecases.VerifyWorkloadAttestation(context.Background(), f.deps, testTenant, usecases.VerifyWorkloadAttestationInput{SubjectToken: token}, f.now)
	assertRefusedWithoutGrant(t, f, grant, err, "unregistered_issuer")
}

// EX-WORKLOADIDENTITY-003-01: `iss` は登録済みの発行者を指しているが、その信頼設定の JWKS
// では署名を検証できない JWT は `reason=invalid_signature` で拒否され、資格情報は返らない。
// 固定しているのは、発行者の一致だけでは足りず、登録済みの鍵による署名が要ることである。
//
// 攻撃者は `iss` を正しく詐称できる。鍵だけが詐称できない。ここが素通りすれば、
// 信頼設定の登録は発行者名の照合に退化する。
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
	grant, err := usecases.VerifyWorkloadAttestation(context.Background(), f.deps, testTenant, usecases.VerifyWorkloadAttestationInput{SubjectToken: token}, f.now)
	assertRefusedWithoutGrant(t, f, grant, err, "invalid_signature")
}

// EX-WORKLOADIDENTITY-004-01: `exp` が過去の JWT-SVID は、署名も発行者も関連付けも正しくても
// `reason=expired` で拒否され、資格情報は返らない。固定しているのは、有効期間の判定が
// 他の検証に合格したことで飛ばされないことである。
//
// 素通りすれば、一度漏れた SVID が期限に関係なく使い続けられる。短い有効期間は、
// ワークロードの資格情報が漏洩したときの被害を限る唯一の仕組みである。
func TestVerifyWorkloadAttestation_Expired(t *testing.T) {
	f := newFixture(t)
	bundle := f.registerBundle(t, testTenant, nil)
	f.registerAgent(t, "agent_1", true)
	f.registerBinding(t, bundle.ID, "spiffe://example.org/ns/prod/sa/*", "agent_1")

	token := signSVID(t, f.key, f.kid, testIssuer, f.now.Add(-2*time.Hour), f.now.Add(-time.Hour))
	grant, err := usecases.VerifyWorkloadAttestation(context.Background(), f.deps, testTenant, usecases.VerifyWorkloadAttestationInput{SubjectToken: token}, f.now)
	assertRefusedWithoutGrant(t, f, grant, err, "expired")
}

// EX-WORKLOADIDENTITY-005-01: `sub` が 2 つの `Enabled` な関連付けの主体パターンに同時に
// 一致するとき、どちらかを選ばずに `reason=ambiguous_match` で拒否し、資格情報を返さない。
// 固定しているのは、Agent が一意に決まらない限り交換しないことである。
//
// 素通りすれば、どの Agent の資格情報が返るかはパターンの評価順という実装の都合で決まる。
// 一方のパターンをあとから足しただけで、既存のワークロードが別の Agent になりうる。
func TestVerifyWorkloadAttestation_AmbiguousMatch(t *testing.T) {
	f := newFixture(t)
	bundle := f.registerBundle(t, testTenant, nil)
	f.registerAgent(t, "agent_a", true)
	f.registerAgent(t, "agent_b", true)
	f.registerBinding(t, bundle.ID, "spiffe://example.org/ns/prod/sa/*", "agent_a")
	f.registerBinding(t, bundle.ID, "spiffe://example.org/ns/prod/sa/worker-*", "agent_b")

	token := signSVID(t, f.key, f.kid, testIssuer, f.now, f.now.Add(10*time.Minute))
	grant, err := usecases.VerifyWorkloadAttestation(context.Background(), f.deps, testTenant, usecases.VerifyWorkloadAttestationInput{SubjectToken: token}, f.now)
	assertRefusedWithoutGrant(t, f, grant, err, "ambiguous_match")
}

// EX-WORKLOADIDENTITY-006-01: 関連付けの対応先 Agent が `killed` に遷移した後は、信頼設定も
// 関連付けも `Enabled` のままでも `reason=agent_not_active` で拒否し、資格情報を返さない。
// 固定しているのは、KillAgent が関連付けを消さなくても交換を止めることである。
//
// KillAgent は Agent を止める操作であって、信頼設定を畳む操作ではない。ここが素通りすれば、
// 停止した Agent の資格情報を、停止したことを知らない経路から取り直せてしまう。
func TestVerifyWorkloadAttestation_KilledAgent(t *testing.T) {
	f := newFixture(t)
	bundle := f.registerBundle(t, testTenant, nil)
	f.registerAgent(t, "agent_1", false)
	f.registerBinding(t, bundle.ID, "spiffe://example.org/ns/prod/sa/*", "agent_1")

	token := signSVID(t, f.key, f.kid, testIssuer, f.now, f.now.Add(10*time.Minute))
	grant, err := usecases.VerifyWorkloadAttestation(context.Background(), f.deps, testTenant, usecases.VerifyWorkloadAttestationInput{SubjectToken: token}, f.now)
	assertRefusedWithoutGrant(t, f, grant, err, "agent_not_active")
}

// EX-WORKLOADIDENTITY-007-01: 同じ発行者の WorkloadTrustBundle が別テナントに登録されていても、
// 当のテナントの実行コンテキストからは見えず、`reason=unregistered_issuer` で拒否される。
// 固定しているのは、他テナントの登録内容が参照されないことである。理由が
// `unregistered_issuer` であること自体が、他テナントの登録を見つけたうえで弾いたのではなく、
// そもそも見えていないことを示す。
//
// 素通りすれば、あるテナントが発行者を登録するだけで、同じ発行者を使う他テナントの
// ワークロードが自テナントの Agent に化ける。テナント境界は信頼設定の探索範囲そのものである。
func TestVerifyWorkloadAttestation_CrossTenant(t *testing.T) {
	f := newFixture(t)
	// tenant-b に登録した bundle は tenant-a のコンテキストからは見えない。
	f.registerBundle(t, "tenant-b", nil)

	token := signSVID(t, f.key, f.kid, testIssuer, f.now, f.now.Add(10*time.Minute))
	grant, err := usecases.VerifyWorkloadAttestation(context.Background(), f.deps, testTenant, usecases.VerifyWorkloadAttestationInput{SubjectToken: token}, f.now)
	assertRefusedWithoutGrant(t, f, grant, err, "unregistered_issuer")
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
	grant, err := usecases.VerifyWorkloadAttestation(context.Background(), f.deps, testTenant, usecases.VerifyWorkloadAttestationInput{SubjectToken: token}, f.now)
	assertRefusedWithoutGrant(t, f, grant, err, "trust_bundle_disabled")
}
