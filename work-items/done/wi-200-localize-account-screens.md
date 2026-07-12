---
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-07-12
depends_on: [wi-158-i18n-ja-en, wi-197-backend-api-errors-english-only]
---

# Account 画面の ja/en 辞書化

## Motivation

アカウントポータルの表示、dialog、状態ラベルが英語選択時にも日本語のまま残る。

## Scope

- SCL: `IdentityManagement.user_experience.localization`。
- `frontend/src/features/account/` の全画面・dialog・状態・日時表示を feature-local ja/en 辞書へ移す。
- 英語既定の component test と日本語明示の i18n test を追加する。

## Out of Scope

- hosted auth、admin、system の画面移行。

## Plan

- 各 account feature に近接辞書を置き、共通ドメインラベルは既存辞書を再利用する。

## Tasks

- [x] T001 [UI] account 全画面を辞書化する。
- [x] T002 [Test] locale 別描画と日時書式を検証する。
- [x] T003 [Verify] `just verify-ui` を通す。

## Verification

- `just verify-ui`

## Risk Notes

自己サービス操作のエラーを backend message と取り違えないよう stable code のみを翻訳する。

## Completion

- Account feature-local ja/en dictionaries cover account portal state, dialog, and action copy.
- Account dates use locale-aware formatters where rendered by localized screens.
- Verified with `just verify-ui`.
