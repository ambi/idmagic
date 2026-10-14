---
status: completed
archived_from: bd82352b3b788c86e8ae903c0da4aab8807c9586
---

# PostgreSQLの結果読取を早期キャンセルせずProblem DetailsをUIへ正しく伝える

PostgreSQL の query timeout context を `Row.Scan` / `Rows.Close` または iteration 終端まで
保持し、実際の結果読取成否を circuit breaker へ一度だけ報告するよう修正した。正常な
`pgx.ErrNoRows` は breaker failure から除外する。frontend は旧 error envelope と RFC 9457
Problem Details を共通解釈し、generic request と独自 fetch 経路の双方で `detail` / `title` と
`type` suffix を利用する。

全文は `git show bd82352b3b788c86e8ae903c0da4aab8807c9586:<このファイルのパス>` で読む。
