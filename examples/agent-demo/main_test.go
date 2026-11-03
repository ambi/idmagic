package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"
)

func newTestDemo(t *testing.T) *demo {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	d := newDemo("http://idmagic.invalid/realms/default", t.TempDir(), key, "", "", "", "")
	upstream := httptest.NewServer(d.docs.handler())
	t.Cleanup(upstream.Close)
	d.upstreamURL = upstream.URL + "/"
	return d
}

// 統制なしの比較が成り立つには、上流が認可なしで全操作を実行し、次の実行へ影響を残さないことが要る。
func TestUnmanagedRunExecutesEveryRequestAndRestoresDocuments(t *testing.T) {
	d := newTestDemo(t)
	for range 2 {
		steps, err := d.run(context.Background(), "unmanaged")
		if err != nil {
			t.Fatal(err)
		}
		if len(steps) != 3 {
			t.Fatalf("steps=%d, want 3", len(steps))
		}
		for _, item := range steps {
			if item.Outcome != "allowed" || !item.Executed {
				t.Fatalf("step %q=%+v, want executed", item.Title, item)
			}
		}
		if !strings.Contains(steps[1].Detail, "給与テーブル") {
			t.Fatalf("secret document was not returned: %q", steps[1].Detail)
		}
	}
}

func TestGovernedScenariosRequireConnection(t *testing.T) {
	d := newTestDemo(t)
	for _, scenario := range []string{"read-public", "read-secret", "delete", "after-stop"} {
		if _, err := d.run(context.Background(), scenario); err == nil {
			t.Fatalf("%s ran before connecting", scenario)
		}
	}
}

func TestPublicKeyIsOnlyThePublicPEM(t *testing.T) {
	text, err := newTestDemo(t).publicPEM()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(text, "-----BEGIN PUBLIC KEY-----") || strings.Contains(text, "PRIVATE") {
		t.Fatalf("public key=%q", text)
	}
}

func TestActionsRejectNonJSONSubmissions(t *testing.T) {
	d := newTestDemo(t)
	request := httptest.NewRequest(http.MethodPost, "/api/run", strings.NewReader("scenario=unmanaged"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	d.routes().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status=%d, want 415", recorder.Code)
	}
	if d.docs.count() != 0 {
		t.Fatal("a form submission reached the MCP server")
	}
}

// 文書の作成・更新は実在の MCP ツールとして本文を変え、空の識別子とタイトルでは何も変えない。
func TestWriteDocumentCreatesUpdatesAndRecreatesBodies(t *testing.T) {
	d := newTestDemo(t)
	ctx := context.Background()
	call := func(tool string, arguments map[string]string) (int, string, bool) {
		t.Helper()
		status, text, executed, err := d.callTool(ctx, d.upstreamURL, "X-Api-Key", "key", tool, arguments)
		if err != nil {
			t.Fatal(err)
		}
		return status, text, executed
	}
	updated := map[string]string{"document_id": publicDocument, "title": "就業規則 2026 年版", "body": "勤務時間は 10:00 から 18:30 です。"}
	if _, text, executed := call("write_document", updated); !executed || !strings.Contains(text, "10:00") {
		t.Fatalf("write text=%q executed=%v", text, executed)
	}
	if _, text, _ := call("read_document", map[string]string{"document_id": publicDocument}); !strings.Contains(text, "10:00") {
		t.Fatalf("read after write=%q, want the updated body", text)
	}
	for _, invalid := range []map[string]string{
		{"document_id": "", "title": "題", "body": "本文"},
		{"document_id": publicDocument, "title": " ", "body": "書き換え"},
	} {
		call("write_document", invalid)
		if _, text, _ := call("read_document", map[string]string{"document_id": publicDocument}); !strings.Contains(text, "10:00") {
			t.Fatalf("invalid write %v changed the body: %q", invalid, text)
		}
	}
	call("delete_document", map[string]string{"document_id": publicDocument})
	if _, text, _ := call("read_document", map[string]string{"document_id": publicDocument}); !strings.Contains(text, "not found") {
		t.Fatalf("read after delete=%q", text)
	}
	if _, text, _ := call("write_document", map[string]string{"document_id": publicDocument, "title": "就業規則", "body": ""}); !strings.Contains(text, "就業規則") {
		t.Fatalf("recreate with an empty body=%q", text)
	}
	d.docs.restore()
	if _, text, _ := call("read_document", map[string]string{"document_id": publicDocument}); !strings.Contains(text, "9:30") {
		t.Fatalf("restore did not return the initial body: %q", text)
	}
}

// カタログの接続では発行スコープを省略し、文書別の接続では閲覧のスコープを明示する。承認の開始も同じ要求を使う。
func TestTokenRequestsOmitScopeOnlyForCatalogConnections(t *testing.T) {
	for _, tc := range []struct {
		catalog bool
		want    []string
	}{{catalog: true, want: nil}, {catalog: false, want: []string{"tools:read"}}} {
		var forms []url.Values
		idmagic := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			forms = append(forms, r.PostForm)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"unauthorized_client"}`))
		}))
		d := newTestDemo(t)
		d.realm = idmagic.URL
		d.clientID, d.gateway, d.catalog = "agent-client", http.NotFoundHandler(), tc.catalog
		d.issueToken(context.Background())
		if _, err := d.startApproval(context.Background()); err == nil {
			t.Fatal("approval started without the IdMagic endpoint")
		}
		idmagic.Close()
		if len(forms) < 2 {
			t.Fatalf("catalog=%v requests=%d, want issuance and approval", tc.catalog, len(forms))
		}
		for _, form := range forms[:2] {
			if form.Get("grant_type") != "client_credentials" || form.Get("resource") != gatewayResource {
				t.Fatalf("catalog=%v form=%v", tc.catalog, form)
			}
			if got := form["scope"]; !slices.Equal(got, tc.want) {
				t.Fatalf("catalog=%v scope=%v, want %v", tc.catalog, got, tc.want)
			}
		}
	}
}
