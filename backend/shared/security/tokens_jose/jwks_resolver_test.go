package tokens_jose

import (
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// roundTripFunc lets a test stand in for the network without a real listener:
// fetch()'s own SSRF guard (safeIPs) still runs against the URL's host, so
// tests route it through a documentation-range IP literal (RFC 5737) that
// IsPublicIP accepts without any DNS lookup, then intercept the HTTP call
// here instead of dialing it.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func newStubResolver(rt roundTripFunc) *JWKResolver {
	return &JWKResolver{
		cache:  map[string]cachedJWKS{},
		client: &http.Client{Transport: rt},
	}
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestValidateJWKSURI(t *testing.T) {
	cases := []struct {
		name    string
		uri     string
		wantErr bool
	}{
		{"valid https", "https://idp.example/jwks", false},
		{"rejects http", "http://idp.example/jwks", true},
		{"rejects userinfo", "https://user@idp.example/jwks", true},
		{"rejects fragment", "https://idp.example/jwks#frag", true},
		{"rejects empty host", "https:///jwks", true},
		{"rejects malformed uri", "https://[::1", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateJWKSURI(tc.uri)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestInlineJWKs(t *testing.T) {
	t.Run("accepts []any keys", func(t *testing.T) {
		keys, err := InlineJWKs(map[string]any{"keys": []any{map[string]any{"kid": "a"}}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(keys) != 1 || keys[0]["kid"] != "a" {
			t.Fatalf("unexpected keys: %+v", keys)
		}
	})
	t.Run("accepts []map[string]any keys", func(t *testing.T) {
		keys, err := InlineJWKs(map[string]any{"keys": []map[string]any{{"kid": "b"}}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(keys) != 1 || keys[0]["kid"] != "b" {
			t.Fatalf("unexpected keys: %+v", keys)
		}
	})
	t.Run("rejects missing keys field", func(t *testing.T) {
		if _, err := InlineJWKs(map[string]any{}); err == nil {
			t.Fatal("expected error for missing keys")
		}
	})
	t.Run("rejects empty keys", func(t *testing.T) {
		if _, err := InlineJWKs(map[string]any{"keys": []any{}}); err == nil {
			t.Fatal("expected error for empty keys")
		}
	})
	t.Run("rejects non-object key entries", func(t *testing.T) {
		if _, err := InlineJWKs(map[string]any{"keys": []any{"not-an-object"}}); err == nil {
			t.Fatal("expected error for non-object key")
		}
	})
}

func TestJWKResolverResolve(t *testing.T) {
	t.Run("returns inline keys without a network call", func(t *testing.T) {
		r := newStubResolver(func(*http.Request) (*http.Response, error) {
			t.Fatal("unexpected network call")
			return nil, errors.New("unreachable")
		})
		client := JWKSource{JWKS: map[string]any{"keys": []any{map[string]any{"kid": "inline"}}}}
		keys, err := r.Resolve(context.Background(), client)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(keys) != 1 || keys[0]["kid"] != "inline" {
			t.Fatalf("unexpected keys: %+v", keys)
		}
	})

	t.Run("errors when neither inline jwks nor jwks_uri is set", func(t *testing.T) {
		r := newStubResolver(nil)
		client := JWKSource{}
		if _, err := r.Resolve(context.Background(), client); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("errors when jwks_uri is empty string", func(t *testing.T) {
		r := newStubResolver(nil)
		empty := ""
		client := JWKSource{JwksURI: &empty}
		if _, err := r.Resolve(context.Background(), client); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("falls back to jwks_uri fetch when inline jwks is absent", func(t *testing.T) {
		uri := "https://192.0.2.1/jwks"
		r := newStubResolver(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != uri {
				t.Fatalf("unexpected request URL: %s", req.URL.String())
			}
			return jsonResponse(http.StatusOK, `{"keys":[{"kid":"remote"}]}`), nil
		})
		client := JWKSource{JwksURI: &uri}
		keys, err := r.Resolve(context.Background(), client)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(keys) != 1 || keys[0]["kid"] != "remote" {
			t.Fatalf("unexpected keys: %+v", keys)
		}
	})
}

func TestJWKResolverResolveJWKSSource(t *testing.T) {
	t.Run("returns inline keys without a network call", func(t *testing.T) {
		r := newStubResolver(func(*http.Request) (*http.Response, error) {
			t.Fatal("unexpected network call")
			return nil, errors.New("unreachable")
		})
		inline := map[string]any{"keys": []any{map[string]any{"kid": "inline"}}}
		keys, err := r.ResolveJWKSSource(context.Background(), nil, inline)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(keys) != 1 {
			t.Fatalf("unexpected keys: %+v", keys)
		}
	})

	t.Run("falls back to jwksURI when inline jwks is invalid", func(t *testing.T) {
		uri := "https://192.0.2.1/jwks"
		r := newStubResolver(func(*http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{"keys":[{"kid":"remote"}]}`), nil
		})
		keys, err := r.ResolveJWKSSource(context.Background(), &uri, map[string]any{"not-keys": true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(keys) != 1 || keys[0]["kid"] != "remote" {
			t.Fatalf("unexpected keys: %+v", keys)
		}
	})

	t.Run("errors when neither source is configured", func(t *testing.T) {
		r := newStubResolver(nil)
		if _, err := r.ResolveJWKSSource(context.Background(), nil, nil); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestJWKResolverFetch(t *testing.T) {
	const uri = "https://192.0.2.1/jwks"

	t.Run("rejects a non-https uri before any request", func(t *testing.T) {
		r := newStubResolver(func(*http.Request) (*http.Response, error) {
			t.Fatal("unexpected network call")
			return nil, errors.New("unreachable")
		})
		if _, err := r.fetch(context.Background(), "http://192.0.2.1/jwks"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("rejects a host that resolves to a non-public address", func(t *testing.T) {
		r := newStubResolver(func(*http.Request) (*http.Response, error) {
			t.Fatal("unexpected network call")
			return nil, errors.New("unreachable")
		})
		if _, err := r.fetch(context.Background(), "https://127.0.0.1/jwks"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("propagates a transport error", func(t *testing.T) {
		r := newStubResolver(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("boom")
		})
		if _, err := r.fetch(context.Background(), uri); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("rejects a non-200 status", func(t *testing.T) {
		r := newStubResolver(func(*http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusInternalServerError, ""), nil
		})
		if _, err := r.fetch(context.Background(), uri); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("rejects a body over the size limit", func(t *testing.T) {
		huge := `{"keys":[{"kid":"` + strings.Repeat("a", maxJWKSBytes) + `"}]}`
		r := newStubResolver(func(*http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, huge), nil
		})
		if _, err := r.fetch(context.Background(), uri); err == nil {
			t.Fatal("expected error for oversized body")
		}
	})

	t.Run("rejects malformed json", func(t *testing.T) {
		r := newStubResolver(func(*http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, "not json"), nil
		})
		if _, err := r.fetch(context.Background(), uri); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("rejects a document with no usable keys", func(t *testing.T) {
		r := newStubResolver(func(*http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{}`), nil
		})
		if _, err := r.fetch(context.Background(), uri); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("caches a successful fetch and skips the next request", func(t *testing.T) {
		calls := 0
		r := newStubResolver(func(*http.Request) (*http.Response, error) {
			calls++
			return jsonResponse(http.StatusOK, `{"keys":[{"kid":"cached"}]}`), nil
		})
		keys, err := r.fetch(context.Background(), uri)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(keys) != 1 || keys[0]["kid"] != "cached" {
			t.Fatalf("unexpected keys: %+v", keys)
		}
		if _, err := r.fetch(context.Background(), uri); err != nil {
			t.Fatalf("unexpected error on cached fetch: %v", err)
		}
		if calls != 1 {
			t.Fatalf("expected 1 network call, got %d", calls)
		}
	})

	t.Run("refetches once the cache entry has expired", func(t *testing.T) {
		calls := 0
		r := newStubResolver(func(*http.Request) (*http.Response, error) {
			calls++
			return jsonResponse(http.StatusOK, `{"keys":[{"kid":"fresh"}]}`), nil
		})
		r.cache[uri] = cachedJWKS{
			keys:      []map[string]any{{"kid": "stale"}},
			expiresAt: time.Now().Add(-time.Second),
		}
		keys, err := r.fetch(context.Background(), uri)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(keys) != 1 || keys[0]["kid"] != "fresh" {
			t.Fatalf("unexpected keys: %+v", keys)
		}
		if calls != 1 {
			t.Fatalf("expected 1 network call, got %d", calls)
		}
	})
}

func TestNewJWKResolver(t *testing.T) {
	r := NewJWKResolver()
	if r == nil || r.cache == nil || r.client == nil {
		t.Fatal("NewJWKResolver returned an incomplete resolver")
	}
}

// 取得元が返す文書と時計だけを差し替え、何回目の取得でどの鍵を返したかを観測する。
//
//spec:covers REQ-WORKLOADIDENTITY-012: jwks_uri の SPIFFE バンドルを spiffe_refresh_hint を 30 秒から 5 分に収めた期間だけ再利用し、期間を過ぎたら取得し直す。
func TestJWKResolverResolveSPIFFEBundle(t *testing.T) {
	uri := "https://192.0.2.1/bundle"
	bundle := func(hint int, kid string) string {
		return `{"spiffe_sequence":1,"spiffe_refresh_hint":` + strconv.Itoa(hint) +
			`,"keys":[{"kid":"x509","use":"x509-svid"},{"kid":"` + kid + `","use":"jwt-svid"}]}`
	}
	cases := []struct {
		name  string
		hint  int
		reuse time.Duration
	}{
		{"短いヒントは 30 秒に引き上げる", 10, 30 * time.Second},
		{"範囲内のヒントはその秒数", 120, 120 * time.Second},
		{"長いヒントは 5 分に縮める", 2419200, 5 * time.Minute},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			r := newStubResolver(func(*http.Request) (*http.Response, error) {
				calls++
				return jsonResponse(http.StatusOK, bundle(tc.hint, "jwt-"+strconv.Itoa(calls))), nil
			})
			now := time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC)
			r.now = func() time.Time { return now }
			resolve := func() []string {
				t.Helper()
				keys, err := r.ResolveSPIFFEBundle(context.Background(), &uri, nil)
				if err != nil {
					t.Fatalf("ResolveSPIFFEBundle: %v", err)
				}
				return keyIDs(keys)
			}

			if got := resolve(); !slices.Equal(got, []string{"jwt-1"}) {
				t.Fatalf("first fetch keys = %v, want [jwt-1]", got)
			}
			now = now.Add(tc.reuse - time.Second)
			if got := resolve(); !slices.Equal(got, []string{"jwt-1"}) || calls != 1 {
				t.Fatalf("keys just before %s = %v after %d fetch(es), want reused [jwt-1]", tc.reuse, got, calls)
			}
			now = now.Add(2 * time.Second)
			if got := resolve(); !slices.Equal(got, []string{"jwt-2"}) || calls != 2 {
				t.Fatalf("keys just after %s = %v after %d fetch(es), want refetched [jwt-2]", tc.reuse, got, calls)
			}
		})
	}

	t.Run("同じ URI の通常の JWKS の取得結果を使わない", func(t *testing.T) {
		r := newStubResolver(func(*http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, bundle(300, "jwt")), nil
		})
		if keys, err := r.ResolveJWKSSource(context.Background(), &uri, nil); err != nil || len(keys) != 2 {
			t.Fatalf("ResolveJWKSSource keys = %v, err = %v; want both keys", keyIDs(keys), err)
		}
		keys, err := r.ResolveSPIFFEBundle(context.Background(), &uri, nil)
		if err != nil || !slices.Equal(keyIDs(keys), []string{"jwt"}) {
			t.Fatalf("ResolveSPIFFEBundle keys = %v, err = %v; want [jwt]", keyIDs(keys), err)
		}
	})

	t.Run("インラインの SPIFFE バンドルは取得元を呼ばない", func(t *testing.T) {
		r := newStubResolver(func(*http.Request) (*http.Response, error) {
			t.Fatal("unexpected network call")
			return nil, errors.New("unreachable")
		})
		inline := map[string]any{"keys": []any{
			map[string]any{"kid": "x509", "use": "x509-svid"}, map[string]any{"kid": "jwt", "use": "jwt-svid"},
		}}
		keys, err := r.ResolveSPIFFEBundle(context.Background(), &uri, inline)
		if err != nil || !slices.Equal(keyIDs(keys), []string{"jwt"}) {
			t.Fatalf("keys = %v, err = %v; want [jwt]", keyIDs(keys), err)
		}
	})

	t.Run("使える鍵のないバンドルは到達できた空の鍵の集合になる", func(t *testing.T) {
		r := newStubResolver(func(*http.Request) (*http.Response, error) {
			return jsonResponse(http.StatusOK, `{"keys":[{"kid":"x509","use":"x509-svid"}]}`), nil
		})
		keys, err := r.ResolveSPIFFEBundle(context.Background(), &uri, nil)
		if err != nil || keys == nil || len(keys) != 0 {
			t.Fatalf("keys = %v, err = %v; want an empty set without error", keys, err)
		}
	})

	t.Run("keys の配列のない応答は取得の失敗にし、再利用しない", func(t *testing.T) {
		calls := 0
		r := newStubResolver(func(*http.Request) (*http.Response, error) {
			calls++
			return jsonResponse(http.StatusOK, `{"spiffe_sequence":1}`), nil
		})
		for range 2 {
			if keys, err := r.ResolveSPIFFEBundle(context.Background(), &uri, nil); err == nil {
				t.Fatalf("keys = %v, want an error", keyIDs(keys))
			}
		}
		if calls != 2 {
			t.Fatalf("fetches = %d, want 2 (a failed fetch is not reused)", calls)
		}
	})

	t.Run("どちらの取得元もなければエラーにする", func(t *testing.T) {
		if _, err := newStubResolver(nil).ResolveSPIFFEBundle(context.Background(), nil, nil); err == nil {
			t.Fatal("expected error")
		}
	})
}
