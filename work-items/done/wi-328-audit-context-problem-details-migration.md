---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# audit context の `WriteBrowserError` 呼び出しを Problem Details へ移行する

`backend/audit/handlers_http/admin_audit_event_handler.go` の 7 箇所
(起票時の調査では 5 箇所、その後の変更で 2 箇所増えていた) をすべて
`support.WriteProblem` へ置き換え、audit context の管理 API エラーが
`application/problem+json` を返すようになった。`invalid_request` 400 と
`event_not_found` 404 のみで、status を変える call site はなかった。
仕様変更はない (`just spec-diff`: no normative specification change)。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
