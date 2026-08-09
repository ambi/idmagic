---
status: completed
authors: [tn]
risk: high
created_at: 2026-08-09
depends_on: []
change_kind: feature
initial_context:
  scl:
    IdManagement:
      - interfaces.ListAdminUsers
      - interfaces.ListGroups
      - interfaces.ListAgents
      - objectives.ListAdminUsersLatency
      - objectives.ListGroupsLatency
      - objectives.ListAgentsLatency
      - scenarios.管理者はユーザー一覧をページングしながら安定して閲覧できる
      - flows.AdminUsers
      - flows.AdminGroups
      - flows.AdminAgents
    Application:
      - interfaces.ListAdminApplications
      - interfaces.ListApplicationAssignments
      - objectives.ListAdminApplicationsLatency
      - flows.AdminApplicationManagement
    Audit:
      - models.AuditEventQuery
      - interfaces.ListAdminAuditEvents
      - objectives.ListAdminAuditEventsLatency
      - scenarios.管理者は監査ログをページングしながら閲覧でき絞り込み変更でcursorが無効化される
      - flows.AdminAuditEvents
    Authentication: [interfaces.ListAuthenticationEventBuckets]
    OAuth2: [interfaces.ListAdminOAuth2Clients, interfaces.ListAdminConsents]
    Provisioning: [interfaces.ListProvisioningDeliveries, objectives.ListProvisioningDeliveriesLatency]
  source:
    - backend/shared/http/support_http/pagination_cursor.go
    - backend/shared/http/support_http/pagination_request.go
    - frontend/src/lib/usePaginatedList.ts
    - frontend/src/features/admin-users/AdminUsersListPage.tsx
  tests:
    - backend/shared/http/support_http/pagination_cursor_test.go
    - backend/idmanagement/user/handlers_http/admin_user_list_pagination_test.go
    - frontend/src/lib/usePaginatedList.test.ts
    - frontend/src/features/admin-users/AdminUsersListPage.test.tsx
  stop_before_reading:
    - backend/idmanagement/user/usecases/user_import.go
    - backend/idmanagement/usecases/data_export.go
affected_spec:
  - { context: IdManagement, kind: interface, element: ListAdminUsers }
  - { context: IdManagement, kind: interface, element: ListGroups }
  - { context: IdManagement, kind: interface, element: ListAgents }
  - { context: IdManagement, kind: objective, element: ListAdminUsersLatency }
  - { context: IdManagement, kind: objective, element: ListGroupsLatency }
  - { context: IdManagement, kind: objective, element: ListAgentsLatency }
  - { context: IdManagement, kind: scenario, element: 管理者はユーザー一覧をページングしながら安定して閲覧できる }
  - { context: IdManagement, kind: flow, element: AdminUsers }
  - { context: IdManagement, kind: flow, element: AdminGroups }
  - { context: IdManagement, kind: flow, element: AdminAgents }
  - { context: Application, kind: interface, element: ListAdminApplications }
  - { context: Application, kind: interface, element: ListApplicationAssignments }
  - { context: Application, kind: objective, element: ListAdminApplicationsLatency }
  - { context: Application, kind: flow, element: AdminApplicationManagement }
  - { context: Audit, kind: model, element: AuditEventQuery }
  - { context: Audit, kind: interface, element: ListAdminAuditEvents }
  - { context: Audit, kind: objective, element: ListAdminAuditEventsLatency }
  - { context: Audit, kind: scenario, element: 管理者は監査ログをページングしながら閲覧でき絞り込み変更でcursorが無効化される }
  - { context: Audit, kind: flow, element: AdminAuditEvents }
  - { context: Authentication, kind: interface, element: ListAuthenticationEventBuckets }
  - { context: OAuth2, kind: interface, element: ListAdminOAuth2Clients }
  - { context: OAuth2, kind: interface, element: ListAdminConsents }
  - { context: Provisioning, kind: interface, element: ListProvisioningDeliveries }
  - { context: Provisioning, kind: objective, element: ListProvisioningDeliveriesLatency }
---

# 管理一覧を正確な集計と共有可能な双方向 cursor URL で閲覧できるようにする

## Motivation
主要な管理一覧は keyset pagination を採用済みだが、UI は取得済みページを累積する
「さらに読み込む」方式で、特定位置の URL を共有できず、前ページへ明示的に戻れない。cursor は
1 時間で失効するため、仮に URL へ載せてもブックマークとして短命である。

ユーザー一覧では検索・状態 filter と概要カードが読み込み済み行だけを対象にしており、複数ページの
tenant では「総ユーザー」「有効」「管理者」「MFA」や検索結果が tenant 全体の値であるかのように見える。
一覧 API の低コスト性を保ちながら、現在位置・条件を URL で再現でき、表示値が嘘にならない UI にする。

## Scope
- **Decision**: ADR-158 の cursor expiry と `rel="prev"` 非対応を再検討し、後続 ADR で部分上書きする。
- **SCL**: cursor 対応済み 10 list interfaces の双方向 Link 契約と無期限 cursor、代表 scenarios・flows・
  objectives、`ListAdminUsers` の tenant-wide query/status filter。
- **Shared pagination**: versioned cursor、forward/backward keyset、`Link rel="next"/"prev"` parsing/building。
- **Persistence**: 各 repository の reverse page query。User は tenant-wide contains 検索と status filter、
  PostgreSQL search column/index、memory contract を追加する。
- **UI**: 現在 `usePaginatedList` を使う Users、Groups、Agents、Applications、AuditEvents を
  一ページ置換型の前へ/次へ UI と URL query 同期へ移行する。
- **User summary**: `AdminSettings.usage.users` による正確な総ユーザーだけを表示し、不正確な
  active/admin/MFA cards を削除する。

## Out of Scope
- offset pagination、総ページ数、任意ページ番号への jump、`rel="last"`。
- 有効ユーザー、管理者、MFA 登録数の新規 summary/read model。必要なら
  `wi-161-large-tenant-performance-foundation` が扱う。
- cursor 未対応 list API の新規 pagination 化。
- 一覧以外の picker/lookup が 200 件 capped である問題。これは WI-161 の検索可能 picker 範囲に残す。
- 時点 snapshot の固定。URL は「第 N ページ」ではなく署名済み query/sort の keyset 位置を表す。

## Design

### Cursor と Link contract
- cursor は tenant ID、query hash、方向、keyset、version を HMAC 署名する。新規 cursor は expiry を
  持たず、認証・認可を代替しない位置 token とする。署名鍵変更時は無効化される。
- 旧 exp 付き cursor は既存 expiry まで decode し、新規発行分だけ無期限形式へ移行する。
- forward request は keyset より後、backward request は keyset より前を index の逆順で最大 limit 件取得し、
  response は canonical ascending/descending order に戻す。current page の first/last key から prev/next を作る。
- `Link` response header は存在する方向だけ `rel="prev"` / `rel="next"` を返す。body は domain data のままとし、
  total count や cursor field を追加しない。
- tenant、query、sort 不一致・署名改ざんは InvalidRequestError。UI は invalid/legacy-expired cursor を
  URL から除去して先頭ページを再取得し、日本語通知を表示する。

### URL 型 UI と user 検索
- route loader は URL の `cursor`、filter、search を API へ渡す。ページ操作は items を置換し、返された
  Link URL に navigation するため、reload・browser history・URL 共有が同じ位置を再現する。
- Users の `query` は username、name、email、ID、roles を case-insensitive contains で tenant 全体から検索する。
  `status` とともに query hash に含め、条件変更時は cursor を削除して先頭へ戻す。
- PostgreSQL は tenant 制約付き normalized search column と trigram GIN index を使い、memory adapter は
  同じ match/order/page contract を実装する。この user-search slice は本 WI が先行所有し、WI-161 の
  broader read-model/search scope では重複実装しない。
- Users route は page と AdminSettings を並列取得する。`usage.users` がある場合だけ「総ユーザー」card を
  表示し、usage が無いときに現在ページ件数へ fallback しない。active/admin/MFA cards は削除する。
- 共通 UI primitive は前へ/次へ、loading、最終方向の非表示、navigation error を辞書化する。

### 適用範囲と移行
- API contract は cursor 対応済み 10 interfaces すべてで統一する。
- この段階でページ UI を持つ Users、Groups、Agents、Applications、AuditEvents の 5 画面を移行する。
  UI がまだ無い／capped lookup として使われる残り interface は API contract と adapter tests のみ更新する。
- 既存 DB の複合 sort index は reverse scan に再利用し、User search のみ schema/index 追加を行う。

## Plan
1. ADR-159 を起票して ADR-158 を部分 supersede し、expiry 無し・bidirectional Link の理由を記録する。
2. 各 context の SCL interface、scenario、flow、objective を更新し、派生物を再生成する。
3. shared cursor/PageRequest/Link parser を versioned bidirectional contract にする。
4. repository/use-case/HTTP を context ごとに reverse paging 対応し、User search schema/index を追加する。
5. frontend page response と hook を `{items, previousURL, nextURL}` 型へ変え、5 route/page を URL navigation へ移す。
6. Users summary/search/filter を正確な server data へ切り替える。
7. contract、adapter、component、E2E、query plan を検証し、completion 後 done へ移す。

## Tasks
- [x] T001 [ADR] ADR-159 を作成し、ADR-158 の expiry と prev 非対応だけを相互参照付きで部分 supersede する。
- [x] T002 [SCL] 10 list interfaces の prev/next Link・無期限 cursor、User query/status、代表 scenario/flow/
  objective を更新し、`just check-scl` / `just scl-render` / `just check-api-compat` を通して派生物と OpenAPI
  baseline を同期した。
- [x] T003 [Shared] versioned cursor と bidirectional Link contract を実装した。RED: 新規 token に version /
  direction がなく expiry が残る `TestSetNextLinkIssuesVersionedForwardCursorWithoutExpiry`、backward direction を
  `PageRequest` が失う `TestParsePageRequestPreservesBackwardDirection`、prev+next builder/issuer が未実装の
  `TestBuildPageLinksIncludesPreviousAndNext` / `TestSetPageLinksSignsPreviousAndNextBoundaries` を先に fail 確認。
  legacy expiry、tamper、tenant/query mismatch を含む shared HTTP package を GREEN（10 list interface contract）。
- [x] T004 [Persistence] 10 interfaces の repository/use-case に canonical order を保つ reverse keyset contract を実装した。
  RED: `TestKeysetPageBeforeAscendingReturnsNearestPageInCanonicalOrder` 等が未定義で fail、GREEN: first/middle/last、
  same-key ID tie-break、途中 insert/delete を shared memory contract で、User/Provisioning の memory/PostgreSQL
  adapter と全 Go suite で tenant isolation を確認した。
- [x] T005 [Search] User query/status と generated normalized search column / partial trigram GIN、memory/PostgreSQL
  adapter を実装した。RED: `TestUserRepositoryListPageFilteredSearchesTenantWideAndSupportsPrevious` が未定義で fail。
  GREEN: case-insensitive name/email/role、status、literal LIKE wildcard、tenant-wide isolation、cursor query binding と
  `TestUserSearchQueryPlanUsesTenantAndTrigramIndex` の `EXPLAIN (ANALYZE, BUFFERS)` を確認した。
- [x] T006 [HTTP] 10 list handlers で存在する方向だけ rel prev/next を返す。RED:
  `TestAdminUserListPreviousLinkReturnsPriorPage` が prev 不在で fail。GREEN: 全10 interface の second-page rel=prev、
  User の実往復、invalid/tampered/別 tenant/query変更、Provisioning status/source_type filter を handler tests で確認した。
- [x] T007 [UI] 共通 `PageNavigation` と5 route/pageを一ページ置換・URL cursor navigationへ移行した。
  `requestPage` の prev/next parse と共通 control の両方向/終端、各 loader のURL復元・invalid cursor先頭復帰、既存
  browser-history testsを含む558 component/unit testsをGREENにした。
- [x] T008 [Users UI] server query/status と `AdminSettings.usage.users` の単一 cardへ移行し、page-local 3 metrics と
  page-size fallbackを削除した。usage missing時の非表示、filter変更時cursor非継承、URL初期値、無効cursor通知を
  component/type testsで確認した。
- [x] T009 [Verify] 全 context contract、query plan、UI E2E、共有 URL の確認を実施した。50件超の手動 seed は
  全10 interface の `limit=2` handler往復テスト、5画面の一ページ置換/URL/history component testで同じ境界を
  決定的に検証した。1時間の実時間待機は、新規token payloadに `exp` が存在しないcodec testで置き換えた。

## Verification
- `just check`
- `just scl-render`
- `just check-api-compat`
- `just check-schema`
- `just verify-go`
- `just verify-ui`
- `just test-ui-e2e`
- 手動: 50 件超の各 resource で next/prev、reload、browser back、URL copy を確認する。
- 手動: cursor 発行から 1 時間超後も URL が有効で、改ざん・別 tenant cursor は先頭へ安全に復帰することを確認する。
- 手動: User query の PostgreSQL plan が tenant 条件と trigram index を使用し、page 内検索にならないことを確認する。

## Risk Notes
cursor は URL に長期間残るが認可 capability ではなく、全 request で通常の認証・tenant authorization を要求する。
HMAC に tenant/query/sort/direction/keyset を含め、改ざん・横断利用を拒否する。無期限 token は署名鍵 rotate で
一括無効化できる。reverse paging と同時更新では「固定された第 N ページ」を保証せず、keyset 境界から見た
重複なしの navigation を保証する。User contains search は index と tenant predicate を必須にし、WI-161 の
大規模 scale profile でも query cost を検証可能な形にする。

cursor / Link は再帰や組み合わせ文法を持たず認可判断も行わない固定形式 parser なので、本 WI では fuzz test を
追加しない。攻撃面は署名改ざん、tenant/query mismatch、legacy expiry、未知 version/direction の例示テストで覆う。

## Completion
- **Completed At**: 2026-08-09
- **Summary**:
  管理一覧10 interfaceをversioned・無期限・双方向の署名付きkeyset cursorへ統一し、5つの管理画面を
  URLで共有・復元可能な一ページ置換型navigationへ移行した。ユーザー一覧はtenant全体のquery/status検索、
  PostgreSQL trigram index、正確なusage totalを使用し、page-localの不正確な集計を廃止した。ADR-159、SCL、
  OpenAPI派生物、architecture design record/ledgerも同期した。
- **Verification Results**:
  - `just verify` - passed（check、API compatibility、traceability、tools、Go、UIの11標準ゲート）
  - `just scl-render` - passed
  - `just check-schema` - passed（空DB適用、dry-run、再適用のschema convergence）
  - `just verify-go` - passed（lint 0 issues、全package race tests）
  - `just verify-ui` - passed（format、lint、558 unit/component tests、typecheck/build）
  - `just test-ui-e2e` - passed（23 browser scenarios）
  - `TestUserSearchQueryPlanUsesTenantAndTrigramIndex` - passed（tenant predicateと`users_search_text_trgm_idx`）
  - pagination contract tests - passed（10 interfaceのprev/next、往復、改ざん、別tenant/query、legacy expiry、
    新規tokenのexpiry非保持）
