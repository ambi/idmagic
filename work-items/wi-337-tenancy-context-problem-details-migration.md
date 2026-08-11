---
status: pending
authors: ["tn"]
risk: low
created_at: 2026-08-08
depends_on: [wi-326-http-error-responses-rfc9457-migration]
---

# tenancy context の `WriteBrowserError` 呼び出しを Problem Details へ移行する

## Motivation

`wi-326` が確立した RFC 9457 Problem Details 移行パターン (ADR-154、
`backend/shared/http/support_http/problem.go`、参照実装
`backend/apitoken/handlers_http/routes.go`) を `tenancy` context に適用する。
`wi-325` で `tenancy.yaml` に新設した 2 個の granular error model
(`PolicyOverrideWeakerError`、`InvalidUserAttributeSchemaError`、いずれも
status: 422) に対応する Go 呼び出し箇所も、この移行と同時に status を揃える。

## Scope

- `backend/tenancy/handlers_http/admin_tenant_handler.go` (12 箇所)
- `backend/tenancy/handlers_http/admin_branding_handler.go` (9 箇所)
- `backend/tenancy/handlers_http/admin_notification_template_handler.go` (5 箇所)
- `backend/tenancy/handlers_http/branding_handler.go` (3 箇所)
- `backend/tenancy/handlers_http/admin_user_attribute_schema_handler.go` (3 箇所)
- `backend/tenancy/handlers_http/integration_endpoints_handler.go` (2 箇所)
- `backend/tenancy/handlers_http/admin_settings_handler.go` (2 箇所)

現状 400 で specification は 422 を宣言している code (実装時に status も変更する):
- `admin_tenant_handler.go` の `policy_override_weaker`
  (→ `PolicyOverrideWeakerError`)。
- `admin_user_attribute_schema_handler.go` の `invalid_attribute_schema`
  (→ `InvalidUserAttributeSchemaError`)。同ファイルの
  `attribute_referenced_by_dynamic_group` (409) は既存 status のまま。

## Out of Scope

- `WriteBrowserError` 自体の削除 (全コンテキスト移行完了後の別 work item)。

## Design

- `support.WriteBrowserError(...)` を `support.WriteProblem(...)` に機械的に
  置き換える。上記 2 箇所は置換と同時に status も 422 に変更する。

## Plan

1. 7 ファイルの `WriteBrowserError` 呼び出しを列挙する。
2. 既存ハンドラテストのうち body 形式・status に依存するものを Problem
   Details/新 status 前提へ更新し RED を確認する。
3. `WriteBrowserError` → `WriteProblem` に置換し、上記 2 箇所は status も揃える。
4. `just verify` を通す。

## Tasks

- [ ] T001 [App] `admin_tenant_handler.go` を移行し、
      `policy_override_weaker` を 422 に揃える。
- [ ] T002 [App] `admin_user_attribute_schema_handler.go` を移行し、
      `invalid_attribute_schema` を 422 に揃える。
- [ ] T003 [App] `admin_branding_handler.go`・
      `admin_notification_template_handler.go`・`branding_handler.go`・
      `integration_endpoints_handler.go`・`admin_settings_handler.go` を
      `WriteProblem` へ移行する (status 変更なし)。
- [ ] T004 [Verify] `just verify` を通す。

## Verification

- `just verify-go`

## Risk Notes

`admin_tenant_handler.go` (12 箇所) が最大。テナントデフォルトサインイン
ポリシー・quota 系の分岐が多いため、置換漏れがないか件数を突き合わせて
確認すること。
