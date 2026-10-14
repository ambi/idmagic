---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# workloadidentity context の `WriteBrowserError` 呼び出しを Problem Details へ移行する

`backend/workloadidentity/handlers_http/routes.go` の 15 箇所を
`support.WriteProblem` へ置き換え、granular 7 code
(`workload_trust_bundle_jwks_required`・`workload_trust_bundle_name_required`・
`workload_trust_bundle_issuer_required`・
`workload_trust_bundle_audiences_required`・`workload_trust_bundle_invalid_ttl`・
`agent_workload_binding_agent_not_found`・
`agent_workload_binding_pattern_required`) の status を 400 から仕様の
宣言値 422 に揃えた。残り 8 箇所 (`invalid_request` 400、
`*_not_found` 404、`*_conflict` 409) は status 据え置き。
このパッケージには HTTP レベルのテストが 1 つもなかったため、
`routes_test.go` を新設して trust bundle 登録の name 欠落ケースを固定した。
仕様変更はない (`just spec-diff`: no normative specification change)。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
