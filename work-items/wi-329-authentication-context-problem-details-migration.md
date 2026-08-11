---
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-08-08
depends_on: [wi-326-http-error-responses-rfc9457-migration]
---

# authentication context の `WriteBrowserError` 呼び出しを Problem Details へ移行する

## Motivation

`wi-326` が確立した RFC 9457 Problem Details 移行パターン (ADR-154、
`backend/shared/http/support_http/problem.go`、参照実装
`backend/apitoken/handlers_http/routes.go`) を `authentication` context に
適用する。`wi-325` で `authentication.yaml` に新設した 3 個の granular error
model (`PasswordReuseError`、`MfaEnrollmentNotAllowedError`、
`AuthenticatorResetNotAllowedError`、いずれも status: 422) に対応する Go 呼び出し
箇所も、この移行と同時に status を揃える。

## Scope

- `backend/authentication/federation/handlers_http/routes.go` (20 箇所)
- `backend/authentication/deps_http/account_helpers.go` (14 箇所)
- `backend/authentication/password/handlers_http/change_password_handler.go` (7 箇所)
- `backend/authentication/password/handlers_http/password_reset_handler.go` (5 箇所)
- `backend/authentication/mfa/handlers_http/account_step_up_handler.go` (4 箇所)
- `backend/authentication/mfa/handlers_http/admin_mfa_enrollment_handler.go` (3 箇所)
- `backend/authentication/webauthn/handlers_http/account_webauthn_handler.go` (3 箇所)
- `backend/authentication/recovery/handlers_http/recovery_codes_handler.go` (2 箇所)
- `backend/authentication/mfa/handlers_http/admin_reset_handler.go` (2 箇所)
- `backend/authentication/mfa/handlers_http/account_totp_enrollment_handler.go` (2 箇所)
- `backend/authentication/handlers_http/account_context_handler.go` (1 箇所)

status 実装済み確認 (現状 → specification 宣言値 422 への実装時変更対象):
- `change_password_handler.go:67`、`password_reset_handler.go:117` の
  `"password_reuse"` (現状 400)。
- `admin_mfa_enrollment_handler.go:72` の `"mfa_enrollment_not_allowed"` (現状 400)。
  **`backend/oauth2/handlers_http/authorize_enrollment.go:170` にも同名コードの
  別呼び出しがあるが、それは現状 403 で状態も異なり、`oauth2` context の
  work item 側で扱う** (published language stub として `authentication.yaml`
  の model を再公開しているため名前は同じでも呼び出し箇所は別コンテキスト)。
- `admin_reset_handler.go:54` の `"authenticator_reset_not_allowed"` (現状 400)。

## Out of Scope

- OAuth2 (`backend/oauth2/handlers_http/authorize_enrollment.go` の
  `mfa_enrollment_not_allowed`) は `oauth2` context の work item側の scope。
- `WriteBrowserError` 自体の削除 (全コンテキスト移行完了後の別 work item)。

## Design

- `support.WriteBrowserError(...)` を `support.WriteProblem(...)` に機械的に
  置き換える。
- 上記 3 箇所は置換と同時に `http.StatusBadRequest` を
  `http.StatusUnprocessableEntity` へ変更する。

## Plan

1. 11 ファイルの `WriteBrowserError` 呼び出しを列挙する。
2. 既存ハンドラテストのうち body 形式に依存するものを Problem Details 形式へ
   更新し RED を確認する (`password_reuse`/`mfa_enrollment_not_allowed`/
   `authenticator_reset_not_allowed` は status 変更も RED に含める)。
3. `WriteBrowserError` → `WriteProblem` に置換し、上記 3 箇所は status も揃える。
4. `just verify` を通す。

## Tasks

- [ ] T001 [App] `federation/handlers_http/routes.go`・`deps_http/account_helpers.go`・
      `webauthn/handlers_http/account_webauthn_handler.go`・
      `recovery/handlers_http/recovery_codes_handler.go`・
      `handlers_http/account_context_handler.go` を `WriteProblem` へ移行する。
- [ ] T002 [App] `password/handlers_http/change_password_handler.go`・
      `password/handlers_http/password_reset_handler.go` を移行し、
      `password_reuse` の status を 422 に揃える。
- [ ] T003 [App] `mfa/handlers_http/*` (`account_step_up_handler.go`、
      `admin_mfa_enrollment_handler.go`、`admin_reset_handler.go`、
      `account_totp_enrollment_handler.go`) を移行し、
      `mfa_enrollment_not_allowed`/`authenticator_reset_not_allowed` の status を
      422 に揃える。
- [ ] T004 [Verify] `just verify` を通す。

## Verification

- `just verify-go`
- 手動: パスワード再利用エラーが `application/problem+json`・`status: 422` で
  返ることを確認する。

## Risk Notes

`federation/handlers_http/routes.go` (20 箇所) と `deps_http/account_helpers.go`
(14 箇所) は federation 系の複雑な分岐を持つため、置換漏れがないか
`grep -c WriteBrowserError` で件数を突き合わせて確認すること。
