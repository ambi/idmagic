---
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-16
depends_on: [wi-232-executable-architecture]
---

# UI container と Go source の複雑性を分割し再増加を ratchet で防ぐ

## Motivation
Frontend には 2,677 行・68 local state の Applications page、2,457 行・38 local state の Users page などがあり、文書化済みの thin container / presentation split を満たしていない。Backend にも 800 行を超える非生成 handler/usecase がある。方針を文章だけで運用すると再肥大化するため、意味単位の分割と機械的な上限が必要である。

## Scope
- Applications、Users、Groups、Agents、Settings の page module を route/resource operation 単位へ分割する。
- container、presentation、form validation、API orchestration、i18n の責務を分離する。
- SCIM usecase、OAuth2 authorize handler、IdManagement admin usecase など巨大 Go source を interface/aggregate operation 単位へ分割する。
- Architecture check に UI page/container 400行以下、local state hook 10個以下、非生成 Go source 新規800行超禁止の budget を追加する。
- 分割した presentation/helper/usecase の characterization/unit test を追加する。

## Out of Scope
- UI デザイン刷新。
- API wire contract や認可ルールの変更。
- 行数だけを減らすための機械的ファイル分割。
- generated route/sqlc code への budget 適用。

## Plan
- 最初に現在の挙動と route/API interaction を test で固定し、resource detail/create/edit/list の意味境界で分割する。
- shared mega-hook や巨大 props object へ複雑性を移さず、section-local container と純粋 presentation を使う。
- budget は後続の新規違反を即時 error とし、既存対象は本 WI の task と紐づく期限付き debt として順次解消する。
- Go は package/API を維持しながら operation 別 source へ分割し、循環依存や新しい shared bucket を作らない。

## Baseline Inventory (T001)

### UI — page module (行数 / local state hook 数)
| Feature | File | 行数 | useState 数 | test 行数 | route 構成 |
|---|---|---|---|---|---|
| Applications | `AdminApplicationsPage.tsx` | 2,677 | 81 | 203 (`AdminApplicationsPage.test.tsx`) | 単一 file に `AdminApplicationsPage`(list)/`AdminApplicationDetailPage`/`AdminApplicationEditPage`/`CreateApplicationDialog`/`AssignmentManager` 等を同居。route (`applications_/$applicationId*.tsx`) は Outlet のみの薄い layout で、実体は全て page module 側 |
| Users | `AdminUsersPage.tsx` | 2,457 | 44 | 317 (`AdminUsersPage.test.tsx`) | 同様に `AdminUsersPage`(list)/`AdminUserDetailPage`/`AdminUserEditPage`/`AdminUserCreatePage`/`AdminUserImportPage` を同居。route (`users_/$id*.tsx`, `new.tsx`, `import.tsx`) は薄い |
| Groups | `AdminGroupsPage.tsx` | 984 | 30 | 172 (`AdminGroupsPage.test.tsx`) | list/detail/edit/create を同居 |
| Agents | `AdminAgentsPage.tsx` | 918 | 26 | 153 (`AdminAgentsPage.test.tsx`) | list/detail を同居 |
| Settings | `AdminSettingsPage.tsx` + `BrandingTab.tsx` | 749 + 424 | 22 + 14 | 124+13 / 62+34 | tab 単位である程度分離済みだが `AdminSettingsPage.tsx` 自体は budget (400行/state10) 超過 |

budget 案 (UI page/container 400行以下、local state hook 10個以下) に対し、**5 feature 全て**が超過。Applications と Users が突出。

### UI — 関数境界 (分割の当たり)
- `AdminApplicationsPage.tsx`: list (`AdminApplicationsPage` L384-)、`AdminApplicationDetailPage`(L690-1059, ~370行)、`AdminApplicationEditPage`(L1059-1867, ~800行)、`CategoryManager`/`AssignmentManager`/`CreateApplicationDialog`(L1867-2677) が明確な境界候補。
- `AdminUsersPage.tsx`: list (`AdminUsersPage` L98-)、`AdminUserDetailPage`(L453-838, ~380行)、`AdminUserEditPage`(L1414-1714, ~300行)、`AdminUserCreatePage`(L1921-2065)、`AdminUserImportPage`(L2252-, import 専用フロー) が候補。属性編集系 (`AdminAttributeField` 等) は presentation helper として切り出し可能。

### Go — 非生成 handler/usecase (行数)
| File | 行数 | budget (800行) | test |
|---|---|---|---|
| `backend/oauth2/adapters/http/authorize_handler.go` | 1,028 | 超過 | `authorize_handler_test.go` (228行) |
| `backend/scim/usecases/usecases.go` | 772 | budget 内 (要注視) | `backend/scim/adapters/http/scim_test.go` 等に分散、`usecases.go` 直下の unit test は無し |
| `backend/idmanagement/usecases/admin_users.go` | 734 | budget 内 (要注視) | `admin_users_test.go` (480行) |
| `backend/idmanagement/usecases/admin_agents.go` | 467 | budget 内 | `admin_agents_test.go` (458行) |
| `backend/idmanagement/usecases/admin_groups.go` | 342 | budget 内 | `admin_groups_test.go` (234行) |

800行を明確に超過するのは `authorize_handler.go` のみ（motivation 記述時点から scim/idmanagement 側は既に縮小済みの可能性）。`authorize_handler.go` は `handleAuthorize` / `handleTransaction` / `handleLoginAPI` / `handleTOTPAPI` / `handleConsentAPI` / `handleEndSession` の 6 エンドポイント + throttle/hash 系 helper が同居しており、endpoint 単位の分割候補が明確。

### 示唆
- T002/T003 の分割対象優先度: Applications > Users > Groups > Agents > Settings(行数・state 数の超過幅順)。
- T004 の Go 側対象は実質 `authorize_handler.go` のみ。scim/idmanagement の usecase は現状 budget 内だが、新規 budget 導入時の再肥大化防止(ratchet)は依然有効。
- T005 のテストは、分割後の presentation/helper(例: `AdminAttributeField`, `AssignmentManager`, `CategoryManager`, throttle helper 群)が現状ページ全体 test 経由でしか検証されておらず、単体 test が無い。

### T001 訂正(T002 着手時に判明)
`just verify` 実行時に判明: **T006 の complexity budget/debt の仕組みは依存元 wi-232-executable-architecture が既に `ARCHITECTURE.md` の `complexity:` セクションへ導入済み**で、本 WI 対象ファイルすべてに `wi234-*` debt entry (budget: `ui-page-lines` 400行 / `ui-page-local-state` 10個、Go は `go-source-lines` 800行) が `expires_at: 2026-10-01` で事前登録されていた(T001 の grep 範囲がこのセクションを拾えていなかった)。T002 以降は「新規分割ファイルを budget 内に収めるか、収まらない場合は debt entry の path/ceiling を更新する」運用になる。

## Tasks
- [x] T001 [Baseline] 対象 file の責務、route、state、test coverage を inventory 化する。
- [x] T002 [UI] Applications と Users を route/resource operation 単位へ分割する。
  - [x] Applications: `AdminApplicationsPage.tsx`(2,677行/state81) を `AdminApplicationsListPage.tsx`(324行/state7)・`AdminApplicationDetailPage.tsx`(400行/state3)・`AdminApplicationEditPage.tsx`(860行/state34)・`AdminApplicationAssignments.tsx`・`AdminApplicationCategories.tsx`・`CreateApplicationDialog.tsx`・`AdminApplicationsShared.tsx` へ分割。route 3ファイルの import 先を更新。既存 test (`AdminApplicationsPage.test.tsx`)は新規挙動を作らず list/detail/edit の3ファイルへ再配置(`AdminApplicationsListPage.test.tsx`/`AdminApplicationDetailPage.test.tsx`/`AdminApplicationEditPage.test.tsx`)。list/detail は budget 内。edit は4プロトコル分岐フォームが同居し残存超過のため `ARCHITECTURE.md` の debt entry (`wi234-ui-page-lines-admin-applications-page` → `wi234-ui-page-lines-admin-application-edit-page`, 同 local-state) を新パス/実測値(860行/state34)へ更新。`just verify` green。
  - [x] Users: `AdminUsersPage.tsx`(2,457行/state44) を `AdminUsersListPage.tsx`(560行/state10)・`AdminUserDetailPage.tsx`(488行/state7)・`AdminUserEditPage.tsx`(489行/state11)・`AdminUserCreatePage.tsx`(131行/state2)・`AdminUserImportPage.tsx`(436行/state8)・`AdminUserDialogs.tsx`(削除/無効化ダイアログ、list/detail 共有)・`AdminUsersShared.tsx`(属性/ロール/グループ presentation、list/detail/edit 共有) へ分割。既存 `AdminUsersPrimitives.tsx` は変更なしで再利用。route 5ファイルの import 先を更新。既存 test は list/create/edit/import の4ファイルへ再配置(detail は元々専用 test が無く追加なし)。create は budget 内、他4ファイルは残存超過のため `ARCHITECTURE.md` の debt entry (`wi234-ui-page-lines-admin-users-page` → list/detail/edit/import の4entry、`wi234-ui-page-local-state-admin-users-page` → edit の1entry) を新パス/実測値へ更新。`just verify` green。
- [x] T003 [UI] Groups、Agents、Settings を同じ規約へ分割する。
  - [x] Groups: `AdminGroupsPage.tsx`(984行/state30) を `AdminGroupsListPage.tsx`(139行/state5)・`AdminGroupDetailPage.tsx`(114行/state3)・`AdminGroupCreatePage.tsx`(146行/state3)・`AdminGroupEditPage.tsx`(303行/state12)・`AdminGroupDetailCard.tsx`(list/detail 共有)・`AdminGroupsShared.tsx`(parseRoles/optionalValue)へ分割。route 4ファイルの import 先を更新。既存 test は list/create/edit の3ファイルへ再配置(detail は元々専用 test 無し)。list/detail/create は budget 内。edit のみ local-state 超過のため debt entry (`wi234-ui-page-local-state-admin-groups-page` → `wi234-ui-page-local-state-admin-group-edit-page`, ceiling 12) を更新、lines debt は不要になり削除。
  - [x] Agents: `AdminAgentsPage.tsx`(918行/state26) を `AdminAgentsListPage.tsx`(234行/state6)・`AdminAgentDetailPage.tsx`(231行/state6)・`AdminAgentDetailCard.tsx`(list/detail 共有)・`AgentEditorDialog.tsx`(list/detail 共有)・`AdminAgentsShared.tsx`(StatusBadge/kindLabel/parseRoles/optionalValue)へ分割。route 2ファイルの import 先を更新。既存 test は list 用として `AdminAgentsListPage.test.tsx` へ改名移動。全ファイル budget 内のため `ARCHITECTURE.md` の debt entry (`wi234-ui-page-lines-admin-agents-page`, `wi234-ui-page-local-state-admin-agents-page`) を削除。
  - [x] Settings: `AdminSettingsPage.tsx`(749行/state22) を tab 単位で `GeneralTab.tsx`・`PasswordPolicyTab.tsx`・`ScimTab.tsx`・`AdminSettingsShared.tsx`(displayNameError/passwordPolicyOverride/ReadSetting)へ分割し、`AdminSettingsPage.tsx` 自体は tab 切替のみの薄いコンテナ(157行/state2)に縮小。route は単一のため変更不要。`AdminSettingsPage.test.tsx`(コンテナ経由の統合的な test)は変更不要、`AdminSettingsPage.test.ts`(displayNameError/passwordPolicyOverride の unit test)は import 先を `AdminSettingsShared.tsx` へ更新。全ファイル budget 内のため debt entry (`wi234-ui-page-lines-admin-settings-page`, `wi234-ui-page-local-state-admin-settings-page`) を削除。
  - `just verify` green(3リソースとも)。
- [x] T004 [Go] 800行超の非生成 handler/usecase を意味単位へ分割する。
  - `backend/oauth2/adapters/http/authorize_handler.go`(1,028行)が唯一の対象(T001 訂正時点の再調査で scim/idmanagement 側は既に budget 内と確認済み)。同ディレクトリに既存の concern 別ファイル規約(`authorize_transaction.go`・`authorize_second_factor.go`・`authorize_consent_details.go`・`authorize_enrollment.go`)が既にあり、かつ実装の無い `end_session_handler_test.go`・`login_throttle_test.go` が分割境界を先取りしていたため、それに追従した。
  - `handleAuthorize`(/authorize 本体)のみ `authorize_handler.go` に残し 136行 に縮小。`handleTransaction` → `authorize_transaction.go`。`handleLoginAPI` + サインインポリシー/golden signal helper → 新規 `authorize_login.go`(312行)。`handleConsentAPI` → 新規 `authorize_consent.go`(70行)。`handleTOTPAPI` → 既存 `authorize_second_factor.go`(WebAuthn/リカバリコードと同じ第二要素グループ、300行)。`completeAfterAuthn`/`issueCodeURL` 等の認可完了 helper → 新規 `authorize_completion.go`(199行)。`handleEndSession` → 新規 `end_session_handler.go`(62行、対応 test と命名一致)。throttle/hash/bucket helper → 新規 `login_throttle.go`(166行、対応 test と命名一致)。共有 request/response 型 → 新規 `authorize_types.go`(55行)。
  - 挙動変更なし(関数を package 内でファイル間移動しただけ)。`go build ./...` clean、`just verify-go`(golangci-lint + race test)green。`ARCHITECTURE.md` の debt entry `wi234-go-source-lines-authorize-handler` を削除(budget 内)。ついでに実測と乖離していた stale debt `wi234-go-source-lines-events`(ceiling 1151 だが実測100行)も削除。
  - scim/idmanagement の usecase は現状 budget 内のため対象外(T001 訂正参照)。
- [ ] T005 [Tests] 抽出した presentation/helper/usecase の test を追加する。
- [x] T006 [Architecture] complexity budget と generated exclusion を有効化する — budget/debt の枠組み自体は wi-232 で導入済み。T002/T003/T004 の分割に伴い debt entry を全て新パス/実測値へ追従済み。残 debt は `wi234-ui-page-lines-admin-application-edit-page`・`wi234-ui-page-local-state-admin-application-edit-page`・4件の Users page debt・`wi234-ui-page-local-state-admin-group-edit-page`の計7件、いずれも `expires_at: 2026-10-01`。Go 側の debt entry は無し(全 Go source が budget 内)。
- [ ] T007 [Verify] wire/route 挙動不変、coverage 非低下、全検証を確認する。

## Verification
- `just test-ui-unit`
- `just verify-ui`
- `just test-go`
- `just verify-go`
- `just yaml-check-architecture`
- `just verify`

## Risk Notes
大規模な挙動不変 refactor で merge conflict と回帰が起きやすい。resource 群ごとに小さく完了させ、既存 lifecycle workflow 変更と重なるファイルはその変更の統合後に扱う。
