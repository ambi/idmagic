---
depends_on: []
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-06
priority: p2
change_kind: tooling
spec_impact: { kind: none, reason: "宣言済みの標準行に、その id を名指しするテストを対応付ける作業である。standards.md の行そのものも製品の振る舞いも変えない。テストが書けない行が見つかった場合、それは製品が宣言した採用を満たしていないということなので、欠陥として個別の work item に切り出す。" }
---

# 標準の被覆負債 134 件を消化し、`standards-coverage-debt.json` を空にする

## Motivation

`docs/standards.md` は自らの表について「各行は、規範 ID を名指しするテストを製品のテストの中に持つ。まだテストの無い行は `tools/check/standards-coverage-debt.json` に理由付きで残っており、この一覧は縮むだけである」と宣言している。

[[wi-418-normative-coverage-gates]] がこの検査を入れたとき、154 行のうち 134 行が名指しを持っていなかったので、134 件がそのまま台帳へ入った。行は 155 行へ増えたが、台帳は 2026-09-06 時点でも 134 件のままである。`git log -- tools/check/standards-coverage-debt.json` が返すコミットは、台帳を作った 1 件だけである。

**この台帳を縮めることを Scope に書いた work item は、リポジトリのどこにも無い。** 134 件すべての理由が `present when the check was introduced` で揃っており、投入以降に 1 件も判断されていないことが理由の欄からも読める。

負債の性質は `Adoption` 列によって違う。全 155 行に対する内訳は次のとおり（2026-09-06 実測）。

| Adoption | 負債 | 全体 |
|---|---:|---:|
| required | 89 | 96 |
| optional | 25 | 27 |
| excluded | 15 | 19 |
| partial | 5 | 13 |

所有する文書ごとの分布は次のとおりで、`oauth2` の 80 件が全体の 6 割を占める。`OIDC-CORE-CODE-FLOW` だけが 2 つの文書から宣言されているため、行数の合計は 155 ではなく 156 になる。

| 文書 | 負債 / 行 |
|---|---:|
| `docs/contexts/oauth2/standards.md` | 80 / 83 |
| `docs/contexts/api-tokens/standards.md` | 11 / 12 |
| `docs/contexts/authentication/standards.md` | 9 / 10 |
| `docs/standards.md` | 7 / 8 |
| `docs/contexts/saml/standards.md` | 6 / 7 |
| `docs/contexts/ws-federation/standards.md` | 6 / 6 |
| `docs/contexts/authorization/standards.md` | 5 / 5 |
| `docs/contexts/sourcing/standards.md` | 5 / 8 |
| `docs/contexts/sharedsignals/standards.md` | 4 / 4 |
| `docs/contexts/provisioning/standards.md` | 1 / 13 |

`provisioning` だけが 13 行中 12 行の名指しを持っている。SCIM の適合作業（[[wi-238-scim-inbound-list-query-conformance]]）が id を名指しするテストを書いたからであり、消化が可能であることの実例である。裏を返せば、他の文書は適合作業を経ていないというだけで負債になっている。

**もう一つ、台帳の宣言と検査が食い違っている。** `tools/check/src/check-specifications.ts:162` は `checkNormativeCoverage` に `debtBaseline` を渡すが、渡しているのは具体例の台帳だけである。標準の台帳には baseline が無いので、新しく足した `standards.md` の行を、テストを書かずに台帳へ追記しても検査は通る。「この一覧は縮むだけである」は文書の宣言であって、検査された性質ではない。消化を始める前にここを塞がないと、消化と追記が競争になる。

## Scope

- 標準の台帳にも受入集合の固定を入れ、新規の追記を拒否する。具体例側の `tools/check/example-coverage-debt-baseline.json` と同じ形にする。
- 134 件を 1 件ずつ確認し、次のいずれかに解決して台帳から外す。
  - 当の行を検証しているテストが実在する → そのテストに `// <ID>: <この行の何を固定しているか>` の注記を足す。
  - 当の行を検証しているテストが無い → 書く。観測は `Adoption` 列に応じた形（Design を参照）にする。
  - 行が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- 名指しの対象が `docs/standards.md`（横断）の 7 件について、どのパッケージのテストが所有するかを決める。
- 実装が宣言した採用を満たしていないことが分かった場合は、**本 work item では直さず欠陥として切り出す**。テストの追加と実装の修正を同じ変更に混ぜると、どちらが何を意味するのか後から読めない。
- 134 件が 0 になった時点で `standards-coverage-debt.json` と受入集合のファイルを落とし、`checkNormativeCoverage` へ標準側から `debt` を渡すのをやめる。例外を持たない検査にする。

## Out of Scope

- `tools/check/example-coverage-debt.json` の 614 件。[[wi-496-burn-down-the-example-coverage-debt]] が持つ。
- 既存の拒否テストが防いだ効果を観測していない件。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。
- `standards.md` の行の追加、削除、`Adoption` および `Strength` の変更。行の内容が現状と食い違うことが判明した場合は規範の変更なので、別の work item が扱う。
- 見つかった実装の欠陥の修正。切り出した先で扱う。
- 行カバレッジ率の目標または閾値。
- 「id を名指しする文字列がある」だけを根拠にした台帳からの削除。

## Design

台帳を縮める根拠は、id が書かれていることではなく、そのテストの入力と観測が当の行の `Statement` を区別できることである。`checkNormativeCoverage` は文字列の一致しか見ないので、名指しさえすれば検査は通る。注記に「何を固定しているか」を書かせることが、名指しが実在の検証に付いていることの唯一の担保になる。id だけの注記は禁止する。

`Adoption` 列は、テストが何を観測すべきかを決める。同じ「テストが名指しする」でも、次の 4 通りは中身が違う。

| Adoption | テストが観測するもの |
|---|---|
| `required` | 宣言した振る舞いが、製品の正式な入口から到達できること。 |
| `partial` | 採用した範囲の振る舞いと、採用していない範囲がどう扱われるか（拒否するのか、単に提供しないのか）。 |
| `optional` | 提供している場合はその振る舞い。提供していないなら、行の `Adoption` が誤っているので規範の変更として切り出す。 |
| `excluded` | 提供していないこと。要求が届いたときに製品が拒否するなら、その拒否と、拒否が防いだ効果。 |

`excluded` の 15 件は「Implicit Grant を提供する」のように、行の `Statement` が製品の振る舞いではなく標準側の機能を書いている。これらのテストは `Statement` を満たすことではなく、満たさないことを観測する。書き方が `required` と逆になるため、最初の 1 件で型を決めてから残りへ広げる。

受入集合の固定は消化より先に入れる。順序を逆にすると、消化している間に新しい行が台帳へ流れ込み、件数が減らない理由が消化の遅さなのか流入なのか区別できなくなる。`checkNormativeCoverage` は `debtBaseline` を任意の引数として既に受け取るので、必要なのは `standards-coverage-debt-baseline.json` を現在の 134 件で作り、`check-specifications.ts` から渡すことだけである。新しい検査の型を作らない。

進める単位は所有文書とする。分類（named / nearby）順に進める案は却下した。`docs/standards.md` の行は分類の材料になる `report-coverage-debt` の対象外であり（同ツールは `example-coverage-debt.json` しか読まない）、標準側には機械的な分類がそもそも存在しない。文書単位なら、標準そのものを 1 度読む文脈で連続した行を判断できる。

`oauth2` の 80 件を 1 つの work item で扱うかは未決である。**最初の 1 文書（`ws-federation` の 6 件、または `sharedsignals` の 4 件）を通しで消化して 1 件あたりの所要を測り、そこで決める。** 測る前に分割の粒度を決めない。

## Plan

1. `standards-coverage-debt-baseline.json` を現在の 134 件で作り、`check-specifications.ts` から `debtBaseline` として渡す。受入集合に無い id を足した fixture が、規則を入れる前は通り、入れた後に落ちることを観測する。
2. `sharedsignals` の 4 件を通しで消化し、注記の型と、1 件あたりの所要を記録する。
3. 記録をもとに、残る文書を本 work item で続けるか子 work item へ割るかを決め、本節へ書く。
4. 文書ごとに消化する。`excluded` の行に当たったら、最初の 1 件で観測の型を決める。
5. 134 件が 0 になったら、台帳と受入集合のファイル、および標準側の `debt` 引数を落とす。

## Tasks

- [ ] T001 [Tooling] 標準の台帳に受入集合を導入し、新規の追記を拒否する。
- [ ] T002 [Baseline] `sharedsignals` の 4 件を消化し、注記の型と所要を本 work item へ記録する。
- [ ] T003 [Plan] 残る 130 件の進め方（本 work item で続けるか分割するか）を決めて記録する。
- [ ] T004 [Ledger] 文書ごとに消化し、解決した id を台帳から外す。
- [ ] T005 [Defect] 宣言した採用を満たしていない行が見つかったら、欠陥の work item を切り出す。
- [ ] T006 [Tooling] 台帳が空になったら、台帳、受入集合、標準側の `debt` 引数を落とす。
- [ ] T007 [Verify] `mise run verify`。

## Verification

- `mise run check-spec` が標準の被覆について例外を持たずに通る。
- 受入集合に無い id を台帳へ足すと `mise run check-spec` が落ちる。
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** 名指しの文字列があれば検査は通るので、読まずに id を貼れば件数は減る。減った件数は何も意味しない。注記に「何を固定しているか」を書かせること、および `Adoption` ごとの観測の型を先に決めることで、貼るだけの作業と区別する。
- **`excluded` の行に書けるテストが無い。** 製品がその機能をそもそも実装していないなら、観測できるのは「入口が存在しない」ことだけになりうる。この場合に何を観測とするかは T002 の前に決める。決められない行は、台帳へ残す理由を `present when the check was introduced` から具体的な理由へ書き換えたうえで残す。理由が更新されていれば、判断済みであることが後から読める。
- **`oauth2` の 80 件が長期化する。** T003 で分割を判断するまで着手を広げない。分割した場合、親である本 work item は受入集合の導入と最後の台帳削除だけを持つ。
