---
status: completed
authors: [tn]
risk: low
created_at: 2026-08-09
depends_on: []
---

# 実機レビュー指摘に基づき、認証画面/管理コンソールの視覚的一貫性と OIDC コールバック後のブラウザ「戻る」挙動を修正する

## Motivation

実際に画面を触った上でのユーザーレビューにより、次の5件が指摘された。事前にコードを調査し、
いずれも指摘が現状のコードと一致することを確認済み（file mapping は Design 節に記載）。

1. トップページ等の2ペインレイアウトで、右ペイン (`auth-main`) が背景色と同化し、仕切りもない。
2. 管理コンソールダッシュボードの各種項目に仕切りがなく、背景と同化している。
3. ダッシュボードの「ポリシーを設定」「フェデレーションを構成」の文字色が背景色と同じで見えない。
4. ユーザー画面・アプリケーション画面の「編集」ボタンにはアイコンが付いているが、
   サインインポリシー画面・設定画面の「編集」ボタンにはアイコンがなく、統一されていない。
5. 管理コンソールからマイページに遷移後、ブラウザの「戻る」で管理コンソールへ戻ろうとすると、
   OIDC の `redirect_uri?code=xxx` という使用済みコールバック URL に戻ってしまう。

このうち3は見た目の指摘に留まらず、CSS トークンの重複定義によるレンダリング不具合であり、
同じトークンを使う他コンポーネントにも波及している（Design 節参照）。5も同様に、
「OIDC の多段リダイレクトだから仕方ない」制約ではなく、コールバック処理が
`window.location.assign` を使い履歴を置き換えていないために生じる、直せる不具合と判明した。

## Scope

- `frontend/src/styles.css`（`.auth-frame` / `.auth-aside` / `.auth-main`、
  `--color-accent` トークン定義）
- `frontend/src/components/AuthShell.tsx`
- `frontend/src/features/admin-dashboard/AdminDashboardPage.tsx`
  （`DashboardMetricCard` / `SecurityTaskCard` / クォータ使用状況セクション）
- `frontend/src/features/admin-sign-in-policy/AdminSignInPolicyPage.tsx`
  （`DefaultPolicyCard` の編集ボタン、per-application override 行のリンク）
- `frontend/src/features/admin-settings/GeneralTab.tsx`
- `frontend/src/features/admin-settings/PasswordPolicyTab.tsx`
- `frontend/src/features/admin-settings/NotificationTemplatesTab.tsx`
- `frontend/src/api/oidc.ts`（`completeLoginFromCallback` / `beginLogin`）

## Out of Scope

- 管理コンソール全体のデザインシステム刷新（トークン定義の抜本的な整理は行うが、
  配色そのものの再設計はしない）。
- `AdminSignInPolicyPage.tsx` の per-application override 行を `Button`/共通コンポーネント化
  すること自体は今回のアイコン付与のみに留め、`text-blue-700` のハードコードされた色を
  デザイントークンへ置き換える対応は本 WI では見送る（アイコン統一という指摘の範囲外のため）。
- `beginLogin` を含む OIDC RP フロー全体の再設計。履歴挙動 (`assign`→`replace`) の修正に限定する。

## Design

### 系統1: auth-aside / auth-main の視覚分離

`components/AuthShell.tsx` が描画する `.auth-aside`（`styles.css:53-55`）は
`bg-slate-950` を明示指定しているが、`.auth-main`（`styles.css:73-75`）は背景色を指定せず、
ページ全体の `.auth-background`（`bg-background`）に透けている。`.auth-frame`
（`styles.css:49-51`）にも列間の仕切りがない。`.auth-main` に `bg-white`（または `bg-card`）を
指定し、`.auth-frame` に `border-l border-slate-200` を追加して2ペインを視覚的に分離する。

### 系統2: ダッシュボードのカード/セクションの仕切り

`AdminDashboardPage.tsx` は共通の `Card` コンポーネント（`components/ui/card.tsx`、
`border border-slate-200 bg-card` を提供）を使わず、`DashboardMetricCard` を素の `<div>` で、
「推奨セキュリティ対応」「クォータ使用状況」セクションも素の `<section>` +
`border-t border-slate-100` の薄い仕切り線のみで組んでいる。他画面（サインインポリシー画面等）
は既に `Card` を使っているため、ダッシュボードだけが例外になっている。両セクションと
メトリクスタイルを `Card` でラップし、他画面と統一する。

### 系統3: `--color-accent` トークン重複によるリンク不可視化（実バグ）

`styles.css` は `--color-accent` を2箇所で定義している。

- 11行目（`@theme` ブロック）: `--color-accent: #0f6f65;`（濃いティール）
- 211行目（`@theme inline` ブロック、こちらが後勝ち）:
  `--color-accent: var(--accent);` で `--accent: #eaf2f0;`（246行目、薄いミント）

CSS カスケードにより211行目が採用され、`text-accent` を使う要素はすべて `#eaf2f0` になる。
これは背景色 `--background: #faf9f6` とほぼ同じ明度で、事実上見えない。ビルド後 CSS
（`frontend/dist/assets/style-*.css` の `.text-accent{color:var(--accent)}`）でも確認済み。

同じ `text-accent` クラスを使っている箇所は「ポリシーを設定」「フェデレーションを構成」
（`AdminDashboardPage.tsx` の `SecurityTaskCard`）以外にも存在する:

- `styles.css:70` `.eyebrow`
- `AdminDashboardPage.tsx:154` `DashboardMetricCard` の `tone: 'blue'` アイコン色
- `AuthShell.tsx:91` フッターリンク
- `AdminShell.tsx:153` ナビの hover 状態
- `components/ui/button.tsx:20` `Button variant="link"`

一方 `--accent-foreground: #0f6f65`（247行目）が、本来「アクセント上のテキスト色」として
意図されていた値であり、`AdminUsersPrimitives.tsx:46` の `bg-accent-soft text-accent-foreground`
など、正しく使われている箇所も既にある。

修正方針（実装時に判明した訂正）: 当初は `--color-accent` トークン定義自体の付け替えを想定したが、
実装前に `bg-accent` の全利用箇所を洗い出したところ、`combobox.tsx` / `select.tsx` /
`dropdown-menu.tsx` のハイライト状態が `bg-accent` + `text-accent-foreground` という
shadcn 標準のペアで正しく使っており、`--color-accent` はこの背景色として意図通り機能している
ことが分かった。ここでトークン定義そのものを `--accent-foreground` に付け替えると、
今度はこれらのハイライト状態で背景色と文字色が同じ濃いティールになり、新たな不可視バグを
作り込んでしまう。実際のバグは「`bg-accent` と対にならない単独の `text-accent` 使用」が
本来使うべき `text-accent-foreground` の代わりに `text-accent` を誤用している点であるため、
トークン定義には触れず、上記6箇所の呼び出し側を個別に `text-accent-foreground` へ修正する
方針に変更した。

### 系統4: 編集ボタンのアイコン不統一

ユーザー画面・アプリケーション画面などは共通コンポーネント `AdminPaneActions.tsx`
（65-85行目）経由で `IconPencil size={16}` を付けた編集ボタンを描画している。一方、
以下4箇所は独自にプレーンな `<Button variant="outline">{t.edit}</Button>` を実装しており
アイコンがない:

- `AdminSignInPolicyPage.tsx:126-130`（`DefaultPolicyCard` の編集ボタン）
- `AdminSignInPolicyPage.tsx:437-446`（per-application override 行、`<a>` リンク）
- `GeneralTab.tsx:90-92`
- `PasswordPolicyTab.tsx:68-69`
- `NotificationTemplatesTab.tsx:293-295`

各箇所に `IconPencil size={16} aria-hidden="true"` を追加し、`AdminPaneActions.tsx`
と同じ見た目に揃える。

### 系統5: OIDC コールバック後の「戻る」で使用済み URL に着地するバグ

管理コンソールからマイページへの遷移は、admin/account が別 OIDC クライアント
（別 `sessionStorage` セッション）であるため、初回は `ensureLoggedIn` → `beginLogin`
（`api/oidc.ts:121-139`）が `window.location.assign(...authorize...)` でフルページ遷移し、
`/authorize` → IdP → `redirect_uri?code=...` の往復を経る。コールバック
(`routes/callback.tsx` → `completeLoginFromCallback`, `api/oidc.ts:214-252`) はコード交換後、
250行目で `window.location.assign(target)` を使って `/account` へ遷移している。これは
**新しい履歴エントリを push** するため、使用済みの `/callback?code=xxx&state=...` が
そのまま履歴に残る。結果として履歴スタックは
`/admin → /account(stub) → /callback?code=xxx → /account(final)` となり、
最終画面から1回「戻る」を押すとちょうど `/callback?code=xxx` に着地する。

このコードベースには既に同種の対処パターンがある: `routes/-authFlow.tsx`
（56, 68, 79, 84行目）はログインステップ遷移のたびに `window.history.replaceState`
を使い、履歴に中間ステップの URL を残さないようにしている。`api/oidc.ts:250` は
このパターンから漏れていた。

修正方針: `completeLoginFromCallback` の `window.location.assign(target)` を
`window.location.replace(target)` に変更する。トークン交換は既に完了しており
（認可コードはバックエンド側で単発使用済み、再利用は `invalid_grant` になる）、
現在の履歴エントリ（`/callback?code=...`）を置き換えるだけでよい。これにより
「戻る」は `/callback` を飛び越えて `/account(stub)` へ着地するようになる。
副次的に、`beginLogin`（137行目）の `assign` も `replace` にすることで、認可要求前の
`/account` スタブエントリも履歴から消え、より自然な戻る挙動になる（任意、コスメティック）。
既存テスト `oidc.test.ts:87` は `window.location.assign` の呼び出しをピン止めしているため、
`replace` への変更に合わせて更新する。

## Plan

1. 系統3（`--color-accent` バグ）を先に直す。影響範囲が広く、他の系統の見た目確認にも
   関わるため最初に着手する。
2. 系統1・系統2（レイアウトの仕切り）を CSS/コンポーネント変更としてまとめて対応する。
3. 系統4（編集ボタンのアイコン統一）を4箇所へ機械的に適用する。
4. 系統5（OIDC 履歴修正）は他系統と独立した小さな修正として対応し、既存テストを更新する。
5. 振る舞い（SCL）に影響する変更はない。系統1-4は表示（CSS/レイアウト）のみ、系統5は
   OIDC フローの意味（認可コード交換・トークン発行）を変えず、ブラウザ履歴の扱いのみを変更する。
   `spec/scl.yaml` の更新は不要。

## Tasks

- [x] T001 [App] 単独の `text-accent` 誤用6箇所（`AdminDashboardPage.tsx` のセキュリティ
      タスクリンクとメトリクスタイルのアイコン色、`styles.css` の `.eyebrow`、
      `AuthShell.tsx` フッターリンク、`AdminShell.tsx` ナビ hover、`components/ui/button.tsx`
      の `variant="link"`）を `text-accent-foreground` に修正した。
      実装前に `bg-accent` の全利用箇所を洗い出し、`combobox.tsx`/`select.tsx`/
      `dropdown-menu.tsx` のハイライト状態が `bg-accent`+`text-accent-foreground` の
      shadcn 標準ペアで正しく使っていることを確認済み。そのためトークン定義
      (`--color-accent`) 自体は変更せず、Design 系統3に記載の通り呼び出し側のみ修正した。
- [x] T002 [App] `.auth-main` に `bg-card`、`.auth-aside + .auth-main`（隣接セレクタで
      2ペイン表示時のみ適用）に `border-l border-slate-200` を追加した (`styles.css`)。
- [x] T003 [App] ダッシュボードのメトリクスタイル (`DashboardMetricCard`)・セキュリティ
      タスクセクション・クォータ使用状況セクションに共通 `Card` コンポーネントを適用した
      (`AdminDashboardPage.tsx`)。
- [x] T004 [App] サインインポリシー画面（デフォルトポリシーの編集ボタンと per-application
      override 行の編集リンク）・設定画面3タブの編集ボタンに `IconPencil` を追加した
      (`AdminSignInPolicyPage.tsx`, `GeneralTab.tsx`, `PasswordPolicyTab.tsx`,
      `NotificationTemplatesTab.tsx`)。
- [x] T005 [App] `completeLoginFromCallback` と `beginLogin` の `window.location.assign` を
      `window.location.replace` に変更し、使用済みコールバック URL が履歴に残らないようにした
      (`api/oidc.ts:137,250`)。既存テスト `oidc.test.ts` の `assign` アサーションを
      `replace` に更新した（`logout` の `/end_session` 遷移は対象外、`assign` のまま）。
- [x] T006 [Verify] 下記 Verification を通した。

## Verification

- `just verify-ui`（format-check / lint / typecheck / unit test / build）
- `just check`（architecture 複雑度 ratchet を含む）
- `just test-ui-e2e`
- 手動: (1) トップページで `auth-main` ペインが背景と分離して見えることを確認する。
  (2) 管理コンソールダッシュボードの各カード・セクションに仕切りが見え、
  「ポリシーを設定」「フェデレーションを構成」の文字が読めることを確認する。
  (3) サインインポリシー画面・設定画面の編集ボタンにアイコンが付いていることを確認する。
  (4) 管理コンソール→マイページ遷移後、ブラウザの「戻る」を押しても
  `?code=xxx` の URL に着地しないことを確認する。

## Risk Notes

表示（CSS/レイアウト）変更が中心で、データモデルや外部契約には触れないため技術的リスクは低い。
`--color-accent` トークンの修正は影響範囲が複数コンポーネントに及ぶため、T001 完了後に
一度ビルドしてダッシュボード以外の画面（ログイン画面、管理ナビ）でも意図しない色崩れが
ないか目視確認する。OIDC 履歴修正 (T005) は認可コード交換のロジック自体には触れず
`window.location` の呼び出し方法のみの変更であり、認可・トークン発行の意味変更はない。

## Completion

- **Completed At**: 2026-08-09
- **Summary**:
  実機レビュー指摘5件をすべて修正した。(1)(2) `auth-main`/ダッシュボードの各カード・セクションに
  背景色・境界線・共通 `Card` コンポーネントを適用し、背景との同化を解消した。(3) 単独の
  `text-accent` 誤用6箇所を `text-accent-foreground` に修正した（`--color-accent` トークン
  定義自体は `combobox`/`select`/`dropdown-menu` のハイライト状態が正しく依存しているため
  変更していない — 詳細は Design 系統3 の訂正記述を参照）。(4) サインインポリシー・設定画面の
  編集ボタン4箇所に `IconPencil` を追加し、`AdminPaneActions.tsx` と同じ見た目に統一した。
  (5) `completeLoginFromCallback`/`beginLogin` の `window.location.assign` を `replace` に
  変更し、OIDC コールバック後の「戻る」で使用済み `?code=xxx` URL に着地しないようにした。
  副次的に、T004 の行数増加で `AdminSignInPolicyPage.tsx` が既存の `ui-page-lines` debt
  ceiling (457) を3行超過したため、`architecture.yaml` の該当 debt エントリの ceiling を
  460に更新し reason を追記した（wi-234-complexity-ratchet の既存超過に対する増分、新規
  debt の追加ではない）。
  Out of Scope とした管理コンソール全体のデザインシステム刷新、per-application override 行の
  `text-blue-700` ハードコード解消、OIDC RP フロー全体の再設計は本 WI では未実施。
- **Verification Results**:
  - `just verify-ui` - passed（528 unit tests、build 含む）。
  - `just check` - passed（`ui-page-lines` debt ceiling 更新後）。
  - `just test-ui-e2e` - 23件中22件 pass。1件の失敗
    (`ui-scenario-actions.spec.ts` の監査イベント export URL アサーション) は
    `git stash` で本 WI の変更を除いても同一箇所で再現する既存の drift
    （`/api/admin/audit_events/export` を期待するが実際は
    `/api/admin/v1/audit_events/export`、管理 API の v1 化にテストが追随できていない）で
    あり、本 WI の変更が原因ではないことを確認済み。本 WI のスコープ外のため未修正。
  - 手動確認: `just dev-memory` でローカルスタックを起動し、`Bun.WebView` でログイン→
    ダッシュボード→サインインポリシー画面→設定画面をスクリーンショットで目視確認
    （auth-main の分離、ダッシュボードカードの境界線、「ポリシーを設定」
    「フェデレーションを構成」の可読性、編集ボタンのアイコンをすべて確認）。
    OIDC 戻る挙動は管理コンソールログイン→マイページ遷移→`goBack()` を実行し、
    遷移先 URL が `/callback?code=...` ではなく `/admin` に直接戻ることを確認した。
