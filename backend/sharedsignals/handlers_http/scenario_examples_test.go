package handlers_http_test

// SharedSignals が宣言する具体例を、製品と同じ `server_http.Register` の組み立てで確かめる。
//
// 入口は製品のものだけを使う。Agent の強制終了は IdManagement の管理 API、トークンは
// `/token` と `/introspect`、SET は受信エンドポイントへ実際に署名したものを送る。
// 受信側の検証器は製品と同じ `verify_jose` であり、外部の送信者の公開鍵をインライン
// JWKS として受信ストリームへ登録する。応答だけでは拒否の後に効果を残す実装を
// 見分けられないので、失効エポック、ストリーム、配送、クォータの利用量、イベントを
// 保存先から読み直す。

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	"github.com/ambi/idmagic/backend/idmanagement"
	agentmemory "github.com/ambi/idmagic/backend/idmanagement/agent/db_memory"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/oauth2"
	oauth2memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	oauthdomain "github.com/ambi/idmagic/backend/oauth2/domain"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	tokensjose "github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/sharedsignals"
	ssmemory "github.com/ambi/idmagic/backend/sharedsignals/db_memory"
	ssdomain "github.com/ambi/idmagic/backend/sharedsignals/domain"
	ssports "github.com/ambi/idmagic/backend/sharedsignals/ports"
	"github.com/ambi/idmagic/backend/signingkeys"
	signingjose "github.com/ambi/idmagic/backend/signingkeys/keys_jose"
	signingmemory "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const (
	exampleIssuer      = "http://idp.test"
	exampleAdmin       = "admin"
	exampleNonAdmin    = "alice"
	exampleOtherTenant = "acme"

	// 外部の送信者と、受信ストリームが受け付ける audience。
	trustedIssuer    = "https://issuer.example"
	receiverAudience = "https://idp.test/ssf"

	agentClientID     = "agent-client"
	agentClientSecret = "agent-client-secret"
	// resourceServerID は `/introspect` のクライアント認証に使う。トークンの状態を
	// 製品の入口から読むために要る。
	resourceServerID     = "resource-server"
	resourceServerSecret = "resource-server-secret"

	sessionRevokedURI = "https://schemas.openid.net/secevent/caep/event-type/session-revoked"
	// 送信者が名乗る鍵の識別子。偽造の事例は同じ識別子を別の鍵に名乗らせる。
	transmitterKID = "transmitter-key"

	// 具体例が名指しする Agent。A1 は既定テナント、A2 はもう一方のテナントに置く。
	exampleAgentID     = "A1"
	otherTenantAgentID = "A2"
)

// eventLog は `Deps.Emit` が受けたイベントを発行順に覚える。
type eventLog struct {
	mu     sync.Mutex
	events []spec.DomainEvent
}

func (l *eventLog) record(event spec.DomainEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.events = append(l.events, event)
}

func (l *eventLog) all() []spec.DomainEvent {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.events)
}

func (l *eventLog) count(eventType string) int {
	n := 0
	for _, event := range l.all() {
		if event.EventType() == eventType {
			n++
		}
	}
	return n
}

// rejections は発行された SecurityEventRejected の検証結果を発行順に返す。
func (l *eventLog) rejections() []ssdomain.SecurityEventVerificationResult {
	var results []ssdomain.SecurityEventVerificationResult
	for _, event := range l.all() {
		if rejected, ok := event.(*ssdomain.SecurityEventRejected); ok {
			results = append(results, rejected.VerificationResult)
		}
	}
	return results
}

// countingTransmitterConfigs と countingReceiverConfigs は保存の回数を数える。
// 付随する設定は stream の id でしか引けないので、拒否した登録が設定だけを残して
// いないことは、保存が呼ばれた回数でしか読めない。
type countingTransmitterConfigs struct {
	ssports.SsfTransmitterConfigRepository
	mu    sync.Mutex
	saves int
}

func (r *countingTransmitterConfigs) Save(ctx context.Context, tenantID string, c *ssdomain.SsfTransmitterConfig) error {
	r.mu.Lock()
	r.saves++
	r.mu.Unlock()
	return r.SsfTransmitterConfigRepository.Save(ctx, tenantID, c)
}

type countingReceiverConfigs struct {
	ssports.SsfReceiverConfigRepository
	mu    sync.Mutex
	saves int
}

func (r *countingReceiverConfigs) Save(ctx context.Context, tenantID string, c *ssdomain.SsfReceiverConfig) error {
	r.mu.Lock()
	r.saves++
	r.mu.Unlock()
	return r.SsfReceiverConfigRepository.Save(ctx, tenantID, c)
}

type exampleStack struct {
	e                  *echo.Echo
	events             *eventLog
	agents             *agentmemory.AgentRepository
	streams            *ssmemory.SsfStreamRepository
	transmitterConfigs *countingTransmitterConfigs
	receiverConfigs    *countingReceiverConfigs
	deliveries         *ssmemory.SecurityEventDeliveryRepository
	epochs             *ssmemory.AgentRevocationEpochRepository
	quotas             *tenancymemory.QuotaRepository
	keyStore           *signingmemory.InMemoryKeyStore
	csrf               string
	cookie             *http.Cookie
}

func newExampleStack(t *testing.T) *exampleStack {
	t.Helper()
	now := time.Now().UTC()

	tenants := tenancymemory.NewTenantRepository()
	for _, tenant := range []*tenancydomain.Tenant{
		{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, DisplayName: "Default", Status: tenancydomain.TenantStatusActive, CreatedAt: now},
		{ID: exampleOtherTenant, Realm: exampleOtherTenant, DisplayName: "Acme", Status: tenancydomain.TenantStatusActive, CreatedAt: now},
	} {
		if err := tenants.Save(context.Background(), tenant); err != nil {
			t.Fatalf("seed tenant: %v", err)
		}
	}
	users := usermemory.NewUserRepository()
	for _, user := range []*userdomain.User{
		{ID: exampleAdmin, PreferredUsername: exampleAdmin, Roles: []string{"admin"}},
		{ID: exampleNonAdmin, PreferredUsername: exampleNonAdmin},
	} {
		user.TenantID = tenancydomain.DefaultTenantID
		user.PasswordHash = "unused"
		user.Lifecycle = userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}
		user.CreatedAt, user.UpdatedAt = now, now
		users.Seed(user)
	}

	clients := oauth2memory.NewClientRepository()
	for id, secret := range map[string]string{agentClientID: agentClientSecret, resourceServerID: resourceServerSecret} {
		hash := oauthdomain.HashClientSecret(secret)
		clients.Seed(&oauthdomain.OAuth2Client{
			TenantID: tenancydomain.DefaultTenantID, ClientID: id, ClientSecretHash: &hash,
			ClientType:              spec.ClientConfidential,
			GrantTypes:              []spec.GrantType{spec.GrantClientCredentials},
			TokenEndpointAuthMethod: oauthdomain.AuthMethodClientSecretBasic,
			Scope:                   "openid",
			FapiProfile:             oauthdomain.FapiNone,
			CreatedAt:               now,
		})
	}

	keyStore, err := signingmemory.NewInMemoryKeyStore()
	if err != nil {
		t.Fatalf("key store: %v", err)
	}
	signer := tokensjose.NewJWTSigner(exampleIssuer, keyStore)
	quotas := tenancymemory.NewQuotaRepository()
	limit := 20
	if err := quotas.SetQuota(context.Background(), tenancydomain.DefaultTenantID, &tenancydomain.TenantQuota{SsfStreams: &limit}); err != nil {
		t.Fatalf("seed quota: %v", err)
	}

	s := &exampleStack{
		e: echo.New(), events: &eventLog{}, agents: agentmemory.NewAgentRepository(),
		streams:            ssmemory.NewSsfStreamRepository(),
		transmitterConfigs: &countingTransmitterConfigs{SsfTransmitterConfigRepository: ssmemory.NewSsfTransmitterConfigRepository()},
		receiverConfigs:    &countingReceiverConfigs{SsfReceiverConfigRepository: ssmemory.NewSsfReceiverConfigRepository()},
		deliveries:         ssmemory.NewSecurityEventDeliveryRepository(),
		epochs:             ssmemory.NewAgentRevocationEpochRepository(),
		quotas:             quotas, keyStore: keyStore,
	}
	httpadapter.Register(s.e, httpadapter.Deps{
		Issuer: exampleIssuer, Emit: s.events.record, TenantRepo: tenants,
		AuthnResolver: authusecases.DemoHeaderResolver{},
		IdManagement:  idmanagement.Module{UserRepo: users, GroupRepo: groupmemory.NewGroupRepository(), AgentRepo: s.agents},
		SigningKeys:   signingkeys.Module{KeyStore: keyStore},
		OAuth2: oauth2.Module{
			ClientRepo: clients, TokenIssuer: signer, TokenIntrospector: signer,
			RefreshStore: oauth2memory.NewRefreshTokenStore(), AccessTokenDenylist: oauth2memory.NewAccessTokenDenylist(),
		},
		Tenancy:     tenancy.Module{QuotaRepo: quotas},
		JWKResolver: tokensjose.NewJWKResolver(),
		SharedSignals: sharedsignals.Module{
			RevocationEpochRepo: s.epochs, StreamRepo: s.streams,
			TransmitterConfigRepo: s.transmitterConfigs, ReceiverConfigRepo: s.receiverConfigs,
			DeliveryRepo: s.deliveries, ReceivedEventRepo: ssmemory.NewReceivedSecurityEventRepository(),
		},
	})
	s.csrf, s.cookie = sharedSignalsAdminCSRF(t, s.e)
	return s
}

// request は sub として管理 API を呼ぶ。CSRF の二重送信と Origin は成立させておき、
// 拒否があれば管理者であるかどうかによるものだけになるようにする。
func (s *exampleStack) request(t *testing.T, sub, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(payload)
	}
	req := httptest.NewRequest(method, "/realms/default"+path, reader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", exampleIssuer)
	req.Header.Set("X-Csrf-Token", s.csrf)
	req.Header.Set("X-Demo-Sub", sub)
	req.AddCookie(s.cookie)
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	return rec
}

func (s *exampleStack) admin(t *testing.T, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	return s.request(t, exampleAdmin, method, path, body)
}

func decodeID(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.ID == "" {
		t.Fatalf("response carries no id: %s (%v)", rec.Body.String(), err)
	}
	return body.ID
}

func (s *exampleStack) registerTransmitter(t *testing.T, endpoint string) string {
	t.Helper()
	rec := s.admin(t, http.MethodPost, "/api/admin/v1/shared-signals/streams/transmitter", map[string]any{
		"delivery_endpoint": endpoint, "audience": "https://receiver.example",
		"event_types": []string{string(ssdomain.CaepEventTypeSessionRevoked)},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("register transmitter status=%d body=%s", rec.Code, rec.Body.String())
	}
	return decodeID(t, rec)
}

func (s *exampleStack) registerReceiver(t *testing.T, key *transmitterKey) string {
	t.Helper()
	rec := s.admin(t, http.MethodPost, "/api/admin/v1/shared-signals/streams/receiver", receiverRequest(key))
	if rec.Code != http.StatusCreated {
		t.Fatalf("register receiver status=%d body=%s", rec.Code, rec.Body.String())
	}
	return decodeID(t, rec)
}

func receiverRequest(key *transmitterKey) map[string]any {
	return map[string]any{
		"trusted_issuer": trustedIssuer, "jwks": key.jwks(),
		"accepted_audiences": []string{receiverAudience}, "event_types": []string{string(ssdomain.CaepEventTypeSessionRevoked)},
	}
}

func (s *exampleStack) listStreams(t *testing.T) []*ssdomain.SsfStream {
	t.Helper()
	streams, err := s.streams.ListAll(context.Background(), tenancydomain.DefaultTenantID)
	if err != nil {
		t.Fatal(err)
	}
	return streams
}

func (s *exampleStack) findStream(t *testing.T, id string) *ssdomain.SsfStream {
	t.Helper()
	stream, err := s.streams.FindByID(context.Background(), tenancydomain.DefaultTenantID, id)
	if err != nil {
		t.Fatal(err)
	}
	return stream
}

// postSET は受信エンドポイントへ SET を 1 通送る。受信エンドポイントは公開の入口で
// あり、管理者の認証も CSRF も要らない。
func (s *exampleStack) postSET(t *testing.T, streamID, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/realms/default/ssf/streams/"+streamID+"/events", strings.NewReader(token))
	req.Header.Set("Content-Type", "application/secevent+jwt")
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	return rec
}

func assertSETRejected(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s, want 400", rec.Code, rec.Body.String())
	}
	var body struct {
		Err string `json:"err"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Err != "security_event_rejected" {
		t.Fatalf("body=%s, want err=security_event_rejected", rec.Body.String())
	}
}

// seedAgent は Active の Agent を置き、agentClientID のクライアントを束縛する。
// 所有者は Active の管理者であり、client_credentials の発行を止める理由を持たない。
func (s *exampleStack) seedAgent(t *testing.T, tenantID, agentID string) {
	t.Helper()
	now := time.Now().UTC()
	if err := s.agents.Save(context.Background(), &agentdomain.Agent{
		ID: agentID, TenantID: tenantID, Name: agentID, Kind: idmdomain.AgentKindAutonomous,
		OwnerUserID: exampleAdmin, Status: idmdomain.AgentStatusActive, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed agent: %v", err)
	}
	if tenantID != tenancydomain.DefaultTenantID {
		return
	}
	if _, err := s.agents.AddBinding(context.Background(), &agentdomain.AgentCredentialBinding{
		AgentID: agentID, ClientID: agentClientID, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed binding: %v", err)
	}
}

func (s *exampleStack) epoch(t *testing.T, tenantID, agentID string) *ssdomain.AgentRevocationEpoch {
	t.Helper()
	epoch, err := s.epochs.FindByAgent(context.Background(), tenantID, agentID)
	if err != nil {
		t.Fatal(err)
	}
	return epoch
}

// issueAgentToken は Agent に束縛されたクライアントで `/token` から access token を取る。
func (s *exampleStack) issueAgentToken(t *testing.T) string {
	t.Helper()
	form := url.Values{"grant_type": {"client_credentials"}, "scope": {"openid"}}
	req := httptest.NewRequest(http.MethodPost, "/realms/default/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(agentClientID, agentClientSecret)
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	var body struct {
		AccessToken string `json:"access_token"`
	}
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &body) != nil || body.AccessToken == "" {
		t.Fatalf("token status=%d body=%s", rec.Code, rec.Body.String())
	}
	return body.AccessToken
}

func (s *exampleStack) introspect(t *testing.T, token string) map[string]any {
	t.Helper()
	form := url.Values{"token": {token}, "token_type_hint": {"access_token"}}
	req := httptest.NewRequest(http.MethodPost, "/realms/default/introspect", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(resourceServerID, resourceServerSecret)
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	var body map[string]any
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &body) != nil {
		t.Fatalf("introspect status=%d body=%s", rec.Code, rec.Body.String())
	}
	return body
}

// reissueAt は製品が発行した access token と同じ claim を持ち、iat だけを issuedAt に
// 置いたトークンを、製品と同じ署名鍵で作る。発行時刻は秒単位なので、強制終了の後に
// 発行したトークンを時計を待たずに作るにはこうするしかない。
func (s *exampleStack) reissueAt(t *testing.T, token string, issuedAt time.Time) string {
	t.Helper()
	parts := strings.Split(token, ".")
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	var claims map[string]any
	if err := json.Unmarshal(raw, &claims); err != nil {
		t.Fatal(err)
	}
	claims["iat"] = issuedAt.Unix()
	claims["jti"] = "reissued-" + strings.ReplaceAll(issuedAt.Format(time.RFC3339Nano), ":", "")
	ctx := tenancy.WithTenant(context.Background(), &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID}, "", "")
	key, err := s.keyStore.GetActiveKey(ctx)
	if err != nil {
		t.Fatal(err)
	}
	reissued, err := tokensjose.SignPS256(key, map[string]string{"typ": "at+jwt"}, claims)
	if err != nil {
		t.Fatal(err)
	}
	return reissued
}

// transmitterKey は外部の送信者の署名鍵である。
type transmitterKey struct {
	private *rsa.PrivateKey
	kid     string
}

func newTransmitterKey(t *testing.T) *transmitterKey {
	t.Helper()
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return &transmitterKey{private: private, kid: transmitterKID}
}

func (k *transmitterKey) jwks() map[string]any {
	jwk := signingjose.PublicJWK(&k.private.PublicKey)
	jwk["kid"] = k.kid
	jwk["alg"] = "PS256"
	jwk["use"] = "sig"
	return map[string]any{"keys": []any{jwk}}
}

// sign は subject を持つ session-revoked の SET を PS256 で署名する。kid は鍵自身の
// ものを名乗るので、別の鍵に同じ kid を持たせれば「署名が不正な SET」になる。
func (k *transmitterKey) sign(t *testing.T, jti string, subject map[string]any) string {
	t.Helper()
	header, err := json.Marshal(map[string]any{"alg": "PS256", "kid": k.kid, "typ": "secevent+jwt"})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"iss": trustedIssuer, "aud": receiverAudience, "jti": jti, "iat": time.Now().Unix(),
		"events": map[string]any{sessionRevokedURI: map[string]any{"subject": subject}},
	})
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPSS(rand.Reader, k.private, crypto.SHA256, digest[:], &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func issSub(sub string) map[string]any {
	return map[string]any{"format": "iss_sub", "iss": trustedIssuer, "sub": sub}
}

// killAgent は管理者として A1 の KillAgent を呼び、呼ぶ直前と直後の時刻を返す。
func (s *exampleStack) killAgent(t *testing.T) (time.Time, time.Time) {
	t.Helper()
	before := time.Now().UTC()
	rec := s.admin(t, http.MethodPost, "/api/admin/v1/agents/"+exampleAgentID+"/kill", nil)
	after := time.Now().UTC()
	if rec.Code != http.StatusNoContent {
		t.Fatalf("kill status=%d body=%s", rec.Code, rec.Body.String())
	}
	return before, after
}

// assertEpochAdvancedByKill は、強制終了が A1 の失効エポックを要求の時刻まで進め、
// RevocationEpochAdvanced と AgentAccessRevoked をその Agent について発行したことを確かめる。
func (s *exampleStack) assertEpochAdvancedByKill(t *testing.T, before, after time.Time) *ssdomain.AgentRevocationEpoch {
	t.Helper()
	const agentID = exampleAgentID
	epoch := s.epoch(t, tenancydomain.DefaultTenantID, agentID)
	if epoch == nil {
		t.Fatal("KillAgent did not advance the revocation epoch")
	}
	if epoch.Epoch.Before(before) || epoch.Epoch.After(after) || epoch.Reason != ssdomain.RevocationReasonAgentKilled {
		t.Fatalf("epoch=%+v, want reason=AgentKilled within [%s, %s]", epoch, before, after)
	}
	var advanced, revoked bool
	for _, event := range s.events.all() {
		switch e := event.(type) {
		case *ssdomain.RevocationEpochAdvanced:
			advanced = advanced || (e.AgentID == agentID && e.Epoch.Equal(epoch.Epoch))
		case *ssdomain.AgentAccessRevoked:
			revoked = revoked || e.AgentID == agentID
		}
	}
	if !advanced || !revoked {
		t.Fatalf("RevocationEpochAdvanced=%v AgentAccessRevoked=%v for %s; events=%v", advanced, revoked, agentID, s.events.all())
	}
	return epoch
}

// 強制終了の前に `/token` が発行したトークンを、強制終了の前後で `/introspect` へ
// 問い合わせる。前で active=true を観測しておくので、false が強制終了の結果だと読める。
//
//spec:covers EX-SHAREDSIGNALS-001-01: KillAgent が失効エポックを要求の時刻へ進めて RevocationEpochAdvanced と AgentAccessRevoked を発行し、強制終了の前に `/token` が発行したトークンの `/introspect` が active=true から active=false へ変わる。
func TestKillAgentRevokesTokensIssuedBeforeIt(t *testing.T) {
	s := newExampleStack(t)
	s.seedAgent(t, tenancydomain.DefaultTenantID, exampleAgentID)
	at1 := s.issueAgentToken(t)
	if active := s.introspect(t, at1)["active"]; active != true {
		t.Fatalf("before kill active=%v, want true", active)
	}

	before, after := s.killAgent(t)

	s.assertEpochAdvancedByKill(t, before, after)
	if body := s.introspect(t, at1); body["active"] != false || len(body) != 1 {
		t.Fatalf("after kill introspection=%v, want only active=false", body)
	}
}

// 強制終了の後に発行したトークンは、同じ Agent のものでも失効対象にしない。比較の向きを
// 取り違えた実装を見分けるため、強制終了の前に発行したトークンが無効のままであることも
// 同じテストで観測する。
//
//spec:covers EX-SHAREDSIGNALS-001-02: KillAgent が失効エポックを進めてイベントを発行した後、`issued_at` が新しいエポックより後のトークンは `/introspect` で active=true と client_id を返し、エポックより前のトークンは active=false のままである。
func TestTokenIssuedAfterTheKillEpochStaysActive(t *testing.T) {
	s := newExampleStack(t)
	s.seedAgent(t, tenancydomain.DefaultTenantID, exampleAgentID)
	at1 := s.issueAgentToken(t)

	before, after := s.killAgent(t)

	epoch := s.assertEpochAdvancedByKill(t, before, after)
	reissued := s.reissueAt(t, at1, epoch.Epoch.Add(time.Second))
	body := s.introspect(t, reissued)
	if body["active"] != true || body["client_id"] != agentClientID {
		t.Fatalf("token issued after the epoch: introspection=%v, want active=true for %s", body, agentClientID)
	}
	if active := s.introspect(t, at1)["active"]; active != false {
		t.Fatalf("token issued before the epoch: active=%v, want false", active)
	}
}

// 登録した鍵の kid を名乗りながら別の鍵で署名した SET を拒否させる。同じ内容を登録した
// 鍵で署名し直すと受理されることを後に置くので、拒否の理由が署名だけであると読める。
//
//spec:covers EX-SHAREDSIGNALS-003-01: 署名が不正な SET は 400 security_event_rejected で拒否され、SecurityEventRejected が rejected_signature で発行され、Agent の失効エポックは作られず SecurityEventReceived も発行されない。
func TestSetWithAnInvalidSignatureIsRejectedWithoutRevoking(t *testing.T) {
	s := newExampleStack(t)
	s.seedAgent(t, tenancydomain.DefaultTenantID, exampleAgentID)
	trusted := newTransmitterKey(t)
	forger := newTransmitterKey(t)
	streamID := s.registerReceiver(t, trusted)

	assertSETRejected(t, s.postSET(t, streamID, forger.sign(t, "J-forged", issSub(exampleAgentID))))

	if got := s.events.rejections(); !slices.Equal(got, []ssdomain.SecurityEventVerificationResult{ssdomain.SecurityEventVerificationRejectedSignature}) {
		t.Fatalf("SecurityEventRejected results=%v, want [rejected_signature]", got)
	}
	if epoch := s.epoch(t, tenancydomain.DefaultTenantID, exampleAgentID); epoch != nil {
		t.Fatalf("a forged SET advanced the revocation epoch: %+v", epoch)
	}
	if n := s.events.count("SecurityEventReceived"); n != 0 {
		t.Fatalf("SecurityEventReceived emitted %d times for a forged SET", n)
	}

	if rec := s.postSET(t, streamID, trusted.sign(t, "J-genuine", issSub(exampleAgentID))); rec.Code != http.StatusAccepted {
		t.Fatalf("the same SET signed by the registered key: status=%d body=%s, want 202", rec.Code, rec.Body.String())
	}
	if s.epoch(t, tenancydomain.DefaultTenantID, exampleAgentID) == nil {
		t.Fatal("the genuinely signed SET did not advance the revocation epoch")
	}
}

// 受理済みの SET と同じトークンをもう一度送る。署名も内容も正しいので、拒否の理由は
// jti の再送だけである。
//
//spec:covers EX-SHAREDSIGNALS-004-01: 受理済みの jti を持つ SET の再送は 400 security_event_rejected で拒否され、SecurityEventRejected が rejected_replay で発行され、失効エポックも SecurityEventReceived も 1 回目のまま増えない。
func TestReplayedSetIsRejected(t *testing.T) {
	s := newExampleStack(t)
	s.seedAgent(t, tenancydomain.DefaultTenantID, exampleAgentID)
	key := newTransmitterKey(t)
	streamID := s.registerReceiver(t, key)
	token := key.sign(t, "J1", issSub(exampleAgentID))
	if rec := s.postSET(t, streamID, token); rec.Code != http.StatusAccepted {
		t.Fatalf("first delivery status=%d body=%s, want 202", rec.Code, rec.Body.String())
	}
	first := s.epoch(t, tenancydomain.DefaultTenantID, exampleAgentID)

	assertSETRejected(t, s.postSET(t, streamID, token))

	if got := s.events.rejections(); !slices.Equal(got, []ssdomain.SecurityEventVerificationResult{ssdomain.SecurityEventVerificationRejectedReplay}) {
		t.Fatalf("SecurityEventRejected results=%v, want [rejected_replay]", got)
	}
	if n := s.events.count("SecurityEventReceived"); n != 1 {
		t.Fatalf("SecurityEventReceived emitted %d times, want 1", n)
	}
	if again := s.epoch(t, tenancydomain.DefaultTenantID, exampleAgentID); again == nil || !again.AdvancedAt.Equal(first.AdvancedAt) {
		t.Fatalf("the replay moved the revocation epoch: first=%+v again=%+v", first, again)
	}
}

// 他テナントの Agent を、テナントを名乗る idmagic 独自の形式と、テナントを持たない
// RFC 9493 の形式の両方で名指す。どちらも受信ストリームのテナントでは解決できない。
//
//spec:covers EX-SHAREDSIGNALS-005-01: テナント "default" の受信ストリームへ送られた、テナント "acme" の Agent を subject とする SET は、形式によらず 400 security_event_rejected で拒否され、SecurityEventRejected が rejected_subject_unresolved で発行され、"acme" の Agent の失効エポックは作られない。
func TestSetNamingAnotherTenantsAgentIsRejected(t *testing.T) {
	subjects := map[string]map[string]any{
		"idmagic format naming the other tenant":  {"subject_type": "Agent", "tenant_id": exampleOtherTenant, "principal_id": otherTenantAgentID},
		"iss_sub naming the other tenant's agent": issSub(otherTenantAgentID),
	}
	for name, subject := range subjects {
		t.Run(name, func(t *testing.T) {
			s := newExampleStack(t)
			s.seedAgent(t, exampleOtherTenant, otherTenantAgentID)
			key := newTransmitterKey(t)
			streamID := s.registerReceiver(t, key)

			assertSETRejected(t, s.postSET(t, streamID, key.sign(t, "J-cross", subject)))

			if got := s.events.rejections(); !slices.Equal(got, []ssdomain.SecurityEventVerificationResult{ssdomain.SecurityEventVerificationRejectedSubjectUnresolved}) {
				t.Fatalf("SecurityEventRejected results=%v, want [rejected_subject_unresolved]", got)
			}
			if epoch := s.epoch(t, exampleOtherTenant, otherTenantAgentID); epoch != nil {
				t.Fatalf("another tenant's agent was revoked: %+v", epoch)
			}
		})
	}
}

// 受信側に到達できない送信ストリームがある状態で強制終了する。強制終了の応答が返った
// 時点で、配送はまだ 1 度も試みられていない。
//
//spec:covers EX-SHAREDSIGNALS-007-01: 受信側へ到達できない送信ストリームがあっても、KillAgent の応答の時点で失効エポックは進んでトークンの `/introspect` は active=false を返し、そのストリームの SecurityEventDelivery は試行 0 回の pending として配送待ちの一覧に載る。
func TestKillAdvancesTheEpochWhileTheReceiverIsUnreachable(t *testing.T) {
	s := newExampleStack(t)
	unreachable := httptest.NewTLSServer(http.NotFoundHandler())
	endpoint := unreachable.URL
	unreachable.Close()
	streamID := s.registerTransmitter(t, endpoint)
	s.seedAgent(t, tenancydomain.DefaultTenantID, exampleAgentID)
	at1 := s.issueAgentToken(t)

	before, after := s.killAgent(t)

	s.assertEpochAdvancedByKill(t, before, after)
	if active := s.introspect(t, at1)["active"]; active != false {
		t.Fatalf("after kill active=%v, want false", active)
	}
	deliveries, err := s.deliveries.ListByStream(context.Background(), tenancydomain.DefaultTenantID, streamID)
	if err != nil {
		t.Fatal(err)
	}
	if len(deliveries) != 1 || deliveries[0].Status != ssdomain.SecurityEventDeliveryStatusPending || deliveries[0].AttemptCount != 0 {
		t.Fatalf("deliveries=%+v, want one pending delivery with no attempt", deliveries)
	}
	due, err := s.deliveries.ListDue(context.Background(), time.Now().UTC(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 1 || due[0].ID != deliveries[0].ID {
		t.Fatalf("due=%+v, want the pending delivery to be due for an attempt", due)
	}
}

// 無効化したストリームと、有効なまま残したストリームを並べる。有効なほうが受理し、
// 配送を作ることが対照になる。
//
//spec:covers EX-SHAREDSIGNALS-008-01: DisableSsfStream の後、受信ストリームへの正しい SET は 400 で拒否されて失効エポックを進めず、送信ストリームには強制終了による SecurityEventDelivery が作られない。同じ SET を再有効化した受信ストリームは受理し、有効な送信ストリームには配送が作られる。
func TestDisabledStreamNeitherReceivesNorTransmits(t *testing.T) {
	t.Run("receive", func(t *testing.T) {
		s := newExampleStack(t)
		s.seedAgent(t, tenancydomain.DefaultTenantID, exampleAgentID)
		key := newTransmitterKey(t)
		streamID := s.registerReceiver(t, key)
		if rec := s.admin(t, http.MethodPost, "/api/admin/v1/shared-signals/streams/"+streamID+"/disable", nil); rec.Code != http.StatusNoContent {
			t.Fatalf("disable status=%d body=%s", rec.Code, rec.Body.String())
		}
		token := key.sign(t, "J-disabled", issSub(exampleAgentID))

		assertSETRejected(t, s.postSET(t, streamID, token))
		if epoch := s.epoch(t, tenancydomain.DefaultTenantID, exampleAgentID); epoch != nil {
			t.Fatalf("a disabled stream applied a SET: %+v", epoch)
		}

		if rec := s.admin(t, http.MethodPost, "/api/admin/v1/shared-signals/streams/"+streamID+"/enable", nil); rec.Code != http.StatusNoContent {
			t.Fatalf("enable status=%d body=%s", rec.Code, rec.Body.String())
		}
		if rec := s.postSET(t, streamID, token); rec.Code != http.StatusAccepted {
			t.Fatalf("re-enabled stream: status=%d body=%s, want 202", rec.Code, rec.Body.String())
		}
	})

	t.Run("transmit", func(t *testing.T) {
		s := newExampleStack(t)
		disabled := s.registerTransmitter(t, "https://receiver.example/disabled")
		enabled := s.registerTransmitter(t, "https://receiver.example/enabled")
		if rec := s.admin(t, http.MethodPost, "/api/admin/v1/shared-signals/streams/"+disabled+"/disable", nil); rec.Code != http.StatusNoContent {
			t.Fatalf("disable status=%d body=%s", rec.Code, rec.Body.String())
		}
		s.seedAgent(t, tenancydomain.DefaultTenantID, exampleAgentID)

		s.killAgent(t)

		for streamID, want := range map[string]int{disabled: 0, enabled: 1} {
			deliveries, err := s.deliveries.ListByStream(context.Background(), tenancydomain.DefaultTenantID, streamID)
			if err != nil {
				t.Fatal(err)
			}
			if len(deliveries) != want {
				t.Fatalf("stream %s: %d deliveries, want %d", streamID, len(deliveries), want)
			}
		}
	})
}

// newStackAtStreamQuota は `ssf_streams` の上限 20 を、管理 API で登録した 20 本の
// ストリームで使い切った状態を作る。
func newStackAtStreamQuota(t *testing.T) *exampleStack {
	t.Helper()
	s := newExampleStack(t)
	for range 20 {
		s.registerTransmitter(t, "https://receiver.example/events")
	}
	if usage := s.streamUsage(t); usage != 20 {
		t.Fatalf("ssf_streams usage=%d after 20 registrations, want 20", usage)
	}
	return s
}

func (s *exampleStack) streamUsage(t *testing.T) int {
	t.Helper()
	usage, err := s.quotas.GetUsage(context.Background(), tenancydomain.DefaultTenantID)
	if err != nil {
		t.Fatal(err)
	}
	return usage.SsfStreams
}

func assertQuotaExceeded(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	var problem support.Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem: %v; body=%s", err, rec.Body.String())
	}
	if rec.Code != http.StatusUnprocessableEntity || problem.Type != "urn:idmagic:error:quota_exceeded" {
		t.Fatalf("status=%d type=%q, want 422 quota_exceeded", rec.Code, problem.Type)
	}
}

// 拒否された登録がストリームも付随する設定も残していないことを、ストリームの件数と
// 設定の保存回数で読む。
//
//spec:covers EX-SHAREDSIGNALS-009-01: 利用量が上限 20 に達したテナントでの RegisterSsfTransmitterStream は 422 quota_exceeded で拒否され、ストリームも送信側設定も作られず、QuotaExceeded が resource=ssf_streams で発行され、利用量は 20 のままである。
func TestTransmitterStreamRegistrationStopsAtTheHardQuota(t *testing.T) {
	s := newStackAtStreamQuota(t)
	configSaves := s.transmitterConfigs.saves

	assertQuotaExceeded(t, s.admin(t, http.MethodPost, "/api/admin/v1/shared-signals/streams/transmitter", map[string]any{
		"delivery_endpoint": "https://receiver.example/events", "audience": "https://receiver.example",
		"event_types": []string{string(ssdomain.CaepEventTypeSessionRevoked)},
	}))

	if n := len(s.listStreams(t)); n != 20 {
		t.Fatalf("streams=%d after the refused registration, want 20", n)
	}
	if s.transmitterConfigs.saves != configSaves {
		t.Fatalf("transmitter config saves went from %d to %d on a refused registration", configSaves, s.transmitterConfigs.saves)
	}
	var exceeded bool
	for _, event := range s.events.all() {
		if e, ok := event.(*tenancydomain.QuotaExceeded); ok && e.Resource == tenancydomain.ResourceSsfStreams && e.HardLimit {
			exceeded = true
		}
	}
	if !exceeded {
		t.Fatalf("QuotaExceeded(ssf_streams) was not emitted: %v", s.events.all())
	}
	if usage := s.streamUsage(t); usage != 20 {
		t.Fatalf("ssf_streams usage=%d after the refusal, want 20", usage)
	}
}

// 20 本はすべて送信側なので、受信側の登録が拒否されるのは送信側と上限を共有している
// ときだけである。
//
//spec:covers EX-SHAREDSIGNALS-009-02: 送信側のストリームで上限 20 を使い切ったテナントでは RegisterSsfReceiverStream も 422 quota_exceeded で拒否され、ストリームも受信側設定も作られない。
func TestReceiverStreamRegistrationSharesTheHardQuota(t *testing.T) {
	s := newStackAtStreamQuota(t)

	assertQuotaExceeded(t, s.admin(t, http.MethodPost, "/api/admin/v1/shared-signals/streams/receiver",
		receiverRequest(newTransmitterKey(t))))

	if n := len(s.listStreams(t)); n != 20 {
		t.Fatalf("streams=%d after the refused registration, want 20", n)
	}
	if s.receiverConfigs.saves != 0 {
		t.Fatalf("receiver config saved %d times on a refused registration", s.receiverConfigs.saves)
	}
}

//spec:covers EX-SHAREDSIGNALS-009-03: 上限 20 に達したテナントで DeleteSsfStream を呼ぶと、ストリームが消えて利用量が 19 に戻り、次の登録が 201 で成功して利用量が 20 に戻る。
func TestDeletingAStreamFreesTheHardQuota(t *testing.T) {
	s := newStackAtStreamQuota(t)
	victim := s.listStreams(t)[0].ID

	if rec := s.admin(t, http.MethodDelete, "/api/admin/v1/shared-signals/streams/"+victim, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", rec.Code, rec.Body.String())
	}
	if s.findStream(t, victim) != nil {
		t.Fatalf("stream %s still exists after delete", victim)
	}
	if usage := s.streamUsage(t); usage != 19 {
		t.Fatalf("ssf_streams usage=%d after delete, want 19", usage)
	}

	s.registerReceiver(t, newTransmitterKey(t))
	if usage := s.streamUsage(t); usage != 20 {
		t.Fatalf("ssf_streams usage=%d after the next registration, want 20", usage)
	}
}

func assertAccessDenied(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	var problem support.Problem
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem: %v; body=%s", err, rec.Body.String())
	}
	if rec.Code != http.StatusForbidden || problem.Type != "urn:idmagic:error:access_denied" {
		t.Fatalf("status=%d type=%q body=%s, want 403 access_denied", rec.Code, problem.Type, rec.Body.String())
	}
}

// streamSnapshot は拒否の前後で比べるストリームの状態である。
type streamSnapshot struct {
	status     ssdomain.SsfStreamStatus
	eventTypes []ssdomain.CaepEventType
}

func (s *exampleStack) snapshot(t *testing.T) map[string]streamSnapshot {
	t.Helper()
	out := map[string]streamSnapshot{}
	for _, stream := range s.listStreams(t) {
		out[stream.ID] = streamSnapshot{status: stream.Status, eventTypes: slices.Clone(stream.EventTypes)}
	}
	return out
}

func assertSnapshotUnchanged(t *testing.T, before, after map[string]streamSnapshot) {
	t.Helper()
	if len(before) != len(after) {
		t.Fatalf("streams went from %d to %d", len(before), len(after))
	}
	for id, want := range before {
		got, ok := after[id]
		if !ok || got.status != want.status || !slices.Equal(got.eventTypes, want.eventTypes) {
			t.Fatalf("stream %s changed from %+v to %+v (present=%v)", id, want, got, ok)
		}
	}
}

var streamLifecycleEvents = []string{"SsfStreamRegistered", "SsfStreamUpdated", "SsfStreamDisabled", "SsfStreamEnabled", "SsfStreamDeleted"}

func (s *exampleStack) lifecycleEventCount() int {
	n := 0
	for _, eventType := range streamLifecycleEvents {
		n += s.events.count(eventType)
	}
	return n
}

// 管理者でない利用者の登録を、既存のストリームがある状態で拒否させる。
//
//spec:covers EX-SHAREDSIGNALS-011-01: admin ロールを持たない "alice" の送信側と受信側の登録はどちらも 403 access_denied で拒否され、ストリームは増えず、既存のストリームの状態も変わらず、SsfStreamRegistered も発行されない。
func TestNonAdminCannotRegisterStreams(t *testing.T) {
	s := newExampleStack(t)
	s.registerTransmitter(t, "https://receiver.example/events")
	before := s.snapshot(t)
	events := s.lifecycleEventCount()

	assertAccessDenied(t, s.request(t, exampleNonAdmin, http.MethodPost, "/api/admin/v1/shared-signals/streams/transmitter", map[string]any{
		"delivery_endpoint": "https://receiver.example/events", "audience": "https://receiver.example",
		"event_types": []string{string(ssdomain.CaepEventTypeSessionRevoked)},
	}))
	assertAccessDenied(t, s.request(t, exampleNonAdmin, http.MethodPost, "/api/admin/v1/shared-signals/streams/receiver",
		receiverRequest(newTransmitterKey(t))))

	assertSnapshotUnchanged(t, before, s.snapshot(t))
	if n := s.lifecycleEventCount(); n != events {
		t.Fatalf("stream lifecycle events went from %d to %d on refused registrations", events, n)
	}
	if s.transmitterConfigs.saves != 1 || s.receiverConfigs.saves != 0 {
		t.Fatalf("config saves transmitter=%d receiver=%d, want 1 and 0", s.transmitterConfigs.saves, s.receiverConfigs.saves)
	}
}

// 有効と無効のストリームを 1 本ずつ置き、どちらの向きの状態変更も拒否させる。有効化の
// 拒否は無効なストリームに対してでなければ、何も変えない要求と見分けられない。
//
//spec:covers EX-SHAREDSIGNALS-011-02: admin ロールを持たない "alice" の無効化、有効化、更新、削除はすべて 403 access_denied で拒否され、ストリームの有無、状態、購読するイベント種別は変わらず、ストリームのライフサイクルイベントも発行されない。
func TestNonAdminCannotChangeStreams(t *testing.T) {
	s := newExampleStack(t)
	enabled := s.registerTransmitter(t, "https://receiver.example/enabled")
	disabled := s.registerReceiver(t, newTransmitterKey(t))
	if rec := s.admin(t, http.MethodPost, "/api/admin/v1/shared-signals/streams/"+disabled+"/disable", nil); rec.Code != http.StatusNoContent {
		t.Fatalf("disable status=%d body=%s", rec.Code, rec.Body.String())
	}
	before := s.snapshot(t)
	events := s.lifecycleEventCount()

	for _, attempt := range []struct {
		method, path string
		body         any
	}{
		{http.MethodPost, enabled + "/disable", nil},
		{http.MethodPost, disabled + "/enable", nil},
		{http.MethodPatch, enabled, map[string]any{"event_types": []string{string(ssdomain.CaepEventTypeCredentialChange)}}},
		{http.MethodDelete, enabled, nil},
		{http.MethodDelete, disabled, nil},
	} {
		assertAccessDenied(t, s.request(t, exampleNonAdmin, attempt.method, "/api/admin/v1/shared-signals/streams/"+attempt.path, attempt.body))
	}

	assertSnapshotUnchanged(t, before, s.snapshot(t))
	if n := s.lifecycleEventCount(); n != events {
		t.Fatalf("stream lifecycle events went from %d to %d on refused changes", events, n)
	}
}
