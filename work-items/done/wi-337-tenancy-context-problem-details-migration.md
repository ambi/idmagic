---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# tenancy context の `WriteBrowserError` 呼び出しを Problem Details へ移行する

`backend/tenancy/handlers_http` の 8 ファイル・38 箇所の `WriteBrowserError`
をすべて `support.WriteProblem` へ置き換えた (起票時の 7 ファイルに加えて
その後追加された `admin_group_attribute_schema_handler.go` を含む)。
status を仕様の宣言値 422 へ揃えたのは 3 箇所:
`admin_tenant_handler.go` の `policy_override_weaker`
(`PolicyOverrideWeakerError`)、`admin_user_attribute_schema_handler.go` と
`admin_group_attribute_schema_handler.go` の `invalid_attribute_schema`
(`InvalidUserAttributeSchemaError`/`InvalidGroupAttributeSchemaError`)。
`attribute_referenced_by_dynamic_group` 409、`tenant_conflict` 409、
`tenant_not_found`/`not_found` 404、`federation_credentials_unavailable` 503
は既存 status のまま envelope だけ変えた。
仕様変更はない (`just spec-diff`: no normative specification change)。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
