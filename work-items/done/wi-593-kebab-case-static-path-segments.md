---
status: completed
authors: [tn]
risk: medium
reversibility: irreversible
created_at: 2026-09-18
priority: p2
depends_on: [wi-585-rewrite-api-rules-as-complete-api-guidelines]
change_kind: feature
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: 汎用 API の静的パスセグメントをケバブケースへ統一する。契約は未配布だが、生成したクライアントを利用する開発者が確認できる変更記録を残す。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-593-kebab-case-static-path-segments.md }
initial_context:
  specification:
    - docs/design/application/api-guidelines.md
    - docs/domain/authentication/scenarios.feature.md#REQ-AUTHENTICATION-010
  typespec:
    - IdMagic.Authentication.Operations.ChangePassword
    - IdMagic.IdManagement.Operations.ClearUserRequiredAction
    - IdMagic.Authentication.Operations.CompleteStepUpAuthentication
    - IdMagic.IdGovernance.Operations.CreateLifecycleWorkflow
    - IdMagic.IdGovernance.Operations.DeleteLifecycleWorkflow
    - IdMagic.IdGovernance.Operations.DisableLifecycleWorkflow
    - IdMagic.IdGovernance.Operations.DryRunLifecycleWorkflow
    - IdMagic.IdGovernance.Operations.EnableLifecycleWorkflow
    - IdMagic.IdManagement.Operations.ExportAccountData
    - IdMagic.Audit.Operations.ExportAdminAuditEvents
    - IdMagic.Audit.Operations.ExportSystemAuditEvents
    - IdMagic.Audit.Operations.GetAdminAuditEvent
    - IdMagic.Audit.Operations.GetAdminAuditEventSearchOptions
    - IdMagic.IdManagement.Operations.GetEmailVerificationContext
    - IdMagic.IdGovernance.Operations.GetLifecycleWorkflow
    - IdMagic.IdGovernance.Operations.GetLifecycleWorkflowRun
    - IdMagic.Authentication.Operations.GetMyNotificationPreferences
    - IdMagic.Tenancy.Operations.GetNotificationTemplate
    - IdMagic.Authentication.Operations.GetPasswordResetContext
    - IdMagic.Audit.Operations.GetSystemAuditEvent
    - IdMagic.Tenancy.Operations.GetTenantGroupAttributeSchema
    - IdMagic.Tenancy.Operations.GetTenantUserAttributeSchema
    - IdMagic.Audit.Operations.ListAdminAuditEvents
    - IdMagic.Authentication.Operations.ListAuthenticationEventBuckets
    - IdMagic.IdGovernance.Operations.ListLifecycleWorkflowRuns
    - IdMagic.IdGovernance.Operations.ListLifecycleWorkflows
    - IdMagic.Authentication.Operations.ListMySignInActivity
    - IdMagic.Authentication.Operations.ListMyTrustedDevices
    - IdMagic.Tenancy.Operations.ListNotificationTemplates
    - IdMagic.Audit.Operations.ListSystemAuditEvents
    - IdMagic.Authentication.Operations.ListUserSignInActivity
    - IdMagic.Tenancy.Operations.PreviewNotificationTemplate
    - IdMagic.IdManagement.Operations.RequestEmailChange
    - IdMagic.Authentication.Operations.RequestPasswordReset
    - IdMagic.Tenancy.Operations.ResetNotificationTemplate
    - IdMagic.Authentication.Operations.ResetPasswordWithToken
    - IdMagic.IdGovernance.Operations.RetryLifecycleWorkflowRun
    - IdMagic.Authentication.Operations.RevokeMyOtherSessions
    - IdMagic.Authentication.Operations.RevokeMyTrustedDevice
    - IdMagic.Authentication.Operations.RevokeMyTrustedDevices
    - IdMagic.Authentication.Operations.RevokeUserSessions
    - IdMagic.Tenancy.Operations.SendTestNotification
    - IdMagic.Tenancy.Operations.SetTenantEndpointStyle
    - IdMagic.IdManagement.Operations.SetUserRequiredAction
    - IdMagic.Authentication.Operations.StartStepUpAuthentication
    - IdMagic.Authentication.Operations.StartStepUpWebAuthnChallenge
    - IdMagic.IdGovernance.Operations.UpdateLifecycleWorkflow
    - IdMagic.Authentication.Operations.UpdateMyNotificationPreferences
    - IdMagic.Tenancy.Operations.UpdateNotificationTemplate
    - IdMagic.Tenancy.Operations.UpdateTenantGroupAttributeSchema
    - IdMagic.Tenancy.Operations.UpdateTenantUserAttributeSchema
  source:
    - backend/authentication/handlers_http
    - backend/audit/handlers_http
    - backend/idmanagement/handlers_http
    - backend/idgovernance/handlers_http
    - backend/tenancy/handlers_http
    - backend/shared/http/server_http
    - frontend/src/api
  tests:
    - backend/shared/http/server_http/priority_class_test.go
    - backend/shared/spec/runtime_contract_test.go
    - frontend/src/api
  stop_before_reading: [spec/generated]
primary_use_cases:
  - id: change-password-at-kebab-case-path
    requirement: REQ-AUTHENTICATION-010
    observable_result: 認証済みのユーザーはケバブケースの `POST /api/auth/change-password` を通じて自身のパスワードを変更できる。
    unit_test: { path: backend/shared/spec/runtime_contract_test.go, name: TestOperationsUseKebabCaseStaticPathSegments_REQ_AUTHENTICATION_010, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/routes_e2e_test.go, name: TestChangePasswordUpdatesCredentialsAndRejectsReuse, task: test-go-race }
    unit_fault_model: 生成した操作メタデータにスネークケースの静的パスセグメントが残る。
    e2e_fault_model: HTTP ルート登録だけを旧パスのまま残し、新しいパスがハンドラーへ到達しない。
affected_spec:
  - { path: docs/domain/authentication/scenarios.feature.md, requirement: REQ-AUTHENTICATION-010 }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.ChangePassword }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.ClearUserRequiredAction }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.CompleteStepUpAuthentication }
  - { path: spec/contexts/identity-governance/main.tsp, symbol: IdMagic.IdGovernance.Operations.CreateLifecycleWorkflow }
  - { path: spec/contexts/identity-governance/main.tsp, symbol: IdMagic.IdGovernance.Operations.DeleteLifecycleWorkflow }
  - { path: spec/contexts/identity-governance/main.tsp, symbol: IdMagic.IdGovernance.Operations.DisableLifecycleWorkflow }
  - { path: spec/contexts/identity-governance/main.tsp, symbol: IdMagic.IdGovernance.Operations.DryRunLifecycleWorkflow }
  - { path: spec/contexts/identity-governance/main.tsp, symbol: IdMagic.IdGovernance.Operations.EnableLifecycleWorkflow }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.ExportAccountData }
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Audit.Operations.ExportAdminAuditEvents }
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Audit.Operations.ExportSystemAuditEvents }
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Audit.Operations.GetAdminAuditEvent }
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Audit.Operations.GetAdminAuditEventSearchOptions }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.GetEmailVerificationContext }
  - { path: spec/contexts/identity-governance/main.tsp, symbol: IdMagic.IdGovernance.Operations.GetLifecycleWorkflow }
  - { path: spec/contexts/identity-governance/main.tsp, symbol: IdMagic.IdGovernance.Operations.GetLifecycleWorkflowRun }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.GetMyNotificationPreferences }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.GetNotificationTemplate }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.GetPasswordResetContext }
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Audit.Operations.GetSystemAuditEvent }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.GetTenantGroupAttributeSchema }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.GetTenantUserAttributeSchema }
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Audit.Operations.ListAdminAuditEvents }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.ListAuthenticationEventBuckets }
  - { path: spec/contexts/identity-governance/main.tsp, symbol: IdMagic.IdGovernance.Operations.ListLifecycleWorkflowRuns }
  - { path: spec/contexts/identity-governance/main.tsp, symbol: IdMagic.IdGovernance.Operations.ListLifecycleWorkflows }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.ListMySignInActivity }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.ListMyTrustedDevices }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.ListNotificationTemplates }
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Audit.Operations.ListSystemAuditEvents }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.ListUserSignInActivity }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.PreviewNotificationTemplate }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.RequestEmailChange }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.RequestPasswordReset }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.ResetNotificationTemplate }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.ResetPasswordWithToken }
  - { path: spec/contexts/identity-governance/main.tsp, symbol: IdMagic.IdGovernance.Operations.RetryLifecycleWorkflowRun }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.RevokeMyOtherSessions }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.RevokeMyTrustedDevice }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.RevokeMyTrustedDevices }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.RevokeUserSessions }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.SendTestNotification }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.SetTenantEndpointStyle }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.SetUserRequiredAction }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.StartStepUpAuthentication }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.StartStepUpWebAuthnChallenge }
  - { path: spec/contexts/identity-governance/main.tsp, symbol: IdMagic.IdGovernance.Operations.UpdateLifecycleWorkflow }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.UpdateMyNotificationPreferences }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.UpdateNotificationTemplate }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.UpdateTenantGroupAttributeSchema }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.UpdateTenantUserAttributeSchema }
---

# 汎用 API の静的パスセグメントをケバブケースに統一する

## Motivation

[API ガイドライン](../../docs/design/application/api-guidelines.md)の「静的パスセグメントの記法」は、パスパラメーター以外のパスセグメントをケバブケースで記述すると定める。
現行の汎用 API では、24 個の静的パスセグメントがスネークケースであり、同じ API の中で `identity-providers` と `audit_events` が混在している。
クライアントはパスを類推で記述できず、セグメントごとに区切り文字を確認することになる。

## Scope

- 次の静的パスセグメントをケバブケースに改める。`audit_events`、`authentication_event_buckets`、`change_password`、`change_request`、`data_export`、`dry_run`、`endpoint_style`、`forgot_password`、`group_attribute_schema`、`lifecycle_workflow_runs`、`lifecycle_workflows`、`notification_preferences`、`notification_templates`、`password_reset_context`、`required_actions`、`reset_password`、`revoke_all`、`revoke_others`、`search_options`、`signin_activity`、`step_up`、`trusted_devices`、`user_attribute_schema`、`verify_context`。
- TypeSpec の `@route`、Go のルート登録、フロントエンドの API クライアント、アドミッションコントロールの経路表、テストを追従させる。
- API ガイドラインの適用状況を「全面適用」に改める。

## Out of Scope

- パスパラメーター、クエリパラメーター、JSON のプロパティの記法。スネークケースのまま維持する。
- プロトコルエンドポイントのパス。各標準が定める。

## Design

旧パスを残す互換経路は設けない。
二つのパスを並存させると、ルート登録、アドミッションコントロールの経路表、スコープ判定の対象が倍になり、どちらが正かを検査で区別できない。
プロダクトは未リリースであり、旧パスに依存する外部クライアントは存在しない。

採用しない案は、`/api/admin/v2/` を新設して移行する案である。
URI バージョニングは配布済みの契約を守るための仕組みであり、未配布の契約に適用すると、使われない `v1` を廃止まで保守することになる。

## Plan

1. リリースベースライン（`spec/idmagic.openapi.baseline.json`）が配布済みの内容を固定したものかを確認する。未配布であれば、この変更の一部としてベースラインを更新する。配布済みであれば、この work item を URI バージョニングの手順に改めてから着手する。
2. TypeSpec の `@route` を改め、`mise run check-generated-contract` を RED にする。
3. Go のルート登録、経路表、フロントエンドを追従させる。

## Tasks

- [x] T001 [Decision] リリースベースラインの配布状況を確認する。`9a1066dd` は既存クライアントが存在しないことを確認して再固定しており、今回も未配布契約として更新する。
- [x] T002 [Spec] TypeSpec のルートを改める。
- [x] T003 [App] Go のルート登録と経路表を改める。
- [x] T004 [UI] フロントエンドの API クライアントを改める。
- [x] T005 [Docs] API ガイドラインの適用状況を改める。
- [x] T006 [Verify] 変更を検証する。

## Verification

- `mise run check-generated-contract`
- `mise run check-route-reference`
- `mise run check-api-compat`
- `mise run verify`
- `mise run test-ui-e2e`

## Risk Notes

ルート登録と経路表の片方だけを改めると、アドミッションコントロールが経路を分類できず、既定の優先度で扱う。`mise run check-route-reference` で両者の一致を確かめる。

## Completion

- **Completed At**: 2026-09-21
- **Summary**:
  `mise run spec-diff` は規範シナリオの追加・変更なしを報告した。TypeSpec、生成済み操作契約、Go ルート、優先度表、フロントエンド API クライアントの 24 個の静的パスセグメントをケバブケースへ統一し、画面 URL、パスパラメーター、プロトコルエンドポイントは変更しなかった。未配布の OpenAPI ベースラインを新契約へ再固定した。
- **Primary Use Case Evidence**:
  - id: change-password-at-kebab-case-path
    unit_red: "TypeSpec 変更直後の `mise run check-generated-contract` は `operations_gen.go is stale` で失敗し、生成済み操作契約が旧パスのままでは仕様変更を受け入れないことを検出した。"
    e2e_red: "`TestChangePasswordUpdatesCredentialsAndRejectsReuse` を `/api/auth/change-password` へ向けると、旧ルート登録は 404 を返して失敗した。"
    unit_fault_injection: "`TestOperationsUseKebabCaseStaticPathSegments_REQ_AUTHENTICATION_010` は生成済み操作の静的セグメントへアンダースコアを戻す変異をエラーにする。"
    e2e_fault_injection: "ChangePassword のルートを旧パスに留めた状態で同 E2E テストは 404 を検出した。"
- **Change-Resistance Results**:
  `mise run test-go-mutation -- backend/shared/spec` は 96 変異を発見し、変更対象パッケージの基準カバレッジは 86.7% だった。生成済み操作の経路を検査する追加テストと、旧ルート登録を残す手動フォールト注入を併用した。
- **Verification Results**:
  - `mise run check-spec`、`mise run spec-render`、`mise run check-generated-contract`、`mise run check-contract-drift` - 成功
  - `mise run check-route-reference`、`mise run check-api-compat`、`mise run check-config-reference` - 成功
  - `mise run test-go-changed`、`mise run lint-go`、`mise run test-ui-unit`、`mise run typecheck-ui` - 成功
  - `mise run verify` - 成功
  - `mise run test-ui-e2e-file -- tests/e2e/ui-scenario-actions.spec.ts 'admin audit log can be filtered and export can be triggered'` - 成功（1 pass）
