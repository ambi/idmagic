---
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-12
depends_on: [wi-158-i18n-ja-en, wi-197-backend-api-errors-english-only]
---

# Admin と System 画面の ja/en 辞書化

## Motivation

管理コンソールとシステムコンソールは画面数・dialog・空状態が多く、英語選択で文言が混在する。

## Scope

- `frontend/src/features/admin-*` と `features/system-tenants` の画面、dialog、aria label、状態、日時・数値を feature-local ja/en 辞書へ移す。
- 英語既定の component test と日本語明示の i18n test を追加する。

## Out of Scope

- hosted auth と account の画面移行。

## Plan

- admin feature ごとに辞書を置き、shared shell と domain labels は既存の共通辞書を利用する。

## Tasks

- [ ] T001 [UI] admin と system の全画面を辞書化する。
- [ ] T002 [Test] locale 別描画、空状態、dialog、日時・数値書式を検証する。
- [ ] T003 [Verify] `just verify-ui` と `just test-ui-e2e` を通す。

## Verification

- `just verify-ui`
- `just test-ui-e2e`

## Risk Notes

多数の管理操作でテキスト locator が使われているため、E2E を locale 非依存の role / state locator へ移す。
