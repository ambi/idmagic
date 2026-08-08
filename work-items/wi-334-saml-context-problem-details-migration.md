---
status: pending
authors: ["tn"]
risk: low
created_at: 2026-08-08
depends_on: [wi-326-http-error-responses-rfc9457-migration]
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

- [ ] T001 [App] `admin_idp_profile_handler.go`・
      `admin_service_provider_handler.go` を `WriteProblem` へ移行する。
- [ ] T002 [Verify] `just verify` を通す。

## Verification

- `just verify-go`

## Risk Notes

小規模 (2 ファイル・13 箇所)。特記事項なし。
