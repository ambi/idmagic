---
status: pending
authors: ["tn"]
risk: low
created_at: 2026-08-08
depends_on: [wi-326-http-error-responses-rfc9457-migration]
---

# application context の `WriteBrowserError` 呼び出しを Problem Details へ移行する

## Motivation

`wi-326` が `backend/shared/http/support_http` に RFC 9457 Problem Details 共通
ヘルパー `WriteProblem` (`problem.go`) を実装し、`backend/apitoken/handlers_http`
で移行パターンを確立した (ADR-154)。`application` context の 5 ハンドラファイル・
53 箇所の `WriteBrowserError` 呼び出しが未移行のまま残っている。

## Scope

- `backend/application/handlers_http/admin_application_handler.go` (12 箇所)
- `backend/application/handlers_http/application_provisioning.go` (24 箇所)
- `backend/application/handlers_http/client_secret_lifecycle.go` (5 箇所)
- `backend/application/handlers_http/account_application_handler.go` (3 箇所)
- `backend/application/handlers_http/admin_category_handler.go` (9 箇所)

`admin_category_handler.go` の `category_name_required`/`unknown_category`
(`wi-326` T003 で新設した distinguishable code) はここで envelope を
Problem Details 化すると同時に status も specification 宣言値 422 へ揃える
(`spec/contexts/application/SPECIFICATION.md` の `CategoryNameRequiredError`/
`UnknownCategoryError`)。他の call site (`category_not_found` 404,
`application_not_found` 404 等) は既存 status のまま envelope だけ変える。

## Out of Scope

- OAuth2/SCIM/DCR、SharedSignals inbound SET receiver (このコンテキストには存在しない)。
- `WriteBrowserError` 自体の削除 (全コンテキスト移行完了後の別 work item)。

## Design

- `support.WriteProblem(c, status, code, detail)` へ機械的に置き換える
  (`backend/apitoken/handlers_http/routes.go` が参照実装)。
- 各 call site の既存 `code` 文字列 (`"invalid_request"` 等) は変えない
  (`category_name_required`/`unknown_category` を除く)。
- Adapters 層のため test-first (ADR-119): 既存ハンドラテストの `Content-Type`/
  body 構造アサーションを Problem Details 前提に更新する RED を先に書く。

## Plan

1. 5 ファイルの `WriteBrowserError` 呼び出しを列挙する。
2. 既存ハンドラテストのうち body 形式 (`{"error":..., "message":...}`) に
   依存するものを Problem Details 形式のアサーションへ更新し、RED を確認する。
3. `WriteBrowserError` → `WriteProblem` に置換する。
   `category_name_required`/`unknown_category` は status も 422 に変更する。
4. `just verify` を通す。

## Tasks

- [ ] T001 [App] `admin_application_handler.go`・`application_provisioning.go`・
      `client_secret_lifecycle.go`・`account_application_handler.go` を
      `WriteProblem` へ移行する。
- [ ] T002 [App] `admin_category_handler.go` を `WriteProblem` へ移行し、
      `category_name_required`/`unknown_category` の status を 422 に揃える。
- [ ] T003 [Verify] `just verify` を通す。

## Verification

- `just verify-go`
- 手動: `POST /api/admin/application-categories` に空 name を送り
  `application/problem+json`・`status: 422`・`type: urn:idmagic:error:category_name_required`
  を確認する。

## Risk Notes

`application_provisioning.go` は 24 箇所と最大。プロビジョニング関連の
外部 SCIM/OAuth2 呼び出しエラーと混同しないよう、`errors.go` 等の別ファイルに
定義された OAuth2/SCIM 用ヘルパーではなく `WriteBrowserError` を直接呼んでいる
箇所だけを対象にする。
