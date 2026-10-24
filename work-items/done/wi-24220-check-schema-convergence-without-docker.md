---
status: completed
---

# スキーマ収束の検査を Docker なしで実行する

`mise run check-schema` は、Docker Compose ではなく `idmagic-dev-infra --empty --postgres-port 0` が起動する組込み PostgreSQL に対し、`mise.toml` が固定した `psqldef` で収束を検査する。
規範仕様の差分はない。
Go の DB テストはスキーマを PostgreSQL へ直接流すので、`psqldef` だけが読めない DDL（予約語の列名など）はこの検査でしか検出できない。
これを Docker のない手元でも実行できるようにした。
接続情報は準備完了ファイルの `database_url` から取り出す。そのために、`jq` も `mise.toml` で版固定した。
