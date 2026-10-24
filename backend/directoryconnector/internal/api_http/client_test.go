package api_http_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/ambi/idmagic/backend/directoryconnector/internal/api_http"
	"github.com/ambi/idmagic/backend/directoryconnector/internal/domain"
	"github.com/ambi/idmagic/backend/directoryconnector/internal/ports"
)

type request struct {
	path, authorization, contentType string
	body                             map[string]any
}

func server(t *testing.T, status int, response string) (*httptest.Server, *[]request) {
	t.Helper()
	var mu sync.Mutex
	var seen []request
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		mu.Lock()
		seen = append(seen, request{path: r.URL.Path, authorization: r.Header.Get("Authorization"), contentType: r.Header.Get("Content-Type"), body: body})
		mu.Unlock()
		w.WriteHeader(status)
		_, _ = io.WriteString(w, response)
	}))
	t.Cleanup(s.Close)
	return s, &seen
}

//spec:covers REQ-DIRECTORYCONNECTOR-005: 要求ごとに新しい jti と 2 分後の exp を持ち、登録の aud と Connector の ID を持つ ES256 の JWT を署名する。
func TestClientSignsAFreshJWTForEveryRequest(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	s, seen := server(t, http.StatusOK, `{"id":"run-7"}`)
	now := time.Now().Truncate(time.Second)
	enrollment := ports.Enrollment{ConnectorID: "c1", KeyID: "k1", Audience: s.URL + "/realms/acme/directory-connector/v1"}
	client := api_http.NewClient(s.URL+"/realms/acme/", s.Client(), key, enrollment, func() time.Time { return now })

	runID, err := client.StartRun(context.Background(), domain.ModeFull)
	if err != nil || runID != "run-7" {
		t.Fatalf("run=%q err=%v", runID, err)
	}
	items := []domain.Item{{ExternalID: "g1", Enabled: false, PreferredUsername: "alice"}}
	if err := client.SendBatch(context.Background(), "run-7", "run-7-1", items); err != nil {
		t.Fatal(err)
	}
	if err := client.CompleteRun(context.Background(), "run-7", 1); err != nil {
		t.Fatal(err)
	}

	wantPaths := []string{
		"/realms/acme/directory-connector/v1/ingestion-runs",
		"/realms/acme/directory-connector/v1/ingestion-runs/run-7/user-batches",
		"/realms/acme/directory-connector/v1/ingestion-runs/run-7/complete",
	}
	ids := map[string]bool{}
	for i, r := range *seen {
		if r.path != wantPaths[i] || r.contentType != "application/json" {
			t.Fatalf("request[%d] path=%s content-type=%q, want %s with json", i, r.path, r.contentType, wantPaths[i])
		}
		raw := strings.TrimPrefix(r.authorization, "Bearer ")
		claims := jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(raw, &claims, func(*jwt.Token) (any, error) { return &key.PublicKey, nil },
			jwt.WithValidMethods([]string{"ES256"}), jwt.WithAudience(enrollment.Audience), jwt.WithIssuer("c1"), jwt.WithSubject("c1"),
			jwt.WithTimeFunc(func() time.Time { return now }))
		if err != nil {
			t.Fatalf("request %d jwt: %v", i, err)
		}
		if token.Header["kid"] != "k1" || !claims.ExpiresAt.Equal(now.Add(2*time.Minute)) {
			t.Fatalf("request %d kid=%v exp=%v", i, token.Header["kid"], claims.ExpiresAt)
		}
		if ids[claims.ID] || claims.ID == "" {
			t.Fatalf("request %d reused jti %q", i, claims.ID)
		}
		ids[claims.ID] = true
	}
	if body := (*seen)[0].body; body["mode"] != "full" {
		t.Fatalf("start body = %v", body)
	}
	batch := (*seen)[1].body
	item := batch["items"].([]any)[0].(map[string]any)
	if batch["batch_id"] != "run-7-1" || item["enabled"] != false || item["external_id"] != "g1" {
		t.Fatalf("batch body = %v", batch)
	}
	if (*seen)[2].body["item_count"] != float64(1) {
		t.Fatalf("complete body = %v", (*seen)[2].body)
	}
}

//spec:covers REQ-DIRECTORYCONNECTOR-005: 2xx 以外の応答を、状態コードと Problem Details のエラーの種類として返す。
func TestClientReportsTheStatusAndErrorType(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	s, _ := server(t, http.StatusConflict, `{"type":"urn:idmagic:error:ingestion_run_closed","status":409}`)
	client := api_http.NewClient(s.URL, s.Client(), key, ports.Enrollment{ConnectorID: "c1"}, time.Now)
	err := client.SendBatch(context.Background(), "r", "r-1", []domain.Item{{ExternalID: "g", PreferredUsername: "u"}})
	var api *ports.APIError
	if !errors.As(err, &api) || api.Status != 409 || api.Type != "ingestion_run_closed" {
		t.Fatalf("error = %v", err)
	}
}

//spec:covers REQ-DIRECTORYCONNECTOR-001: 登録は JWT を付けずに登録トークンと JWK を送り、応答の Connector の ID、鍵の ID、aud を返す。
func TestEnrollmentClientSendsTheTokenAndKey(t *testing.T) {
	s, seen := server(t, http.StatusOK, `{"connector_id":"c1","key_id":"k1","audience":"https://idp.test/realms/acme/directory-connector/v1"}`)
	got, err := api_http.NewEnrollmentClient(s.URL+"/realms/acme", s.Client()).Enroll(context.Background(), "idmagic_dce_t", map[string]any{"kty": "EC"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ConnectorID != "c1" || got.KeyID != "k1" || !strings.HasSuffix(got.Audience, "/directory-connector/v1") {
		t.Fatalf("enrollment = %+v", got)
	}
	r := (*seen)[0]
	if r.path != "/realms/acme/directory-connector/v1/enrollment" || r.authorization != "" || r.body["enrollment_token"] != "idmagic_dce_t" {
		t.Fatalf("request = %+v", r)
	}
}
