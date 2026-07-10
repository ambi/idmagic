---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-10
---

# 管理リソース画面のプレゼンテーション分離と単体テスト拡充

## Motivation
管理コンソールのリソース一覧・詳細・編集画面では、取得や更新の副作用と一覧・カード・ダイアログの描画が混在している。画面単位で変更が大きくなりすぎないよう、認証フローから切り離して管理リソース画面だけを一貫した Container / Presentation 構造へ移行する。

## Scope
- `ui/src/features/admin-users/`、`admin-applications/`、`admin-agents/`、`admin-groups/`、`admin-roles/`、`admin-audit-events/`、`admin-authz-detail-types/`、`admin-consents/`、`admin-dashboard/`、`admin-entra-federation/`。
- 一覧、詳細、フォーム、ダイアログごとの presentation component と pure な表示・フィルター関数の抽出。
- 各抽出物の Vitest/React Testing Library 単体テスト。

## Out of Scope
- API 契約、認可、画面遷移、見た目・動作の変更。
- テナント設定、鍵、システムコンソール画面（`wi-167`）。

## Plan
- SCL は変更しない。page container に API 呼び出し・状態・ナビゲーションを残し、描画単位を小さな props 契約で抽出する。
- 既に抽出済みの部品は export と単体テスト追加を優先し、巨大な page 全体のラッパーは導入しない。

## Tasks
- [x] T001 [UI] ユーザー・アプリケーション・エージェント・グループ・ロール画面を分離する。
- [x] T002 [UI] 監査・認可詳細・同意・ダッシュボード・Entra 画面を分離する。
- [x] T003 [Test] Presentation / pure function の単体テストを追加する。
- [x] T004 [Verify] `just yaml-check`、`just test-ui-unit`、`just verify-ui` を通す。

## Verification
- `just verify-ui`
- `just test-ui-unit`

## Risk Notes
一覧の選択状態と削除・保存操作のコールバック接続を壊すリスクがある。presentation の props 契約を小さく保ち、各操作を単体テストで確認する。

## Completion

- **Completed At**: 2026-07-10
- **Summary**: 管理リソース画面の既存抽出物を公開可能な presentation / pure function として整備し、同意の検索・表示、ダッシュボードのカードとリンク、Entra フェデレーション一覧を container の副作用から独立してテストできるようにした。
- **Affected Guarantees State**: API 契約、認可、画面遷移、既存の表示スタイルと操作結果は維持されている。SCL の仕様には変更がない。
- **Verification Results**:
  - `just yaml-check` — passed
  - `just test-ui-unit` — passed (166 tests, 29 files)
  - `just verify-ui` — passed (format check, lint, typecheck, unit tests, production build)
- **Evidence**:
  - 実行日: 2026-07-10
  - 実行環境: ローカル開発環境
  - 実行主体: Codex
  - 対象ソース版: `main`（コミット前）
  - 保存先: CI 外部成果物なし。上記コマンドの成功結果を本記録に要約。
