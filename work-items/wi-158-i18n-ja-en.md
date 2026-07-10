---
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-10
---

# 日本語と英語に対応した I18n 基盤と文言管理を導入する

## Motivation
idmagic の SCL は `user_experience.locales` と `UX-LOCALE` で日本語と英語を提供することを要求しているが、
実装には日本語直書きの UI 文言、内部値から日本語表示名へ変換する helper、英語に切り替える入口の欠如が残っている。
管理者・利用者・外部プロトコル利用者が同じ hosted UI を使うため、表示言語を明示的かつ一貫して選べない状態は、
デモ・評価・本番運用のいずれでも誤解や導入障壁になる。

この WI は「多言語一般」ではなく、サポート対象を日本語と英語に限定し、文言、locale 解決、フォールバック、
テストの境界を定めて実装と仕様を一致させる。

## Scope
- **scl**:
  - `spec/contexts/system.yaml` の `user_experience.locales` / `UX-LOCALE` を精密化し、対象 screen を admin / account / hosted auth へ広げる。
  - 必要に応じて `glossary` に Locale / Display Language / Fallback Locale などの用語を追加する。
  - `scenarios` に日本語表示、英語表示、未対応 locale のフォールバック、OIDC `ui_locales` hint の代表例を追加する。
  - `objectives` に文言欠落検出、fallback、ビルド時またはテスト時の translation key 検証方針を追加する。
- **ui**:
  - React UI に i18n runtime と typed translation key 管理を導入し、admin / account / hosted auth の利用者向け文言を `ja` / `en` 辞書へ移す。
  - locale 解決順を定義する。例: 明示選択、OIDC `ui_locales`、保存済み設定、ブラウザ言語、既定 locale。
  - 日付・時刻・数値・状態ラベルを locale に応じて整形し、内部 enum 値や key を直接表示しない。
  - 言語切り替え UI を hosted auth / account / admin の適切な chrome に追加する。
- **go/http**:
  - OIDC `ui_locales_supported` と UI の実対応 locale を一致させる。
  - hosted auth で `ui_locales` hint を受け取り、UI へ渡す経路を明確にする。
  - API error / validation error を UI が安定した key として翻訳できる境界を整理する。
- **tests**:
  - unit / component test で `ja` / `en` の代表画面が描画できること、未対応 locale が既定 locale に落ちることを確認する。
  - translation key の欠落、未使用、直書き回帰を検出する仕組みを追加する。
- **documentation**:
  - README または開発者向け文書に文言追加手順、locale 追加禁止の現時点方針、`ja` / `en` 辞書の管理ルールを書く。

## Out of Scope
- 日本語・英語以外の locale 対応。
- 管理者が任意に文言を編集するテナント別翻訳管理。
- メールテンプレートや監査ログ本文の全面翻訳。ただし UI に表示する error / status key の翻訳境界は扱う。
- 任意 CSS / HTML を含むブランディング文言。これは [[wi-89-tenant-login-branding]] の制約に従う。
- 外部プロトコル仕様で固定された claim 名、scope 名、error code の翻訳。

## Plan
- まず SCL の `UX-LOCALE` を実装対象 screen と解決順まで読めるように補強し、派生物を再生成する。
- UI は共通 i18n module を薄く導入し、feature ごとに辞書を近接配置する。巨大な単一辞書に集約しない。
- 既存の日本語 helper は、内部値から translation key を得る helper と、locale 辞書で表示する責務へ分離する。
- locale は `ja` / `en` の union として扱い、未対応 locale は実行時に既定 locale へ落とす。辞書欠落はテストまたはビルドで落とす。
- `ui_locales` は OIDC の hint として扱い、ユーザーの明示選択を上書きしない。

## Tasks
- [ ] T001 [SCL] `UX-LOCALE`、関連 glossary / scenarios / objectives を更新し、対象 screen と locale 解決順を明記する。
- [ ] T002 [Render] `just scl-render` で SCL 派生物を更新する。
- [ ] T003 [UI] i18n runtime、typed locale / key、辞書構造、fallback を導入する。
- [ ] T004 [UI] admin / account / hosted auth の利用者向け文言、状態ラベル、日時・数値表示を `ja` / `en` 化する。
- [ ] T005 [HTTP] `ui_locales_supported`、hosted auth の `ui_locales` hint、UI への locale 伝播を実対応に合わせる。
- [ ] T006 [Test] locale 解決、辞書欠落検出、代表画面の `ja` / `en` 描画 test を追加する。
- [ ] T007 [Docs] 文言追加手順と二言語限定方針を記録する。
- [ ] T008 [Verify] `just yaml-check`、`just verify-go`、`just verify-ui`、必要に応じて `just test-ui-e2e` を通す。

## Verification
- `just yaml-check`
- `just scl-render`
- `just verify-go`
- `just verify-ui`
- `just test-ui-e2e`
  - reason: 言語切り替えと hosted auth の `ui_locales` hint は browser behavior を含むため。
- 手動: `ja` / `en` / 未対応 locale の代表ブラウザ設定で、login、consent、account、admin の主要画面が意図した言語で表示されることを確認する。
- 手動: OIDC authorize request の `ui_locales` に `en ja`、`ja`、未対応 locale を渡し、明示選択と fallback が仕様どおりになることを確認する。

## Risk Notes
I18n は文言置換に見えて、認証画面、管理画面、アカウント画面、OIDC hint、テストにまたがる横断変更である。
辞書 key の欠落や直書きの混在は利用者ごとの表示不整合を生むため、typed key と欠落検出を導入する。
API error を自然文として翻訳しようとすると server / UI の責務が曖昧になるため、UI は安定した error key と補助値を翻訳し、
protocol 固有の error code は仕様名として保持する。
