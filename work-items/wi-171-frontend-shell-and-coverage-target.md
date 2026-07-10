---
status: pending
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
- [ ] T001 [Test] 共有シェルとインタラクション部品を追加する。
- [ ] T002 [Verify] `just test-ui-cover` で Lines 40% 以上を確認する。
- [ ] T003 [Verify] `just verify-ui` を成功させる。

## Verification
- `just test-ui-cover` — 全体 Lines 40% 以上。
- `just verify-ui`

## Risk Notes
集約カバレッジは先行 work item の完了度に依存する。達成値と未カバー領域を毎回記録する。
