---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-10
---

# 共有シェルを保護しフロントエンド Lines カバレッジ40%を達成する

## Motivation
API と主要画面のテストを積み上げても、共有ナビゲーションや再認証 UI が未保護なら回帰検知に穴が残る。横断 UI を保護し、全体カバレッジ目標を客観的に達成する必要がある。

## Scope
- `ui/src/components/AdminShell.tsx`、`SystemShell.tsx`、`AdminPaneActions.tsx`、`StepUpDialog.tsx` の重要操作。
- 先行 work item の完了後に `just test-ui-cover` で全体 Lines 40% 以上を確認する。

## Out of Scope
- カバレッジ除外設定による数値調整。
- E2E テストと CI の閾値強制。

## Plan
- シェルのナビゲーション、権限・状態表示、確認・キャンセルを公開 UI から検証する。
- 数値達成のためだけのアサーションは追加しない。

## Tasks
- [x] T001 [Test] 共有シェルとインタラクション部品を追加する。
- [x] T002 [Verify] `just test-ui-cover` で Lines 40% 以上を確認する。
- [x] T003 [Verify] `just verify-ui` を成功させる。

## Verification
- `just test-ui-cover` — 全体 Lines 40% 以上。
- `just verify-ui`

## Risk Notes
集約カバレッジは先行 work item の完了度に依存する。達成値と未カバー領域を毎回記録する。

## Completion

- **Completed At**: 2026-07-11
- **Summary**: `AdminShell`・`SystemShell`・`AdminPaneActions` にコンポーネントテストを新設し、`StepUpDialog` に passkey (webauthn) 経路のテストを追加した。ナビゲーションの active 状態、パンくずの一段/二段表示、アカウントメニューの開閉とログアウト、権限・状態表示（busy / disabled / danger tone）、確認・キャンセルを公開 UI から検証する。特に `SystemShell.tsx` は本 work item 以前は 0% カバレッジだった。
- **Affected Guarantees State**: UI の仕様・振る舞いは変更していない。共有シェルと再認証ダイアログの回帰検知をテストで強化した。
- **Verification Results**:
  - `just test-ui-cover` — passed（48 test files / 281 tests。全体 Lines 54.18%、`src/components` Lines 97.33%、`SystemShell.tsx` 0% → フル網羅）
  - `just verify-ui` — passed（format check、lint、typecheck、test、production build）
- **Evidence**:
  - 実行日: 2026-07-11
  - 実行環境: ローカル開発環境
  - 実行主体: Claude Code
  - 対象ソース版: main（完了時点）
  - 保存先: 外部成果物なし。上記検証結果を本記録に要約。
