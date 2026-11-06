---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-11-07
priority: p3
depends_on: []
change_kind: maintenance
spec_impact: { kind: none, reason: "画面の操作、表示する情報、文言、検証とエラー表示を変えない。英数字の書体と、select と dropdown menu の開閉の動きだけを変える。" }
---

# Inter、tw-animate-css、clsx の直接依存を外す

## 動機

[ライブラリ選定の見直し](../done/wi-90339-review-library-adoption.md)で、フロントエンドの三つの直接依存を外すと決めた。

- `@fontsource-variable/inter` 5.3.0：機能上の必要がない静的資産である。日本語は既に OS のフォントで表示しており、変わるのは英数字の書体だけになる。
- `tw-animate-css` 1.4.0：アニメーションを使うのは `select.tsx` と `dropdown-menu.tsx` だけで、dialog や combobox は動かない。消せば UI 全体の動きが揃い、[UI 設計](../../docs/design/application/user-interface.md)が求める動きを減らす設定への対応も不要になる。
- `clsx` 2.1.1：`cn()` の 63 か所の呼び出しは、文字列、`&&`、三項演算子だけを渡す。`tailwind-merge` の `twMerge` はこれらを直接受け取るため、出力を変えずに外せる。

`tailwind-merge` は、利用側の `className` で部品のデフォルトのクラスを上書きする設計に必要なので維持する。
clsx は class-variance-authority 0.7.1 の依存として lockfile に残る。

## 対象範囲

- `styles.css` の Inter の import と `--font-sans` の先頭の `"Inter Variable"` を外す。
- `styles.css` の tw-animate-css の import と、select と dropdown menu の `animate-*`、`fade-*`、`zoom-*`、`slide-*` のクラスを外す。
- `cn()` を `twMerge(...inputs)` にし、`ClassValue` 型の代わりに tailwind-merge の型を使う。
- 三つを `package.json` から外す。

## 対象外

- CVA と tailwind-merge の置き換え。
- 書体と配色の再設計。

## 設計

Base UI の popup は、閉じるときの動きが終わるまで要素を残す。
動きのクラスを消すと即座に外れるため、開閉、フォーカスの戻り、途中で閉じた場合の表示をブラウザーで確かめる。

## 計画

1. `cn()` の出力が変わらないことを単体テストで確かめてから clsx を外す。
2. 書体と動きを外し、E2E とブラウザーで確かめる。

## タスク

- [ ] T001 [UI] `cn()` から clsx を外す。
- [ ] T002 [UI] Inter と tw-animate-css を外す。
- [ ] T003 [Verify] ブラウザーで開閉とフォーカスを確かめる。

## 検証

- `mise run test-ui-unit` と `mise run lint-ui` を通す。
- `mise run check-ui-dependencies` で依存を確かめる。
- `mise run test-ui-e2e` で select と dropdown menu の操作を確かめる。

## リスク

Base UI の閉じる処理が動きの終了を待つ前提で書かれている場合、表示が残る可能性がある。
ブラウザーで確かめる。
