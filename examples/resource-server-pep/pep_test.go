package pep

import (
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The PEP is deployed separately from IdMagic, so only the public contract may connect them.
func TestImportsOnlyTheStandardLibrary(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "pep.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, spec := range file.Imports {
		if strings.Contains(spec.Path.Value, ".") {
			t.Errorf("non-standard import %s", spec.Path.Value)
		}
	}
}

// Every outcome other than decision=true must leave the wrapped handler uncalled.
func TestProtectRunsTheHandlerOnlyOnAnExplicitPermit(t *testing.T) {
	cases := []struct {
		name          string
		status        int
		body          string
		authorization string
		want          int
		effects       int
	}{
		{"permit", http.StatusOK, `{"decision":true}`, "Bearer t", http.StatusOK, 1},
		{"deny", http.StatusOK, `{"decision":false,"reasons":["token_rejected"]}`, "Bearer t", http.StatusForbidden, 0},
		{"missing decision", http.StatusOK, `{}`, "Bearer t", http.StatusForbidden, 0},
		{"malformed body", http.StatusOK, `{`, "Bearer t", http.StatusServiceUnavailable, 0},
		{"unavailable", http.StatusServiceUnavailable, `{"decision":true}`, "Bearer t", http.StatusServiceUnavailable, 0},
		{"unbound", http.StatusForbidden, `{"decision":true}`, "Bearer t", http.StatusServiceUnavailable, 0},
		{"no token", http.StatusOK, `{"decision":true}`, "", http.StatusForbidden, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			effects := 0
			handler := Enforcer{
				EvaluateURL: "http://evaluator.test/resource-access/evaluate",
				HTTPClient: &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
					recorder := httptest.NewRecorder()
					recorder.WriteHeader(tc.status)
					_, _ = recorder.WriteString(tc.body)
					return recorder.Result(), nil
				})},
			}.Protect(func(*http.Request) Operation { return Operation{} }, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				effects++
			}))
			request := httptest.NewRequest(http.MethodPost, "http://rs.test/tool", http.NoBody)
			if tc.authorization != "" {
				request.Header.Set("Authorization", tc.authorization)
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != tc.want || effects != tc.effects {
				t.Fatalf("status=%d effects=%d, want %d %d", recorder.Code, effects, tc.want, tc.effects)
			}
		})
	}
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
