---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# oauth2 context の generic (非 RFC 6749) ハンドラの `WriteBrowserError` 呼び出しを Problem Details へ移行する

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

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
