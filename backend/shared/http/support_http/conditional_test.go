package support_http

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/shared/kernel"
	"github.com/labstack/echo/v5"
)

//spec:covers REQ-SYSTEM-037: If-Match は強い比較、候補の OR と存在条件を使い、不正な構文を拒否する。
func TestIfMatchConditions(t *testing.T) {
	for _, tc := range []struct {
		name, header, tag          string
		exists, malformed, matches bool
	}{
		{"strong", `"one"`, `"one"`, true, false, true},
		{"stale", `"one"`, `"two"`, true, false, false},
		{"weak", `W/"one"`, `"one"`, true, false, false},
		{"list", `W/"one", "two"`, `"two"`, true, false, true},
		{"quoted comma", `"one,two"`, `"one,two"`, true, false, true},
		{"empty members", `, "one", ,`, `"one"`, true, false, true},
		{"wildcard", `*`, `"one"`, true, false, true},
		{"missing", `*`, `"one"`, false, false, false},
		{"tag without resource", `"one"`, `"one"`, false, false, false},
		{"unquoted", `one`, `"one"`, true, true, false},
		{"mixed wildcard", `*, "one"`, `"one"`, true, true, false},
		{"unterminated", `"one`, `"one"`, true, true, false},
		{"space in tag", `"one two"`, `"one"`, true, true, false},
		{"empty", ``, `"one"`, true, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPatch, "/", http.NoBody)
			req.Header.Set("If-Match", tc.header)
			condition, err := ReadIfMatch(echo.New().NewContext(req, httptest.NewRecorder()))
			if tc.malformed {
				if !errors.Is(err, ErrInvalidIfMatch) {
					t.Fatalf("error=%v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			err = condition.Match(tc.tag, tc.exists)
			if tc.matches && err != nil || !tc.matches && !errors.Is(err, kernel.ErrPreconditionFailed) {
				t.Fatalf("match error=%v", err)
			}
		})
	}
	req := httptest.NewRequest(http.MethodPatch, "/", http.NoBody)
	condition, err := ReadIfMatch(echo.New().NewContext(req, httptest.NewRecorder()))
	if err != nil || condition.Present() || condition.Match("", false) != nil {
		t.Fatalf("omitted condition=%+v error=%v", condition, err)
	}
	req.Header.Add("If-Match", `"old"`)
	req.Header.Add("If-Match", `"new"`)
	condition, err = ReadIfMatch(echo.New().NewContext(req, httptest.NewRecorder()))
	if err != nil || condition.Match(`"new"`, true) != nil {
		t.Fatalf("multiple headers: %v", err)
	}
}

//spec:covers REQ-SYSTEM-037: GET/PATCH のタグは実際の JSON バイトを識別し、変換する PUT はタグを返さない。
func TestETagJSONIdentifiesResponseBytes(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPatch, http.MethodPut} {
		rec := httptest.NewRecorder()
		c := echo.New().NewContext(httptest.NewRequest(method, "/", http.NoBody), rec)
		if err := ETagJSON(c, http.StatusOK, map[string]any{"name": "sample", "items": []string{}}); err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(rec.Body.Bytes())
		expected := `"` + hex.EncodeToString(hash[:]) + `"`
		if method == http.MethodPut {
			expected = ""
		}
		if rec.Header().Get("ETag") != expected {
			t.Fatalf("%s tag=%s expected=%s", method, rec.Header().Get("ETag"), expected)
		}
	}
}

func FuzzIfMatch(f *testing.F) {
	for _, seed := range []string{`"tag"`, `W/"tag", "other"`, `"a,b"`, `*`, `,`, `"unterminated`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		tags, wildcard, err := parseIfMatch(raw)
		if err != nil {
			return
		}
		condition := MatchCondition{tags: tags, wildcard: wildcard, present: true}
		if wildcard {
			if condition.Match("", true) != nil {
				t.Fatal("wildcard refused existing resource")
			}
			return
		}
		for _, tag := range tags {
			if strings.HasPrefix(tag, "W/") {
				continue
			}
			if condition.Match(tag, true) != nil {
				t.Fatalf("parsed strong tag did not match: %q", tag)
			}
		}
		if condition.Match("", false) == nil {
			t.Fatal("matched missing resource")
		}
	})
}
