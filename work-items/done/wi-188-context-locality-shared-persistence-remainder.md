---
status: completed
archived_from: 9ded39c77097f72b22ada22e09d5b596271c6e3b
---

# shared 永続化アダプタに残る OAuth2 / Authentication 固有実装をコンテキストローカリティ化する

OAuth2 と Authentication 固有の memory / Postgres / Valkey 永続化
アダプタを各コンテキストへ移設し、shared には接続設定と共通ユーティリティのみを残した。
HTTP の中央 `Deps` から対象ストアを撤去し、各 Module を唯一の配線窓口にした。

全文は `git show 9ded39c77097f72b22ada22e09d5b596271c6e3b:<このファイルのパス>` で読む。
