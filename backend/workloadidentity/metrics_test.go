package workloadidentity

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/idmanagement"
	"github.com/ambi/idmagic/backend/shared/spec"
	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// 組み立てた検証器は、取得に失敗して信頼設定に保存した鍵を使ったとき、モジュールが公開する
// 指標の source="stored" を増やす。観測者を配線しない組み立ては、使っても数えない。
//
//spec:covers REQ-WORKLOADIDENTITY-018: 保存した鍵を使った検証を、指標の出所 stored で区別する。
func TestTokenVerifierCountsStoredKeyFallback(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)
	m := NewMemoryModule()
	uri := "https://issuer.example/jwks.json"
	fetchedAt := now.Add(-time.Hour)
	bundle := &workloaddomain.WorkloadTrustBundle{
		ID: "bundle-1", TenantID: "tenant-a", Name: "prod", Profile: workloaddomain.WorkloadTrustBundleProfileOIDC,
		TrustDomain: "example.org", Issuer: "https://issuer.example", JWKSURI: &uri,
		AcceptedAudiences: []string{"https://idmagic.example/token"}, MaxSubjectTokenTTLSeconds: 3600,
		Status: workloaddomain.WorkloadTrustBundleStatusEnabled, CreatedAt: now,
	}
	if err := m.TrustBundleRepo.Save(ctx, bundle); err != nil {
		t.Fatal(err)
	}
	if err := m.TrustBundleRepo.RecordJWKSFetched(ctx, "tenant-a", "bundle-1", &uri, workloaddomain.JWKSSnapshot{
		Keys: []map[string]any{{"kty": "RSA", "kid": "k1", "n": "AQAB", "e": "AQAB"}}, FetchedAt: fetchedAt,
	}); err != nil {
		t.Fatal(err)
	}
	down := func(context.Context, *workloaddomain.WorkloadTrustBundle) ([]map[string]any, error) {
		return nil, errors.New("jwks endpoint unreachable")
	}
	verifier := m.TokenVerifier(idmanagement.NewMemoryModule().AgentRepo, down, func(spec.DomainEvent) {})
	// 署名は検証できなくてよい。鍵の出所は署名の検証より前に決まる。
	segment := func(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }
	token := segment(`{"alg":"RS256","kid":"k1"}`) + "." + segment(`{"iss":"https://issuer.example","sub":"workload"}`) + "." + segment("sig")
	_, _ = verifier.VerifyWorkloadToken(ctx, "tenant-a", "client-1", token, now)

	counter, ok := m.JWKSFallbackCollector().(*prometheus.CounterVec)
	if !ok {
		t.Fatalf("collector = %T, want the fallback counter", m.JWKSFallbackCollector())
	}
	if got := testutil.ToFloat64(counter.WithLabelValues("stored")); got != 1 {
		t.Fatalf("stored fallback count = %v, want 1", got)
	}
	if got := testutil.ToFloat64(counter.WithLabelValues("memory")); got != 0 {
		t.Fatalf("memory fallback count = %v, want 0", got)
	}
}
