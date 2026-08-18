---
status: completed
authors: ["tn"]
risk: high
created_at: 2026-08-08
depends_on: [wi-326-http-error-responses-rfc9457-migration]
initial_context:
  specification: [spec/SPECIFICATION.md, spec/contexts/oauth2/SPECIFICATION.md]
  source: [backend/oauth2/handlers_http, backend/shared/http/support_http/problem.go]
  tests: [backend/oauth2/handlers_http, backend/shared/http/server_http]
  stop_before_reading: [frontend]
---

# oauth2 context の generic (非 RFC 6749) ハンドラの `WriteBrowserError` 呼び出しを Problem Details へ移行する

## Motivation

`wi-326` が確立した RFC 9457 Problem Details 移行パターン (ADR-154、
`backend/shared/http/support_http/problem.go`、参照実装
`backend/apitoken/handlers_http/routes.go`) を `oauth2` context に適用する。
ただし `oauth2` context は RFC 6749 §5.2 の `{error, error_description}` 形式を
維持する protocol endpoint (`error_format: oauth2`、`wi-325` が 12 interface に
設定済み) と、ブラウザ向けの一般 API (ログイン画面・同意画面・MFA step-up 等の
JSON API) が同じ `backend/oauth2/handlers_http` パッケージに混在するため、
`WriteBrowserError` 呼び出し 1 つずつについてどちらに属すかを判定してから
移行する必要がある (他コンテキストより判定コストが高いため risk: high)。

## Scope

- `backend/oauth2/handlers_http/authorize_second_factor.go` (27 箇所)
- `backend/oauth2/handlers_http/authorize_login.go` (12 箇所)
- `backend/oauth2/handlers_http/admin_client_handler.go` (8 箇所)
- `backend/oauth2/handlers_http/admin_authorization_detail_type_handler.go` (8 箇所)
- `backend/oauth2/handlers_http/admin_mcp_resource_server_handler.go` (7 箇所)
- `backend/oauth2/handlers_http/authorize_enrollment.go` (6 箇所)
- `backend/oauth2/handlers_http/authorize_transaction.go` (4 箇所)
- `backend/oauth2/handlers_http/authorize_resume.go` (4 箇所)
- `backend/oauth2/handlers_http/device_handler.go` (3 箇所、`handleDeviceAPI` のみ。
  `handleDeviceAuthorization`/`handleDeviceContext` は RFC 8628 protocol
  endpoint で対象外)
- `backend/oauth2/handlers_http/authorize_consent.go` (3 箇所)

`admin_authorization_detail_type_handler.go` の `invalid_type` (現状 400、
`spec/contexts/oauth2/SPECIFICATION.md` の `InvalidAuthorizationDetailTypeError` は 422) は
envelope 移行と同時に status も 422 へ揃える。

`authorize_enrollment.go:170` の `mfa_enrollment_not_allowed` (現状 **403**) は
`authentication.yaml` の `MfaEnrollmentNotAllowedError` (422) の published
language stub。`authentication` context 側 (`wi-329`) の同名コードは現状 400 で
別箇所だが、specification 上は同一 model を指す。ここも 403→422 に揃える。

## Out of Scope

- RFC 6749/7591 protocol endpoint (`Authorize`、`Token`、`Introspect`、
  `Revoke`、`UserInfo`、`PostUserInfo`、`DeviceAuthorization` 等、`wi-325` が
  `error_format: oauth2` を設定した 12 interface) — 既存の `writeOAuthError`/
  `OAuthErrorBody` のまま維持し Problem Details 化しない (ADR-154)。
- `authorize_second_factor.go`/`authorize_login.go`/`authorize_consent.go`/
  `authorize_transaction.go`/`authorize_resume.go` 各ファイル内で
  `authorizationErrorURL`/`redirectAuthorizationError` (OAuth2 プロトコル上の
  redirect 経由エラー、RFC 6749 §4.1.2.1) を使っている箇所は対象外。
  **`WriteBrowserError` を直接呼んでいる箇所だけが対象** — 同じファイル内に
  両方のパターンが混在するため、置換前に呼び出し木を確認すること。
- `WriteBrowserError` 自体の削除 (全コンテキスト移行完了後の別 work item)。

## Design

- 各 `WriteBrowserError` 呼び出しについて、それが JSON API (ログイン/MFA/同意
  画面が fetch する API) か OAuth2 protocol endpoint の一部かを、呼び出し元
  handler が bind される interface (`spec/contexts/oauth2/SPECIFICATION.md` の
  `bindings.path`) と `wi-325` Design 節の 12 interface リストを突き合わせて
  判定する。`device_handler.go` はファイル内に両方の handler
  (`handleDeviceAuthorization` は protocol、`handleDeviceAPI` はブラウザ向け)
  が同居する実例として確認済み。
- 対象と判定した箇所は `support.WriteProblem(...)` に置き換える。

## Plan

1. **T000 を最初に実施**: 10 ファイルの `WriteBrowserError` 呼び出しをすべて
   列挙し、それぞれ「JSON API」か「protocol endpoint の一部」かを分類した
   一覧を Tasks か Risk Notes に残す (判定結果を証跡化する)。
2. 既存ハンドラテストのうち body 形式に依存するものを Problem Details 形式へ
   更新し RED を確認する。
3. 「JSON API」に分類した箇所だけ `WriteProblem` へ置換する。
   `invalid_type`/`mfa_enrollment_not_allowed` は status も揃える。
4. `just verify` を通す。

## Tasks

- [x] T000 [Research] 分類結果: `WriteBrowserError` を直接呼んでいる箇所は
      すべてブラウザ向け JSON API 側だった。protocol endpoint
      (`/token`・`/introspect`・`/revoke`・`/userinfo`・`/register`・`/par`・
      `/device_authorization`・`/authorize` の redirect 経路・CIBA の
      `handleBackchannelAuthenticate`) は `writeOAuthError`/
      `redirectAuthorizationError` を使っており、`WriteBrowserError` は
      1 箇所も呼んでいない。`device_handler.go` は
      `handleDeviceAuthorization`/`handleDeviceContext` に呼び出しがなく、
      3 箇所とも `handleDeviceAPI` (`POST /api/auth/device`) 内であることを
      関数境界の行番号で確認した。
- [x] T001 [App] `authorize_second_factor.go`・`authorize_login.go`・
      `authorize_consent.go`・`authorize_transaction.go`・`authorize_resume.go`・
      `authorize_enrollment.go` の JSON API 箇所を `WriteProblem` へ移行する。
      `mfa_enrollment_not_allowed` の status は据え置いた (下記 Completion 参照)。
      RED→GREEN: `TestDisabledUserCannotLogIn`・
      `TestDisabledUserLoginAndExistingSessionAreRejected`。
- [x] T002 [App] `admin_client_handler.go`・
      `admin_authorization_detail_type_handler.go`・
      `admin_mcp_resource_server_handler.go` を `WriteProblem` へ移行する
      (`invalid_type` は 422 に揃える)。起票後に増えていた
      `approval_handler.go` (account 向け承認 API) と `admin_consent_handler.go`
      も同じ分類基準で移行した。
      RED→GREEN: `TestAdminRejectsInvalidTypeSchema`。
- [x] T003 [App] `device_handler.go` の `handleDeviceAPI` だけを
      `WriteProblem` へ移行する。
- [x] T004 [Verify] `just verify` を通す。

## Verification

- `just verify-go`
- 回帰確認: `/token`、`/authorize` (redirect エラー経路)、`/userinfo`、
  `/device_authorization` など protocol endpoint のレスポンス形式が
  変化していないことを curl で確認する。

## Risk Notes

このコンテキストは唯一 `error_format: oauth2` の protocol endpoint と一般 API が
同一パッケージに混在する。誤って protocol endpoint 側を Problem Details 化すると
標準クライアントライブラリの `{error, error_description}` 解析が壊れる
(ADR-154 却下代替案 参照)。T000 の分類作業を省略しないこと。

## Completion

- **Completed At**: 2026-08-19
- **Summary**:
  `backend/oauth2/handlers_http` の 12 ファイル・89 箇所の `WriteBrowserError`
  を `support.WriteProblem` へ置き換えた。T000 の分類の結果、
  `WriteBrowserError` の呼び出しはすべてブラウザ向け JSON API 側にあり、
  RFC 6749/7591/8628 の protocol endpoint は `writeOAuthError`/
  `redirectAuthorizationError` しか使っていなかったため、
  protocol 応答の形式 (`{error, error_description}`) は 1 箇所も変えていない。
  起票時の一覧になかった `approval_handler.go` (account の承認 API 7 箇所) と
  `admin_consent_handler.go` (1 箇所) も同じ基準でブラウザ API と判定して
  移行した。
  status を仕様の宣言値へ揃えたのは `admin_authorization_detail_type_handler.go`
  の `invalid_type` 2 箇所 (400 → 422、`InvalidAuthorizationDetailTypeError` は
  create/update/replace 3 operation の 422 union の要素)。
  仕様変更はない (`just spec-diff`: no normative specification change)。

  **起票時の指示から外れた判断**:
  - `authorize_enrollment.go` の `mfa_enrollment_not_allowed` は **403 のまま**
    にした。起票時は「`authentication.yaml` の 422 model を published language
    stub として再公開しているので同一 model」という前提だったが、現在の
    TypeSpec で `MfaEnrollmentNotAllowedError` (422) を宣言しているのは
    `IssueMfaEnrollmentBypass` (`spec/contexts/authentication/main.tsp`、
    管理者が bypass を発行する別 operation、`wi-329` で実装を 422 へ揃え済み)
    だけである。ここで 403 を返す `POST /api/auth/mfa/enrollment/totp/*` は
    `StartBrowserMfaEnrollment`/`ConfirmBrowserMfaEnrollment`
    (`spec/contexts/authentication/main.tsp`) として宣言されてはいるが、その
    error union は 400 が `InvalidRequestError`、403 が `AccessDeniedError` で、
    `MfaEnrollmentNotAllowedError` も 422 も含まない。つまりこの operation には
    揃える先の宣言値が存在しない。さらにこの分岐は `ErrMfaAlreadyEnrolled`
    (管理 API 側では 409) も同じ code・同じ status で扱っているため、宣言の
    ないまま 422 にすると別概念を 1 つの status に押し込むことになる。
    契約の欠落を埋めるのが先で、それは `wi-382` (TypeSpec contract fidelity)
    の T009 に引き継いだ。
- **Verification Results**:
  - `just test-go-package ./backend/oauth2/handlers_http` - RED
    (`TestAdminRejectsInvalidTypeSchema` が status 400 で失敗) → GREEN
  - `just test-go` - passed (全パッケージ。protocol endpoint のテスト
    (`token_handler_test.go`・`register_handler_test.go`・`par_handler_test.go`
    など `{"error": ...}` を照合するもの) は 1 つも変更していない)
  - `just verify` - passed
  - `just spec-diff` - no normative specification change against main
