package server_http_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/http/testing_stack"
)

// 正式なルーターが、Directory Connector の管理 API と Connector API を path style の正規位置に
// 載せ、Connector の JWT の jti を組み立て地点の保存先で一度だけ受け付けることを確かめる。
//
//spec:covers REQ-SOURCING-008 REQ-SOURCING-009 REQ-SOURCING-010
//verify:operation CreateDirectoryConnector: 正規位置の管理 API が 201 で登録トークンを返す。
//verify:operation EnrollDirectoryConnector: 正規位置の Connector API が登録の応答で aud を返す。
//verify:operation ReportDirectoryConnectorHeartbeat: 同じ jti の二度目の提示を 401 で拒否する。
func TestDirectoryConnectorThroughTheRouter(t *testing.T) {
	s := testing_stack.New(t, testing_stack.WithAuthorizationCodeFlow(), testing_stack.WithTokenIssuance(), testing_stack.WithSCIM())
	session := s.AdminSessionCookie(t, "default")
	send := func(path string, body any, header http.Header, cookies ...*http.Cookie) *httptest.ResponseRecorder {
		t.Helper()
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest(http.MethodPost, testing_stack.Issuer+"/realms/default"+path, bytes.NewReader(payload))
		request.Header.Set("Content-Type", "application/json")
		maps.Copy(request.Header, header)
		for _, cookie := range cookies {
			request.AddCookie(cookie)
		}
		recorder := httptest.NewRecorder()
		s.Echo.ServeHTTP(recorder, request)
		return recorder
	}

	created := send("/api/admin/v1/directory-connectors", map[string]any{"display_name": "東京の AD"},
		http.Header{"Origin": {testing_stack.Issuer}, support.CSRFHeader: {"csrf"}},
		session, &http.Cookie{Name: support.CSRFCookie, Value: "csrf"})
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", created.Code, created.Body.String())
	}
	var grant struct {
		EnrollmentToken string `json:"enrollment_token"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &grant); err != nil {
		t.Fatal(err)
	}

	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	point, err := private.PublicKey.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	enrolled := send("/directory-connector/v1/enrollment", map[string]any{
		"enrollment_token": grant.EnrollmentToken,
		"jwk": map[string]any{
			"kty": "EC", "crv": "P-256",
			"x": base64.RawURLEncoding.EncodeToString(point[1:33]), "y": base64.RawURLEncoding.EncodeToString(point[33:65]),
		},
	}, nil)
	if enrolled.Code != http.StatusOK {
		t.Fatalf("enroll status = %d body=%s", enrolled.Code, enrolled.Body.String())
	}
	var enrollment struct {
		ConnectorID string `json:"connector_id"`
		Audience    string `json:"audience"`
	}
	if err := json.Unmarshal(enrolled.Body.Bytes(), &enrollment); err != nil {
		t.Fatal(err)
	}
	if want := testing_stack.Issuer + "/realms/default/directory-connector/v1"; enrollment.Audience != want {
		t.Fatalf("audience = %q, want %q", enrollment.Audience, want)
	}

	encode := func(value map[string]any) string {
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	signingInput := encode(map[string]any{"alg": "ES256", "typ": "JWT"}) + "." + encode(map[string]any{
		"iss": enrollment.ConnectorID, "sub": enrollment.ConnectorID, "aud": enrollment.Audience,
		"exp": time.Now().Add(time.Minute).Unix(), "jti": uuid.NewString(),
	})
	digest := sha256.Sum256([]byte(signingInput))
	r, sig, err := ecdsa.Sign(rand.Reader, private, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	signature := make([]byte, 64)
	r.FillBytes(signature[:32])
	sig.FillBytes(signature[32:])
	assertion := signingInput + "." + base64.RawURLEncoding.EncodeToString(signature)

	bearer := http.Header{"Authorization": {"Bearer " + assertion}}
	if rec := send("/directory-connector/v1/heartbeat", nil, bearer); rec.Code != http.StatusNoContent {
		t.Fatalf("heartbeat status = %d body=%s", rec.Code, rec.Body.String())
	}
	if rec := send("/directory-connector/v1/heartbeat", nil, bearer); rec.Code != http.StatusUnauthorized {
		t.Fatalf("replayed heartbeat status = %d body=%s", rec.Code, rec.Body.String())
	}
}
