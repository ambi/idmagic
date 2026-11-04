---
status: pending
authors: [tn]
risk: low
created_at: 2026-11-05
priority: p3
depends_on: []
change_kind: docs
spec_impact: { kind: none, reason: "実装が既に受け入れている AuthnRequest の省略を宣言の行と試験に加える作業であり、製品の振る舞いと既存の規範の結果は変えない。" }
---

# SAML の AuthnRequest で省略できる要素の受け入れを宣言と試験に加える

## 動機

SAML 2.0 Core §3.4.1 は、AuthnRequest の `Destination`、`AssertionConsumerServiceURL`、`ProtocolBinding`、`NameIDPolicy` を省略できるとする。
`ValidateSignIn` は、省いた要求を SP の既定の ACS URL と、SP の既定の NameID 形式で受け入れる。
`NameIDPolicy` の `unspecified` も既定として扱う。
宣言の行（`SAML2Core-BearerAssertion`、`SAML2Profile-WebBrowserSSO`）は検査と拒否の項目を並べるが、この受け入れを書いていない。
[[wi-77372-inventory-standards-acceptance-rules]] の棚卸しで見つけた。

## 対象範囲

- `standards.md` に、各要素を省いた要求の結果を書いた受け入れの行を加える。
- 各要素を省いた AuthnRequest を SSO の入口へ送る試験を加え、行を参照する。既存の試験が省いた形を送っていれば、その試験から行を参照する。

## 対象外

- 拒否を宣言した要素（ACS のインデックス、HTTP-POST 以外の `ProtocolBinding`）。

## タスク

- [ ] T001 [Spec] 受け入れの行を加える。
- [ ] T002 [Test] 省いた要求の試験を加えるか、既存の試験から参照する。
- [ ] T003 [Verify] `mise run check-spec` と変更した試験を通す。

## 検証

- `mise run check-spec`：加えた行の形式と参照を確かめるため。

## リスク

試験で受け入れられないと分かった要素は、この項目で直さず bugfix として起票する。
