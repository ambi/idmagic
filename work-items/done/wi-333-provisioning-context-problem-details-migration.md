---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# provisioning context の `WriteBrowserError` 呼び出しを Problem Details へ移行する

`backend/provisioning/handlers_http` の `handlers.go` (8 箇所)・`routes.go`
(3 箇所) をすべて `support.WriteProblem` へ置き換えた (起票時の調査より
4 箇所増えていた)。Design の想定どおり status 変更は不要で、
`provisioning_not_found` 404・`provisioning_conflict` 409・
`invalid_request` 400 のまま envelope だけ変えた。
仕様変更はない (`just spec-diff`: no normative specification change)。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
