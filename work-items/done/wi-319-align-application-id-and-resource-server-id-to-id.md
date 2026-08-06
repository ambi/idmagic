---
status: completed
authors: [tn]
risk: high
created_at: 2026-08-07
depends_on: []
---

# `applications.application_id` と `mcp_resource_servers.resource_server_id` を `id` に統一する

## Motivation

[[wi-318-align-internal-generated-id-columns-to-id]] は、idmagic が内部生成する代理キーを
グローバル一意な `id` へ寄せる方針(ADR-082/083)に沿って、命名がドリフトしていた4対象を
`id` へ統一した。その際、`applications.application_id` と `mcp_resource_servers.resource_server_id`
は「`oauth2_clients.client_id` と同じ設計哲学で意図的に entity 固有名を選んでいる」という理由で
まとめて Out of Scope とされたが、この判断は再検証すると次の点で不正確だった。

- `client_id` がリネーム対象外である本当の理由は、OAuth2/OIDC の RFC 6749 が規定する
  ワイヤーパラメータ名そのもの(トークンリクエスト等に literal に登場する)だからであり、
  ADR-083 由来ではない。ADR-083 は「`client_id` をグローバル一意なシステム生成値にし、
  複合 PK をやめて単一カラム PK にする」という**一意性ポリシー**の決定であって、
  「`client_id` という**列名**を維持すべき」とは述べていない。
- `mcp_resource_servers.resource_server_id` の SCL コメント(`spec/contexts/oauth2.yaml:1410`)
  「tenant を越えて globally unique (ADR-083 の client_id と同方針)」は、ADR-083 と**一意性ポリシー**が
  同じだと言っているだけで、列名を `client_id` のように維持すべき理由にはならない。
  RFC 8707 (Resource Indicators) や RFC 9728 (Protected Resource Metadata) のいずれも
  `resource_server_id` という語を規定しておらず、外部仕様のワイヤー語彙ではない。
- `applications.application_id` も同様に外部仕様の語彙ではなく、`spec/contexts/application.yaml:92`
  の説明どおり「PostgreSQL の global UUID primary key」という、idmagic 独自の内部識別子に過ぎない。

したがって両者は wi-318 の分類でいう「系統B: 列名自体が外部仕様/既存ADRで明示的に固定された語彙」
ではなく、実質的に「系統C: 内部生成 UUID で、名前も idmagic 独自の命名にすぎない」に該当する。
wi-318 が `applications.application_id` を対象外にした本当の理由は影響範囲の大きさであり、
`mcp_resource_servers.resource_server_id` はその判断に(検証不足のまま)引きずられて一緒に
対象外とされた。

## Scope

- `spec/contexts/oauth2.yaml`: `McpResourceServer.identity`(`resource_server_id` → `id`)、
  `McpResourceServerResponse` の `resource_server_id`、`ListAdminMcpResourceServers` /
  `GetAdminMcpResourceServer` / `CreateAdminMcpResourceServer` / `UpdateAdminMcpResourceServer` /
  `DeleteAdminMcpResourceServer` の path param(`{resource_server_id}` → `{id}`)。
- `spec/contexts/application.yaml`: `Application.identity`(`application_id` → `id`)と、
  それを運ぶ value object・interface 全般(`AdminApplicationResponse` / `MyApplication` /
  `AdminApplicationCreateRequest` の応答、`GetAdminApplication` / `UpdateAdminApplication` /
  `DeleteAdminApplication` などの admin interface の path param、`GetApplicationIcon` /
  `UploadApplicationIcon` などの `{application_id}` セグメント、`ApplicationCategoriesRequest`
  の付与先指定など)。
- `spec/contexts/oauth2.yaml` / `saml.yaml` / `ws-federation.yaml` / `provisioning.yaml` /
  `identity-governance.yaml`: `Application` を参照する側(`OAuth2Client.application_id`、
  `SamlServiceProvider.application_id`、`WsFedRelyingParty.application_id`、
  `ProvisioningConnection.identity`(`application_id`)等)の **列名自体は維持**し、
  参照先の型/意味だけが `Application.id` に追従する(wi-318 の
  `federated_identities.provider_id` と同じ扱い)。`oauth2.yaml:3340-3346` の
  `Application` publish stub の `identity` も `id` に追従させる。
- `infra/schema/postgres.sql`: `applications` と `mcp_resource_servers` の主キー、および
  `applications.application_id` を参照する全 FK
  (`application_icons` / `application_sign_in_policies` / `application_assignments` /
  `oauth2_clients` / `saml_service_providers` / `wsfed_relying_parties` /
  `provisioning_connections`(自身の PK が `application_id` を再利用) /
  `scim_group_pushes` 等の `provisioning_connections(application_id)` 参照)。
- 対応する Go 実装: `backend/application`、`backend/oauth2`(client 側 `application_id` FK と
  MCP resource server 管理)、`backend/saml`、`backend/wsfederation`、`backend/provisioning`、
  `backend/idgovernance` の domain / usecases / db_postgres(sqlc 再生成)/ handlers_http /
  db_memory。
- フロントエンドの該当 API フィールド参照・URL 組み立て(admin-applications 系、
  admin MCP resource server 管理画面等)。

## Out of Scope

- `oauth2_clients.client_id`: OAuth2/OIDC のワイヤー語彙そのもの(RFC 6749)。ADR-083 が
  グローバル一意化のみを決定し、列名の維持自体は別理由(外部仕様)による。維持する。
- `webauthn_credentials.credential_id` / `federated_response_replays.response_id` /
  `saml_service_providers.entity_id` / `wsfed_relying_parties.wtrealm` / `signing_keys.kid`:
  値そのものを外部プロトコルが決める識別子(wi-318 と同じ理由)。変更しない。
- `Application` を参照する側の列名(`oauth2_clients.application_id` 等、上記 Scope 参照)は
  維持し、参照先 PK のみを `id` に変更する。

## Design

**`mcp_resource_servers.resource_server_id` → `id`**: 他テーブルからの FK 参照がなく、
`(tenant_id, resource)` の UNIQUE 制約はあるが `tenant_id` を PK に含めていないため、
スキーマ変更は列名リネームのみ(`id UUID PRIMARY KEY` のまま構造は変わらない)。
wi-318 の `oauth2_client_secrets.credential_id` と同程度の小さな変更。

**`applications.application_id` → `id`**: `applications` は既に `application_id UUID PRIMARY KEY`
(tenant_id は属性列)であり ADR-083 の一意性ポリシーは満たしているため、スキーマ構造自体の
変更は不要で列名リネームのみ。ただし参照側が広い:

- 直接 FK: `application_icons`、`application_sign_in_policies`、`application_assignments`
  (以上は `application_id` カラム名のまま `applications(id)` を参照するよう FK 定義だけ更新)。
- 複合 FK(`(application_id, tenant_id, protocol_type)` → `applications(application_id,
  tenant_id, protocol_type)`): `oauth2_clients` / `saml_service_providers` /
  `wsfed_relying_parties`。参照先カラムを `id` に更新するのみで、複合 FK の形自体
  (`tenant_id` / `protocol_type` を含む)は業務上の理由(protocol 種別ごとの整合性検査)が
  あるため wi-318 のような単純化(複合→単純)はここでは行わない。
- `provisioning_connections.application_id` は同テーブル自身の PK として `applications.id` を
  再利用する設計(1 Application 1 connection を構造的に強制)であり、列名 `application_id` は
  維持したまま参照先だけ `id` に更新する。

この2件は互いに独立(`mcp_resource_servers` は `applications` を参照しない)なので、
`resource_server_id` を先に着手して小さく完了させ、`application_id` は別セッションで
腰を据えて進める、という分割も可能。

## Plan

1. `resource_server_id` から着手する(影響範囲が小さく、SCL → schema → Go(`backend/oauth2` の
   MCP resource server 管理)→ frontend まで 1 セッションで完結できる見込み)。
2. `application_id` は対象 bounded context(application / oauth2 / saml / wsfederation /
   provisioning / idgovernance)ごとに区切って進める。各 context の SCL 更新 → schema FK 更新
   → Go 実装 → frontend の順で、`just check-schema` を都度確認する。
3. `applications` テーブルへの FK が多数のテーブルに及ぶため、`infra/schema/postgres.sql` の
   一括変更は 1 コミットにまとめず、`just check-schema` の apply→dry-run no-op を各段階で
   確認しながら進める。

## Tasks

- [x] T001 [SCL] `spec/contexts/oauth2.yaml` の `McpResourceServer.identity` を
      `resource_server_id` → `id` に変更し、応答 value object と admin CRUD interface の
      path param も追従させる。`just scl-render`。
- [x] T002 [Schema] `mcp_resource_servers.resource_server_id` → `id`。`just check-schema`。
- [x] T003 [App] `backend/oauth2` の MCP resource server 管理(domain/usecases/db_postgres/
      handlers_http/db_memory)を `id` ベースに更新。
- [x] T004 [Frontend] MCP resource server 管理画面の API フィールド参照・URL 組み立てを更新
      (`types.ts` / `AdminMcpResourceServers*` / `admin.test.ts`)。
- [x] T005 [SCL] `spec/contexts/application.yaml` の `Application.identity` を
      `application_id` → `id` に変更し、`AdminApplicationResponse` / `MyApplication` /
      `PortalApplicationCategory` 等の value object と admin/account interface(約20本)の
      path param・入出力フィールドを追従させた。`ApplicationAssignment.application_id` /
      `AppSignInPolicy.identity`(FK 型 identity)は列名維持で対象外。`GetApplicationIcon` の
      外側 `{application_id}` セグメントは、内側の `{id}`(wi-318 由来)との衝突を避けるため
      維持(唯一の技術的例外)。
- [x] T006 [SCL] `Application` を参照する他 context の参照先型を `id` に追従させ、列名は維持:
      `oauth2.yaml`(`OAuth2Client.application_id`、`Application` publish stub の identity)、
      `saml.yaml`(`SamlServiceProvider.application_id`)、`ws-federation.yaml`
      (`WsFedRelyingParty.application_id`)。`provisioning.yaml` は当初想定より広く、
      `ProvisioningConnection.identity`(FK 型、維持)に加えて `RegisterProvisioningConnection`
      等 admin interface 11 本の `application_id` 入力/path param も `Application` 自身の
      identity として `id` に統一した(Design 未記載の追加発見、T009 参照)。
      `identity-governance.yaml` の `WorkflowActionDef.application_id` は他 entity への
      FK 型フィールドであり維持。
- [x] T007 [Schema] `applications.application_id` → `id`。直接 FK
      (`application_icons`/`application_sign_in_policies`/`application_assignments`)と
      複合 FK(`oauth2_clients`/`saml_service_providers`/`wsfed_relying_parties`、
      `applications_protocol_identity_unique`)、`provisioning_connections` の参照先を
      `applications(id, ...)` に更新。`just check-schema` 通過。
- [x] T008 [App] `backend/application` の domain(`Application.ApplicationID`→`ID`)/
      usecases / db_postgres(sqlc 再生成 + 手書き `ListApplicationAssignmentsBySubjects` の
      JOIN 条件)/ handlers_http(`applicationResponse`/`myApplicationResponse` の JSON キー
      `application_id`→`id`)/ db_memory を更新。
- [x] T009 [App] `backend/oauth2`(client の `application_id` FK 部分)/ `backend/saml` /
      `backend/wsfederation` / `backend/idgovernance` の `application_id` 参照(列名維持、
      `app.ApplicationID`→`app.ID` の読み替えのみ)を更新。`backend/provisioning` は
      T006 で判明した admin interface 群のルート(`routes.go` の `:application_id`→`:id`)と
      ハンドラの `c.Param` キーもあわせて更新。
- [x] T010 [Frontend] `types.ts`(`AdminApplication`/`MyApplication`)と admin-applications /
      account / admin-lifecycle-workflows / admin-provisioning / admin-sign-in-policy 系
      コンポーネントの `.application_id` 読み出しを `.id` に更新。`WorkflowAction` /
      `ProvisioningConnection` の同名(維持対象)フィールドとの混同を個別に確認して除外した。
- [x] T011 [Verify] `just check` / `just check-schema` / `just verify` / `go build ./...` /
      `go vet ./...` / `go test -race ./backend/...` すべて green。

## Verification

- `just check-schema`(psqldef 収束: apply → dry-run no-op → apply → dry-run no-op)
- `just scl-render` 後の derived artifacts(OpenAPI 等)の差分確認
- `go test ./...`(対象 context: application, oauth2, saml, wsfederation, provisioning,
  idgovernance)
- フロントエンドの該当機能(アプリ管理 CRUD、アイコン、割当、MCP resource server 管理)の
  手動確認または e2e

## Risk Notes

- `application_id` は 6 bounded context・SCL 上約130箇所・Go 実装約100ファイル・frontend
  約20ファイルに及ぶ idmagic 最大級の識別子であり、`resource_server_id`(単一 context・
  SCL上9箇所・Go実装約44ファイル)とは影響範囲が大きく異なる。両者を同一 WI にまとめたのは
  motivation 上の経緯(wi-318 での誤った一括除外の是正)によるが、実装セッションは
  Tasks の区切りに従い分割してよい。
- 未リリース前提(ADR-084/085 と同様)のため、データ移行や API 後方互換は不要。
- `applications` への複合 FK(`oauth2_clients` / `saml_service_providers` /
  `wsfed_relying_parties`)は `protocol_type` の整合性検査を兼ねており、wi-318 の
  `identity_provider_connections` のように複合 FK を単純化する変更ではない
  (`tenant_id`, `protocol_type` は維持し、`application_id` 部分だけを `id` に置き換える)。
  この点を FK 再定義時に見落とさないこと。

## Completion

- **Completed At**: 2026-08-07
- **Summary**:
  Motivation で指摘した誤った一括除外を是正し、`mcp_resource_servers.resource_server_id`
  (小規模・単一 context)と `applications.application_id`(6 bounded context に波及)の
  両方を `id` に統一した。SCL → Postgres schema → Go(domain/usecases/db_postgres/
  db_memory/handlers_http)→ frontend の順で一貫させた。
  `Application` を参照する他 entity の FK フィールド(`ApplicationAssignment.application_id`、
  `AppSignInPolicy`/`ProvisioningConnection` の FK 型 identity、`OAuth2Client` /
  `SamlServiceProvider` / `WsFedRelyingParty` / `WorkflowActionDef` の `application_id`)は
  wi-318 の `federated_identities.provider_id` と同じ理屈で列名を維持し、参照先の型だけを
  `id` に追従させた。
  作業中に Design 段階では想定していなかった追加スコープが判明した: `provisioning.yaml` の
  admin interface 群(`RegisterProvisioningConnection` 等 11 本)も `application_id` を
  `Application` 自身の identity として path param に持っており、`ProvisioningConnection` の
  FK 型 identity(維持)とは別に `id` へ統一する対象だった。同様に `GetApplicationIcon` は
  wi-318 で既に内側の `object_key` を `id` 化していたため、外側の `application_id` を
  `id` にすると同一パスに `{id}/{id}` の重複が生じる。これは唯一 `id` 化を見送った箇所で、
  `application_id` のまま維持した。
  test-first からの逸脱 (self-attest): 本 WI も wi-318 と同様、新規振る舞いを追加しない
  列名/フィールド名のリネームであり、新しい RED テストは書いていない。リネーム後にコンパイル
  エラー・JSON キー不一致となった既存テストを追従させ、`just verify` の green を確認した。
- **Verification Results**:
  - `just check` - passed。
  - `just check-schema`(apply→dry-run no-op×2、2 回実施: resource_server_id 分/
    application_id 分)- passed。
  - `just verify`(lint-go/lint-ui/format-check-ui/test-go/test-ui-unit/typecheck-tools/
    test-tools/check/traceability-strict)- passed。
  - `go build ./...` / `go vet ./...` - passed。
  - `go test -race ./backend/...` - passed。
  - `just typecheck-ui` - passed。
