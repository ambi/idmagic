---
status: pending
authors: [tn]
risk: low
created_at: 2026-11-05
priority: p3
depends_on: []
change_kind: docs
spec_impact: { kind: none, reason: "実装が既に受け入れている wsignin1.0 の省略を標準要件と試験に加える作業であり、製品の振る舞いと既存の規範の結果は変えない。" }
---

# WS-Federation の `wsignin1.0` で省略できる要素の受け入れを宣言と試験に加える

## 動機

WS-Federation 1.2 の受動的な要求では、`wreply`、`wctx`、`wfresh`、`whr` を省略できる。
`resolveReplyURL` は、`wreply` を省いた要求を最初の許可済みの返信先で受け入れる。
`WSFED-PASSIVE-SIGN-IN` は、登録済みの `wtrealm` と許可済みの `wreply` にだけ返すという検査を書くが、省いたときの結果を書いていない。
[[wi-77372-inventory-standards-acceptance-rules]] の棚卸しで見つけた。

## 対象範囲

- `standards.md` に、`wreply`、`wctx`、`wfresh` を省いた要求の結果を書いた受け入れの標準要件を加える。
- 省いた要求を受動的な入口へ送る試験を加えるか、既存の試験から標準要件を参照する。

## 対象外

- WS-Trust の能動的な STS。`MSAL-WSTRUST-RST` が MSAL の要求の形を宣言している。

## タスク

- [ ] T001 [Spec] 受け入れの標準要件を加える。
- [ ] T002 [Test] 省いた要求の試験を加えるか、既存の試験から参照する。
- [ ] T003 [Verify] `mise run check-spec` と変更した試験を通す。

## 検証

- `mise run check-spec`：加えた標準要件の形式と参照を確かめるため。

## リスク

試験で受け入れられないと分かった要素は、この項目で直さず bugfix として起票する。
