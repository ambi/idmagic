---
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-10
---

# 管理画面と認証フロー画面におけるプレゼンテーションコンポーネントとロジックの分離および単体テストの拡充

## Motivation
`wi-133` でアカウントポータル配下（`ui/src/features/account/`）の主要 `*Page.tsx` はすべて Container / Presentation 分離と単体テスト追加、`ui/ARCHITECTURE.md` へのガイドライン明記まで完了した。しかし対象は account 配下に絞ったため、管理コンソール（`ui/src/features/admin-*/`, `ui/src/features/system-tenants/`）と認証フロー（`ui/src/features/auth-flow/`）の主要画面は依然として密結合なままである。
`wi-133` と同じ分離アプローチ・テスト方針を、残りの主要画面へ適用してコードベース全体の一貫性を保ち、テストカバレッジを引き続き引き上げる必要がある。

## Scope
- `ui/src/features/admin-*/` 配下の未対応 `*Page.tsx`: `AdminUsersPage`, `AdminApplicationsPage`, `AdminSettingsPage`, `AdminAgentsPage`, `AdminAuditEventsPage`, `AdminAuthorizationDetailTypesPage`, `AdminConsentsPage`, `AdminDashboardPage`, `AdminEntraFederationPage`, `AdminGroupsPage`, `AdminKeysPage`/`SystemKeyHealthPage`, `AdminRolesPage`, `AdminTenantAttributesPage`。
  - `AdminSignInPolicyPage`（`wi-132` で対応済み）は対象外。
- `ui/src/features/system-tenants/SystemTenantsPage.tsx`。
- `ui/src/features/auth-flow/` 配下の主要 `*Page.tsx`: `CallbackPage`, `ConsentPage`, `DevicePage`, `EmailVerifyPage`, `ForgotPasswordPage`, `HomePage`, `LoginPage`, `ResetPasswordPage`, `StatusPage`, `TotpPage`。
- 上記画面から API 呼び出し・状態管理（Container）を分離し、フォーム表示・バリデーション・一覧表示などのセクション単位でプレゼンテーションコンポーネントへ切り出す。
- 各画面の入力バリデーションロジックを pure な関数（`ui/src/lib/validation.ts` か画面ローカルの pure function）へ切り出す。
- 切り出したすべての Presentation コンポーネントおよび pure function に対する Vitest/React Testing Library 単体テストを追加する。

## Out of Scope
- 各画面の挙動やスタイルの変更（UI の見た目や動作仕様は一切変更しない）。
- E2E テストシナリオの変更。
- `ui/src/features/account/` 配下（`wi-133` で対応済み）、`AdminSignInPolicyPage`（`wi-132` で対応済み）の再分割。

## Plan
- SCL のドメイン仕様・外部契約・画面遷移仕様は変更しない。`spec/scl.yaml` は更新しない。
- 分離方針は [ui/ARCHITECTURE.md](file:///Users/tn/src/idmagic/ui/ARCHITECTURE.md) の「Container / Presentation component split」セクション（`wi-132`/`wi-133` で明記済み）にそのまま従う。
  - ページ全体を 1 個の `XxxPresentation` に丸ごと包む split は避ける。意味のあるセクション（フォーム・一覧・カードなど）ごとに小さいプレゼンテーションコンポーネントへ切り出し、props は概ね 10 未満に抑える。
  - 静的な read-only マークアップは無理に切り出さず、container にインラインで残してよい。
  - `AccountShell`/`AdminShell`/`AuthShell`/`SystemShell` など TanStack Router の `Link` を使うラッパーをテストで render する場合は `src/test/renderWithRouter.tsx` を使う。
- 画面数が多いため、1 セッションで全て終わらせる前提を置かず、管理画面グループ → 認証フローグループのように区切って検証ゲートを都度通す。

## Tasks
- [ ] T001 [UI] `admin-*` 配下（`AdminSignInPolicyPage` を除く）の主要 `*Page.tsx` と `SystemTenantsPage` を Container / Presentation に分離する。
- [ ] T002 [UI] `auth-flow` 配下の主要 `*Page.tsx` を Container / Presentation に分離する。
- [ ] T003 [Test] 抽出した Presentation / pure function の単体テストを追加する。
- [ ] T004 [Verify] `just yaml-check`、`just test-ui-unit`、`just verify-ui` を通す。

## Verification
- `just verify-ui`
- `just test-ui-unit`
- 対象画面が開発サーバー（`just dev-ui`）で正常にレンダリング・動作することの確認。

## Risk Notes
管理コンソールの主要画面構造を広く変更するため、ルーティング接続や API 呼び出しとの接続が壊れるリスクがある。グループごとに検証ゲート（`just verify-ui`）を通してから次のグループへ進み、未検証の変更を積み上げない。仕上げに既存の E2E テスト（`just test-ui-e2e`）で回帰がないことを確認する。
