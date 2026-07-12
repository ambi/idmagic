---
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-07-12
depends_on: [wi-158-i18n-ja-en, wi-197-backend-api-errors-english-only]
---

# Hosted auth 画面の ja/en 辞書化

## Motivation

ログイン以外の hosted auth 画面にも利用者向けの日本語直書きが残り、英語選択時に混在する。

## Scope

- `frontend/src/components/AuthShell.tsx` と hosted auth の callback、email verify、forgot/reset password、home、status、device 画面を feature-local ja/en 辞書へ移す。
- 既知の stable error code を選択 locale の辞書で表示し、未知の backend message は英語のまま保持する。
- 英語を既定とする代表描画テストと、日本語を明示指定する i18n テストを追加する。

## Out of Scope

- account、admin、system 画面の文言移行。

## Plan

- AuthShell の共有文言は近接する shared 辞書を使い、画面固有文言は各ページの `.i18n.ts` に置く。

## Tasks

- [x] T001 [UI] shared shell と各 hosted auth 画面を辞書化する。
- [x] T002 [Test] 英語既定・日本語明示の描画と error 境界を検証する。
- [x] T003 [Verify] `just verify-ui` と `just test-ui-e2e` を通す。

## Verification

- `just verify-ui`
- `just test-ui-e2e`

## Risk Notes

認可フローの画面では文言テストが操作経路に依存するため、role と状態を優先して検証する。

## Completion

- **Completed At**: 2026-07-12
- **Summary**:
  AuthShell と hosted auth の callback、email verification、password recovery、home、status、device の利用者向け文言を ja/en 辞書へ移行した。
  英語既定の代表画面テストを追加・更新し、既知 error code の UI 翻訳と未知 backend message の維持を確認した。
- **Affected Guarantees State**:
  hosted auth のすべての利用者向け文言は選択 locale の辞書から表示され、英語が既定 locale として利用される。
- **Verification Results**:
  - `just verify-ui` — passed
  - `just test-ui-e2e` — passed
  - `just yaml-check` — passed
- **Evidence**:
  - 実行日: 2026-07-12
  - 実行環境: ローカル開発環境 (macOS)
  - 実行主体: Codex (GPT-5)
  - 対象ソース版: main (コミット前作業ツリー)
  - 保存先: 外部成果物なし。UI unit / E2E テストで代表画面を確認。
