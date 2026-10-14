---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# saml context の `WriteBrowserError` 呼び出しを Problem Details へ移行する

`backend/saml/handlers_http` の `admin_idp_profile_handler.go` (8 箇所)・
`admin_service_provider_handler.go` (5 箇所) をすべて
`support.WriteProblem` へ置き換えた。status 変更は不要
(`profile_in_use`/`application_owned_protocol` は既に 409、`not_found` は 404)。
長さ違反を大域のエラー写像に 422 として作らせている箇所 (wi-128) の
コメントが旧ヘルパー名を指していたため現在の名前へ直した。
SAML プロトコル応答 (AuthnRequest/Response、metadata XML) は対象外のまま
変更していない。
仕様変更はない (`just spec-diff`: no normative specification change)。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
