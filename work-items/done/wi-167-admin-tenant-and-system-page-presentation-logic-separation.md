---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-10
---

# テナント設定とシステム運用画面のプレゼンテーション分離と単体テスト拡充

## Motivation
テナント設定・属性・署名鍵・システムテナントの画面は、運用操作の副作用と表示・入力部品が密結合である。高リスクな運用画面を独立してテスト可能にし、意図しない操作回帰を防ぐ。

## Scope
- `ui/src/features/admin-settings/`、`admin-tenants/`、`admin-keys/`、`system-tenants/`。
- 設定フォーム、鍵一覧・詳細、鍵ヘルス表、テナント詳細・編集の presentation component と pure function の抽出。
- 抽出物の Vitest/React Testing Library 単体テスト。

## Out of Scope
- API 契約、認可、画面遷移、見た目・動作の変更。
- 管理リソース画面（`wi-166`）。

## Plan
- SCL は変更しない。副作用を container に残し、テーブル・フォーム・詳細カードを小さい props 契約へ分割する。
- 破壊的な鍵操作は、busy・確認状態・コールバックを独立部品のテストで確認する。

## Tasks
- [x] T001 [UI] 設定・テナント属性・署名鍵の画面を分離する。
- [x] T002 [UI] システムテナントと鍵ヘルスの画面を分離する。
- [x] T003 [Test] Presentation / pure function の単体テストを追加する。
- [x] T004 [Verify] `just yaml-check`、`just test-ui-unit`、`just verify-ui` を通す。

## Verification
- `just verify-ui`
- `just test-ui-unit`

## Risk Notes
鍵ローテーションやテナント更新の確認・busy 状態を損なうリスクがある。API 呼び出しを container に固定し、presentation は操作の通知だけを行う。

## Completion

- **Completed At**: 2026-07-10
- **Summary**: 署名鍵・システムテナント・鍵ヘルスの一覧／詳細を小さな presentation component に分離し、設定とテナント属性のフォーム正規化を pure function として切り出した。各 props 契約と表示分岐を UI 単体テストで確認できるようにした。
- **Affected Guarantees State**: API 契約、認可、画面遷移、既存の表示スタイルと操作結果は維持されている。SCL の仕様には変更がない。
- **Verification Results**:
  - `just yaml-check` — passed
  - `just test-ui-unit` — passed (170 tests, 33 files)
  - `just verify-ui` — passed (format check, lint, typecheck, unit tests, production build)
- **Evidence**:
  - 実行日: 2026-07-10
  - 実行環境: ローカル開発環境
  - 実行主体: Codex
  - 対象ソース版: `main`（コミット前）
  - 保存先: CI 外部成果物なし。上記コマンドの成功結果を本記録に要約。
