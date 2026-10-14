---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# idgovernance context の `WriteBrowserError` 呼び出しを Problem Details へ移行する

`backend/idgovernance/handlers_http/admin_lifecycle_workflow_handler.go` の
10 箇所をすべて `support.WriteProblem` へ置き換えた。Design の想定どおり
status を変える call site はなかった (`workflow_revision_conflict`/
`workflow_name_conflict` は既に 409、`workflow_not_found` は 404、
残りは `invalid_request`/`invalid_workflow` の 400)。
既存ハンドラテストは status しか固定していなかったため、dry_run の
存在しないユーザー指定ケースに Content-Type と `type` URN の照合を足して
RED を作った。
仕様変更はない (`just spec-diff`: no normative specification change)。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
