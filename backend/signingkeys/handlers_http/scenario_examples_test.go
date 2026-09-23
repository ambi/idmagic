package handlers_http_test

// SigningKeys が宣言する具体例のうち、管理 API と JWKS から観測できるものを、製品と同じ
// `server_http.Register` の組み立てで観測する。
//
// ローテーションと無効化の効果は、応答ではなく `/jwks` と鍵ストアの現在の鍵から読み直す。拒否を
// 返しながらローテーションだけ済ませる実装は、応答だけを読むテストを通ってしまう。

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	stack "github.com/ambi/idmagic/backend/shared/http/testing_stack"
	"github.com/ambi/idmagic/backend/shared/spec"
	signingdomain "github.com/ambi/idmagic/backend/signingkeys/domain"
	signinghttp "github.com/ambi/idmagic/backend/signingkeys/handlers_http"
	signingports "github.com/ambi/idmagic/backend/signingkeys/ports"
	"github.com/ambi/idmagic/backend/signingkeys/usecases"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

const (
	realm          = tenancydomain.DefaultRealm
	adminKeysPath  = "/realms/" + realm + "/api/admin/v1/keys"
	rotatePath     = adminKeysPath + "/rotate"
	csrfTokenValue = "csrf-token-for-examples"
)

func newExampleStack(t *testing.T) *stack.Stack {
	t.Helper()
	return stack.New(t, stack.WithAuthorizationCodeFlow(), stack.WithApiTokens())
}

// sessionClient はログインセッションを持つブラウザーとして管理 API を呼ぶ。
// Origin と CSRF の二重送信は成立させておき、拒否があれば認証と認可によるものだけにする。
type sessionClient struct {
	s         *stack.Stack
	sessionID string
}

func signedInAdmin(t *testing.T, s *stack.Stack) sessionClient {
	t.Helper()
	authn, err := s.Sessions.Create(s.RealmContext(t, realm), stack.AdminUserID, []string{"pwd"}, time.Now().UTC())
	if err != nil || authn == nil || authn.SessionID == "" {
		t.Fatalf("create admin session: authn=%+v err=%v", authn, err)
	}
	return sessionClient{s: s, sessionID: authn.SessionID}
}

func (c sessionClient) post(t *testing.T, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := newRequest(t, http.MethodPost, path, body)
	request.Header.Set("Origin", stack.Issuer)
	request.Header.Set(support.CSRFHeader, csrfTokenValue)
	request.AddCookie(&http.Cookie{Name: sessionusecases.SessionCookie, Value: c.sessionID})
	request.AddCookie(&http.Cookie{Name: support.CSRFCookie, Value: csrfTokenValue})
	recorder := httptest.NewRecorder()
	c.s.Echo.ServeHTTP(recorder, request)
	return recorder
}

func bearerPost(t *testing.T, s *stack.Stack, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	request := newRequest(t, http.MethodPost, path, "")
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	s.Echo.ServeHTTP(recorder, request)
	return recorder
}

func newRequest(t *testing.T, method, path, body string) *http.Request {
	t.Helper()
	request := httptest.NewRequestWithContext(t.Context(), method, stack.Issuer+path, bytes.NewReader([]byte(body)))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	return request
}

// jwksKids は `/realms/<realm>/jwks` が公開している kid の集合を返す。
func jwksKids(t *testing.T, s *stack.Stack, jwksRealm string) []string {
	t.Helper()
	recorder := httptest.NewRecorder()
	s.Echo.ServeHTTP(recorder, newRequest(t, http.MethodGet, "/realms/"+jwksRealm+"/jwks", ""))
	if recorder.Code != http.StatusOK {
		t.Fatalf("jwks %s status=%d body=%s", jwksRealm, recorder.Code, recorder.Body.String())
	}
	var body struct {
		Keys []struct {
			Kid string `json:"kid"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode jwks: %v", err)
	}
	kids := make([]string, 0, len(body.Keys))
	for _, key := range body.Keys {
		kids = append(kids, key.Kid)
	}
	return kids
}

func activeKid(t *testing.T, s *stack.Stack, keyRealm string, usage signingdomain.KeyUsage) string {
	t.Helper()
	key, err := s.KeyStore.GetActiveKey(signingports.WithKeyUsage(s.RealmContext(t, keyRealm), usage))
	if err != nil || key == nil {
		t.Fatalf("active %s key of %s: key=%v err=%v", usage, keyRealm, key, err)
	}
	return key.Kid
}

func decodeRotation(t *testing.T, recorder *httptest.ResponseRecorder) signinghttp.AdminRotateKeyResponse {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("rotate status=%d body=%s, want 200", recorder.Code, recorder.Body.String())
	}
	var rotation signinghttp.AdminRotateKeyResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &rotation); err != nil {
		t.Fatalf("decode rotation: %v", err)
	}
	return rotation
}

func problemType(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var problem support.Problem
	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil {
		t.Fatalf("decode problem: %v; body=%s", err, recorder.Body.String())
	}
	return problem.Type
}

// rotatedTenant は管理者のセッションで 1 回ローテートし、現在の鍵 K2 と、JWKS に残る検証用鍵 K1 を返す。
func rotatedTenant(t *testing.T, s *stack.Stack) (admin sessionClient, k1, k2 string) {
	t.Helper()
	k1 = activeKid(t, s, realm, signingdomain.KeyUsageSigning)
	admin = signedInAdmin(t, s)
	rotation := decodeRotation(t, admin.post(t, rotatePath, ""))
	return admin, k1, rotation.Next.Kid
}

//spec:covers EX-SIGNINGKEYS-001-01: admin のログインセッションでローテートすると応答の next が新しい有効鍵として鍵ストアの現在の鍵になり、previous がローテーション前の kid を指し、/jwks には旧 kid と新 kid の両方が載る。
func TestAdministratorRotationKeepsThePreviousKidOnTheJWKS(t *testing.T) {
	s := newExampleStack(t)
	kidOld := activeKid(t, s, realm, signingdomain.KeyUsageSigning)

	rotation := decodeRotation(t, signedInAdmin(t, s).post(t, rotatePath, ""))

	kidNew := rotation.Next.Kid
	if kidNew == kidOld || !rotation.Next.Active {
		t.Fatalf("next = %+v, want a new active key distinct from %q", rotation.Next, kidOld)
	}
	if rotation.Previous == nil || rotation.Previous.Kid != kidOld || rotation.Previous.Active {
		t.Fatalf("previous = %+v, want the inactive %q", rotation.Previous, kidOld)
	}
	if current := activeKid(t, s, realm, signingdomain.KeyUsageSigning); current != kidNew {
		t.Fatalf("active kid = %q, want the rotated %q", current, kidNew)
	}
	if kids := jwksKids(t, s, realm); !slices.Contains(kids, kidOld) || !slices.Contains(kids, kidNew) {
		t.Fatalf("jwks kids = %v, want both %q and %q", kids, kidOld, kidNew)
	}
}

//spec:covers EX-SIGNINGKEYS-002-01: expires_at を過ぎた検証用鍵に、スケジューラーが呼ぶのと同じアーカイブのユースケースを実行すると、/jwks からその kid が消えて現在の鍵だけが残り、SigningKeyArchived がその kid と retiredAt、expiresAt、disposedAt を記録する。
func TestArchivingAnExpiredVerifyingKeyRemovesItFromTheJWKS(t *testing.T) {
	s := newExampleStack(t)
	ctx := s.RealmContext(t, realm)
	kidOld := activeKid(t, s, realm, signingdomain.KeyUsageSigning)
	now := time.Now().UTC()
	retiredAt := now.Add(-8 * 24 * time.Hour)
	const grace = 7 * 24 * time.Hour
	current, err := s.KeyStore.Rotate(ctx, retiredAt, grace)
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}

	var events []spec.DomainEvent
	archived, err := usecases.ArchiveExpiredSigningKeys(ctx, usecases.ArchiveExpiredSigningKeysDeps{
		KeyStore: s.KeyStore,
		Emit:     func(event spec.DomainEvent) { events = append(events, event) },
	}, now)
	if err != nil {
		t.Fatalf("archive: %v", err)
	}
	if len(archived) != 1 || archived[0].Kid != kidOld {
		t.Fatalf("archived = %+v, want only %q", archived, kidOld)
	}

	if kids := jwksKids(t, s, realm); slices.Contains(kids, kidOld) || !slices.Contains(kids, current.Kid) {
		t.Fatalf("jwks kids = %v, want %q gone and the current %q kept", kids, kidOld, current.Kid)
	}
	if len(events) != 1 {
		t.Fatalf("events = %+v, want exactly one SigningKeyArchived", events)
	}
	event, ok := events[0].(*signingdomain.SigningKeyArchived)
	if !ok {
		t.Fatalf("event = %T, want *SigningKeyArchived", events[0])
	}
	wantExpiry := retiredAt.Add(grace)
	if event.Kid != kidOld || event.RetiredAt == nil || !event.RetiredAt.Equal(retiredAt) ||
		event.ExpiresAt == nil || !event.ExpiresAt.Equal(wantExpiry) || !event.DisposedAt.Equal(now) {
		t.Fatalf("event = %+v, want kid %q retiredAt %v expiresAt %v disposedAt %v", event, kidOld, retiredAt, wantExpiry, now)
	}
}

//spec:covers EX-SIGNINGKEYS-004-01: 両テナントが鍵を持つ状態で default テナントの管理者がローテートすると、default の /jwks は自テナントの kid だけを載せ、acme の kid を載せない。acme の /jwks も同様に default の kid を載せない。
func TestTenantJwksCarryOnlyTheirOwnKids(t *testing.T) {
	s := newExampleStack(t)
	otherKid := activeKid(t, s, stack.OtherRealm, signingdomain.KeyUsageSigning)
	_, k1, k2 := rotatedTenant(t, s)

	ownKids := jwksKids(t, s, realm)
	if !slices.Contains(ownKids, k1) || !slices.Contains(ownKids, k2) || slices.Contains(ownKids, otherKid) {
		t.Fatalf("%s jwks = %v, want %q and %q without %s's %q", realm, ownKids, k1, k2, stack.OtherRealm, otherKid)
	}
	otherKids := jwksKids(t, s, stack.OtherRealm)
	if !slices.Contains(otherKids, otherKid) || slices.Contains(otherKids, k1) || slices.Contains(otherKids, k2) {
		t.Fatalf("%s jwks = %v, want only its own %q", stack.OtherRealm, otherKids, otherKid)
	}
}

//spec:covers EX-SIGNINGKEYS-010-01: ローテーション後に管理者が検証用鍵 K1 を無効化すると、K1 は /jwks から消え、K2 は現在の署名鍵のまま /jwks に残る。
func TestAdministratorDisablesTheVerifyingKey(t *testing.T) {
	s := newExampleStack(t)
	admin, k1, k2 := rotatedTenant(t, s)

	disabled := admin.post(t, adminKeysPath+"/"+k1+"/disable", "")
	if disabled.Code != http.StatusOK {
		t.Fatalf("disable K1 status=%d body=%s, want 200", disabled.Code, disabled.Body.String())
	}
	if kids := jwksKids(t, s, realm); slices.Contains(kids, k1) || !slices.Contains(kids, k2) {
		t.Fatalf("jwks kids = %v, want %q gone and %q kept", kids, k1, k2)
	}
	if current := activeKid(t, s, realm, signingdomain.KeyUsageSigning); current != k2 {
		t.Fatalf("active kid = %q, want %q to stay current", current, k2)
	}
}

//spec:covers EX-SIGNINGKEYS-010-02: K1 を無効化した後に現在の署名鍵 K2 の無効化を求めると invalid_request で拒否され、K2 は現在の署名鍵として /jwks に残る。
func TestAdministratorCannotDisableTheCurrentSigningKey(t *testing.T) {
	s := newExampleStack(t)
	admin, k1, k2 := rotatedTenant(t, s)
	if disabled := admin.post(t, adminKeysPath+"/"+k1+"/disable", ""); disabled.Code != http.StatusOK {
		t.Fatalf("disable K1 status=%d body=%s, want 200", disabled.Code, disabled.Body.String())
	}

	refused := admin.post(t, adminKeysPath+"/"+k2+"/disable", "")
	if refused.Code != http.StatusBadRequest || problemType(t, refused) != "urn:idmagic:error:invalid_request" {
		t.Fatalf("disable K2 status=%d body=%s, want 400 invalid_request", refused.Code, refused.Body.String())
	}
	if current := activeKid(t, s, realm, signingdomain.KeyUsageSigning); current != k2 {
		t.Fatalf("active kid = %q, want %q to stay current", current, k2)
	}
	if kids := jwksKids(t, s, realm); !slices.Contains(kids, k2) {
		t.Fatalf("jwks kids = %v, want %q kept", kids, k2)
	}
}

//spec:covers EX-SIGNINGKEYS-011-02: signing-keys:read だけの API アクセストークンでローテーションと無効化を求めると、どちらも signing-keys:write を WWW-Authenticate に示す insufficient_scope で拒否され、現在の鍵も /jwks の K1 も変わらず、SigningKeyRotated も出ない。
func TestReadOnlyApiTokenCannotRotateOrDisable(t *testing.T) {
	s := newExampleStack(t)
	_, k1, k2 := rotatedTenant(t, s)
	before := len(s.Events.Types())
	token, _ := s.IssueApiToken(t, realm, apitokendomain.ScopeSigningKeysRead)

	for _, path := range []string{rotatePath, adminKeysPath + "/" + k1 + "/disable"} {
		refused := bearerPost(t, s, path, token)
		if refused.Code != http.StatusForbidden || problemType(t, refused) != "urn:idmagic:error:insufficient_scope" {
			t.Fatalf("%s status=%d body=%s, want 403 insufficient_scope", path, refused.Code, refused.Body.String())
		}
		want := `Bearer error="insufficient_scope", scope="signing-keys:write"`
		if got := refused.Header().Get("WWW-Authenticate"); !strings.HasPrefix(got, want) {
			t.Fatalf("%s WWW-Authenticate = %q, want it to start with %q", path, got, want)
		}
	}
	if current := activeKid(t, s, realm, signingdomain.KeyUsageSigning); current != k2 {
		t.Fatalf("active kid = %q, want %q unchanged", current, k2)
	}
	if kids := jwksKids(t, s, realm); !slices.Contains(kids, k1) {
		t.Fatalf("jwks kids = %v, want %q kept", kids, k1)
	}
	if after := s.Events.Types()[before:]; slices.Contains(after, "SigningKeyRotated") {
		t.Fatalf("events after the refusals = %v, want no SigningKeyRotated", after)
	}
}

// usage に XmlFederationSigning を指定したローテーションは XML フェデレーション鍵だけを入れ替え、
// JWT の署名鍵には触れない。EX-SIGNINGKEYS-006-01 の入口であり、具体例の観測そのものは
// SAML のメタデータと Assertion を読む backend/saml/handlers_http が持つ。
func TestRotationWithXmlFederationUsageRotatesOnlyTheXmlKey(t *testing.T) {
	s := newExampleStack(t)
	jwtKid := activeKid(t, s, realm, signingdomain.KeyUsageSigning)
	xmlKid := activeKid(t, s, realm, signingdomain.KeyUsageXMLFederationSigning)

	rotation := decodeRotation(t, signedInAdmin(t, s).post(t, rotatePath, `{"usage":"XmlFederationSigning"}`))

	if rotation.Previous == nil || rotation.Previous.Kid != xmlKid {
		t.Fatalf("previous = %+v, want the XML key %q", rotation.Previous, xmlKid)
	}
	if current := activeKid(t, s, realm, signingdomain.KeyUsageXMLFederationSigning); current != rotation.Next.Kid || current == xmlKid {
		t.Fatalf("active XML kid = %q, want the rotated %q", current, rotation.Next.Kid)
	}
	if current := activeKid(t, s, realm, signingdomain.KeyUsageSigning); current != jwtKid {
		t.Fatalf("active JWT kid = %q, want %q untouched", current, jwtKid)
	}
}

// KeyUsage のどれでもない usage は invalid_request で拒否され、どちらの用途の鍵も回らない。
func TestRotationRefusesAnUnknownUsage(t *testing.T) {
	s := newExampleStack(t)
	jwtKid := activeKid(t, s, realm, signingdomain.KeyUsageSigning)
	xmlKid := activeKid(t, s, realm, signingdomain.KeyUsageXMLFederationSigning)

	for _, body := range []string{`{"usage":"Encryption"}`, `{"usage":1}`, `not json`} {
		refused := signedInAdmin(t, s).post(t, rotatePath, body)
		if refused.Code != http.StatusBadRequest || problemType(t, refused) != "urn:idmagic:error:invalid_request" {
			t.Fatalf("%s: status=%d body=%s, want 400 invalid_request", body, refused.Code, refused.Body.String())
		}
	}
	if current := activeKid(t, s, realm, signingdomain.KeyUsageSigning); current != jwtKid {
		t.Fatalf("active JWT kid = %q, want %q untouched", current, jwtKid)
	}
	if current := activeKid(t, s, realm, signingdomain.KeyUsageXMLFederationSigning); current != xmlKid {
		t.Fatalf("active XML kid = %q, want %q untouched", current, xmlKid)
	}
}
