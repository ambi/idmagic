---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-10
priority: p3
depends_on: [wi-92970-adopt-information-hiding-modular-design]
change_kind: docs
spec_impact: { kind: none, reason: "文書の散文とコメントの語を改名し、用語の検査に規則を足すだけである。規範 ID、仕様と TypeSpec のディレクトリ名と名前空間、HTTP の応答、永続状態、ドメインイベントは変えない。" }
---

# 残りの文書の Context をモジュールへ改名し、用語の検査で固定する

## 動機

モジュール設計の項目は、概念を定義する文書と、その項目で変えた検査の識別子と出力だけを改名した。
方針の変更と機械的な置換を同じ差分に混ぜると、方針の変更をレビューで読み分けられなくなるからである。
その結果、ほかの文書、コードのコメント、各モジュールの仕様の散文には「Bounded Context」「Context Map」「Context 間」「公開言語」などの語が残り、「モジュール」と混在している。
語が混在すると、読み手は二つの語が同じものを指すのかを毎回判断しなければならない。

## 対象範囲

- 着手時に `rg` で残りの語を検索し、文書、コードのコメント、検査の出力文言を改名する。
  語の対応は[モジュール設計の項目](done/wi-92970-adopt-information-hiding-modular-design.md)の「用語の改名」の表に従う。
- 改名の完了後に、`mise run check-terminology` へ「Bounded Context」「Context Map」「Context 間」と、区分の意味の「Subdomain」「サブドメイン」を採らない表記として追加する。

## 対象外

- 外部文献のタイトル、C4 の System Context、Go の `context.Context`、DNS のサブドメインなど区分ではない意味の語。
- 固定したパスと規範 ID（`docs/domain/<名前>/`、`spec/contexts/<名前>/`、`REQ-<名前>-NNN`）と TypeSpec の名前空間。
- 完了済みの作業記録とリリース文書。

## タスク

- [ ] T001 [Inventory] 残りの語を検索し、改名の対象と除外を分ける。
- [ ] T002 [Docs] 文書とコメントを改名する。
- [ ] T003 [Tooling] 用語の検査に規則を追加し、除外が誤検出されないことを確かめる。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-terminology`
- `mise run verify`

## リスク

- 機械的な置換が、区分ではない意味の語や固定した識別子まで変えるおそれがある。
  置換は文脈を確かめながら行い、検査の規則には除外を明示する。
