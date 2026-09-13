package handlers_http_test

// REQ-IDMANAGEMENT-005 が宣言する具体例のうち、admin_user_list_pagination_test.go の
// 個々のテストでは 1 つの `Then` しか押さえられないものを引き取る。
//
// 005-01 は 11 個の `Then` を 1 本の連なりとして書いている。段ごとに別のテストへ
// 散らすと、「一覧の途中で削除が起きても重複せず読み進められる」という連なりそのものが
// 誰の持ち物でもなくなる。ここは連なりのまま通す。

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	userports "github.com/ambi/idmagic/backend/idmanagement/user/ports"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
)

// adminUserListPage は一覧の 1 ページを、応答から読める形で写し取る。
type adminUserListPage struct {
	usernames  []string
	link       string
	totalItems string
	totalPages string
	page       string
	totalUsers int64
}

func readAdminUserListPage(t *testing.T, response *http.Response, body []byte) adminUserListPage {
	t.Helper()
	page := adminUserListPage{
		link:       response.Header.Get("Link"),
		totalItems: response.Header.Get("Pagination-Total-Items"),
		totalPages: response.Header.Get("Pagination-Total-Pages"),
		page:       response.Header.Get("Pagination-Current-Page"),
	}
	for _, user := range decodeAdminUserListBody(t, body) {
		page.usernames = append(page.usernames, user["preferred_username"].(string))
	}
	var decoded struct {
		TotalUsers int64 `json:"total_users"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("decode total_users: %v", err)
	}
	page.totalUsers = decoded.TotalUsers
	return page
}

func getAdminUserListPage(t *testing.T, e interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}, path string,
) adminUserListPage {
	t.Helper()
	recorder := adminUserListRequest(e, path)
	if recorder.Code != http.StatusOK {
		t.Fatalf("%s status=%d body=%s", path, recorder.Code, recorder.Body.String())
	}
	return readAdminUserListPage(t, recorder.Result(), recorder.Body.Bytes())
}

// **一覧の途中で 1 件消える。** そこが具体例の要点である。削除を挟まないページングは、
// カーソルを行位置 (offset) で持つ実装でも通ってしまう。安定したキーセットで
// 読み進めていることは、消えた行を挟んでも重複も取りこぼしも起きないことでしか読めない。
//
//spec:covers EX-IDMANAGEMENT-005-01: 先頭ページの件数とカーソル、途中の削除が一覧から外れること、next / prev / last / 先頭ページのそれぞれが正規の並び順で重複なく返ること。
func TestAdminUserListPagesStablyAcrossAMidListDeletion(t *testing.T) {
	e, repo := newAdminUserHandler(t)
	now := time.Now().UTC()
	// admin と regular は fixture が置く。合わせて 6 人になる。
	for _, name := range []string{"charlie", "delta", "echo", "foxtrot"} {
		repo.Seed(&userdomain.User{
			ID: name + "-id", PreferredUsername: name, PasswordHash: "unused",
			Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
			CreatedAt: now, UpdatedAt: now,
		})
	}

	first := getAdminUserListPage(t, e, "/api/admin/v1/users?limit=2")
	if first.totalItems != "6" || first.totalPages != "3" || first.page != "1" || first.totalUsers != 6 {
		t.Fatalf("先頭ページのメタデータ = %+v", first)
	}
	if !strings.Contains(first.link, `rel="next"`) || !strings.Contains(first.link, "cursor=") {
		t.Fatalf("先頭ページの Link にコンパクトな next カーソルが無い: %q", first.link)
	}
	nextPath := linkURLForRel(t, first.link, "next")

	// 一覧の途中で、他の管理者が 2 ページ目に居るはずのユーザーを 1 件削除する。
	deleted := deleteAdminListUser(t, repo, "delta")
	if deleted == "" {
		t.Fatal("削除対象を選べなかった")
	}

	second := getAdminUserListPage(t, e, nextPath)
	// 削除された行は一覧対象から外れ、既に返した行とも重ならない。
	for _, name := range second.usernames {
		if name == deleted {
			t.Fatalf("削除されたユーザーが 2 ページ目に現れた: %v", second.usernames)
		}
		for _, seen := range first.usernames {
			if name == seen {
				t.Fatalf("%q が 1 ページ目と 2 ページ目の両方に現れた", name)
			}
		}
	}
	if !strings.Contains(second.link, `rel="prev"`) {
		t.Fatalf("2 ページ目の Link に prev が無い: %q", second.link)
	}

	// prev は 1 ページ目を正規の並び順で返す。
	previous := getAdminUserListPage(t, e, linkURLForRel(t, second.link, "prev"))
	if !sameUsernames(previous.usernames, first.usernames) {
		t.Fatalf("prev = %v, want %v", previous.usernames, first.usernames)
	}

	// last は端数のページを返す。削除で 5 人になったので 3 ページ目は 1 人である。
	last := getAdminUserListPage(t, e, linkURLForRel(t, second.link, "last"))
	if len(last.usernames) != 1 || last.page != "3" || last.totalPages != "3" {
		t.Fatalf("last = %+v, want the 1-row remainder on page 3", last)
	}
	if strings.Contains(last.link, `rel="last"`) || !strings.Contains(last.link, `rel="first"`) {
		t.Fatalf("最終ページの Link = %q", last.link)
	}

	// カーソルを含まない URL は正規の先頭ページへ戻る。
	restart := getAdminUserListPage(t, e, linkURLForRel(t, last.link, "first"))
	if !sameUsernames(restart.usernames, first.usernames) {
		t.Fatalf("first = %v, want %v", restart.usernames, first.usernames)
	}
}

// deleteAdminListUser は一覧の途中で起きる削除を、他の管理者の操作として起こす。
// 完全削除は tombstone を残し、一覧の対象からは外れる。
func deleteAdminListUser(t *testing.T, repo *usermemory.UserRepository, username string) string {
	t.Helper()
	user, err := repo.FindByUsername(context.Background(), "", username)
	if err != nil || user == nil {
		t.Fatalf("FindByUsername(%s) = (%+v, %v)", username, user, err)
	}
	removed := *user
	removed.Lifecycle.Status = idmdomain.UserStatusDeleted
	repo.Seed(&removed)
	return username
}

func sameUsernames(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

// 具体例は空の結果について「`0 / 0 / 0`」と「4 つの `Link` を返さない」の 2 つを言う。
// **`Link` を返さないことまで読む。** 0 件でも next を返す実装は、ページャーを
// 無限に回してしまう。
//
//spec:covers EX-IDMANAGEMENT-005-03: 条件に一致する User が 0 件のとき、一覧が空で総項目数・総ページ数・現在ページが 0 / 0 / 0 になり、first / prev / next / last の Link を 1 つも返さないこと。
func TestAdminUserListWithNoMatchesReturnsZeroCountsAndNoLinks(t *testing.T) {
	e, _ := newAdminUserHandler(t)

	page := getAdminUserListPage(t, e, "/api/admin/v1/users?query=nobody-matches-this&limit=2")
	if len(page.usernames) != 0 {
		t.Fatalf("users = %v, want empty", page.usernames)
	}
	if page.totalItems != "0" || page.totalPages != "0" || page.page != "0" {
		t.Fatalf("counts = %s / %s / %s, want 0 / 0 / 0", page.totalItems, page.totalPages, page.page)
	}
	for _, rel := range []string{"first", "prev", "next", "last"} {
		if strings.Contains(page.link, `rel="`+rel+`"`) {
			t.Fatalf("空の一覧が rel=%s を返した: %q", rel, page.link)
		}
	}
}

// countFailingUserRepository は正確な件数の取得だけを失敗させる。行の取得は成功する
// ので、「0 件として成功させる」実装はここで 200 を返してしまう。
type countFailingUserRepository struct {
	userports.UserRepository
	err error
}

func (r countFailingUserRepository) Count(context.Context, string) (int64, error) {
	return 0, r.err
}

func (r countFailingUserRepository) CountFiltered(
	context.Context, string, string, *idmdomain.UserStatus,
) (int64, error) {
	return 0, r.err
}

// **失敗を 0 件に読み替えないことが具体例の要点である。** 件数の取得だけが落ちたとき、
// 一覧そのものは返せてしまう。空の一覧を 200 で返す実装は、管理者に「このテナントに
// 利用者は居ない」と伝えることになる。
//
//spec:covers EX-IDMANAGEMENT-005-04: 正確な件数の取得が失敗したとき、0 件として成功させずリクエスト全体をサーバーエラーで失敗させること。
func TestAdminUserListFailsTheRequestWhenTheCountCannotBeRead(t *testing.T) {
	var repo *usermemory.UserRepository
	e, seeded := newAdminUserHandler(t, func(deps *httpadapter.Deps) {
		repo = deps.UserRepo.(*usermemory.UserRepository)
		deps.UserRepo = countFailingUserRepository{
			UserRepository: repo, err: errAdminUserListCountUnavailable,
		}
	})
	_ = seeded

	response := adminUserListRequest(e, "/api/admin/v1/users?limit=2")
	if response.Code == http.StatusOK {
		t.Fatalf("件数の取得が落ちたのに status=200 body=%s", response.Body.String())
	}
	if response.Code < http.StatusInternalServerError {
		t.Fatalf("status=%d, want a server error", response.Code)
	}
	// 応答が利用者を 1 人も運んでいないこと。件数だけ諦めて一覧を返す実装を落とす。
	if strings.Contains(response.Body.String(), `"preferred_username"`) {
		t.Fatalf("失敗した応答が一覧を運んだ: %s", response.Body.String())
	}
}

var errAdminUserListCountUnavailable = errAdminUserListCount{}

type errAdminUserListCount struct{}

func (errAdminUserListCount) Error() string { return "the exact count is unavailable" }
