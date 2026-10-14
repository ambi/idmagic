---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# Postgres 列型選定ポリシーの明文化と既存 schema の棚卸し

Postgres 列型選定ポリシーを ADR として明文化したうえで、レビュー判断により
「idmagic が内部生成する id 列を UUID 型に閉じる」方針転換を取り込み、schema・adapter・
seed・UI・テストを実際に整列させた。tenants.id の UUID 化と文字列長制約は follow-up に分離。
- `decisions/ADR-084-postgres-column-type-policy.md` を追加。文字列（制約なし varchar
不使用）・JSONB 許容基準・TIMESTAMPTZ 精度・UUID と domain string id の境界・有限集合値
表現（既定 TEXT+CHECK、PG enum は原則不採用）の 5 基準と、全列の型カテゴリ別棚卸し表を掲載。
§4 で「内部生成 id は UUID 型、外部が値を決める id（`entity_id`/`wtrealm`/`scim_id`/`kid`）は
TEXT」を確定。ADR-071 / ADR-082 / ADR-083 と整合。
- **内部生成 id の UUID 型化**: `deploy/schema/postgres.sql` で `users.id`・`clients.client_id`・
`groups.id`・`agents.id`・`audit_events.id`・`scim_tokens.id` と、それらを参照する FK 列
（`user_id` 系・`owner_user_id`・`group_id`・`agent_id`・`client_id` 系・`subject_id`・
`audit_events.user_id`）を `TEXT`→`UUID` に変更。外部が値を決める id と `tenants.id`（slug）は
TEXT を維持。
- **Go は string を維持**: `internal/shared/adapters/persistence/postgres/base.go` の
`AfterConnect` で uuid OID に text codec を登録（`RegisterUUIDAsText`）し、UUID 列を Go の
`string` で read/write。test harness も同 codec を登録。id 型は全層で `string` のまま。
- `application_assignments.ListBySubjects` の UNNEST 比較を、param は `text[]` のまま列側を
`subject_id::text` にキャストして uuid=text 不一致を解消。
- **seed / UI の固定 UUID 化**: `internal/bootstrap/seed.go` の非 UUID id（`user_alice` /
`user_root` / `demo-client` / `idmagic-admin-console` / `idmagic-account-portal` /
`group_engineering` / `group_support`）を固定 UUID（`00000000-0000-4000-8000-...`）へ定数化。
application binding と UI の OIDC 設定（`ui/src/api/oidc.ts` / `authFlow.ts`）を同値に追随。
- Postgres roundtrip テストの id fixture / inline リテラルを UUID（`newUUID(t)`）に更新。
- 時刻精度マップを ADR §3 に明記（PostgreSQL マイクロ秒保存、pgx `time.Time` 精度保持、内部
JSON は RFC3339Nano、秒精度化は SCIM/SAML/WS-Fed/audit クエリの外部境界のみ、UI はミリ秒）。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
