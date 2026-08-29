package handlers_http_test

// /userinfo + DPoP PoP 検証 (RFC 9449 §7) のテスト。SenderConstraintDPoP の AT は
// 同じ鍵で署名された DPoP proof と htm / htu / iat / jti / ath を伴わない限り受理しない。

import (
	"bytes"
	"context"
	cryptostd "crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/oauth2"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	memory "github.com/ambi/idmagic/backend/oauth2/db_memory"
	"github.com/ambi/idmagic/backend/oauth2/domain"
	oauthports "github.com/ambi/idmagic/backend/oauth2/ports"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	tokensJOSE "github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"

	"github.com/labstack/echo/v5"
)

type fakeIntrospector struct {
	result *oauthports.IntrospectionResult
}

func (f *fakeIntrospector) IntrospectAccessToken(_ context.Context, _ string) (*oauthports.IntrospectionResult, error) {
	return f.result, nil
}

func rsaJWK(pub *rsa.PublicKey) map[string]any {
	return map[string]any{
		"kty": "RSA",
		"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(new(big.Int).SetInt64(int64(pub.E)).Bytes()),
	}
}

func rsaJWKThumbprint(t *testing.T, jwk map[string]any) string {
	t.Helper()
	// RFC 7638: required メンバーのみ、辞書順、空白なしの canonical JSON を SHA-256。
	canonical, err := json.Marshal(map[string]any{"e": jwk["e"], "kty": jwk["kty"], "n": jwk["n"]})
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(canonical)
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// signDPoPProof signs a GET proof for a protected resource. ath is derived from
// accessToken; an empty accessToken yields a proof without ath (a REQ-OAUTH2-045
// rejection case).
func signDPoPProof(t *testing.T, key *rsa.PrivateKey, jwk map[string]any, htu, jti, accessToken string, now time.Time) string {
	t.Helper()
	header, err := json.Marshal(map[string]any{"typ": "dpop+jwt", "alg": "PS256", "jwk": jwk})
	if err != nil {
		t.Fatal(err)
	}
	claims := map[string]any{"htm": http.MethodGet, "htu": htu, "jti": jti, "iat": now.Unix()}
	if accessToken != "" {
		claims["ath"] = tokensJOSE.AccessTokenHash(accessToken)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	sig, err := rsa.SignPSS(
		rand.Reader, key, cryptostd.SHA256, digest[:],
		&rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash},
	)
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func TestUserInfoDPoPBoundRequiresMatchingProof(t *testing.T) {
	now := time.Now().UTC()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwk := rsaJWK(&key.PublicKey)
	jkt := rsaJWKThumbprint(t, jwk)

	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(&userdomain.User{
		ID: "user_alice", PreferredUsername: "alice", TenantID: tenancydomain.DefaultTenantID,
		CreatedAt: now, UpdatedAt: now,
	})

	intro := &fakeIntrospector{result: &oauthports.IntrospectionResult{
		Active: true, Sub: "user_alice", Scope: "openid profile",
		ClientID:         "demo-client",
		SenderConstraint: &domain.SenderConstraint{Type: spec.SenderConstraintDPoP, JKT: jkt},
	}}

	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "http://test", UserRepo: userRepo,
		OAuth2:            oauth2.Module{DpopReplayStore: memory.NewDpopReplayStore()},
		TokenIntrospector: intro,
	})

	call := func(authHeader, proof string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/realms/default/userinfo", http.NoBody)
		req.Header.Set("Authorization", authHeader)
		if proof != "" {
			req.Header.Set("DPoP", proof)
		}
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}

	mustReject := func(name string, rec *httptest.ResponseRecorder) {
		t.Helper()
		if rec.Code == http.StatusOK ||
			!bytes.Contains(rec.Body.Bytes(), []byte(`"error":"invalid_token"`)) {
			t.Fatalf("%s: status=%d body=%s", name, rec.Code, rec.Body.String())
		}
	}

	// 有効プルーフ。htu は requestHTU と一致する形 (base + path)。
	validProof := signDPoPProof(t, key, jwk, "http://test/realms/default/userinfo", "jti-valid", "atoken", now)
	if rec := call("DPoP atoken", validProof); rec.Code != http.StatusOK {
		t.Fatalf("valid proof status=%d body=%s", rec.Code, rec.Body.String())
	}

	// DPoP ヘッダー欠落 → invalid_token。
	mustReject("missing DPoP proof", call("DPoP atoken", ""))

	// REQ-OAUTH2-045: a proof without ath only shows key possession, so invalid_token.
	noATHProof := signDPoPProof(t, key, jwk, "http://test/realms/default/userinfo", "jti-no-ath", "", now)
	mustReject("missing ath", call("DPoP atoken", noATHProof))

	// REQ-OAUTH2-045: a proof made for another access token cannot be reused here.
	otherATHProof := signDPoPProof(t, key, jwk, "http://test/realms/default/userinfo", "jti-other-ath", "othertoken", now)
	mustReject("ath of another access token", call("DPoP atoken", otherATHProof))

	// 別鍵で署名された proof → 署名検証は通っても jkt が一致せず invalid_token。
	attacker, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	attJWK := rsaJWK(&attacker.PublicKey)
	attProof := signDPoPProof(t, attacker, attJWK, "http://test/realms/default/userinfo", "jti-attacker", "atoken", now)
	mustReject("wrong jkt", call("DPoP atoken", attProof))
}

func TestUserInfoDPoPHTUUsesTenantPrefix(t *testing.T) {
	// /realms/{tenant}/userinfo にアクセスした場合、DPoP proof の htu に
	// テナント prefix が含まれていれば受理される (Phase 1 #3 の回帰防止)。
	now := time.Now().UTC()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwk := rsaJWK(&key.PublicKey)
	jkt := rsaJWKThumbprint(t, jwk)

	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(&userdomain.User{
		ID: "user_bob", PreferredUsername: "bob", TenantID: "acme",
		CreatedAt: now, UpdatedAt: now,
	})

	intro := &fakeIntrospector{result: &oauthports.IntrospectionResult{
		Active: true, Sub: "user_bob", Scope: "openid", ClientID: "tenant-client",
		SenderConstraint: &domain.SenderConstraint{Type: spec.SenderConstraintDPoP, JKT: jkt},
	}}

	// "acme" テナントを返す TenantRepository をその場で組む。
	tenantRepo := newSingleTenantRepo()

	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:     "http://test",
		TenantRepo: tenantRepo, UserRepo: userRepo,
		OAuth2:            oauth2.Module{DpopReplayStore: memory.NewDpopReplayStore()},
		TokenIntrospector: intro,
	})

	htu := "http://test/realms/acme/userinfo"
	proof := signDPoPProof(t, key, jwk, htu, "jti-acme", "atoken", now)

	req := httptest.NewRequest(http.MethodGet, "/realms/acme/userinfo", http.NoBody)
	req.Header.Set("Authorization", "DPoP atoken")
	req.Header.Set("DPoP", proof)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("tenant-prefixed userinfo status=%d body=%s", rec.Code, rec.Body.String())
	}
}

// newSingleTenantRepo は指定 ID の Active テナントだけを返す最小の TenantRepository。
type singleTenantRepo struct {
	tenant *tenancydomain.Tenant
}

func newSingleTenantRepo() *singleTenantRepo {
	now := time.Now().UTC()
	return &singleTenantRepo{tenant: &tenancydomain.Tenant{
		ID: "acme", Realm: "acme", Status: tenancydomain.TenantStatusActive, CreatedAt: now,
	}}
}

func (r *singleTenantRepo) FindByID(_ context.Context, id string) (*tenancydomain.Tenant, error) {
	if r.tenant.ID == id {
		return r.tenant, nil
	}
	if id == tenancydomain.DefaultTenantID {
		return &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, Status: tenancydomain.TenantStatusActive}, nil
	}
	return nil, nil
}

func (r *singleTenantRepo) FindByRealm(_ context.Context, realm string) (*tenancydomain.Tenant, error) {
	if r.tenant.Realm == realm {
		return r.tenant, nil
	}
	if realm == tenancydomain.DefaultRealm {
		return &tenancydomain.Tenant{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, Status: tenancydomain.TenantStatusActive}, nil
	}
	return nil, nil
}

func (r *singleTenantRepo) FindAll(_ context.Context) ([]*tenancydomain.Tenant, error) {
	return []*tenancydomain.Tenant{r.tenant}, nil
}

func (r *singleTenantRepo) Save(_ context.Context, _ *tenancydomain.Tenant) error { return nil }
func (r *singleTenantRepo) Delete(_ context.Context, _ string) error              { return nil }

// fakeDenylist は AccessToken denylist を任意の jti セットで再現する。
type fakeDenylist struct {
	revoked map[string]bool
}

func (f *fakeDenylist) Add(_ context.Context, jti string, _ time.Time) error {
	if f.revoked == nil {
		f.revoked = map[string]bool{}
	}
	f.revoked[jti] = true
	return nil
}

func (f *fakeDenylist) IsRevoked(_ context.Context, jti string) (bool, error) {
	return f.revoked[jti], nil
}

func newUserInfoServer(t *testing.T, intro *fakeIntrospector, denylist *fakeDenylist) *echo.Echo {
	t.Helper()
	userRepo := usermemory.NewUserRepository()
	now := time.Now().UTC()
	userRepo.Seed(&userdomain.User{
		ID: "user_alice", PreferredUsername: "alice",
		TenantID: tenancydomain.DefaultTenantID, CreatedAt: now, UpdatedAt: now,
	})
	e := echo.New()
	deps := httpadapter.Deps{
		Issuer: "http://test", UserRepo: userRepo,
		OAuth2:            oauth2.Module{DpopReplayStore: memory.NewDpopReplayStore()},
		TokenIntrospector: intro,
	}
	if denylist != nil {
		deps.OAuth2.AccessTokenDenylist = denylist
	}
	httpadapter.Register(e, deps)
	return e
}

func TestUserInfoRejectsTokenWithoutOpenIDScope(t *testing.T) {
	// SCL シナリオ "openid スコープのないトークンのユーザー情報取得は拒否される"。
	intro := &fakeIntrospector{result: &oauthports.IntrospectionResult{
		Active: true, Sub: "user_alice", Scope: "profile", ClientID: "demo-client",
	}}
	e := newUserInfoServer(t, intro, nil)
	req := httptest.NewRequest(http.MethodGet, "/realms/default/userinfo", http.NoBody)
	req.Header.Set("Authorization", "Bearer atoken")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code == http.StatusOK ||
		!bytes.Contains(rec.Body.Bytes(), []byte(`"error":"insufficient_scope"`)) {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestUserInfoRejectsRevokedAccessToken(t *testing.T) {
	// SCL シナリオ "失効した access_token でユーザー情報取得は invalid_token で拒否される"。
	intro := &fakeIntrospector{result: &oauthports.IntrospectionResult{
		Active: true, Sub: "user_alice", Scope: "openid", ClientID: "demo-client",
		JTI: "revoked-jti",
	}}
	denylist := &fakeDenylist{revoked: map[string]bool{"revoked-jti": true}}
	e := newUserInfoServer(t, intro, denylist)
	req := httptest.NewRequest(http.MethodGet, "/realms/default/userinfo", http.NoBody)
	req.Header.Set("Authorization", "Bearer atoken")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code == http.StatusOK ||
		!bytes.Contains(rec.Body.Bytes(), []byte(`"error":"invalid_token"`)) {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestUserInfoMTLSBoundRequiresMatchingThumbprint(t *testing.T) {
	// SCL シナリオ "mTLS バインド AT は同じ証明書のリクエストでのみ受理される"。
	// 期待 thumbprint と異なる証明書を提示すると invalid_token。
	intro := &fakeIntrospector{result: &oauthports.IntrospectionResult{
		Active: true, Sub: "user_alice", Scope: "openid", ClientID: "demo-client",
		SenderConstraint: &domain.SenderConstraint{
			Type: spec.SenderConstraintMTLS, X5TS256: "expected-thumbprint-not-matching-any-real-cert",
		},
	}}
	e := newUserInfoServer(t, intro, nil)
	req := httptest.NewRequest(http.MethodGet, "/realms/default/userinfo", http.NoBody)
	req.Header.Set("Authorization", "Bearer atoken")
	req.Header.Set("X-Client-Certificate", clientCertificateHeader(t, "client"))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code == http.StatusOK ||
		!bytes.Contains(rec.Body.Bytes(), []byte(`"error":"invalid_token"`)) {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

// clientCertificateHeader はテスト用の自己署名証明書を PEM/URL エンコードして返す。
// router 経由の mTLS 検証テストで X-Client-Certificate ヘッダーに用いる。
func clientCertificateHeader(t *testing.T, commonName string) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return url.QueryEscape(string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})))
}

// ecJWK は P-256 公開鍵を RFC 7518 §6.2.1 の JWK として表す。
func ecJWK(t *testing.T, pub *ecdsa.PublicKey) map[string]any {
	t.Helper()
	point, err := pub.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	return map[string]any{
		"kty": "EC",
		"crv": "P-256",
		"x":   base64.RawURLEncoding.EncodeToString(point[1:33]),
		"y":   base64.RawURLEncoding.EncodeToString(point[33:65]),
	}
}

// ecJWKThumbprint は EC 鍵の RFC 7638 サムプリントを、正規メンバー集合
// {crv, kty, x, y} から検証対象の実装とは独立に組み立てる。
func ecJWKThumbprint(t *testing.T, jwk map[string]any) string {
	t.Helper()
	canonical, err := json.Marshal(map[string]any{
		"crv": jwk["crv"], "kty": jwk["kty"], "x": jwk["x"], "y": jwk["y"],
	})
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(canonical)
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// signECDPoPProof は ES256 (ECDSA P-256 + SHA-256, JWS の R||S 形式) で GET proof を署名する。
func signECDPoPProof(t *testing.T, key *ecdsa.PrivateKey, jwk map[string]any, htu, jti, accessToken string, now time.Time) string {
	t.Helper()
	header, err := json.Marshal(map[string]any{"typ": "dpop+jwt", "alg": "ES256", "jwk": jwk})
	if err != nil {
		t.Fatal(err)
	}
	claims := map[string]any{"htm": http.MethodGet, "htu": htu, "jti": jti, "iat": now.Unix()}
	if accessToken != "" {
		claims["ath"] = tokensJOSE.AccessTokenHash(accessToken)
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." +
		base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	r, s, err := ecdsa.Sign(rand.Reader, key, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])
	return input + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func TestUserInfoDPoPAcceptsES256Proof(t *testing.T) {
	// RFC9449-TOKEN-BINDING / REQ-OAUTH2-045: DPoP proof の alg として ES256 を
	// 宣言どおり受理する以上、EC 鍵でも jkt の照合が成立しなければならない。
	now := time.Now().UTC()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	jwk := ecJWK(t, &key.PublicKey)
	jkt := ecJWKThumbprint(t, jwk)

	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(&userdomain.User{
		ID: "user_carol", PreferredUsername: "carol", TenantID: tenancydomain.DefaultTenantID,
		CreatedAt: now, UpdatedAt: now,
	})

	intro := &fakeIntrospector{result: &oauthports.IntrospectionResult{
		Active: true, Sub: "user_carol", Scope: "openid profile",
		ClientID:         "demo-client",
		SenderConstraint: &domain.SenderConstraint{Type: spec.SenderConstraintDPoP, JKT: jkt},
	}}

	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer: "http://test", UserRepo: userRepo,
		OAuth2:            oauth2.Module{DpopReplayStore: memory.NewDpopReplayStore()},
		TokenIntrospector: intro,
	})

	call := func(proof string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/realms/default/userinfo", http.NoBody)
		req.Header.Set("Authorization", "DPoP atoken")
		req.Header.Set("DPoP", proof)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec
	}

	htu := "http://test/realms/default/userinfo"
	if rec := call(signECDPoPProof(t, key, jwk, htu, "es256-valid", "atoken", now)); rec.Code != http.StatusOK {
		t.Fatalf("valid ES256 proof status=%d body=%s", rec.Code, rec.Body.String())
	}

	// 結合が実際に効いていること。別の EC 鍵の proof は jkt が一致せず拒否される。
	attacker, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	attProof := signECDPoPProof(t, attacker, ecJWK(t, &attacker.PublicKey), htu, "es256-attacker", "atoken", now)
	if rec := call(attProof); rec.Code == http.StatusOK ||
		!bytes.Contains(rec.Body.Bytes(), []byte(`"error":"invalid_token"`)) {
		t.Fatalf("wrong ES256 jkt: status=%d body=%s", rec.Code, rec.Body.String())
	}
}
