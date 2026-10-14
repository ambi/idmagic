---
status: completed
archived_from: 0a6f958e82e03c4352fa2c1402ff5b90b3faccf4
---

# 依存の少ないモジュールを一つ選び、Go の internal/ へ移す

`mise run spec-diff` は main に対して規範仕様の差分がないことを示した。
Authorization の本番パッケージ 8 個（`domain`、`ports`、`usecases`、`db_memory`、`db_postgres`、`handlers_http`、`principals_idmanagement`、`testing_contract`）を `backend/authorization/internal/` へ移し、責務表の公開方式を `internal`、公開パッケージをなしとした。
ルートパッケージは `NewMemoryModule`、`NewPostgresModule`、`Module.RegisterRoutes` を持ち、組み立て地点（`cmd/internal/bootstrap`、`server_http`）はこの三つだけを使う。
移行で見つかった手順の不足を二つ検査へ反映し、構造に移行の手順として書いた。
`codeSlices` は `internal` の区画を飛ばして機能スライスを読み、主要ユースケースの証拠の検査は、完了した記録のテストのパスを `internal/` の下で読み直す。
故障の注入で見つけた、組み立てを固定するテストの欠けを二つ埋めた。

全文は `git show 0a6f958e82e03c4352fa2c1402ff5b90b3faccf4:<このファイルのパス>` で読む。
