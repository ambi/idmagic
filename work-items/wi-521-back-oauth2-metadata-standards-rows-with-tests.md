---
depends_on: []
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-09
priority: p2
change_kind: maintenance
spec_impact: { kind: none, reason: "宣言済みの標準行に、その id を名指しするテストを対応付ける作業である。standards.md の行そのものも製品の振る舞いも変えない。テストが書けない行が見つかった場合、それは製品が宣言した採用を満たしていないということなので、欠陥として個別の work item に切り出す。" }
---

# メタデータ配信が宣言する標準 7 行にテストを対応付け、標準の被覆台帳から外す

## Motivation

[[wi-499-back-oauth2-standards-rows-with-tests]] は `docs/contexts/oauth2/standards.md` の 80 行を引き取り、最初の節（`OAuth Client ID Metadata Document` の 7 行）を消化したうえで、残る 73 行を**行が共有する製品の入口**を単位に 7 件へ割った。本項目はそのうち 7 行を持つ。

7 行は、クライアントとリソースサーバーが製品の設定を読む唯一の経路を定める。7 行とも `required` である。ここに書かれた内容が実際の振る舞いとずれると、適合クライアントは製品が受け付けない要求を組み立てる。

## Scope

- 次の 7 行を消化する。

| ID | Adoption | 節 |
|---|---|---|
| `RFC7517-JWKS` | required | JSON Web Key |
| `RFC8414-METADATA` | required | OAuth 2.0 Authorization Server Metadata |
| `RFC9728-METADATA` | required | OAuth 2.0 Protected Resource Metadata |
| `RFC9728-WELL-KNOWN` | required | OAuth 2.0 Protected Resource Metadata |
| `RFC9728-IDMAGIC-API` | required | OAuth 2.0 Protected Resource Metadata |
| `RFC9728-CHALLENGE` | required | OAuth 2.0 Protected Resource Metadata |
| `OIDC-DISCOVERY-CONFIGURATION` | required | OpenID Connect Discovery 1.0 incorporating errata set 1 |

- 1 行ごとに、その行の `Statement` を区別できる入力と観測を持つテストを対応付け、`// <ID>: <この行の何を固定しているか>` の注記を足す。
- 観測の形は行の `Adoption` に従う。型は [[wi-495-burn-down-the-standards-coverage-debt]] の Design が定める。
- 消化した id を `tools/check/standards-coverage-debt.json` から外す。

## Out of Scope

- 他の work item が持つ標準行。[[wi-499-back-oauth2-standards-rows-with-tests]] の Design にある分割表が所属を定める。
- `standards.md` の行の追加、削除、`Adoption` および `Strength` の変更。行の内容が現状と食い違うと判明した場合は規範の変更であり、別の work item が扱う。
- 見つかった実装の欠陥の修正。欠陥として切り出した先で扱う。
- 受入集合の導入と、台帳ファイルそのものの削除。[[wi-495-burn-down-the-standards-coverage-debt]] が持つ。
- 具体例の被覆負債。[[wi-496-burn-down-the-example-coverage-debt]] が持つ。
- id を名指しする文字列があるだけを根拠にした台帳からの削除。

## Design

**観測の入口は `/.well-known` の各文書と JWKS の配信経路 である。** [[wi-499-back-oauth2-standards-rows-with-tests]] が 7 行を測って出した結論は、消化の費用を支配するのが行数ではなく「行が共有する入口にハーネスを 1 つ組むこと」だという点にある。本項目の 7 行はこの入口を共有するので、ハーネスは 1 度組めば足りる。

この入口の観測は、文書を取得して内容を読むだけでは足りない。文書が「対応している」と宣言した機能が実際に動くことを対で読まなければ、宣言と実装のずれをこのテストは見逃す。少なくとも、宣言したエンドポイントが実在すること、宣言した鍵で発行済みトークンの署名が検証できることは併せて読む。

`RFC9728-CHALLENGE` は、文書そのものではなく保護リソースの `401` と `403` に `resource_metadata` が載ることを言っているので、観測の入口が他の 6 行と違う。

行ごとの観測の形は `Adoption` が決める。`required` は宣言した振る舞いが正式な入口から到達できること、`optional` は提供しているならその振る舞い、`excluded` は提供していないことと拒否が防いだ効果、`partial` は採った範囲と採らなかった範囲の扱いを、それぞれ観測する。`excluded` の観測は 1 つの型に収まらない。行の `Statement` が製品の制約を書いているのか標準側の機能を書いているのかで観測が裏返るので、本項目の `excluded` の 1 件目でどちらかを決めてから残りへ広げる。

## Plan

1. 7 行が共有する入口にハーネスを組み、`required` の 1 件目を通しで消化して型を決める。
2. `optional` があれば、その 1 件目で「提供している」と言える根拠の形を決める。提供していなければ規範の変更として切り出す。
3. `excluded` があれば、その 1 件目で観測の型を決める。
4. 残りを消化し、解決した id を台帳から外す。
5. 宣言した採用を満たしていない行が見つかったら、欠陥の work item を切り出す。

## Tasks

- [ ] T001 [Acceptance] 消化する id を台帳から先に外し、`mise run check-spec` が当該 id ごとに `is declared, but no test names it` を報告することを観測する。
- [ ] T002 [Harness] `/.well-known` の各文書と JWKS の配信経路 にハーネスを組み、`required` の 1 件目で型を決める。
- [ ] T003 [Type] `optional` と `excluded` の観測の型を、それぞれ 1 件目で決める。
- [ ] T004 [Ledger] 残りを消化し、解決した id を `tools/check/standards-coverage-debt.json` から外す。
- [ ] T005 [Resistance] 行が言っている判断を production 側で崩し、対応するテストが落ちることを行ごとに観測する。
- [ ] T006 [Defect] 宣言した採用を満たしていない行が見つかったら、欠陥の work item を切り出す。
- [ ] T007 [Verify] `mise run verify`。

## Verification

- 本項目が持つ 7 件が `tools/check/standards-coverage-debt.json` から 1 件残らず消えている。
- 注記を足した各テストについて、対応する production の判断を崩すとそのテストが落ちる。
- `mise run check-spec`
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** 名指しの文字列があれば検査は通るので、読まずに id を貼れば件数は減る。注記に「何を固定しているか」を書かせ、崩して落ちることを T005 で確かめる。
- **ハーネスが本番とずれる。** 入口を通すという方針は、その入口が製品と同じ部品でできているときだけ意味を持つ。[[wi-500-back-api-tokens-standards-rows-with-tests]] はここを一度間違え、製品の欠陥でないものを欠陥として起票した。組み立ては `cmd/internal/bootstrap` が作る形に合わせる。
- **1 行が 2 つのことを言っている。** `Statement` に動詞が 2 つあれば観測も 2 つ要る。
- **宣言を読むだけで、宣言どおりに動くことを読まない。** メタデータは宣言と実装がずれても取得できてしまう。宣言した機能が動くことを対で読む。
- **台帳の同時編集。** 台帳は id 順に 1 エントリー 1 id なので、並行しても衝突はエントリー単位に収まる。自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。
