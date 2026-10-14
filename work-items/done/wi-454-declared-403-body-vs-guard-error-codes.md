---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 403 が名乗る本体を、guard が実際に返すエラーコードに合わせる

`mise run spec-diff` は `REQ-PLATFORM-004`、`InsufficientScopeError`、`InvalidOriginError`、`CsrfFailedError` などの追加と `REQ-APITOKENS-004` の変更を報告した。各 operation の 403 本文を、到達可能な管理権限、API トークンスコープ、origin、CSRF、ステップアップ、リンク解除の拒否コードへ合わせた。`check-status-drift` は Go の guard 呼び出し鎖から `(status, code)` を伝播し、OpenAPI の `$ref` と union を解決して不足する 403 型を検出する。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
