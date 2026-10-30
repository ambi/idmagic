package domain

import (
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/shared/spec"
)

//spec:covers REQ-OAUTH2-065: 引数の比較がキーの順序と空白に依存せず、丸められる数と重複キーを受けないこと
func TestCanonicalizeOperationParameters(t *testing.T) {
	approved, err := DigestOperationFields(map[string]any{"amount": float64(100), "to": "acct-1"})
	if err != nil {
		t.Fatal(err)
	}
	empty, err := DigestOperationFields(nil)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		raw  string
		want string
		fail bool
	}{
		{name: "same", raw: `{"amount":100,"to":"acct-1"}`, want: approved},
		{name: "reordered with spaces", raw: ` { "to" : "acct-1", "amount" : 100 } `, want: approved},
		{name: "omitted equals empty object", raw: "", want: empty},
		{name: "empty object", raw: `{}`, want: empty},
		{name: "array", raw: `[1]`, fail: true},
		{name: "scalar", raw: `1`, fail: true},
		{name: "malformed", raw: `{"amount":`, fail: true},
		{name: "trailing value", raw: `{} {}`, fail: true},
		{name: "duplicate key", raw: `{"amount":100,"amount":1}`, fail: true},
		{name: "nested duplicate key", raw: `{"a":{"b":1,"b":2}}`, fail: true},
		{name: "integer a float64 rounds", raw: `{"amount":9007199254740993}`, fail: true},
		{name: "integer at the limit", raw: `{"amount":9007199254740992}`, fail: true},
		{name: "decimal a float64 rounds", raw: `{"amount":0.1000000000000000001}`, fail: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CanonicalizeOperationParameters(tc.raw)
			if tc.fail {
				if err == nil {
					t.Fatalf("digest=%s, want error", got)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("digest=%s err=%v, want %s", got, err, tc.want)
			}
		})
	}
	changed, err := CanonicalizeOperationParameters(`{"amount":101,"to":"acct-1"}`)
	if err != nil || changed == approved {
		t.Fatalf("changed amount digest=%s err=%v", changed, err)
	}
	// 文字列を正規化しない。合成済みと分解済みの「é」は別の値である。
	composed, _ := CanonicalizeOperationParameters("{\"to\":\"é\"}")
	decomposed, _ := CanonicalizeOperationParameters("{\"to\":\"é\"}")
	if composed == decomposed {
		t.Fatal("strings must be compared without normalization")
	}
}

//spec:covers REQ-OAUTH2-065: 対象の一致の両側と期限の境界、空の identifier を全許可と読まないこと
func TestMatchApprovedOperation(t *testing.T) {
	now := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	detail := spec.AuthorizationDetail{
		Type: "payment", Locations: []string{"https://rs/"}, Actions: []string{"execute"}, Identifier: "acct-1",
		Fields: map[string]any{"amount": float64(100)},
	}
	actual := OperationBinding{DetailType: "payment", Location: "https://rs/", Action: "execute", Identifier: "acct-1"}
	digest, err := CanonicalizeOperationParameters(`{"amount":100}`)
	if err != nil {
		t.Fatal(err)
	}
	wildcard := detail
	wildcard.Identifier = ""
	cases := []struct {
		name    string
		mutate  func(*OperationBinding)
		details []spec.AuthorizationDetail
		expires time.Time
		digest  string
		want    ApprovedOperationMatch
	}{
		{name: "same", want: ApprovedOperationMatched},
		{name: "one nanosecond before expiry", expires: now.Add(time.Nanosecond), want: ApprovedOperationMatched},
		{name: "at expiry", expires: now, want: ApprovedOperationExpired},
		{name: "other type", mutate: func(o *OperationBinding) { o.DetailType = "x" }, want: ApprovedOperationMismatched},
		{name: "other location", mutate: func(o *OperationBinding) { o.Location = "https://other/" }, want: ApprovedOperationMismatched},
		{name: "other action", mutate: func(o *OperationBinding) { o.Action = "x" }, want: ApprovedOperationMismatched},
		{name: "other identifier", mutate: func(o *OperationBinding) { o.Identifier = "acct-2" }, want: ApprovedOperationMismatched},
		{name: "empty identifier is no wildcard", details: []spec.AuthorizationDetail{wildcard}, want: ApprovedOperationMismatched},
		{name: "other arguments", digest: "other", want: ApprovedOperationMismatched},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			candidate := actual
			if tc.mutate != nil {
				tc.mutate(&candidate)
			}
			rec := &ApprovalRequest{AuthorizationDetails: []spec.AuthorizationDetail{detail}, ExpiresAt: now.Add(time.Minute)}
			if tc.details != nil {
				rec.AuthorizationDetails = tc.details
			}
			if !tc.expires.IsZero() {
				rec.ExpiresAt = tc.expires
			}
			used := digest
			if tc.digest != "" {
				used = tc.digest
			}
			if got := MatchApprovedOperation(rec, candidate, used, now); got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}

//spec:covers REQ-OAUTH2-065: 同じ鍵でも別のリソースサーバーとは使用権を共有しないこと
func TestHashOperationKeyIsScopedToTheResourceServer(t *testing.T) {
	if HashOperationKey("rs-a", "k") == HashOperationKey("rs-b", "k") {
		t.Fatal("key must be scoped to the resource server")
	}
	if HashOperationKey("rs", "a\x00b") == HashOperationKey("rs\x00a", "b") {
		t.Fatal("client id and key must not be confusable")
	}
}
