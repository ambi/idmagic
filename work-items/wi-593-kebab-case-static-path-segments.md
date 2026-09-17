---
status: pending
authors: [tn]
risk: medium
reversibility: irreversible
created_at: 2026-09-18
priority: p2
depends_on: [wi-585-rewrite-api-rules-as-complete-api-guidelines]
change_kind: feature
affected_spec:
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

[API ガイドライン](../docs/design/application/api-guidelines.md)の「静的パスセグメントの記法」は、パスパラメーター以外のパスセグメントをケバブケースで記述すると定める。
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

- [ ] T001 [Decision] リリースベースラインの配布状況を確認する。
- [ ] T002 [Spec] TypeSpec のルートを改める。
- [ ] T003 [App] Go のルート登録と経路表を改める。
- [ ] T004 [UI] フロントエンドの API クライアントを改める。
- [ ] T005 [Docs] API ガイドラインの適用状況を改める。
- [ ] T006 [Verify] 変更を検証する。

## Verification

- `mise run check-generated-contract`
- `mise run check-route-reference`
- `mise run check-api-compat`
- `mise run verify`
- `mise run test-ui-e2e`

## Risk Notes

ルート登録と経路表の片方だけを改めると、アドミッションコントロールが経路を分類できず、既定の優先度で扱う。`mise run check-route-reference` で両者の一致を確かめる。
