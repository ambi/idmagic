package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
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
