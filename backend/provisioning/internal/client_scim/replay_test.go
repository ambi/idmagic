package client_scim

import (
	"context"
	"testing"

	provisioningdomain "github.com/ambi/idmagic/backend/provisioning/internal/domain"
	"github.com/ambi/idmagic/backend/provisioning/internal/testing_scimtwin"
)

// 下流の実際の応答の形に対して、クライアントが取り出す結果を確かめる。
//
// 応答は RFC 7644 と RFC 7643 の例を転記した記録から返す。期待値は製品の定数からではなく、
// 記録が示す値と RFC の文面から書く。記録した要求と、クライアントの要求の方式、パスの形、
// Accept と Content-Type が食い違うと、再生サーバーがテストを失敗させる。
//
//spec:covers RFC7644-OUT-PATCH: 下流が PATCH (属性の更新と Group のメンバーの追加) に返す 200 と 204 No Content は、どちらも成功として扱う
func TestReplay_RFC7644Recordings(t *testing.T) {
	rec := testing_scimtwin.RFC7644Recording(t)
	ctx := context.Background()
	rules := []provisioningdomain.AttributeMappingRule{simpleRule("userName", "username")}
	attrs := map[string]any{"username": "bjensen"}
	const recordedUserID = "2819c223-7f76-453a-919d-413861904646"

	tests := []struct {
		name     string
		exchange string
		run      func(t *testing.T, client *Client)
	}{
		{"discover", "discover", func(t *testing.T, client *Client) {
			t.Helper()
			caps, err := client.Discover(ctx)
			if err != nil {
				t.Fatalf("Discover() error = %v", err)
			}
			want := provisioningdomain.ProvisioningCapabilities{SupportsPatch: true, SupportsBulk: true, SupportsFilter: true, SupportsEtag: true, SupportsSort: true}
			if caps != want {
				t.Errorf("Discover() = %+v, want %+v", caps, want)
			}
		}},
		{"create-user", "create-user", func(t *testing.T, client *Client) {
			t.Helper()
			id, etag, err := client.CreateUser(ctx, rules, attrs)
			if err != nil {
				t.Fatalf("CreateUser() error = %v", err)
			}
			if id != recordedUserID {
				t.Errorf("CreateUser() id = %q, want %q", id, recordedUserID)
			}
			if etag == nil || *etag != `W/"e180ee84f0671b1"` {
				t.Errorf("CreateUser() etag = %v, want W/\"e180ee84f0671b1\"", etag)
			}
		}},
		{"search-user", "query-users", func(t *testing.T, client *Client) {
			t.Helper()
			id, found, err := client.SearchUserByAttribute(ctx, "userName", "bjensen")
			if err != nil {
				t.Fatalf("SearchUserByAttribute() error = %v", err)
			}
			if !found || id != recordedUserID {
				t.Errorf("SearchUserByAttribute() = (%q, %v), want (%q, true)", id, found, recordedUserID)
			}
		}},
		{"replace-user", "replace-user", func(t *testing.T, client *Client) {
			t.Helper()
			etag, err := client.UpdateUser(ctx, recordedUserID, rules, attrs, false)
			if err != nil {
				t.Fatalf("UpdateUser(put) error = %v", err)
			}
			if etag == nil || *etag != `W/"b431af54f0671a2"` {
				t.Errorf("UpdateUser(put) etag = %v, want W/\"b431af54f0671a2\"", etag)
			}
		}},
		{"add-group-member", "add-group-member", func(t *testing.T, client *Client) {
			t.Helper()
			if err := client.PatchGroupMembers(ctx, "acbf3ae7-8463-4692-b4fd-9b4da3f908ce", "add", []string{recordedUserID}); err != nil {
				t.Fatalf("PatchGroupMembers() error = %v", err)
			}
		}},
		{"patch-group-no-content", "add-group-member", func(t *testing.T, client *Client) {
			t.Helper()
			// 同じ 204 の応答を、属性の更新の PATCH に対しても成功として受け取る。
			etag, err := client.UpdateGroup(ctx, "acbf3ae7-8463-4692-b4fd-9b4da3f908ce", []provisioningdomain.AttributeMappingRule{simpleRule("displayName", "name")}, map[string]any{"name": "Tour Guides"}, true)
			if err != nil {
				t.Fatalf("UpdateGroup(patch) error = %v", err)
			}
			if etag == nil || *etag != `W/"b431af54f0671a2"` {
				t.Errorf("UpdateGroup(patch) etag = %v, want W/\"b431af54f0671a2\"", etag)
			}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := testing_scimtwin.NewReplayServer(t, rec.Exchange(t, tc.exchange))
			client := &Client{HTTPClient: server.Client(), BaseURL: server.URL, tokenSource: staticToken("test-token")}
			tc.run(t, client)
		})
	}
}
