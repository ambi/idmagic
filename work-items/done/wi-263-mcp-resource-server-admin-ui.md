---
status: completed
authors: ["tn"]
risk: low
created_at: 2026-07-19
depends_on: [wi-56-mcp-authorization-server]
---

# McpResourceServer の管理フロントエンド UI を追加する

## Motivation
[[wi-56-mcp-authorization-server]] は `McpResourceServer` (MCP resource server 登録、
RFC 9728 Protected Resource Metadata / RFC 8707 resource indicator の基準) の管理 API
(`/api/admin/mcp-resource-servers` の list/get/create/update/delete) を実装したが、
フロントエンドの管理画面は追加しなかった（API のみ、`## Out of Scope` で開示済み）。
運用者は現状 curl 等で直接 API を叩く必要があり、他の tenant-scoped registry
（例: `AuthorizationDetailType`）と同様に管理コンソールから操作できるようにする。

本 WI は [[wi-262-mcp-resource-indicator-remaining-grants]]（resource indicator の
グラント対応拡張、バックエンドのみ）とは無関係な関心事として分離している。

## Scope
- フロントエンドのみ。バックエンド API・SCL に変更はない
  (`McpResourceServerResponse`/admin CRUD interfaces は wi-56 で確定済み)。
- SCL review: `spec/contexts/oauth2.yaml` の `models.McpResourceServer` /
  `models.McpResourceServerRequest` / `models.McpResourceServerResponse` /
  `models.McpResourceServerListResponse` と
  `interfaces.ListAdminMcpResourceServers` / `CreateAdminMcpResourceServer` /
  `UpdateAdminMcpResourceServer` / `DeleteAdminMcpResourceServer` を既存契約として利用する。
  UI は契約の意味を変えないため SCL 自体は変更しない。

## Out of Scope
- バックエンド側の変更（API・SCL は無変更で流用する）。

## Plan
承認済みプラン `/Users/tn/.claude/plans/wi-56-2-1-2-precious-walrus.md` の Phase D を正本とする。
`frontend/src/features/admin-authz-detail-types/`（`AuthorizationDetailType` 管理画面）を
構造的な型として完全に踏襲する: 一覧は Card 積み上げ、作成/編集はページ上部の
インラインフォーム（モーダル/ドロワーなし）、汎用テーブルコンポーネントは使わない
（このコードベースには存在しないため新規に作らない）。

- `frontend/src/types.ts` — `McpResourceServer` 型を追加
  (`tenant_id`/`resource_server_id`/`resource`/`name`/`scopes: string[]`/
  `state: 'Active' | 'Disabled'`/`created_at`/`updated_at`)。
- `frontend/src/api/admin.ts` — `McpResourceServerInput` 型と
  `listMcpResourceServers`/`createMcpResourceServer`/`updateMcpResourceServer`/
  `deleteMcpResourceServer` を、`AuthorizationDetailType*` ラッパーの形をそのまま
  踏襲して追加する（list は `{ resource_servers: [...] }` を展開、update/delete は
  `resource_server_id` をパスパラメータにする）。
- `frontend/src/features/admin-mcp-resource-servers/`:
  - `AdminMcpResourceServersPage.tsx` — フォーム項目: `resource`（テキスト、編集時は
    disabled — バックエンド側で不変）、`name`（テキスト）、`scopes`（空白/カンマ区切りの
    テキスト入力を送信時に `string[]` へ分割・表示時に結合 — タグ入力コンポーネントは
    このコードベースに存在しないため使わない）、`state` セレクト（`Active`/`Disabled`、
    作成時は既定 `Active`）。一覧行は resource（等幅・主要）、状態バッジ
    （`AdminAuthorizationDetailTypesPage.tsx` の emerald/slate バッジクラスを再利用）、
    name、scope チップを表示する。
  - `AdminMcpResourceServersPage.i18n.ts` — `AdminAuthorizationDetailTypesPage.i18n.ts`
    と同じキー形状の ja/en 辞書。
  - `AdminMcpResourceServersPage.test.tsx` — 既存テストファイルと同型
    （en/ja 描画、空状態）。アサーションは `adminMcpResourceServersDictionary.en.xxx`/
    `.ja.xxx` を参照し、翻訳済み文字列をハードコードしない。
- `frontend/src/routes/admin/mcp-resource-servers.tsx` — 新規 TanStack Router
  file route（`routes/admin/authorization-detail-types.tsx` を踏襲、
  `requirePortalAccount('admin', ...)` + `listMcpResourceServers()` を loader で、
  `PageMarker kind="admin-mcp-resource-servers"` でページをラップ）。
- `frontend/src/lib/adminNav.ts` — `AdminNavKey` に `'mcp-resource-servers'` を追加し、
  ナビ項目（アイコンは既存インポートのスタイルに合う候補、例 `IconPlugConnected`。
  実装時に見た目で確定）、ラベル `t.mcpResourceServers`、href
  `tenantURL('/admin/mcp-resource-servers')`。
- `frontend/src/components/shell.i18n.ts` — `mcpResourceServers` ラベル ja/en を追加。
- `frontend/src/routes/-page.tsx` — `PAGE_TITLES` に `'admin-mcp-resource-servers'` を追加。
- `routeTree.gen.ts` は `just dev-ui`/`just build-ui` 実行時に自動再生成される
  （手動編集不要）。

## Tasks
- [x] T001 [Frontend/Adapter] `types.ts`/`api/admin.ts` に型と CRUD ラッパーを追加する。
      RED: admin API client test を先に fail 確認
      (`interfaces.ListAdminMcpResourceServers` / `CreateAdminMcpResourceServer` /
      `UpdateAdminMcpResourceServer` / `DeleteAdminMcpResourceServer`) → GREEN。
- [x] T002 [Frontend/UI] `admin-mcp-resource-servers` feature（page/i18n/test）を追加する。
      RED: en/ja・空状態・scope 表示/入力の component test を先に fail 確認
      (`models.McpResourceServerResponse` / `models.McpResourceServerRequest`) → GREEN。
- [x] T003 [Frontend] ルート・ナビ・PAGE_TITLES を配線する。
- [x] T004 [Verify] `just verify-ui`（format-check/lint/typecheck/build/test-ui-unit）を通し、
      実ブラウザで一覧・作成・編集・削除を確認する。

## Verification
- `just verify-ui`（format-check / lint / typecheck / build / test-ui-unit）
  - reason: 新規 admin 画面の型・lint・build・component test。
- E2E: `SEED_ENVIRONMENT=development SEED_PROFILE=development just test-ui-e2e` で、
  管理者として「MCP resource servers」へ到達し、一覧・登録・編集・削除が実 API を通して
  往復することを Bun.WebView で確認する。

## Risk Notes
バックエンド API・SCL は無変更で流用するため、フロントエンドのみの低リスクな追加。
既存ナビ項目・他画面への影響はない。
scope の空白/カンマ区切りは単純な固定形式で認証・認可判断を行わないため、
fuzz/property test は採用せず component test で変換を確認する。

## Completion

- **Completed At**: 2026-07-20
- **Summary**:
  McpResourceServer の tenant-scoped 管理 UI、型付き API client、管理ナビと route を追加し、
  実ブラウザ相当 E2E で一覧・登録・編集・削除の往復を確認した。
- **Affected Guarantees State**:
  - 既存の McpResourceServer admin CRUD 契約と TenantAdministrator access policy を変更せず、管理 UI から利用可能にした。
  - resource は編集時に変更不可で、scope は空白またはカンマ区切りの入力を API の `String[]` へ変換する。
  - SCL、バックエンド API、永続化、McpResourceServer の発行・検証セマンティクスは変更していない。
- **Verification Results**:
  - `just verify-ui` — passed（74 test files and 409 tests、format / lint / typecheck / build）
  - `just verify` — passed（SCL / Work Item / Architecture / Go / UI）
  - `SEED_ENVIRONMENT=development SEED_PROFILE=development just test-ui-e2e` — passed（UI action 14/14、smoke 3/3、session recovery 1/1、OIDC golden path 2/2）
- **Evidence**:
  API client contract、英日表示、空状態、scope 変換の unit/component tests と、Bun.WebView による
  登録・編集・Disabled 化・削除の実 API 往復を確認した。
