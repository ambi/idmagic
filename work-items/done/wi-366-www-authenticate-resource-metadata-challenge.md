---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 401 応答の WWW-Authenticate に resource_metadata を付し、保護リソースメタデータへ誘導する

RFC 9728 の `RFC9728-CHALLENGE` と REQ-OAUTH2-044 を追加した。共通 Bearer challenge helper は
tenant middleware の canonical issuer と TypeSpec-generated `GetProtectedResourceMetadata` path
から `resource_metadata` URL を導出し、共通 API と SCIM の 401 `invalid_token` および 403
`insufficient_scope` に quoted auth-param として付加する。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
