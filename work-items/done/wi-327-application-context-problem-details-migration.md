---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# application context の `WriteBrowserError` 呼び出しを Problem Details へ移行する

`backend/application/handlers_http` の 5 ファイル・54 箇所の
`WriteBrowserError` をすべて `support.WriteProblem` へ置き換え、application
context の汎用 API エラーが `{error, message}` ではなく RFC 9457 Problem
Details (`application/problem+json`、`type`/`title`/`status`/`detail`/`instance`)
を返すようになった。
status を仕様の宣言値へ揃えたのは 3 箇所:
`category_name_required`・`unknown_category` (400 → 422、
`CreateApplicationCategoryError422`/`UpdateApplicationCategoryError422`/
`SetApplicationCategoriesError422`)、および実装時に判明した
`client_secret_limit_exceeded` (409 → 422)。後者は work item 起票時の
一覧になかったが、`IssueApplicationClientSecret` が宣言するエラー status は
400/403/422 のみで 409 は宣言外であり、`ClientSecretLimitExceededError` は
422 union の要素だったため、envelope 移行と同時に揃えた。
それ以外の call site (`invalid_request` 400、`not_found`/`application_not_found`/
`category_not_found` 404、`authentication_required` 401、`invalid_icon`/
`invalid_sign_in_policy` 400) は status を変えず envelope だけ変えた。
仕様変更はない (`just spec-diff`: no normative specification change) —
TypeSpec 側は wi-325 時点で宣言済みで、この work item は実装をその宣言に
追従させただけである。
**対応していないこと**:
- `WriteBrowserError` 自体の削除は `wi-340`。
- このコンテキストに OAuth2/SCIM/DCR、SharedSignals inbound SET receiver の
call site はないため、protocol 応答形式は一切触れていない。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
