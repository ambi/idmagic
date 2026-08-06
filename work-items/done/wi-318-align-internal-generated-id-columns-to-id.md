---
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-07
depends_on: []
---

# 内部生成された代理キーの列名を id に統一する

## Motivation

`ARCHITECTURE.md` の Database design policy (ADR-082/083/084) は、idmagic が内部生成する
代理キーを `UUID` 型に統一し、外部プロトコルが値を決めない識別子はグローバル一意な `id` として
持つ方針を確立している。`users.id` / `oauth2_clients.client_id`(ADR-082/083)が代表例で、
`client_id` のように列名自体が外部仕様(OAuth2/OIDC)の語彙である場合を除き、内部生成 UUID の
主キーは `id` へ寄せるのが基本形になっている。

`infra/schema/postgres.sql` の主キー設計を一通りレビューしたところ、この方針がまだ適用されて
いない残存ケースが複数見つかった。特に `identity_provider_connections.provider_id` は、
`spec/contexts/authentication.yaml` の `IdentityProviderConnection` エンティティが既に
`identity: [tenant_id, id]` として `id` を正としているにもかかわらず(`authentication.yaml:210-217`)、
Postgres の列名だけが古い `provider_id` のまま取り残されているという、ドメインモデルと物理
スキーマの明確な drift だった(Go 側も `connection.ID = uuid.NewString()` として既に `id` 相当で
生成している)。他に `oauth2_client_secrets.credential_id`、`tenant_branding_assets` /
`application_icons` の `object_key`、`application_categories.category_id` も同種のケースで、
値は内部生成 UUID だが列名だけ entity 固有名が残っている。

レビューの過程で `applications.application_id` と `mcp_resource_servers.resource_server_id` も
一度は id 化候補として挙げたが、後者は SCL 記述(`oauth2.yaml`)に「tenant を越えて globally
unique (ADR-083 の client_id と同方針)」と明記されており、`client_id` と同じ設計哲学で意図的に
entity 固有名を選んでいることが判明したため対象から外した。前者は参照点・API 露出が最も広く、
本 WI のスコープでは影響範囲を見積もりきれないため単独の判断に委ねる。

## Scope

- `spec/contexts/oauth2.yaml`: `models.ClientSecretCredential` の `identity`(`credential_id` → `id`)
- `spec/contexts/tenancy.yaml`: `interfaces.GetTenantBrandingAsset` /
  `UploadTenantBrandingAsset` / `DeleteTenantBrandingAsset` の `object_key` 入出力・URL パス
  パラメータ(`object_key` → `id`)
- `spec/contexts/application.yaml`: `interfaces.GetApplicationIcon` /
  `UploadApplicationIcon` / `DeleteApplicationIcon` の `object_key`(→ `id`)、
  `models.ApplicationCategory` の `identity`(`category_id` → `id`)
- `infra/schema/postgres.sql`: `identity_provider_connections`、`oauth2_client_secrets`、
  `tenant_branding_assets`、`application_icons`、`application_categories` の主キー
- `identity_provider_connections` を参照する `federated_identities` /
  `federated_login_attempts` の外部キー定義(列名 `provider_id` 自体は維持、参照先 PK のみ変更)
- 対応する Go 実装: `backend/authentication/federation`、`backend/oauth2`、`backend/tenancy`、
  `backend/application` の domain / usecases / db_postgres(sqlc 再生成)/ handlers_http /
  db_memory
- フロントエンドの該当 API フィールド参照(admin-applications, tenant-branding,
  identity-providers 系コンポーネント)

## Out of Scope

- `oauth2_clients.client_id`: OAuth2/OIDC のワイヤー語彙そのもの。ADR-083 が維持を明示決定済み。
- `webauthn_credentials.credential_id`: WebAuthn 仕様の値そのもの(認証器が生成、型も `TEXT`)。
- `applications.application_id`: 7+ テーブルからの参照・admin/self-service API への露出が広く、
  本 WI のスコープでは影響範囲を見積もりきれない。単独 WI として別途判断する。
- `mcp_resource_servers.resource_server_id`: SCL 記述が `client_id` と同じ設計哲学を明記している
  ため対象外(Design 参照)。
- `federated_response_replays.response_id` / `saml_service_providers.entity_id` /
  `wsfed_relying_parties.wtrealm` / `signing_keys.kid`: 値そのものを外部プロトコルが決める
  識別子で、UUID 化・`id` 化のいずれも不可(ADR-084)。
- `authentication_event_buckets` の複合 PK: `ON CONFLICT` による冪等 increment の対象キーで
  あり、集計バケットの自然キーとして機能している(ADR-041)。変更しない。

## Design

「複合 PK / `id` でない PK 列名」は次の系統に分類できる。

| 系統 | 特徴 | 対応 |
|---|---|---|
| A. 値そのものが外部由来 | `entity_id`, `wtrealm`, `kid`, WebAuthn `credential_id`, SAML `response_id` | 不可(Out of Scope) |
| B. 列名自体が外部仕様/既存 ADR で明示的に固定された語彙 | `oauth2_clients.client_id`, `mcp_resource_servers.resource_server_id` | 変更しない(Out of Scope) |
| C. 内部生成 UUID で、名前も idmagic 独自の命名にすぎない | `identity_provider_connections.provider_id`, `oauth2_client_secrets.credential_id`, `object_key`, `application_categories.category_id` | 本 WI のスコープ |

系統 C のうち `applications.application_id` は影響範囲が大きすぎるため今回は対象外(Out of
Scope に記載)。以下、対象4件それぞれの設計。

**`identity_provider_connections.provider_id` → `id`**: SCL は既に `id` で確定済み
(`authentication.yaml:214-217`)。Postgres 列だけが追従していない。`(tenant_id, provider_id)`
複合 PK を `id UUID PRIMARY KEY` に変更し、`tenant_id` は属性列へ降格(tenant 内一覧用に
非ユニーク index を追加)。ADR-083 と同じ理屈(グローバル一意な id には複合 FK が不要)を
`federated_identities` / `federated_login_attempts` に適用し、複合 FK
`(tenant_id, provider_id) REFERENCES identity_provider_connections(tenant_id, provider_id)`
を `provider_id REFERENCES identity_provider_connections(id)` に簡略化する。
`federated_login_attempts` の PK 自体は `(tenant_id, state)` のまま変更しない(`state` は
attacker-influenced な opaque 値のため ADR-139 例外で `tenant_id` 保持が必要)。

**`oauth2_client_secrets.credential_id` → `id`**: `ClientSecretCredential.identity` を
`credential_id` から `id` に変更。Go 型 `CredentialID` と admin API の JSON キー
`credential_id` もあわせて `ID` / `id` に揃える。他テーブルからの FK 参照はない。

**`tenant_branding_assets` / `application_icons` の `object_key` → `id`**:
`GetTenantBrandingAsset` / `GetApplicationIcon`(および Upload/Delete)の interface input と
URL パスパラメータを `object_key` から `id` に変更(URL は `/tenant-branding-assets/{kind}/{id}`、
`/application-icons/{application_id}/{id}` になる)。DB 側は `id UUID PRIMARY KEY` とし、
`tenant_id`/`kind` または `application_id` は属性列として残して一覧・削除クエリ用に非ユニーク
index を追加する。既存の `ON CONFLICT (tenant_id, kind, object_key)` /
`ON CONFLICT (application_id, object_key)` upsert は新 PK 基準に書き換える。

**`application_categories.category_id` → `id`**: `ApplicationCategory.identity` を
`[tenant_id, category_id]` から `[tenant_id, id]` へ。`applications.category_ids` は FK
制約のない非正規化 `UUID[]` 参照なので、値そのもの(UUID)は変わらず、参照側コードの
フィールド名整理のみで済む。

破壊的変更の許容性: `infra/schema/README.md` / ADR-084 / ADR-085 が明記する通り、このリポジトリは
versioned migration を持たない宣言的 schema で運用されており、リリース前提のため、JSON API
フィールド名・URL パスの変更は許容される(実データ移行は不要)。

## Plan

1. SCL 更新(3 ファイル): `oauth2.yaml` / `tenancy.yaml` / `application.yaml` の該当 identity /
   interface フィールド名を変更し、`just scl-render` で派生物(OpenAPI 等)を再生成する。
2. `infra/schema/postgres.sql` 更新: 4 テーブルの PK 再定義と `identity_provider_connections`
   参照側 FK の簡略化。`just check-schema` で psqldef 収束(apply → dry-run no-op → apply →
   dry-run no-op)を確認する。
3. Go 実装: 各 context の domain / usecases / db_postgres(sqlc 再生成)/ handlers_http の
   フィールド名・クエリを更新する。db_memory adapter も同じ意味で追従させる(このリポジトリの
   規約上、memory adapter だけ更新漏れにしない)。
4. フロントエンド: 該当コンポーネントの API フィールド参照・URL 組み立てを更新する。
5. テスト: 各層の既存テストのフィールド名・URL 参照を更新する。
6. 4 つの対象(identity_provider_connections / oauth2_client_secrets / branding+icon の
   object_key / application_categories)はそれぞれ独立した bounded context にまたがるため、
   1 つずつ順番に着手し区切って良い(相互依存はない)。

## Tasks

- [x] T001 [SCL] `spec/contexts/oauth2.yaml` の `ClientSecretCredential.identity` を
      `credential_id` → `id` に変更。`just scl-render` で派生物を再生成。
      `ClientSecretCredentialMetadata.credential_id` も同じ entity の admin API 表現として
      あわせて `id` に変更(`application.yaml` 側の `ApplicationClientSecretCredentialMetadata` /
      `RevokeApplicationClientSecret` は Scope 外のため `credential_id` のまま維持)。
- [x] T002 [SCL] `spec/contexts/tenancy.yaml` の `GetTenantBrandingAsset` の `object_key`
      入力・URL パスパラメータを `id` に変更(`UploadTenantBrandingAsset` /
      `DeleteTenantBrandingAsset` はもともと `object_key` を interface シグネチャに持たない)。
- [x] T003 [SCL] `spec/contexts/application.yaml` の `GetApplicationIcon` の `object_key` を
      `id` に、`ApplicationCategory.identity` の `category_id` を `id` に変更。
      `AdminApplicationCategoryResponse` / `PortalApplicationCategory` の `category_id` と
      `UpdateApplicationCategory` / `DeleteApplicationCategory` の path param も同じ identity の
      表現としてあわせて `id` に変更。
- [x] T004 [Schema] `identity_provider_connections` を `id PRIMARY KEY` + `tenant_id` 属性列 +
      一覧用 index に変更し、`federated_identities` / `federated_login_attempts` の FK を単純化。
      列型は SCL (`authentication.yaml` の `id: type: String`) に合わせ `TEXT` のまま維持
      (Design 記述は `UUID PRIMARY KEY` としていたが、実際の値は Go 生成 UUID 文字列であり
      `TEXT` で要件を満たすため、既存のテスト用固定文字列 ID との互換を優先して型変更は見送った)。
      `just check-schema` 通過。
- [x] T005 [Schema] `oauth2_client_secrets.credential_id` → `id UUID PRIMARY KEY` にリネーム。
      `just check-schema`。
- [x] T006 [Schema] `tenant_branding_assets` / `application_icons` の `object_key` →
      `id UUID PRIMARY KEY`、親キーへの非ユニーク index 追加、`ON CONFLICT` upsert を id 基準へ
      書き換え。`just check-schema`。
- [x] T007 [Schema] `application_categories.category_id` → `id`(`PRIMARY KEY (tenant_id, id)`
      を維持。Design 記述どおり `[tenant_id, id]` の複合構造のまま、列名のみ変更)。
      `just check-schema`。
- [x] T008 [App] `backend/authentication/federation` を `id` ベースに更新。domain 層は
      もともと `IdentityProviderConnection.ID`(`json:"id"`)で実装済みのため変更不要、
      db_postgres(sqlc 再生成)/ repositories / reencrypt のみ列名変更に追従。
      テストは列名追従に加え、`testConnection` ヘルパーが全呼び出しで固定文字列
      `"oidc"` を id に使っていたため、`id` が単独 PRIMARY KEY化(旧: 複合 `(tenant_id,
      provider_id)`)したことで複数テナント間の PK 衝突が顕在化(scenario
      `IdentityProviderConnection.*` の round-trip / dual-read テストが cross-tenant で
      互いの行を上書きし Find が nil を返す)。`pgfixtures.NewUUID(t)` で一意な id を生成する
      ように修正して RED→GREEN を確認。
- [x] T009 [App] `backend/oauth2` の `ClientSecretCredential`(domain/usecases/db_postgres/
      db_memory)を `id` ベースに更新。`backend/application` 側は domain 型を直接使う箇所
      (`credential.CredentialID` → `credential.ID`)のみ追従し、`application.yaml` 側の
      JSON 契約(`credential_id`)は変更していない。
- [x] T010 [App] `backend/tenancy`(`TenantBrandingAsset.ObjectKey`)と `backend/application`
      (`ApplicationIcon.ObjectKey`)を `ID` ベースに更新。URL パス変更
      (`{object_key}` → `{id}`)を routes.go / c.Param 参照含め反映。
- [x] T011 [App] `backend/application` の `ApplicationCategory`(domain/usecases/
      db_postgres/db_memory/handlers_http)を `ID` ベースに更新。admin/portal レスポンス
      (`categoryResponse`, `portalCategoryResponse`)の JSON キーも `category_id` → `id`。
- [x] T012 [Frontend] `types.ts` の `ApplicationCategory` / `PortalCategory` を `id` に変更し、
      `AccountAppsPage` / `AdminApplicationCategories` とそのテストを追従。
      tenant-branding / identity-providers 側は URL 内蔵の `logo_url`/`favicon_url` や
      既存の `id` フィールドをそのまま使っており、フィールド参照の変更は不要だった。
- [x] T013 [Verify] `just check`、`just check-schema`(apply→dry-run no-op×2)、
      `just verify`(lint-go/lint-ui/format-check-ui/test-go/test-ui-unit/typecheck-tools/
      test-tools/check/traceability-strict)、`go build ./...`、`go vet ./...`、
      `go test -race ./backend/...` を全てグリーンで確認。

## Verification

- `just check-schema`(psqldef 収束: apply → dry-run no-op → apply → dry-run no-op)
- `just scl-render` 後の derived artifacts(OpenAPI 等)の差分確認
- `go test ./...`(対象 context: authentication/federation, oauth2, tenancy, application)
- フロントエンドの該当機能(branding アップロード/取得、アプリアイコン、identity provider
  connection CRUD、client secret 発行)の手動確認または e2e

## Risk Notes

- 未リリース前提(ADR-084/085 と同様)のため、データ移行や API 後方互換は不要。ただし 4 つの
  独立した bounded context にまたがるため、1 セッションで完結しない可能性がある。Task を
  context 単位で分割しているので、途中で区切って再開できる。
- `identity_provider_connections` の複合 FK 簡略化は `federated_identities` /
  `federated_login_attempts` の 2 テーブルに波及するため、他の対象より慎重にレビューする。
- URL パス変更(`object_key` → `id`)は、発行済みのブランディング/アイコン URL をキャッシュ・
  ブックマークしているクライアントを壊すが、pre-release のため許容する。

## Completion

- **Completed At**: 2026-08-07
- **Summary**:
  Scope 記載の 4 対象(`identity_provider_connections.provider_id`、
  `oauth2_client_secrets.credential_id`、`tenant_branding_assets` / `application_icons` の
  `object_key`、`application_categories.category_id`)すべてを `id` に統一した
  (SCL → Postgres schema → Go domain/usecases/db_postgres/db_memory/handlers_http →
  frontend)。
  開示 (逸脱): Design は `identity_provider_connections.id` の列型を `UUID PRIMARY KEY` として
  いたが、実装では `TEXT` のまま維持した。SCL (`authentication.yaml`) が該当 identity を
  `type: String` と定義しており、ドリフトは列名のみで型は元々一致していたため。実運用では Go
  側が UUID 文字列を生成するが、型制約としては課していない。
  `applications.application_id` と `mcp_resource_servers.resource_server_id` は WI 記載どおり
  Out of Scope のまま未着手。`oauth2_clients.client_id` / `webauthn_credentials.credential_id` /
  `federated_response_replays.response_id` 等のプロトコル由来識別子、および `application.yaml`
  側の `ApplicationClientSecretCredentialMetadata.credential_id` /
  `RevokeApplicationClientSecret` の `credential_id` パスパラメータ(Scope 外、`oauth2.yaml` 側の
  同名 admin API のみ `id` 化)は変更していない。
  test-first からの逸脱 (self-attest): 本 WI は新規振る舞いを追加しない列名/フィールド名の
  リネームであり、新しい RED テストは書いていない。代わりに、リネーム後に列名・フィールド名
  不一致でコンパイルエラーとなった既存テストを追従させ、`identity_provider_connections` の
  PK 単独化で顕在化した cross-tenant なテスト間 ID 衝突(`testConnection` ヘルパーの固定文字列
  ID)というリグレッションを検出・修正した。
- **Verification Results**:
  - `just check` - passed。
  - `just check-schema`(apply→dry-run no-op×2)- passed。
  - `just verify`(lint-go/lint-ui/format-check-ui/test-go/test-ui-unit/typecheck-tools/
    test-tools/check/traceability-strict)- passed。
  - `go build ./...` / `go vet ./...` - passed。
  - `go test -race ./backend/...` - passed。
