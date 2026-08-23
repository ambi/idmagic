---
status: completed
authors: ["tn"]
risk: low
created_at: 2026-08-08
depends_on: [wi-326-http-error-responses-rfc9457-migration]
initial_context:
  specification: [docs/SPECIFICATION.md, docs/contexts/application/SPECIFICATION.md]
  typespec: [IdMagic.Contract.CategoryNameRequiredError, IdMagic.Contract.UnknownCategoryError, IdMagic.Contract.ClientSecretLimitExceededError]
  source: [backend/application/handlers_http, backend/shared/http/support_http/problem.go]
  tests: [backend/application/handlers_http]
  stop_before_reading: [frontend]
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
(`docs/contexts/application/SPECIFICATION.md` の `CategoryNameRequiredError`/
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

- [x] T001 [App] `admin_application_handler.go`・`application_provisioning.go`・
      `client_secret_lifecycle.go`・`account_application_handler.go` を
      `WriteProblem` へ移行する。あわせて `client_secret_limit_exceeded` の
      status を宣言外の 409 から `IssueApplicationClientSecretError422` の
      422 へ揃えた。RED→GREEN: `TestAdminApplicationClientSecretLifecycle`。
- [x] T002 [App] `admin_category_handler.go` を `WriteProblem` へ移行し、
      `category_name_required`/`unknown_category` の status を 422 に揃える。
      RED→GREEN: `TestCreateApplicationCategory_EmptyNameYieldsDistinguishableCode`・
      `TestSetApplicationCategories_UnknownCategoryYieldsDistinguishableCode`。
- [x] T003 [Verify] `just verify` を通す。

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

## Completion

- **Completed At**: 2026-08-19
- **Summary**:
  `backend/application/handlers_http` の 5 ファイル・54 箇所の
  `WriteBrowserError` をすべて `support.WriteProblem` へ置き換え、application
  context の汎用 API エラーが `{error, message}` ではなく RFC 9457 Problem
  Details (`application/problem+json`、`type`/`title`/`status`/`detail`/`instance`)
  を返すようになった。
  status を仕様の宣言値へ揃えたのは 3 箇所:
  `category_name_required`・`unknown_category` (400 → 422、
  `CreateApplicationCategoryError422`/`UpdateApplicationCategoryError422`/
  `SetApplicationCategoriesError422`)、および実装時に判明した
  `client_secret_limit_exceeded` (409 → 422)。後者は work item 起票時の
  一覧になかったが、`IssueApplicationClientSecret` が宣言するエラー status は
  400/403/422 のみで 409 は宣言外であり、`ClientSecretLimitExceededError` は
  422 union の要素だったため、envelope 移行と同時に揃えた。
  それ以外の call site (`invalid_request` 400、`not_found`/`application_not_found`/
  `category_not_found` 404、`authentication_required` 401、`invalid_icon`/
  `invalid_sign_in_policy` 400) は status を変えず envelope だけ変えた。
  仕様変更はない (`just spec-diff`: no normative specification change) —
  TypeSpec 側は wi-325 時点で宣言済みで、この work item は実装をその宣言に
  追従させただけである。

  **対応していないこと**:
  - `WriteBrowserError` 自体の削除は `wi-340`。
  - このコンテキストに OAuth2/SCIM/DCR、SharedSignals inbound SET receiver の
    call site はないため、protocol 応答形式は一切触れていない。
- **Verification Results**:
  - `just test-go-package ./backend/application/handlers_http` - RED
    (`TestCreateApplicationCategory_EmptyNameYieldsDistinguishableCode`・
    `TestSetApplicationCategories_UnknownCategoryYieldsDistinguishableCode`・
    `TestAdminApplicationClientSecretLifecycle` の 3 件が status 400/409 で失敗)
    → GREEN
  - `just verify` - passed (check / check-api-compat / test-tools /
    typecheck-tools / lint-go / test-go / format-check-ui / lint-ui /
    test-ui-unit / build-ui すべて green)
  - `just spec-diff` - no normative specification change against main
