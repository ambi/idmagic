---
status: completed
authors: ["tn"]
risk: high
created_at: 2026-07-12
depends_on: [wi-158-i18n-ja-en, wi-197-backend-api-errors-english-only]
scope:
  - System:invariants:FrontendLocalizationCompleteness
  - System:scenarios:選択した表示言語で全UI画面が描画される
  - System:scenarios:既知のバックエンドエラーコードはUIで翻訳される
  - System:objectives:TranslationKeyIntegrity
  - System:user_experience:UX-LOCALE
  - System:user_experience:UX-API-ERROR-ENGLISH-ONLY
  - System:glossary:FallbackLocale
  - System:glossary:ConfiguredDefaultLocale
  - System:scenarios:起動時設定の既定localeがフォールバックに使われる
---

# すべての UI 画面を日本語と英語で表示できるようにする

## Motivation

wi-158 で i18n の基盤と主要 hosted auth 画面を導入したが、account、admin、system を含む画面には日本語直書きが残っている。サポート済みの ja / en を選んだ利用者が画面ごとに異なる言語を見る状態を解消する。

## Scope

- `spec/contexts/system.yaml` の invariants、scenarios、objectives、`user_experience.UX-LOCALE` を全 UI screen の完全な ja/en 表示保証へ更新する。
- `frontend/src/` の account、admin、system、hosted auth、shell、dialog、form、empty state、aria label、日時・数値・状態ラベルを feature-local の ja/en 辞書へ移す。
- UI が既知の stable error code を翻訳し、wi-197 により英語固定となる未知 backend message はそのまま表示する境界を適用する。
- 日本語直書き、辞書キーの欠落、ja/en の代表画面描画を検出するテストを追加する。

## Out of Scope

- ja / en 以外の locale。
- メール、監査ログ、テナント任意入力値の翻訳。

## Plan

- SCL の対象 screen 一覧を正本として、shared chrome、hosted auth、account、admin、system の順に辞書移行する。
- 既存の feature-local 辞書を維持し、巨大な共通辞書を作らない。
- 各画面の component test を ja/en で実行し、E2E の文言検索を locale 非依存の操作確認へ移す。
- 実装は wi-199（hosted auth）、wi-200（account）、wi-201（admin / system）へ分割し、本 WI は横断仕様・共通基盤・統合検証を管理する。

## Tasks

- [x] T001 [SCL] 全画面の ja/en 表示・fallback・error 表示境界を更新する。
- [x] T002 [UI] shared shell と hosted auth の残存文言を辞書化する（wi-199）。
- [x] T003 [UI] account / admin / system の全画面・dialog・aria label・状態表示を辞書化する（wi-200 / wi-201）。
- [x] T004 [Test] 直書き検出、辞書完全性、ja/en 描画、locale 非依存 E2E を追加する（wi-199〜201）。
- [x] T005 [Docs] 文言追加と画面追加時の ja/en 要件を開発者文書へ追記する。
- [x] T006 [Verify] `just yaml-check`、`just verify-ui`、`just test-ui-e2e`、`just verify` を通す。

## Verification

- `just yaml-check`
- `just verify-ui`
- `just test-ui-e2e`
- `just verify`

## Risk Notes

画面数が多く、直書き漏れは利用者ごとの表示不整合になる。feature 単位で辞書とテストを近接配置し、UI error と backend error の責務境界を保つ。

## Completion

- **Completed At**: 2026-07-12
- **Summary**:
  子 WI-199〜201 による hosted auth、account、admin、system の辞書化を統合した。残存していたページタイトル、ルーターエラー画面、step-up dialog、共通 Select / Toast、API フォールバックも ja/en の表示責務または英語固定の backend error 境界へ揃えた。
- **Affected Guarantees State**:
  `FrontendLocalizationCompleteness`、`TranslationKeyIntegrity`、`UX-LOCALE`、`UX-API-ERROR-ENGLISH-ONLY` を満たす実装と検証を完了した。
- **Verification Results**:
  - `just verify-ui` — passed
  - `just test-ui-e2e` — passed
  - `just verify` — passed
  - `just yaml-check` — passed
- **Evidence**:
  - 実行日: 2026-07-12
  - 実行環境: ローカル開発環境 (macOS)
  - 実行主体: Codex (GPT-5)
  - 対象ソース版: main（コミット前作業ツリー）
  - 保存先: 外部成果物なし。UI unit / E2E と全体検証の結果を上記に記録。
