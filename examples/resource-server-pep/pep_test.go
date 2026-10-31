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
//
//spec:covers REQ-OAUTH2-062: 429 を含む非 200 で業務処理を呼ばず、自動再送もしない
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
		{"rate limited", http.StatusTooManyRequests, `{"decision":true,"retry_after_seconds":30}`, "Bearer t", http.StatusServiceUnavailable, 0},
		{"unbound", http.StatusForbidden, `{"decision":true}`, "Bearer t", http.StatusServiceUnavailable, 0},
		{"no token", http.StatusOK, `{"decision":true}`, "", http.StatusForbidden, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			effects := 0
			calls := 0
			handler := Enforcer{
				EvaluateURL: "http://evaluator.test/resource-access/evaluate",
				HTTPClient: &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
					calls++
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
			wantCalls := 1
			if tc.authorization == "" {
				wantCalls = 0
			}
			if calls != wantCalls {
				t.Fatalf("evaluation calls=%d, want %d", calls, wantCalls)
			}
		})
	}
}

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// The ledger is what makes a retried request safe: IdMagic permits the same key again.
func TestOperationLedgerDistinguishesFailureFromUnknown(t *testing.T) {
	cases := []struct {
		name  string
		first ExecuteResult
		runs  int
	}{
		{name: "success is not repeated", first: ExecuteSucceeded, runs: 1},
		{name: "unknown is not repeated", first: ExecuteUnknown, runs: 1},
		{name: "definite failure may run again", first: ExecuteFailed, runs: 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var ledger OperationLedger
			runs := 0
			effect := func() ExecuteResult {
				runs++
				if runs == 1 {
					return tc.first
				}
				return ExecuteSucceeded
			}
			if got := ledger.Run("k", effect); got != tc.first {
				t.Fatalf("first result = %d", got)
			}
			ledger.Run("k", effect)
			if runs != tc.runs {
				t.Fatalf("runs = %d, want %d", runs, tc.runs)
			}
		})
	}
}
