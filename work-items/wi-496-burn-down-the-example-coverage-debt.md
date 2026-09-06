---
depends_on: []
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-06
priority: p2
change_kind: tooling
spec_impact: { kind: none, reason: "宣言済みの具体例に、その id を名指しするテストを対応付ける作業である。シナリオも製品の振る舞いも変えない。テストが書けない具体例が見つかった場合、それは実装が具体例のとおりに振る舞っていないということなので、欠陥として個別の work item に切り出す。" }
---

# 具体例の被覆負債 614 件を Context 単位で消化し、`example-coverage-debt.json` を空にする

## Motivation

[[wi-491-adopt-markdown-with-gherkin-scenarios]] は規範シナリオを Markdown with Gherkin へ移し、被覆の管理単位を規則の `REQ-*` から具体例の `EX-*` へ変えた。移行時点でテストの名指しを持たなかった 711 件が `tools/check/example-coverage-debt.json` へ入り、Context ごとの拒否 work item 群（`wi-472` から `wi-488`、いずれも完了）が 711 件から 614 件まで縮めた。

**残る 614 件を Scope に持つ work item は無い。** 完了した 17 件はいずれも「当該 Context の宣言された拒否」だけを対象にしており、[[wi-392-refusal-tests-assert-the-absent-effect]] も既存の拒否テストの改善を持つだけで、台帳の消化は持たない。台帳は「縮むだけ」と自称しているが、縮める力を持つ work item が今は存在しない。

2026-09-06 時点の 614 件の分布は次のとおり。「拒否」は、具体例の結果ステップが `*Error` 型を名指しするものを数えた。

| Context | 負債 | うち拒否 | うち拒否以外 |
|---|---:|---:|---:|
| oauth2 | 105 | 46 | 59 |
| authentication | 71 | 8 | 63 |
| identity-management | 66 | 7 | 59 |
| system | 45 | 0 | 45 |
| tenancy | 39 | 9 | 30 |
| application | 32 | 13 | 19 |
| authorization | 31 | 10 | 21 |
| provisioning | 27 | 8 | 19 |
| sourcing | 25 | 11 | 14 |
| jobs | 24 | 2 | 22 |
| identity-governance | 21 | 6 | 15 |
| sharedsignals | 20 | 10 | 10 |
| saml | 18 | 3 | 15 |
| api-tokens | 16 | 5 | 11 |
| audit | 14 | 3 | 11 |
| signing-keys | 14 | 3 | 11 |
| workloadidentity | 13 | 11 | 2 |
| ws-federation | 11 | 3 | 8 |
| data-keys | 10 | 6 | 4 |
| seeding | 5 | 0 | 5 |
| (横断) | 4 | 0 | 4 |
| claim-mapping | 3 | 0 | 3 |
| **合計** | **614** | **164** | **450** |

`mise run report-coverage-debt` の同日の実測では、614 件のうち 91 件は同じエラーを名指しするテストが同一パッケージにあり（注記だけが欠けている見込み）、474 件は他の拒否テストが近傍にあり、49 件はその Context に拒否を assert するテストが 1 つも届いていない。

この 3 分類は機械判定であり、危険度の順位ではない。分類が言っているのは「id を名指しするテストが無い」だけであって「振る舞いが検証されていない」ではない。逆に、`named` だからといって当の具体例を検証しているとは限らない。**分類は読む順を決める材料であり、台帳から外す根拠にはならない。**

報告そのものにも 2 つの穴がある。`report-coverage-debt` は `docs/contexts/*/scenarios.feature.md` しか走査しないため、横断シナリオ `docs/scenarios.feature.md` の 4 件（`EX-PLATFORM-001-01` など）は Context 不明として `(unknown)` に落ちる。また `system` の 45 件は `backend/system` というディレクトリが無いため候補が 1 件も解決せず、全件が `none` に入る。どちらも「テストが無い」ことの証拠ではなく、報告が見ていないことの証拠である。

## Scope

- 614 件を Context 単位で 1 件ずつ確認し、次のいずれかに解決して台帳から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `// EX-<CONTEXT>-NNN-MM: <この具体例の何を固定しているか>` の注記を足す。
  - 当の具体例を検証しているテストが無い → 書く。具体例が拒否なら、[[wi-392-refusal-tests-assert-the-absent-effect]] が定める形（拒否応答と、拒否が防いだ効果の双方を観測する）で書く。
  - 具体例が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- `system` の 45 件について、どの Go パッケージのテストが所有するかを決める。
- 横断シナリオ `docs/scenarios.feature.md` の 4 件について、所有するテストを決める。あわせて `report-coverage-debt` が同文書も走査するようにし、`(unknown)` を消す。
- 実装が具体例のとおりに振る舞っていないことが分かった場合は、**本 work item では直さず欠陥として切り出す**。テストの追加と実装の修正を同じ変更に混ぜると、どちらが何を意味するのか後から読めない。
- 614 件が 0 になった時点で `example-coverage-debt.json` と `example-coverage-debt-baseline.json` を落とし、`checkNormativeCoverage` へ具体例側から `debt` を渡すのをやめる。例外を持たない検査にする。

## Out of Scope

- `tools/check/standards-coverage-debt.json` の 134 件。[[wi-495-burn-down-the-standards-coverage-debt]] が持つ。
- 既に台帳に載っていない拒否テストが、防いだ効果を観測していない件（2026-09-06 実測で 159 件中 112 件）。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。本 work item が新しく書く拒否テストは、その規範を満たす形で書く。既存テストに注記を足すだけの件で、そのテストが効果の不在を見ていない場合は、注記を足したうえで wi-392 の対象として残す。
- シナリオと具体例の追加、削除、書き換え。具体例の記述が実装と食い違うことが判明した場合は規範の変更なので、別の work item が扱う。
- 見つかった実装の欠陥の修正。切り出した先で扱う。
- 行カバレッジ率の目標または閾値。
- `named` または `nearby` に分類されたことを根拠にした台帳からの削除。

## Design

台帳を縮める根拠は、id が書かれていることではなく、そのテストの入力と観測が当の具体例の Given、When、Then を区別できることである。`checkNormativeCoverage` は文字列の一致しか見ないので、名指しさえすれば検査は通る。注記に「何を固定しているか」を書かせることが、名指しが実在の検証に付いていることの唯一の担保になる。id だけの注記は禁止する。

進める単位は Context とする。分類（`named` → `nearby` → `none`）順に進める案は却下した。`named` の 91 件を先に片付ければ台帳は速く縮むが、`named` が言っているのは「同じエラー型を名指しするテストが同じパッケージにある」以上のことではなく、読んだら別の具体例のテストだった、という取りこぼしが起きる。Context 順なら、`scenarios.feature.md` を 1 度読む文脈で連続した具体例を判断でき、`workloadidentity` のように件数の少ない Context を早く 0 にできる。

**拒否と拒否以外で work item を分けない。** 分けるには「この具体例は拒否か」を機械的に決める必要があるが、その判定は安定しない。Motivation の表は結果ステップが `*Error` 型を名指しするかどうかで数えており、これは `report-coverage-debt` が重みを導くのと同じ材料である。しかし同じ日の実測では、型を名指しせずに拒否の語で結果を書いている具体例が 133 件あり、そこには `ProvisioningDelivery` のロールバックのように拒否ではないものも混ざる。境界がぶれる基準で所有を分ければ、どちらの work item にも入らない具体例が生まれる。それは本 work item が解消しようとしている状態そのものである。台帳の所有者は 1 つにし、拒否である具体例には wi-392 の書き方を適用する、という形で規範だけを共有する。

`system` の 45 件は、`system` という Context に対応する Go パッケージが無いために報告が候補を解決できていない。本当にテストが無いのか、別のパッケージにあるのかはまだ誰も見ていない。件数が 45 と大きく、他の Context と性質が違うため、最初にではなく、注記の型と所要が固まってから着手する。

分割の粒度は未決である。**最初の 1 Context（`claim-mapping` の 3 件、続けて `workloadidentity` の 13 件）を通しで消化して 1 件あたりの所要を測り、そこで決める。** 上位 4 つ（`oauth2` 105、`authentication` 71、`identity-management` 66、`system` 45）で 287 件、全体の 47 % を占めるので、少なくともこの 4 つは子 work item へ割る見込みが高い。測る前に粒度を決めない。

## Plan

1. `claim-mapping` の 3 件を通しで消化し、注記の型と 1 件あたりの所要を記録する。
2. `workloadidentity` の 13 件を消化し、`named` の見込みがどれだけ当たるかを測る。分類を読む順の材料として信用してよいかは、ここで決まる。
3. 記録をもとに、残る Context を本 work item で続けるか子 work item へ割るかを決め、本節へ書く。
4. `report-coverage-debt` に横断シナリオの走査を足し、`(unknown)` の 4 件を解決する。
5. Context ごとに消化する。`system` の 45 件は所有パッケージを決めてから着手する。
6. 614 件が 0 になったら、台帳、受入集合、具体例側の `debt` 引数を落とす。

## Tasks

- [ ] T001 [Baseline] `claim-mapping` の 3 件を消化し、注記の型と所要を本 work item へ記録する。
- [ ] T002 [Baseline] `workloadidentity` の 13 件を消化し、`named` の的中率を記録する。
- [ ] T003 [Plan] 残る 598 件の進め方（本 work item で続けるか分割するか）を決めて記録する。
- [ ] T004 [Tooling] `report-coverage-debt` が `docs/scenarios.feature.md` も走査するようにし、横断の 4 件を解決する。
- [ ] T005 [Ledger] Context ごとに消化し、解決した id を台帳から外す。
- [ ] T006 [Ledger] `system` の 45 件の所有パッケージを決めて消化する。
- [ ] T007 [Defect] 具体例のとおりに振る舞っていない実装が見つかったら、欠陥の work item を切り出す。
- [ ] T008 [Tooling] 台帳が空になったら、台帳、受入集合、具体例側の `debt` 引数を落とす。
- [ ] T009 [Verify] `mise run verify`。

## Verification

- `mise run check-spec` が具体例の被覆について例外を持たずに通る。
- `mise run report-coverage-debt` が `(unknown)` の行を出さない。
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** 名指しの文字列があれば検査は通るので、読まずに id を貼れば 614 件は速く減る。減った件数は何も意味しない。注記に「何を固定しているか」を書かせること、および `named` と `nearby` を削除の根拠にしないことを Scope に明記して区別する。
- **件数が大きく、着手が広がったまま止まる。** T003 で分割を判断するまで、T001 と T002 の 2 Context 以外に着手しない。分割した場合、親である本 work item は報告ツールの修正と最後の台帳削除だけを持つ。
- **拒否である具体例のテストが、応答の字面だけを見て書かれる。** 新しく書く拒否テストには wi-392 の規範が効く。本 work item の側では、拒否の具体例に対して「効果の不在を何で観測したか」を注記へ書かせることで、後から区別できるようにする。
- **消化中に具体例が増える。** 受入集合 `example-coverage-debt-baseline.json` が既に効いており、新しい具体例を台帳へ足すことは検査が拒否する。増えるのは台帳ではなくテストの側なので、消化と流入の競争にはならない。
