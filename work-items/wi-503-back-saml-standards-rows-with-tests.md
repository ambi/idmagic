---
depends_on: []
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-06
priority: p2
change_kind: maintenance
spec_impact: { kind: none, reason: "宣言済みの標準行に、その id を名指しするテストを対応付ける作業である。standards.md の行そのものも製品の振る舞いも変えない。テストが書けない行が見つかった場合、それは製品が宣言した採用を満たしていないということなので、欠陥として個別の work item に切り出す。" }
---

# SAML が宣言する標準 6 行にテストを対応付け、標準の被覆台帳から外す

## Motivation

[[wi-495-burn-down-the-standards-coverage-debt]] は標準の被覆台帳へ受入集合を入れ、消化の単位を所有文書と決めた。本項目はそのうち `docs/contexts/saml/standards.md` の 6 行を引き取る。この文書は 7 行のうち 6 行が名指しを持たない。

SAML の行 id は `SAML2Core-BearerAssertion` のように大文字と小文字が混じる。[[wi-418-normative-coverage-gates]] の当初の実装は「ハイフンで繋いだ大文字の並び」を id の形と決め打ちしていたため、SAML と WS-Federation の 13 行はどれほど明白に名指されても被覆と認められなかった。この形の推測は既に取り除かれ、宣言された id そのものを探すようになっている。したがって本項目の消化は、名指しさえ書けば必ず届く。

## Scope

- 次の 6 行を消化する。

| ID | Adoption |
|---|---|
| `SAML2Profile-WebBrowserSSO` | required |
| `SAML2Bindings-RedirectPost` | required |
| `SAML2Metadata-IDPSSODescriptor` | required |
| `SAML2Metadata-WantAuthnRequestsSigned` | optional |
| `SAML2Core-EncryptedAssertion` | excluded |
| `SAML2Profile-ECP` | excluded |

- 1 行ごとに、その行の `Statement` を区別できる入力と観測を持つテストを対応付け、`// <ID>: <この行の何を固定しているか>` の注記を足す。
- 観測の形は行の `Adoption` に従う。型は [[wi-495-burn-down-the-standards-coverage-debt]] の Design が定める。
- 消化した id を `tools/check/standards-coverage-debt.json` から外す。

## Out of Scope

- 他の文書が宣言する標準行。文書ごとに別の work item が持つ。
- `standards.md` の行の追加、削除、`Adoption` および `Strength` の変更。
- 見つかった実装の欠陥の修正。欠陥として切り出した先で扱う。
- 受入集合の導入と、台帳ファイルそのものの削除。親の [[wi-495-burn-down-the-standards-coverage-debt]] が持つ。
- 具体例の被覆負債。[[wi-496-burn-down-the-example-coverage-debt]] が持つ。
- SAML の適合範囲そのものの拡張。`excluded` の 2 行を提供へ変えるのは規範の変更であり、別の work item が扱う。
- id を名指しする文字列があるだけを根拠にした台帳からの削除。

## Design

6 行のうち 2 行が `excluded` である。台帳全体で `excluded` は 15 行あり、そのうち 2 行がここにある。**`excluded` の観測は `required` と逆向きになる。** 行の `Statement` は製品の振る舞いではなく標準側の機能を書いているので、テストはそれを満たすことではなく満たさないことを観測する。

`SAML2Core-EncryptedAssertion` は暗号化アサーションを提供しないという行である。観測は、暗号化アサーションを要求または提示されたときに製品が何をするかである。拒否するなら、その拒否と、拒否が防いだ効果を観測する。単に提供していないだけなら、その機能への入口が存在しないことを観測する。どちらであるかは実装を読んで決める。

`SAML2Profile-ECP` は ECP プロファイルを提供しないという行である。ECP は `PAOS` バインディングを使うので、入口の有無は要求の形で区別できる。

`SAML2Metadata-WantAuthnRequestsSigned` は `optional` である。提供しているならその振る舞い、つまりメタデータに `WantAuthnRequestsSigned` を出し、署名の無い `AuthnRequest` をどう扱うかを観測する。提供していないなら行の `Adoption` が誤っているので規範の変更として切り出す。

`required` の 3 行は、宣言した振る舞いが製品の正式な入口から到達できることを観測する。`SAML2Bindings-RedirectPost` は Redirect と POST の 2 つのバインディングを名指しているので、片方だけを観測しても行を区別できない。両方を観測する。

## Plan

1. `SAML2Profile-WebBrowserSSO` から着手し、`required` の観測の型を決める。
2. `SAML2Bindings-RedirectPost` を、Redirect と POST の両方で消化する。
3. `SAML2Metadata-IDPSSODescriptor` を、生成されたメタデータを読む形で消化する。
4. `SAML2Core-EncryptedAssertion` の実装を読み、`excluded` の観測の型をここで決める。
5. その型で `SAML2Profile-ECP` を消化する。
6. `SAML2Metadata-WantAuthnRequestsSigned` の実装の有無を確かめ、観測するか切り出すかを決める。
7. 解決した id を台帳から外す。

## Tasks

- [ ] T001 [Acceptance] `SAML2Profile-WebBrowserSSO` を消化し、`required` の型を決める。
- [ ] T002 [Acceptance] `SAML2Bindings-RedirectPost` を、Redirect と POST の両方で消化する。
- [ ] T003 [Acceptance] `SAML2Metadata-IDPSSODescriptor` を、生成されたメタデータから消化する。
- [ ] T004 [Type] `SAML2Core-EncryptedAssertion` で `excluded` の観測の型を決め、消化する。
- [ ] T005 [Acceptance] `SAML2Profile-ECP` を同じ型で消化する。
- [ ] T006 [Inventory] `SAML2Metadata-WantAuthnRequestsSigned` の実装の有無を確かめ、観測するか切り出すかを決める。
- [ ] T007 [Ledger] 解決した id を `standards-coverage-debt.json` から外す。
- [ ] T008 [Verify] `mise run verify`。

## Verification

- 本項目が持つ 6 件が `tools/check/standards-coverage-debt.json` から 1 件残らず消えている。
- 注記を足した各テストについて、対応する production の判断を崩すとそのテストが落ちる。
- `mise run check-spec`
- `mise run verify`

## Risk Notes

- **`excluded` に書けるテストが無い。** 製品がその機能をそもそも実装していないなら、観測できるのは「入口が存在しない」ことだけになりうる。この場合に何を観測とするかは T004 で決める。決められない行は、台帳へ残す理由を `present when the check was introduced` から具体的な理由へ書き換えたうえで残す。理由が更新されていれば、判断済みであることが後から読める。
- **`SAML2Bindings-RedirectPost` を片方のバインディングだけで消化する。** 片方だけでは、もう片方を提供していない実装と区別できない。両方を観測する。
- **台帳の同時編集。** 自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。
