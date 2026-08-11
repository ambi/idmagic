---
status: completed
authors: [tn]
risk: high
created_at: 2026-08-09
depends_on: [wi-346-admin-list-addressable-cursor-pages]
change_kind: feature
initial_context:
  scl:
    IdManagement:
      - models.AdminUserListResponse
      - interfaces.ListAdminUsers
      - interfaces.ListGroups
      - interfaces.ListAgents
      - scenarios.管理者はユーザー一覧をページングしながら安定して閲覧できる
    Application:
      - interfaces.ListAdminApplications
    Audit:
      - interfaces.ListAdminAuditEvents
      - scenarios.管理者は監査ログをページングしながら閲覧でき絞り込み変更でcursorが無効化される
  source:
    - backend/shared/http/support_http/pagination_cursor.go
    - backend/shared/http/support_http/pagination_request.go
    - frontend/src/components/ui/page-navigation.tsx
    - frontend/src/api/core.ts
  tests:
    - backend/shared/http/support_http/pagination_cursor_test.go
    - backend/shared/http/support_http/pagination_request_test.go
    - backend/idmanagement/user/handlers_http/admin_user_list_pagination_test.go
    - frontend/src/components/ui/page-navigation.test.tsx
  stop_before_reading:
    - backend/idmanagement/user/usecases/user_import.go
    - backend/idmanagement/usecases/data_export.go
affected_spec:
  - { context: IdManagement, kind: model, element: AdminUserListResponse }
  - { context: IdManagement, kind: interface, element: ListAdminUsers }
  - { context: IdManagement, kind: interface, element: ListGroups }
  - { context: IdManagement, kind: interface, element: ListAgents }
  - { context: IdManagement, kind: scenario, element: 管理者はユーザー一覧をページングしながら安定して閲覧できる }
  - { context: Application, kind: interface, element: ListAdminApplications }
  - { context: Audit, kind: interface, element: ListAdminAuditEvents }
  - { context: Audit, kind: scenario, element: 管理者は監査ログをページングしながら閲覧でき絞り込み変更でcursorが無効化される }
---

# 管理一覧で正確な件数とページ位置を表示し短い cursor で先頭・末尾へ移動できるようにする

## Motivation
Users、Groups、Agents、Applications、Audit Events の管理一覧は双方向 keyset pagination に対応したが、
UI は「前へ」「次へ」だけで、全件数、全ページ数、現在ページを確認できず、先頭・末尾へ直接移動できない。
また、2 ページ目以降の URL に現れる cursor は tenant ID、query fingerprint、JSON、二重 base64、完全長の
HMAC tag を含むため長く、共有・確認時の可読性を大きく損なう。

ユーザー一覧の「総ユーザー」は一覧の実データではなく quota 用 `tenant_usages.users` を表示している。
既存データの backfill / reconciliation が未実装な環境では、この counter が 0 のままでも実ユーザーが存在し、
管理画面が誤った値を表示する。さらに、実運用では有効ユーザーのみなどの filter を既定にするため、画面全体の
総ユーザー数と現在の一覧条件に一致する件数を区別しながら、同じ pagination 要件を満たす必要がある。

offset pagination や任意ページ番号への jump は URL を短くできる一方、深いページほど走査コストが増え、
wi-346 で確立した keyset pagination の一定 latency を失う。keyset 性能、reload、browser history、URL 共有を
維持したまま、正確な count と compact な stateless cursor を追加する。

## Scope
- **Decision**: ADR-159 の cursor wire format と Link contract を再検討し、ADR-160 で compact cursor、
  正確な count、`rel="first"` / `rel="last"` を部分的に上書きする。
- **SCL**: IdManagement の `AdminUserListResponse`、Users / Groups / Agents / Applications / Audit Events の
  list interfaces、既存 latency objectives、代表 scenarios、flows に exact pagination metadata、4方向移動、
  compact cursor の保証を追加する。
- **Shared pagination**: page number と end anchor を持つ version 3 cursor、legacy v0/v2 decode、
  first / prev / next / last Link、正確な pagination metadata を共通化する。
- **Persistence**: 5 一覧の filter と同じ predicate を使う exact count と、末尾から端数件を取得する reverse
  keyset query を memory / PostgreSQL adapter に追加する。User には lifecycle status count 用 index を追加する。
- **API**: 5 一覧 API は現在の filter に一致する `Pagination-Total-Items`、`Pagination-Total-Pages`、
  `Pagination-Current-Page`、`Pagination-Page-Size` response headers と、4 方向の Link を返す。
  `AdminUserListResponse` は filter 非依存で削除済みを除く正確な `total_users` も返す。
- **UI**: Users、Groups、Agents、Applications、Audit Events の共通 navigation に件数・ページ位置と
  「最初へ」「前へ」「次へ」「最後へ」を表示する。Users は active を既定 filter にする。

## Out of Scope
- offset pagination、`page=200` のような page-number URL、任意ページ番号への jump。
- server-side cursor table / cache、browser session 内だけで有効な page-to-cursor map。
- 一覧結果を同一 snapshot に固定すること。同時更新時も keyset 境界からの重複なし navigation を保証するが、
  永続 cursor のページ番号は発行時の navigation 上の位置である。
- cursor 未対応 API、UI のない残り 5 cursor API、picker / lookup の pagination UI。
- quota usage counter の backfill / reconciliation と、Dashboard / Settings に残る usage 表示の補正。
- page-size selector。各 endpoint の既存 default / max limit は変更しない。

## Design

### 正確な count とページ契約
- `total_items` は tenant 境界、query、status、time range、audit structured filter など、一覧 query と同じ
  filter predicate に一致する committed row をリクエストごとに `COUNT(*)` して返す。推定値、quota counter、
  stale read model へ fallback しない。count に失敗した場合は 0 を返さず request 全体を失敗させる。
- `total_pages = ceil(total_items / requested_limit)` とする。空結果は total items / pages / current page を
  `0 / 0 / 0`、1 件以上の先頭ページは current page `1` とする。
- 通常の prev / next cursor は遷移先 page number を署名対象に含める。first は同じ filter / limit を保持して
  cursor を除いた URL、last は `anchor=end` とその時点の total pages を持つ backward cursor とする。
- end anchor を受けた handler は当該 request で total count を再計算し、現在の `total_pages` と
  `last_page_size = total_items % limit`（0 なら limit）を使って末尾から取得する。105 件 / limit 50 なら
  最終ページは 50 件ではなく 5 件だけ返し、第 2 ページと重複させない。
- 通常 cursor の発行後に insert / delete が起きた場合、total と total pages は request 時点の正確な値へ更新する。
  cursor page number が新しい total pages を超えた、または keyset の先に行が無い場合も metadata と first / last
  Link を返し、利用者が有効な端へ移動できるようにする。snapshot 固定や offset による順位再計算は行わない。
- API body を横断的な envelope へ変更せず、共通 pagination metadata は response headers に置く。
  既存 body consumer と picker は追加 header を無視でき、後方互換を保つ。

### Compact stateless cursor
- v3 token は `v3.<base64url(payload)>.<base64url(tag)>` とし、payload は direction、anchor、target page、
  primary sort value、ID tie-break を曖昧性のない length-prefixed binary で表す。JSON と primary の二重 base64 は
  廃止する。
- tenant ID と query/filter fingerprint は payload に格納せず、version とともに HMAC input の associated data に
  length-prefix 付きで含める。別 tenant / query での再利用は v2 と同様に拒否する。
- tag は HMAC-SHA256 の先頭 128 bit とする。cursor は認可 capability ではなく、全 request が通常の認証・tenant
  authorization を要求するため、128-bit forgery resistance を位置 token に十分な安全余裕とする。
- 新規発行は v3 のみとし、v0 expiry 付き token と v2 無期限 token は既存ルールのまま decode する。未知 version、
  malformed length、invalid direction / anchor、改ざん、tenant/query mismatch は `InvalidRequestError` とする。
- UUID ID は16 byte表現にcompact化し、UUIDでない既存 tie-break値はlength-prefixed textで保持する。
  最大長 primary と UUID を使う代表 fixture で v3 token が同じ境界の v2 token の60%以下であることを保証する。

### User count と active 既定表示
- `AdminUserListResponse.total_users` は tenant 内で lifecycle status が deleted ではない全 User の exact count とし、
  query / status の影響を受けない。画面上部の「総ユーザー」はこの値だけを使い、`AdminSettings.usage.users` と
  page length fallback を廃止する。
- pagination の `total_items` は現在の query / status に一致する件数とする。たとえば全体 73 件、active 50 件なら、
  上部は 73、active 一覧は「全 50 件」と表示する。
- browser route で status 未指定は active と解釈し、API には `status=active` を明示する。API 自体の status 省略
  semantics は「全 non-deleted User」のまま変えない。UI で「すべて」を選んだときは URL を `status=all` とし、
  API request から status を省く。active は canonical default のため URL では status を省略する。
- query / status の変更時は cursor を破棄して先頭へ戻す。compact cursor も query hash に bind し、変更前の
  cursor を新しい filter に適用しない。

### Performance と適用範囲
- page query と count query は独立して並列実行し、response latency を直列和にしない。Users の unfiltered total と
  filtered total が同じ条件なら count 結果を再利用し、異なる場合だけ2つの count を並列実行する。
- User lifecycle status 用に tenant + normalized status の partial expression index を追加し、non-deleted predicate、
  keyset index、trigram indexと組み合わせる。Audit の time/type/user/structured filter は既存 index を使用し、
  representative query plan で sequential full-table scan が latency objective を壊さないことを確認する。
- 既存 SCL の p95 target (Users / Groups / Agents / Applications は 200ms、Audit は 300ms) は、page data と exact metadata を合わせた endpoint latency として維持する。
  default quotaを超える代表データでも planと実測を記録し、targetを満たせない場合に近似値へ黙って切り替えない。
- 新しい bounded context、module、directory convention は追加しないため `ARCHITECTURE.md` / `architecture.yaml` の
  同期は不要とする。

## Plan
1. ADR-160 を起票し、compact stateless cursorをoffset pagination、server-side cursor storage、session-local page mapと
   比較して、ADR-159 の wire format / Link 部分だけを相互参照付きで supersede する。
2. 対象 context の interfaces / model / objectives / scenarios / flows を更新し、SCL 派生物と OpenAPI を再生成する。
3. shared cursor codec / PageRequest / Link builderへv3、page、end anchor、metadata headerを追加し、legacy decodeを保つ。
4. 5一覧のrepository portsとmemory/PostgreSQL adaptersへexact countとend-anchor reverse queryを追加し、schema indexを
   更新して生成コードを同期する。
5. handlersでpage queryとcountを並列実行し、端数last page、空結果、stale cursorを含む4方向contractを実装する。
6. frontend `requestPage` と5 route/pageをmetadata / first / last対応にし、共通 `PageNavigation` とi18nを更新する。
7. Users routeをactive既定へ変更し、上部総数を`AdminUserListResponse.total_users`へ切り替える。
8. contract、adapter、query plan、component、E2E、token length / compatibilityを検証し、completion後doneへ移す。

## Tasks
- [x] T001 [ADR] ADR-160 でcompact stateless cursor、exact count、first/last Linkを決定し、ADR-159を部分supersedeする。
- [x] T002 [SCL] 対象5 interfaces、`AdminUserListResponse`、objectives、scenarios、flowsを更新し、正常・空結果・
  filter変更拒否・count失敗を受け入れ例にする。
- [x] T003 [Shared] v3 cursor、legacy decode、page/end anchor、4方向Link、pagination headersを実装する。
  RED: v3短縮率、v0/v2互換、tamper / tenant / query mismatch、空・端数pageのtestsを先にfail確認
  （scenario `IdManagement.管理者はユーザー一覧をページングしながら安定して閲覧できる` および
  `Audit.管理者は監査ログをページングしながら閲覧でき絞り込み変更でcursorが無効化される`）→ GREEN。
  RED は `PageAnchor` / page metadata / 4方向Link未実装で fail を確認し、
  `just test-go-package ./backend/shared/http/support_http` で GREEN。外部入力のlength-prefix binary parserは
  組み合わせ入力が攻撃面になるため fuzz testを採用し、単純なheader整数parserはproperty test不要と判断した。
- [x] T004 [Persistence] 5一覧のexact countと末尾reverse取得、User status indexをmemory/PostgreSQLへ実装する。
  RED: filter一致count、105件/limit 50のlast 5件、tenant isolation、query plan testsを先にfail確認
  （interfaces `ListAdminUsers` / `ListGroups` / `ListAgents` / `ListAdminApplications` / `ListAdminAuditEvents`）→ GREEN。
  RED は各 repository port の `Count` / end-page method 未定義でfail確認。memory/PostgreSQL adapter tests、User 15,000件の
  status count plan、Audit 2 tenant・計40,000件のcount/page planをGREENにし、schema convergenceも確認した。
- [x] T005 [HTTP] 5 handlersでcountとpageを並列取得し、pagination headersとfirst/prev/next/last Linkを返す。
  RED: 0件、1件、ちょうど、端数、中間、stale cursor、count error contract testsを先にfail確認
  （対象5 list interfaces）→ GREEN。RED は既存first-page contractでrequired headers / `rel="last"` 不在を確認し、
  5 handler packageと全Go suiteをGREEN化した。page/countは `WaitGroup.Go` で並列実行し、count errorはrequest全体を失敗させる。
- [x] T006 [Users] exact `total_users`、active既定route、all URL semanticsを実装する。
  RED: quota usageが0でも実数を表示し、全体73/active50を区別するhandler/component testsを先にfail確認
  （model `AdminUserListResponse`、scenario `IdManagement.管理者はユーザー一覧をページングしながら安定して閲覧できる`）→ GREEN。
  handler testでfilter一致件数とfilter非依存総数を分離し、route unitで未指定→active、all→API status省略を固定した。
  UIはquota usageを参照せずresponseのtotalのみを表示し、同一cursorでのrefresh後もrows/metadata/totalを同期する。
- [x] T007 [UI] 共通navigationと5画面を件数・現在位置・最初/前/次/最後へ対応する。
  RED: 0/0、single page、middle、last、disabled boundary、history/reload/share testsを先にfail確認
  （対象flows）→ GREEN。RED は共通navigationに4方向・metadata表示がなくcomponent testでfail確認。
  strict header parser、5 route/page、562 unit/component testsをGREEN化し、Audit `limit=1` browser E2Eで
  next、reload、URL copy、back/forward、first/last、境界disabledを確認した。
- [x] T008 [Verify] SCL/OpenAPI、schema、Go/UI、E2E、cursor length、legacy compatibility、performanceを検証した。

## Verification
- `just check-scl`
- `just scl-render`
- `just check-api-compat`
- `just check-schema`
- `just sqlc-generate`
- `just verify-go`
- `just verify-ui`
- `just test-ui-e2e`
- `just check`
- 自動: total 0 / 1 / 50 / 51 / 100 / 105、各filter、first/prev/next/lastの往復と端数last pageを検証する。
- 自動: v3が最大長代表fixtureで同じv2 tokenの60%以下、v0/v2 decode、改ざん・別tenant/query・未知version拒否を検証する。
- 自動: User 15,000件と高件数Audit fixtureでcount/page query plan、Users 200ms / Audit 300ms objectiveを検証する。
- 手動: 5画面で件数・ページ位置、境界button、reload、browser back/forward、URL copyを確認する。
- 手動: Usersでstatus未指定がactive、`status=all`が全non-deleted、query/status変更でcursorが消えることを確認する。

## Risk Notes
exact count はkeyset page queryとは異なり一致集合を数えるため、データ量とfilter選択度に比例するDB workを追加する。
特に7年保持のAuditと部分一致User検索は高コストになり得る。page/countの並列化、tenant-first predicate、status / trigram /
audit index、query-plan regression testsでUsers 200ms / Audit 300ms objectiveを守る。近似値やstale cacheは選択しないため、targetを満たせない
data profileが見つかった場合は値を偽らず実装を止め、別のread model設計を相談する。

cursor v3 はpayloadを短くするためtagを128 bitへ切り詰めるが、cursorは認可情報や機密を持たず、通常の認証・認可を
代替しない。tenant/queryをassociated dataにbindし、固定時間比較、厳密なlength/version/anchor検証を維持する。
旧tokenを無期限で受理する期間は既存署名鍵のrotationまで続くため、legacy decode pathも同じnegative testsで保護する。

同時更新下ではtotal countとpage rowが完全な同一snapshotではなく、永続cursorのpage numberも絶対順位を保証しない。
これはwi-346のkeyset semanticsを維持する意図的なtrade-offであり、UIは正確なrequest-time countとnavigation位置を示す。

## Completion
- **Completed At**: 2026-08-09
- **Summary**:
  管理一覧5 interfaceへexact count metadata、first/prev/next/last Link、端数を保つend anchor、compact v3 stateless
  cursorを追加し、v0/v2 decode互換を維持した。Usersはfilter非依存のexact `total_users` とactive既定表示へ切り替え、
  5画面の共通navigation、URL履歴・reload・共有、一覧内refresh後のmetadata同期を実装した。ADR-160、SCL、OpenAPI、
  sqlc生成物を同期し、bounded contextやdirectory conventionは変更していないためarchitecture recordの更新は不要だった。
- **Verification Results**:
  - `just check` - passed（SCL、work item、ID、architecture、traceabilityを含むRA全チェック）
  - `just check-scl` / `just scl-render` / `just check-api-compat` - passed
  - `just sqlc-generate` / `just check-schema` - passed（PostgreSQL schema convergence）
  - `just verify-go` - passed（lint 0 issues、全package race tests）
  - `just test-go-fuzz ./backend/shared/http/support_http 5s` - passed（91,110 executions、panicなし）
  - `just verify-ui` - passed（format、lint、562 unit/component tests、typecheck、build）
  - `just test-ui-e2e` - passed（24 browser scenarios、pagination history/end navigationを含む）
  - performance tests - passed（User 15,000件のstatus count index、Audit 40,000件のcount/pageが300ms未満）
