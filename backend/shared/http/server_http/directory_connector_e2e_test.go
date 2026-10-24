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
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// directoryConnectorClient は、default の realm に登録した Connector として Connector API を呼ぶ。
type directoryConnectorClient struct {
	t        *testing.T
	s        *testing_stack.Stack
	private  *ecdsa.PrivateKey
	id       string
	audience string
}

// postRealmJSON は default の realm の path へ JSON を POST する。
func postRealmJSON(t *testing.T, s *testing_stack.Stack, path string, body any, header http.Header, cookies ...*http.Cookie) *httptest.ResponseRecorder {
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

// adminPost は、管理者のセッションと CSRF のヘッダーで default の realm の管理 API へ POST する。
func adminPost(t *testing.T, s *testing_stack.Stack, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	return postRealmJSON(t, s, path, body, http.Header{"Origin": {testing_stack.Issuer}, support.CSRFHeader: {"csrf"}},
		s.AdminSessionCookie(t, "default"), &http.Cookie{Name: support.CSRFCookie, Value: "csrf"})
}

// enrollDirectoryConnector は、管理 API で Connector を作り、手元の鍵対の公開鍵で登録する。
func enrollDirectoryConnector(t *testing.T, s *testing_stack.Stack) *directoryConnectorClient {
	t.Helper()
	created := adminPost(t, s, "/api/admin/v1/directory-connectors", map[string]any{"display_name": "東京の AD"})
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
	enrolled := postRealmJSON(t, s, "/directory-connector/v1/enrollment", map[string]any{
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
	return &directoryConnectorClient{t: t, s: s, private: private, id: enrollment.ConnectorID, audience: enrollment.Audience}
}

// sign は、新しい jti の Connector の JWT を載せた Authorization ヘッダーを返す。
func (c *directoryConnectorClient) sign() http.Header {
	c.t.Helper()
	encode := func(value map[string]any) string {
		raw, err := json.Marshal(value)
		if err != nil {
			c.t.Fatal(err)
		}
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	input := encode(map[string]any{"alg": "ES256", "typ": "JWT"}) + "." + encode(map[string]any{
		"iss": c.id, "sub": c.id, "aud": c.audience,
		"exp": time.Now().Add(time.Minute).Unix(), "jti": uuid.NewString(),
	})
	digest := sha256.Sum256([]byte(input))
	r, sig, err := ecdsa.Sign(rand.Reader, c.private, digest[:])
	if err != nil {
		c.t.Fatal(err)
	}
	signature := make([]byte, 64)
	r.FillBytes(signature[:32])
	sig.FillBytes(signature[32:])
	return http.Header{"Authorization": {"Bearer " + input + "." + base64.RawURLEncoding.EncodeToString(signature)}}
}

// send は、新しい JWT で Connector API を呼ぶ。
func (c *directoryConnectorClient) send(path string, body any) *httptest.ResponseRecorder {
	c.t.Helper()
	return postRealmJSON(c.t, c.s, path, body, c.sign())
}

// 正式なルーターが、Directory Connector の管理 API と Connector API を path style の正規位置に
// 載せ、Connector の JWT の jti を組み立て地点の保存先で一度だけ受け付けることを確かめる。
//
//spec:covers REQ-SOURCING-008 REQ-SOURCING-009 REQ-SOURCING-010 REQ-SOURCING-016
//verify:operation CreateDirectoryConnector: 正規位置の管理 API が 201 で登録トークンを返す。
//verify:operation EnrollDirectoryConnector: 正規位置の Connector API が登録の応答で aud を返す。
//verify:operation ReportDirectoryConnectorHeartbeat: 同じ jti の二度目の提示を 401 で拒否する。
//verify:operation StartDirectoryIngestionRun: 正規位置の Connector API が 201 で実行を返す。
//verify:operation SendDirectoryUserBatch: 正規位置の Connector API が項目から User を作る。
//verify:operation CompleteDirectoryIngestionRun: 正規位置の Connector API が 200 で実行を完了する。
func TestDirectoryConnectorThroughTheRouter(t *testing.T) {
	s := testing_stack.New(t, testing_stack.WithAuthorizationCodeFlow(), testing_stack.WithTokenIssuance(), testing_stack.WithSCIM())
	connector := enrollDirectoryConnector(t, s)
	if want := testing_stack.Issuer + "/realms/default/directory-connector/v1"; connector.audience != want {
		t.Fatalf("audience = %q, want %q", connector.audience, want)
	}

	bearer := connector.sign()
	if rec := postRealmJSON(t, s, "/directory-connector/v1/heartbeat", nil, bearer); rec.Code != http.StatusNoContent {
		t.Fatalf("heartbeat status = %d body=%s", rec.Code, rec.Body.String())
	}
	if rec := postRealmJSON(t, s, "/directory-connector/v1/heartbeat", nil, bearer); rec.Code != http.StatusUnauthorized {
		t.Fatalf("replayed heartbeat status = %d body=%s", rec.Code, rec.Body.String())
	}

	started := connector.send("/directory-connector/v1/ingestion-runs", map[string]any{"mode": "full"})
	if started.Code != http.StatusCreated {
		t.Fatalf("start status = %d body=%s", started.Code, started.Body.String())
	}
	var run struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(started.Body.Bytes(), &run); err != nil {
		t.Fatal(err)
	}
	batch := connector.send("/directory-connector/v1/ingestion-runs/"+run.ID+"/user-batches", map[string]any{
		"batch_id": "b1", "items": []any{map[string]any{"external_id": "guid", "preferred_username": "from-ad"}},
	})
	if batch.Code != http.StatusOK || !bytes.Contains(batch.Body.Bytes(), []byte(`"outcome":"created"`)) {
		t.Fatalf("batch status = %d body=%s", batch.Code, batch.Body.String())
	}
	if user, err := s.Users.FindByUsername(t.Context(), tenancydomain.DefaultTenantID, "from-ad"); err != nil || user == nil {
		t.Fatalf("ingested user = %v, %v", user, err)
	}
	if rec := connector.send("/directory-connector/v1/ingestion-runs/"+run.ID+"/complete", map[string]any{"item_count": 1}); rec.Code != http.StatusOK {
		t.Fatalf("complete status = %d body=%s", rec.Code, rec.Body.String())
	}
}
