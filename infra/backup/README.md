# PostgreSQL バックアップ・復元スクリプト

ここにあるスクリプトは、バックアップと災害復旧戦略のうち、`pg_dump` と論理復元の経路を実装する。手順、災害復旧のシナリオ、検証チェックリストは [`infra/runbooks/backup-restore-dr.md`](../runbooks/backup-restore-dr.md) にあり、この README はスクリプトの実行方法だけを扱う。

## 必要なもの

- `pg_dump` / `pg_restore` / `psql`（`infra/docker/docker-compose.dev.yaml` のサーバーバージョンに合わせた PostgreSQL 17 クライアントツール）。
- `psqldef`（インストール手順は `infra/schema/README.md` を参照）。
- Go ツールチェーン（`restore-postgres.sh` が `go run` 経由で呼び出す `idmagic-batch restore-consistency-check` 用）。

PostgreSQL クライアントは開発用の `mise` 管理には含めない。バックアップまたは復元を実行する運用環境に PostgreSQL 17 クライアントを用意する。

## 接続変数

すべてのスクリプトは標準の libpq 環境変数で接続する。これは `infra/schema/README.md` が `psqldef` に使うのと同じ規約である。

```bash
export PGHOST=localhost
export PGPORT=5432
export PGUSER=idmagic
export PGPASSWORD=idmagic
export PGDATABASE=idmagic
```

デフォルトの対象はない。バックアップや復元が誤ったデータベースを暗黙に対象としないよう、すべてのスクリプトがこれらの明示的な設定を要求する。

## 使用方法

```bash
# バックアップを取得する（pg_dump カスタム形式と SHA-256 チェックサム）。
mise run backup-postgres <output-dir>

# 空の対象データベースへバックアップを復元する。本番環境以外であることを保証するため、データベース名を明示的に入力しなければならない。
mise run restore-postgres <backup-file> <db-name>

# 使い捨ての Docker Compose プロジェクトに対して、ローカルバックアップ、損失の模擬、復元、整合性検査からなる訓練全体を実行する。
mise run restore-drill
```

`restore-postgres.sh` は、テナントの行がすでにあるデータベースに対する実行を拒否する。新しく作った空のデータベースへ復元すること。最初に `psqldef` で `infra/schema/postgres.sql` を適用し、データを復元し、一時的な `UNLOGGED` / `LOGGED` テーブルを空にして、最後に `idmagic-batch restore-consistency-check` を実行する。
