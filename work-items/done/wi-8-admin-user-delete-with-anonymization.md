---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# 管理画面からのユーザ削除 (anonymize cascade) を導入する

`/admin/users` に DELETE 経路を追加し、削除を「即時 anonymize cascade」
として実装した。ADR-036 を新規に採用し、SCL では User 状態機械
`UserLifecycle` の追加・終端状態 `Deleted`・`DeleteUser` interface・
`UserDeleted` event・`AdminUserDelete` permission を入れた。
バックエンドは新規 use case `authusecases.DeleteUser` で tombstone 化と
関連 aggregate (Consent / RefreshToken / Session / PasswordHistory /
MfaFactor / DeviceAuthorization) の cascade 削除を一括で行う。HTTP は
`DELETE /api/admin/users/:sub` を既存 `verifyBrowserRequest` +
`requireAdmin` の上に乗せ、UI は AdminUsersPage に "アカウントを削除"
危険ボタンと preferred_username typing 確認ダイアログを追加した。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
