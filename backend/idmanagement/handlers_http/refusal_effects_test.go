package handlers_http_test

// IdentityManagement が宣言する拒否について、応答と「拒否が変えなかった状態」の
// 両方を確かめる。
//
// 拒否の応答は防護とは別の分岐で書き出されるため、ステータスとエラー種別だけを読む
// テストは「拒否を書き、そのうえで操作も続ける」実装をそのまま通してしまう。実際に
// それが出荷され、レビューと行カバレッジを素通りしたのが wi-390 の欠陥だった。
//
// 入口は production と同じ HTTP の境界に統一する。ここで守られているのは Cookie、
// Origin、CSRF、粒度スコープ、ロール、テナントの判定であり、use case を直接呼ぶ
// テストはそれらを一つも通らないため、配線の外れた実装を検出できない。
//
// このファイルは組み立てと共通ヘルパー、およびロールと粒度スコープの拒否を持つ。
// エクスポート、アカウント API、ライフサイクルの拒否は同じ組み立てを使う別ファイルにある。

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ambi/idmagic/backend/apitoken"
	apitokenmemory "github.com/ambi/idmagic/backend/apitoken/db_memory"
	apitokendomain "github.com/ambi/idmagic/backend/apitoken/domain"
	apitokenusecases "github.com/ambi/idmagic/backend/apitoken/usecases"
	"github.com/ambi/idmagic/backend/authentication"
	sessionmemory "github.com/ambi/idmagic/backend/authentication/session/db_memory"
	sessiondomain "github.com/ambi/idmagic/backend/authentication/session/domain"
	sessionusecases "github.com/ambi/idmagic/backend/authentication/session/usecases"
	authusecases "github.com/ambi/idmagic/backend/authentication/usecases"
	"github.com/ambi/idmagic/backend/idmanagement"
	agentmemory "github.com/ambi/idmagic/backend/idmanagement/agent/db_memory"
	agentdomain "github.com/ambi/idmagic/backend/idmanagement/agent/domain"
	idmmemory "github.com/ambi/idmagic/backend/idmanagement/db_memory"
	idmdomain "github.com/ambi/idmagic/backend/idmanagement/domain"
	groupmemory "github.com/ambi/idmagic/backend/idmanagement/group/db_memory"
	groupdomain "github.com/ambi/idmagic/backend/idmanagement/group/domain"
	usermemory "github.com/ambi/idmagic/backend/idmanagement/user/db_memory"
	userdomain "github.com/ambi/idmagic/backend/idmanagement/user/domain"
	"github.com/ambi/idmagic/backend/jobs"
	jobsmemory "github.com/ambi/idmagic/backend/jobs/db_memory"
	jobsdomain "github.com/ambi/idmagic/backend/jobs/domain"
	jobsports "github.com/ambi/idmagic/backend/jobs/ports"
	httpadapter "github.com/ambi/idmagic/backend/shared/http/server_http"
	support "github.com/ambi/idmagic/backend/shared/http/support_http"
	emailmemory "github.com/ambi/idmagic/backend/shared/notification/email_memory"
	sharednotification "github.com/ambi/idmagic/backend/shared/notification/ports"
	"github.com/ambi/idmagic/backend/shared/security/passwords_argon2id"
	tokensjose "github.com/ambi/idmagic/backend/shared/security/tokens_jose"
	"github.com/ambi/idmagic/backend/shared/spec"
	signingcrypto "github.com/ambi/idmagic/backend/signingkeys/keys_memory"
	"github.com/ambi/idmagic/backend/tenancy"
	tenancymemory "github.com/ambi/idmagic/backend/tenancy/db_memory"
	tenancydomain "github.com/ambi/idmagic/backend/tenancy/domain"

	"github.com/labstack/echo/v5"
)

const (
	idmRefusalIssuer      = "http://idp.test"
	idmRefusalOtherTenant = "acme"
	// idmRefusalCSRF は二重送信 CSRF が成立する任意の値。
	idmRefusalCSRF = "idm-refusal-csrf-token"

	// idmRefusalAdmin は自テナントで管理 API を通せる操作者。
	idmRefusalAdmin = "admin-default"
	// idmRefusalForeignAdmin は別テナントで正当な管理者。
	idmRefusalForeignAdmin = "admin-acme"
	// idmRefusalForeignMember は別テナントの利用者。越境したカーソルが 2 ページ目を
	// 持つために要る。1 件しか居ないテナントは next のカーソルを発行しない。
	idmRefusalForeignMember = "user-acme"
	// idmRefusalAlice は拒否が動かしてはならない属性とメンバーシップを持つ利用者。
	idmRefusalAlice = "user-alice"
	// idmRefusalBob はロールを 1 つも持たない利用者。管理 API のロール境界を名指す。
	idmRefusalBob = "user-bob"

	// idmRefusalManualGroup は手動メンバーシップのグループ。alice が所属する。
	idmRefusalManualGroup = "group-engineering"
	// idmRefusalDynamicGroup は動的メンバーシップのグループ。手動操作の拒否がここに宿る。
	idmRefusalDynamicGroup = "group-dynamic"
	// idmRefusalAgent は在籍している Agent。キルと削除の拒否の対象。
	idmRefusalAgent = "agent-batch"
)

// idmRefusalFixture は拒否の効果を読み直すための保存層を、組み立てたサーバと一緒に持つ。
//
// テナントは 2 つ持つ。テナント境界の拒否は、越境した要求が拒否されるだけでなく、
// 越境された側の資源が無傷であることまで確かめて初めて意味を持つ。
type idmRefusalFixture struct {
	e           *echo.Echo
	users       *usermemory.UserRepository
	groups      *groupmemory.GroupRepository
	agents      *agentmemory.AgentRepository
	jobs        *jobsmemory.JobRepository
	jobClock    *backdatingJobRepository
	artifacts   *idmmemory.CSVArtifactStore
	sessions    *sessionmemory.SessionStore
	emailTokens *usermemory.EmailChangeTokenStore
	emails      *emailmemory.NoopEmailSender
	apiTokens   *apitokenusecases.Service
	apiTokenDB  *apitokenmemory.Repository
	events      *[]spec.DomainEvent
}

func newIdmRefusalServer(t *testing.T) *idmRefusalFixture {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()

	users := usermemory.NewUserRepository()
	for _, seed := range []struct {
		id, tenantID string
		roles        []string
	}{
		{idmRefusalAdmin, tenancydomain.DefaultTenantID, []string{"admin"}},
		{idmRefusalForeignAdmin, idmRefusalOtherTenant, []string{"admin"}},
		{idmRefusalForeignMember, idmRefusalOtherTenant, nil},
		{idmRefusalAlice, tenancydomain.DefaultTenantID, nil},
		{idmRefusalBob, tenancydomain.DefaultTenantID, nil},
	} {
		email := seed.id + "@example.test"
		users.Seed(&userdomain.User{
			ID: seed.id, TenantID: seed.tenantID, PreferredUsername: seed.id,
			Email: &email, EmailVerified: true, PasswordHash: "unused", Roles: seed.roles,
			Name:      new(seed.id),
			Lifecycle: userdomain.UserLifecycle{Status: idmdomain.UserStatusActive},
			CreatedAt: now, UpdatedAt: now,
		})
	}

	groups := groupmemory.NewGroupRepository()
	for _, group := range []*groupdomain.Group{
		{
			ID: idmRefusalManualGroup, TenantID: tenancydomain.DefaultTenantID, Name: "engineering",
			Roles: []string{"catalog:read"}, MembershipType: groupdomain.GroupMembershipManual,
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: idmRefusalDynamicGroup, TenantID: tenancydomain.DefaultTenantID, Name: "dynamic-engineering",
			MembershipType: groupdomain.GroupMembershipDynamic, CreatedAt: now, UpdatedAt: now,
		},
	} {
		if err := groups.Save(ctx, group); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := groups.AddMember(ctx, &groupdomain.GroupMember{
		GroupID: idmRefusalManualGroup, UserID: idmRefusalAlice,
		Source: groupdomain.MembershipSourceManual, CreatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	agents := agentmemory.NewAgentRepository()
	if err := agents.Save(ctx, &agentdomain.Agent{
		ID: idmRefusalAgent, TenantID: tenancydomain.DefaultTenantID, Name: "batch-agent",
		Kind: idmdomain.AgentKindAutonomous, OwnerUserID: idmRefusalAdmin,
		Status: idmdomain.AgentStatusActive, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}

	tenants := tenancymemory.NewTenantRepository()
	for _, tenant := range []*tenancydomain.Tenant{
		{ID: tenancydomain.DefaultTenantID, Realm: tenancydomain.DefaultRealm, Status: tenancydomain.TenantStatusActive},
		{ID: idmRefusalOtherTenant, Realm: idmRefusalOtherTenant, Status: tenancydomain.TenantStatusActive},
	} {
		if err := tenants.Save(ctx, tenant); err != nil {
			t.Fatal(err)
		}
	}

	// 本物の署名器を使う。API アクセストークンのテナント束縛は `aud` と `jti` の照合として
	// イントロスペクション側にあり、偽の introspector で置き換えるとその照合ごと消える。
	keyStore, err := signingcrypto.NewInMemoryKeyStore()
	if err != nil {
		t.Fatal(err)
	}
	signer := tokensjose.NewJWTSigner(idmRefusalIssuer, keyStore)

	apiTokenRepo := apitokenmemory.NewRepository()
	jobRepo := jobsmemory.NewJobRepository()
	fixture := &idmRefusalFixture{
		e:           echo.New(),
		users:       users,
		groups:      groups,
		agents:      agents,
		jobs:        jobRepo,
		jobClock:    &backdatingJobRepository{JobRepository: jobRepo},
		artifacts:   idmmemory.NewCSVArtifactStore(),
		sessions:    sessionmemory.NewSessionStore(),
		emailTokens: usermemory.NewEmailChangeTokenStore(),
		emails:      &emailmemory.NoopEmailSender{},
		apiTokens: apitoken.Module{
			Repo: apiTokenRepo, TokenIssuer: signer, TokenIntrospector: signer,
		}.Service(),
		apiTokenDB: apiTokenRepo,
		events:     &[]spec.DomainEvent{},
	}

	sessionManager := sessionusecases.NewSessionManager(fixture.sessions)
	deps := httpadapter.Deps{
		Issuer: idmRefusalIssuer,
		// カーソルの署名鍵は組み立ての一部である。テナントと絞り込みは付随データとして
		// 署名へ束ねられるので、codec を省くとカーソルの拒否そのものが消える。
		PaginationCodec: support.NewCursorCodec([]byte("refusal-effects-pagination-secret")),
		TenantRepo:      tenants,
		Emit:            func(event spec.DomainEvent) { *fixture.events = append(*fixture.events, event) },
		IdManagement: idmanagement.Module{
			UserRepo: users, GroupRepo: groups, AgentRepo: agents,
			CSVArtifacts: fixture.artifacts, EmailChangeTokenStore: fixture.emailTokens,
		},
		Tenancy: tenancy.Module{AttrSchemaRepo: usermemory.NewTenantUserAttributeSchemaRepository()},
		Authentication: authentication.Module{
			SessionStore: fixture.sessions, SessionManager: sessionManager, AuthnResolver: sessionManager,
			PasswordHasher: passwords_argon2id.NewArgon2idPasswordHasher(),
		},
		Notification:      sharednotification.Module{EmailSender: fixture.emails},
		Jobs:              jobs.Module{Repo: fixture.jobClock},
		ApiTokens:         apitoken.Module{Repo: apiTokenRepo},
		KeyStore:          keyStore,
		TokenIssuer:       signer,
		TokenIntrospector: signer,
	}
	httpadapter.Register(fixture.e, deps)
	return fixture
}

// idmRealmFor はテナント id に対応する公開レルム名を返す。
func idmRealmFor(tenantID string) string {
	if tenantID == tenancydomain.DefaultTenantID {
		return tenancydomain.DefaultRealm
	}
	return tenantID
}

// seedSession は指定した利用者のログインセッションを保存し、その id を返す。
func (f *idmRefusalFixture) seedSession(
	t *testing.T, id, tenantID, userID string, mutate ...func(*sessiondomain.LoginSession),
) string {
	t.Helper()
	now := time.Now().UTC()
	session := &sessiondomain.LoginSession{
		ID: id, TenantID: tenantID, UserID: userID,
		AuthTime: now.Unix(), AMR: []string{"pwd"}, ACR: authusecases.DeriveACR([]string{"pwd"}),
		ExpiresAt: now.Add(time.Hour),
	}
	for _, apply := range mutate {
		apply(session)
	}
	if err := f.sessions.Save(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	return id
}

// withFreshStepUp は step-up 再認証を今まさに済ませたセッションにする。
func withFreshStepUp(session *sessiondomain.LoginSession) {
	session.StepUpAt = time.Now().UTC().Unix()
}

// issueApiToken は default テナントの利用者に固定した API アクセストークンを発行する。
// テナント文脈を通して発行するので、`iss` と `aud` がそのままトークンのテナント束縛になる。
// 越境の拒否は、ここで発行したトークンを別レルムへ提示することで確かめる。
func (f *idmRefusalFixture) issueApiToken(
	t *testing.T, userID string, scopes ...apitokendomain.Scope,
) string {
	t.Helper()
	tenantID := tenancydomain.DefaultTenantID
	realm := idmRealmFor(tenantID)
	ctx := tenancy.WithTenant(
		context.Background(),
		&tenancydomain.Tenant{ID: tenantID, Realm: realm},
		idmRefusalIssuer+"/realms/"+realm,
		"/realms/"+realm,
	)
	token, _, err := f.apiTokens.Issue(
		ctx, tenantID, userID, "refusal effects", apitokendomain.Scopes(scopes).Strings(), 7, "",
	)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// idmRefusalRequest は 1 本のリクエストを組み立てて送る。
type idmRefusalRequest struct {
	method   string
	tenantID string
	path     string
	body     any
	// rawBody は CSV のように JSON でない本文を送るときに使う。body とは排他。
	rawBody     string
	contentType string
	sessionID   string
	// bearer は API アクセストークン。セッション Cookie とは排他に使う。
	bearer string
	// csrf を空にすると二重送信の照合が成立しない要求になる。
	csrf string
	// csrfCookie は cookie 側だけを別の値にしたいときに使う。空なら csrf と同じ値。
	csrfCookie string
	// origin を空にすると発行者と一致する既定の Origin が付く。
	origin string
}

func (f *idmRefusalFixture) send(t *testing.T, request idmRefusalRequest) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	switch {
	case request.rawBody != "":
		payload = []byte(request.rawBody)
	case request.body != nil:
		encoded, err := json.Marshal(request.body)
		if err != nil {
			t.Fatal(err)
		}
		payload = encoded
	}
	tenantID := request.tenantID
	if tenantID == "" {
		tenantID = tenancydomain.DefaultTenantID
	}
	target := "/realms/" + idmRealmFor(tenantID) + request.path
	httpRequest := httptest.NewRequest(request.method, target, bytes.NewReader(payload))
	if len(payload) > 0 {
		contentType := request.contentType
		if contentType == "" {
			contentType = "application/json"
		}
		httpRequest.Header.Set("Content-Type", contentType)
	}
	origin := request.origin
	if origin == "" {
		origin = idmRefusalIssuer
	}
	httpRequest.Header.Set("Origin", origin)
	if request.bearer != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+request.bearer)
	}
	if request.sessionID != "" {
		httpRequest.AddCookie(&http.Cookie{Name: sessionusecases.SessionCookie, Value: request.sessionID})
	}
	if request.csrf != "" {
		cookieValue := request.csrfCookie
		if cookieValue == "" {
			cookieValue = request.csrf
		}
		httpRequest.Header.Set(support.CSRFHeader, request.csrf)
		httpRequest.AddCookie(&http.Cookie{Name: support.CSRFCookie, Value: cookieValue})
	}
	recorder := httptest.NewRecorder()
	f.e.ServeHTTP(recorder, httpRequest)
	return recorder
}

// idmProblemCode は Problem Details の type URN (urn:idmagic:error:<code>) から code を取り出す。
func idmProblemCode(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var problem support.Problem
	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil {
		return ""
	}
	return strings.TrimPrefix(problem.Type, "urn:idmagic:error:")
}

// runnableJobs は実行待ちの Job を取り出す。「拒否がジョブを作らなかった」を、
// 保存層の内部表現ではなく worker が掴めるものとして読む。
// CSV のインポートとエクスポートはどちらも bulk レーンなので、そのレーンだけを見る。
func (f *idmRefusalFixture) runnableJobs(t *testing.T) []*jobsdomain.Job {
	t.Helper()
	claimed, err := f.jobs.ClaimBatch(
		context.Background(), "refusal-effects-worker", jobsdomain.LaneBulk, 50, time.Minute, time.Now().UTC(),
	)
	if err != nil {
		t.Fatal(err)
	}
	return claimed
}

// user は保存層から利用者を読み直す。
func (f *idmRefusalFixture) user(t *testing.T, sub string) *userdomain.User {
	t.Helper()
	user, err := f.users.FindBySubIncludingDeleted(context.Background(), sub)
	if err != nil {
		t.Fatal(err)
	}
	if user == nil {
		t.Fatalf("利用者 %s が消えている", sub)
	}
	return user
}

// groupNames は default テナントに残っているグループ名の一覧を返す。
func (f *idmRefusalFixture) groupNames(t *testing.T) []string {
	t.Helper()
	groups, err := f.groups.ListAll(context.Background(), tenancydomain.DefaultTenantID)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(groups))
	for _, group := range groups {
		names = append(names, group.Name)
	}
	return names
}

// memberIDs は保存層に残っているグループのメンバー id を返す。
func (f *idmRefusalFixture) memberIDs(t *testing.T, groupID string) []string {
	t.Helper()
	members, err := f.groups.ListMembersByGroup(context.Background(), tenancydomain.DefaultTenantID, groupID)
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(members))
	for _, member := range members {
		ids = append(ids, member.UserID)
	}
	return ids
}

// agent は保存層から Agent を読み直す。存在しなければ nil を返す。
func (f *idmRefusalFixture) agent(t *testing.T, id string) *agentdomain.Agent {
	t.Helper()
	agent, err := f.agents.FindByID(context.Background(), tenancydomain.DefaultTenantID, id)
	if err != nil {
		t.Fatal(err)
	}
	return agent
}

// EX-IDMANAGEMENT-014-02: `admin` ロールを持たないユーザーの管理 API 呼び出しは拒否され、
// 応答に管理対象の一覧は含まれず、作成も通らない。
//
// 一覧の拒否は 403 の本文を返すので応答だけでも見分けが付くが、作成の側は
// 「403 を書いてから作る」実装と区別できない。作成のあとで一覧を読み直す。
func TestAdminApiWithoutAdminRoleListsNothingAndCreatesNothing(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	plain := fixture.seedSession(t, "sess-bob", tenancydomain.DefaultTenantID, idmRefusalBob)

	listed := fixture.send(t, idmRefusalRequest{
		method: http.MethodGet, path: "/api/admin/v1/users", sessionID: plain,
	})
	if listed.Code != http.StatusForbidden {
		t.Fatalf("ロールを持たない利用者の一覧が status=%d body=%s", listed.Code, listed.Body.String())
	}
	if code := idmProblemCode(t, listed); code != "access_denied" {
		t.Fatalf("error code=%q, want access_denied", code)
	}
	// 拒否が何も返していないこと。応答に既存の利用者が 1 人も現れない。
	for _, sub := range []string{idmRefusalAdmin, idmRefusalAlice} {
		if strings.Contains(listed.Body.String(), sub) {
			t.Fatalf("拒否した応答が %s を含む: %s", sub, listed.Body.String())
		}
	}

	created := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/users", sessionID: plain, csrf: idmRefusalCSRF,
		body: map[string]any{"preferred_username": "mallory", "password": "refusal-password-1234"},
	})
	if created.Code == http.StatusCreated {
		t.Fatalf("ロールを持たない利用者の作成が受理された: %s", created.Body.String())
	}
	// 拒否が何も作っていないこと。作成対象は保存層に現れない。
	if user, err := fixture.users.FindByUsername(
		context.Background(), tenancydomain.DefaultTenantID, "mallory",
	); err != nil {
		t.Fatal(err)
	} else if user != nil {
		t.Fatal("拒否されたのに利用者が作成されている")
	}

	// 対照: admin ロールを持つ操作者では同じ 2 つの要求が通る。
	// これが無いと「そもそも管理 API が動かない構成だった」と区別できない。
	admin := fixture.seedSession(t, "sess-admin", tenancydomain.DefaultTenantID, idmRefusalAdmin)
	accepted := fixture.send(t, idmRefusalRequest{
		method: http.MethodGet, path: "/api/admin/v1/users", sessionID: admin,
	})
	if accepted.Code != http.StatusOK || !strings.Contains(accepted.Body.String(), idmRefusalAlice) {
		t.Fatalf("前提が壊れている: 管理者の一覧が status=%d body=%s", accepted.Code, accepted.Body.String())
	}
}

// EX-IDMANAGEMENT-005-05: TenantAdministrator ロールを持たない実行者の
// `ListAdminUsers` は `AccessDeniedError` で拒否され、応答にユーザーは 1 件も含まれない。
//
// 014-02 と同じ防護だが、こちらが名指すのは一覧の総件数とページングのメタデータまで
// 漏れないことである。件数だけを返す実装は「一覧は返していない」と言えてしまう。
func TestListAdminUsersWithoutAdminRoleReturnsNoUsersAndNoCounts(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	plain := fixture.seedSession(t, "sess-bob-list", tenancydomain.DefaultTenantID, idmRefusalBob)

	refused := fixture.send(t, idmRefusalRequest{
		method: http.MethodGet, path: "/api/admin/v1/users?limit=1", sessionID: plain,
	})
	if refused.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s, want 403", refused.Code, refused.Body.String())
	}
	if code := idmProblemCode(t, refused); code != "access_denied" {
		t.Fatalf("error code=%q, want access_denied", code)
	}
	if total := refused.Header().Get("Pagination-Total-Items"); total != "" {
		t.Fatalf("拒否した応答が総件数 %q を返した", total)
	}
	if link := refused.Header().Get("Link"); link != "" {
		t.Fatalf("拒否した応答がページングのカーソル %q を返した", link)
	}
	if strings.Contains(refused.Body.String(), idmRefusalAlice) {
		t.Fatalf("拒否した応答が利用者を含む: %s", refused.Body.String())
	}

	// 対照: 同じ要求が管理者では通り、件数とカーソルが返る。
	admin := fixture.seedSession(t, "sess-admin-list", tenancydomain.DefaultTenantID, idmRefusalAdmin)
	accepted := fixture.send(t, idmRefusalRequest{
		method: http.MethodGet, path: "/api/admin/v1/users?limit=1", sessionID: admin,
	})
	if accepted.Code != http.StatusOK || accepted.Header().Get("Pagination-Total-Items") == "" {
		t.Fatalf("前提が壊れている: 管理者の一覧が status=%d headers=%v", accepted.Code, accepted.Header())
	}
}

// EX-IDMANAGEMENT-005-06: 別テナントで発行された、改ざんされた、または発行時と
// `query` / `status` が異なるカーソルは `InvalidRequestError` で拒否され、
// 別テナントのページは返らない。
//
// カーソルはテナントと絞り込みを HMAC の付随データとして束ねる。したがって
// 「拒否が変えなかったもの」は、越境したページの中身が応答に出ないことである。
func TestListAdminUsersRejectsForeignTamperedAndRefilteredCursors(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	admin := fixture.seedSession(t, "sess-admin-cursor", tenancydomain.DefaultTenantID, idmRefusalAdmin)
	foreignAdmin := fixture.seedSession(
		t, "sess-acme-cursor", idmRefusalOtherTenant, idmRefusalForeignAdmin,
	)

	// acme のレルムで 1 件だけ取り、その next カーソルを default のレルムへ持ち込む。
	foreignPage := fixture.send(t, idmRefusalRequest{
		method: http.MethodGet, path: "/api/admin/v1/users?limit=1",
		tenantID: idmRefusalOtherTenant, sessionID: foreignAdmin,
	})
	if foreignPage.Code != http.StatusOK {
		t.Fatalf("前提が壊れている: acme の一覧が status=%d body=%s", foreignPage.Code, foreignPage.Body.String())
	}

	homePage := fixture.send(t, idmRefusalRequest{
		method: http.MethodGet, path: "/api/admin/v1/users?limit=1", sessionID: admin,
	})
	if homePage.Code != http.StatusOK {
		t.Fatalf("前提が壊れている: default の一覧が status=%d body=%s", homePage.Code, homePage.Body.String())
	}
	homeCursor := cursorFromLink(t, homePage.Header().Get("Link"), "next")

	for _, refusal := range []struct {
		name string
		path string
	}{
		{
			"別テナントで発行されたカーソル",
			"/api/admin/v1/users?limit=1&cursor=" + cursorFromLink(t, foreignPage.Header().Get("Link"), "next"),
		},
		{"改ざんされたカーソル", "/api/admin/v1/users?limit=1&cursor=" + tamperCursor(homeCursor)},
		{"発行時と条件が異なるカーソル", "/api/admin/v1/users?limit=1&status=disabled&cursor=" + homeCursor},
	} {
		t.Run(refusal.name, func(t *testing.T) {
			refused := fixture.send(t, idmRefusalRequest{
				method: http.MethodGet, path: refusal.path, sessionID: admin,
			})
			if refused.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s, want 400", refused.Code, refused.Body.String())
			}
			if code := idmProblemCode(t, refused); code != "invalid_request" {
				t.Fatalf("error code=%q, want invalid_request", code)
			}
			// 拒否が何も返していないこと。越境したページの利用者が応答に出ない。
			if strings.Contains(refused.Body.String(), idmRefusalForeignAdmin) {
				t.Fatalf("拒否した応答が acme の利用者を含む: %s", refused.Body.String())
			}
			if strings.Contains(refused.Body.String(), `"users"`) {
				t.Fatalf("拒否した応答が一覧を含む: %s", refused.Body.String())
			}
		})
	}

	// 対照: 発行元のテナントで発行時と同じ条件なら、同じカーソルが次ページを返す。
	accepted := fixture.send(t, idmRefusalRequest{
		method: http.MethodGet, path: "/api/admin/v1/users?limit=1&cursor=" + homeCursor, sessionID: admin,
	})
	if accepted.Code != http.StatusOK {
		t.Fatalf("前提が壊れている: 正しいカーソルが status=%d body=%s", accepted.Code, accepted.Body.String())
	}
}

// cursorFromLink は Link ヘッダーから指定した rel のカーソル値を取り出す。
func cursorFromLink(t *testing.T, link, rel string) string {
	t.Helper()
	for part := range strings.SplitSeq(link, ",") {
		if !strings.Contains(part, `rel="`+rel+`"`) {
			continue
		}
		_, value, found := strings.Cut(part, "cursor=")
		if !found {
			continue
		}
		if end := strings.IndexAny(value, "&>;"); end >= 0 {
			value = value[:end]
		}
		return value
	}
	t.Fatalf("Link ヘッダーに rel=%q のカーソルが無い: %q", rel, link)
	return ""
}

// tamperCursor は本体の中ほどを 1 文字書き換える。
//
// 末尾を書き換えないのは、base64 の最終文字が有効なビットを持たないことがあり、
// 書き換えても復号結果が変わらないためである。実際にそれで拒否が観測できなかった。
func tamperCursor(cursor string) string {
	body := strings.TrimPrefix(cursor, "v3.")
	if len(body) < 4 {
		return cursor
	}
	at := len(body) / 2
	replacement := byte('A')
	if body[at] == 'A' {
		replacement = 'B'
	}
	return "v3." + body[:at] + string(replacement) + body[at+1:]
}

// EX-IDMANAGEMENT-025-02: `users:read` だけの API アクセストークンによる User の変更と
// CSV インポートは `AccessDeniedError` で拒否され、User は変更されず、
// インポートのジョブも作られない。
func TestUsersReadScopeChangesNoUserAndStartsNoImport(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	readOnly := fixture.issueApiToken(
		t, idmRefusalAdmin, apitokendomain.ScopeUsersRead,
	)
	before := *fixture.user(t, idmRefusalAlice)

	renamed := fixture.send(t, idmRefusalRequest{
		method: http.MethodPatch, path: "/api/admin/v1/users/" + idmRefusalAlice,
		bearer: readOnly, body: map[string]any{"name": "renamed-by-read-scope"},
	})
	if renamed.Code != http.StatusForbidden {
		t.Fatalf("users:read の変更が status=%d body=%s, want 403", renamed.Code, renamed.Body.String())
	}
	if code := idmProblemCode(t, renamed); code != "insufficient_scope" {
		t.Fatalf("error code=%q, want insufficient_scope", code)
	}
	// 拒否が何も変えていないこと。表示名も更新時刻も動いていない。
	after := fixture.user(t, idmRefusalAlice)
	if after.Name == nil || *after.Name != *before.Name || !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("拒否されたのに利用者が変わった: name=%v updated_at=%s", after.Name, after.UpdatedAt)
	}

	imported := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/users/imports", bearer: readOnly,
		contentType: "text/csv", rawBody: "preferred_username\nmallory\n",
	})
	if imported.Code != http.StatusForbidden {
		t.Fatalf("users:read のインポートが status=%d body=%s, want 403", imported.Code, imported.Body.String())
	}
	// 拒否がジョブを作っていないこと。作ってから失敗させる実装と区別する。
	if claimed := fixture.runnableJobs(t); len(claimed) != 0 {
		t.Fatalf("拒否されたインポートが %d 件のジョブを残した: %+v", len(claimed), claimed)
	}

	// 対照: users:write を持つトークンなら同じ変更が通る。
	writable := fixture.issueApiToken(
		t, idmRefusalAdmin, apitokendomain.ScopeUsersWrite,
	)
	accepted := fixture.send(t, idmRefusalRequest{
		method: http.MethodPatch, path: "/api/admin/v1/users/" + idmRefusalAlice,
		bearer: writable, body: map[string]any{"name": "renamed-by-write-scope"},
	})
	if accepted.Code != http.StatusOK {
		t.Fatalf("前提が壊れている: users:write の変更が status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	if name := fixture.user(t, idmRefusalAlice).Name; name == nil || *name != "renamed-by-write-scope" {
		t.Fatalf("前提が壊れている: users:write でも表示名が変わらない: %v", name)
	}
}

// EX-IDMANAGEMENT-025-03: `groups:read` だけの API アクセストークンによる Group CSV の
// インポートとその適用は `AccessDeniedError` で拒否され、`Group` は 1 件も作成、更新、
// 削除されない。
//
// この具体例は副作用の不在まで宣言している。宣言がその形なのに引用するテストが
// 無かったので、テスト側も同じ形で読む。
func TestGroupsReadScopeImportsNothingAndLeavesGroupsUnchanged(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	readOnly := fixture.issueApiToken(
		t, idmRefusalAdmin, apitokendomain.ScopeGroupsRead,
	)
	before := fixture.groupNames(t)

	for _, refusal := range []struct {
		name, method, path, body string
	}{
		{
			"プレビューの投入", http.MethodPost, "/api/admin/v1/groups/imports",
			"name,roles\nsales,catalog:read\n",
		},
		{"適用の開始", http.MethodPost, "/api/admin/v1/groups/imports/any-preview-job/apply", ""},
	} {
		t.Run(refusal.name, func(t *testing.T) {
			refused := fixture.send(t, idmRefusalRequest{
				method: refusal.method, path: refusal.path, bearer: readOnly,
				contentType: "text/csv", rawBody: refusal.body,
			})
			if refused.Code != http.StatusForbidden {
				t.Fatalf("status=%d body=%s, want 403", refused.Code, refused.Body.String())
			}
			if code := idmProblemCode(t, refused); code != "insufficient_scope" {
				t.Fatalf("error code=%q, want insufficient_scope", code)
			}
		})
	}

	// 拒否が Group を 1 件も作成、更新、削除していないこと。
	if after := fixture.groupNames(t); !sameStrings(before, after) {
		t.Fatalf("拒否されたのにグループの集合が変わった: before=%v after=%v", before, after)
	}
	if claimed := fixture.runnableJobs(t); len(claimed) != 0 {
		t.Fatalf("拒否されたインポートが %d 件のジョブを残した: %+v", len(claimed), claimed)
	}

	// 対照: groups:write を持つトークンなら同じプレビューが受理される。
	writable := fixture.issueApiToken(
		t, idmRefusalAdmin, apitokendomain.ScopeGroupsWrite,
	)
	accepted := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/groups/imports", bearer: writable,
		contentType: "text/csv", rawBody: "name,roles\nsales,catalog:read\n",
	})
	if accepted.Code != http.StatusAccepted {
		t.Fatalf("前提が壊れている: groups:write の投入が status=%d body=%s", accepted.Code, accepted.Body.String())
	}
}

// EX-IDMANAGEMENT-025-04: `users:*` だけの API アクセストークンによる Group と Agent の
// 操作は `AccessDeniedError` で拒否され、Group も Agent も作成されない。
func TestUsersScopeCreatesNoGroupAndNoAgent(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	usersOnly := fixture.issueApiToken(
		t, idmRefusalAdmin,
		apitokendomain.ScopeUsersRead, apitokendomain.ScopeUsersWrite,
	)
	beforeGroups := fixture.groupNames(t)

	group := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/groups", bearer: usersOnly,
		body: map[string]any{"name": "sales", "roles": []string{"catalog:read"}},
	})
	if group.Code != http.StatusForbidden {
		t.Fatalf("users:* のグループ作成が status=%d body=%s, want 403", group.Code, group.Body.String())
	}
	if after := fixture.groupNames(t); !sameStrings(beforeGroups, after) {
		t.Fatalf("拒否されたのにグループが作られた: before=%v after=%v", beforeGroups, after)
	}

	agent := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/agents", bearer: usersOnly,
		body: map[string]any{"name": "rogue-agent", "kind": "autonomous"},
	})
	if agent.Code != http.StatusForbidden {
		t.Fatalf("users:* のエージェント登録が status=%d body=%s, want 403", agent.Code, agent.Body.String())
	}
	agents, err := fixture.agents.ListAll(context.Background(), tenancydomain.DefaultTenantID)
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 1 || agents[0].ID != idmRefusalAgent {
		t.Fatalf("拒否されたのにエージェントが作られた: %+v", agents)
	}

	// 対照: それぞれのスコープを持つトークンなら同じ 2 つの要求が通る。
	groupWrite := fixture.issueApiToken(
		t, idmRefusalAdmin, apitokendomain.ScopeGroupsWrite,
	)
	if accepted := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/groups", bearer: groupWrite,
		body: map[string]any{"name": "sales", "roles": []string{"catalog:read"}},
	}); accepted.Code != http.StatusCreated {
		t.Fatalf("前提が壊れている: groups:write の作成が status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	agentWrite := fixture.issueApiToken(
		t, idmRefusalAdmin, apitokendomain.ScopeAgentsWrite,
	)
	if accepted := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/agents", bearer: agentWrite,
		body: map[string]any{"name": "rogue-agent", "kind": "autonomous"},
	}); accepted.Code != http.StatusCreated {
		t.Fatalf("前提が壊れている: agents:write の登録が status=%d body=%s", accepted.Code, accepted.Body.String())
	}
}

// EX-IDMANAGEMENT-025-05: `agents:read` だけの API アクセストークンによる Agent の
// キルと削除は `AccessDeniedError` で拒否され、対象の Agent は在籍したままである。
//
// キルも削除も応答の本文を持たないので、拒否と成功は本文では見分けられない。
// どちらのあとでも Agent を読み直す。
func TestAgentsReadScopeKillsAndDeletesNothing(t *testing.T) {
	fixture := newIdmRefusalServer(t)
	readOnly := fixture.issueApiToken(
		t, idmRefusalAdmin, apitokendomain.ScopeAgentsRead,
	)

	for _, refusal := range []struct{ name, method, path string }{
		{"キル", http.MethodPost, "/api/admin/v1/agents/" + idmRefusalAgent + "/kill"},
		{"削除", http.MethodDelete, "/api/admin/v1/agents/" + idmRefusalAgent},
	} {
		t.Run(refusal.name, func(t *testing.T) {
			refused := fixture.send(t, idmRefusalRequest{
				method: refusal.method, path: refusal.path, bearer: readOnly, body: map[string]any{},
			})
			if refused.Code != http.StatusForbidden {
				t.Fatalf("status=%d body=%s, want 403", refused.Code, refused.Body.String())
			}
			if code := idmProblemCode(t, refused); code != "insufficient_scope" {
				t.Fatalf("error code=%q, want insufficient_scope", code)
			}
			// 拒否が何も変えていないこと。Agent は在籍し、キルもされていない。
			agent := fixture.agent(t, idmRefusalAgent)
			if agent == nil {
				t.Fatal("拒否されたのに Agent が削除されている")
			}
			if agent.Status != idmdomain.AgentStatusActive || agent.KilledAt != nil {
				t.Fatalf("拒否されたのに Agent の状態が変わった: status=%s killed_at=%v", agent.Status, agent.KilledAt)
			}
		})
	}

	// 対照: agents:write を持つトークンなら同じキルが通り、状態が変わる。
	writable := fixture.issueApiToken(
		t, idmRefusalAdmin, apitokendomain.ScopeAgentsWrite,
	)
	accepted := fixture.send(t, idmRefusalRequest{
		method: http.MethodPost, path: "/api/admin/v1/agents/" + idmRefusalAgent + "/kill",
		bearer: writable, body: map[string]any{},
	})
	if accepted.Code >= http.StatusBadRequest {
		t.Fatalf("前提が壊れている: agents:write のキルが status=%d body=%s", accepted.Code, accepted.Body.String())
	}
	if agent := fixture.agent(t, idmRefusalAgent); agent == nil || agent.Status != idmdomain.AgentStatusKilled {
		t.Fatalf("前提が壊れている: agents:write でもキルされない: %+v", agent)
	}
}

// sameStrings は順序を無視して 2 つの集合が一致するかを返す。
func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	counts := map[string]int{}
	for _, value := range left {
		counts[value]++
	}
	for _, value := range right {
		counts[value]--
		if counts[value] < 0 {
			return false
		}
	}
	return true
}

// backdatingJobRepository は Enqueue が刻む時刻だけを過去へずらす装飾。
//
// エクスポートの保持期限は `job.CreatedAt + DataExportTTL` と現在時刻の比較で決まり、
// HTTP のハンドラーは時計を注入していない。期限切れの拒否を production と同じ入口から
// 観測するには、ジョブが生まれた時刻をずらすほかない。production の時計にも期限の
// 判定にも触れないので、判定そのものを消してしまうことはない。
//
// backdate が 0 のあいだは素通しなので、これを使わないテストは影響を受けない。
type backdatingJobRepository struct {
	jobsports.JobRepository
	backdate time.Duration
}

func (r *backdatingJobRepository) Enqueue(
	ctx context.Context, input jobsports.EnqueueInput,
) (*jobsdomain.Job, bool, error) {
	if r.backdate > 0 {
		now := input.Now
		if now.IsZero() {
			now = time.Now().UTC()
		}
		input.Now = now.Add(-r.backdate)
		if !input.RunAt.IsZero() {
			input.RunAt = input.RunAt.Add(-r.backdate)
		}
	}
	return r.JobRepository.Enqueue(ctx, input)
}
