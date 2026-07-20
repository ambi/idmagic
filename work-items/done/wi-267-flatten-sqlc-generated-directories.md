---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-20
depends_on: [wi-266-flat-wikipedia-architecture]
---

# sqlc の生成コードとクエリを `db_postgres` 直下へフラット化する

## Motivation
`wi-266` によって backend のアダプター階層はフラットになったが、`sqlc` が使用する `queries/` ディレクトリと、出力される `sqlcgen/` ディレクトリが依然として深くネストされた状態になっている。これらを `db_postgres` 階層へ直接配置し、リポジトリ層と同じパッケージ (`db_postgres`) に同居させることで、不要なサブディレクトリを排除し、さらに可読性と凝集度を向上させる。

## Scope
- `sqlc.yaml` におけるすべての `db_postgres` ターゲットの `queries` および `out` ディレクトリの設定変更
- `backend/` 配下すべてのコンテキストにおける `queries/*.sql` の `db_postgres/` 直下への移動
- 生成先パッケージ名 `sqlcgen` の廃止と `db_postgres` への統一
- `backend/oauth2` における `client`, `consent`, `token` モジュールから親階層への不要なエイリアス依存（`feature_aliases.go`）の解決
- 手書きの Go コード（リポジトリ層）からの `sqlcgen` プレフィックスの除去

## Out of Scope
- SQL クエリそのもののロジック変更
- データベースのマイグレーション
- HTTP 契約や SCL スキーマの変更

## Plan
1. `sqlc.yaml` の各ターゲットに対し、`queries` と `out` を直接 `db_postgres` に向けるよう修正し、生成パッケージを `db_postgres` とする。
2. 既存の `queries/` 内の SQL ファイルを `db_postgres/` 直下へ移動し、`queries/` および古い `sqlcgen/` ディレクトリを削除する。
3. `backend/oauth2/db_postgres/feature_aliases.go` を削除し、`cmd/internal/bootstrap/postgres_valkey.go` および `shared/storage/fixtures_postgres/pgfixtures.go` で、直接サブモジュール (`client`, `consent`, `token`) の `db_postgres` パッケージをインポートするようにして依存の循環を解消する。
4. 全コンテキストの `db_postgres` 配下の Go ファイルから、`sqlcgen` パッケージのインポートとプレフィックスを削除する。
5. `just sqlc-generate` でコードを再生成し、`just verify` を通じて静的解析・テストがグリーンであることを確認する。

## Tasks
- [x] T001 [Codegen] `sqlc.yaml` を更新し、既存のディレクトリ構造（`queries`, `sqlcgen`）を整理する。
- [x] T002 [App] 各手書きのリポジトリ実装の Go コードから、`sqlcgen` への依存を排除する。
- [x] T003 [App] `oauth2` サブモジュールの依存循環を解消するため、`feature_aliases.go` を削除して直接依存へ切り替える。
- [x] T004 [Verify] 自動生成とビルドを実行し、すべてのテストを通過させる。

## Verification
- `just sqlc-generate`
- `just format-go`
- `just yaml-check-work-items`
- `just verify`

## Risk Notes
- 大規模なディレクトリ移動とリファクタリングを伴うため、インポートの循環依存が発生するリスクがある。事前に依存グラフを検証し、Go コンパイラのチェック (`go build`) を通すことで担保する。

## Completion
- **Completed At**: 2026-07-20
- **Summary**:
  `sqlcgen` ディレクトリと `queries` ディレクトリをすべて `db_postgres` 階層へ統合し、`sqlc.yaml` と対象ソースコードからのインポートをフラット化した。`oauth2` コンテキストでのエイリアス経由の循環依存と、手動リポジトリでの名前衝突 (`saveUser`) およびテストファイル（`_test.go`）でのインポート循環をそれぞれ解消し、全テストパスを確認した。
- **Verification Results**:
  - `just verify-go` - passed
  - `just yaml-check-work-items` - passed
