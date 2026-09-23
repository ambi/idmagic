package handlers_http_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/audit"
	auditmemory "github.com/ambi/idmagic/backend/audit/db_memory"
	auditports "github.com/ambi/idmagic/backend/audit/ports"
	authdomain "github.com/ambi/idmagic/backend/authentication/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/labstack/echo/v5"
)

// newAuditAdminPaginationServer は既存 newAuditAdminServer と異なり PaginationCodec を
// 設定する。既存 helper は Deps.PaginationCodec が nil のままなので cursor pagination の
// テストには使えない。
func newAuditAdminPaginationServer(t *testing.T, events []*auditports.AuditEventRecord) *echo.Echo {
	t.Helper()
	actor := auditUser("admin", "acme", []string{"admin"})
	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(actor)
	auditStore := auditmemory.NewAuditEventStore(0)
	for _, ev := range events {
		if err := auditStore.Append(t.Context(), ev); err != nil {
			t.Fatalf("append audit event: %v", err)
		}
	}
	resolver := &fakeAuthnResolver{ctx: &authdomain.AuthenticationContext{
		UserID: actor.ID, AuthTime: time.Now().Unix(), AMR: []string{"pwd"},
	}}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:          "http://test",
		TenantRepo:      newSingleTenantRepo(),
		PaginationCodec: support.NewCursorCodec([]byte("test-pagination-secret")),
		UserRepo:        userRepo,
		Audit:           audit.Module{AuditEventRepo: auditStore},
		AuthnResolver:   resolver,
	})
	return e
}

func decodeAuditEventListBody(t *testing.T, body []byte) []map[string]any {
	t.Helper()
	var parsed struct {
		Events []map[string]any `json:"events"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("unmarshal audit events body: %v (body=%s)", err, body)
	}
	return parsed.Events
}

//spec:covers EX-AUDIT-004-01: 先頭ページの応答は絞り込みに一致する正確な総件数・総ページ数・現在のページ・ページサイズを返し、Link ヘッダーに rel="next" のカーソルを含むことを固定する。
func TestAdminAuditEventListSetsLinkHeaderWhenMorePagesExist(t *testing.T) {
	base := time.Now().UTC().Add(-time.Hour)
	events := []*auditports.AuditEventRecord{
		auditEvent("acme", "TypeA", "admin", base),
		auditEvent("acme", "TypeB", "admin", base.Add(time.Minute)),
		auditEvent("acme", "TypeC", "admin", base.Add(2*time.Minute)),
	}
	e := newAuditAdminPaginationServer(t, events)

	resp := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit-events?limit=2")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	link := resp.Header().Get("Link")
	if link == "" || !strings.Contains(link, `rel="next"`) {
		t.Fatalf("expected a rel=next Link header, got %q", link)
	}
	if !strings.Contains(link, `rel="last"`) || resp.Header().Get("Pagination-Total-Items") != "3" || resp.Header().Get("Pagination-Total-Pages") != "2" || resp.Header().Get("Pagination-Current-Page") != "1" || resp.Header().Get("Pagination-Page-Size") != "2" {
		t.Fatalf("unexpected pagination contract: Link=%q headers=%#v", link, resp.Header())
	}
}

func TestAdminAuditEventListOmitsLinkHeaderOnLastPage(t *testing.T) {
	events := []*auditports.AuditEventRecord{
		auditEvent("acme", "TypeA", "admin", time.Now().UTC()),
	}
	e := newAuditAdminPaginationServer(t, events)

	resp := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit-events?limit=200")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if link := resp.Header().Get("Link"); link != "" {
		t.Fatalf("expected no Link header on the last page, got %q", link)
	}
}

//spec:covers EX-AUDIT-004-01: 取得済みのカーソルで次ページを取得すると、直前のページと重複や欠落なく後続のイベントが返ることを固定する。
func TestAdminAuditEventListNextPageContinuesWithoutOverlap(t *testing.T) {
	base := time.Now().UTC().Add(-time.Hour)
	events := []*auditports.AuditEventRecord{
		auditEvent("acme", "TypeA", "admin", base),
		auditEvent("acme", "TypeB", "admin", base.Add(time.Minute)),
		auditEvent("acme", "TypeC", "admin", base.Add(2*time.Minute)),
	}
	e := newAuditAdminPaginationServer(t, events)

	first := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit-events?limit=2")
	link := first.Header().Get("Link")
	if link == "" {
		t.Fatal("expected a Link header on the first page")
	}
	nextPath := link[strings.Index(link, "<")+1 : strings.Index(link, ">")]
	nextPath = strings.TrimPrefix(nextPath, "http://test")

	second := getAdminAuditEvents(e, nextPath)
	if second.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", second.Code, second.Body.String())
	}
	if !strings.Contains(second.Header().Get("Link"), `rel="prev"`) {
		t.Fatalf("second page missing rel=prev: %q", second.Header().Get("Link"))
	}
	firstBody := decodeAuditEventListBody(t, first.Body.Bytes())
	secondBody := decodeAuditEventListBody(t, second.Body.Bytes())
	seen := map[string]bool{}
	for _, ev := range firstBody {
		seen[ev["id"].(string)] = true
	}
	for _, ev := range secondBody {
		id := ev["id"].(string)
		if seen[id] {
			t.Fatalf("event id %q appeared on both pages", id)
		}
	}
	if len(secondBody) == 0 {
		t.Fatal("expected the second page to return at least one event")
	}
}

func TestAdminAuditEventListRejectsInvalidCursor(t *testing.T) {
	e := newAuditAdminPaginationServer(t, nil)
	resp := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit-events?cursor=not-a-real-cursor")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}

//spec:covers EX-AUDIT-004-06: 別テナントで発行されたカーソルは InvalidRequestError で拒否されることを固定する。
func TestAdminAuditEventListRejectsCursorFromAnotherTenant(t *testing.T) {
	e := newAuditAdminPaginationServer(t, nil)
	codec := support.NewCursorCodec([]byte("test-pagination-secret"))
	// auditEventQueryHash はフィルタ無しのリクエスト (?limit=... のみ) では空文字列になる
	// (cursor/limit を除いた残りの query をそのまま hash にするため)。
	foreignCursor, err := codec.Encode(support.Cursor{
		TenantID: "some-other-tenant", QueryHash: "",
		After: "zzz", ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("encode foreign cursor: %v", err)
	}
	resp := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit-events?cursor="+foreignCursor)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}

//spec:covers EX-AUDIT-004-06: 改ざんされたカーソルは署名検証で InvalidRequestError として拒否されることを固定する。
func TestAdminAuditEventListRejectsTamperedCursor(t *testing.T) {
	base := time.Now().UTC().Add(-time.Hour)
	events := []*auditports.AuditEventRecord{
		auditEvent("acme", "TypeA", "admin", base),
		auditEvent("acme", "TypeB", "admin", base.Add(time.Minute)),
		auditEvent("acme", "TypeC", "admin", base.Add(2*time.Minute)),
	}
	e := newAuditAdminPaginationServer(t, events)

	first := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit-events?limit=2")
	link := first.Header().Get("Link")
	nextPath := link[strings.Index(link, "<")+1 : strings.Index(link, ">")]
	nextPath = strings.TrimPrefix(nextPath, "http://test")
	parsed, err := url.Parse(nextPath)
	if err != nil {
		t.Fatalf("parse next link: %v", err)
	}
	cursor := parsed.Query().Get("cursor")
	if cursor == "" {
		t.Fatalf("no cursor in the next link %q", nextPath)
	}
	// "v3." の直後、payload の先頭 1 文字を隣接する別の文字へ書き換える。base64 の
	// フルグループの先頭文字は zero-padding を持たないので、どの文字へ変えても確実に
	// 復号バイト列が変わる (末尾の文字は padding bit のせいで一部の置き換えが無効になる)。
	dot := strings.Index(cursor, ".")
	if dot < 0 || dot+1 >= len(cursor) {
		t.Fatalf("cursor has no payload segment: %q", cursor)
	}
	target := dot + 1
	replacement := byte('A')
	if cursor[target] == 'A' {
		replacement = 'B'
	}
	tamperedCursor := cursor[:target] + string(replacement) + cursor[target+1:]
	q := parsed.Query()
	q.Set("cursor", tamperedCursor)
	parsed.RawQuery = q.Encode()

	resp := getAdminAuditEvents(e, parsed.String())
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("tampered cursor: status=%d body=%s", resp.Code, resp.Body.String())
	}
}

//spec:covers EX-AUDIT-004-06: 旧方式のカーソルが有効期限を超過していれば InvalidRequestError で拒否されることを固定する。
func TestAdminAuditEventListRejectsExpiredLegacyCursor(t *testing.T) {
	e := newAuditAdminPaginationServer(t, nil)
	codec := support.NewCursorCodec([]byte("test-pagination-secret"))
	// auditEventQueryHash はフィルタ無しのリクエスト (?limit=... のみ) では空文字列になる。
	expired, err := codec.Encode(support.Cursor{
		TenantID: "acme", QueryHash: "",
		After: "zzz", ExpiresAt: time.Now().Add(-time.Hour),
	})
	if err != nil {
		t.Fatalf("encode expired cursor: %v", err)
	}
	resp := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit-events?cursor="+expired)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}

//spec:covers EX-AUDIT-004-02: 絞り込みに一致するイベントが 0 件のとき、空の一覧と総件数・総ページ数・現在ページとして 0/0/0 を返し、first/prev/next/last のいずれの Link も返さないことを固定する。
func TestAdminAuditEventListReturnsEmptyPageWhenFilterMatchesNothing(t *testing.T) {
	base := time.Now().UTC().Add(-time.Hour)
	events := []*auditports.AuditEventRecord{
		auditEvent("acme", "TypeA", "admin", base),
		auditEvent("acme", "TypeB", "admin", base.Add(time.Minute)),
		auditEvent("acme", "TypeC", "admin", base.Add(2*time.Minute)),
	}
	e := newAuditAdminPaginationServer(t, events)

	resp := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit-events?limit=2&type=NoSuchType")
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if got := decodeAuditEventListBody(t, resp.Body.Bytes()); len(got) != 0 {
		t.Fatalf("expected 0 events, got %+v", got)
	}
	for header, want := range map[string]string{
		"Pagination-Total-Items": "0", "Pagination-Total-Pages": "0",
		"Pagination-Current-Page": "0", "Pagination-Page-Size": "2",
	} {
		if got := resp.Header().Get(header); got != want {
			t.Fatalf("%s = %q, want %q", header, got, want)
		}
	}
	if link := resp.Header().Get("Link"); link != "" {
		t.Fatalf("expected no Link header when nothing matches, got %q", link)
	}
}

// countFailingAuditRepo は List はそのまま通し、Count だけ常に失敗させる。EX-AUDIT-004-03 の
// 「正確な件数の取得に失敗する」を、他は成功する構成から Count だけ個別に崩して観測するための
// 最小の test double。
type countFailingAuditRepo struct {
	*auditmemory.AuditEventStore
}

func (r *countFailingAuditRepo) Count(context.Context, auditports.AuditEventQuery) (int64, error) {
	return 0, errors.New("count backend unavailable")
}

//spec:covers EX-AUDIT-004-03: 正確な件数の取得に失敗したとき、0 件として成功させず、リクエスト全体をサーバーエラーで失敗させることを固定する。
func TestAdminAuditEventListFailsClosedWhenCountFails(t *testing.T) {
	actor := auditUser("admin", "acme", []string{"admin"})
	userRepo := usermemory.NewUserRepository()
	userRepo.Seed(actor)
	store := auditmemory.NewAuditEventStore(0)
	if err := store.Append(t.Context(), auditEvent("acme", "TypeA", "admin", time.Now().UTC())); err != nil {
		t.Fatalf("append: %v", err)
	}
	resolver := &fakeAuthnResolver{ctx: &authdomain.AuthenticationContext{
		UserID: actor.ID, AuthTime: time.Now().Unix(), AMR: []string{"pwd"},
	}}
	e := echo.New()
	httpadapter.Register(e, httpadapter.Deps{
		Issuer:          "http://test",
		TenantRepo:      newSingleTenantRepo(),
		PaginationCodec: support.NewCursorCodec([]byte("test-pagination-secret")),
		UserRepo:        userRepo,
		Audit:           audit.Module{AuditEventRepo: &countFailingAuditRepo{AuditEventStore: store}},
		AuthnResolver:   resolver,
	})

	resp := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit-events")
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("count failure must fail the whole request as a server error, got status=%d body=%s", resp.Code, resp.Body.String())
	}
}

//spec:covers EX-AUDIT-004-05: category や filter などを変更した状態で、元の絞り込み条件で発行されたカーソルを送ると InvalidRequestError で拒否され、先頭ページからの検索し直しを促すことを固定する。
func TestAdminAuditEventListRejectsCursorWhenFilterChanges(t *testing.T) {
	base := time.Now().UTC().Add(-time.Hour)
	events := []*auditports.AuditEventRecord{
		auditEvent("acme", "TypeA", "admin", base),
		auditEvent("acme", "TypeB", "admin", base.Add(time.Minute)),
		auditEvent("acme", "TypeC", "admin", base.Add(2*time.Minute)),
	}
	e := newAuditAdminPaginationServer(t, events)

	first := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit-events?limit=2")
	link := first.Header().Get("Link")
	nextPath := link[strings.Index(link, "<")+1 : strings.Index(link, ">")]
	nextPath = strings.TrimPrefix(nextPath, "http://test")

	// 同じカーソルのまま、発行時には無かった filter を足して送る。auditEventQueryHash は
	// cursor/limit 以外の query をすべて指紋へ混ぜるため、これだけで指紋が変わる。
	changed := nextPath + "&filter=outcome:eq:failure"
	resp := getAdminAuditEvents(e, changed)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("cursor with a changed filter must be rejected: status=%d body=%s", resp.Code, resp.Body.String())
	}
}

// rel の URL を Link ヘッダーから取り出す。無ければ空文字列を返す。
func linkPath(header, rel string) string {
	for part := range strings.SplitSeq(header, ", ") {
		if !strings.Contains(part, `rel="`+rel+`"`) {
			continue
		}
		start := strings.Index(part, "<") + 1
		end := strings.Index(part, ">")
		return strings.TrimPrefix(part[start:end], "http://test")
	}
	return ""
}

// EX-AUDIT-004-01 の残り: rel="prev"/rel="last"/rel="first" の観測。
//
//spec:covers EX-AUDIT-004-01: rel="prev" のカーソルで前ページへ戻ると正規の時系列降順で返り、rel="last" の終端カーソルは端数を含む最終ページを返し、rel="first" のカーソルを含まない URL は正規の先頭ページを返すことを固定する。
func TestAdminAuditEventListFollowsPrevLastAndFirstLinks(t *testing.T) {
	base := time.Now().UTC().Add(-time.Hour)
	events := make([]*auditports.AuditEventRecord, 5)
	for i := range events {
		events[i] = auditEvent("acme", "Type"+string(rune('A'+i)), "admin", base.Add(time.Duration(i)*time.Minute))
	}
	e := newAuditAdminPaginationServer(t, events)

	first := getAdminAuditEvents(e, "/realms/acme/api/admin/v1/audit-events?limit=2")
	if first.Code != http.StatusOK {
		t.Fatalf("first page: status=%d body=%s", first.Code, first.Body.String())
	}
	firstLink := first.Header().Get("Link")
	if linkPath(firstLink, "prev") != "" {
		t.Fatalf("first page must not have a rel=prev link: %q", firstLink)
	}
	firstBody := decodeAuditEventListBody(t, first.Body.Bytes())
	if len(firstBody) != 2 {
		t.Fatalf("first page: got %d events, want 2: %+v", len(firstBody), firstBody)
	}

	nextPath := linkPath(firstLink, "next")
	if nextPath == "" {
		t.Fatalf("first page missing rel=next: %q", firstLink)
	}
	second := getAdminAuditEvents(e, nextPath)
	if second.Code != http.StatusOK {
		t.Fatalf("second page: status=%d body=%s", second.Code, second.Body.String())
	}
	secondLink := second.Header().Get("Link")
	if second.Header().Get("Pagination-Current-Page") != "2" {
		t.Fatalf("second page current-page = %q, want 2", second.Header().Get("Pagination-Current-Page"))
	}

	// rel="prev" は前ページへ、正規の時系列降順で戻る。
	prevPath := linkPath(secondLink, "prev")
	if prevPath == "" {
		t.Fatalf("second page missing rel=prev: %q", secondLink)
	}
	prevPage := getAdminAuditEvents(e, prevPath)
	if prevPage.Code != http.StatusOK {
		t.Fatalf("prev page: status=%d body=%s", prevPage.Code, prevPage.Body.String())
	}
	prevBody := decodeAuditEventListBody(t, prevPage.Body.Bytes())
	if len(prevBody) != len(firstBody) {
		t.Fatalf("prev page = %+v, want the first page %+v", prevBody, firstBody)
	}
	for i := range firstBody {
		if prevBody[i]["id"] != firstBody[i]["id"] {
			t.Fatalf("prev page order = %+v, want the first page order %+v", prevBody, firstBody)
		}
	}

	// rel="last" は端数を含む最終ページ (5 件を limit=2 で割った残り 1 件) を返す。
	lastPath := linkPath(firstLink, "last")
	if lastPath == "" {
		t.Fatalf("first page missing rel=last: %q", firstLink)
	}
	lastPage := getAdminAuditEvents(e, lastPath)
	if lastPage.Code != http.StatusOK {
		t.Fatalf("last page: status=%d body=%s", lastPage.Code, lastPage.Body.String())
	}
	lastBody := decodeAuditEventListBody(t, lastPage.Body.Bytes())
	if len(lastBody) != 1 {
		t.Fatalf("last page = %+v, want the 1-event remainder", lastBody)
	}
	if lastPage.Header().Get("Pagination-Current-Page") != "3" {
		t.Fatalf("last page current-page = %q, want 3", lastPage.Header().Get("Pagination-Current-Page"))
	}

	// rel="first" はカーソルを含まない URL で、先頭ページの続きになる。
	firstAgainPath := linkPath(secondLink, "first")
	if firstAgainPath == "" {
		t.Fatalf("second page missing rel=first: %q", secondLink)
	}
	if strings.Contains(firstAgainPath, "cursor=") {
		t.Fatalf("rel=first must not carry a cursor: %q", firstAgainPath)
	}
	firstAgain := getAdminAuditEvents(e, firstAgainPath)
	if firstAgain.Code != http.StatusOK {
		t.Fatalf("first-again page: status=%d body=%s", firstAgain.Code, firstAgain.Body.String())
	}
	firstAgainBody := decodeAuditEventListBody(t, firstAgain.Body.Bytes())
	if len(firstAgainBody) != len(firstBody) {
		t.Fatalf("rel=first page = %+v, want the canonical first page %+v", firstAgainBody, firstBody)
	}
	for i := range firstBody {
		if firstAgainBody[i]["id"] != firstBody[i]["id"] {
			t.Fatalf("rel=first page order = %+v, want the canonical first page order %+v", firstAgainBody, firstBody)
		}
	}
}
