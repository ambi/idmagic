# idp-ADR-084: Postgres 列型選定ポリシーと内部 id の UUID 型化

## ステータス

採用。`deploy/schema/postgres.sql` を横断する列型選定基準を定め、あわせて idmagic が
内部生成する id 列を `UUID` 型へ整列させる。永続化層の内部方針であり SCL の公開
contract・非機能保証を変更しないため `spec/scl.yaml` の変更は伴わない。
[idp-ADR-071](file:///Users/tn/src/idmagic/decisions/idp-ADR-071-declarative-postgres-schema-with-sqldef.md)
（sqldef による宣言的 schema）を前提とし、
[idp-ADR-082](file:///Users/tn/src/idmagic/decisions/idp-ADR-082-user-domain-id-and-tenant-key-policy.md)
／[idp-ADR-083](file:///Users/tn/src/idmagic/decisions/idp-ADR-083-globally-unique-client-id.md)
の ID・tenant key 方針と整合する。具体的な最大文字数は
`wi-128-string-length-limits-policy`、`tenants.id` の UUID + realm 分割は別 WI に委ねる。
wi-127 で導入。

## コンテキスト

`postgres.sql` は複数の bounded context の永続化を単一ファイルで横断しているが、列型の
選定基準が明文化されておらず、`TEXT` / `JSONB` / `TIMESTAMPTZ` / `UUID` / 状態値表現の
使い分けが実装時の局所判断に寄り、新規テーブル追加時に判断を再現できなかった。

特に id 列は、内部生成 surrogate（`refresh_tokens.id`、`applications.application_id`、
`application_categories.category_id`）が `UUID` 型である一方、同じく idmagic が
`spec.NewUUIDv4()` で内部生成する `users.id` / `groups.id` / `agents.id` /
`clients.client_id` / `audit_events.id` / `scim_tokens.id` は `TEXT` 型で、UUID 値を
`TEXT` 列に格納する不整合があった。値域は idmagic 自身が定義できるため、非 UUID の
seed（`user_alice` 等）を UUID に是正すれば、これらは `UUID` 型に閉じられる。未リリースの
今なら FK 参照列もろとも宣言的 schema を書き換えるだけで済み、後からの変更より安い。

一方で、外部が値を決める id は UUID 化できない。SAML `entity_id` / WS-Fed `wtrealm` は
外部 SP/RP のメタデータが定める URI、`signing_keys.kid` は `jwkThumbprint()`（RFC 7638
JWK thumbprint）由来の指紋、`scim_id` は SCIM リソース id（tenant スコープの protocol
表現）であり、いずれも idmagic が採番しない。`tenants.id` は URL（`/realms/{id}/`）に
現れる mutable な slug で、UUID 化には別カラムへの分離が要る（別 WI）。

## 決定

### 1. 文字列型: 制約なし `varchar` を使わない

上限を持たない自由文字列は `TEXT`。仕様・UI・運用上の上限がある値は `TEXT` +
`CHECK (char_length(col) <= N)` または `varchar(N)` に統一する。使い分けと具体的な `N` は
`wi-128-string-length-limits-policy` で決める。固定書式の識別子は `CHECK (... ~ 正規表現)`
で守る（`tenants.id` の既存方式を新規テーブルにも適用）。

### 2. `JSONB` の許容基準

次を許容する: 外部仕様由来の可変メタデータ／集合（`redirect_uris` / `grant_types` /
`jwks` / `acs_urls` / `reply_urls` / `entra_profile` / `public_jwk` / `private_jwk`）、
claim / policy の半構造化設定（`*_sign_in_policies.rules` / `claim_policy` /
`authorization_detail_types.schema` / `bindings` / `attributes` / `scopes` / `roles`）、
append-only payload（`audit_events.payload` / `outbox.payload` /
`refresh_tokens.sender_constraint`）。

一方、頻繁に join / filter する、FK / uniqueness / lifecycle 制約を要する、状態機械の遷移
対象になる値は `JSONB` 内に置かない。現状で該当するのは `users.lifecycle`（下記 §5 と
棚卸し表を参照。正規化候補）。

### 3. `TIMESTAMPTZ` の精度と表現

保存型は一貫して `TIMESTAMPTZ`（マイクロ秒精度を真値とし schema で丸めない）。Go adapter は
pgx v5 で `time.Time` に精度保持で scan。内部 JSON contract は `encoding/json` 既定
（RFC3339Nano）で精度を落とさず、domain event も `RFC3339Nano`。秒精度への丸めは外部
プロトコル契約の書式都合でのみ意図的に行う（SCIM `meta.*`、SAML / WS-Fed `IssueInstant`
/ `validUntil`、audit クエリ param。いずれも `time.RFC3339`）。UI（JS `Date`）はミリ秒精度。

### 4. `UUID` と domain string id の境界

- **idmagic が内部生成し、外部プロトコル語彙に値域を縛られない id は `UUID` 型**にする。
  対象: `users.id`、`clients.client_id`、`groups.id`、`agents.id`、`audit_events.id`、
  `scim_tokens.id`、および既に UUID 型の `refresh_tokens.{id,family_id,parent_id}` /
  `applications.application_id` / `application_categories.category_id`。値域は idmagic が
  `spec.NewUUIDv4()` で定義するため UUID に閉じる。seed / bootstrap も UUID を用いる
  （非 UUID の便宜 id は固定 UUID `00000000-0000-4000-8000-...` に是正する）。
  `clients.client_id` は OIDC の opaque string でもあり値として UUID は許容されるため、
  first-party クライアントも固定 UUID を割り当て、application bindings・SPA の OIDC 設定・
  discovery をこれに追随させる。
- これらを参照する FK / 参照列も同時に `UUID` 型にする（`user_id` 系、`owner_user_id`、
  `group_id`、`agent_id`、`client_id` 系、`application_assignments.subject_id`〔user/group
  いずれも UUID になるため〕、`audit_events.user_id`）。PK と参照列の型は必ず一致させる。
- **外部が値を決める id は `TEXT` を維持する**。`entity_id`（外部 SP の EntityID URI）、
  `wtrealm`（外部 RP の realm URI）、`scim_id`（SCIM リソース id、tenant スコープ）、
  `kid`（JWK thumbprint）。`tenants.id` は URL に現れる mutable slug のため当面 `TEXT`
  （UUID + realm 分割は別 WI）。
- **Go 側の id 型は `string` のまま**とする。`UUID` 列と Go `string` の相互変換は、pgx の
  接続時（`AfterConnect`）に uuid OID へ text codec を登録して一括で扱う。既存の
  `refresh_tokens` 実装（SELECT の `::text` cast と string param）とも互換。
- 新規テーブルの id は、外部に露出しない内部 surrogate なら `UUID`、外部が値を決めるなら
  `TEXT` を既定とし、既存 ADR との互換を確認して個別判断する。

### 5. 有限集合値（status / state / kind / visibility / subject_type）の表現

既定は `TEXT` + `CHECK (col IN (...))` で、値集合を DB 制約とアプリ側 enum の両方で持ち、
一致をテストする。PostgreSQL enum は削除・rename・並行 deploy の migration friction が
あるため原則採用しない（採るなら変更頻度が極めて低い値のみ）。lookup table は値に付随属性・
参照整合性が必要な場合に検討する。CHECK を持たない有限集合 `TEXT` 列（`agents.kind`、
`applications.kind` / `status`、`clients.fapi_profile` 等）は「制約追加候補」として棚卸し表に
記録し、DB 制約と Go validation の値集合一致テストを伴う形で列ごとに個別に切り出す。外部
プロトコルが語彙を決める `TEXT` 列（`token_endpoint_auth_method`、
`id_token_signed_response_alg`、`signing_keys.alg`）は DB CHECK を課さず現状維持とする。

## 棚卸し（型カテゴリ別の分類）

| カテゴリ | 代表列 | 処置 | 根拠 |
| --- | --- | --- | --- |
| 自由文字列（上限なし） | `*_hash`、`*_certificate_pem`、`secret`、PEM/JWK 由来 | 現状維持（`TEXT`） | §1 |
| 上限のある文字列 | `name` / `display_name` / `description` / `*_url` / `email` / `preferred_username` | 制約追加候補 | §1。要否と `N` は wi-128 |
| 固定書式の識別子 | `tenants.id`（`CHECK ~ 正規表現`、mutable slug） | 現状維持（`TEXT`） | §1 / §4。UUID+realm 分割は別 WI |
| 内部生成 id（本 WI で UUID 化） | `users.id`、`clients.client_id`、`groups.id`、`agents.id`、`audit_events.id`、`scim_tokens.id` | 型変更（`TEXT`→`UUID`） | §4。idmagic 内部生成で値域を UUID に閉じる |
| 内部生成 id 参照（本 WI で UUID 化） | `user_id` 系、`owner_user_id`、`group_id`、`agent_id`、`client_id` 系、`subject_id`、`audit_events.user_id` | 型変更（`TEXT`→`UUID`） | §4。PK と型を一致させる |
| 既存 UUID id | `refresh_tokens.{id,family_id,parent_id}`、`applications.application_id`、`application_categories.category_id` | 現状維持（`UUID`） | §4 |
| 外部が値を決める id | `entity_id`、`wtrealm`、`scim_id`、`kid` | 現状維持（`TEXT`） | §4。idmagic が採番しない |
| 半構造化 / append-only JSONB | `redirect_uris` 他、`*.rules` / `claim_policy` / `payload` / `roles` / `scopes` | 現状維持（`JSONB`） | §2 |
| 状態値を含む JSONB | `users.lifecycle` | 正規化候補 | §2 / §5 |
| 有限集合 `TEXT`（CHECK 済み） | `tenants.status`、`clients.client_type`、`agents.status`、`authorization_detail_types.state`、`application_assignments.subject_type` / `visibility` | 現状維持 | §5 |
| 有限集合 `TEXT`（CHECK なし） | `agents.kind`、`applications.kind` / `status`、`clients.fapi_profile` | 制約追加候補 | §5 |
| 外部語彙の `TEXT` | `token_endpoint_auth_method`、`id_token_signed_response_alg`、`signing_keys.alg` | 現状維持 | §5 |
| 時刻 | `created_at` / `updated_at` / domain 時刻 | 現状維持（`TIMESTAMPTZ`） | §3 |
| 数値・真偽・配列・バイナリ | `size_bytes`、`count`、boolean、`*_ids`（`TEXT[]`）、`data`（`BYTEA`） | 現状維持 | 用途に整合 |

## follow-up 候補（本 WI の Out of Scope）

- **`tenants.id` の UUID + realm 分割**: 不変 UUID PK と mutable な一意 `realm` カラムに
  分離し realm rename を可能にする。tenant_id FK の全参照・URL ルーティング・
  `spec.DefaultTenantID` に波及するため独立 ADR/WI（カラム名 `realm`）。
- **文字列長の DB 防衛線**: `wi-128-string-length-limits-policy`。
- **`users.lifecycle` の状態列昇格**: `lifecycle->>'status'` を専用列 + `CHECK` へ正規化。
- **CHECK なし有限集合列の CHECK 付与**: `agents.kind` 等を列ごとに個別 WI 化。

## 却下した代替案

- **内部生成 id も含め全 domain id を `TEXT` のまま残す。** UUID 値を TEXT 列に格納する
  不整合と、既存 UUID 列との非対称を温存する。未リリースで是正コストが最小の今、内部生成 id は
  UUID 型に閉じる。
- **id の Go 型も `[16]byte` / `uuid.UUID` に変更する。** OAuth2/OIDC の claim 直列化や
  protocol 境界で id は文字列として扱われ、全層の型変更は波及が大きい。列型のみ `UUID` にし、
  Go は `string` を保ち、pgx の text codec で橋渡しする。
- **外部が値を決める id（`entity_id` / `wtrealm` / `scim_id` / `kid`）も UUID 化する。**
  値を idmagic が採番せず、URI や thumbprint など UUID でない値が入るため不可能。
- **`tenants.id` を本 WI で即時 UUID 化する。** tenant_id FK の全参照・URL・
  `DefaultTenantID` に波及し、mutable slug の分離も要る。realm 分割として別 WI に切り出す。
- **全有限集合列を PostgreSQL enum 化 / 全 `TEXT` を `varchar(N)` 化する。** migration
  friction と未確定の `N` を先取りする。既定は `TEXT` + `CHECK`、長さは wi-128。

## 影響

- `deploy/schema/postgres.sql`: 冒頭に本 ADR を指す「Column type policy」コメントを追加。
  内部生成 id の PK と参照 FK 列を `TEXT`→`UUID` に変更。
- `internal/shared/adapters/persistence/postgres/base.go`: `AfterConnect` で uuid OID に
  text codec を登録し、`UUID` 列を Go `string` で読み書きできるようにする。
- `internal/bootstrap/seed.go` と UI（`ui/src/api/oidc.ts` / `authFlow.ts`）: first-party /
  demo の非 UUID id（`user_alice` / `demo-client` / `idmagic-admin-console` /
  `idmagic-account-portal` / `group_engineering` 等）を固定 UUID に是正し、application
  bindings・SPA の OIDC 設定を追随させる。
- 既存テストの非 UUID id リテラルを UUID に更新する（embedded-postgres が uuid 列型を強制）。
- SCL・公開 API・derived artifacts への影響は無い。id の JSON 表現は従来どおり文字列。
