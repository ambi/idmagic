---
status: completed
archived_from: 3517ed843c38ab50bab11762434092cd2619364c
---

# sqlc の生成モデルを、その context の query が使う型だけに絞る

`mise run spec-diff` は `no normative specification change against main` を返す。
規範的シナリオ、標準要件、TypeSpec 宣言のいずれも追加、変更、退役していない。
意味上の差分は生成コードの側だけにある。
`sqlc.yaml` の 30 個の生成エントリーすべてが `omit_unused_structs: true` を宣言するようになり、
各 context の `models.go` は、その context の query が参照するテーブルの構造体だけを持つ。
30 ファイルの合計は 31,620 行 2,640 型から 810 行 45 型へ減り、30,810 行 2,595 型が消えた。
代表 3 パッケージでは `backend/jobs/db_postgres` が 1054 行 88 型から 30 行 1 型へ、
`backend/oauth2/db_postgres` が 106 行 7 型へ、`backend/tenancy/db_postgres` が 87 行 5 型へ変わった。
`querier.go` と `*.sql.go` は 1 バイトも変わっておらず、
変更されたのは 30 個の `models.go` と `sqlc.yaml` だけである。
query 関数の署名が変わらないという Design の前提は、この diff の形そのもので確かめられた。
意図は `sqlc.yaml` 冒頭のコメントに置いた。

全文は `git show 3517ed843c38ab50bab11762434092cd2619364c:<このファイルのパス>` で読む。
