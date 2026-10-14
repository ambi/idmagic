---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 別テナントの Application API へ提示した API トークンの拒否を宣言と一致させる

`mise run spec-diff` は REQ-APPLICATION-003 と REQ-APPLICATION-004 のシナリオ変更、および `InvalidAccessTokenError` と `AuthenticationRequiredResponseBody` の TypeSpec 宣言追加を報告した。
発行先テナントとは異なる既知のテナントへ API アクセストークンを提示した場合は、account API と admin API の双方で 401 `invalid_token` を返す公開契約になった。
資格情報がない場合は 401 `authentication_required`、同一テナントで認証済みの主体に権限がない場合は 403、未知のテナントまたは認証および認可後に見つからない資源は 404 とする区分も正準設計へ記録した。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
