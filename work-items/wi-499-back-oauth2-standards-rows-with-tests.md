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

# OAuth2 が宣言する標準 80 行にテストを対応付け、標準の被覆台帳から外す

## Motivation

[[wi-495-burn-down-the-standards-coverage-debt]] は標準の被覆台帳へ受入集合を入れ、消化の単位を所有文書と決めた。本項目はそのうち `docs/contexts/oauth2/standards.md` の 80 行を引き取る。

80 行は台帳全体 130 件の 6 割であり、他のどの文書よりひと桁多い。この文書だけが取り残されると、台帳は残り 8 文書を消化しても空にならず、[[wi-495-burn-down-the-standards-coverage-debt]] の最後の台帳削除が実行できない。

80 行は 33 の標準節にまたがる。採用の内訳は `required` 50、`optional` 20、`excluded` 9、`partial` 1 である。`optional` と `excluded` で 29 行、つまり 4 割弱が「提供していること」以外を観測する行である。

## Scope

- `docs/contexts/oauth2/standards.md` が宣言し、`tools/check/standards-coverage-debt.json` に載る 80 行を消化する。
- 1 行ごとに、その行の `Statement` を区別できる入力と観測を持つテストを対応付け、`// <ID>: <この行の何を固定しているか>` の注記を足す。
- 観測の形は行の `Adoption` に従う。型は [[wi-495-burn-down-the-standards-coverage-debt]] の Design が定める。
- 消化した id を `tools/check/standards-coverage-debt.json` から外す。
- 最初の標準節を通しで消化した時点で、本項目をさらに分割するかを決めて本節へ書く。

## Out of Scope

- 他の文書が宣言する標準行。文書ごとに別の work item が持つ。
- `standards.md` の行の追加、削除、`Adoption` および `Strength` の変更。行の内容が現状と食い違うと判明した場合は規範の変更であり、別の work item が扱う。
- 見つかった実装の欠陥の修正。欠陥として切り出した先で扱う。
- 受入集合の導入と、台帳ファイルそのものの削除。親の [[wi-495-burn-down-the-standards-coverage-debt]] が持つ。
- 具体例の被覆負債。[[wi-496-burn-down-the-example-coverage-debt]] が持つ。
- id を名指しする文字列があるだけを根拠にした台帳からの削除。

## Design

80 行を 1 度に読むことはできないので、標準節を単位に進める。節は標準そのものの単位であり、1 つの RFC を 1 度読む文脈で連続した行を判断できる。

節ごとの負債の分布は次のとおりである（2026-09-06 実測）。

| 節 | 負債 |
|---|---:|
| OAuth Client ID Metadata Document | 7 |
| OpenID Connect Client-Initiated Backchannel Authentication Flow Core 1.0 | 6 |
| The OAuth 2.0 Authorization Framework | 4 |
| OAuth 2.0 Token Exchange | 4 |
| OAuth 2.0 Protected Resource Metadata | 4 |
| FAPI 2.0 Security Profile | 4 |
| Best Current Practice for OAuth 2.0 Security | 4 |
| 3 行の節が 6 つ | 18 |
| 2 行の節が 12 個 | 24 |
| 1 行の節が 9 個 | 9 |

`optional` の 20 行は、提供しているならその振る舞いを観測する。提供していないなら、その行の `Adoption` が誤っているということなので、規範の変更として切り出す。この判断が 20 回出てくるため、`optional` の 1 件目で「提供している」と言える根拠の形を決め、残りへ広げる。

`excluded` の 9 行は、提供していないことを観測する。要求が届いたときに製品が拒否するなら、その拒否と、拒否が防いだ効果を観測する。書き方が `required` と逆になるため、1 件目で型を決めてから残りへ広げる。

**本項目をさらに分割するかは未決である。** 最も大きい `OAuth Client ID Metadata Document` の 7 行を通しで消化し、1 行あたりの所要を測ってから決める。測る前に分割の粒度を決めない。

## Plan

1. `OAuth Client ID Metadata Document` の 7 行を通しで消化し、1 行あたりの所要を記録する。
2. 記録をもとに、残る 73 行を本項目で続けるか子 work item へ割るかを決め、本節へ書く。
3. `optional` の 1 件目で「提供している」と言える根拠の形を決める。
4. `excluded` の 1 件目で観測の型を決める。
5. 節ごとに消化し、解決した id を台帳から外す。
6. 宣言した採用を満たしていない行が見つかったら、欠陥の work item を切り出す。

## Tasks

- [ ] T001 [Baseline] `OAuth Client ID Metadata Document` の 7 行を消化し、1 行あたりの所要を記録する。
- [ ] T002 [Plan] 残る 73 行の進め方を決めて記録する。
- [ ] T003 [Type] `optional` と `excluded` の観測の型を、それぞれ 1 件目で決める。
- [ ] T004 [Ledger] 節ごとに消化し、解決した id を台帳から外す。
- [ ] T005 [Defect] 宣言した採用を満たしていない行が見つかったら、欠陥の work item を切り出す。
- [ ] T006 [Verify] `mise run verify`。

## Verification

- 本項目が持つ 80 件が `tools/check/standards-coverage-debt.json` から 1 件残らず消えている。
- 注記を足した各テストについて、対応する production の判断を崩すとそのテストが落ちる。
- `mise run check-spec`
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** 名指しの文字列があれば検査は通るので、読まずに id を貼れば件数は減る。80 行という数がこの誘惑を最も強くする。注記に「何を固定しているか」を書かせ、崩して落ちることを確かめる。
- **80 行が長期化し、台帳が空にならない。** T002 で分割を判断するまで着手を広げない。分割した場合、本項目は最初の節と分割の記録だけを持つ。
- **台帳の同時編集。** 台帳は id 順に 1 エントリー 1 id なので、文書ごとの work item が並行しても衝突はエントリー単位に収まる。各項目は自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。
