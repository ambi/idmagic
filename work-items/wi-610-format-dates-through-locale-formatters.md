---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-18
priority: p2
depends_on: [wi-586-frontend-architecture-and-user-interface-design]
change_kind: refactor
spec_impact:
  kind: none
  reason: "日時の整形を共通の関数へ寄せるだけで、表示する値の意味は変えない。"
---

# 画面の日時と数値の整形を `useFormatters` へ揃える

## Motivation

[ユーザーインターフェース設計](../docs/design/application/user-interface.md#国際化)は、日時と数値を `useFormatters` が返す関数で、現在のロケールに合わせて表示すると定める。
次のファイルは `toLocaleString`、`toLocaleDateString`、`new Intl.DateTimeFormat` を直接呼んでいる。

- `frontend/src/features/account/`：`AccountActivityPage.tsx`、`AccountApplicationsPage.tsx`、`AccountApprovalsPage.tsx`、`AccountHomePage.tsx`、`accountSecurityPresentation.ts`、`LinkedIdentitiesCard.tsx`
- `frontend/src/features/admin-applications/`：`AdminApplicationDetailPage.tsx`、`AdminApplicationProvisioningShared.tsx`、`AdminApplicationsListPage.tsx`、`ClientSecretRotationPanel.tsx`
- `frontend/src/features/admin-audit-events/AuditEventsBrowser.tsx`
- `frontend/src/features/admin-consents/AdminConsentsPage.tsx`
- `frontend/src/features/admin-exports/DataExportPage.tsx`
- `frontend/src/features/admin-identity-providers/AdminIdentityProviderDetailPage.tsx`
- `frontend/src/features/admin-jobs/JobsBrowser.tsx`
- `frontend/src/features/admin-keys/AdminKeysPage.tsx`
- `frontend/src/features/admin-provisioning/AdminProvisioningOverviewPage.tsx`
- `frontend/src/features/admin-settings/ApiTokensTab.tsx`
- `frontend/src/features/admin-users/AdminUsersPrimitives.tsx`
- `frontend/src/features/admin-workload-identity/presentation.ts`
- `frontend/src/features/system-tenants/SystemTenantsShared.tsx`

画面ごとに書式が異なり、`AccountApplicationsPage.tsx` は英語の画面でも日本語の書式（`ja-JP`）で日付を表示する。

## Scope

- 各箇所の書式が、画面の要件として意図的に異なるのかを確認する。秒まで必要な監査イベント、日付だけでよい一覧などを区別する。
- 意図的な違いは、`useFormatters` に書式の種類を足して表す。
- 意図的でないものは、既存の `formatDate`、`formatDateTime`、`formatNumber` へ置き換える。
- `AccountApplicationsPage.tsx` の日本語固定を解消する。
- 置き換えの結果を[ユーザーインターフェース設計](../docs/design/application/user-interface.md#国際化)の書式の種類として書く。

## Out of Scope

- タイムゾーンの選択機能。
- 相対時刻（「3 分前」）の表示。

## Design

書式の種類は、日付、日時（分まで）、日時（秒まで）の三つを候補とし、確認の結果で確定する。
`React` のフックを呼べない純粋な関数（`accountSecurityPresentation.ts`、`presentation.ts`）には、ロケールを引数で受け取る整形関数を `lib/i18n/` から渡す。

## Plan

1. 各箇所の書式を一覧にし、意図の有無を判断する。
2. `AccountApplicationsPage.tsx` の英語表示で日本語書式が出ることを単体テストで確認する（Acceptance RED）。
3. 書式の種類を `useFormatters` に足し、各箇所を置き換える。

## Tasks

- [ ] T001 [Design] 各箇所の書式の意図を確認し、書式の種類を決める。
- [ ] T002 [Acceptance] 英語の画面で日本語書式が表示される RED を確認する。
- [ ] T003 [App] 整形を置き換える。
- [ ] T004 [Docs] 書式の種類を設計文書に書く。
- [ ] T005 [Verify] `mise run verify-ui` を通す。

## Verification

- `mise run verify-ui`

## Risk Notes

既存のテストが整形後の文字列を直接比較している場合、書式の統一で期待値が変わる。期待値は同じ整形関数で作った値と比較する形へ直す。
