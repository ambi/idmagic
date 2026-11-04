---
status: pending
authors: [tn]
risk: low
created_at: 2026-11-05
priority: p1
depends_on: [wi-62545-accept-client-assertions-without-client-id]
change_kind: docs
spec_impact: { kind: none, reason: "standards.md の書き方の規則を足し、既存の規範の行、要件、API の振る舞いを変えない。" }
---

# 標準の宣言に、相手が省略または追加できる要素の受け入れを書かせる

## 動機

[[wi-62545-accept-client-assertions-without-client-id]] で、IdMagic は RFC 7521 §4.2 が OPTIONAL とする `client_id` を必須にしていた。
規範の文書と試験の両方がこの違反を検出できなかった原因は三つある。

- [標準仕様](../../docs/formats/specification-format.md#標準仕様)の規則は、`Statement` に「製品が実行すること、または拒否すること」を書かせる。サーバーとしての製品では検査の規則が行になり、相手が省いてよいものを受け入れる規則は行になりにくい。RFC 7523 は検証の 1 行だけだった。
- 採用した標準の土台になる標準を宣言する規則がない。`client_id` の規定は、RFC 7523 が使う枠組みの RFC 7521 にあり、RFC 7521 は宣言されていなかった。
- 被覆の検査は「ID をテストが参照すること」だけを見る。試験の要求は自分で組み立てていて `client_id` を常に付けたので、省いた要求は一度も送られなかった。

## 対象範囲

- [標準仕様](../../docs/formats/specification-format.md#標準仕様)へ次の規則を足す。
  - 製品が標準を受ける側のとき、相手が省略できる要素、相手が追加できる要素、複数の形を取れる要素について、製品が受け入れることを行にする。
  - 採用する標準が別の標準の枠組みや定義に依存し、その標準が製品の受け入れに規則を課すとき、その標準も節として宣言する。
  - 受け入れの行を引く試験は、相手がその要素を省略または追加した要求を、製品の入口へ送る。
- 書き方の例を、RFC 7521 の `RFC7521-CLIENT-ID-OPTIONAL` に揃える。

## 対象外

- 既存の `standards.md` の棚卸しと、見つかった違反の修正。[[wi-77372-inventory-standards-acceptance-rules]] が扱う。
- 受け入れの行を機械で見分ける列や検査の追加。棚卸しで行の数と形が分かってから判断する。今は `Statement` の書き方と試験の規則だけで足りるかを確かめる。
- 外部の適合性試験と第三者の実装との相互運用試験。[[wi-33-protocol-conformance-ci]]、[[wi-38172-verify-id-jag-interoperability-with-an-external-idp]]、[[wi-36722-publish-an-end-to-end-mcp-enforcement-walkthrough]] が扱う。

## 設計

`Adoption` と `Strength` の列は変えない。
受け入れの行も、製品が実行すること（受け入れること）を書く行であり、既存の列の意味で表せるためである。
列を足す案は採らない。
既存の 93 行を含む全ての行に値を求めることになり、棚卸しの前に形を固定してしまう。

試験の規則は、既存の「すべての ID をテストから参照する」に一文を足す形にする。
参照の検査は機械が行い、要求の形が相手の形になっているかはレビューで読む。

## 計画

1. 規則を書き、`RFC7521-CLIENT-ID-OPTIONAL` が規則どおりの例になっていることを確かめる。
2. 文書の検査を通す。

## タスク

- [ ] T001 [Docs] 標準仕様の規則に、受け入れの行、土台の標準、受け入れの行の試験の三つを足す。
- [ ] T002 [Verify] 文書の検査を通す。

## 検証

- `mise run check-links`：文書へのリンクとアンカーが壊れていないことを確かめるため。
- `mise run check-spec`：形式の規則の文書を読む検査が通ることを確かめるため。
- `mise run check-work-items`。

## リスク

規則が抽象的だと、次の宣言でも検査の行だけが書かれる。
規則に具体例を添え、棚卸しの作業項目で規則が使えるかを確かめる。
