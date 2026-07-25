---
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-25
depends_on: []
change_kind: feature
initial_context:
  scl:
    System:
      - standards.WCAG22.WCAG22-KEYBOARD
      - standards.WCAG22.WCAG22-FOCUS
      - standards.WCAG22.WCAG22-LABELS-ERRORS
      - standards.WCAG22.WCAG22-STATUS
  decisions:
    - decisions/ADR-086-ui-navigation-consistency-and-page-title-policy.md
    - decisions/ADR-097-tenant-branding-color-contrast-is-advisory.md
    - decisions/ADR-105-system-runtime-hardening-and-i18n-tooling.md
  source:
    - frontend/src/components
    - frontend/src/features
    - frontend/tests/e2e
  tests:
    - frontend/tests/e2e
    - frontend/src/test
  stop_before_reading:
    - backend
affected_spec:
  - { context: System, kind: standard_requirement, standard: WCAG22, requirement: WCAG22-KEYBOARD }
  - { context: System, kind: standard_requirement, standard: WCAG22, requirement: WCAG22-FOCUS }
  - { context: System, kind: standard_requirement, standard: WCAG22, requirement: WCAG22-LABELS-ERRORS }
  - { context: System, kind: standard_requirement, standard: WCAG22, requirement: WCAG22-STATUS }
---

# WCAG 2.2 AA 準拠を自動検査で保証し、認証 UI のアクセシビリティを担保する

## Motivation

SCL の `System.standards.WCAG22` は 4 件の要件を **adoption: required / strength: MUST** で
宣言している:

- `WCAG22-KEYBOARD` (2.1.1) — すべての認証操作をキーボードだけで完了可能にする
- `WCAG22-FOCUS` (2.4.7, 2.4.11) — フォーカスを視認可能にし、重要要素が完全に隠れない
- `WCAG22-LABELS-ERRORS` (3.3.1, 3.3.2) — 入力にラベル、エラーをテキストで識別し修正方法を示す
- `WCAG22-STATUS` (4.1.3) — 認証結果や送信エラーをフォーカス移動なしに支援技術へ通知する

しかし **これらを検証する手段が存在しない**。UI 側には E2E (`frontend/tests/e2e`) と
unit test があるが、アクセシビリティ検査 (axe-core 等) は入っていない。
つまり SCL が MUST と宣言した要件が、実装で満たされているかどうか誰も確認していない
「宣言だけの保証」になっている。Regenerative Architecture の建て方 (仕様が正本で、
実装と検証がそれに従う) からすると、これは最も直すべき類の乖離である。

実務上の影響:

1. **調達で落ちる**。公共・金融・大企業の調達ではアクセシビリティ適合表明
   (VPAT / ACR、日本では JIS X 8341-3 の試験結果) を要求される。ログイン画面は
   全ユーザーが必ず通る画面なので、免除されない。
2. **ログインできないユーザーが出る**。IdP のログイン画面がキーボード操作で完了できない、
   エラーがスクリーンリーダーに通知されない場合、当該ユーザーは**業務システム全体から
   ロックアウトされる**。アプリ 1 つの不便ではなく、全アプリの利用不能になる。
3. **既定のスクリーンリーダー通知が壊れやすい**。SPA (React) はページ遷移でフォーカスが
   失われやすく、非同期エラーは DOM 更新だけでは読み上げられない。
   `WCAG22-STATUS` は明示的な `aria-live` 実装が無ければ満たせない。

競合は Okta / Entra ID がともに VPAT を公開しており、Keycloak もアクセシビリティ改善を
継続的に扱っている。「production-ready」を掲げる以上、検査されていない MUST は解消する。

## Scope

- **scl**:
  - `System.standards.WCAG22` の各要件に、検証手段 (自動検査 + 手動確認) の対応を明示する。
  - `System.scenarios` に「キーボードのみでログインを完了できる」「送信エラーが
    フォーカス移動なしに支援技術へ通知される」「フォーカスが常に視認できる」を追加する。
  - 対象画面の範囲を明示する: 認証 UI (ログイン / MFA / パスワードリセット / consent /
    device 確認) を必達、アカウントポータルを次点、管理コンソールを第 3 段とする。
- **decision**:
  - 新規 ADR (アクセシビリティ適合の範囲と検証方法): 準拠レベル (WCAG 2.2 AA)、
    必達対象画面の段階、自動検査で担保する範囲と手動確認に残す範囲
    (axe-core は全項目を検出できないため、境界を明記する)、
    [[ADR-097-tenant-branding-color-contrast-is-advisory]] との整合
    (テナントブランディングのコントラストは advisory のままで、既定テーマは AA を満たす)、
    違反を CI で落とす閾値を記録する。
- **frontend**:
  - E2E に axe-core ベースのアクセシビリティ検査を組み込み、対象画面ごとに violations 0 を
    検証する (深刻度 `critical` / `serious` を CI ブロッキング)。
  - キーボードのみの操作 E2E を追加する: ログイン → MFA → 完了までを Tab / Enter /
    Space だけで到達する。
  - フォーカス可視性: 共通コンポーネント (Button / Input / Select / Dialog / Menu) の
    focus-visible スタイルを点検し、ブランディング色でもコントラストが確保される既定にする。
  - フォーカストラップ: Dialog / Drawer でフォーカスが閉じ込められ、Esc で閉じ、
    閉じたら起点へ戻ることを保証する。
  - ラベルとエラー: すべての入力に可視ラベル (または `aria-label`) と、
    エラーの `aria-describedby` 関連付け、エラー文の「修正方法」記述を確認・修正する。
  - ステータス通知: 非同期の成功 / 失敗を `role="status"` / `role="alert"` の live region で
    通知する共通機構を追加し、既存の各画面のエラー表示をそこへ寄せる。
  - SPA ルーティングのフォーカス管理: 画面遷移後に見出しへフォーカスを移し、
    ページタイトルを更新する ([[ADR-086-ui-navigation-consistency-and-page-title-policy]] と整合)。
- **tooling**:
  - `justfile` に `just test-ui-a11y` を追加し、`verify-ui` または CI の UI ジョブに組み込む。
- **documentation**:
  - `frontend/README.md` にアクセシビリティ方針、対象範囲、検査の実行方法、
    手動確認チェックリスト (スクリーンリーダーでの確認手順) を追記する。
  - 適合表明のための試験結果 (対象画面 × 達成基準 × 結果) を書ける形式を用意する。

## Out of Scope

- WCAG 2.2 AAA レベル。
- 管理コンソール全画面の AA 準拠 (第 3 段として範囲に入れるが、本 WI の必達は認証 UI と
  アカウントポータル)。
- テナントが設定したブランディング色のコントラスト強制。
  → [[ADR-097-tenant-branding-color-contrast-is-advisory]] の advisory 方針を維持する。
- 実ユーザー (支援技術利用者) を招いたユーザビリティテスト。
- 正式な VPAT / ACR の第三者監査。本 WI は自己試験結果を書ける状態を作るところまで。
- メール本文のアクセシビリティ。→ [[wi-288-localized-notification-template-catalog-and-tenant-customization]]

## Plan

- **自動検査の限界を最初に線引きする**。axe-core は WCAG 達成基準の 3〜4 割程度しか
  自動検出できない。「自動で担保する項目」と「手動チェックリストで担保する項目」を
  ADR で分け、後者を `frontend/README.md` の手順として残す。自動検査だけで
  「AA 準拠」と言わないことを明記する。
- **認証 UI を最優先にする**。ここが通れないと全アプリからロックアウトされるため、
  影響度が管理コンソールと桁違いである。段階を SCL に明記して、部分適合の状態を
  「未完了」ではなく「宣言済みの段階」として扱えるようにする。
- **共通コンポーネントで解く**。画面ごとに個別対応すると回帰する。focus-visible、
  live region、Dialog のフォーカストラップは共通コンポーネント側で解決し、
  各画面はそれを使うだけにする。既存の presentation logic 分離
  ([[wi-133-frontend-all-pages-presentation-logic-separation]]) の構造に沿って入れる。
- **エラー文の「修正方法」は文言の問題**である。`WCAG22-LABELS-ERRORS` は
  「エラーを識別し修正方法を示す」ことを要求するため、「入力が不正です」では不足する。
  i18n 辞書側の文言修正を含み、テストは辞書値を参照する (ハードコード訳文を assert しない)。
- **CI ブロッキングは violations の深刻度で絞る**。`critical` / `serious` で落とし、
  `moderate` / `minor` はレポートに留める。初回は棚卸しして 0 件にしてからブロッキングにする。
- 未決定: スクリーンリーダーの手動確認をどの組み合わせで行うか (VoiceOver / NVDA)。
  最低 1 つを手順として固定し、`frontend/README.md` に記載する。

## Tasks

- [ ] T001 [SCL] `System.standards.WCAG22` の各要件に検証手段の対応を追記し、
      scenario 3 件と対象画面の段階を追加して `just check-scl` を通す。
- [ ] T002 [ADR] アクセシビリティ適合の範囲と検証方法の ADR を起票する
      (準拠レベル・対象段階・自動 / 手動の分界・ブランディングとの整合・CI 閾値)。
- [ ] T003 [Tooling] E2E に axe-core を組み込み、`just test-ui-a11y` を `justfile` に追加する。
      対象画面のリストを設定として持つ。
- [ ] T004 [Baseline] 認証 UI とアカウントポータルの現状 violations を棚卸しし、
      深刻度別の件数を記録する。
- [ ] T005 [Focus] 共通コンポーネントの focus-visible スタイルを点検・修正し、
      Dialog / Drawer のフォーカストラップと Esc / 復帰を実装する。
      RED: フォーカストラップの E2E / unit test を先に書く → GREEN。
- [ ] T006 [Status] `role="status"` / `role="alert"` の live region 共通機構を追加し、
      認証 UI の成功 / エラー表示をそこへ寄せる。RED: 送信エラーが live region に入る
      テスト (scenario `System.status_message_announced`) → GREEN。
- [ ] T007 [Labels] 認証 UI とアカウントポータルの全入力にラベルと `aria-describedby` の
      エラー関連付けを入れる。エラー文言を「修正方法を含む」形に i18n 辞書側で直す
      (ja / en 両方)。RED: 辞書値を参照する unit test → GREEN。
- [ ] T008 [Keyboard] キーボードのみでログイン → MFA → 完了に到達する E2E を追加する。
      RED: 先に落ちる E2E を書く → GREEN。
- [ ] T009 [Routing] SPA 遷移後のフォーカス移動とページタイトル更新を共通化する
      ([[ADR-086-ui-navigation-consistency-and-page-title-policy]] と整合)。RED → GREEN。
- [ ] T010 [CI] `test-ui-a11y` を `.github/workflows/idmagic-ci.yaml` の UI ジョブに追加し、
      `critical` / `serious` で落とす閾値を設定する。
- [ ] T011 [Docs] `frontend/README.md` にアクセシビリティ方針・検査手順・手動チェックリスト・
      自己試験結果の様式を追記する。
- [ ] T012 [Verify] 下記 Verification を緑にする。手動でスクリーンリーダー確認を 1 巡する。

## Verification

- `just check` / `just check-scl` / `just check-work-items`
- `just verify-ui` / `just test-ui-unit` / `just test-ui-e2e` / `just test-ui-a11y` (新設)
- 手動: キーボードのみ (マウス不使用) でログイン → MFA → アカウントポータルの
  主要操作を完了できることを確認する。
- 手動: スクリーンリーダー (VoiceOver または NVDA) で、(1) 入力ラベルが読まれる、
  (2) 送信エラーがフォーカス移動なしに読まれる、(3) 画面遷移が通知される、を確認する。
- 手動: ブラウザのズーム 200% と 400% でログイン画面が操作可能であることを確認する。

## Risk Notes

共通コンポーネント (Button / Input / Dialog) の変更は**全画面に波及する**。
視覚的な回帰を避けるため、focus-visible とフォーカストラップの変更は
既存 E2E を通した状態で段階的に入れる。
既存のエラー文言を変えるため、i18n 辞書の変更が広範囲になる。テストは辞書値を参照する
方針を守り、ハードコードされた訳文を assert しない。
axe-core を CI ブロッキングにすると、ライブラリ更新でルールが増えたときに無関係な PR が
落ちうる。深刻度で絞り、棚卸しでベースラインを 0 件にしてから有効化する。
自動検査で検出できない達成基準が多数残るため、「自動検査が緑 = AA 準拠」と誤読されないよう
ADR と `frontend/README.md` に明記する。
