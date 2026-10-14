---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# authentication context の `WriteBrowserError` 呼び出しを Problem Details へ移行する

`backend/authentication` 配下 14 ファイル・78 箇所の `WriteBrowserError` を
すべて `support.WriteProblem` へ置き換えた。起票時の一覧は 11 ファイルだったが、
その後追加された `admin_auth_event_bucket_handler.go`・
`account_notification_preferences_handler.go`・
`account_trusted_devices_handler.go` も同一コンテキストなので含めた。
status を仕様の宣言値 422 へ揃えたのは 4 箇所:
`password_reuse` (`change_password_handler.go`・`password_reset_handler.go`)、
`mfa_enrollment_not_allowed` (`admin_mfa_enrollment_handler.go`)、
`authenticator_reset_not_allowed` (`admin_reset_handler.go`)。
それ以外の call site は status を変えず envelope だけ変えた。
管理者の MFA 操作 2 件 (対象なしの authenticator reset、範囲外 TTL の
enrollment bypass) は既存テストが status を固定していなかったため、
`admin_authenticator_reset_e2e_test.go` に実 HTTP スタックを通す
`TestAdminMfaOperationsRejectDisallowedRequests` を新設して 422 と
Problem Details を固定した。
仕様変更はない (`just spec-diff`: no normative specification change)。
**対応していないこと**:
- `oauth2` context の `authorize_enrollment.go:172` にある同名 code
`mfa_enrollment_not_allowed` (現状 403) は `wi-332` の scope。
- step-up gate の対象表テスト (`account_step_up_handler_test.go` の
`errorCode`) は、まだ移行していないコンテキスト (email 変更 =
`idmanagement`) のエンドポイントも含むため、Problem Details と
`{error, message}` の両方から code を読む形にした。`wi-340` で
`{error, message}` 側の分岐を落とす。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
