---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# `/userinfo` の `invalid_token` を RFC 6750 のとおり 401 と `WWW-Authenticate` で返す

`mise run spec-diff` は `RFC6750-INVALID-TOKEN` の追加と `REQ-OAUTH2-020` の変更を報告した。GET/POST の UserInfo は、失効済みアクセストークンを `401 invalid_token` と Bearer challenge で拒否し、クレームを返さない。`insufficient_scope` も同じ保護資源境界で 403 と challenge に写像し、認可サーバー用の汎用 400 写像とは分離した。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
