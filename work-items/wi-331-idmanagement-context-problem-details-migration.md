---
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-08-08
depends_on: [wi-326-http-error-responses-rfc9457-migration]
---

# idmanagement context の `WriteBrowserError` 呼び出しを Problem Details へ移行する

## Motivation

`wi-326` が確立した RFC 9457 Problem Details 移行パターン (ADR-154、
`backend/shared/http/support_http/problem.go`、参照実装
`backend/apitoken/handlers_http/routes.go`) を `idmanagement` context に
適用する。`wi-325` で `identity-management.yaml` に新設した 17 個の granular
error model に対応する Go 呼び出し箇所も、この移行と同時に status を揃える
(このコンテキストが 37 個中 17 個と最多)。

## Scope

- `backend/idmanagement/user/handlers_http/admin_user_handler.go` (15 箇所)
- `backend/idmanagement/handlers_http/admin_data_export_handler.go` (13 箇所)
- `backend/idmanagement/agent/handlers_http/admin_agent_handler.go` (12 箇所)
- `backend/idmanagement/group/handlers_http/admin_group_handler.go` (11 箇所)
- `backend/idmanagement/user/handlers_http/email_change_handler.go` (7 箇所)
- `backend/idmanagement/user/handlers_http/account_handler.go` (7 箇所)
- `backend/idmanagement/user/handlers_http/admin_user_import_handler.go` (5 箇所)

現状 400 (`http.StatusBadRequest`) で SCL は 422 (`http.StatusUnprocessableEntity`)
を宣言している code (`InvalidRequestError` からの分離元、実装時に status も
変更する):
- `admin_user_handler.go`: `invalid_role`、`self_delete_forbidden`、
  `self_disable_forbidden`、`invalid_attribute`、`invalid_required_action`。
- `admin_group_handler.go`: `invalid_role`、`group_name_required`、
  `invalid_dynamic_group_rule`。
- `admin_agent_handler.go`: `invalid_role`、`agent_name_required`、
  `agent_owner_required`、`agent_owner_not_found`。
- `email_change_handler.go`: `invalid_email`、`email_unchanged`。
- `admin_data_export_handler.go`: `invalid_columns` (→`InvalidExportColumnsError`)、
  `invalid_target` (→`InvalidExportTargetError`)、
  `invalid_filter` (→`InvalidExportFilterError`)。

現状 400 で SCL は **409** (状態競合) を宣言している code
(`admin_user_handler.go`):
- `not_pending_deletion` (→ `UserNotPendingDeletionError`)。
- `restore_grace_expired` (→ `RestoreGracePeriodExpiredError`)。

上記以外の code (`user_not_found` 404、`username_conflict` 409、
`jobs_unavailable` 503、`data_export_not_found` 404 等) は既存 status のまま
envelope だけ変える。

## Out of Scope

- `WriteBrowserError` 自体の削除 (全コンテキスト移行完了後の別 work item)。

## Design

- `support.WriteBrowserError(...)` を `support.WriteProblem(...)` に機械的に
  置き換える。
- 上記 status 変更対象は置換と同時に該当 `http.Status*` 定数を変更する。

## Plan

1. 7 ファイルの `WriteBrowserError` 呼び出しを列挙する。
2. 既存ハンドラテストのうち body 形式・status に依存するものを Problem
   Details/新 status 前提へ更新し RED を確認する。
3. `WriteBrowserError` → `WriteProblem` に置換し、上記 status を揃える。
4. `just verify` を通す。

## Tasks

- [ ] T001 [App] `admin_user_handler.go` を移行し、`invalid_role`・
      `self_delete_forbidden`・`self_disable_forbidden`・`invalid_attribute`・
      `invalid_required_action` を 422、`not_pending_deletion`・
      `restore_grace_expired` を 409 に揃える。
- [ ] T002 [App] `admin_group_handler.go` を移行し、`invalid_role`・
      `group_name_required`・`invalid_dynamic_group_rule` を 422 に揃える。
- [ ] T003 [App] `admin_agent_handler.go` を移行し、`invalid_role`・
      `agent_name_required`・`agent_owner_required`・`agent_owner_not_found` を
      422 に揃える。
- [ ] T004 [App] `email_change_handler.go` を移行し、`invalid_email`・
      `email_unchanged` を 422 に揃える。
- [ ] T005 [App] `admin_data_export_handler.go` を移行し、`invalid_columns`・
      `invalid_target`・`invalid_filter` を 422 に揃える。
- [ ] T006 [App] `account_handler.go`・`admin_user_import_handler.go` を
      `WriteProblem` へ移行する (status 変更なし)。
- [ ] T007 [Verify] `just verify` を通す。

## Verification

- `just verify-go`
- 手動: `invalid_role` エラーが `application/problem+json`・`status: 422` で
  返ることを確認する。

## Risk Notes

`invalid_role` は `admin_user_handler.go`・`admin_group_handler.go`・
`admin_agent_handler.go` の 3 ファイルすべてに出現する同名 code。SCL 側は
`identity-management.yaml` 内の単一 `InvalidRoleError` model を共有するため、
3 箇所とも同じ 422 に揃える。`wi-325` Design 節の対応表と突き合わせて
取り違えを防ぐこと (wi-326 Risk Notes 参照)。
