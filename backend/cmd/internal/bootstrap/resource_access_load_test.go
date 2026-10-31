package bootstrap

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/authentication"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/logging"
	metrics "github.com/ambi/idmagic/backend/shared/observability/metrics_prometheus"
	"github.com/ambi/idmagic/backend/shared/spec"
	pgtest "github.com/ambi/idmagic/backend/shared/storage/testing_postgres"
	pgfixtures "github.com/ambi/idmagic/backend/shared/testing/fixtures_postgres"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"

	"github.com/labstack/echo/v5"
)

// 明示した性能測定だけで走らせる。起動、シード、DPoP の署名時間を応答時間へ混ぜない。
func BenchmarkResourceAccessLoad(t *testing.B) {
	t.Logf("実行環境: %s %s/%s cpu=%d", runtime.Version(), runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
	for _, storage := range []string{"memory", "postgres"} {
		t.Run(storage, func(t *testing.B) {
			f := newResourceLoadFixture(t, storage, map[string]string{"RATE_LIMIT_RESOURCE_ACCESS_MAX_REQUESTS": "1000000", "RATE_LIMIT_RESOURCE_ACCESS_WINDOW_SECONDS": "60"})
			for _, scenario := range []struct {
				name    string
				deep    bool
				dpop    bool
				revoked bool
			}{
				{name: "direct-bearer"},
				{name: "depth4-bearer", deep: true},
				{name: "direct-dpop", dpop: true},
				{name: "depth4-dpop", deep: true, dpop: true},
				{name: "revoked", revoked: true},
			} {
				t.Run(scenario.name, func(t *testing.B) {
					const warmup, samples, workers = 64, 2048, 8
					key, jwk, thumbprint := resourceLoadDPoPKey(t)
					client := &oauthdomain.OAuth2Client{ClientID: f.agentClient}
					input := oauthports.AccessTokenInput{
						Client: client, Sub: f.agentClient, AgentID: f.agentID,
						Scopes: []string{"tools:read"}, Audiences: []string{f.resource},
					}
					if scenario.dpop {
						input.SenderConstraint = &oauthdomain.SenderConstraint{Type: spec.SenderConstraintDPoP, JKT: thumbprint}
					}
					token, _, err := f.signer.SignAccessToken(f.ctx, input)
					if err != nil {
						t.Fatal(err)
					}
					if scenario.revoked {
						now := time.Now().UTC().Add(time.Second)
						if err := f.deps.SharedSignals.RevocationEpochRepo.Advance(f.ctx, ssdomain.AgentRevocationEpoch{
							AgentID: f.agentID, TenantID: f.tenantID, Epoch: now, AdvancedAt: now, Reason: ssdomain.RevocationReasonAgentKilled,
						}); err != nil {
							t.Fatal(err)
						}
					}
					forms := make([]string, warmup+samples)
					sum := sha256.Sum256([]byte(token))
					for i := range forms {
						id := fmt.Sprintf("d%d", i%128)
						if scenario.deep {
							id = "nested"
						}
						id = f.evaluator + "-" + id
						form := url.Values{"token": {token}, "resource_type": {"document"}, "resource_id": {id}, "relation": {"viewer"}}
						if scenario.dpop {
							htu := f.resource + id
							form.Set("htm", http.MethodPost)
							form.Set("htu", htu)
							form.Set("dpop_proof", resourceLoadProof(t, key, jwk, htu, base64.RawURLEncoding.EncodeToString(sum[:]), fmt.Sprintf("%s-%d", token[len(token)-16:], i)))
						}
						forms[i] = form.Encode()
					}
					for _, form := range forms[:warmup] {
						f.evaluate(t, form, !scenario.revoked)
					}
					durations := make([]time.Duration, samples)
					responses := make([]*httptest.ResponseRecorder, samples)
					started := time.Now()
					var group sync.WaitGroup
					for worker := range workers {
						group.Go(func() {
							for i := worker; i < samples; i += workers {
								now := time.Now()
								responses[i] = f.response(forms[warmup+i])
								durations[i] = time.Since(now)
							}
						})
					}
					group.Wait()
					elapsed := time.Since(started)
					for _, response := range responses {
						assertResourceLoadResponse(t, response, !scenario.revoked)
					}
					slices.Sort(durations)
					rps, p99 := float64(samples)/elapsed.Seconds(), durations[(samples*99+99)/100-1]
					t.Logf("resource-load samples=%d workers=%d elapsed=%s rps=%.2f p50=%s p95=%s p99=%s", samples, workers, elapsed,
						rps, durations[(samples+1)/2-1], durations[(samples*95+99)/100-1], p99)
					if rps < 500 || p99 > 100*time.Millisecond {
						t.Errorf("局所測定の目標未達: rps=%.2f p99=%s", rps, p99)
					}
				})
			}
		})
	}
}

type resourceLoadFixture struct {
	deps                                                        *Dependencies
	e                                                           *echo.Echo
	ctx                                                         context.Context
	signer                                                      oauthports.TokenIssuer
	tenantID, agentID, agentClient, evaluator, secret, resource string
}

//spec:covers REQ-OAUTH2-062: 設定から正式な両アダプターを経て HTTP の評価が上限で止まる
//verify:operation EvaluateResourceAccess: 許可の本文と、設定した固定窓の 429 の本文と再試行指示
func TestResourceAccessConfiguredBudgetReachesHTTP(t *testing.T) {
	for _, storage := range []string{"memory", "postgres"} {
		t.Run(storage, func(t *testing.T) {
			f := newResourceLoadFixture(t, storage, map[string]string{"RATE_LIMIT_RESOURCE_ACCESS_MAX_REQUESTS": "2", "RATE_LIMIT_RESOURCE_ACCESS_WINDOW_SECONDS": "60", "TRUSTED_FORWARDED_HOPS": "1"})
			token, _, err := f.signer.SignAccessToken(f.ctx, oauthports.AccessTokenInput{
				Client: &oauthdomain.OAuth2Client{ClientID: f.agentClient}, Sub: f.agentClient, AgentID: f.agentID,
				Scopes: []string{"tools:read"}, Audiences: []string{f.resource},
			})
			if err != nil {
				t.Fatal(err)
			}
			form := url.Values{"token": {token}, "resource_type": {"document"}, "resource_id": {f.evaluator + "-d0"}, "relation": {"viewer"}}.Encode()
			f.evaluate(t, form, true)
			f.evaluate(t, form, true)
			w := f.response(form, func(r *http.Request) { r.Header.Set("X-Forwarded-For", "203.0.113.10, 10.0.0.1") })
			var body map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if w.Code != http.StatusTooManyRequests || body["error"] != "rate_limited" || body["retry_after_seconds"] != float64(60) || w.Header().Get("Retry-After") != "60" || w.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("設定した予算: status=%d headers=%v body=%v", w.Code, w.Header(), body)
			}
		})
	}
}

// 二つの組み立ては別々の接続プールとリミッターを持ち、同じ PostgreSQL だけを共有するので、
// API プロセスを増やした構成と同じ条件で計数の原子性を観測できる。
//
//spec:covers REQ-OAUTH2-062: 複数のプロセスの並行した評価でも許可の合計が上限と等しいこと
func TestResourceAccessLimitHoldsAcrossProcesses(t *testing.T) {
	const limit, attemptsPerProcess = 9, 24
	settings := map[string]string{"RATE_LIMIT_RESOURCE_ACCESS_MAX_REQUESTS": "100000", "RATE_LIMIT_RESOURCE_ACCESS_WINDOW_SECONDS": "60"}
	first := newResourceLoadFixture(t, "postgres", settings)
	second := *first
	second.e = first.newProcess(t, settings)
	server, err := first.deps.OAuth2.McpResourceServerRepo.FindByClientID(first.ctx, first.tenantID, first.evaluator)
	if err != nil || server == nil {
		t.Fatalf("登録: %v %v", server, err)
	}
	server.AccessRules[0].Limits = []oauthdomain.ResourceAccessLimit{{
		By: []oauthdomain.ResourceAccessLimitUnit{oauthdomain.ResourceAccessLimitByAgent}, MaxRequests: limit, WindowSeconds: 3600,
	}}
	if err := first.deps.OAuth2.McpResourceServerRepo.Save(first.ctx, server); err != nil {
		t.Fatal(err)
	}
	token, _, err := first.signer.SignAccessToken(first.ctx, oauthports.AccessTokenInput{
		Client: &oauthdomain.OAuth2Client{ClientID: first.agentClient}, Sub: first.agentClient, AgentID: first.agentID,
		Scopes: []string{"tools:read"}, Audiences: []string{first.resource},
	})
	if err != nil {
		t.Fatal(err)
	}
	form := url.Values{"token": {token}, "resource_type": {"document"}, "resource_id": {first.evaluator + "-d0"}, "relation": {"viewer"}}.Encode()
	var permitted, exceeded atomic.Int64
	var wg sync.WaitGroup
	for _, process := range []*resourceLoadFixture{first, &second} {
		for range attemptsPerProcess {
			wg.Go(func() {
				w := process.response(form)
				var body struct {
					Decision bool     `json:"decision"`
					Reasons  []string `json:"reasons"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || w.Code != http.StatusOK {
					t.Errorf("評価: status=%d body=%s err=%v", w.Code, w.Body.String(), err)
					return
				}
				switch {
				case body.Decision:
					permitted.Add(1)
				case len(body.Reasons) == 1 && body.Reasons[0] == oauthdomain.ReasonLimitExceeded:
					exceeded.Add(1)
				default:
					t.Errorf("想定しない不許可: %s", w.Body.String())
				}
			})
		}
	}
	wg.Wait()
	if permitted.Load() != limit || exceeded.Load() != 2*attemptsPerProcess-limit {
		t.Fatalf("許可 %d、超過 %d、上限 %d", permitted.Load(), exceeded.Load(), limit)
	}
}

// newProcess は同じ設定で別の組み立てを作り、そこへ評価の入口を登録する。DB だけを共有する。
func (f *resourceLoadFixture) newProcess(tb testing.TB, settings map[string]string) *echo.Echo {
	tb.Helper()
	env := map[string]string{
		"PERSISTENCE": "postgres", "ISSUER": "https://load.example", "KEY_PROVIDER": "local",
		"DATABASE_URL": pgtest.Require(tb).Config().ConnString(),
	}
	maps.Copy(env, settings)
	loader := NewConfigLoader(func(key string) string { return env[key] })
	shared, api := LoadSharedConfig(loader), LoadAPIConfig(loader)
	if err := loader.Err(); err != nil {
		tb.Fatal(err)
	}
	deps, err := Assemble(context.Background(), shared)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(deps.Close)
	e := echo.New()
	registerResourceLoadRoutes(tb, e, deps, shared, api)
	return e
}

// 公開モジュールの正式なアダプターを Assemble し、モデルとタプルは管理 HTTP 契約から入れる。
func newResourceLoadFixture(tb testing.TB, storage string, settings map[string]string) *resourceLoadFixture {
	tb.Helper()
	env := map[string]string{"PERSISTENCE": storage, "ISSUER": "https://load.example", "KEY_PROVIDER": "local"}
	maps.Copy(env, settings)
	if storage == "postgres" {
		env["DATABASE_URL"] = pgtest.Require(tb).Config().ConnString()
	}
	loader := NewConfigLoader(func(key string) string { return env[key] })
	shared, api := LoadSharedConfig(loader), LoadAPIConfig(loader)
	if err := loader.Err(); err != nil {
		tb.Fatal(err)
	}
	deps, err := Assemble(context.Background(), shared)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(deps.Close)
	if err := deps.Tenancy.EnsureDefault(context.Background(), time.Now().UTC()); err != nil {
		tb.Fatal(err)
	}
	tenant, err := deps.Tenancy.TenantRepo.FindByRealm(context.Background(), "default")
	if err != nil || tenant == nil {
		tb.Fatalf("テナント: %v", err)
	}
	ctx := tenantports.WithTenant(context.Background(), tenant, shared.Issuer+"/realms/default", "/realms/default")
	f := &resourceLoadFixture{
		deps: deps, e: echo.New(), ctx: ctx, tenantID: tenant.ID,
		agentID: pgfixtures.NewUUID(tb), agentClient: pgfixtures.NewUUID(tb), evaluator: pgfixtures.NewUUID(tb), secret: "load-secret",
	}
	f.resource = "https://resource.example/" + f.evaluator + "/"
	now := time.Now().UTC()
	owner := &userdomain.User{
		ID: pgfixtures.NewUUID(tb), TenantID: tenant.ID, PreferredUsername: pgfixtures.UniqueID("load-owner"),
		Roles: []string{"admin"}, PasswordHash: "unused", Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}, CreatedAt: now, UpdatedAt: now,
	}
	if err := deps.IdManagement.UserRepo.Save(ctx, owner); err != nil {
		tb.Fatal(err)
	}
	if err := deps.IdManagement.AgentRepo.Save(ctx, &agentdomain.Agent{
		ID: f.agentID, TenantID: tenant.ID, Name: f.agentClient,
		OwnerUserID: owner.ID, Kind: idmdomain.AgentKindAutonomous, Status: idmdomain.AgentStatusActive, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		tb.Fatal(err)
	}
	hash := oauthdomain.HashClientSecret(f.secret)
	for _, client := range []string{f.agentClient, f.evaluator} {
		if err := deps.OAuth2.ClientRepo.Save(ctx, &oauthdomain.OAuth2Client{
			TenantID: tenant.ID, ClientID: client,
			ClientType: spec.ClientConfidential, ClientSecretHash: &hash, TokenEndpointAuthMethod: oauthdomain.AuthMethodClientSecretBasic,
			FapiProfile: oauthdomain.FapiNone, IDTokenSignedResponseAlg: "PS256",
			GrantTypes: []spec.GrantType{spec.GrantClientCredentials}, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			tb.Fatal(err)
		}
	}
	saved, err := deps.OAuth2.ClientRepo.FindByID(ctx, tenant.ID, f.evaluator)
	if err != nil || saved == nil || saved.ClientSecretHash == nil || !oauthdomain.VerifyClientSecret(f.secret, *saved.ClientSecretHash) || saved.TokenEndpointAuthMethod != oauthdomain.AuthMethodClientSecretBasic {
		tb.Fatalf("評価クライアントの認証情報の往復: client=%v err=%v", saved != nil, err)
	}
	if _, err := deps.OAuth2.ClientRepo.ListClientSecretCredentials(ctx, f.evaluator); err != nil {
		tb.Fatalf("評価クライアントの認証情報の一覧: %v", err)
	}
	if err := deps.OAuth2.McpResourceServerRepo.Save(ctx, &oauthdomain.McpResourceServer{
		ID: pgfixtures.NewUUID(tb), TenantID: tenant.ID,
		Resource: f.resource, Name: f.evaluator, ClientID: f.evaluator, State: oauthdomain.McpResourceServerActive,
		Scopes: []string{"tools:read"}, AccessRules: []oauthdomain.ResourceAccessRule{{ResourceType: "document", Relation: "viewer", Scopes: []string{"tools:read"}}},
		CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		tb.Fatal(err)
	}
	f.signer = registerResourceLoadRoutes(tb, f.e, deps, shared, api)
	authn, err := deps.Authentication.SessionManager.Create(ctx, owner.ID, []string{"pwd"}, now)
	if err != nil {
		tb.Fatal(err)
	}
	admin := func(path string, body any) {
		tb.Helper()
		raw, err := json.Marshal(body)
		if err != nil {
			tb.Fatal(err)
		}
		request := httptest.NewRequest(http.MethodPost, shared.Issuer+"/realms/default"+path, strings.NewReader(string(raw)))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", shared.Issuer)
		request.Header.Set(support.CSRFHeader, "load-csrf")
		request.AddCookie(&http.Cookie{Name: support.CSRFCookie, Value: "load-csrf"})
		request.AddCookie(&http.Cookie{Name: authentication.SessionCookie, Value: authn.SessionID})
		recorder := httptest.NewRecorder()
		f.e.ServeHTTP(recorder, request)
		if recorder.Code/100 != 2 {
			tb.Fatalf("前提の投入 %s: %d %s", path, recorder.Code, recorder.Body.String())
		}
	}
	admin("/api/admin/v1/authorization/model", map[string]any{"resource_types": []any{
		map[string]any{"name": "user"},
		map[string]any{"name": "agent"},
		map[string]any{"name": "document", "relations": []any{map[string]any{"name": "viewer", "rewrites": []any{map[string]any{
			"kind": "direct", "direct_subject_types": []string{"agent", "document#viewer"},
		}}}}},
	}})
	writes := make([]map[string]any, 0, 132)
	for i := range 128 {
		writes = append(writes, map[string]any{"resource_type": "document", "resource_id": fmt.Sprintf("%s-d%d", f.evaluator, i), "relation": "viewer", "subject_type": "agent", "subject_id": f.agentID})
	}
	for i, id := range []string{"nested", "parent1", "parent2"} {
		next := []string{"parent1", "parent2", "d0"}[i]
		writes = append(writes, map[string]any{"resource_type": "document", "resource_id": f.evaluator + "-" + id, "relation": "viewer", "subject_type": "document", "subject_id": f.evaluator + "-" + next, "subject_relation": "viewer"})
	}
	admin("/api/admin/v1/authorization/relation-tuples", map[string]any{"writes": writes})
	return f
}

// registerResourceLoadRoutes は組み立てを本番と同じ HTTP の入口へ登録し、トークンの署名器を返す。
func registerResourceLoadRoutes(tb testing.TB, e *echo.Echo, deps *Dependencies, shared SharedConfig, api APIConfig) oauthports.TokenIssuer {
	tb.Helper()
	signer := oauth2.NewJWTSigner(shared.Issuer, deps.SigningKeys.KeyStore)
	deps.OAuth2.TokenIssuer, deps.OAuth2.TokenIntrospector = signer, oauth2.NewJWTSigner(shared.Issuer, deps.SigningKeys.KeyStore)
	deps.OAuth2.Authorizer = AssembleAuthorizer(shared)
	deps.Authentication.SessionManager = deps.Authentication.NewSessionManager()
	// ログ出力先の速度を測らず、イベントの直列化と監査 DB への追記は本番と同じにする。
	deps.OAuth2.EventSink = resourceLoadSink{}
	emit := deps.NewEmitFunc(logging.Default())
	observations, err := metrics.NewMetrics("resource-load", "")
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		if err := observations.Shutdown(context.Background()); err != nil {
			tb.Error(err)
		}
	})
	e.Use(support.MetricsMiddleware(observations))
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: shared.Issuer, TenantRepo: deps.Tenancy.TenantRepo,
		Contract: spec.CurrentRuntimeContract(), RateLimiter: deps.RateLimit.NewRateLimiter(api.RateLimits), Emit: emit, Metrics: observations,
		TrustedForwardedHops: api.TrustedForwardedHops,
		Tenancy:              deps.Tenancy, IdManagement: deps.IdManagement, Authentication: deps.Authentication, OAuth2: deps.OAuth2,
		ApiTokens: deps.ApiTokens, SigningKeys: deps.SigningKeys, AccessControl: deps.AccessControl, SharedSignals: deps.SharedSignals,
	})
	return signer
}

func (f *resourceLoadFixture) evaluate(tb testing.TB, form string, permitted bool) {
	tb.Helper()
	assertResourceLoadResponse(tb, f.response(form), permitted)
}

func assertResourceLoadResponse(tb testing.TB, w *httptest.ResponseRecorder, permitted bool) {
	tb.Helper()
	var body struct {
		Decision bool     `json:"decision"`
		Reasons  []string `json:"reasons"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || w.Code != http.StatusOK || body.Decision != permitted {
		tb.Fatalf("評価: status=%d body=%s error=%v want=%v", w.Code, w.Body.String(), err, permitted)
	}
	if !permitted && (len(body.Reasons) != 1 || body.Reasons[0] != "token_rejected") {
		tb.Fatalf("失効の拒否: %s", w.Body.String())
	}
}

func (f *resourceLoadFixture) response(form string, mutate ...func(*http.Request)) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "https://load.example/realms/default/resource-access/evaluate", strings.NewReader(form))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetBasicAuth(f.evaluator, f.secret)
	for _, change := range mutate {
		change(r)
	}
	w := httptest.NewRecorder()
	f.e.ServeHTTP(w, r)
	return w
}

type resourceLoadSink struct{}

func (resourceLoadSink) Emit(_ context.Context, event spec.DomainEvent) error {
	raw, err := spec.MarshalDomainEvent(event)
	if err != nil {
		return err
	}
	_, err = io.Discard.Write(raw)
	return err
}

func resourceLoadDPoPKey(tb testing.TB) (*rsa.PrivateKey, map[string]any, string) {
	tb.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		tb.Fatal(err)
	}
	jwk := map[string]any{"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()), "kty": "RSA", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes())}
	raw, err := json.Marshal(jwk)
	if err != nil {
		tb.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	return key, jwk, base64.RawURLEncoding.EncodeToString(sum[:])
}

func resourceLoadProof(tb testing.TB, key *rsa.PrivateKey, jwk map[string]any, htu, ath, id string) string {
	tb.Helper()
	header, err := json.Marshal(map[string]any{"typ": "dpop+jwt", "alg": "PS256", "jwk": jwk})
	if err != nil {
		tb.Fatal(err)
	}
	claims, err := json.Marshal(map[string]any{"htm": "POST", "htu": htu, "ath": ath, "iat": time.Now().Unix(), "jti": id})
	if err != nil {
		tb.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPSS(rand.Reader, key, crypto.SHA256, digest[:], &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
	if err != nil {
		tb.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}
