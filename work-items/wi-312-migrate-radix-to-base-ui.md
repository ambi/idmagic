---
status: pending
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

- [ ] T001 [App] `npx shadcn@latest init -b base` を実行し `components.json` を作成する。
      生成される Tailwind/CSS 変数の構成を確認する。
- [ ] T002 [App] `npx shadcn add button dropdown-menu label select` でコンポーネントを
      生成し、実際の公開 API(`asChild`/`render` の扱い、`Label` の実装方式)を確認して
      Design を必要なら更新する。
- [ ] T003 [App] 生成された `button.tsx` / `dropdown-menu.tsx` / `label.tsx` /
      (必要なら)`select.tsx` を idmagic の規約に合わせて調整し、ユニットテストを更新する。
- [ ] T004 [App] 呼び出し側の `asChild` 箇所(実測 51 箇所・約 34 ファイル、着手前に
      再計測)を新しい API へ変換する(`Button` 単体、`DropdownMenuTrigger`/
      `DropdownMenuItem` が `Button`/`Link`/`button` をラップするケースを含む)。
- [ ] T005 [App] `frontend/package.json` の `@radix-ui/*` を除去し、shadcn 導入の依存に揃える。
- [ ] T006 [App] 新しい配色・デザイントークンを決定し適用する(具体案は実装時に確定)。
- [ ] T007 [Verify] `just verify-ui` を通す。
- [ ] T008 [Verify] `just test-ui-e2e` を通す。
- [ ] T009 [Verify] AdminShell/AccountShell/SystemShell のユーザーメニュー開閉・キーボード
      操作、フォームラベルのクリックでのフォーカス移譲、一覧ページのリンクボタンを
      新デザインでブラウザ目視確認する。
- [ ] T010 [提案] 完了後、`Dialog`/`Toast`/`Card`/`Alert`/`Input` を含む frontend 全体の
      デザイン刷新を別 WI として起票することをユーザーに提案する。

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
