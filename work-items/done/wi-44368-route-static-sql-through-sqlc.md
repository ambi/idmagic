---
status: completed
archived_from: 633b478a13e9438aae91e1fcddb6990b13d00e1a
---

# 静的な SQL 文を sqlc へ移し、直接の SQL 文字列を検査で拒否する

`mise run spec-diff` は規範仕様の差分を報告しない（`no normative specification change against main`）。
`backend/` のテスト以外の Go コードで SQL 文字列を直接渡していた 15 ファイル 42 か所のうち、11 ファイルの文を sqlc の問い合わせへ移した。
残る 4 ファイル 9 か所（監査イベントの検索、`ResilientDB` の中継、DDL の適用、接続ごとのセッション設定）には `//sql:raw <理由>` を置いた。
リポジトリ検査 `raw-sql` を `check-repository` に加え、理由のある印のない直接の SQL 呼び出しを拒否する。
データベース設計に、sqlc で表せない場合の一覧と例外の書き方を追記した。
移行の途中で、相関 salt の PostgreSQL 実装を `backend/shared/security/salts_postgres` へ移し、CSV の成果物の読み取りを 16 チャンクずつの問い合わせに変えた。
追加した特性テストで、SCIM の一括取得（`FindUserRefsByUserIDs`、`FindGroupRefsByGroupIDs`）が PostgreSQL では必ず失敗する既存の不具合が見つかったので、引数を `text[]` で受けるよう直した。

全文は `git show 633b478a13e9438aae91e1fcddb6990b13d00e1a:<このファイルのパス>` で読む。
