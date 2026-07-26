---
status: accepted
authors: [tn]
created_at: 2026-07-05
---

# ADR-084: Postgres 列型選定ポリシーと内部 id の UUID 型化

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

`postgres.sql` を横断する列型選定基準を定め、あわせて idmagic が内部生成する id 列を `UUID` 型へ
整列させる。永続化層の内部方針であり SCL の公開 contract・非機能保証を変更しないため
`spec/scl.yaml` の変更は伴わない。wi-127 で導入。

**規則そのものは [`ARCHITECTURE.md`](../ARCHITECTURE.md) の `## Cross-cutting Concerns` >
データベース設計ポリシー が現行の正本である**（ADR-143 で移送）。この ADR が保つのは、そこから
再導出できない「なぜその基準にしたか」だけである。

- 内部生成 id を `UUID` 型に閉じられるのは、値域を idmagic 自身が定義できるためである。
  非 UUID の seed（`user_alice` 等）を UUID に是正すれば済む。未リリースの今なら FK 参照列もろとも
  宣言的 schema を書き換えるだけで、後からの変更より安い。
- 外部が値を決める id はそもそも UUID 化できない。SAML `entity_id` / WS-Fed `wtrealm` は外部 SP/RP の
  メタデータが定める URI、`signing_keys.kid` は RFC 7638 JWK thumbprint 由来の指紋、`scim_id` は SCIM
  リソース id であり、いずれも idmagic が採番しない。
- 有限集合に PostgreSQL enum を使わないのは migration friction のためである。値の追加が
  ALTER TYPE を要し、宣言的 schema の差分適用と相性が悪い。
- `TIMESTAMPTZ` を丸めないのは、秒精度への丸めが必要なのは外部プロトコル境界
  （SCIM/SAML/WS-Fed の書式）だけで、保存側で丸めると復元できない情報を失うためである。
- 具体的な最大文字数は `wi-128-string-length-limits-policy`、`tenants.id` の UUID + realm 分割は
  ADR-085 に委ねた。本 ADR で先取りすると未確定の `N` と広い波及を抱え込む。

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

- `infra/schema/postgres.sql`: 内部生成 id の PK と参照 FK 列を `TEXT`→`UUID` に変更。
  規則の本文は `ARCHITECTURE.md` が持ち、schema 冒頭はそこへのポインタだけを置く（ADR-143）。
- `internal/shared/adapters/persistence/postgres/base.go`: `AfterConnect` で uuid OID に
  text codec を登録し、`UUID` 列を Go `string` で読み書きできるようにする。
- `internal/bootstrap/seed.go` と UI（`ui/src/api/oidc.ts` / `authFlow.ts`）: first-party /
  demo の非 UUID id（`user_alice` / `demo-client` / `idmagic-admin-console` /
  `idmagic-account-portal` / `group_engineering` 等）を固定 UUID に是正し、application
  bindings・SPA の OIDC 設定を追随させる。
- 既存テストの非 UUID id リテラルを UUID に更新する（embedded-postgres が uuid 列型を強制）。
- SCL・公開 API・derived artifacts への影響は無い。id の JSON 表現は従来どおり文字列。
