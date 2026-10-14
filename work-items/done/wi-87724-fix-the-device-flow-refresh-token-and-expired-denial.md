---
status: completed
archived_from: eb01710bbdc5e722cd30e78cd7107b282a06b854
---

# デバイス認可で、offline_access のない交換にリフレッシュトークンを発行せず、期限切れの拒否を受け付けない

`mise run spec-diff -- main` の結果、REQ-OAUTH2-021 と REQ-OAUTH2-027 の要件文が変わった。
REQ-OAUTH2-021 は、リフレッシュトークンを `offline_access` のスコープのときだけ発行する規則を、認可コードと `device_code` の両方の交換に定める。REQ-OAUTH2-027 の交換の条項は、`offline_access` のスコープにだけリフレッシュトークンを返すと定める。
`device_code` の交換は、スコープに `offline_access` を含まないとき、アクセストークンと ID トークンだけを返し、リフレッシュトークンを保存せず、`RefreshTokenIssued` を発行しない。発行の判定はドメインの `GrantsRefreshToken` に置き、認可コードの交換と共有する。
有効期間を過ぎた `user_code` の拒否は、承認と同じく `expired_token` で拒否され、記録は `Denied` にならず、`DeviceAuthorizationDenied` を発行しない。

全文は `git show eb01710bbdc5e722cd30e78cd7107b282a06b854:<このファイルのパス>` で読む。
