package handlers_http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// RFC 7591 §3.1 の登録の要求の例を /register へ流す。期待値は RFC 7591 の文面から書く。
// §2 は理解しないメタデータを無視すること、grant_types と response_types の省略を authorization_code と code とみなすことを、
// §3.2.1 は成功に 201 と client_id と登録したメタデータを返すことを求める。
//
// 応答の client_name と client_secret_expires_at（§3.2.1）は、今の実装が返さないので wi-97659 が検査を足す。
//
//spec:covers RFC7591-REGISTER, EX-OAUTH2-016-01: RFC 7591 の例の要求を、理解しないメタデータと言語の付いた名前を無視して受け付け、省略した grant_types と response_types を既定で補って登録すること,
//spec:covers RFC7591-REDIRECT-URI: RFC 7591 の例の二つの redirect_uris をそのまま登録すること
func TestRegisterClient_AcceptsTheRFC7591Examples(t *testing.T) {
	data, err := os.ReadFile("testdata/rfc7591-examples.json")
	if err != nil {
		t.Fatal(err)
	}
	rec, err := testing_httprecord.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"open-registration", "registration-with-initial-access-token-and-jwks"} {
		t.Run(name, func(t *testing.T) {
			ex := rec.Exchange(t, name)
			var sent map[string]any
			if err := json.Unmarshal(ex.Request.Body, &sent); err != nil {
				t.Fatal(err)
			}
			fix := newRegisterServer()
			req := httptest.NewRequest(http.MethodPost, "/realms/default"+ex.Request.Path, bytes.NewReader(ex.Request.Body))
			for key, value := range ex.Request.Headers {
				req.Header.Set(key, value)
			}
			w := httptest.NewRecorder()
			fix.e.ServeHTTP(w, req)
			if w.Code != http.StatusCreated {
				t.Fatalf("status %d, want 201; body=%s", w.Code, w.Body.String())
			}
			var got map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			clientID, _ := got["client_id"].(string)
			if secret, _ := got["client_secret"].(string); clientID == "" || secret == "" {
				t.Fatalf("response %v, want a client_id and, for client_secret_basic, a client_secret", got)
			}
			for key, want := range map[string]any{
				"redirect_uris": sent["redirect_uris"], "token_endpoint_auth_method": "client_secret_basic",
				"grant_types": []any{"authorization_code"}, "response_types": []any{"code"},
			} {
				if fmt.Sprint(got[key]) != fmt.Sprint(want) {
					t.Errorf("%s = %v, want %v", key, got[key], want)
				}
			}
			for _, key := range []string{"jwks_uri", "jwks"} {
				if want, ok := sent[key]; ok && fmt.Sprint(got[key]) != fmt.Sprint(want) {
					t.Errorf("%s = %v, want the registered %v", key, got[key], want)
				}
			}
			for _, ignored := range []string{"example_extension_parameter", "client_name#ja-Jpan-JP"} {
				if _, echoed := got[ignored]; echoed {
					t.Errorf("%s was echoed, but the server does not understand it", ignored)
				}
			}
			stored, err := fix.clientRepo.FindByID(context.Background(), tenancydomain.DefaultTenantID, clientID)
			if err != nil || stored == nil || stored.ClientName == nil || *stored.ClientName != sent["client_name"] {
				t.Fatalf("stored client = %+v (%v), want client_name %v", stored, err, sent["client_name"])
			}
		})
	}
}
