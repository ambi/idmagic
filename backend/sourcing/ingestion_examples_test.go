package sourcing_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"

	"github.com/ambi/idmagic/backend/authentication"
	"github.com/ambi/idmagic/backend/idmanagement"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/shared/http/server_http/sourcingdeps"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	"github.com/ambi/idmagic/backend/shared/spec"
	"github.com/ambi/idmagic/backend/sourcing"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"
	tenantports "github.com/ambi/idmagic/backend/tenancy/ports"
)

const testIssuer = "http://idp.test"

// otherTenant は、X-Test-Realm: other の要求が解決するテナント。
const otherTenant = "00000000-0000-0000-0000-0000000000b2"

// ingestion は、本番と同じ部品で Sourcing を組み立て、Connector と管理者の要求を送る。
type ingestion struct {
	t        *testing.T
	e        *echo.Echo
	identity idmanagement.Module
	module   sourcing.Module
	quotas   tenantports.QuotaRepository
	mu       sync.Mutex
	events   []spec.DomainEvent
}

func newIngestion(t *testing.T) *ingestion {
	t.Helper()
	h := &ingestion{t: t, identity: idmanagement.NewMemoryModule()}
	for _, u := range []userdomain.User{
		{ID: "admin", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "admin", Roles: []string{"admin"}},
		{ID: "member", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "member"},
		{ID: "other-admin", TenantID: otherTenant, PreferredUsername: "admin", Roles: []string{"admin"}},
		{ID: "taken", TenantID: tenancydomain.DefaultTenantID, PreferredUsername: "taken"},
	} {
		u.PasswordHash, u.CreatedAt, u.UpdatedAt = "unused", time.Now(), time.Now()
		u.Lifecycle = userdomain.UserLifecycle{Status: idmdomain.UserStatusActive}
		if err := h.identity.UserRepo.Save(context.Background(), &u); err != nil {
			t.Fatal(err)
		}
	}
	emit := func(event spec.DomainEvent) {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.events = append(h.events, event)
	}
	h.quotas = tenancy.NewMemoryModule().QuotaRepo
	lifecycle := idmanagement.LifecycleDeps{QuotaRepo: h.quotas}
	module := sourcing.NewMemoryModule()
	module.DirectoryUserLifecycle = h.identity.UserLifecycleCommands(lifecycle, func(event spec.DomainEvent) error { emit(event); return nil }, "directory_connector")
	module.DirectoryWrites = sourcingdeps.NewDirectoryWrites(h.identity, lifecycle, module, emit)
	module.Writes = sourcingdeps.NewWrites(h.identity, lifecycle, module, emit)
	authenticator := &support.Authenticator{
		Principals: h.identity.Principals(),
		Sessions:   authentication.Module{AuthnResolver: authentication.DemoHeaderResolver{}}.SessionAuthentications(),
	}
	h.module = module
	h.e = echo.New()
	h.e.HTTPErrorHandler = support.ErrorHandler(nil, nil)
	module.Register(h.e.Group("", testRealm), support.Deps{Issuer: testIssuer}, authenticator,
		h.identity.UserRepo, h.identity.GroupRepo, emit, nil, &memoryReplay{seen: map[string]bool{}})
	return h
}

type memoryReplay struct {
	mu   sync.Mutex
	seen map[string]bool
}

func (s *memoryReplay) RecordIfNew(_ context.Context, jti string, _ int, _ time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.seen[jti] {
		return false, nil
	}
	s.seen[jti] = true
	return true, nil
}

// testRealm は、テナントの解決の代わりに X-Test-Realm の値でテナントを選ぶ。
func testRealm(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		id, realm := tenancydomain.DefaultTenantID, tenancydomain.DefaultRealm
		if c.Request().Header.Get("X-Test-Realm") == "other" {
			id, realm = otherTenant, "other"
		}
		prefix := "/realms/" + realm
		tenant := &tenancydomain.Tenant{ID: id, Realm: realm, Status: tenancydomain.TenantStatusActive, EndpointStyle: tenancydomain.TenantEndpointStylePath}
		c.SetRequest(c.Request().WithContext(tenantports.WithTenant(c.Request().Context(), tenant, testIssuer+prefix, prefix)))
		return next(c)
	}
}

type request struct {
	method, path string
	body         any
	admin        string // 管理画面の利用者
	connector    *connector
	realm        string
}

func (h *ingestion) do(r request) *httptest.ResponseRecorder {
	h.t.Helper()
	var payload []byte
	if r.body != nil {
		var err error
		if payload, err = json.Marshal(r.body); err != nil {
			h.t.Fatal(err)
		}
	}
	req := httptest.NewRequest(r.method, r.path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if r.admin != "" {
		req.Header.Set("X-Demo-Sub", r.admin)
		req.Header.Set("Origin", testIssuer)
		req.Header.Set(support.CSRFHeader, "csrf")
		req.AddCookie(&http.Cookie{Name: support.CSRFCookie, Value: "csrf"})
	}
	if r.connector != nil {
		req.Header.Set("Authorization", "Bearer "+r.connector.sign(h.t))
	}
	if r.realm != "" {
		req.Header.Set("X-Test-Realm", r.realm)
	}
	rec := httptest.NewRecorder()
	h.e.ServeHTTP(rec, req)
	return rec
}

func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(rec.Body.Bytes(), &value); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
	return value
}

func expectStatus(t *testing.T, rec *httptest.ResponseRecorder, status int, problem string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, status, rec.Body.String())
	}
	if problem != "" {
		body := decodeBody[map[string]any](t, rec)
		if body["type"] != "urn:idmagic:error:"+problem {
			t.Fatalf("problem = %v, want %s", body["type"], problem)
		}
	}
}

type connector struct {
	id, keyID, audience string
	private             *ecdsa.PrivateKey
}

// sign は Connector が作る ES256 の JWT を返す。
func (c *connector) sign(t *testing.T) string {
	t.Helper()
	encode := func(value any) string {
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	input := encode(map[string]any{"alg": "ES256", "typ": "JWT", "kid": c.keyID}) + "." + encode(map[string]any{
		"iss": c.id, "sub": c.id, "aud": c.audience, "exp": time.Now().Add(5 * time.Minute).Unix(), "jti": uuid.NewString(),
	})
	digest := sha256.Sum256([]byte(input))
	r, s, err := ecdsa.Sign(rand.Reader, c.private, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(append(fixedBytes(r), fixedBytes(s)...))
}

func fixedBytes(n *big.Int) []byte {
	out := make([]byte, 32)
	n.FillBytes(out)
	return out
}

// enrolled は既定のテナントの管理者が作った Connector を登録し、呼び出せる Connector を返す。
func (h *ingestion) enrolled() *connector {
	h.t.Helper()
	created := h.do(request{method: http.MethodPost, path: "/api/admin/v1/directory-connectors", body: map[string]any{"display_name": "AD"}, admin: "admin"})
	expectStatus(h.t, created, http.StatusCreated, "")
	grant := decodeBody[struct {
		EnrollmentToken string `json:"enrollment_token"`
	}](h.t, created)
	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		h.t.Fatal(err)
	}
	point, err := private.PublicKey.Bytes()
	if err != nil {
		h.t.Fatal(err)
	}
	jwk := map[string]any{
		"kty": "EC", "crv": "P-256",
		"x": base64.RawURLEncoding.EncodeToString(point[1:33]), "y": base64.RawURLEncoding.EncodeToString(point[33:65]),
	}
	enrollment := h.do(request{
		method: http.MethodPost, path: "/directory-connector/v1/enrollment",
		body: map[string]any{"enrollment_token": grant.EnrollmentToken, "jwk": jwk},
	})
	expectStatus(h.t, enrollment, http.StatusOK, "")
	enrolled := decodeBody[struct {
		ConnectorID string `json:"connector_id"`
		KeyID       string `json:"key_id"`
		Audience    string `json:"audience"`
	}](h.t, enrollment)
	return &connector{id: enrolled.ConnectorID, keyID: enrolled.KeyID, audience: enrolled.Audience, private: private}
}

func (h *ingestion) startRun(c *connector, mode string) string {
	h.t.Helper()
	rec := h.do(request{method: http.MethodPost, path: "/directory-connector/v1/ingestion-runs", body: map[string]any{"mode": mode}, connector: c})
	expectStatus(h.t, rec, http.StatusCreated, "")
	run := decodeBody[runJSON](h.t, rec)
	if run.Status != "running" || run.Mode != mode {
		h.t.Fatalf("started run = %+v", run)
	}
	return run.ID
}

type item = map[string]any

type result struct {
	ExternalID string `json:"external_id"`
	Outcome    string `json:"outcome"`
	Error      string `json:"error,omitempty"`
}

func (h *ingestion) sendBatch(c *connector, runID, batchID string, items ...item) *httptest.ResponseRecorder {
	h.t.Helper()
	return h.do(request{
		method: http.MethodPost, path: "/directory-connector/v1/ingestion-runs/" + runID + "/user-batches",
		body: map[string]any{"batch_id": batchID, "items": items}, connector: c,
	})
}

// apply はバッチを送り、項目ごとの結果を返す。
func (h *ingestion) apply(c *connector, runID, batchID string, items ...item) []result {
	h.t.Helper()
	rec := h.sendBatch(c, runID, batchID, items...)
	expectStatus(h.t, rec, http.StatusOK, "")
	body := decodeBody[struct {
		BatchID string   `json:"batch_id"`
		Results []result `json:"results"`
	}](h.t, rec)
	if body.BatchID != batchID {
		h.t.Fatalf("batch_id = %q, want %q", body.BatchID, batchID)
	}
	return body.Results
}

func (h *ingestion) complete(c *connector, runID string, itemCount int) *httptest.ResponseRecorder {
	h.t.Helper()
	return h.do(request{
		method: http.MethodPost, path: "/directory-connector/v1/ingestion-runs/" + runID + "/complete",
		body: map[string]any{"item_count": itemCount}, connector: c,
	})
}

type runJSON struct {
	ID          string     `json:"id"`
	Mode        string     `json:"mode"`
	Status      string     `json:"status"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	Counts      struct {
		Batches, Items, Created, Updated, Unchanged, Deleted, Failed, Disabled int
		DisabledComputers                                                      int `json:"disabled_computers"`
	} `json:"counts"`
	Failures []struct {
		BatchID    string `json:"batch_id"`
		ExternalID string `json:"external_id"`
		Error      string `json:"error"`
	} `json:"failures"`
}

func (h *ingestion) user(username string) *userdomain.User {
	h.t.Helper()
	user, err := h.identity.UserRepo.FindByUsername(context.Background(), tenancydomain.DefaultTenantID, username)
	if err != nil {
		h.t.Fatal(err)
	}
	return user
}

func (h *ingestion) status(username string) idmdomain.UserStatus {
	h.t.Helper()
	user := h.user(username)
	if user == nil {
		h.t.Fatalf("user %q does not exist", username)
	}
	return user.Lifecycle.EffectiveStatus()
}

func (h *ingestion) eventCount(eventType string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	count := 0
	for _, event := range h.events {
		if event.EventType() == eventType {
			count++
		}
	}
	return count
}

func expectResults(t *testing.T, got []result, want ...result) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("results = %+v, want %+v", got, want)
	}
}

func alice(username string) item {
	return item{
		"external_id": "guid-alice", "preferred_username": username, "name": "Alice", "given_name": "Alice",
		"family_name": "Liddell", "email": "alice@example.test", "department": "Sales", "employee_number": "E1",
	}
}

// 衝突と削除済みを失敗として返し、本文の誤りと別の Connector の実行を拒否する。
//
//spec:covers REQ-SOURCING-015, REQ-SOURCING-016: 外部の不変 ID で作成、名前の変更、無効化と再有効化、削除の予約を同じ User へ反映し、
func TestDirectoryUserBatchesCorrelateByExternalID(t *testing.T) {
	h := newIngestion(t)
	ad := h.enrolled()
	run := h.startRun(ad, "incremental")

	expectResults(t, h.apply(ad, run, "b1",
		alice("alice"),
		item{"external_id": "guid-bob", "preferred_username": "bob", "enabled": false},
		item{"external_id": "guid-carol", "preferred_username": "taken"},
	),
		result{"guid-alice", "created", ""}, result{"guid-bob", "created", ""}, result{"guid-carol", "failed", "username_conflict"})
	created := h.user("alice")
	if created == nil || created.Lifecycle.EffectiveStatus() != idmdomain.UserStatusActive || created.PasswordHash != "" ||
		created.Email == nil || *created.Email != "alice@example.test" || created.Attributes["department"].String == nil {
		t.Fatalf("created alice = %+v", created)
	}
	if h.status("bob") != idmdomain.UserStatusDisabled {
		t.Fatalf("bob = %s, want disabled", h.status("bob"))
	}
	if taken := h.user("taken"); taken.Name != nil {
		t.Fatalf("an unmapped user with the same username changed: %+v", taken)
	}
	conflicting := item{"external_id": "guid-dave", "preferred_username": "dave", "email": "alice@example.test"}
	expectResults(t, h.apply(ad, run, "b1-email", conflicting), result{"guid-dave", "failed", "email_conflict"})
	if h.user("dave") != nil {
		t.Fatalf("an item with a conflicting email created a user")
	}
	h.mu.Lock()
	for _, event := range h.events {
		if created, ok := event.(*idmdomain.UserCreated); ok && created.ActorUserID != "directory_connector" {
			t.Errorf("UserCreated actor = %q", created.ActorUserID)
		}
	}
	h.mu.Unlock()

	renamed := alice("alice.liddell")
	delete(renamed, "department")
	expectResults(t, h.apply(ad, run, "b2",
		renamed,
		item{"external_id": "guid-bob", "preferred_username": "bob", "enabled": true},
		item{"external_id": "guid-unknown", "deleted": true},
	),
		result{"guid-alice", "updated", ""}, result{"guid-bob", "updated", ""}, result{"guid-unknown", "unchanged", ""})
	moved := h.user("alice.liddell")
	if moved == nil || moved.ID != created.ID || h.user("alice") != nil {
		t.Fatalf("renamed alice = %+v, want the same user %s", moved, created.ID)
	}
	if _, ok := moved.Attributes["department"]; ok {
		t.Fatalf("an attribute missing from the item was kept: %+v", moved.Attributes)
	}
	if h.status("bob") != idmdomain.UserStatusActive {
		t.Fatalf("bob = %s, want active", h.status("bob"))
	}

	updates := h.eventCount("UserUpdated")
	expectResults(t, h.apply(ad, run, "b3", renamed, item{"external_id": "guid-bob", "deleted": true}),
		result{"guid-alice", "unchanged", ""}, result{"guid-bob", "deleted", ""})
	if h.eventCount("UserUpdated") != updates {
		t.Fatalf("an unchanged item emitted UserUpdated")
	}
	if h.status("bob") != idmdomain.UserStatusPendingDeletion {
		t.Fatalf("bob = %s, want pending_deletion", h.status("bob"))
	}
	expectResults(t, h.apply(ad, run, "b4",
		item{"external_id": "guid-bob", "preferred_username": "bob"},
		item{"external_id": "guid-bob-again", "deleted": true},
	),
		result{"guid-bob", "failed", "user_deleted"}, result{"guid-bob-again", "unchanged", ""})

	// 別の Connector の対応は別であり、その User とはユーザー名が衝突する。
	other := h.enrolled()
	otherRun := h.startRun(other, "incremental")
	expectResults(t, h.apply(other, otherRun, "b1", alice("alice.liddell")), result{"guid-alice", "failed", "username_conflict"})
	expectStatus(t, h.sendBatch(other, run, "b9", alice("x")), http.StatusNotFound, "ingestion_run_not_found")

	tooMany := make([]item, 101)
	for n := range tooMany {
		tooMany[n] = item{"external_id": fmt.Sprintf("g%d", n), "preferred_username": fmt.Sprintf("u%d", n)}
	}
	for name, items := range map[string][]item{
		"too many items":        tooMany,
		"no items":              {},
		"duplicate external id": {alice("a"), alice("b")},
		"missing username":      {{"external_id": "guid-x"}},
		"empty external id":     {{"external_id": "", "preferred_username": "x"}},
	} {
		if rec := h.sendBatch(ad, run, "invalid", items...); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d body=%s", name, rec.Code, rec.Body.String())
		}
	}
	if h.user("u0") != nil || h.user("a") != nil {
		t.Fatalf("a refused batch applied an item")
	}
	rec := h.do(request{method: http.MethodPost, path: "/directory-connector/v1/ingestion-runs", body: map[string]any{"mode": "partial"}, connector: ad})
	expectStatus(t, rec, http.StatusBadRequest, "invalid_request")
	rec = h.do(request{method: http.MethodPost, path: "/directory-connector/v1/ingestion-runs", body: map[string]any{"mode": "full"}})
	expectStatus(t, rec, http.StatusUnauthorized, "invalid_token")
}

// 内容の違う同じ ID と閉じた実行への新しいバッチを拒否する。
//
//spec:covers REQ-SOURCING-016: User の Hard Quota を超える作成の項目は、User と対応を作らずに quota_exceeded で失敗し、後の項目の適用を続ける。
func TestUserItemsBeyondTheQuotaFail(t *testing.T) {
	h := newIngestion(t)
	if err := h.quotas.SetQuota(t.Context(), tenancydomain.DefaultTenantID, &tenancydomain.TenantQuota{Users: new(1)}); err != nil {
		t.Fatal(err)
	}
	ad := h.enrolled()
	run := h.startRun(ad, "incremental")
	expectResults(t, h.apply(ad, run, "b1",
		item{"external_id": "g1", "preferred_username": "first"},
		item{"external_id": "g2", "preferred_username": "second"},
		item{"external_id": "g3", "deleted": true},
	), result{"g1", "created", ""}, result{"g2", "failed", "quota_exceeded"}, result{"g3", "unchanged", ""})
	if h.user("second") != nil {
		t.Fatalf("an item beyond the quota created a user")
	}
	managed, err := h.module.UserOwnershipGuard().SourceManagedUserIDs(t.Context(), tenancydomain.DefaultTenantID, []string{h.user("first").ID})
	if err != nil || len(managed) != 1 {
		t.Fatalf("correlations = %v, %v", managed, err)
	}
}

//spec:covers REQ-SOURCING-017: 同じバッチの再送は最初の結果を返して User、イベント、件数を変えず、
func TestResentUserBatchReturnsTheFirstResults(t *testing.T) {
	h := newIngestion(t)
	ad := h.enrolled()
	run := h.startRun(ad, "incremental")

	first := h.apply(ad, run, "b1", alice("alice"))
	expectResults(t, first, result{"guid-alice", "created", ""})
	expectResults(t, h.apply(ad, run, "b1", alice("alice")), first...)
	if h.eventCount("UserCreated") != 1 {
		t.Fatalf("UserCreated = %d, want 1", h.eventCount("UserCreated"))
	}
	expectStatus(t, h.sendBatch(ad, run, "b1", alice("alice.liddell")), http.StatusConflict, "batch_conflict")
	if h.user("alice.liddell") != nil {
		t.Fatalf("a conflicting resend was applied")
	}

	expectStatus(t, h.complete(ad, run, 1), http.StatusOK, "")
	expectResults(t, h.apply(ad, run, "b1", alice("alice")), first...)
	expectStatus(t, h.sendBatch(ad, run, "b2", alice("alice.liddell")), http.StatusConflict, "ingestion_run_closed")
	if h.user("alice.liddell") != nil {
		t.Fatalf("a batch for a completed run was applied")
	}

	detail := h.do(request{method: http.MethodGet, path: "/api/admin/v1/directory-connectors/" + ad.id + "/ingestion-runs/" + run, admin: "admin"})
	expectStatus(t, detail, http.StatusOK, "")
	if counts := decodeBody[runJSON](t, detail).Counts; counts.Batches != 1 || counts.Items != 1 || counts.Created != 1 {
		t.Fatalf("counts after resends = %+v", counts)
	}
}

// 差分の完了、項目数の合わない完了、打ち切られた実行の完了は User を無効にしない。
//
//spec:covers REQ-SOURCING-015, REQ-SOURCING-018: 完全な再同期の完了は、その実行に現れなかった同じ Connector の対応の User だけを無効にし、
func TestFullResyncCompletionDisablesOnlyUnseenUsers(t *testing.T) {
	h := newIngestion(t)
	ad := h.enrolled()
	other := h.enrolled()
	seed := h.startRun(ad, "incremental")
	h.apply(ad, seed, "b1",
		item{"external_id": "g1", "preferred_username": "kept"},
		item{"external_id": "g2", "preferred_username": "gone"},
		item{"external_id": "g3", "preferred_username": "broken"},
		item{"external_id": "g4", "preferred_username": "removed"},
		item{"external_id": "g5", "preferred_username": "stopped", "enabled": false},
	)
	h.apply(ad, seed, "b2", item{"external_id": "g4", "deleted": true})
	otherRun := h.startRun(other, "incremental")
	h.apply(other, otherRun, "b1", item{"external_id": "g2", "preferred_username": "elsewhere"})
	expectStatus(t, h.complete(ad, seed, 6), http.StatusOK, "")
	partial := h.startRun(ad, "incremental")
	h.apply(ad, partial, "b1", item{"external_id": "g1", "preferred_username": "kept"})
	expectStatus(t, h.complete(ad, partial, 1), http.StatusOK, "")
	for _, name := range []string{"kept", "gone", "broken", "elsewhere"} {
		if h.status(name) != idmdomain.UserStatusActive {
			t.Fatalf("an incremental completion changed %s to %s", name, h.status(name))
		}
	}

	abandoned := h.startRun(ad, "full")
	full := h.startRun(ad, "full")
	expectStatus(t, h.complete(ad, abandoned, 0), http.StatusConflict, "ingestion_run_closed")
	// g3 は失敗しても、実行に現れた対応として無効にしない。
	expectResults(t, h.apply(ad, full, "b1",
		item{"external_id": "g1", "preferred_username": "kept"},
		item{"external_id": "g3", "preferred_username": "taken"},
	), result{"g1", "unchanged", ""}, result{"g3", "failed", "username_conflict"})

	expectStatus(t, h.complete(ad, full, -1), http.StatusBadRequest, "invalid_request")
	expectStatus(t, h.complete(ad, full, 3), http.StatusConflict, "ingestion_run_incomplete")
	if h.status("gone") != idmdomain.UserStatusActive {
		t.Fatalf("an incomplete run disabled gone")
	}
	rec := h.complete(ad, full, 2)
	expectStatus(t, rec, http.StatusOK, "")
	if completed := decodeBody[runJSON](t, rec); completed.Status != "completed" || completed.CompletedAt == nil {
		t.Fatalf("completed run = %+v", completed)
	}
	want := map[string]idmdomain.UserStatus{
		"gone": idmdomain.UserStatusDisabled, "kept": idmdomain.UserStatusActive, "broken": idmdomain.UserStatusActive,
		"removed": idmdomain.UserStatusPendingDeletion, "stopped": idmdomain.UserStatusDisabled,
		"elsewhere": idmdomain.UserStatusActive, "taken": idmdomain.UserStatusActive,
	}
	for name, status := range want {
		if h.status(name) != status {
			t.Errorf("%s = %s, want %s", name, h.status(name), status)
		}
	}
	expectStatus(t, h.complete(ad, full, 2), http.StatusOK, "")

	detail := h.do(request{method: http.MethodGet, path: "/api/admin/v1/directory-connectors/" + ad.id + "/ingestion-runs/" + full, admin: "admin"})
	expectStatus(t, detail, http.StatusOK, "")
	if counts := decodeBody[runJSON](t, detail).Counts; counts.Disabled != 1 {
		t.Fatalf("disabled = %d, want 1", counts.Disabled)
	}
}

//spec:covers REQ-IDMANAGEMENT-004: Directory Connector が取り込んだ User は、CSV のインポートが変更しない取り込み元の管理下の User になる。
func TestDirectoryUsersAreSourceManaged(t *testing.T) {
	h := newIngestion(t)
	ad := h.enrolled()
	run := h.startRun(ad, "incremental")
	h.apply(ad, run, "b1", alice("alice"))
	ingested := h.user("alice")
	managed, err := h.module.UserOwnershipGuard().SourceManagedUserIDs(t.Context(), tenancydomain.DefaultTenantID, []string{ingested.ID, "taken"})
	if err != nil {
		t.Fatal(err)
	}
	if !managed[ingested.ID] || managed["taken"] {
		t.Fatalf("source managed = %v, want only %s", managed, ingested.ID)
	}
	other, err := h.module.UserOwnershipGuard().SourceManagedUserIDs(t.Context(), otherTenant, []string{ingested.ID})
	if err != nil || other[ingested.ID] {
		t.Fatalf("another tenant sees the correlation: %v, %v", other, err)
	}
}

//spec:covers REQ-SOURCING-019: 管理者は実行の一覧で結果ごとの件数を、詳細で失敗した項目を読み、別のテナントと権限のない利用者は拒否される。
//verify:operation ListDirectoryIngestionRuns: 200 で新しい順の実行と結果ごとの件数を、別のテナントに 404、権限のない利用者に 403 を返す。
//verify:operation GetDirectoryIngestionRun: 200 で件数と失敗した項目を、存在しない実行に 404 ingestion_run_not_found を返す。
func TestAdministratorsReadIngestionRunsAndFailures(t *testing.T) {
	h := newIngestion(t)
	ad := h.enrolled()
	older := h.startRun(ad, "incremental")
	h.apply(ad, older, "b1", item{"external_id": "g1", "preferred_username": "one"})
	run := h.startRun(ad, "full")
	h.apply(ad, run, "b1", item{"external_id": "g1", "preferred_username": "one"}, item{"external_id": "g2", "preferred_username": "taken"})
	h.apply(ad, run, "b2", item{"external_id": "g3", "preferred_username": "three"}, item{"external_id": "g9", "deleted": true})

	base := "/api/admin/v1/directory-connectors/" + ad.id + "/ingestion-runs"
	list := h.do(request{method: http.MethodGet, path: base, admin: "admin"})
	expectStatus(t, list, http.StatusOK, "")
	runs := decodeBody[struct {
		Runs []runJSON `json:"runs"`
	}](t, list).Runs
	if len(runs) != 2 || runs[0].ID != run || runs[1].ID != older || runs[1].Status != "superseded" {
		t.Fatalf("runs = %+v", runs)
	}
	counts := runs[0].Counts
	if counts.Batches != 2 || counts.Items != 4 || counts.Created != 1 || counts.Unchanged != 2 || counts.Failed != 1 || counts.Updated != 0 || counts.Deleted != 0 {
		t.Fatalf("counts = %+v", counts)
	}

	detail := h.do(request{method: http.MethodGet, path: base + "/" + run, admin: "admin"})
	expectStatus(t, detail, http.StatusOK, "")
	failures := decodeBody[runJSON](t, detail).Failures
	if len(failures) != 1 || failures[0].BatchID != "b1" || failures[0].ExternalID != "g2" || failures[0].Error != "username_conflict" {
		t.Fatalf("failures = %+v", failures)
	}

	expectStatus(t, h.do(request{method: http.MethodGet, path: base, admin: "other-admin", realm: "other"}), http.StatusNotFound, "directory_connector_not_found")
	expectStatus(t, h.do(request{method: http.MethodGet, path: base + "/" + uuid.NewString(), admin: "admin"}), http.StatusNotFound, "ingestion_run_not_found")
	expectStatus(t, h.do(request{method: http.MethodGet, path: base, admin: "member"}), http.StatusForbidden, "access_denied")
}
