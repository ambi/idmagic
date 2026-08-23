---
status: completed
authors: ["tn"]
risk: low
created_at: 2026-08-08
depends_on: [wi-326-http-error-responses-rfc9457-migration]
initial_context:
  specification: [docs/SPECIFICATION.md, docs/contexts/saml/SPECIFICATION.md]
  source: [backend/saml/handlers_http, backend/shared/http/support_http/problem.go]
  tests: [backend/saml/handlers_http]
  stop_before_reading: [frontend]
---

# saml context の `WriteBrowserError` 呼び出しを Problem Details へ移行する

## Motivation

`wi-326` が確立した RFC 9457 Problem Details 移行パターン (ADR-154、
`backend/shared/http/support_http/problem.go`、参照実装
`backend/apitoken/handlers_http/routes.go`) を `saml` context に適用する。

## Scope

- `backend/saml/handlers_http/admin_idp_profile_handler.go` (8 箇所)
- `backend/saml/handlers_http/admin_service_provider_handler.go` (5 箇所)

`admin_idp_profile_handler.go` の `profile_in_use`/`default_idp_profile` 等は
wi-325 が `IdPProfileInUseError`/`DefaultIdPProfileError` (いずれも 409) として
既に Go 実装と一致させて確認済み — status 変更は不要、envelope だけ変える。

## Out of Scope

- SAML プロトコル応答 (AuthnRequest/Response、metadata XML) 自体の形式変更
  (このコンテキストの HTTP エラー JSON API のみが対象)。
- `WriteBrowserError` 自体の削除 (全コンテキスト移行完了後の別 work item)。

## Design

- `support.WriteBrowserError(...)` を `support.WriteProblem(...)` に機械的に
  置き換える。

## Plan

1. 既存ハンドラテストのうち body 形式に依存するものを Problem Details 形式へ
   更新し RED を確認する。
2. `WriteBrowserError` → `WriteProblem` に置換する。
3. `just verify` を通す。

## Tasks

- [x] T001 [App] `admin_idp_profile_handler.go`・
      `admin_service_provider_handler.go` を `WriteProblem` へ移行する。
      RED→GREEN: `TestAdminServiceProvider_RejectsInvalid`。
- [x] T002 [Verify] `just verify` を通す。

## Verification

- `just verify-go`

## Risk Notes

小規模 (2 ファイル・13 箇所)。特記事項なし。

## Completion

- **Completed At**: 2026-08-19
- **Summary**:
  `backend/saml/handlers_http` の `admin_idp_profile_handler.go` (8 箇所)・
  `admin_service_provider_handler.go` (5 箇所) をすべて
  `support.WriteProblem` へ置き換えた。status 変更は不要
  (`profile_in_use`/`application_owned_protocol` は既に 409、`not_found` は 404)。
  長さ違反を大域のエラー写像に 422 として作らせている箇所 (wi-128) の
  コメントが旧ヘルパー名を指していたため現在の名前へ直した。
  SAML プロトコル応答 (AuthnRequest/Response、metadata XML) は対象外のまま
  変更していない。
  仕様変更はない (`just spec-diff`: no normative specification change)。
- **Verification Results**:
  - `just test-go-package ./backend/saml/handlers_http` - RED
    (`TestAdminServiceProvider_RejectsInvalid` が
    `Content-Type: application/json` で失敗) → GREEN
  - `just verify` - passed
  - `just spec-diff` - no normative specification change against main
