/** 境界検査の受け入れテストが共有する、宣言と違反のない最小の作業ツリー。 */
export const BOUNDARY_FIXTURE: Record<string, string> = {
  'go.mod': 'module example.com/product\n',
  'docs/design/architecture/logical.md': [
    '# Logical architecture',
    '',
    '| モジュール | 公開方式 | 公開パッケージ | Go パッケージ | 責務 |',
    '| --- | --- | --- | --- | --- |',
    '| [System](system.md) | 組み立て地点 | なし | `backend/cmd` | Wiring |',
    '| [Supplier](supplier.md) | `legacy` | `domain` または `ports` の区画 | `backend/supplier` | Supplies a model |',
    '| [Customer](customer.md) | `legacy` | `domain` または `ports` の区画 | `backend/customer` | Uses the model |',
    '',
  ].join('\n'),
  'docs/design/data/database.md': [
    '| テーブル | 役割 | 所有モジュール | テーブル種別 | `tenant_id` カラム |',
    '| --- | --- | --- | --- | --- |',
    '| `supplies` | Supplies | Supplier | `LOGGED` | なし |',
    '| `orders` | Orders | Customer | `LOGGED` | なし |',
    '| `counters` | Counters | 共通基盤 | `LOGGED` | なし |',
    '',
  ].join('\n'),
  'infra/schema/postgres.sql': [
    'CREATE TABLE supplies (id uuid PRIMARY KEY, name text, stock int);',
    'CREATE TABLE orders (id uuid PRIMARY KEY, supply_id uuid);',
    'CREATE TABLE counters (key text PRIMARY KEY, hits int);',
    '',
  ].join('\n'),
  'sqlc.yaml': [
    'version: "2"',
    'sql:',
    '  - engine: "postgresql"',
    '    schema: "infra/schema/postgres.sql"',
    '    queries: "backend/customer/db_postgres"',
    '  - engine: "postgresql"',
    '    schema: "infra/schema/postgres.sql"',
    '    queries: "backend/shared/counter/db_postgres"',
    '',
  ].join('\n'),
  'backend/customer/db_postgres/orders.sql':
    '-- name: AddOrder :exec\nINSERT INTO orders (id, supply_id) VALUES ($1, $2);\n',
  'backend/shared/counter/db_postgres/counters.sql':
    '-- name: Hit :exec\nUPDATE counters SET hits = hits + 1 WHERE key = $1;\n',
  'tools/check/boundary-debt.json': '{"violations":[]}\n',
}
