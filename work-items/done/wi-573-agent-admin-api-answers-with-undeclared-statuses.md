---
status: completed
archived_from: e50ccb1f62ec962e4c3e592b991b0b096589a9d2
---

# Agent 管理 API が、契約の宣言に無い 404 と 409 で拒否する

`mise run spec-diff` は `REQ-IDMANAGEMENT-009` の具体例の変更と、`AgentNotFoundError`、`AgentKilledError`、`AgentClientAlreadyBoundError` の 3 個の TypeSpec 宣言の追加を報告した。起票時の方針（実装を宣言済みの 400 へ寄せる）は、API ガイドラインの「400 と 422 の区別」「一意性違反と状態の競合」、および既存 API の 404 の宣言と食い違っていた。このため、ガイドラインと既存 API に合わせて宣言を広げる方向へ改めた。Agent 管理 API 8 operation に 404、409、422 を宣言し、状態変更 4 operation には欠けていた 401 と 403 の union も宣言した。実装の変更は、本文の `client_id` が見つからない拒否を 404 から 422 へ移した 1 箇所だけである。`EX-IDMANAGEMENT-009-04` は `OAuth2ClientNotFoundError` と、存在しない `client_id` と同じ応答であることを要求する形へ訂正し、負債台帳から外した。

全文は `git show e50ccb1f62ec962e4c3e592b991b0b096589a9d2:<このファイルのパス>` で読む。
