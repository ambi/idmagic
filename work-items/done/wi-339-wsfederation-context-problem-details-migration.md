---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# wsfederation context の `WriteBrowserError` 呼び出しを Problem Details へ移行する

`backend/wsfederation/handlers_http` の `admin_relying_party_handler.go`
(4 箇所)・`admin_entra_handler.go` (3 箇所) をすべて `support.WriteProblem`
へ置き換えた。status 変更は不要 (`invalid_request` 400・
`application_owned_protocol` 409)。WS-Federation プロトコル応答
(`wsignin1.0`/`wsignout1.0` の XML・redirect、WS-Trust SOAP) は対象外のまま
変更していない。長さ違反が既に 422 Problem Details で返っていること
(`admin_relying_party_length_test.go`、wi-128) も引き続き green。
仕様変更はない (`just spec-diff`: no normative specification change)。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
