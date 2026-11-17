package domain

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

const stateResource = "https://mcp.example/"

func sealedState(t *testing.T, key []byte, state RequestState) string {
	t.Helper()
	sealed, err := SealRequestState(key, stateResource, state, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return sealed
}

//spec:covers REQ-MCPGATEWAY-014: 封緘した状態は同じ鍵と resource でだけ開け、承認リクエストの識別子と digest を平文で運ばないこと
func TestRequestStateOpensOnlyWithTheSameKeyAndResource(t *testing.T) {
	key := RequestStateKey("client-secret")
	expires := time.Date(2026, 11, 18, 0, 10, 0, 0, time.UTC)
	state := RequestState{ApprovalRequestID: "0b6f2c8e-6a59-4f1e-9a0f-3f1d6c2b7a10", OperationDigest: strings.Repeat("d", 64), ExpiresAt: expires}
	sealed := sealedState(t, key, state)
	if !strings.HasPrefix(sealed, RequestStatePrefix) || strings.Contains(sealed, state.ApprovalRequestID) || strings.Contains(sealed, state.OperationDigest) {
		t.Fatalf("sealed=%q", sealed)
	}
	opened, err := OpenRequestState(key, stateResource, sealed, expires.Add(-time.Second))
	if err != nil || opened.ApprovalRequestID != state.ApprovalRequestID || opened.OperationDigest != state.OperationDigest || !opened.ExpiresAt.Equal(expires) {
		t.Fatalf("opened=%+v err=%v", opened, err)
	}
	if _, err := OpenRequestState(RequestStateKey("rotated-secret"), stateResource, sealed, expires.Add(-time.Second)); !errors.Is(err, ErrRequestStateInvalid) {
		t.Fatalf("another key: %v", err)
	}
	if _, err := OpenRequestState(key, "https://other.example/", sealed, expires.Add(-time.Second)); !errors.Is(err, ErrRequestStateInvalid) {
		t.Fatalf("another resource: %v", err)
	}
	if sealedState(t, key, state) == sealed {
		t.Fatal("two seals of the same state are equal; the nonce is not random")
	}
}

//spec:covers REQ-MCPGATEWAY-014: 期限の直前は開け、期限ちょうどと後は期限切れとして区別すること
func TestRequestStateExpiresAtItsDeadline(t *testing.T) {
	key := RequestStateKey("client-secret")
	expires := time.Date(2026, 11, 18, 0, 10, 0, 0, time.UTC)
	sealed := sealedState(t, key, RequestState{ApprovalRequestID: "id", OperationDigest: "digest", ExpiresAt: expires})
	if _, err := OpenRequestState(key, stateResource, sealed, expires.Add(-time.Nanosecond)); err != nil {
		t.Fatalf("before the deadline: %v", err)
	}
	for _, now := range []time.Time{expires, expires.Add(time.Second)} {
		if _, err := OpenRequestState(key, stateResource, sealed, now); !errors.Is(err, ErrRequestStateExpired) {
			t.Fatalf("at %v: %v, want expired", now, err)
		}
	}
}

//spec:covers REQ-MCPGATEWAY-014: 書き換え、切り詰め、接頭辞の誤り、空の識別子を開けない状態として拒否すること
func TestRequestStateRejectsTampering(t *testing.T) {
	key := RequestStateKey("client-secret")
	now := time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC)
	sealed := sealedState(t, key, RequestState{ApprovalRequestID: "id", OperationDigest: "digest", ExpiresAt: now.Add(time.Minute)})
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(sealed, RequestStatePrefix))
	if err != nil {
		t.Fatal(err)
	}
	flipped := bytes.Clone(raw)
	flipped[len(flipped)-1] ^= 1
	// 同じ鍵で封緘した、識別子のない中身。
	empty := sealedState(t, key, RequestState{OperationDigest: "digest", ExpiresAt: now.Add(time.Minute)})
	for name, value := range map[string]string{
		"flipped":   RequestStatePrefix + base64.RawURLEncoding.EncodeToString(flipped),
		"truncated": RequestStatePrefix + base64.RawURLEncoding.EncodeToString(raw[:8]),
		"no base64": RequestStatePrefix + "!!",
		"prefix":    "igw2." + strings.TrimPrefix(sealed, RequestStatePrefix),
		"empty id":  empty,
	} {
		if _, err := OpenRequestState(key, stateResource, value, now); !errors.Is(err, ErrRequestStateInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

//spec:covers REQ-MCPGATEWAY-014: 引数の digest がキーの順序と空白に依存せず、ツール名と値の違いで変わること
func TestOperationDigestIgnoresKeyOrder(t *testing.T) {
	arguments := func(raw string) map[string]json.RawMessage {
		var decoded map[string]json.RawMessage
		if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
			t.Fatal(err)
		}
		return decoded
	}
	base, err := OperationDigest("execute", arguments(`{"amount":100,"currency":"JPY","nested":{"b":1,"a":2}}`))
	if err != nil {
		t.Fatal(err)
	}
	same, _ := OperationDigest("execute", arguments(`{ "nested":{"a":2,"b":1}, "currency":"JPY", "amount":100 }`))
	if same != base {
		t.Fatal("key order changed the digest")
	}
	for name, digest := range map[string]func() (string, error){
		"value": func() (string, error) {
			return OperationDigest("execute", arguments(`{"amount":101,"currency":"JPY","nested":{"a":2,"b":1}}`))
		},
		"tool": func() (string, error) {
			return OperationDigest("other", arguments(`{"amount":100,"currency":"JPY","nested":{"a":2,"b":1}}`))
		},
	} {
		if got, err := digest(); err != nil || got == base {
			t.Errorf("changing the %s kept the digest (err=%v)", name, err)
		}
	}
	empty, _ := OperationDigest("execute", nil)
	braces, _ := OperationDigest("execute", map[string]json.RawMessage{})
	if empty != braces {
		t.Fatal("absent and empty arguments differ")
	}
}
