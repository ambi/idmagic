---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# signingkeys context の `WriteBrowserError` 呼び出しを Problem Details へ移行する

`backend/signingkeys/handlers_http/admin_key_handler.go` の 6 箇所を
`support.WriteProblem` へ置き換えた。status 変更は不要
(`key_not_found` 404・`key_store_unavailable` 503・
`active_key_cannot_be_disabled` 400)。
このパッケージ自身にハンドラテストはないが、`/api/admin/v1/keys` は
`backend/oauth2/handlers_http/admin_key_handler_test.go` の harness から
実際に叩かれているため、そこの 404 ケースに Content-Type と `type` URN の
照合を足して RED を作った。
仕様変更はない (`just spec-diff`: no normative specification change)。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
