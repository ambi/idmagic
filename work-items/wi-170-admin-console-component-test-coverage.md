---
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-10
---

# 管理コンソールの高リスク操作をコンポーネントテストで保護する

## Motivation
管理コンソールのユーザー、アプリケーション、グループ、エージェント操作は影響範囲が広い一方、主要画面の多くが未テストである。管理者が観測する成功・失敗・権限状態を保護する必要がある。

## Scope
- `ui/src/features/admin-users/`、`admin-applications/`、`admin-groups/`、`admin-agents/` の代表的な読み込み・作成・更新・削除失敗経路。
- 管理ダッシュボード、設定、監査、鍵、同意の高リスク操作。

## Out of Scope
- API クライアントの網羅（`wi-168`）。
- 認証・アカウント画面（`wi-169`）。
- E2E テストと UI 仕様変更。

## Plan
- 高リスクの大規模画面から、API モックを介してロード・主要操作・失敗表示を検証する。
- 画面の公開 UI と利用者操作のみをアサートする。

## Tasks
- [ ] T001 [Test] ユーザー・アプリケーション管理の代表操作を追加する。
- [ ] T002 [Test] グループ・エージェント・補助管理画面を追加する。
- [ ] T003 [Verify] `just test-ui-cover` と `just verify-ui` を成功させる。

## Verification
- `just test-ui-cover`
- `just verify-ui`

## Risk Notes
巨大な画面を一度に網羅しようとすると脆いテストになるため、変更リスクの高い利用者操作を優先する。
