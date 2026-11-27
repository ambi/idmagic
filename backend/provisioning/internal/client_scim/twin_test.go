package client_scim

import (
	"context"
	"testing"

	provisioningdomain "github.com/ambi/idmagic/backend/provisioning/internal/domain"
	"github.com/ambi/idmagic/backend/provisioning/internal/testing_scimtwin"
)

// 相手の状態を通して観測する本文の形と障害を、状態を持つ下流の模倣に対して確かめる。
// 一つの応答で決まる状態コードの扱いは client_test.go が確かめる。
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

// Group のメンバーの追加の本文は、相手が解釈して状態へ届く形である。
// 200 と 204 のどちらも成功とすることは、client_test.go と記録の再生が確かめる。
func TestTwin_PatchGroupMembersReachesTheCounterpart(t *testing.T) {
	tw := testing_scimtwin.NewTwin()
	tw.Put("Groups", "g-1", map[string]any{"displayName": "Engineering"})
	client := newTwinClient(t, tw)

	if err := client.PatchGroupMembers(context.Background(), "g-1", "add", []string{"u-1", "u-2"}); err != nil {
		t.Fatalf("PatchGroupMembers() error = %v", err)
	}
	group, _ := tw.Get("Groups", "g-1")
	members, _ := group["members"].([]any)
	if len(members) != 2 {
		t.Errorf("members = %v, want u-1 and u-2", group["members"])
	}
}
