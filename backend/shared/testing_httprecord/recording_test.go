package testing_httprecord

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
)

// 記録は出どころを持たなければ読み込めない。古くなった記録を区別する手がかりを失わないためである。
func TestDecode_RequiresProvenance(t *testing.T) {
	complete := map[string]string{"counterpart": "x", "counterpart_version": "1", "retrieved_at": "2026-11-26", "method": "m"}
	for missing := range complete {
		t.Run(missing, func(t *testing.T) {
			fields := map[string]any{"exchanges": []any{}}
			for k, v := range complete {
				if k != missing {
					fields[k] = v
				}
			}
			data, _ := json.Marshal(fields)
			if _, err := Decode(data); err == nil {
				t.Errorf("Decode() accepted a recording without %s", missing)
			}
		})
	}
}

// JSON の文字列で持つ本文はその中身を、JSON の値で持つ本文はそのまま送受信する。
func TestBodyBytes(t *testing.T) {
	tests := []struct {
		raw, want string
	}{
		{`"grant_type=client_credentials"`, "grant_type=client_credentials"},
		{`{"error":"invalid_request"}`, `{"error":"invalid_request"}`},
		{``, ``},
	}
	for _, tc := range tests {
		if got := string(BodyBytes(json.RawMessage(tc.raw))); got != tc.want {
			t.Errorf("BodyBytes(%s) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

// 照合は、記録と同じ応答の模倣には何も報告せず、値まで比べるヘッダー、揮発しない値、属性の欠落の食い違いを報告する。
func TestFidelity_ReportsDivergence(t *testing.T) {
	rec := Recording{Exchanges: []Exchange{{
		Name:    "token",
		Request: RecordedRequest{Method: http.MethodPost, Path: "/token", Body: json.RawMessage(`"grant_type=x"`)},
		Response: RecordedResponse{
			Status:  200,
			Headers: map[string]string{"Content-Type": "application/json;charset=UTF-8", "Cache-Control": "no-store"},
			Body:    json.RawMessage(`{"access_token":"a","token_type":"Bearer","expires_in":3600}`),
		},
	}}}
	fidelity := Fidelity{VolatileKeys: map[string]bool{"access_token": true}, Headers: []string{"Content-Type"}, ExactHeaders: []string{"Cache-Control"}}
	twin := func(cacheControl, body string) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", cacheControl)
			_, _ = w.Write([]byte(body))
		})
	}

	if diffs := fidelity.Diverge(twin("no-store", `{"access_token":"other","token_type":"Bearer","expires_in":3600,"extra":1}`), rec); len(diffs) != 0 {
		t.Errorf("Diverge() = %q for a twin that matches the recording", diffs)
	}
	diffs := fidelity.Diverge(twin("no-cache", `{"access_token":1,"token_type":"bearer"}`), rec)
	for _, want := range []string{
		`token: header Cache-Control = "no-cache", recorded "no-store"`,
		"token: body.access_token is number, recorded string",
		"token: body.token_type = bearer, recorded Bearer",
		"token: body.expires_in is missing",
	} {
		if !strings.Contains(strings.Join(diffs, "\n"), want) {
			t.Errorf("Diverge() = %q, want a line %q", diffs, want)
		}
	}
}

// Decode を指定すると、記録と模倣の本文をその読み方で比べる。JSON でない本文（SSE など）を比べるために使う。
func TestFidelity_DecodesBodiesWithTheGivenReader(t *testing.T) {
	rec := Recording{Exchanges: []Exchange{{
		Name:     "stream",
		Request:  RecordedRequest{Method: http.MethodPost, Path: "/mcp"},
		Response: RecordedResponse{Status: 200, Headers: map[string]string{"Content-Type": "text/plain"}, Body: json.RawMessage(`"a=1\nb=2"`)},
	}}}
	var contentTypes []string
	fidelity := Fidelity{Decode: func(contentType string, body []byte) (any, error) {
		contentTypes = append(contentTypes, contentType)
		fields := map[string]any{}
		for line := range strings.SplitSeq(string(body), "\n") {
			key, value, _ := strings.Cut(line, "=")
			fields[key] = value
		}
		return fields, nil
	}}
	twin := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("a=1\nb=3"))
	})

	diffs := fidelity.Diverge(twin, rec)
	if want := "stream: body.b = 3, recorded 2"; strings.Join(diffs, "\n") != want {
		t.Errorf("Diverge() = %q, want %q", diffs, want)
	}
	if want := []string{"text/plain", "text/plain; charset=utf-8"}; !slices.Equal(contentTypes, want) {
		t.Errorf("Decode got content types %q, want %q", contentTypes, want)
	}
}
