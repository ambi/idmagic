package usecases_test

// 主要ユースケース追跡: REQ-WORKLOADIDENTITY-018。

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	workloaddomain "github.com/ambi/idmagic/backend/workloadidentity/domain"
	workloadports "github.com/ambi/idmagic/backend/workloadidentity/internal/ports"
	workloadusecases "github.com/ambi/idmagic/backend/workloadidentity/internal/usecases"
	"github.com/ambi/idmagic/backend/workloadidentity/internal/verification_jose"
)

const testJWKSURI = "https://issuer.example/jwks.json"

var errJWKSEndpointDown = errors.New("jwks endpoint unreachable")

// lastGoodFixture は jwks_uri を持つ信頼設定と、それに一致する Agent の関連付けを用意する。
func lastGoodFixture(t *testing.T) (*fixture, *workloaddomain.WorkloadTrustBundle) {
	t.Helper()
	f := newFixture(t)
	bundle := f.registerBundle(t, testTenant, func(b *workloaddomain.WorkloadTrustBundle) {
		uri := testJWKSURI
		b.JWKSURI = &uri
	})
	f.registerAgent(t, "agent_1", true)
	f.registerBinding(t, bundle.ID, "spiffe://example.org/ns/prod/sa/*", "agent_1")
	return f, bundle
}

func (f *fixture) exchangeAt(t *testing.T, at time.Time) (*workloaddomain.WorkloadIdentityGrant, error) {
	t.Helper()
	f.rejects = nil
	token := signSVID(t, f.key, f.kid, testIssuer, at, at.Add(10*time.Minute))
	return workloadusecases.VerifyWorkloadAttestation(context.Background(), f.deps, testTenant, workloadusecases.VerifyWorkloadAttestationInput{
		ClientID: "agent_1-client", SubjectToken: token,
	}, at)
}

// restartWithEndpointDown は、メモリの鍵を持たない新しいプロセスで、取得元に届かない状況を作る。
func (f *fixture) restartWithEndpointDown() {
	f.deps.SVIDVerifier = verification_jose.NewVerifier(nil)
	f.deps.FetchJWKS = func(context.Context, *workloaddomain.WorkloadTrustBundle) ([]map[string]any, error) {
		return nil, errJWKSEndpointDown
	}
}

// 取得に成功した交換が鍵を信頼設定へ保存し、再起動したプロセスは取得元に届かなくても、
// 保存した取得時刻から 24 時間ちょうどまではその鍵で交換し、それを過ぎると拒否する。
// 保存しない実装は再起動直後に拒否し、境界を外した実装は 24 時間を過ぎても交換する。
//
//spec:covers REQ-WORKLOADIDENTITY-018: 再起動の前後を問わず、jwks_uri から最後に取得できた鍵を取得から 24 時間以内に限って使い、過ぎたら jwks_unavailable で拒否する。
func TestVerifyWorkloadAttestation_UsesStoredKeysAfterRestart(t *testing.T) {
	for _, tc := range []struct {
		name   string
		after  time.Duration
		reason string
	}{
		{name: "直後", after: time.Minute},
		{name: "24 時間ちょうど", after: 24 * time.Hour},
		{name: "24 時間を 1 秒過ぎた", after: 24*time.Hour + time.Second, reason: "jwks_unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, _ := lastGoodFixture(t)
			if _, err := f.exchangeAt(t, f.now); err != nil {
				t.Fatalf("取得できる間の交換: %v", err)
			}
			f.restartWithEndpointDown()

			grant, err := f.exchangeAt(t, f.now.Add(tc.after))
			if tc.reason != "" {
				assertRefusedWithoutGrant(t, f, grant, err, tc.reason)
				return
			}
			if err != nil || grant == nil || grant.AgentID != "agent_1" {
				t.Fatalf("再起動後の交換 = (%+v, %v), want agent_1 from the stored keys", grant, err)
			}
		})
	}

	t.Run("保存した鍵がなければ再起動直後に拒否する", func(t *testing.T) {
		f, _ := lastGoodFixture(t)
		f.restartWithEndpointDown()
		grant, err := f.exchangeAt(t, f.now)
		assertRefusedWithoutGrant(t, f, grant, err, "jwks_unavailable")
	})
}

// 保存は、取得した集合が変わったときか、保存から 1 時間を過ぎたときだけ行う。
// 毎回書く実装は時刻が進み、書かない実装は鍵の変化を保存しない。
//
//spec:covers REQ-WORKLOADIDENTITY-018: 取得した鍵の集合が保存した集合と異なるか、保存から 1 時間を過ぎたときだけ、集合と取得の時刻を保存する。
func TestVerifyWorkloadAttestation_StoresFetchedKeysOnChangeOrHourly(t *testing.T) {
	f, bundle := lastGoodFixture(t)
	signing, err := f.deps.FetchJWKS(context.Background(), bundle)
	if err != nil {
		t.Fatal(err)
	}
	rotated := append([]map[string]any{{"kty": "RSA", "kid": "rotated-in", "n": "AQAB", "e": "AQAB"}}, signing...)
	var keys []map[string]any
	f.deps.FetchJWKS = func(context.Context, *workloaddomain.WorkloadTrustBundle) ([]map[string]any, error) {
		return keys, nil
	}
	stored := func() *workloaddomain.WorkloadTrustBundle {
		t.Helper()
		b, err := f.deps.TrustBundleRepo.FindByID(context.Background(), testTenant, bundle.ID)
		if err != nil || b == nil {
			t.Fatalf("FindByID = (%+v, %v)", b, err)
		}
		return b
	}
	steps := []struct {
		name      string
		after     time.Duration
		addKey    bool
		wantSaved time.Duration
	}{
		{name: "最初の取得", after: 0, wantSaved: 0},
		{name: "1 時間以内の同じ集合", after: time.Hour, wantSaved: 0},
		{name: "1 時間を過ぎた同じ集合", after: time.Hour + time.Second, wantSaved: time.Hour + time.Second},
		{name: "集合の変化", after: time.Hour + 2*time.Second, addKey: true, wantSaved: time.Hour + 2*time.Second},
	}
	for _, step := range steps {
		keys = signing
		if step.addKey {
			keys = rotated
		}
		if _, err := f.exchangeAt(t, f.now.Add(step.after)); err != nil {
			t.Fatalf("%s: exchange: %v", step.name, err)
		}
		got := stored()
		want := f.now.Add(step.wantSaved)
		if got.JWKSCachedAt == nil || !got.JWKSCachedAt.Equal(want) {
			t.Fatalf("%s: jwks_cached_at = %v, want %s", step.name, got.JWKSCachedAt, want)
		}
		if !reflect.DeepEqual(got.JWKSCached, keys) {
			t.Fatalf("%s: stored keys = %v, want %v", step.name, got.JWKSCached, keys)
		}
	}
}

// failingRecordRepo は鍵の保存だけを失敗させる。
type failingRecordRepo struct {
	workloadports.WorkloadTrustBundleRepository
}

func (failingRecordRepo) RecordJWKSFetched(context.Context, string, string, *string, workloaddomain.JWKSSnapshot) error {
	return errors.New("database unavailable")
}

//spec:covers REQ-WORKLOADIDENTITY-018: 鍵の集合の保存に失敗しても、検証の結果を変えない。
func TestVerifyWorkloadAttestation_IgnoresKeyStoreFailure(t *testing.T) {
	f, _ := lastGoodFixture(t)
	f.deps.TrustBundleRepo = failingRecordRepo{f.deps.TrustBundleRepo}
	grant, err := f.exchangeAt(t, f.now)
	if err != nil || grant == nil || grant.AgentID != "agent_1" {
		t.Fatalf("exchange = (%+v, %v), want success despite the store failure", grant, err)
	}
}

// jwks_uri を変えた後は、古い取得元から保存した鍵を使わない。消さない実装は、新しい取得元に
// 届かない間も古い取得元の鍵で交換を通す。
//
//spec:covers REQ-WORKLOADIDENTITY-018: 管理者が jwks_uri を変えたとき、保存した鍵の集合と jwks_cached_at を消す。
func TestVerifyWorkloadAttestation_ForgetsStoredKeysWhenTheJWKSURIChanges(t *testing.T) {
	f, bundle := lastGoodFixture(t)
	if _, err := f.exchangeAt(t, f.now); err != nil {
		t.Fatalf("取得できる間の交換: %v", err)
	}
	admin := workloadusecases.AdminWorkloadIdentityDeps{TrustBundleRepo: f.deps.TrustBundleRepo, BindingRepo: f.deps.BindingRepo}
	uri := "https://issuer.example/replacement-jwks.json"
	updated, err := workloadusecases.UpdateWorkloadTrustBundle(withTenant(testTenant), admin, bundle.ID, workloadusecases.UpdateWorkloadTrustBundleInput{JWKSURI: &uri}, f.now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if updated.JWKSCachedAt != nil || updated.JWKSCached != nil {
		t.Fatalf("updated = %+v, want the stored keys cleared", updated)
	}
	stored, err := f.deps.TrustBundleRepo.FindByID(context.Background(), testTenant, bundle.ID)
	if err != nil || stored == nil || stored.JWKSCachedAt != nil || stored.JWKSCached != nil {
		t.Fatalf("stored = (%+v, %v), want the stored keys cleared", stored, err)
	}

	f.restartWithEndpointDown()
	grant, err := f.exchangeAt(t, f.now.Add(2*time.Minute))
	assertRefusedWithoutGrant(t, f, grant, err, "jwks_unavailable")
}
