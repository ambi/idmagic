package handlers_http_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"maps"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"github.com/ambi/idmagic/backend/authentication"
	"github.com/ambi/idmagic/backend/idmanagement"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/db_memory"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/domain"
	directoryhttp "github.com/ambi/idmagic/backend/sourcing/internal/directory/handlers_http"
	"github.com/ambi/idmagic/backend/sourcing/internal/directory/usecases"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

const issuer = "http://idp.test"

// otherTenantID は、X-Test-Realm: other の要求が解決するテナント。
const otherTenantID = "00000000-0000-0000-0000-0000000000b2"

type replayStore struct {
	mu   sync.Mutex
	seen map[string]bool
}

func (s *replayStore) RecordIfNew(_ context.Context, jti string, _ int, _ time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.seen[jti] {
		return false, nil
	}
	s.seen[jti] = true
	return true, nil
}

type server struct {
	t      *testing.T
	e      *echo.Echo
	now    time.Time
	mu     sync.Mutex
	events []string
}

func newServer(t *testing.T) *server {
	t.Helper()
	s := &server{t: t, now: time.Date(2026, 10, 24, 9, 0, 0, 0, time.UTC)}
	users := idmanagement.NewMemoryModule().UserRepo
	for _, u := range []userdomain.User{
		{ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin", Roles: []string{"admin"}},
		{ID: "member", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "member"},
		{ID: "other-admin", TenantID: otherTenantID, PreferredUsername: "admin", Roles: []string{"admin"}},
	} {
		u.PasswordHash, u.CreatedAt, u.UpdatedAt = "unused", s.now, s.now
		if err := users.Save(context.Background(), &u); err != nil {
			t.Fatal(err)
		}
	}
	service := usecases.New(db_memory.NewConnectorRepository(), &replayStore{seen: map[string]bool{}},
		func(event spec.DomainEvent) {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.events = append(s.events, event.EventType())
		},
		usecases.WithClock(func() time.Time { return s.now }),
	)
	deps := support.Deps{Issuer: issuer}
	authenticator := &support.Authenticator{
		Principals: idmanagement.Module{UserRepo: users}.Principals(),
		Sessions:   authentication.Module{AuthnResolver: authentication.DemoHeaderResolver{}}.SessionAuthentications(),
	}
	s.e = echo.New()
	s.e.HTTPErrorHandler = support.ErrorHandler(nil, nil)
	directoryhttp.RegisterRoutes(s.e.Group("", resolveRealm), directoryhttp.Deps{Deps: deps, Authenticator: authenticator, Service: service})
	return s
}

// resolveRealm は、テナントの解決の代わりに X-Test-Realm の値でテナントを選ぶ。
func resolveRealm(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		id, realm := tenancydomain.DefaultTenantID, tenancydomain.DefaultRealm
		if c.Request().Header.Get("X-Test-Realm") == "other" {
			id, realm = otherTenantID, "other"
		}
		prefix := "/realms/" + realm
		tenant := &tenancydomain.Tenant{ID: id, Realm: realm, Status: tenancydomain.TenantStatusActive, EndpointStyle: tenancydomain.TenantEndpointStylePath}
		c.SetRequest(c.Request().WithContext(tenantports.WithTenant(c.Request().Context(), tenant, issuer+prefix, prefix)))
		return next(c)
	}
}

type call struct {
	method, path string
	body         any
	user         string // 管理画面の利用者。空なら Connector の要求にする
	bearer       string
	realm        string
	noCSRF       bool // 管理画面の要求から CSRF の二重送信を外す
}

func (s *server) do(c call) *httptest.ResponseRecorder {
	s.t.Helper()
	var payload []byte
	if c.body != nil {
		var err error
		if payload, err = json.Marshal(c.body); err != nil {
			s.t.Fatal(err)
		}
	}
	req := httptest.NewRequest(c.method, c.path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if c.user != "" {
		req.Header.Set("X-Demo-Sub", c.user)
		req.Header.Set("Origin", issuer)
		if !c.noCSRF {
			req.Header.Set(support.CSRFHeader, "csrf")
			req.AddCookie(&http.Cookie{Name: support.CSRFCookie, Value: "csrf"})
		}
	}
	if c.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearer)
	}
	if c.realm != "" {
		req.Header.Set("X-Test-Realm", c.realm)
	}
	rec := httptest.NewRecorder()
	s.e.ServeHTTP(rec, req)
	return rec
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(rec.Body.Bytes(), &value); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
	return value
}

type connectorJSON struct {
	ID                  string     `json:"id"`
	DisplayName         string     `json:"display_name"`
	Status              string     `json:"status"`
	CreatedAt           time.Time  `json:"created_at"`
	EnrollmentExpiresAt *time.Time `json:"enrollment_expires_at"`
	EnrolledAt          *time.Time `json:"enrolled_at"`
	KeyID               string     `json:"key_id"`
	LastSeenAt          *time.Time `json:"last_seen_at"`
	DisabledAt          *time.Time `json:"disabled_at"`
}

type grantJSON struct {
	Connector       connectorJSON `json:"connector"`
	EnrollmentToken string        `json:"enrollment_token"`
}

type enrollmentJSON struct {
	ConnectorID string `json:"connector_id"`
	KeyID       string `json:"key_id"`
	Audience    string `json:"audience"`
}

func (s *server) create(name string) grantJSON {
	s.t.Helper()
	rec := s.do(call{method: http.MethodPost, path: "/api/admin/v1/directory-connectors", body: map[string]any{"display_name": name}, user: "admin"})
	if rec.Code != http.StatusCreated {
		s.t.Fatalf("create status = %d body=%s", rec.Code, rec.Body.String())
	}
	return decode[grantJSON](s.t, rec)
}

func (s *server) get(id string) connectorJSON {
	s.t.Helper()
	rec := s.do(call{method: http.MethodGet, path: "/api/admin/v1/directory-connectors/" + id, user: "admin"})
	if rec.Code != http.StatusOK {
		s.t.Fatalf("get status = %d body=%s", rec.Code, rec.Body.String())
	}
	return decode[connectorJSON](s.t, rec)
}

type connectorKey struct {
	private *ecdsa.PrivateKey
	jwk     map[string]any
}

func newKey(t *testing.T) connectorKey {
	t.Helper()
	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	point, err := private.PublicKey.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	return connectorKey{private: private, jwk: map[string]any{
		"kty": "EC", "crv": "P-256",
		"x": base64.RawURLEncoding.EncodeToString(point[1:33]),
		"y": base64.RawURLEncoding.EncodeToString(point[33:65]),
	}}
}

func (s *server) enroll(token string, key connectorKey, realm string) *httptest.ResponseRecorder {
	s.t.Helper()
	return s.do(call{
		method: http.MethodPost, path: "/directory-connector/v1/enrollment",
		body: map[string]any{"enrollment_token": token, "jwk": key.jwk}, realm: realm,
	})
}

// sign は Connector が作る ES256 の JWT を返す。claims は既定の値を上書きする。
func (s *server) sign(key connectorKey, enrollment enrollmentJSON, claims map[string]any) string {
	s.t.Helper()
	payload := map[string]any{
		"iss": enrollment.ConnectorID, "sub": enrollment.ConnectorID, "aud": enrollment.Audience,
		"exp": s.now.Add(5 * time.Minute).Unix(), "jti": uuid.NewString(),
	}
	maps.Copy(payload, claims)
	encode := func(value any) string {
		raw, err := json.Marshal(value)
		if err != nil {
			s.t.Fatal(err)
		}
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	signingInput := encode(map[string]any{"alg": "ES256", "typ": "JWT", "kid": enrollment.KeyID}) + "." + encode(payload)
	digest := sha256.Sum256([]byte(signingInput))
	r, sigS, err := ecdsa.Sign(rand.Reader, key.private, digest[:])
	if err != nil {
		s.t.Fatal(err)
	}
	signature := append(fixed(r), fixed(sigS)...)
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func fixed(n *big.Int) []byte {
	out := make([]byte, 32)
	n.FillBytes(out)
	return out
}

func (s *server) heartbeat(bearer string) int {
	s.t.Helper()
	return s.do(call{method: http.MethodPost, path: "/directory-connector/v1/heartbeat", bearer: bearer}).Code
}

// enrolled は、作成と登録を済ませた Connector とその鍵を返す。
func (s *server) enrolled(name string) (enrollmentJSON, connectorKey) {
	s.t.Helper()
	grant := s.create(name)
	key := newKey(s.t)
	rec := s.enroll(grant.EnrollmentToken, key, "")
	if rec.Code != http.StatusOK {
		s.t.Fatalf("enroll status = %d body=%s", rec.Code, rec.Body.String())
	}
	return decode[enrollmentJSON](s.t, rec), key
}

// 主要ユースケース追跡: REQ-SOURCING-009。
//
//spec:covers REQ-SOURCING-008 REQ-SOURCING-009 REQ-SOURCING-010 REQ-SOURCING-013
func TestDirectoryConnector_EnrollsAndReportsWithItsKey(t *testing.T) {
	s := newServer(t)
	created := s.do(call{method: http.MethodPost, path: "/api/admin/v1/directory-connectors", body: map[string]any{"display_name": "  東京の AD  "}, user: "admin"})
	if created.Code != http.StatusCreated || created.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("create status=%d cache=%q body=%s", created.Code, created.Header().Get("Cache-Control"), created.Body.String())
	}
	grant := decode[grantJSON](t, created)
	if grant.Connector.Status != "awaiting_enrollment" || grant.Connector.DisplayName != "東京の AD" ||
		!strings.HasPrefix(grant.EnrollmentToken, "idmagic_dce_") || len(grant.EnrollmentToken) != len("idmagic_dce_")+43 ||
		grant.Connector.EnrollmentExpiresAt == nil || !grant.Connector.EnrollmentExpiresAt.Equal(s.now.Add(time.Hour)) {
		t.Fatalf("grant = %+v", grant)
	}

	key := newKey(t)
	s.now = s.now.Add(time.Hour)
	enrolledRec := s.enroll(grant.EnrollmentToken, key, "")
	if enrolledRec.Code != http.StatusOK {
		t.Fatalf("enroll status = %d body=%s", enrolledRec.Code, enrolledRec.Body.String())
	}
	enrollment := decode[enrollmentJSON](t, enrolledRec)
	if enrollment.ConnectorID != grant.Connector.ID || enrollment.KeyID == "" ||
		enrollment.Audience != issuer+"/realms/default/directory-connector/v1" {
		t.Fatalf("enrollment = %+v", enrollment)
	}
	if rec := s.enroll(grant.EnrollmentToken, newKey(t), ""); rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "invalid_token") {
		t.Fatalf("reused enrollment token status = %d body=%s", rec.Code, rec.Body.String())
	}

	s.now = s.now.Add(time.Minute)
	if code := s.heartbeat(s.sign(key, enrollment, nil)); code != http.StatusNoContent {
		t.Fatalf("heartbeat status = %d", code)
	}
	got := s.get(grant.Connector.ID)
	if got.Status != "active" || got.KeyID != enrollment.KeyID || got.EnrollmentExpiresAt != nil ||
		got.EnrolledAt == nil || !got.EnrolledAt.Equal(s.now.Add(-time.Minute)) ||
		got.LastSeenAt == nil || !got.LastSeenAt.Equal(s.now) {
		t.Fatalf("connector after heartbeat = %+v", got)
	}

	listed := s.do(call{method: http.MethodGet, path: "/api/admin/v1/directory-connectors", user: "admin"})
	if listed.Code != http.StatusOK || strings.Contains(listed.Body.String(), grant.EnrollmentToken) ||
		strings.Contains(listed.Body.String(), "hash") || strings.Contains(listed.Body.String(), "enrollment_token") {
		t.Fatalf("list status = %d body=%s", listed.Code, listed.Body.String())
	}
	if list := decode[struct {
		Connectors []connectorJSON `json:"connectors"`
	}](t, listed); len(list.Connectors) != 1 || list.Connectors[0].ID != grant.Connector.ID {
		t.Fatalf("list = %+v", list)
	}
	if want := []string{"DirectoryConnectorCreated", "DirectoryConnectorEnrolled"}; strings.Join(s.events, ",") != strings.Join(want, ",") {
		t.Fatalf("events = %v, want %v", s.events, want)
	}
}

// 主要ユースケース追跡: REQ-SOURCING-010。
//
//spec:covers REQ-SOURCING-010 REQ-SOURCING-011 REQ-SOURCING-012
func TestDirectoryConnector_RejectsCallsItCannotTrust(t *testing.T) {
	s := newServer(t)
	enrollment, key := s.enrolled("東京の AD")
	accepted := s.sign(key, enrollment, nil)
	if code := s.heartbeat(accepted); code != http.StatusNoContent {
		t.Fatalf("first heartbeat status = %d", code)
	}
	lastSeen := s.get(enrollment.ConnectorID).LastSeenAt
	s.now = s.now.Add(time.Second)

	otherEnrollment, otherKey := s.enrolled("大阪の AD")
	foreign := newKey(t)
	cases := []struct {
		name   string
		bearer string
		realm  string
	}{
		{"同じ jti の再利用", accepted, ""},
		{"登録していない鍵の署名", s.sign(foreign, enrollment, nil), ""},
		{"別の Connector の鍵の署名", s.sign(otherKey, enrollment, map[string]any{"kid": otherEnrollment.KeyID}), ""},
		{"発行者そのものの aud", s.sign(key, enrollment, map[string]any{"aud": issuer + "/realms/default"}), ""},
		{"5 分を超える有効期間", s.sign(key, enrollment, map[string]any{"exp": s.now.Add(7 * time.Minute).Unix()}), ""},
		{"期限切れ", s.sign(key, enrollment, map[string]any{"exp": s.now.Add(-2 * time.Minute).Unix()}), ""},
		{"jti がない", s.sign(key, enrollment, map[string]any{"jti": ""}), ""},
		{"sub が異なる", s.sign(key, enrollment, map[string]any{"sub": otherEnrollment.ConnectorID}), ""},
		{"別のテナント", s.sign(key, enrollment, map[string]any{"aud": issuer + "/realms/other/directory-connector/v1"}), "other"},
		{"形式が JWT でない", "not-a-jwt", ""},
		{"JWT がない", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := s.do(call{method: http.MethodPost, path: "/directory-connector/v1/heartbeat", bearer: tc.bearer, realm: tc.realm})
			if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "invalid_token") {
				t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
	if got := s.get(enrollment.ConnectorID).LastSeenAt; got == nil || !got.Equal(*lastSeen) {
		t.Fatalf("last seen changed to %v, want %v", got, lastSeen)
	}

	// 再発行は登録済みの鍵を破棄し、その鍵の JWT を拒否する。
	reissued := s.do(call{method: http.MethodPost, path: "/api/admin/v1/directory-connectors/" + enrollment.ConnectorID + "/enrollment-tokens", user: "admin"})
	if reissued.Code != http.StatusCreated || reissued.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("reissue status = %d body=%s", reissued.Code, reissued.Body.String())
	}
	grant := decode[grantJSON](t, reissued)
	if grant.Connector.Status != "awaiting_enrollment" || grant.Connector.KeyID != "" || grant.Connector.EnrolledAt != nil ||
		!strings.HasPrefix(grant.EnrollmentToken, "idmagic_dce_") {
		t.Fatalf("reissued = %+v", grant)
	}
	if code := s.heartbeat(s.sign(key, enrollment, nil)); code != http.StatusUnauthorized {
		t.Fatalf("heartbeat with the discarded key status = %d", code)
	}
	rekey := newKey(t)
	reenrolled := decode[enrollmentJSON](t, s.enroll(grant.EnrollmentToken, rekey, ""))
	if code := s.heartbeat(s.sign(rekey, reenrolled, nil)); code != http.StatusNoContent {
		t.Fatalf("heartbeat with the new key status = %d", code)
	}

	// 無効化した Connector の呼び出しを拒否する。
	disabledRec := s.do(call{method: http.MethodPost, path: "/api/admin/v1/directory-connectors/" + enrollment.ConnectorID + "/disable", user: "admin"})
	if disabledRec.Code != http.StatusOK {
		t.Fatalf("disable status = %d body=%s", disabledRec.Code, disabledRec.Body.String())
	}
	if got := decode[connectorJSON](t, disabledRec); got.Status != "disabled" || got.KeyID != "" || got.EnrolledAt == nil {
		t.Fatalf("disabled connector = %+v", got)
	}
	if code := s.heartbeat(s.sign(rekey, reenrolled, nil)); code != http.StatusUnauthorized {
		t.Fatalf("heartbeat after disabling status = %d", code)
	}
}

//spec:covers REQ-SOURCING-008 REQ-SOURCING-009 REQ-SOURCING-011 REQ-SOURCING-012 REQ-SOURCING-013 REQ-SOURCING-014
func TestDirectoryConnector_AdminOperations(t *testing.T) {
	s := newServer(t)
	admin := func(method, path string, body any) *httptest.ResponseRecorder {
		return s.do(call{method: method, path: path, body: body, user: "admin"})
	}
	expect := func(label string, rec *httptest.ResponseRecorder, status int, code string) {
		t.Helper()
		if rec.Code != status || (code != "" && !strings.Contains(rec.Body.String(), `"urn:idmagic:error:`+code+`"`)) {
			t.Fatalf("%s: status = %d body=%s, want %d %s", label, rec.Code, rec.Body.String(), status, code)
		}
	}

	expect("blank name", admin(http.MethodPost, "/api/admin/v1/directory-connectors", map[string]any{"display_name": " "}), http.StatusBadRequest, "invalid_request")
	expect("long name", admin(http.MethodPost, "/api/admin/v1/directory-connectors", map[string]any{"display_name": strings.Repeat("あ", 201)}), http.StatusUnprocessableEntity, "field_length_exceeded")
	for _, rec := range []*httptest.ResponseRecorder{
		s.do(call{method: http.MethodPost, path: "/api/admin/v1/directory-connectors", body: map[string]any{"display_name": "x"}, user: "member"}),
		s.do(call{method: http.MethodGet, path: "/api/admin/v1/directory-connectors", user: "member"}),
	} {
		expect("member", rec, http.StatusForbidden, "access_denied")
	}
	expect("without CSRF", s.do(call{method: http.MethodPost, path: "/api/admin/v1/directory-connectors", body: map[string]any{"display_name": "x"}, user: "admin", noCSRF: true}), http.StatusForbidden, "csrf_failed")
	expect("unauthenticated", s.do(call{method: http.MethodGet, path: "/api/admin/v1/directory-connectors"}), http.StatusUnauthorized, "")
	if len(s.events) != 0 {
		t.Fatalf("refused requests emitted %v", s.events)
	}

	first := s.create("先の Connector")
	s.now = s.now.Add(time.Second)
	second := s.create("後の Connector")
	list := decode[struct {
		Connectors []connectorJSON `json:"connectors"`
	}](t, admin(http.MethodGet, "/api/admin/v1/directory-connectors", nil))
	if len(list.Connectors) != 2 || list.Connectors[0].ID != first.Connector.ID || list.Connectors[1].ID != second.Connector.ID {
		t.Fatalf("list order = %+v", list.Connectors)
	}
	for _, id := range []string{uuid.NewString(), "not-a-uuid"} {
		expect("get unknown", admin(http.MethodGet, "/api/admin/v1/directory-connectors/"+id, nil), http.StatusNotFound, "directory_connector_not_found")
		expect("reissue unknown", admin(http.MethodPost, "/api/admin/v1/directory-connectors/"+id+"/enrollment-tokens", nil), http.StatusNotFound, "directory_connector_not_found")
		expect("disable unknown", admin(http.MethodPost, "/api/admin/v1/directory-connectors/"+id+"/disable", nil), http.StatusNotFound, "directory_connector_not_found")
	}
	other := s.do(call{method: http.MethodGet, path: "/api/admin/v1/directory-connectors/" + first.Connector.ID, user: "other-admin", realm: "other"})
	expect("other tenant", other, http.StatusNotFound, "directory_connector_not_found")

	// 公開鍵が条件を満たさない登録は、トークンを使用済みにしない。
	bad := newKey(t)
	bad.jwk["d"] = "AAAA"
	expect("private member", s.enroll(first.EnrollmentToken, bad, ""), http.StatusBadRequest, "invalid_request")
	expect("other tenant token", s.enroll(first.EnrollmentToken, newKey(t), "other"), http.StatusUnauthorized, "invalid_token")
	expect("valid after a bad key", s.enroll(first.EnrollmentToken, newKey(t), ""), http.StatusOK, "")

	// 有効期限を過ぎた登録トークンを拒否する。
	s.now = s.now.Add(time.Hour + time.Nanosecond)
	expect("expired token with a bad key", s.enroll(second.EnrollmentToken, bad, ""), http.StatusUnauthorized, "invalid_token")
	expect("expired token", s.enroll(second.EnrollmentToken, newKey(t), ""), http.StatusUnauthorized, "invalid_token")
	if got := s.get(second.Connector.ID); got.Status != "awaiting_enrollment" {
		t.Fatalf("expired enrollment changed the connector to %+v", got)
	}

	s.events = nil
	disabled := admin(http.MethodPost, "/api/admin/v1/directory-connectors/"+second.Connector.ID+"/disable", nil)
	expect("disable", disabled, http.StatusOK, "")
	if got := decode[connectorJSON](t, disabled); got.Status != "disabled" || got.DisabledAt == nil || got.EnrollmentExpiresAt != nil {
		t.Fatalf("disabled = %+v", got)
	}
	expect("disable again", admin(http.MethodPost, "/api/admin/v1/directory-connectors/"+second.Connector.ID+"/disable", nil), http.StatusOK, "")
	expect("reissue disabled", admin(http.MethodPost, "/api/admin/v1/directory-connectors/"+second.Connector.ID+"/enrollment-tokens", nil), http.StatusConflict, "directory_connector_disabled")
	expect("enroll disabled", s.enroll(second.EnrollmentToken, newKey(t), ""), http.StatusUnauthorized, "invalid_token")
	if want := "DirectoryConnectorDisabled"; strings.Join(s.events, ",") != want {
		t.Fatalf("events = %v, want %v", s.events, want)
	}
	if got := s.get(second.Connector.ID); got.Status != domainStatus(domain.StatusDisabled) {
		t.Fatalf("connector = %+v", got)
	}
}

func domainStatus(status domain.ConnectorStatus) string { return string(status) }
