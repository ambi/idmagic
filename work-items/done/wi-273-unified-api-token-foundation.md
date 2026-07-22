---
status: completed
authors: [tn]
risk: high
created_at: 2026-07-22
depends_on: []
---

# scope 付き統一 API アクセストークン基盤を導入し、SCIM トークンをそこへ統合する

## Motivation

管理用 API をアクセストークンで公開する（[api-management-api の計画] 参照）ための土台。現状トークンは SCIM 専用の `scim_tokens`（tenant + sha256 hash + description + expiry、**scope なし**）しかなく、認証も SCIM ハンドラ内の非公開メソッドに閉じていて再利用できない。管理 API を scope で細かく認可するには、scope を持つ統一トークンと、複数 bounded context から使える発行/失効/認証の基盤が要る。

未リリースのため破壊的変更が可能。`scim_tokens` を廃し、scope 付きの統一テーブル `api_tokens` に一本化する。SCIM アクセスは `scim` scope の一つとして表現し、この時点で SCIM が新トークンで動作するところまでを本 WI のゴールとする。管理 API 側の認可カーネル・エンドポイント公開は後続 WI（認証カーネル / 各 context 公開）で行う。

## Scope

- 新規 `spec/contexts/api-tokens.yaml`（新 bounded context `apitokens`）
  - `models`: `ApiToken` entity（`identity: [tenant_id, id]`、`scopes` を持つ）、`ApiTokenScope` 相当の scope 語彙、関連エラー。
  - `interfaces`: `ListApiTokens` / `IssueApiToken` / `RevokeApiToken`（`/api/admin/api-tokens`）。
  - `authorization`: 発行/失効/一覧を守る `TenantAdministrator` 系 principal/policy。
  - `glossary`: API アクセストークン・scope の定義。
- `spec/scl.yaml` の `context_map` に `ApiTokens` context を追加。
- `spec/contexts/scim.yaml`
  - `models.ScimToken`（192-209 付近）と token admin interfaces（`ListScimTokens`/`CreateScimToken`/`RevokeScimToken`、614-670 付近）を削除。
  - SCIM 保護エンドポイントの認証を `api_tokens` の SCIM 専用 scope で行うよう invariant/capability を改訂。scope は `scim:users:read` / `scim:users:write` / `scim:groups:read` / `scim:groups:write` に分割し、各 SCIM interface の `access.policies` を対応 policy（`ScimReadUsers`/`ScimWriteUsers`/`ScimReadGroups`/`ScimWriteGroups`）へ差し替え。discovery endpoint（ServiceProviderConfig/ResourceTypes/Schemas）は `scim:*` のいずれかで参照可能（`ScimDiscovery`）。
  - `ManageScimSettings` policy・`TenantAdministrator` principal（token 管理専用だった）を撤去。
  - scope 語彙は `ApiTokenScope` enum に集約。全 `/api/admin/*` + SCIM の各エンドポイントを scope へマッピングし網羅性を確認済み（アプリ領域は [wi-274] に保留、認証/公開/アカウント自己管理は対象外、トークン発行/失効は管理者セッション限定）。

## Out of Scope

- 管理 API の認可カーネル（PAT 解決 / `AuthorizeAdminOrScope` / CSRF スキップ / 監査アクター帰属）— 後続 WI。
- users/groups/agents 等の管理 API 公開・ページネーション — 後続 WI。
- レートリミット（wi-27 で対応済み、スコープ外）。

## Plan

- **トークン値**: `idmagic_pat_<hex64>` の識別プレフィックス付き。保存は sha256 hash のみ。生成/検証は `backend/scim/usecases/usecases.go` の `GenerateToken`/`AuthenticateToken` を踏襲し `backend/apitoken/` へ。
- **配置**: 横断関心のため新 bounded context `apitoken`（`backend/apitoken/{domain,usecases,ports,db_postgres,db_memory,handlers_http,module.go}`）。`backend/scim/` をレイアウトの手本にする。
- **DB**: `infra/schema/postgres.sql` の `scim_tokens`（781 付近）を `api_tokens`（`scopes TEXT[] NOT NULL` 追加、`(tenant_id, created_at)` index）へ置換。`schema_test.go` を更新。
- **SCIM 移行**: `backend/scim/handlers_http/handlers.go:28` の `authenticate` を apitoken 認証（`scim:users:read`/`scim:users:write`/`scim:groups:read`/`scim:groups:write` の該当 scope 要求、discovery は `scim:*` いずれか）へ差し替え。SCIM の token 発行/失効ハンドラ・usecase・repo・SQL を撤去し `backend/apitoken` へ集約。
- **UI**: `frontend/src/features/admin-settings/ScimTab.tsx` を「API アクセストークン」タブへ一般化し、発行時に scope 選択を追加。API クライアント `frontend/src/api/admin.ts:1158-1174` を `list/issue/revokeApiToken`（scopes 付き）へ。型 `frontend/src/types.ts:875` に `scopes` 追加。i18n は辞書値参照。
- **ADR**: 「トークンモデル統合（scim_tokens 廃止 → api_tokens、scope 付き）」を `new-adr` で記録。

## Tasks

- [x] T001 [SCL] 新 context `api-tokens.yaml`（models/interfaces/authorization/glossary/scenarios）を追加し、`scl.yaml` context_map に登録。
- [x] T002 [SCL] `scim.yaml` から token モデル/glossary/interface/scenario を削除し、SCIM 認証を `scim` scope 化（`ScimBearerClient` に `"scim" in context.token_scopes`）。
- [x] T003 [SCL] `just yaml-check`（全 23 SCL OK）/ `just scl-render`（OpenAPI に api-tokens 反映、scim/tokens 消滅）で確認。
- [x] T003b [Arch] 新 bounded context `ApiTokens` を `ARCHITECTURE.md` に同期（`new-architecture`）。
- [x] T004 [DB] `postgres.sql` を `api_tokens` へ置換、`schema_test.go` 更新。
- [x] T005 [App] `backend/apitoken` パッケージ（domain/usecases/ports/db_postgres/db_memory）を実装。
- [x] T006 [App] 発行/失効/一覧の admin ハンドラ + route 登録。
- [x] T007 [App] SCIM `authenticate` を apitoken（`scim` scope）へ差し替え、旧 SCIM token 経路を撤去。
- [x] T008 [UI] トークン管理タブを一般化（scope 選択 + `idmagic_pat_` 一度表示）、api クライアント/型/i18n 更新。
- [x] T009 [ADR] トークンモデル統合の ADR を記録。
- [x] T010 [Verify] 下記検証を実施。
- [x] T011 [Domain] RED: `TestParseTokenLiteral` / `TestParseScopesAndMembership` を先に未実装 symbol で fail 確認（scenario `APIアクセストークンは有効なscope付きtokenだけを認証する`）→ GREEN。
- [x] T012 [UseCase] RED: `TestIssueListAndRevokeApiToken` / `TestIssueRejectsInvalidInput` / `TestAuthenticateApiTokenFailClosed` を先に未実装 symbol で fail 確認（interfaces `IssueApiToken` / `ListApiTokens` / `RevokeApiToken` / `AuthenticateApiToken`）→ GREEN。
- [x] T013 [Adapter] RED: memory/PostgreSQL roundtrip、admin HTTP、SCIM route 別 scope のテストを先に fail 確認（policies `ManageApiTokens` / `ScimDiscovery` / `ScimReadUsers` / `ScimWriteUsers` / `ScimReadGroups` / `ScimWriteGroups`）→ GREEN。
- [x] T014 [UI] RED: API client path/scopes と scope 選択・一度表示・失効 UI のテストを先に fail 確認（scenario `管理者はAPIアクセストークンを発行・失効できる`）→ GREEN。

## Verification

- `just yaml-check` / `just yaml-check-work-items` / `just check-ids` / `just scl-render`。
- `just build-go` / `just test-go`（apitoken usecase 単体、SCIM 認証切替、schema_test）。
- `just verify-ui` / `just test-ui-e2e`（トークン発行・scope 選択・一度表示・失効）。
- 実クライアント（`run`/`verify` skill で dev 起動後）:
  - UI で `scim:users:read`+`scim:users:write` scope トークンを発行 → SCIM クライアント（`Authorization: Bearer idmagic_pat_...`）で Users CRUD 可能、Groups 書込みは 401。
  - discovery（ServiceProviderConfig 等）は scim:* いずれかで 200。失効後・scope なしは 401。

## Follow-up / Open items

- **scope 網羅性**: 全 `/api/admin/*` + SCIM を scope へマッピング済み（ギャップ0）。アプリ領域（applications/oauth-clients/saml/wsfed/provisioning）は [wi-274]、アカウント自己管理（account:*）は [wi-275] で定義・enforcement。
- **system_admin / control-plane**: `tenants:*` はテナント横断のコントロールプレーン操作。API トークンはテナント境界内が既定のため、これらを叩ける system スコープトークンの扱いは認証カーネル実装時に決める（優先度低）。
- 認証カーネル・監査アクター帰属・CSRF スキップは後続 WI（管理 API 認証カーネル）で実装。
- **`ARCHITECTURE.md` 同期**（T003b）。

## Risk Notes

- **破壊的変更（高）**: `scim_tokens` 廃止で既存 SCIM トークンは全て無効化。未リリース前提のため許容。移行スクリプトは作らない。
- **認証経路の差し替え**: SCIM 認証のリグレッションが本番相当機能に影響しうる。SCIM の既存 e2e/単体で回帰を担保する。
- **新 bounded context 追加**: `ARCHITECTURE.md` の同期が必要（core 構造変更）。
- **fuzz/property test 判断**: 不採用。認証 token の入力文法は固定 prefix + 64 桁 hex で再帰・組み合わせ爆発がなく、domain/usecase の表駆動境界テストと route 別 scope の adapter テストで攻撃面を直接検証する。

## Completion

- **Completed At**: 2026-07-23
- **Summary**:
  SCIM 専用トークンを scope 付き統一 API アクセストークンへ置換し、発行・一覧・失効、永続化、
  SCIM route 別認可、管理 UI を SCL-first で実装した。
- **Affected Guarantees State**:
  - API アクセストークンの平文は発行時に一度だけ返し、永続化は SHA-256 hash のみに限定する。
  - SCIM の全 15 route は discovery/users/groups の read/write scope に応じて fail closed で認可する。
  - tenant 境界、期限切れ、失効済み、未知・空 scope のトークンを認証しない。
- **Verification Results**:
  - `just yaml-check-scl` — passed（SCL 23 files）
  - `just check-ids` — passed
  - `just scl-render` — passed
  - `just verify-go` — passed
  - `just verify-ui` — passed
  - `just test-ui-e2e` — passed（UI action 15/15、smoke 3/3、session recovery 1/1、OIDC golden path 2/2）
- **Evidence**:
  domain/usecase、memory/PostgreSQL adapter、admin HTTP、SCIM route 別 scope の RED/GREEN test と、
  管理 UI の scope 選択・一度表示・一覧・失効を実 API 経由で確認した。
- **Known Repository Baseline**:
  当時の repository-wide check は、既存 wi-216 の Completion 欠落と
  `AdminUserEditPage.tsx` の complexity debt により非ゼロ終了だった。
