---
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-01
depends_on: []
affected_spec:
  - { path: spec/contexts/identity-management/SPECIFICATION.md, requirement: REQ-IDMANAGEMENT-024 }
  - { path: spec/contexts/tenancy/SPECIFICATION.md, requirement: REQ-TENANCY-020 }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.Group }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.GroupAttributeDef }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.Contract.IdentityManagement.CreateGroup }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.Contract.IdentityManagement.UpdateGroup }
  - { path: spec/contexts/tenancy/models.tsp, symbol: IdMagic.Contract.TenantGroupAttributeSchema }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Contract.Tenancy.GetTenantGroupAttributeSchema }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Contract.Tenancy.UpdateTenantGroupAttributeSchema }
---

# グループにメールアドレスとカスタム属性を追加できるようにする

## Motivation

[[wi-314-hosted-and-admin-console-ui-wording-and-navigation-consistency]] のレビューで、
グループが持てる属性が名前・説明・ロール・メンバー所属のみであり、ユーザーが持てる
カスタム属性（[[wi-19-rich-user-attributes]] 参照）や連絡先情報に相当するものを
グループには一切持たせられない点が指摘された。組織のグループ（部署・チームなど）が
連絡先メールアドレスや任意の管理用属性を持てないのは、実運用（例: グループ宛のメール
配信、外部システム連携用のメタデータ保持）を想定すると狭すぎる。

バックエンドのグループ集約・永続化スキーマの変更を伴うため、UI 文言・ナビゲーション整理を
扱う wi-314 の範囲からは明示的に外し、本 WI として切り出す。

## Scope

- `spec/contexts/identity-management/{models,main}.tsp`・`SPECIFICATION.md`（`Group` への
  `email`/`attributes` フィールド追加、`GroupAttributeDef` 新設）
- `spec/contexts/tenancy/{models,main}.tsp`・`SPECIFICATION.md`（`TenantGroupAttributeSchema`
  新設、`GetTenantGroupAttributeSchema`/`UpdateTenantGroupAttributeSchema` API）
- グループ集約のドメイン・ユースケース・永続化層（Postgres スキーマ変更を含む）
- `frontend/src/features/admin-groups/`（グループ詳細・編集画面への属性入力欄追加）

## Out of Scope

- カスタム属性の型システムをユーザー属性と完全に共通化する設計。設計時に評価した結果、
  `GroupAttributeDef`/`TenantGroupAttributeSchema` は `UserAttributeDef`/
  `TenantUserAttributeSchema` とは別の並行した機構として新設する方針とした（Design 節参照）。
  値の型 (`AttributeValue`/`AttributeType`) は共有する。
- SCIM 経由でのグループ属性の外部連携（別途の provisioning 対応が必要になる場合は
  さらに別 WI とする）。

## Design

- **email**: `Group.email` を一級フィールドとして追加する（ユーザーの `email` が
  一級フィールドであることとの一貫性を優先）。`User.email` と同じ形式検証のみを行うが、
  グループには self-service actor が存在しないため、検証済みフラグ・変更確認フロー・
  一意制約は持たない。
- **attributes のスキーマ方式**: Okta / Microsoft Entra ID はグループ属性も User 属性と
  同様に管理者が事前定義するスキーマ駆動方式を取る（Keycloak は自由記述の
  key/value）。本 WI は既存の `User` 属性機構（wi-19）と一貫させ、スキーマ駆動方式を採用する。
- **共有範囲**: `UserAttributeDef`/`TenantUserAttributeSchema` をそのまま Group に
  拡張するのではなく、`GroupAttributeDef`/`TenantGroupAttributeSchema` を新設する。
  理由は、`UserAttributeDef` が持つ `editable_by_user`（self-service 編集）・
  `claim_name`/`oidc_scope`（OIDC claim 露出）は Group に対応する概念が無く、また
  `User` 側は OIDC §5.1 / SCIM `enterprise:User` 相当の builtin catalog を持つが
  Group には対応する標準語彙が存在しないため、無理に共通化すると意味のないフィールドが
  増えるだけになる。値の型 (`AttributeValue`/`AttributeType`) は共通のまま再利用し、
  属性定義の「二重管理」は key の意味論が違う（User の custom attribute と Group の
  custom attribute は本質的に別のスキーマ）ため許容する。
- **配置**: `TenantGroupAttributeSchema` は `TenantUserAttributeSchema` と同じ配置方針
  （tenant-scoped 独立 aggregate、port/usecase/HTTP は `Tenancy` コンテキストが所有、
  tenant 削除時に cascade）を踏襲する。
- **マイグレーション**: `groups.email` は NULL 許容、`groups.attributes` は
  `JSONB NOT NULL DEFAULT '{}'::jsonb` で追加し、既存行は無変更のまま後方互換を保つ。
  `tenant_group_attribute_schemas` は新規テーブル。

## Plan

1. **Domain**: `backend/idmanagement/group/domain/groups.go` に `Email *string` /
   `Attributes map[string]userdomain.AttributeValue`（`userdomain.AttributeValue` を
   再利用）、`GroupAttributeDef`、`TenantGroupAttributeSchema`、
   `ValidateGroupAttributes`/`ValidateGroupAttributeValue` を追加する。
2. **Usecase**: `CreateGroup`/`UpdateGroup`（`group/usecases/admin_groups.go`）に email
   形式検証・attributes のスキーマ検証を追加する。`backend/tenancy/usecases/
   manage_group_attribute_schema.go` に `GetGroupAttributeSchema`/
   `UpdateGroupAttributeSchema` を追加する（`manage_user_attribute_schema.go` と同型）。
3. **Persistence**: `infra/schema/postgres.sql` に `groups.email`/`groups.attributes`
   カラムと `tenant_group_attribute_schemas` テーブルを追加。sqlc クエリ追加後
   `just sqlc-generate`。`group/db_memory`・`group/db_postgres`・`tenancy/db_memory`・
   `tenancy/db_postgres` の adapter を実装する。
4. **HTTP**: `group/handlers_http/admin_group_handler.go` の DTO に email/attributes を
   追加。`tenancy/handlers_http/admin_group_attribute_schema_handler.go` を新設し、
   `/api/admin/v1/tenant/group_attribute_schema` (`GET`/`PUT`) を `tenancy/handlers_http/
   routes.go` に登録する。
5. **UI**: `frontend/src/features/admin-groups/` の編集画面に email 入力欄と属性エディタ
   （`admin-users` の `AdminUserAttributeEditor` パターンを踏襲）を追加する。テナントの
   Group 属性スキーマ管理ページを新設する。`frontend/src/types.ts` を更新する。

## Tasks

- [x] T001 [Spec] TypeSpec (`Group`/`GroupAttributeDef`/`GroupCreateRequest`/
  `GroupUpdateRequest`/`GroupSummaryResponse`, `TenantGroupAttributeSchema` 一式) と
  `SPECIFICATION.md` (identity-management REQ-IDMANAGEMENT-024, tenancy
  REQ-TENANCY-020, 両 Design section) を更新し、`just check-spec` /
  `just check-api-compat` を通した。
- [x] T002 [Go/Domain] `Group.Email`/`Group.Attributes`、`GroupAttributeDef`、
  `TenantGroupAttributeSchema`、`ValidateGroupAttributes` を実装した（単体テスト先行、
  `group_attributes_test.go`）。
- [x] T003 [Go/Usecase] `CreateGroup`/`UpdateGroup` に email・attributes 検証を追加し、
  tenancy に group attribute schema の use case を追加した
  (REQ-IDMANAGEMENT-024, REQ-TENANCY-020、`admin_groups_attributes_test.go`、
  `manage_group_attribute_schema_test.go`)。
- [x] T004 [Go/Persistence] Postgres スキーマ・sqlc クエリ・db_memory/db_postgres
  adapter を実装した(embedded Postgres 相手の round-trip テスト込み)。
- [x] T005 [Go/HTTP] admin group ハンドラに email/attributes を配線し、tenancy に
  group attribute schema の GET/PUT ハンドラとルートを追加した(実サーバー越しの
  HTTP 統合テスト込み)。
- [x] T006 [App] admin-groups 編集画面に email 入力欄・属性エディタを追加し、テナント
  Group 属性スキーマ管理ページ (`/admin/tenant/group_attributes`) を新設した。
- [x] T007 [Verify] `just verify` を通し、Completion を記録した。

## Verification

- `go test ./...` (in: idmagic) — Group domain/usecase/HTTP の新規単体テスト
  （email 検証、attributes スキーマ検証、schema CRUD、cross-tenant 拒否）を含む。
- `golangci-lint run ./...` (in: idmagic)
- `go build ./...` (in: idmagic)
- `bun --cwd idmagic/ui typecheck`
- `bun --cwd idmagic/ui lint`
- `bun --cwd idmagic/ui build`
- 手動 1: admin がグループ編集画面で email と custom attribute を設定 → 保存後に一覧・
  詳細に反映される。
- 手動 2: テナントの Group 属性スキーマに `cost_center` (string, required=false) を
  追加 → グループ編集画面に入力欄が現れ、未定義 key は拒否される。

## Risk Notes

グループ永続化スキーマの変更を伴う。既存テナントのデータに対する後方互換性
（新規カラムは NULL 許容にする等）を確保すること。

## Completion

- **Completed At**: 2026-08-12
- **Summary**:
  `Group` に一級フィールド `email`（形式検証のみ、検証フロー・一意制約なし）と
  `attributes`（sparse bag、値の型は `User` の `AttributeValue` を再利用）を追加した。
  カスタム属性はテナント管理者が事前定義するスキーマ駆動方式(Okta/Entra ID 型)を
  採用し、`User` の `UserAttributeDef`/`TenantUserAttributeSchema` とは別の
  `GroupAttributeDef`/`TenantGroupAttributeSchema` を新設した(Group には builtin
  catalog・self-service 編集主体・claim 露出が無いため)。`TenantGroupAttributeSchema`
  は `TenantUserAttributeSchema` と同じ配置方針で `Tenancy` コンテキストが
  port/usecase/HTTP を所有する。バックエンドは Domain → Usecase →
  Persistence(Postgres スキーマ・sqlc・db_memory/db_postgres) → HTTP
  (`/api/admin/v1/groups` の email/attributes 拡張、新規
  `/api/admin/v1/tenant/group_attribute_schema` GET/PUT)の順に実装し、各層に
  単体テスト・統合テスト(embedded Postgres、実 HTTP サーバー)を追加した。フロントは
  グループ作成・編集画面への email 入力欄とカスタム属性エディタ、テナントの
  Group 属性スキーマ管理画面 (`/admin/tenant/group_attributes`、追加/編集を
  1 画面のダイアログに統合し `User` 版の 2 画面構成より簡略化)を追加した。
- **Verification Results**:
  - `just verify` (check / check-api-compat / test-tools / typecheck-tools /
    lint-go / test-go / format-check-ui / lint-ui / test-ui-unit / build-ui)
    — result: すべて ok
  - `go test ./...` (in: idmagic) — result: ok (embedded Postgres を使う
    Group/TenantGroupAttributeSchema の round-trip テスト、実 HTTP サーバー越しの
    admin group API・tenant group attribute schema API テストを含む)
  - `golangci-lint run ./...` (in: idmagic) — result: 0 issues
  - `go build ./...` (in: idmagic) — result: ok
  - `bun run typecheck` (in: frontend) — result: ok
  - `bun run lint` (in: frontend) — result: ok (既存の無関係な警告 35 件のみ、
    新規ファイルへの警告なし)
  - `bun test src/` (in: frontend) — result: 565 pass / 0 fail
  - `bun run build` (in: frontend) — result: ok
  - 手動確認: 本セッションの環境では chromium-cli・接続済みブラウザ拡張・専用の
    アプリ起動 skill のいずれも利用できず、実ブラウザでの目視確認は実施していない
    (既知のギャップ)。代わりに実 Echo サーバー + embedded Postgres を通した
    HTTP 統合テスト (`TestAdminGroupAPICreateWithEmailAndAttributes` 等) と、
    実 DOM 操作を伴う React Testing Library テスト
    (`AdminTenantGroupAttributesPage.test.tsx` 等) で挙動を検証した。
- **Affected Guarantees State**:
  - tenant isolation: `TenantGroupAttributeSchemaRepository`/`GroupRepository` は
    既存の tenant-scoped persistence 規約に従う。cross-tenant 参照は拒否される
    (既存の Group tenant 境界テストで確認)。
  - 後方互換性: `groups.email`/`groups.attributes` は NULL 許容/デフォルト空、
    `GroupSummaryResponse.attributes` は optional のため `check-api-compat` は
    breaking change なしを確認済み。
  - SCL/spec coherence: TypeSpec (`Group`/`GroupAttributeDef`/
    `TenantGroupAttributeSchema` 一式) と `SPECIFICATION.md`
    (REQ-IDMANAGEMENT-024, REQ-TENANCY-020, 両 Design section) を同期し
    `just check-spec` で確認済み。
  - Out of Scope に記載した「ユーザー属性機構との完全共通化」は見送り、値の型
    (`AttributeValue`/`AttributeType`) のみ共有する設計を採用(理由は Design 節)。
  - SCIM 経由のグループ属性連携は引き続き対象外 (Out of Scope のまま)。
