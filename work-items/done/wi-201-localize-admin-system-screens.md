---
status: completed
authors: ["tn"]
risk: high
created_at: 2026-07-12
depends_on: [wi-158-i18n-ja-en, wi-197-backend-api-errors-english-only]
---

# Admin と System 画面の ja/en 辞書化

## Motivation

管理コンソールとシステムコンソールは画面数・dialog・空状態が多く、英語選択で文言が混在する。

## Scope

- `frontend/src/features/admin-*` と `features/system-tenants` の画面、dialog、aria label、状態、日時・数値を feature-local ja/en 辞書へ移す。
- 英語既定の component test と日本語明示の i18n test を追加する。

## Out of Scope

- hosted auth と account の画面移行。

## Plan

- admin feature ごとに辞書を置き、shared shell と domain labels は既存の共通辞書を利用する。

## Tasks

- [x] T000 [SCL] `spec/contexts/system.yaml` の UX-LOCALE screens に不足していた AdminRoles / AdminAuditEvents / AdminSignInPolicy / SystemKeyHealth を追加する。
- [x] T001 [UI] admin と system の全画面を辞書化する。
  - [x] admin-dashboard/AdminDashboardPage
  - [x] admin-users/AdminUsersPage (+ AdminUsersPrimitives)
  - [x] admin-groups/AdminGroupsPage
  - [x] admin-agents/AdminAgentsPage
  - [x] admin-roles/AdminRolesPage
  - [x] admin-applications/AdminApplicationsPage
  - [x] admin-authz-detail-types/AdminAuthorizationDetailTypesPage
  - [x] admin-sign-in-policy/AdminSignInPolicyPage
  - [x] admin-consents/AdminConsentsPage
  - [x] admin-audit-events/AdminAuditEventsPage
  - [x] admin-keys/AdminKeysPage
  - [x] admin-keys/SystemKeyHealthPage
  - [x] admin-tenants/AdminTenantAttributesPage
  - [x] admin-settings/AdminSettingsPage (+ BrandingTab)
  - [x] admin-entra-federation/AdminEntraFederationPage
  - [x] system-tenants/SystemTenantsPage
- [x] T002 [Test] locale 別描画、空状態、dialog、日時・数値書式を検証する。
- [x] T003 [Verify] `just verify-ui` と `just test-ui-e2e` を通す。

## Verification

- `just verify-ui`
- `just test-ui-e2e`

## Risk Notes

多数の管理操作でテキスト locator が使われているため、E2E を locale 非依存の role / state locator へ移す。

## Completion

- **Completed At**: 2026-07-12
- **Summary**:
  Admin と System の各 feature に ja/en 辞書を追加し、画面、dialog、空状態、aria label、日時・数値表示を locale に従って表示するようにした。
  E2E 操作シナリオを英語既定の表示に合わせて更新し、ローカライズ後も主要管理操作を確認できるようにした。
- **Affected Guarantees State**:
  Admin と System の対象画面は英語を既定として表示し、明示的な日本語選択時には feature-local 辞書から日本語を表示する。
- **Verification Results**:
  - `just yaml-check` — passed
  - `just scl-render` — passed
  - `just verify-ui` — passed
  - `just test-ui-e2e` — passed
- **Evidence**:
  - 実行日: 2026-07-12
  - 実行環境: ローカル開発環境 (macOS)
  - 実行主体: Codex (GPT-5)
  - 対象ソース版: main (コミット前作業ツリー)
  - 保存先: 外部成果物なし。UI unit / E2E テストと SCL 派生物で確認。
