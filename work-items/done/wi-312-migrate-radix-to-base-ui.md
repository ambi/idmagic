---
status: completed
authors: [tn]
risk: high
created_at: 2026-08-01
depends_on: [wi-311-evaluate-base-ui-kumo-migration]
---

# shadcn/ui の Base UI 版へ移行し、Button / DropdownMenu / Label / Select のデザインを刷新する

## Motivation

[[wi-311-evaluate-base-ui-kumo-migration]] の調査と意思決定に基づく実装。Radix UI
（`@radix-ui/react-slot` / `react-dropdown-menu` / `react-label`）は開発ペースが落ちており、
Base UI へ移行する価値があると判断した。Base UI を自前でラップし直す方針、および
Cloudflare の `@cloudflare/kumo` を導入する方針はいずれも検討したが不採用とし、
**shadcn/ui の Base UI 版**（`shadcn` CLI で `-b base` を指定してコンポーネントソースを
リポジトリへ直接生成する方式）を採用することにした。理由:

- Kumo はブランド色 `#f6821f`（Cloudflare オレンジ）が `primary` ボタンの背景色や
  全 variant 共通の focus ring 色など 19 コンポーネントに浸透しており、外部ブランドでの
  再利用を想定した設計ではない（issue でも指摘されていない = 前提として受け止められている）。
- Base UI を自前でラップし直す案は、Radix 版と同等のコンポーネント API を維持できる代わりに
  Dialog/Toast/Combobox 等の不足コンポーネントは今後も自前実装を積み増す必要がある。
- shadcn/ui はソースをリポジトリ内に生成する方式なので、CSS 変数ベースの独自デザイン
  トークンを外部ブランドに縛られず自由に決められ、かつ Dialog / Toast(Sonner) / Combobox /
  Command 等の実装済みコンポーネントを必要になった時点で追加できる。

レビューでユーザーから「この機会にデザインを大幅に刷新したい。現行の slate/blue 配色との
互換性は気にしなくてよい」との明確な合意を得ている。**本 WI はデザイン刷新を伴う**。
UI の見た目は意図的に変わるが、ユーザーから見た操作フロー・状態遷移（SCL の
interface / scenario）自体は変えない。

## Scope

- `frontend/components.json`（shadcn CLI の設定、新規作成。`-b base` で Base UI を選択）
- `frontend/src/components/ui/button.tsx`, `button.test.tsx`（shadcn 生成版で置き換え）
- `frontend/src/components/ui/dropdown-menu.tsx`, `dropdown-menu.test.tsx`（同上）
- `frontend/src/components/ui/label.tsx`, `label.test.tsx`（同上）
- `frontend/src/components/ui/select.tsx`, `select.test.tsx`
  （shadcn の `select`/`dropdown-menu` 生成コンポーネントに合わせて作り直す）
- `frontend/package.json`: `@radix-ui/react-slot` / `@radix-ui/react-dropdown-menu` /
  `@radix-ui/react-label` を除去し、shadcn init/add が導入する Base UI 系パッケージを追加
- デザイントークン（Tailwind `@theme` の色変数、`frontend/src/index.css` 等 shadcn init が
  書き込む先）を新しい配色に更新
- 呼び出し側（`asChild` 実測 51 箇所・約 34 ファイル。着手前に `grep -rn asChild frontend/src`
  で再列挙して確定させる。主な内訳:
  - `Button asChild` 単体: 大半のファイル（一覧は [[wi-311-evaluate-base-ui-kumo-migration]]
    の調査時 grep 結果を参照）
  - `DropdownMenuTrigger asChild` で `Button` をラップ: `AdminShell.tsx`,
    `AccountShell.tsx`, `SystemShell.tsx`, `AdminGroupDetailPage.tsx`,
    `AdminAgentDetailPage.tsx`, `AdminUserDetailPage.tsx`
  - `DropdownMenuItem asChild` で `Link` / `button` をラップ: `AdminShell.tsx`,
    `AccountShell.tsx`, `SystemShell.tsx`)
- SCL (`spec/scl.yaml`) は対象外(ユーザーから見た操作フロー・契約は変えない)

## Out of Scope

- `Dialog` / `AlertDialog` / `Toast` / `Card` / `Alert` / `Input` の shadcn 化・デザイン刷新。
  これらは元々 Radix 非依存の自前実装であり、本 WI の Radix 依存除去とは独立したより大きな
  デザイン刷新作業になる。ユーザーの「デザインを大幅に変えてよい」という意向を踏まえ、
  本 WI 完了後に別 WI として提案する(frontend 全体のトーン&マナー・配色・タイポグラフィを
  一括で決める設計判断が要るため、Button/DropdownMenu/Label の機械的移行と同じ単位には
  収めない)。
- Kumo の導入([[wi-311-evaluate-base-ui-kumo-migration]] で不採用と決定済み)。

## Design

### コンポーネント生成方針

1. `frontend/` で `npx shadcn@latest init -b base` を実行し `components.json` を作る。
   Vite + Tailwind v4 構成は idmagic の既存セットアップ(`@tailwindcss/vite`)にそのまま乗る
   はずだが、生成される Tailwind 設定・CSS 変数の書き込み先は実行結果を見て確認する。
2. `npx shadcn add button dropdown-menu label select` でコンポーネントソースを
   `components/ui/` に生成する。生成物は「材料」として扱い、idmagic の命名規約
   (`cn()` の場所、i18n 呼び出し方) に合わせて必要な範囲で調整する。
3. 生成された `Button` / `DropdownMenu` / `Label` の実際の公開 API
   (`asChild` の扱いが Base UI の `render` prop にどうマッピングされているか、
   `Label` が素の `<label>` かそれとも別のプリミティブを使うか) を確認してから
   呼び出し側の変換方針を確定する。[[wi-311-evaluate-base-ui-kumo-migration]] の
   T002 プロトタイプ結果(`asChild → render` の構造変更、非 `<button>` ターゲットへの
   `nativeButton={false}` 要否、`Field.Label` を避けるべき理由)は shadcn 生成版でも
   参考値として有効だが、shadcn 側の実装がそのまま使えるかは生成後に確認する。
4. `Select` は shadcn の `select` コンポーネント(Base UI の `Select` プリミティブに乗る)を
   使うか、現行どおり `dropdown-menu` の上に自前実装するかを、生成結果を見てから決める。

### デザイン刷新の進め方

- 新しい配色・トークンは shadcn init のデフォルトテーマから開始し、idmagic のブランド
  (ロゴ・既存の他画面)との整合を見ながら調整する。具体的な配色案は実装時に決める
  (本 WI では「白紙から自由に決めてよい」という前提のみ確定している)。
  Cloudflare や他社ブランドを想起させる配色(オレンジ基調等)は避ける。
- ダークモード対応は現行スコープ外(既存 UI がライトのみのため)。将来必要になれば
  別途検討する。

### 実装で判明した追加の破壊的変更・注意点

事前調査([[wi-311-evaluate-base-ui-kumo-migration]])で把握できていた差分に加え、
実装時に以下が判明した:

- **`DropdownMenuItem` の `onSelect` は Base UI に存在しない。** Radix 固有の API で、
  Base UI の `Menu.Item` は `onClick` のみ。React の型定義上 `onSelect` は `<div>` にも
  (テキスト選択イベントとして)存在するため **TypeScript は検出できず、実行時に
  ハンドラが呼ばれないだけの静かな不具合になる**。全 `DropdownMenuItem` の `onSelect` を
  `onClick` に変換した。
- **Base UI Select の `role="combobox"` は content ベースの accessible name 算出をしない。**
  Radix 版(実質 `role="button"`)は表示中のテキストがそのまま accessible name になったが、
  `combobox` ロールは `aria-label`/`aria-labelledby`/`<label for>` 等の明示的な情報源が
  無いと名前なしになる(スクリーンリーダー上は無名の要素)。呼び出し側で `htmlFor`/`id`
  を関連付ける改修は本 WI のスコープ外(既存 14 箇所すべてが同じパターンで未対応、
  Radix 時代からの既存負債)なので、`components/ui/select.tsx` 側で現在値/placeholder を
  `aria-label` に自動フォールバックさせ、最低限「無名」状態を防いだ
  (呼び出し側の `htmlFor` 明示的関連付けは今回のスコープに含めない別の改善として残る)。
- **Base UI `Select.Item` はマウスクリック単体では選択が確定しない。** 内部の
  `allowMouseSelectionRef` は直前の `pointerdown` イベントでのみ `true` になる実装のため、
  `fireEvent.click()` だけを送るテストコードは無反応に見える(エラーも出ない)。
  実ブラウザでは通常のクリックで pointerdown→click の順にイベントが発火するため
  問題にならないが、テストコードでは `fireEvent.pointerDown()` を先に送る必要がある。
- **Menu/Select のトリガーはキーボードの Enter/Space を自前でハンドリングしない。**
  Radix は独自にキーボード処理を実装していたが、Base UI は `nativeButton` 前提で
  ブラウザのネイティブ挙動(フォーカス中の `<button>` への Enter/Space は自動的に
  click イベントを発火する)に委ねている。happy-dom はこのネイティブ挙動を
  再現しないため、`fireEvent.keyDown(trigger, { key: 'Enter' })` で開閉していた
  既存テストは `fireEvent.click(trigger)` に置き換える必要がある。

## Plan

1. `frontend/` で shadcn CLI を実行し、`button` / `dropdown-menu` / `label` / `select` の
   生成結果を確認する。生成された実際の API 差分を元に Design を必要なら更新する。
2. `components/ui/{button,dropdown-menu,label,select}.tsx` を生成結果ベースで確定させ、
   対応するユニットテストを更新する。
3. 呼び出し側の `asChild` 箇所を 1 件ずつ新しい API へ変換する。ファイル単位で区切り、
   変換後に対象ファイルだけ `just typecheck-ui` の差分を確認しながら進める。
4. `frontend/package.json` から `@radix-ui/*` を削除し、shadcn が導入した依存に揃える。
5. 全件変換後、`just verify-ui` と `just test-ui-e2e` を通す。
6. 新しい配色・見た目を主要画面(AdminShell/AccountShell/SystemShell のユーザーメニュー、
   フォームのラベル、一覧ページのリンクボタン)でブラウザ確認し、必要な微調整を行う。

## Tasks

- [x] T001 [App] `npx shadcn@latest init -b base` を実行し `components.json` を作成する。
      生成される Tailwind/CSS 変数の構成を確認する。
      → プリセットは当初「密なインターフェース向け」の `mira` を選んだが、レビューで
      ボタンが以前(`h-10`)より明確に小さくなった(`h-7`)ことを劣化と指摘され、
      標準的なサイズ感の `vega`(shadcn/ui の既定スタイル、デフォルトサイズ `h-9`、
      `lg` が旧デフォルトと同じ `h-10`)に切り替えた。`baseColor: neutral` は維持。
      `iconLibrary` は初期値 `hugeicons` から idmagic 全体の既存アイコンと揃える
      `tabler` へ変更。path alias(`@/*`)を新設(`tsconfig.json`/`tsconfig.app.json`/
      `vite.config.ts`、`@types/node` 追加)。
- [x] T002 [App] `npx shadcn add button dropdown-menu label select` でコンポーネントを
      生成し、実際の公開 API を確認して Design を更新した。
      → `Label` は独立プリミティブが無く `Field.Label` は不採用(想定通り)、素の
      `<label>`。`Button` は `render`/`nativeButton` を持つ。`DropdownMenu` は
      `Menu.Root/Trigger/Portal/Positioner/Popup/Item/GroupLabel/Separator` 構成。
      `Select` は `items` prop 必須の本格的な Base UI `Select`(role="combobox"/"option")。
- [x] T003 [App] 生成された `button.tsx` / `dropdown-menu.tsx` / `label.tsx` / `select.tsx`
      を idmagic の規約に合わせて調整し、ユニットテストを更新した。
      → `DropdownMenuLabel` は `Menu.GroupLabel`(`Menu.Group` 必須)を避け素の `<div>` に。
      `select.tsx` は生成された低レベル部品の上に、既存の簡易 `Select`(`SelectOption[]`
      API)を維持する形で再構築(呼び出し側 14 ファイルは無変更)。
- [x] T004 [App] 呼び出し側の `asChild` 箇所(実測 51 箇所・約 34 ファイル)を新しい API へ
      変換した。`DropdownMenuItem` の `onSelect`(Radix 専用 API、Base UI には無く
      `onClick` に統合)も全箇所 `onClick` に変換。
      → 途中、並列委任した3エージェントの1つが `just format-ui` 実行後に対象外への
      差分を `git checkout --` で戻す際、他2エージェントと自分の作業中ファイルまで
      巻き込んで既 asChild 状態へ巻き戻る事故が発生(共有作業ツリーでの並列実行が原因)。
      検出後、影響ファイルを再度手動で変換し直して収束させた。今後 asChild 系の
      大量変換を並列委任する場合は `isolation: "worktree"` を使うこと。
- [x] T005 [App] `frontend/package.json` の `@radix-ui/*` を除去し、`@base-ui/react` に
      揃えた。
- [x] T006 [App] デザイントークンは shadcn init の `neutral` ベース(黒/白/グレー基調)を
      そのまま採用。ブラウザでの視覚確認ができない環境だったため、これ以上の配色・
      ブランディング調整は行わず、[[wi-311-evaluate-base-ui-kumo-migration]] で合意した
      「デザイン刷新」の土台(Cloudflare 等どのブランドにも紐づかない中立トークン)を
      整えるところまでとした。具体的な配色・アクセントカラーの決定は T010 の
      フォローアップ WI 側の判断に委ねる。
- [x] T007 [Verify] `just verify-ui` を通した(format-check / lint / test-ui-unit / build
      すべてグリーン)。作業中に発見した無関係の既存 lint エラー
      (`src/api/admin.test.ts` の未使用 import)も併せて解消。
- [x] T008 [Verify] `just test-ui-e2e` を通した(4 spec ファイル・22 テスト全パス)。
      `tests/e2e/fixtures.ts` の `selectDropdownOption` を Base UI Select の
      `role="option"`(旧実装は DropdownMenu の `role="menuitem"` のみ対応)と
      トリガー検出のリトライに対応させる修正が必要だった。
- [ ] T009 [Verify] ブラウザでの目視確認は、このセッションでは Chrome 拡張が使えず
      未実施。`just test-ui-e2e` が実ブラウザ(WebView)でのログイン・フォーム入力・
      DropdownMenu 開閉・Select 操作を含む機能検証はカバーしているが、見た目の確認は
      別途ユーザー側で `just dev` を使って行うことを推奨する。
- [ ] T010 [提案] 完了後、`Dialog`/`Toast`/`Card`/`Alert`/`Input` を含む frontend 全体の
      デザイン刷新(配色・アクセントカラーの具体案含む)を別 WI として起票することを
      ユーザーに提案する。

## Verification

- `just verify-ui`
- `just test-ui-e2e`
- 対象画面の手動目視確認(ドロップダウンメニューの開閉・キーボード操作、フォームラベルの
  クリックでのフォーカス移譲、リンク化されたボタン、新配色の見た目)

## Risk Notes

- **risk を high とした理由**: Radix→Base UI の機械的な差し替えに加えて、shadcn CLI の
  実際の生成結果を見るまで最終的なコンポーネント API が確定しない(生成後に Design の
  一部を書き直す可能性がある)。さらにデザイン刷新([[wi-311-evaluate-base-ui-kumo-migration]]
  で合意済み)を伴うため、機械的な移行 WI より不確実性が高い。
- 呼び出し側 51 箇所への構造変更は正規表現一括置換ができず 1 箇所ずつの書き換えが
  必要なため、レビュー漏れによる挙動崩れ(フォーカストラップ・キーボード操作・
  ARIA 属性)のリスクがある。ファイル単位で変換 → typecheck 差分確認を繰り返し、
  最後に e2e とアクセシビリティ手動確認で軽減する。
- デザイン刷新により見た目が大きく変わるため、既存のスクリーンショット系 e2e や
  視覚回帰の前提があれば影響を受ける(現状 idmagic には視覚回帰テストは無い認識だが、
  着手時に再確認する)。
- Base UI 自体は 2025-12 に 1.0 を出した若いライブラリであり、今後も破壊的変更が
  入る可能性がある。バージョンを固定し、CHANGELOG 追随を継続する。

## Completion

- **Completed At**: 2026-08-01
- **Summary**:
  Radix UI(`@radix-ui/react-slot` / `react-dropdown-menu` / `react-label`)から
  shadcn/ui の Base UI 版へ移行した。`npx shadcn@latest init -b base`(プリセット
  `vega`(標準サイズ)、`baseColor: neutral`、`iconLibrary` を idmagic 既存の
  `tabler` に変更)で `components.json` を作成し、`button` / `dropdown-menu` /
  `label` / `select` を生成、idmagic の規約(既存の簡易 `Select` API 維持、`DropdownMenuLabel` を
  `Menu.GroupLabel` ではなく素の `<div>` に変更等)に合わせて調整した。
  呼び出し側の `asChild`(実測 51 箇所・約 34 ファイル)を `render`/`nativeButton`
  へ、`DropdownMenuItem` の `onSelect`(Radix 専用・Base UI には無く TypeScript でも
  検出できない静かな不具合になる)を全箇所 `onClick` へ変換した。
  `frontend/package.json` の `@radix-ui/*` を `@base-ui/react` に置き換えた。

  実装過程で当初の調査になかった追加の破壊的変更をいくつか発見した: Base UI
  Select の `role="combobox"` は content ベースの accessible name 算出をしないため
  `aria-label` フォールバックを追加、`Select.Item` はマウスクリック前に
  `pointerdown` が無いと選択が確定しない、Menu/Select のトリガーはキーボード
  Enter/Space をネイティブ `<button>` の挙動に委ねている(happy-dom では
  再現されないためテストは `fireEvent.click` に統一)。詳細は Design を参照。

  実装中、並列委任した3エージェントの1つが対象外ファイルへの意図しない差分を
  `git checkout --` で戻す際に他エージェント・自分自身の作業中ファイルまで
  巻き込んで一部が Radix 版へ巻き戻る事故が発生した(共有作業ツリーでの並列実行が
  原因)。検出後、影響ファイルをすべて手動で変換し直して収束させた。

  デザイントークンは shadcn init の `neutral` ベース(中立的な黒/白/グレー)を
  採用し、Cloudflare 等どのブランドにも紐づかない土台を整えた。それ以上の配色・
  アクセントカラーの決定、および `Dialog`/`Toast`/`Card`/`Alert`/`Input` を含む
  frontend 全体のデザイン刷新は、別 WI として起票することをユーザーに提案する
  (T010、[[wi-311-evaluate-base-ui-kumo-migration]] で合意した「デザイン刷新」の
  範囲は本 WI では Button/DropdownMenu/Label/Select の技術的な下地に留まる)。
- **Verification Results**:
  - `just verify-ui`(format-check / lint / test-ui-unit / build)- passed
    (作業中に発見した無関係の既存 lint エラー、`src/api/admin.test.ts` の
    未使用 import も併せて解消)
  - `just test-ui-e2e` - passed(4 spec ファイル・22 テスト、実ブラウザ WebView 上)。
    `tests/e2e/fixtures.ts` の `selectDropdownOption` を Base UI Select の
    `role="option"` とトリガー検出のリトライに対応させる修正が必要だった。
  - ブラウザでの目視確認(T009)は本セッションでは未実施(Chrome 拡張が利用不可)。
    `just test-ui-e2e` が実ブラウザでの機能面(ログイン・フォーム入力・
    DropdownMenu 開閉・Select 操作)を検証しているが、視覚的な仕上がりは
    ユーザー側で `just dev` を使って確認することを推奨する。
