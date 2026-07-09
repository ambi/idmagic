---
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-07-05
---

# SCIM の有効/無効フラグを廃止し用語を SCIM アクセストークンへ統一する

## Motivation
SCIM はプロトコル API であり、テナントごとに明示的な有効/無効フラグ
(`ScimConfig.enabled`) を持たせる必然性がない。有効なアクセストークンが
1 本もなければ API は事実上利用できず、トークン発行がそのまま有効化に
等しいため、`enabled` フラグはトークンの有無と冗長で、UI も両者を連動
させているだけだった。ユーザ判断として SCIM は常時有効とみなし、有効/無効
の概念とデータ (カラム) を完全に廃止する。

あわせて UI 用語を整理する。実体は OAuth Bearer トークンなので「SCIM 認証
トークン」より「SCIM アクセストークン」が正確で、UI 内で「認証トークン」
「Bearer トークン」が混在していたのを「アクセストークン」に統一する。
日本語 UI に混じっていた英語「Inbound Provisioning」も日本語表現に置き換える。

## Scope
- **scl** (`spec/contexts/scim.yaml`):
  - `models.ScimConfig` エンティティを削除。
  - `interfaces.GetScimConfig` / `interfaces.UpdateScimConfig` を削除。
  - `permissions.ManageScimSettings` の `resource` を `ScimConfig` から
    `ScimToken` へ変更し description を「SCIM アクセストークンを管理する権限」に。
  - `scenarios` の precondition「SCIM Bearer Token が有効化されている」を
    「有効な SCIM アクセストークンが発行されている」に修正。
  - token 系 interface の description を「アクセストークン」表記に統一。
  - `glossary.ScimToken` を追加 (aliases: SCIM アクセストークン / access token)。
    認証方式を仕様として決めつけない中立的定義とする。
- **go**:
  - `internal/scim/ports/repository.go`: `ScimConfig` 型と `GetConfig`/`SaveConfig`
    を削除。
  - `internal/scim/usecases/usecases.go`: `GetConfig`/`UpdateConfig` を削除。
  - `internal/scim/adapters/http/handlers.go`: `authenticate` から enabled
    ゲートを除去 (有効なトークンで認証成功すれば通す)。`handleGetAdminConfig`
    / `handleUpdateAdminConfig` と `scimConfigResponse` / `scimConfigUpdateRequest`
    を削除。
  - `internal/scim/adapters/http/routes.go`: `/api/admin/scim/config` の
    GET/PUT ルートを削除。
  - `internal/shared/adapters/persistence/memory/scim.go`: `configs` フィールドと
    `GetConfig`/`SaveConfig` を削除。
  - `internal/shared/adapters/persistence/postgres/scim.go`: `GetConfig`/`SaveConfig`
    を削除。
- **db**:
  - `deploy/schema/postgres.sql`: `scim_configs` テーブル定義を削除。
  - `internal/shared/adapters/persistence/postgres/schema_test.go`: 期待テーブル
    一覧から `scim_configs` を除去。
- **ui**:
  - `ui/src/types.ts`: `ScimConfig` 型を削除。
  - `ui/src/api/admin.ts`: `getScimConfig` / `updateScimConfig` と関連 import を削除。
  - `ui/src/features/admin-settings/AdminSettingsPage.tsx`: 有効/無効トグルと
    無効時ビューを撤去し常にトークン管理を表示。「SCIM 認証トークン」→
    「SCIM アクセストークン」、「Bearer トークン」→「アクセストークン」に統一。
    見出し・説明から「Inbound Provisioning」英語を日本語化。

## Out of Scope
- OAuth 2.0 トークンエンドポイント経由での SCIM アクセストークン発行の実装
  (仕様上は排除しないが本 WI では作らない)。
- Outbound SCIM Provisioning (wi-45)。
- SCIM プロビジョニングロジック自体の挙動変更。

## Verification
- `just yaml-check-scl` が OK。
- `go build ./...` / `go vet ./...` / `go test ./...` が pass。
- `grep -rn "ScimConfig" internal/ spec/ deploy/` が実装・仕様に残らない。
- `grep -rn "認証トークン\|Inbound Provisioning" ui/src` が空。
- UI ビルド (`bun run build` 等) が通る。

## Risk Notes
`scim_configs` テーブル削除は破壊的スキーマ変更。単一 schema ファイル運用の
ため既存データは移行不要だが、既に enabled=true で運用中のテナントがあれば
挙動が「常時有効」に変わる点に留意 (今回はそれが意図)。

## Completion
- **Completed At**: 2026-07-05
- **Summary**:
  SCIM を常時有効とみなす方針に統一し、テナントごとの有効/無効設定
  (`ScimConfig` エンティティ・`scim_configs` テーブル・`/api/admin/scim/config`
  エンドポイント・UI トグル) を全廃した。SCIM API のゲートは「有効な
  アクセストークンで認証成功すること」のみに簡素化した。UI 用語を
  「認証トークン」「Bearer トークン」から「アクセストークン」へ統一し、
  日本語 UI に混在していた英語「Inbound Provisioning」を日本語表現に置換した。
  glossary には認証方式を仕様として決めつけない中立的な `ScimToken` 定義を追加し、
  将来の OAuth 2.0 発行アクセストークン受け入れの余地を残した。
- **Verification Results**:
  - `just yaml-check-scl`
    - result: ok (All 12 file(s) OK)
  - `go build ./...`
    - result: ok
  - `go vet ./...` / `just lint-go`
    - result: ok (0 issues)
  - `go test ./...`
    - result: ok (0 failures)
  - `just typecheck-ui` / `just lint-ui` / `just build-ui`
    - result: ok
  - `grep -rn "ScimConfig|scim_configs" internal/ spec/contexts deploy/schema`
    - result: empty
  - `grep -rn "認証トークン|Inbound Provisioning" ui/src`
    - result: empty
