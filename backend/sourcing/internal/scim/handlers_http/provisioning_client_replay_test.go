package handlers_http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/ambi/idmagic/backend/shared/testing_httprecord"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
)

// entraNotReplayed は、記録のうち再生しない往復と、その理由である。
// 担当の項目が往復を受け付けるようにしたら、ここから消して再生の列へ入れる。
var entraNotReplayed = map[string]string{
	"update-user-multi-valued": "path の emails[type eq \"work\"].value は wi-248 が対応する",
	"create-user-with-nulls":   "null の phoneNumbers と addresses を未割り当てとして受け付けるのは wi-56158 が扱う",
	"add-manager": "manager は RFC 7643 §4.3 で単一の値の属性であり、配列の value は規範にない形である。" +
		"文書のこの例は参照実装の節にあり、Entra ID は manager を文字列で送る（mutation.go の parseManagerScimID）。実テナントで配列を観測したら見直す（wi-79）",
	"discover-schema": "/Schemas の一覧を ListResponse で返すのは wi-90508 が扱う",
}

// clientReplay は、SCIM のクライアント（Entra ID、Okta）の文書から転記した要求の記録を、SCIM の受け口へ送る。
type clientReplay struct {
	t     *testing.T
	h     scimHarness
	token string
	rec   testing_httprecord.Recording
	// base は記録のパスの前に付ける接頭辞である。記録がエンドポイントの基底からの相対で書かれていれば /scim/v2 にする。
	base string
	sent map[string]bool
}

func newClientReplay(t *testing.T, recording, base string) *clientReplay {
	t.Helper()
	data, err := os.ReadFile(recording)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := testing_httprecord.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	h := newScimHarness()
	return &clientReplay{t: t, h: h, token: issueAllScimToken(t, h.apiTokens), rec: rec, base: base, sent: map[string]bool{}}
}

// assertAccounted は、記録のすべての往復が、再生したか notReplayed に理由とともに挙げたかの一方であることを確かめる。
func (r *clientReplay) assertAccounted(notReplayed map[string]string) {
	r.t.Helper()
	for _, ex := range r.rec.Exchanges {
		_, skipped := notReplayed[ex.Name]
		if r.sent[ex.Name] == skipped {
			r.t.Errorf("exchange %q must be either replayed or listed with a reason, not both or neither", ex.Name)
		}
	}
	for name := range notReplayed {
		if !slices.ContainsFunc(r.rec.Exchanges, func(ex testing_httprecord.Exchange) bool { return ex.Name == name }) {
			r.t.Errorf("not-replayed list names %q, which the recording does not have", name)
		}
	}
}

// send は名前の往復の要求を送る。ids は記録の識別子を IdMagic が割り当てた識別子へ置き換え、
// パスと本文の両方に当てる。
func (r *clientReplay) send(name string, ids map[string]string) (*httptest.ResponseRecorder, map[string]any) {
	r.t.Helper()
	r.sent[name] = true
	ex := r.rec.Exchange(r.t, name)
	replace := func(s string) string {
		for recorded, assigned := range ids {
			s = strings.ReplaceAll(s, recorded, assigned)
		}
		return s
	}
	target := r.base + replace(ex.Request.Path)
	if ex.Request.Query != "" {
		query, err := url.ParseQuery(ex.Request.Query)
		if err != nil {
			r.t.Fatalf("%s: query: %v", name, err)
		}
		target += "?" + query.Encode()
	}
	var body []byte
	if len(ex.Request.Body) > 0 {
		body = []byte(replace(string(testing_httprecord.BodyBytes(ex.Request.Body))))
	}
	req := httptest.NewRequest(ex.Request.Method, target, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+r.token)
	for key, value := range ex.Request.Headers {
		req.Header.Set(key, value)
	}
	w := httptest.NewRecorder()
	r.h.echo.ServeHTTP(w, req)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return w, out
}

// read は資源を読み直し、要求の作用を観測する。
func (r *clientReplay) read(path string) (int, map[string]any) {
	r.t.Helper()
	w, out := doScimGet(r.t, r.h.echo, r.token, path)
	return w.Code, out
}

func expectStatus(t *testing.T, name string, w *httptest.ResponseRecorder, accepted ...int) {
	t.Helper()
	if slices.Contains(accepted, w.Code) {
		return
	}
	t.Fatalf("%s: status %d, want one of %v; body=%s", name, w.Code, accepted, w.Body.String())
}

func listResources(t *testing.T, name string, body map[string]any) []any {
	t.Helper()
	schemas, _ := body["schemas"].([]any)
	if len(schemas) != 1 || schemas[0] != "urn:ietf:params:scim:api:messages:2.0:ListResponse" {
		t.Fatalf("%s: a query must answer a ListResponse, got %v", name, body)
	}
	resources, _ := body["Resources"].([]any)
	return resources
}

func memberValues(body map[string]any) []string {
	members, _ := body["members"].([]any)
	values := make([]string, 0, len(members))
	for _, member := range members {
		object, _ := member.(map[string]any)
		value, _ := object["value"].(string)
		values = append(values, value)
	}
	return values
}

// 期待値は Microsoft Learn の同じ文書の文面から書く。
// 作成は 201 と作成した資源、照会は常に ListResponse で一致がなければ totalResults 0、
// PATCH は 2xx（資源の全体を返す必要はない。Group は 204 を勧める）、削除は 204 である。
// 作用は SCIM Validator の手順と同じく、要求の後に資源を読み直して確かめる。
//
//spec:covers RFC7644-RESOURCE-OPERATIONS: Entra ID が送る形の User と Group の作成、参照、削除が、作成した資源を返し、削除の後に 404 になること,
//spec:covers RFC7644-PATCH: 大文字で始まる op（Replace、Add、Remove）と、value を持つ members の Remove を、読み直した資源に反映すること,
//spec:covers RFC7644-FILTERING: userName と displayName の eq の照会が、一致を ListResponse で返し、一致がなければ totalResults 0 を返すこと,
//spec:covers RFC7644-ERROR-RESPONSE: 存在しない User の参照に、status が "404" の SCIM エラーを返すこと
func TestEntraProvisioning_AcceptsTheDocumentedRequests(t *testing.T) {
	r := newClientReplay(t, "testdata/entra-provisioning.json", "/scim/v2")

	w, user := r.send("create-user", nil)
	expectStatus(t, "create-user", w, http.StatusCreated)
	uid, _ := user["id"].(string)
	if uid == "" || user["userName"] != "Test_User_00aa00aa-bb11-cc22-dd33-44ee44ee44ee" || user["active"] != true {
		t.Fatalf("create-user: created resource %v does not carry the sent userName and active", user)
	}
	if name, _ := user["name"].(map[string]any); name["givenName"] != "givenName" || name["familyName"] != "familyName" {
		t.Fatalf("create-user: name = %v, want the sent givenName and familyName", user["name"])
	}
	emails, _ := user["emails"].([]any)
	if len(emails) != 1 || emails[0].(map[string]any)["value"] != "Test_User_11bb11bb-cc22-dd33-ee44-55ff55ff55ff@testuser.com" {
		t.Fatalf("create-user: emails = %v, want the sent work email", user["emails"])
	}

	w, got := r.send("get-user", map[string]string{"5d48a0a8e9f04aa38008": uid})
	expectStatus(t, "get-user", w, http.StatusOK)
	if got["id"] != uid {
		t.Fatalf("get-user: id = %v, want %s", got["id"], uid)
	}

	// 記録の識別子は IdMagic が割り当てていないので、置き換えずに送ると存在しない User になる。
	w, got = r.send("get-user-not-found", nil)
	expectStatus(t, "get-user-not-found", w, http.StatusNotFound)
	if schemas, _ := got["schemas"].([]any); len(schemas) != 1 || schemas[0] != "urn:ietf:params:scim:api:messages:2.0:Error" || got["status"] != "404" {
		t.Fatalf("get-user-not-found: body %v is not a SCIM error with status \"404\"", got)
	}

	w, got = r.send("query-user", nil)
	expectStatus(t, "query-user", w, http.StatusOK)
	if resources := listResources(t, "query-user", got); got["totalResults"] != float64(1) || len(resources) != 1 || resources[0].(map[string]any)["id"] != uid {
		t.Fatalf("query-user: %v, want the created user alone", got)
	}

	w, got = r.send("query-user-zero-results", nil)
	expectStatus(t, "query-user-zero-results", w, http.StatusOK)
	if resources := listResources(t, "query-user-zero-results", got); got["totalResults"] != float64(0) || len(resources) != 0 {
		t.Fatalf("query-user-zero-results: %v, want totalResults 0 and no resources", got)
	}

	userIDs := map[string]string{"5171a35d82074e068ce2": uid}
	w, _ = r.send("update-user-single-valued", userIDs)
	expectStatus(t, "update-user-single-valued", w, http.StatusOK, http.StatusNoContent)
	if _, got := r.read("/scim/v2/Users?filter=" + url.QueryEscape(`userName eq "5b50642d-79fc-4410-9e90-4c077cdd1a59@testuser.com"`)); got["totalResults"] != float64(1) {
		t.Fatalf("update-user-single-valued: the new userName does not find the user: %v", got)
	}

	w, _ = r.send("disable-user", userIDs)
	expectStatus(t, "disable-user", w, http.StatusOK, http.StatusNoContent)
	if _, got := r.read("/scim/v2/Users/" + uid); got["active"] != false {
		t.Fatalf("disable-user: active = %v after the PATCH, want false", got["active"])
	}

	w, group := r.send("create-group", nil)
	expectStatus(t, "create-group", w, http.StatusCreated)
	gid, _ := group["id"].(string)
	if gid == "" || group["displayName"] != "displayName" || len(memberValues(group)) != 0 {
		t.Fatalf("create-group: created resource %v, want the sent displayName and no members", group)
	}
	groupIDs := map[string]string{
		"40734ae655284ad3abcc": gid, "fa2ce26709934589afc5": gid, "a99962b9f99d4c4fac67": gid, "cdb1ce18f65944079d37": gid,
		"f648f8d5ea4e4cd38e9c": uid,
	}

	w, got = r.send("get-group", groupIDs)
	expectStatus(t, "get-group", w, http.StatusOK)
	if got["id"] != gid {
		t.Fatalf("get-group: id = %v, want %s", got["id"], gid)
	}

	w, got = r.send("query-group", nil)
	expectStatus(t, "query-group", w, http.StatusOK)
	if resources := listResources(t, "query-group", got); got["totalResults"] != float64(1) || len(resources) != 1 || resources[0].(map[string]any)["id"] != gid {
		t.Fatalf("query-group: %v, want the created group alone", got)
	}

	w, _ = r.send("update-group-display-name", groupIDs)
	expectStatus(t, "update-group-display-name", w, http.StatusOK, http.StatusNoContent)
	if _, got := r.read("/scim/v2/Groups/" + gid); got["displayName"] != "1879db59-3bdf-4490-ad68-ab880a269474updatedDisplayName" {
		t.Fatalf("update-group-display-name: displayName = %v after the PATCH", got["displayName"])
	}

	w, _ = r.send("add-members", groupIDs)
	expectStatus(t, "add-members", w, http.StatusOK, http.StatusNoContent)
	if _, got := r.read("/scim/v2/Groups/" + gid); len(memberValues(got)) != 1 || memberValues(got)[0] != uid {
		t.Fatalf("add-members: members = %v after the PATCH, want the user %s", memberValues(got), uid)
	}

	// Remove が value を持つので、members の全体ではなく、名指した User だけを外す。
	// それを観測できるよう、記録の外で名指されない二人目のメンバーを足しておく。
	_, other := doScimJSON(t, r.h.echo, http.MethodPost, r.token, "/scim/v2/Users", map[string]any{"userName": "other@example.com"})
	otherID, _ := other["id"].(string)
	if rec, _ := doScimJSON(t, r.h.echo, http.MethodPatch, r.token, "/scim/v2/Groups/"+gid, patchOp("add", "members", []any{map[string]any{"value": otherID}})); rec.Code != http.StatusOK {
		t.Fatalf("setup: adding the second member answered %d", rec.Code)
	}
	w, _ = r.send("remove-members", groupIDs)
	expectStatus(t, "remove-members", w, http.StatusOK, http.StatusNoContent)
	if _, got := r.read("/scim/v2/Groups/" + gid); len(memberValues(got)) != 1 || memberValues(got)[0] != otherID {
		t.Fatalf("remove-members: members = %v after the PATCH, want only the member it did not name (%s)", memberValues(got), otherID)
	}

	w, _ = r.send("delete-group", groupIDs)
	expectStatus(t, "delete-group", w, http.StatusNoContent)
	if status, _ := r.read("/scim/v2/Groups/" + gid); status != http.StatusNotFound {
		t.Fatalf("delete-group: re-read status %d, want 404", status)
	}

	w, _ = r.send("delete-user", userIDs)
	expectStatus(t, "delete-user", w, http.StatusNoContent)
	if status, _ := r.read("/scim/v2/Users/" + uid); status != http.StatusNotFound {
		t.Fatalf("delete-user: re-read status %d, want 404", status)
	}

	r.assertAccounted(entraNotReplayed)
}

// oktaNotReplayed は、Okta の記録のうち再生しない往復と、その理由である。
var oktaNotReplayed = map[string]string{
	"rename-group-patch":      "path を省いた replace の value に今と同じ値の id を含めた要求を受け付けるのは wi-41091 が扱う",
	"update-membership-patch": "path の members[value eq \"...\"] は wi-248 が対応する",
}

// 期待値は Okta の SCIM 2.0 protocol reference の文面から書く。
// 接続の確認と照会は ListResponse を返し、一致がなければ totalResults 0 である。
// 作成は 201 と資源を返し、同じ userName の作成は 409 である。PATCH は 200 か 204 である。
// Okta は User を削除せず、active を false にする。作成の password は仮の値で意味を持たない。
//
//spec:covers RFC7644-RESOURCE-OPERATIONS: Okta が送る形の User の作成と置換、Group の作成、置換、削除が、作成した資源を返し、読み直した資源に反映されること,
//spec:covers RFC7644-PATCH: path を省いた replace の active と、members の replace を、読み直した資源に反映すること,
//spec:covers RFC7644-FILTERING: startIndex と count を伴う userName と displayName の eq の照会が、一致を ListResponse で返し、一致がなければ totalResults 0 を返すこと,
//spec:covers RFC7644-ERROR-RESPONSE: 同じ userName の作成に、scimType が uniqueness の 409 を返すこと
func TestOktaProvisioning_AcceptsTheDocumentedRequests(t *testing.T) {
	r := newClientReplay(t, "testdata/okta-provisioning.json", "")

	for _, name := range []string{"list-users-connection-test", "list-groups-connection-test", "find-user"} {
		w, got := r.send(name, nil)
		expectStatus(t, name, w, http.StatusOK)
		if resources := listResources(t, name, got); got["totalResults"] != float64(0) || len(resources) != 0 {
			t.Fatalf("%s: %v, want an empty ListResponse before provisioning", name, got)
		}
	}

	w, user := r.send("create-user", nil)
	expectStatus(t, "create-user", w, http.StatusCreated)
	uid, _ := user["id"].(string)
	if uid == "" || user["userName"] != "test.user@okta.local" || user["active"] != true {
		t.Fatalf("create-user: created resource %v does not carry the sent userName and active", user)
	}
	stored, err := r.h.userRepo.FindByUsername(t.Context(), tenancydomain.DefaultTenantID, "test.user@okta.local")
	if err != nil || stored == nil {
		t.Fatalf("create-user: the created user is not stored: %v", err)
	}
	if stored.PasswordHash != "" {
		t.Fatal("create-user: Okta's placeholder password must not become the user's password")
	}

	w, got := r.send("create-user-conflict", nil)
	expectStatus(t, "create-user-conflict", w, http.StatusConflict)
	if got["scimType"] != "uniqueness" {
		t.Fatalf("create-user-conflict: scimType = %v, want uniqueness", got["scimType"])
	}

	w, got = r.send("retrieve-users", nil)
	expectStatus(t, "retrieve-users", w, http.StatusOK)
	if resources := listResources(t, "retrieve-users", got); got["totalResults"] != float64(1) || len(resources) != 1 || resources[0].(map[string]any)["id"] != uid {
		t.Fatalf("retrieve-users: %v, want the created user alone", got)
	}

	userIDs := map[string]string{"23a35c27-23d3-4c03-b4c5-6443c09e7173": uid}
	w, got = r.send("get-user", userIDs)
	expectStatus(t, "get-user", w, http.StatusOK)
	if got["id"] != uid {
		t.Fatalf("get-user: id = %v, want %s", got["id"], uid)
	}

	w, _ = r.send("replace-user", userIDs)
	expectStatus(t, "replace-user", w, http.StatusOK)
	if _, got := r.read("/scim/v2/Users/" + uid); got["name"].(map[string]any)["givenName"] != "Another" {
		t.Fatalf("replace-user: name = %v after the PUT, want givenName Another", got["name"])
	}

	w, _ = r.send("deactivate-user", userIDs)
	expectStatus(t, "deactivate-user", w, http.StatusOK, http.StatusNoContent)
	if _, got := r.read("/scim/v2/Users/" + uid); got["active"] != false {
		t.Fatalf("deactivate-user: active = %v after the PATCH, want false", got["active"])
	}

	w, group := r.send("create-group", nil)
	expectStatus(t, "create-group", w, http.StatusCreated)
	gid, _ := group["id"].(string)
	if gid == "" || group["displayName"] != "Test SCIMv2" {
		t.Fatalf("create-group: created resource %v, want the sent displayName", group)
	}

	// 記録の members が指す User を、記録の外で作っておく。
	member := func(userName string) string {
		_, created := doScimJSON(t, r.h.echo, http.MethodPost, r.token, "/scim/v2/Users", map[string]any{"userName": userName})
		id, _ := created["id"].(string)
		return id
	}
	otherID, babsID := member("other@okta.local"), member("babs@okta.local")
	groupIDs := map[string]string{
		"abf4dd94-a4c0-4f67-89c9-76b03340cb9b": gid, "U0FP0NMEE": gid, "e9e30dba-f08f-4109-8486-d5c6a331660a": gid,
		"23a35c27-23d3-4c03-b4c5-6443c09e7173": uid, "89bb1940-b905-4575-9e7f-6f887cfb368e": otherID,
		"some-member-1": babsID, "some-member-2": otherID,
	}

	w, got = r.send("find-group", nil)
	expectStatus(t, "find-group", w, http.StatusOK)
	if resources := listResources(t, "find-group", got); got["totalResults"] != float64(1) || len(resources) != 1 || resources[0].(map[string]any)["id"] != gid {
		t.Fatalf("find-group: %v, want the created group alone", got)
	}

	// Okta は、照会の引数なしの GET /Groups/{id} に members を返すことを求める。
	w, got = r.send("get-group", groupIDs)
	expectStatus(t, "get-group", w, http.StatusOK)
	if _, present := got["members"]; got["id"] != gid || !present {
		t.Fatalf("get-group: %v, want the group with its members attribute", got)
	}

	w, _ = r.send("rename-group-put", groupIDs)
	expectStatus(t, "rename-group-put", w, http.StatusOK)
	if _, got := r.read("/scim/v2/Groups/" + gid); got["displayName"] != "Tour Guides" || !sameMembers(memberValues(got), babsID, otherID) {
		t.Fatalf("rename-group-put: %v after the PUT, want displayName Tour Guides with the two sent members", got)
	}

	w, _ = r.send("update-membership-put", groupIDs)
	expectStatus(t, "update-membership-put", w, http.StatusOK)
	if _, got := r.read("/scim/v2/Groups/" + gid); got["displayName"] != "Test SCIMv2" || !sameMembers(memberValues(got), uid) {
		t.Fatalf("update-membership-put: %v after the PUT, want only the sent member", got)
	}

	w, _ = r.send("replace-members-patch", groupIDs)
	expectStatus(t, "replace-members-patch", w, http.StatusOK, http.StatusNoContent)
	if _, got := r.read("/scim/v2/Groups/" + gid); !sameMembers(memberValues(got), uid, otherID) {
		t.Fatalf("replace-members-patch: members = %v after the PATCH, want the two sent members", memberValues(got))
	}

	w, _ = r.send("delete-group", groupIDs)
	expectStatus(t, "delete-group", w, http.StatusNoContent)
	if status, _ := r.read("/scim/v2/Groups/" + gid); status != http.StatusNotFound {
		t.Fatalf("delete-group: re-read status %d, want 404", status)
	}

	r.assertAccounted(oktaNotReplayed)
}

func sameMembers(got []string, want ...string) bool {
	return len(got) == len(want) && !slices.ContainsFunc(want, func(id string) bool { return !slices.Contains(got, id) })
}
