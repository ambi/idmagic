---
depends_on:
  - wi-538-back-oauth2-examples-with-tests
  - wi-539-back-authentication-examples-with-tests
  - wi-540-back-identity-management-examples-with-tests
  - wi-541-back-system-examples-with-tests
  - wi-542-back-tenancy-examples-with-tests
  - wi-543-back-application-examples-with-tests
  - wi-544-back-authorization-examples-with-tests
  - wi-545-back-provisioning-examples-with-tests
  - wi-546-back-sourcing-examples-with-tests
  - wi-547-back-jobs-examples-with-tests
  - wi-548-back-identity-governance-examples-with-tests
  - wi-549-back-sharedsignals-examples-with-tests
  - wi-550-back-saml-examples-with-tests
  - wi-551-back-api-tokens-examples-with-tests
  - wi-552-back-signing-keys-examples-with-tests
  - wi-553-back-audit-examples-with-tests
  - wi-554-back-ws-federation-examples-with-tests
  - wi-555-back-data-keys-examples-with-tests
  - wi-556-back-seeding-examples-with-tests
  - wi-557-back-cross-context-examples-with-tests
status: in_progress
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-06
priority: p2
change_kind: tooling
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 宣言済みの具体例にテストを対応付ける作業であり、製品の振る舞いも公開契約も変わらないので、リリースの読み手に見えるものが無い。
  references: []
spec_impact: { kind: none, reason: "宣言済みの具体例に、その id を名指しするテストを対応付ける作業である。シナリオも製品の振る舞いも変えない。テストが書けない具体例が見つかった場合、それは実装が具体例のとおりに振る舞っていないということなので、欠陥として個別の work item に切り出す。" }
initial_context:
  specification:
    - docs/contexts/claim-mapping/scenarios.feature.md
    - docs/contexts/workloadidentity/scenarios.feature.md
  typespec: []
  source:
    - tools/check/src/normative-coverage.ts
    - tools/check/example-coverage-debt.json
    - backend/claimmapping/usecases/floor.go
    - backend/claimmapping/usecases/projection.go
  tests:
    - backend/claimmapping/usecases/floor_test.go
    - backend/workloadidentity/usecases/verify_workload_attestation_test.go
    - backend/workloadidentity/usecases/admin_trust_bundles_test.go
    - backend/workloadidentity/usecases/admin_bindings_test.go
    - backend/workloadidentity/handlers_http/routes_test.go
    - backend/oauth2/handlers_http/token_exchange_handler_test.go
  stop_before_reading:
    - docs/contexts/oauth2/scenarios.feature.md
    - docs/contexts/authentication/scenarios.feature.md
    - frontend
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
- 614 件が 0 になった時点で `example-coverage-debt.json` を落とし、`checkNormativeCoverage` へ具体例側から `debt` を渡すのをやめる。例外を持たない検査にする。

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

### 測定の結果（T001 と T002、16 件）

| 具体例 | 既存テストの状態 | やったこと |
|---|---|---|
| `EX-CLAIMMAPPING-001-01` | テストが無い | 規則の無い属性が発行されないことを見るテストを新設 |
| `EX-CLAIMMAPPING-002-01` | 2 件あるが、発行物を観測していない | 注記。クレーム集合と NameID が空であること、および対照を追加 |
| `EX-CLAIMMAPPING-003-01` | テストが無い | 部分的な発行が起きないことを見るテストを新設 |
| `EX-WORKLOADIDENTITY-001-01` | 2 件あるが、有効期間を観測していない | 注記。`expires_in` の観測を追加 |
| `EX-WORKLOADIDENTITY-002-01` から `-007-01` | 6 件とも理由は観測していたが、返り値を観測していない | 注記。資格情報が返らないことの観測を共通ヘルパーへ追加 |
| `EX-WORKLOADIDENTITY-008-01` | あるが、イベントを観測していない | 注記。3 つの遷移のイベント列を追加 |
| `EX-WORKLOADIDENTITY-008-02`、`-008-03` | 具体例を区別できていた | 注記のみ |
| `EX-WORKLOADIDENTITY-009-01` | あるが、関連付けが増えていないことを観測していない | 注記。読み直しを追加 |
| `EX-WORKLOADIDENTITY-010-01` | 具体例を区別できていた | 注記のみ |
| `EX-WORKLOADIDENTITY-010-02` | テストが無い | 登録以外の 9 操作を列挙する拒否テストを新設 |

**16 件のうち、注記だけで済んだのは 4 件（25 %）である。** 3 件はテストが 1 つも無く、9 件は既存テストへ新しい観測を足す必要があった。台帳の件数は作業量の目安にならない。

**`report-coverage-debt` の分類は、テストの有無を予測しない。** 2 Context とも `named` は 0 件で、16 件すべてが `nearby` だった。T002 が測ると決めた `named` の的中率は、この 2 Context では測れない。一方で `nearby` の当たり方は Context で割れた。`workloadidentity` は 13 件とも当の振る舞いへ届くテストがあり、`claim-mapping` は 3 件中 2 件が無かった。差は分類ではなく、その Context のテストがシナリオから書かれたか（`workloadidentity`）、関数から書かれたか（`claim-mapping`）に由来する。**分類ではなく、その Context のテストの出自を先に見るほうが当たる。**

`EX-WORKLOADIDENTITY-002-01` から `-007-01` が名指す `WorkloadAttestationRejectedError` は、Go に対応する型を持たない。TypeSpec 側では本体を持たない領域条件として宣言されており（`spec/contexts/workloadidentity/models.tsp`）、具体例が言う `reason` は `WorkloadAttestationRejected` イベントに載る。**欠陥ではないと判断した。** 仕様が本体を持たないと宣言しているものに、実装が型を持たないことは食い違いではない。

固定費は Context ごとに立った。その Context の `scenarios.feature.md`、domain、use case、既存テストを 1 度読む費用が支配的で、1 件あたりの手数はそれに比べて小さい。この固定費は 1 Context の中では償却されるが、Context をまたぐと償却されない。

### T003 の判断: Context ごとに子 work item へ割る

固定費が Context ごとに立つという測定結果から、残る 598 件は **Context ごとに 1 件の子 work item へ割る**。20 件を起票した。

| 子 | 宣言する文書 | 件数 |
|---|---|---:|
| [[wi-538-back-oauth2-examples-with-tests]] | `docs/contexts/oauth2/scenarios.feature.md` | 105 |
| [[wi-539-back-authentication-examples-with-tests]] | `docs/contexts/authentication/scenarios.feature.md` | 71 |
| [[wi-540-back-identity-management-examples-with-tests]] | `docs/contexts/identity-management/scenarios.feature.md` | 66 |
| [[wi-541-back-system-examples-with-tests]] | `docs/contexts/system/scenarios.feature.md` | 45 |
| [[wi-542-back-tenancy-examples-with-tests]] | `docs/contexts/tenancy/scenarios.feature.md` | 39 |
| [[wi-543-back-application-examples-with-tests]] | `docs/contexts/application/scenarios.feature.md` | 32 |
| [[wi-544-back-authorization-examples-with-tests]] | `docs/contexts/authorization/scenarios.feature.md` | 31 |
| [[wi-545-back-provisioning-examples-with-tests]] | `docs/contexts/provisioning/scenarios.feature.md` | 27 |
| [[wi-546-back-sourcing-examples-with-tests]] | `docs/contexts/sourcing/scenarios.feature.md` | 25 |
| [[wi-547-back-jobs-examples-with-tests]] | `docs/contexts/jobs/scenarios.feature.md` | 24 |
| [[wi-548-back-identity-governance-examples-with-tests]] | `docs/contexts/identity-governance/scenarios.feature.md` | 21 |
| [[wi-549-back-sharedsignals-examples-with-tests]] | `docs/contexts/sharedsignals/scenarios.feature.md` | 20 |
| [[wi-550-back-saml-examples-with-tests]] | `docs/contexts/saml/scenarios.feature.md` | 18 |
| [[wi-551-back-api-tokens-examples-with-tests]] | `docs/contexts/api-tokens/scenarios.feature.md` | 16 |
| [[wi-552-back-signing-keys-examples-with-tests]] | `docs/contexts/signing-keys/scenarios.feature.md` | 14 |
| [[wi-553-back-audit-examples-with-tests]] | `docs/contexts/audit/scenarios.feature.md` | 14 |
| [[wi-554-back-ws-federation-examples-with-tests]] | `docs/contexts/ws-federation/scenarios.feature.md` | 11 |
| [[wi-555-back-data-keys-examples-with-tests]] | `docs/contexts/data-keys/scenarios.feature.md` | 10 |
| [[wi-556-back-seeding-examples-with-tests]] | `docs/contexts/seeding/scenarios.feature.md` | 5 |
| [[wi-557-back-cross-context-examples-with-tests]] | `docs/scenarios.feature.md` | 4 |

`oauth2` の 105 件をここでさらに割らないのは、割る根拠がまだ無いからである。**分割の判断そのものを [[wi-538-back-oauth2-examples-with-tests]] へ渡す。** 同項目は最も件数の多い規則群を通しで消化してから決める。本項目が `claim-mapping` と `workloadidentity` で採ったのと同じ順序であり、[[wi-495-burn-down-the-standards-coverage-debt]] が `oauth2` について採ったのと同じ形である。

`system` の 45 件は [[wi-541-back-system-examples-with-tests]] が持ち、所有パッケージを決めることから始める。横断シナリオの 4 件は、どの Context の子にも属さないため [[wi-557-back-cross-context-examples-with-tests]] が持つ。

本項目はこれ以降、報告ツールの維持と、20 件がすべて完了した後の台帳削除だけを持つ。`depends_on` がその順序を機械で拘束する。

## Plan

1. ~~`claim-mapping` の 3 件を通しで消化し、注記の型と 1 件あたりの所要を記録する。~~ 完了。記録は Design の「測定の結果」節。
2. ~~`workloadidentity` の 13 件を消化し、`named` の見込みがどれだけ当たるかを測る。~~ 完了。2 Context とも `named` は 0 件で的中率は測れず、代わりに分類そのものが予測力を持たないことが分かった。同節に書いた。
3. ~~記録をもとに、残る Context を本 work item で続けるか子 work item へ割るかを決め、本節へ書く。~~ 完了。Context ごとに 20 件へ割った。
4. ~~`report-coverage-debt` に横断シナリオの走査を足し、`(unknown)` を消す。~~ 完了。横断の 4 件は `(cross-context)` として解決する。4 件の消化そのものは [[wi-557-back-cross-context-examples-with-tests]] が持つ。
5. 20 件の子 work item の完了を待つ。`system` の所有パッケージの決定は [[wi-541-back-system-examples-with-tests]] が、`oauth2` の再分割の判断は [[wi-538-back-oauth2-examples-with-tests]] が持つ。
6. 598 件が 0 になったら、台帳と具体例側の `debt` 引数を落とす。

## Tasks

- [x] T001 [Baseline] `claim-mapping` の 3 件を消化し、注記の型と所要を本 work item へ記録する。
  3 件とも消化し、台帳は 614 → 611 件。うち 2 件はテストを新設した。再実行の recipe は
  `mise run test-go-package -- ./backend/claimmapping/usecases`。
- [x] T002 [Baseline] `workloadidentity` の 13 件を消化し、`named` の的中率を記録する。
  13 件とも消化し、台帳は 611 → 598 件。`named` は 0 件だったため的中率は測れず、その事実を
  Design へ記録した。再実行の recipe は `mise run test-go-package -- ./backend/workloadidentity/usecases`、
  `mise run test-go-package -- ./backend/workloadidentity/handlers_http`、
  `mise run test-go-test -- ./backend/oauth2/handlers_http TestTokenExchangeIssuesWorkloadCredential`。
- [x] T003 [Plan] 残る 598 件の進め方（本 work item で続けるか分割するか）を決めて記録する。
  Context ごとに 20 件へ割った。判断と根拠は Design の「T003 の判断」節。
- [x] T004 [Tooling] `report-coverage-debt` が `docs/scenarios.feature.md` も走査するようにし、`(unknown)` を消す。
  `scenarioDocuments()` を足し、横断文書を `(cross-context)` として読むようにした。再実行の recipe は
  `mise run report-coverage-debt` と `mise run typecheck-tools`。
- [ ] T005 [Ledger] 20 件の子 work item の完了を待つ。消化そのものは各子が持つ。
- [ ] T006 [Defect] 具体例のとおりに振る舞っていない実装が見つかったら、欠陥の work item を切り出す。
  T001 と T002 では 1 件も見つからなかった。`WorkloadAttestationRejectedError` に対応する Go の型が
  無い件は、仕様が本体を持たないと宣言しているため欠陥ではないと判断した。
- [ ] T007 [Tooling] 台帳が空になったら、台帳と具体例側の `debt` 引数を落とす。
- [ ] T008 [Verify] `mise run verify`。

## Verification

- `mise run check-spec` が具体例の被覆について例外を持たずに通る。
- `mise run report-coverage-debt` が `(unknown)` の行を出さない。
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** 名指しの文字列があれば検査は通るので、読まずに id を貼れば 614 件は速く減る。減った件数は何も意味しない。注記に「何を固定しているか」を書かせること、および `named` と `nearby` を削除の根拠にしないことを Scope に明記して区別する。
- **件数が大きく、着手が広がったまま止まる。** T003 で分割を判断するまで、T001 と T002 の 2 Context 以外に着手しない。分割した場合、親である本 work item は報告ツールの修正と最後の台帳削除だけを持つ。
- **拒否である具体例のテストが、応答の字面だけを見て書かれる。** 新しく書く拒否テストには wi-392 の規範が効く。本 work item の側では、拒否の具体例に対して「効果の不在を何で観測したか」を注記へ書かせることで、後から区別できるようにする。
- **消化中に具体例が増える。** Git ratchet が基準 revision にない id の追加を拒否する。増えるのは台帳ではなくテストの側なので、消化と流入の競争にはならない。
