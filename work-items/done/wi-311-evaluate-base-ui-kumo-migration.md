---
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-01
depends_on: []
---

# Radix UI + shadcn/ui から Base UI (+ Kumo) への移行要否を調査し、方針を決める

## Motivation

現在 `frontend/src/components/ui/` は shadcn/ui 由来の手書きコンポーネント（`components.json` を
使う shadcn CLI 生成ではなく、パターンだけを踏襲したコピー実装）で、`@radix-ui/react-slot`
(`Button` の `asChild`)、`@radix-ui/react-dropdown-menu`、`@radix-ui/react-label` の 3 パッケージに
依存している。2026 年に入り以下の外部変化があった:

- Radix UI は WorkOS に買収され、以降更新ペースが低下している（特に Combobox / 複雑な
  multi-select 系）。
- MUI チームが Radix 開発者と共同で Base UI を開発しており、2025-12 に 1.0.0 を出荷、
  2026-08 時点で `@base-ui/react@1.6.0` 相当・週間 600 万 DL 超まで成長している。
- shadcn/ui は 2026-07-03 付けで新規プロジェクトの既定プリミティブを Radix から Base UI に
  切り替えた。CLI に `--base radix|base` フラグと `migrate radix` コマンドがあり、
  2026-01 時点で全コンポーネントの Base UI 対応が完了している。
- Cloudflare が Base UI 上に自社デザインシステムを載せた `@cloudflare/kumo`
  （Tailwind v4 前提、37 の Base UI プリミティブを再エクスポート、CLI/Figma プラグイン/
  AI 可読レジストリ付き）を OSS で公開している。

「移行する価値があるか」を判断するための調査と、判断結果に応じた移行タスクの選定を行う。
本 WI は UI の見た目や振る舞い（ユーザーから見た挙動）を変えない実装詳細の変更であり、
SCL の interface / scenario には影響しない。

## Scope

- `frontend/package.json` の `@radix-ui/*` 依存
- `frontend/src/components/ui/button.tsx`（`asChild` / `Slot`）
- `frontend/src/components/ui/dropdown-menu.tsx`（`DropdownMenuPrimitive`）
- `frontend/src/components/ui/label.tsx`（`LabelPrimitive`）
- `frontend/src/components/ui/select.tsx`（`dropdown-menu.tsx` の上に自前実装、Radix Select 自体は未使用）
- 上記を呼び出す `frontend/src/features/**`、`frontend/src/components/{AdminShell,AccountShell,SystemShell,AdminPaneActions}.tsx`
  （`asChild` 呼び出しが実測 51 箇所・約 34 ファイル、`Label` 呼び出しが約 45 ファイル）
- SCL (`spec/scl.yaml`) は対象外（ユーザー可視の振る舞い・契約変更なし）

## Out of Scope

- `Dialog` / `AlertDialog` / `Toast` / `Card` / `Alert` / `Input` は現状 Radix 非依存の自前実装で、
  今回の Radix 依存除去とは無関係。Base UI 版に置き換えるかどうかは別 WI で検討する。
- Kumo のビジュアルデザイン（Cloudflare ブランドトークン・配色）をそのまま採用すること。

## Design

### 調査結果のまとめ

**現状の Radix 依存範囲は小さいが、呼び出し側への波及は広い。**
直接 import しているのは 3 ファイル・3 パッケージのみだが、`Button` の `asChild` は
実測 51 箇所・約 34 ファイル、`Label` は約 45 ファイルから使われている。Dialog 系コンポーネントは元々
Radix を使っておらず自前実装なので、Base UI 移行で Dialog/Toast 周りの書き換えは発生しない。

**Base UI への移行と Kumo 導入は別の意思決定である。**
- **Base UI（プリミティブ）への移行**: 今の `components/ui/*` は「Radix プリミティブを
  Tailwind でスタイリングする」薄いラッパーであり、この設計思想はそのまま Base UI に
  スライドできる。npm 依存の置き換え + `asChild → render` prop 変換 + `Overlay → Backdrop` /
  `Positioner/Popup` モデルへの追従が主な作業で、見た目やコンポーネント API
  （`Button`/`DropdownMenu`/`Label` の公開 props）は現状維持できる。
  メリットは Radix の開発停滞リスクの解消、shadcn エコシステムの今後のデフォルトとの
  整合、将来 Combobox/Autocomplete/NumberField が必要になった際に Radix には無い
  選択肢が増えること。
- **Kumo（デザインシステム）の導入**: Kumo は Base UI の上に Cloudflare 自身の
  デザイントークン・配色・コンポーネント群を載せた「完成品」であり、shadcn 的な
  「プリミティブを自分たちのスタイルで包む」薄いラッパーの発想とは異なる。
  idmagic は既に slate/blue 基調の独自 Tailwind テーマと日本語 UI 文言を持つ
  hand-rolled コンポーネント群を持っており、Kumo を丸ごと採用すると
  Cloudflare のブランド／トークン体系を上書きするか、あるいは二重にテーマ層を持つ
  ことになり、現状の資産（`buttonVariants` 等の cva 定義、既存の a11y/i18n 実装）を
  活かしにくい。Kumo のトークンが完全に再定義可能かは未検証。

**当初案**: Kumo を丸ごと導入するのではなく、Base UI プリミティブを自前でラップし直す
（＝現行の「shadcn 流に自分でラップする」設計を維持したまま下回りだけ差し替える）方針を
検討した。Kumo は導入コストとブランド上書きリスクに対して得られる価値が小さいと判断。
この評価はレビューで @tn から「デザインを大幅に変えてよい、互換性は気にしなくてよい」と
明確な合意を得たため、Kumo のブランド上書き問題を再検証した（下記「Kumo のブランド色に
関する追加調査」）。その結果を踏まえた最終方針は T003 を参照。

### T001 調査結果: Kumo のトークンは再定義前提の設計ではない

`@cloudflare/kumo`（v2.9.0）のソースを実際に取得して検証した。トークンは
`packages/kumo/src/styles/theme-kumo.css` に **AUTO-GENERATED FILE - DO NOT EDIT DIRECTLY**
として生成されており、生成元は非公開の内部スクリプト（`scripts/theme-generator/config.ts`、
npm 配布物には含まれない）。ブランド色は
`--text-color-kumo-brand: light-dark(#f6821f, #f6821f)` のように Cloudflare オレンジが
直接ハードコードされ、`--color-kumo-neutral-*` などのプリミティブ色も同様に固定値。
外部の消費者が任意のブランド色に再定義できる公開 API は無い（CSS 変数を丸ごと上書きする
force override は可能だが、それは「トークンシステムを使う」ではなく「トークンシステムを
無効化して独自 CSS を当てる」に等しく、Kumo を採用する意味がない）。**Kumo 導入は却下**。

### T002 調査結果: Base UI 直接移行は実機プロトタイプで概ね裏付けられたが、想定より詳細な差分がある

`@base-ui/react@1.6.0` を実際に `frontend/` へ追加し、`button.tsx` / `dropdown-menu.tsx` /
`label.tsx` / `select.tsx` を書き換え、呼び出し側を数件変換して `just typecheck-ui` /
`bun test src/components/ui/` を実行した（検証後、リポジトリは元の Radix 版に revert 済み）。

判明した点:

1. **DropdownMenu → Base UI `Menu` は Design 記載どおり機械的に移行できる。**
   `Menu.Root/Trigger/Portal/Positioner/Popup/Item` の構成で、`data-highlighted` /
   `data-disabled` 属性は Radix と同じ命名で出力されるため既存の Tailwind
   `data-[highlighted]:...` クラスはそのまま使える。`DropdownMenuItem` の `onSelect` は
   `onClick` に読み替える（`closeOnClick` は既定 `true` で Radix の自動クローズ挙動と同じ）。
   `Separator` は Menu 専用ではなく `@base-ui/react/separator` の汎用プリミティブを使う。
2. **Base UI には Radix `@radix-ui/react-label` に相当する独立した Label プリミティブが無い。**
   ラベル機能は `Field` コンポジット配下の `Field.Label` としてのみ提供され、
   `Field.Root` の祖先が無いと実行時に
   `Base UI: FieldRootContext is missing. Field parts must be placed within <Field.Root>.`
   で例外になる（実機のユニットテストで確認）。idmagic の `Label` は `Field` のバリデーション
   機能を使わない単純な `<label htmlFor>` としての利用のみなので、**`Field.Label` は採用せず、
   同じ Tailwind クラスを持つ素の `<label>` 要素にする**（Base UI 依存自体が不要になる）。
3. **`Button` の `render` prop は機能するが、非 `<button>` ターゲットには
   `nativeButton={false}` の明示が要る。** `render={<a href="..." />}` だけだと開発時に
   `Base UI: A component that acts as a button expected a native <button> because the
   nativeButton prop is true...` という警告が出る（実機テストで確認、ビルドや Test 自体は
   失敗しない）。既存呼び出し側 51 箇所（後述）のほとんどは `<a>` へのリンク化なので、
   変換時に `nativeButton={false}` を毎回セットする必要がある。
   （`asChild → render` の単純なプロパティ改名だけでは不十分）
4. **呼び出し側の実件数は 51 箇所・約 34 ファイル**（`grep -rn asChild` で再計測。
   内訳は `Button asChild` が大半、`DropdownMenuTrigger asChild` /
   `DropdownMenuItem asChild` で `Button` や `Link` をラップする箇所が
   `AdminShell` / `AccountShell` / `SystemShell` /
   `AdminGroupDetailPage` / `AdminAgentDetailPage` / `AdminUserDetailPage` に点在）。
   `<Button asChild><X>children</X></Button>` → `<Button render={<X />}>children</Button>`
   という構造変更（子要素の props を `render` へ、子要素の children を親の children へ移す）
   になるため、正規表現一括置換ではなく 1 箇所ずつの書き換えが必要。

### Kumo のブランド色に関する追加調査、および最終方針の転換

レビューで「ハードコードされたブランド色は Kumo の GitHub issue で問題視されていないのか」
という指摘を受け、`cloudflare/kumo` の issue を `gh api search/issues` で
`theme` / `brand color` / `rebrand` / `white-label` / `"not intended"` 等のクエリで
検索したが、再テーマ／外部ブランド利用を求める issue は見つからなかった。理由は README
の一文に表れている: `Kumo — Cloudflare's component library for building modern web
applications.` であり、Shopify Polaris や GitHub Primer、Atlassian Design System と
同種の「自社製品用の内製デザインシステムを OSS 公開した」プロジェクトであって、
汎用の白ラベル UI キットを標榜していない。この種のライブラリでは「他社ブランドに
差し替えにくい」ことは不具合ではなく前提であり、コミュニティもそう受け止めている。

さらに調査を深めると、ブランド色の浸透度は想定より深刻だった。
`--color-kumo-brand`（`#f6821f`、Cloudflare オレンジ）は単なる装飾アクセントではなく:

- `Button` の `primary` variant の背景色
- **全 variant 共通の focus ring 色**（`focus-visible:ring-kumo-brand`、ソース確認済み)
- Badge / Checkbox / Dropdown / Radio / Select / Switch / Tabs / Toolbar など
  19 コンポーネントで参照

という形で全体に浸透しており、加えて `CloudflareLogo` / `PoweredByCloudflare` という
Cloudflare 自身のブランド用コンポーネントまで同梱されている。つまり Kumo をそのまま
採用すると「配色が違う」以上に「Cloudflare のブランドアイデンティティを丸ごと被る」
ことになる。CSS カスタムプロパティなので技術的に `--color-kumo-brand` だけ後勝ちで
上書きすることは可能だが、Kumo 側が公式サポートする再テーマ経路ではないため、
Kumo のバージョンアップで新しいコンポーネントが追加されるたびに同様の上書きが
必要か確認し続ける保守コストが発生する。

この点をユーザーに確認したところ、「デザイン刷新は歓迎、互換性は気にしなくてよいが、
Cloudflare のブランド色をそのまま被るのは避けたい」という意向のもと、
**Kumo（コンポーネント一式込み）ではなく、shadcn/ui の Base UI 版を採用する**方針で
合意した。shadcn/ui は npm パッケージ依存ではなく CLI (`shadcn@4.16.1`、
`npx shadcn@latest init -b base`) でコンポーネントのソースコードを
`components.json` の設定に従って直接リポジトリへ生成する方式であり:

- CSS 変数（`--primary` 等）でテーマを持つため、生成されたソースは**自分たちの
  デザイントークンをゼロから決められる**。Cloudflare や他社ブランドに紐づく色は
  一切含まれない。
- `-b base` で Base UI を採用する版が公式に提供されている（2026-07 付けで
  shadcn/ui の新規プロジェクト既定が Radix から Base UI に切り替わっている）。
- `npx shadcn add button dialog dropdown-menu ...` で必要なコンポーネントだけを
  都度生成でき、Kumo よりコンポーネント網羅性が高い（Dialog / Toast(Sonner) /
  Combobox / Command / DataTable 等、idmagic の自前実装や不足部分も置き換え・
  拡充できる）。
- 生成されたソースはリポジトリ内の通常のコードなので、Kumo のような「非公開の
  内部生成スクリプトでしか変更できないトークン」問題が発生しない。

### 移行時の主要な破壊的変更（Base UI 採用時、確定）

- `asChild` prop → `render` prop への構造変更（呼び出し側 51 箇所）。非 `<button>` ターゲットには
  追加で `nativeButton={false}` の明示が要る。
- `DropdownMenuPrimitive.Content` は `Portal/Positioner/Popup` 3 層構造に分解される。
  `align` は `Positioner` 側のプロパティになる。
- `Label` は Base UI の `Field.Label` を使わず、素の `<label>` 要素として実装する
  （Base UI 依存を持たせない）。
- Select は Radix Select を使っていないため直接の影響はなく、`DropdownMenuItem` の
  `onSelect → onClick` 読み替えのみ必要。

## Plan

1. 調査タスク（T001-T003）で Kumo のトークン再定義可否・Base UI 単体導入の実作業量を
   実機で確認し、上記推奨方針を確定させる。（完了）
2. 実際の移行作業（Radix 依存の置換、呼び出し側 51 箇所の書き換え、検証）は
   [[wi-312-migrate-radix-to-base-ui]] で行う。本 WI は調査と意思決定に閉じる。

## Tasks

- [x] T001 [調査] Kumo のデザイントークンが idmagic の既存 Tailwind テーマ（slate/blue 基調）に
      合わせて再定義可能か、実際に `@cloudflare/kumo` のソースを取得して確認する。
      → 再定義前提の設計ではない（Design 参照）。
- [x] T002 [調査] Base UI 単体導入（Kumo なし）で `button.tsx` / `dropdown-menu.tsx` /
      `label.tsx` / `select.tsx` を実際に書き換えて試作し、`just typecheck-ui` /
      `bun test src/components/ui/` で差分規模と挙動を確認した（検証後 revert 済み）。
      → 機械的だが `Field.Label` 不採用・`nativeButton={false}` 要件など詳細差分あり（Design 参照）。
- [x] T003 [決定] **shadcn/ui の Base UI 版（`shadcn init -b base` によるソース生成方式）を
      採用する。Kumo は不採用（Cloudflare ブランド色が全コンポーネントに浸透しており、
      デザイン刷新を歓迎する今回の方針とも合わない）。** デザインは大幅刷新を前提とし、
      現行の slate/blue 配色との互換性は求めない。`Label` を含め、生成されたコンポーネントの
      構造にそのまま従う（Base UI を自前ラップし直す当初案は不採用）。実装は
      [[wi-312-migrate-radix-to-base-ui]] に分割する（コンポーネント生成・呼び出し側
      51 箇所以上の書き換え・デザイン刷新に及ぶ実装作業のため、調査 WI とは別単位で
      実装前に Plan/Tasks を持たせる）。

## Verification

- `@cloudflare/kumo` の GitHub リポジトリから `theme-kumo.css` / `kumo-binding.css` を
  直接取得し、トークン定義を確認した。
- `@base-ui/react@1.6.0` を `frontend/` に一時追加し、`button.tsx` / `dropdown-menu.tsx` /
  `label.tsx` / `select.tsx` を書き換え、呼び出し側を数件変換した上で
  `just typecheck-ui` と `bun test src/components/ui/` を実行し、実際のエラー内容・件数を
  確認した。検証後、`git checkout -- frontend/` で Radix 版に revert し、`@base-ui/react` も
  `node_modules` から除去して `bun install` で依存を復元。`just verify-ui` がグリーンな
  状態（変更前と同一）であることを確認済み。

## Risk Notes

- Base UI 自体は 2025-12 に 1.0 を出した若いライブラリであり、今後も破壊的変更が入る
  可能性がある。実装 WI ([[wi-312-migrate-radix-to-base-ui]]) でバージョン固定と
  CHANGELOG 追随を方針に含める。
- 呼び出し側 51 箇所への広範な構造変更（`asChild → render` + `nativeButton={false}`）は
  正規表現一括置換ができず 1 箇所ずつの書き換えが要るため、レビュー漏れによる挙動崩れ
  （フォーカストラップ・キーボード操作・ARIA 属性）のリスクがある。実装 WI 側で
  e2e とアクセシビリティ手動確認による軽減を計画する。

## Completion

- **Completed At**: 2026-08-01
- **Summary**:
  Radix UI + shadcn/ui から Base UI (+ Kumo) への移行要否を実機検証した。
  `@cloudflare/kumo` のソースを取得した結果、デザイントークンは非公開の内部生成
  スクリプトによる Cloudflare ブランド色ハードコードで、外部からの再定義を前提とした
  設計ではないため当初は Kumo を却下し「Base UI を自前でラップし直す」方針を検討した。
  `@base-ui/react` を実際に組み込み `button.tsx` / `dropdown-menu.tsx` / `label.tsx` /
  `select.tsx` を試作した結果、DropdownMenu の Base UI `Menu` への移行は機械的に可能だが、
  (1) Base UI には独立した Label プリミティブが無く `Field.Label` は `Field.Root` 祖先
  必須、(2) `Button` の `asChild → render` 変換は非 `<button>` ターゲットに
  `nativeButton={false}` の追加指定が要る、(3) 呼び出し側の実件数は 51 箇所・約 34
  ファイルで構造変更が必要、という当初想定より詳細な差分があることも判明した。
  レビューで「Kumo のブランド色ハードコードは GitHub issue で問題視されていないか」との
  指摘を受けて追加調査したところ、Kumo は Shopify Polaris 等と同種の「自社製品用
  内製デザインシステムの OSS 公開」であり外部ブランド利用は元々想定外と判明。同時に
  ユーザーから「デザイン刷新は歓迎、互換性は不要だが Cloudflare のブランド色を
  そのまま被るのは避けたい」との意向を確認し、**最終方針を shadcn/ui の Base UI 版
  （`shadcn init -b base` によるソース生成方式）の採用に転換した**。これにより
  Kumo のブランド上書き問題も、Base UI 自前ラップの保守負担も回避しつつ、
  Dialog/Toast/Combobox 等の不足コンポーネントも同じ枠組みで拡充できる。
  この決定に基づく実装は [[wi-312-migrate-radix-to-base-ui]] に分割した。
  プロトタイプ変更は検証後に `git checkout` で元の Radix 実装へ revert 済みで、
  本 WI 自体はコード変更を残さない。
- **Verification Results**:
  - `just verify-ui` - passed (revert 後、変更前と同一状態であることを確認)
  - `just typecheck-ui` / `bun test src/components/ui/` - Base UI プロトタイプに対して実行し、
    上記の差分点を実機確認（revert 済みのため最終状態への寄与はなし、調査記録として使用）
