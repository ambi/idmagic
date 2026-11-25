package client_scim

import (
	"context"
	"errors"
	"testing"
	"time"

	provisioningdomain "github.com/ambi/idmagic/backend/provisioning/internal/domain"
	provisioningports "github.com/ambi/idmagic/backend/provisioning/internal/ports"
	"github.com/ambi/idmagic/backend/provisioning/internal/testing_scimtwin"
)

// 記録にない操作の組み合わせと障害を、状態を持つ下流の模倣に対して確かめる。
// 模倣が記録と同じ応答を返すことは testing_scimtwin のテストが確かめる。

func newTwinClient(t *testing.T, tw *testing_scimtwin.Twin) *Client {
	t.Helper()
	server := tw.Start(t)
	return &Client{HTTPClient: server.Client(), BaseURL: server.URL, tokenSource: staticToken("test-token")}
}

//spec:covers RFC7644-OUT-FILTERING: 値は JSON の文字列として符号化する
func TestTwin_SearchUserByAttribute_EncodesValueAsJSONString(t *testing.T) {
	for _, userName := range []string{"tab\vseparated", "bell\a", "ctl\x01", `quote"and\backslash`, "<html>&amp;", "日本語"} {
		t.Run(userName, func(t *testing.T) {
			tw := testing_scimtwin.NewTwin()
			tw.Put("Users", "remote-1", map[string]any{"userName": userName})
			client := newTwinClient(t, tw)

			id, found, err := client.SearchUserByAttribute(context.Background(), "userName", userName)
			if err != nil {
				t.Fatalf("SearchUserByAttribute(%q) error = %v", userName, err)
			}
			if !found || id != "remote-1" {
				t.Errorf("SearchUserByAttribute(%q) = (%q, %v), want (remote-1, true)", userName, id, found)
			}
		})
	}
}

// 作成が重複で 409 になったら、一致判定の属性で既存のリソースを見つけられる。
func TestTwin_CreateConflictResolvesBySearch(t *testing.T) {
	tw := testing_scimtwin.NewTwin()
	tw.Put("Users", "existing", map[string]any{"userName": "Alice"})
	client := newTwinClient(t, tw)
	ctx := context.Background()

	_, _, err := client.CreateUser(ctx, []provisioningdomain.AttributeMappingRule{simpleRule("userName", "username")}, map[string]any{"username": "alice"})
	if _, ok := errors.AsType[*provisioningports.ConflictError](err); !ok {
		t.Fatalf("CreateUser() error = %v, want ConflictError", err)
	}
	id, found, err := client.SearchUserByAttribute(ctx, "userName", "alice")
	if err != nil || !found || id != "existing" {
		t.Errorf("SearchUserByAttribute() = (%q, %v, %v), want (existing, true, nil)", id, found, err)
	}
}

// 429 の連続は、どれも Retry-After を持つ再試行の対象として返り、相手の状態を変えない。
func TestTwin_RateLimitedRequestsAreRetryable(t *testing.T) {
	tw := testing_scimtwin.NewTwin()
	tw.InjectFaults(
		testing_scimtwin.Fault{Status: 429, RetryAfter: "7"},
		testing_scimtwin.Fault{Status: 429, RetryAfter: "7"},
	)
	client := newTwinClient(t, tw)
	ctx := context.Background()
	rules := []provisioningdomain.AttributeMappingRule{simpleRule("userName", "username")}
	attrs := map[string]any{"username": "alice"}

	for range 2 {
		_, _, err := client.CreateUser(ctx, rules, attrs)
		retryable, ok := errors.AsType[*provisioningports.RetryableError](err)
		if !ok || retryable.StatusCode != 429 || retryable.RetryAfter != 7*time.Second {
			t.Fatalf("CreateUser() error = %#v, want RetryableError{429, 7s}", err)
		}
	}
	id, _, err := client.CreateUser(ctx, rules, attrs)
	if err != nil {
		t.Fatalf("CreateUser() after the rate limit error = %v", err)
	}
	if _, ok := tw.Get("Users", id); !ok {
		t.Errorf("twin does not hold the created user %q", id)
	}
}

// 解釈できない本文は、相関の ID を作らずに失敗する。
func TestTwin_MalformedCreateResponseFails(t *testing.T) {
	tw := testing_scimtwin.NewTwin()
	tw.InjectFaults(testing_scimtwin.Fault{Status: 201, MalformedBody: true})
	client := newTwinClient(t, tw)

	id, _, err := client.CreateUser(context.Background(), []provisioningdomain.AttributeMappingRule{simpleRule("userName", "username")}, map[string]any{"username": "alice"})
	if err == nil || id != "" {
		t.Errorf("CreateUser() = (%q, %v), want a decode error and no id", id, err)
	}
}

// Group のメンバーの追加は、204 と 200 のどちらを返す下流でも相手の状態へ届く。
func TestTwin_PatchGroupMembersReachesTheCounterpart(t *testing.T) {
	for _, returnsResource := range []bool{false, true} {
		tw := testing_scimtwin.NewTwin()
		tw.PatchReturnsResource = returnsResource
		tw.Put("Groups", "g-1", map[string]any{"displayName": "Engineering"})
		client := newTwinClient(t, tw)

		if err := client.PatchGroupMembers(context.Background(), "g-1", "add", []string{"u-1", "u-2"}); err != nil {
			t.Fatalf("PatchGroupMembers(returnsResource=%v) error = %v", returnsResource, err)
		}
		group, _ := tw.Get("Groups", "g-1")
		members, _ := group["members"].([]any)
		if len(members) != 2 {
			t.Errorf("returnsResource=%v: members = %v, want u-1 and u-2", returnsResource, group["members"])
		}
	}
}
