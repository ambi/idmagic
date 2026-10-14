---
status: completed
archived_from: 61f5d0fe18c17eadc4a23f1b9048389f098c55ca
---

# 管理者が認証器をリセットし再登録を強制する緊急復旧導線を追加する

- 管理者操作 `ResetUserAuthenticators` (`POST /api/admin/users/{user_id}/authenticator-reset`,
policy `TenantAdministrator`) を追加し、対象 user の TOTP factor / WebAuthn credential /
recovery code から選んだ種別だけを削除する。管理者は代わりの factor を登録できない。
- 削除後は既存 `SyncMfaEnrolled` で `mfa_enrolled` を再計算し、TOTP と WebAuthn が両方
無くなった場合だけ既存の `IssueMfaEnrollmentBypass` (wi-245) をそのまま呼んで単発
enrollment bypass を自動発行する。新しい状態機械や pending 概念は追加せず、既存の
fail-closed な `EvaluateMfaEnrollment` / `beginMfaEnrollment` gate がそのまま次回ログインを
Enrollment pending へ導く。一部 factor のみ削除した場合は bypass を発行せず、残存要素で
通常ログインを継続できる。
- 新規監査イベント `AuthenticatorResetRequested` / `AuthenticatorResetCompleted` を追加し、
再登録要求そのものは既存 `MfaEnrollmentBypassIssued` を再利用した (二重イベント化を避けた)。
- 管理 UI (`AdminUserDetailPage.tsx` / `AdminUserDialogs.tsx`) に「認証器をリセット」操作と
対象選択・警告文を備えた確認ダイアログを追加した。
- 永続化層の追加は不要だった (TOTP/WebAuthn/recovery code の `DeleteAllForSub` と
`mfa_enrollment_bypasses` テーブルを既存のまま再利用)。

全文は `git show 61f5d0fe18c17eadc4a23f1b9048389f098c55ca:<このファイルのパス>` で読む。
