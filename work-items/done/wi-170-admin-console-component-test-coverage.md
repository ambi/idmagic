---
status: completed
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
- [x] T001 [Test] ユーザー・アプリケーション管理の代表操作を追加する。
- [x] T002 [Test] グループ・エージェント・補助管理画面を追加する。
- [x] T003 [Verify] `just test-ui-cover` と `just verify-ui` を成功させる。

## Verification
- `just test-ui-cover`
- `just verify-ui`

## Risk Notes
巨大な画面を一度に網羅しようとすると脆いテストになるため、変更リスクの高い利用者操作を優先する。

## Completion

- **Completed At**: 2026-07-11
- **Summary**: `admin-users`・`admin-applications`・`admin-groups`・`admin-agents`・`admin-audit-events` の各画面に、API 境界をモックしたコンポーネントテストを新設した。一覧の再読み込み失敗、作成・更新・削除の成功/失敗、確認ダイアログのキャンセルなど、代表的な高リスク操作の経路を公開 UI を通じて検証する。
- **Affected Guarantees State**: UI の仕様・API 契約・管理操作の振る舞いは変更していない。利用者が観測する主要な管理操作の回帰検知をテストで強化した。
- **Verification Results**:
  - `just test-ui-cover` — passed（45 test files / 266 tests。`admin-users` 53.4%、`admin-applications` 45.3%、`admin-groups` 70.4%、`admin-agents` 57.2%、`admin-audit-events` 65.2%）
  - `just verify-ui` — passed（format check、lint、typecheck、production build）
- **Evidence**:
  - 実行日: 2026-07-11
  - 実行環境: ローカル開発環境
  - 実行主体: Claude Code
  - 対象ソース版: main（完了時点）
  - 保存先: 外部成果物なし。上記検証結果を本記録に要約。
