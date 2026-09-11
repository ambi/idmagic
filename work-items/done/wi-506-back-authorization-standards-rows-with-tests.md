---
depends_on: []
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-06
priority: p2
change_kind: maintenance
spec_impact: { kind: none, reason: "宣言済みの標準行に、その id を名指しするテストを対応付ける作業である。standards.md の行そのものも製品の振る舞いも変えない。テストが書けない行が見つかった場合、それは製品が宣言した採用を満たしていないということなので、欠陥として個別の work item に切り出す。" }
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 標準の行にも製品の振る舞いにも変更が無く、増えたのはテストと注記だけなので、リリースの読み手に見えるものが無い。
  references: []
initial_context:
  specification:
    - docs/contexts/authorization/standards.md
  typespec: []
  source:
    - backend/authorization/usecases/check_access.go
    - backend/authorization/usecases/deps.go
    - backend/authorization/usecases/errors.go
    - backend/authorization/domain/evaluator.go
    - backend/authorization/ports/repository.go
    - backend/authorization/db_memory/repository.go
    - backend/authorization/handlers_http/routes.go
    - backend/shared/spec/policy.go
    - backend/shared/policy/authorization_http/authorizer.go
    - backend/shared/http/server_http/routes.go
    - tools/check/src/check-documents.ts
    - tools/check/src/normative-coverage.ts
    - tools/check/standards-coverage-debt.json
  tests:
    - backend/authorization/usecases/check_access_test.go
    - backend/authorization/domain/evaluator_test.go
    - backend/authorization/handlers_http/routes_test.go
  stop_before_reading:
    - backend/authorization/db_postgres
    - backend/oauth2
    - docs/contexts/oauth2/standards.md
    - frontend
---

# Authorization が宣言する標準 5 行にテストを対応付け、標準の被覆台帳から外す

## Motivation

[[wi-495-burn-down-the-standards-coverage-debt]] は標準の被覆台帳へ受入集合を入れ、消化の単位を所有文書と決めた。本項目はそのうち `docs/contexts/authorization/standards.md` の 5 行を引き取る。**この文書は 5 行すべてが名指しを持たない。** WsFederation と並んで、台帳の中で全滅している 2 文書のうちの 1 つである。

5 行が扱うのは認可の判定そのものである。`AUTHZEN-FGA-FAIL-CLOSED` は、判定できないときに拒否する側へ倒れることを宣言している。この行が守られていなければ、他の 4 行が正しくても、判定器が答えられない場面で通ってしまう。全滅している 5 行の中で最も先に観測すべき行である。

## Scope

- 次の 5 行を消化する。

| ID | Adoption |
|---|---|
| `AUTHZEN-FGA-EVALUATION` | required |
| `AUTHZEN-FGA-FAIL-CLOSED` | required |
| `AUTHZEN-FGA-ACTOR-CHAIN` | required |
| `RFC8693-FGA-ACTOR-AND` | required |
| `AUTHZEN-FGA-SEARCH` | optional |

- 1 行ごとに、その行の `Statement` を区別できる入力と観測を持つテストを対応付け、`// <ID>: <この行の何を固定しているか>` の注記を足す。
- 観測の形は行の `Adoption` に従う。型は [[wi-495-burn-down-the-standards-coverage-debt]] の Design が定める。
- 消化した id を `tools/check/standards-coverage-debt.json` から外す。

## Out of Scope

- 他の文書が宣言する標準行。文書ごとに別の work item が持つ。OAuth2 が持つ `RFC8693-*` の他の行（委譲、なりすまし、subject token）は [[wi-499-back-oauth2-standards-rows-with-tests]] が持つ。
- `standards.md` の行の追加、削除、`Adoption` および `Strength` の変更。
- 見つかった実装の欠陥の修正。欠陥として切り出した先で扱う。
- 受入集合の導入と、台帳ファイルそのものの削除。親の [[wi-495-burn-down-the-standards-coverage-debt]] が持つ。
- 具体例の被覆負債。[[wi-496-burn-down-the-example-coverage-debt]] が持つ。
- 認可モデルそのものの変更。[[wi-371-authorization-rebac-admin-ui]] が別の側面を持つ。
- id を名指しする文字列があるだけを根拠にした台帳からの削除。

## Design

`AUTHZEN-FGA-FAIL-CLOSED` の観測は、判定器が答えを返せない状況を実際に作ることである。到達できない外部依存、壊れた関係データ、時間切れなど、答えが出ない入力を与えて、結果が「拒否」になることを読む。**答えが出る入力で「拒否が返る」ことを観測しても、この行は区別できない。** それは `AUTHZEN-FGA-EVALUATION` が扱う普通の否認であり、フェイルクローズではない。この 2 行を同じテストで名指すと、片方が失われても気づけない。

`AUTHZEN-FGA-ACTOR-CHAIN` と `RFC8693-FGA-ACTOR-AND` は、どちらも代理の連鎖を扱う。前者は連鎖そのものの解釈、後者は連鎖に含まれる主体の権限が**積で効く**こと、つまり連鎖のどの主体も持っていない権限は結果として持てないことを宣言している。したがって `RFC8693-FGA-ACTOR-AND` の観測は、連鎖の 1 つの主体だけが持つ権限が、連鎖全体としては通らないことである。連鎖が解釈されることだけを観測しても、和で効く実装と区別できない。

`AUTHZEN-FGA-SEARCH` は `optional` である。提供しているならその振る舞いを観測する。提供していないなら行の `Adoption` が誤っているので規範の変更として切り出す。

### T001 の棚卸し

既存のテストを読み、行ごとに「どこまで区別できているか」を書き出した。

| ID | 既存テストが届いている範囲 | 届いていない範囲 |
|---|---|---|
| `AUTHZEN-FGA-EVALUATION` | 無い。`TestCheckAccessFailsClosed` の最後の区画が `spec.Evaluate` を直接呼び、事実の無い `resource:access` が落ちることを観測しているが、これは規則表の試験であって `CheckAccess` が事実を供給することの観測ではない | **全部。** 判定器へ渡る `AuthZRequest` を読むテストが 1 件も無い。関係の成否を自分で判断して `permit` だけを渡す実装も、そもそも評価に載せず結果だけ返す実装も、既存テストを 1 件も落とさずに通る |
| `AUTHZEN-FGA-ACTOR-CHAIN` | `TestCheckAccessRequiresSubjectAndActorChain` が、関係を持たない actor と無効な actor で拒否されることを拒否理由まで含めて観測している | **判定 context の形を観測していない。** チェーンを 1 個の真偽値へ畳んで渡す実装、種別と識別子を連結して 1 本の文字列にする実装、有効性を落とす実装は、どれも既存の拒否理由をそのまま再現できる |
| `AUTHZEN-FGA-FAIL-CLOSED` | 4 つのうち 3 つ。モデル未登録、ストア障害、評価器 `nil` を `TestCheckAccessFailsClosed` が、事実の欠落を規則表側が観測している | **深さ上限が無い。** 上限超過を観測しているのは `domain` の `TestCheckDeniesUnreachableSubjects` だけで、判定の入口から見たものではない。**評価器が error を返す事例も無い。** `nil` の事例は `Authorize` を呼ぶ前に落ちるので、呼んだ先の失敗を許可へ変える実装を区別できない。**製品の正式な入口での観測が 1 件も無い。** error を `permitted: true` に変える handler を既存テストは落とせない |
| `RFC8693-FGA-ACTOR-AND` | 1 段のチェーンについて両方向（actor だけ欠ける／subject だけ欠ける）を観測している | **`act` チェーンが複数段の事例が無い。** 行は「`act` チェーン上のすべての actor」と言っている。先頭の 1 段だけを見る実装は、既存の 1 段の事例をすべて通す |
| `AUTHZEN-FGA-SEARCH` | `TestListAccessibleResourcesIsBoundedAndFiltered` が絞り込みと打ち切りを観測している | **既定の上限を観測していない。** 打ち切りの観測は `MaxEnumeratedResources = 2` を注入した場合だけで、既定を持たない（上限無しで走査する）実装を区別できない。製品の入口では上限は注入されず既定の 500 になる |

5 行のうち、既存の観測が行の `Statement` を区別できている行は 1 つも無かった。

### 判定の入口と `required` の観測の型（T001 が決めた）

5 行はすべて「判定 1 回」を指しており、その製品の正式な入口は
`POST /realms/{realm}/api/admin/v1/authorization/check` と
`POST /realms/{realm}/api/admin/v1/authorization/list-accessible-resources` である。したがって観測は
`backend/authorization/handlers_http/standards_test.go` に集め、要求は HTTP から入れる。

`AUTHZEN-FGA-EVALUATION` と `AUTHZEN-FGA-ACTOR-CHAIN` が言う「判定 context に載せる」は、応答からは
読めない。`Authorizer` は差し替え可能なポートであり、`AUTHZEN=remote` では `AuthZRequest` がそのまま
AuthZEN の PDP へ JSON で渡る。つまり**この 2 行が固定しているのは、判定器へ渡る `AuthZRequest` そのものの
形である**。観測の型は、`Local` を包んで `AuthZRequest` を記録する `Authorizer` を製品の組み立てへ差し、
HTTP から入った要求が作った `AuthZRequest` を読むことにする。応答だけを読む観測では、関係を自分で判断して
結果だけ渡す実装を区別できない。

`AUTHZEN-FGA-FAIL-CLOSED` の 4 つの事例は、同じ差し替えの口で作る。評価器の失敗は失敗する `Authorizer`、
ストア到達不能は失敗する `RelationTupleRepository`、事実の欠落は評価器 `nil`、深さ上限は既定の上限より
深い group の連なりである。深さ上限だけは**同じ形で上限に収まる連なりが許可になること**と対にする。
そうしないと、関係がそもそも無い普通の否認と区別できない。

### 意図した RED 検査

- **Acceptance RED**: `mise run check-spec`。5 件を `tools/check/standards-coverage-debt.json` から先に外し、
  テストを書く前の状態で走らせる。5 行それぞれについて `<ID> is declared, but no test names it.` が出ることを
  観測する。製品の規範要求に対応する受入境界はこの作業に無いので、Requirement は N/A である。
- **Unit RED**: 各行に対応付けたテストへの故障注入。行が言っていることを production 側で崩し、対応する
  テストが落ちることを 1 件ずつ観測する。`checkNormativeCoverage` は文字列の一致しか見ないので、名指しが
  実在の検証に付いていることの担保はこの観測しかない。recipe は
  `mise run test-go-test -- ./backend/authorization/handlers_http <Test>` と
  `mise run test-go-package -- ./backend/authorization/...`。

## Plan

1. `AUTHZEN-FGA-EVALUATION` から着手し、判定の入口と `required` の観測の型を決める。
2. `AUTHZEN-FGA-FAIL-CLOSED` を、判定器が答えを返せない入力で消化する。普通の否認とは別のテストにする。
3. `AUTHZEN-FGA-ACTOR-CHAIN` を消化する。
4. `RFC8693-FGA-ACTOR-AND` を、連鎖の 1 主体だけが持つ権限が通らないことで消化する。
5. `AUTHZEN-FGA-SEARCH` の実装の有無を確かめ、観測するか切り出すかを決める。
6. 解決した id を台帳から外す。

## Tasks

すべての観測の recipe は `mise run test-go-test -- ./backend/authorization/handlers_http <Test>`、
パッケージ単位の確認は `mise run test-go-package -- ./backend/authorization/handlers_http` である。

- [x] T001 [Acceptance] `AUTHZEN-FGA-EVALUATION` を消化し、判定の入口と `required` の型を決める。
  `TestAuthorizationDecisionRidesOnTheSubjectActionResourceContextEvaluation`。入口は
  `POST /api/admin/v1/authorization/check`、観測の型は製品の組み立てへ差した記録用 `Authorizer` が
  受け取った `AuthZRequest` を読むこと。判断と根拠は Design の「判定の入口と `required` の観測の型」節。
  3 区画で、要求の 4 要素と関係の事実、成立しない場合の事実の形、そして答えが評価器の側から来ることを観測する。
- [x] T002 [Acceptance] `AUTHZEN-FGA-FAIL-CLOSED` を、答えを返せない入力で消化する。
  `TestAuthorizationFailsClosedWhenTheDecisionCannotBeAnswered`。行が挙げる 4 つの状況を、失敗する
  `Authorizer`、評価器 `nil`、失敗する `RelationTupleRepository`、未発行のモデル、既定の上限より深い
  group の連なりの 5 区画で作る。深さの区画は**段数だけが違う許可になる入力と対**にしてあり、普通の否認とは
  別物であることが読める。規則表の側の「事実が欠けている」は
  `backend/authorization/usecases/check_access_test.go` の `TestCheckAccessFailsClosed` に注記を足した。
- [x] T003 [Acceptance] `AUTHZEN-FGA-ACTOR-CHAIN` を消化する。
  `TestAuthorizationDecisionCarriesEveryDelegationStageSeparately`。代行者 3 者すべてに関係を持たせた
  うえで、記録した `Context.ActorChain` が `{Type, ID, Active}` の並びとして期待どおりであることを読む。
  有効な代行者、無効化された代行者、登録の無い代行者、そして代行なしの 4 通り。
- [x] T004 [Acceptance] `RFC8693-FGA-ACTOR-AND` を、連鎖の 1 主体だけの権限が通らないことで消化する。
  `TestDelegatedAccessRequiresTheRelationOnEveryActorInTheChain`。`sub` と 2 段の `act` が関係を持つか
  持たないかの全 8 通りを通し、許可が 3 つとも持つ 1 通りだけであることを読む。1 主体だけが持つ入力が
  この表に 3 つ含まれる。
- [x] T005 [Inventory] `AUTHZEN-FGA-SEARCH` の実装の有無を確かめ、観測するか切り出すかを決める。
  **提供している。** `POST /api/admin/v1/authorization/list-accessible-resources` が入口で、
  `usecases.ListAccessibleResources` が上限つきで走査する。`Adoption: optional` は正しいので規範の変更は
  要らない。`TestAccessibleResourceSearchIsSubjectFixedBoundedAndReportsTruncation` で観測した。
  上限は注入せず、製品の既定 (`usecases.DefaultMaxEnumeratedResources` = 500) のまま候補を 1 件多く置く。
- [x] T006 [Ledger] 解決した id を `standards-coverage-debt.json` から外す。
  5 件を削除し、台帳は 6 件から 1 件になった。コメント配列と、Provisioning が持つ
  `RFC7643-OUT-GROUP-RESOURCES` には触れていない。
- [x] T007 [Verify] `mise run verify`。
  詳細は Completion の Verification Results。

## Verification

- 本項目が持つ 5 件が `tools/check/standards-coverage-debt.json` から 1 件残らず消えている。
- 注記を足した各テストについて、対応する production の判断を崩すとそのテストが落ちる。
- `mise run check-spec`
- `mise run verify`

## Risk Notes

- **フェイルクローズを普通の否認で観測する。** 拒否が返ることは両方に共通なので、`AUTHZEN-FGA-FAIL-CLOSED` と `AUTHZEN-FGA-EVALUATION` を同じテストが名指しやすい。前者は判定器が答えを返せない入力でしか区別できない。別のテストにする。
- **積を和で観測する。** `RFC8693-FGA-ACTOR-AND` は連鎖が解釈されることではなく、連鎖のどの主体も持たない権限が通らないことである。連鎖の 1 主体だけが持つ権限を入力にする。
- **台帳の同時編集。** 自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。

## Completion

- **Completed At**: 2026-09-12
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範の変更は無い。
  差分は `docs/contexts/authorization/standards.md` の 5 行に対する被覆の状態である。5 行すべてが
  その行の `Statement` を区別できる入力と観測を持つテストを得て `tools/check/standards-coverage-debt.json`
  から消え、台帳は 6 件から 1 件になった。名指しを持つ id は 283 件から 288 件へ増えた（stash して測った）。
  新設したテストは Go 5 件（部分試験を数えると 21 件）で、**製品コードは 1 行も変わっていない**。
  **既存テストが行を区別できていた行は 1 つも無かった。** 5 行のうち 4 行は既存テストが拒否の効果までは
  観測していたが、行が言っているのは効果ではなく**判定器へ渡す要求の形**（`AUTHZEN-FGA-EVALUATION`、
  `AUTHZEN-FGA-ACTOR-CHAIN`）、**答えが出ない入力での退避先**（`AUTHZEN-FGA-FAIL-CLOSED`）、
  **チェーン全体に対する積**（`RFC8693-FGA-ACTOR-AND`）であり、どれも観測されていなかった。
  **観測の型が他の文書と違った。** Sourcing や WS-Federation の行は応答の形を宣言しているので HTTP の
  応答を読めば足りるが、`AUTHZEN-FGA-EVALUATION` と `AUTHZEN-FGA-ACTOR-CHAIN` が言う「判定 context に
  載せる」は応答にまったく現れない。`Authorizer` は差し替え可能なポートで `AUTHZEN=remote` では
  `AuthZRequest` がそのまま PDP へ渡るので、この 2 行が固定しているのは要求そのものの形である。したがって
  観測は、製品の組み立てへ記録用 `Authorizer` を差し、HTTP から入った要求が作った `AuthZRequest` を読む形に
  なった。応答だけを読む観測では、関係を自分で判断して結果だけ渡す実装を 1 つも落とせない。
  **欠陥は 1 件も見つからなかった。** 5 行はいずれも宣言した採用を満たしている。`AUTHZEN-FGA-SEARCH` は
  `optional` だが実装があるので、`Adoption` の変更も要らない。
  **故障注入で 1 つ、観測の分け方の限界が分かった。** `ListAccessibleResources` では走査を打ち切る判断と
  結果に載せる `Truncated` が同じ変数なので、「上限で止まる」を崩す注入は必ず「打ち切りを示す」側も落とす。
  2 つを分けて観測していることは、`ListedResources` の `Truncated` フィールドだけを `false` にする注入
  （SEARCH/4）でしか確かめられなかった。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（5 件を台帳から外し、テストを書く前の状態で）
  - **Requirement**: N/A: 標準の被覆はテストの有無についての性質であり、製品の規範要求ではない。
  - **Observed Failure**: exit 1。`docs/contexts/authorization/standards.md` の 5 行それぞれについて
    `<ID> is declared, but no test names it. Cite the id from the test that exercises it, or list it in
    tools/check/standards-coverage-debt.json with a reason.`（9 行目 `AUTHZEN-FGA-EVALUATION`、
    10 行目 `AUTHZEN-FGA-ACTOR-CHAIN`、11 行目 `AUTHZEN-FGA-FAIL-CLOSED`、12 行目 `AUTHZEN-FGA-SEARCH`、
    20 行目 `RFC8693-FGA-ACTOR-AND`）。報告はこの 5 行だけだった。
  - **Detection Reason**: 検査は、宣言された id・テストが名指す id・台帳の 3 つを突き合わせる。台帳から
    外した id は、名指すテストが実在しない限り必ず報告される。消化後の同じコマンドは
    `ok normative coverage (157 standard(s), 311 rule(s), 747 example(s), 288 id(s) named by a test)`
    を返す（283 → 288）。
- **Unit RED Evidence**:
  - **Test**: 各行に対応付けたテストへの故障注入（Change-Resistance Results を参照）
  - **Requirement**: N/A: 行ごとの観測は標準の `Statement` に対応し、`REQ` 番号には対応しない。
  - **Observed Failure**: 21 件の故障すべてを、対応するテストが検出した。生存はゼロ。
  - **Detection Reason**: `checkNormativeCoverage` は文字列の一致しか見ないので、id を書くだけで検査は
    通ってしまう。名指しが実在の検証に付いていることの担保は、production を崩したときにそのテストが
    落ちるという観測しかない。
- **Change-Resistance Results**:
  5 行に対する 21 件の故障すべてについて、行が言っていることを production 側で崩し、対応するテストが落ちることを
  観測した。故障は注入のたびに `git diff --stat` で着弾を確かめ、観測後に元へ戻している。
  ビルドが通らないだけの注入は観測にならないので、1 件（代行チェーンを `Context` へ渡さない）は
  `actors[:0]` を渡す形へ組み直した。

  | 行 | 注入した故障 | 落ちたテストと観測 |
  |---|---|---|
  | `AUTHZEN-FGA-EVALUATION` | `AuthZContext` から `Relationship` を外す | `…RidesOnTheSubjectActionResourceContextEvaluation/the evaluation carries…`: 規則表が `relationship_facts_present` で落とし、事実は `<nil>` |
  | 〃 | 事実の `Evaluated` を `false` にする | 同テスト: `Evaluated:false` のまま届く |
  | 〃 | 結果を `response.Permit` ではなく事実の論理積にする | `…/the evaluator's answer is the decision`: 評価器が許可と答えた要求が拒否になる |
  | 〃 | 事実の `RelationPath` を落とす | `…/the evaluation carries…`: `RelationPath` が空 |
  | `AUTHZEN-FGA-ACTOR-CHAIN` | 各段の `Active` を無条件に `true` にする | `…CarriesEveryDelegationStageSeparately/each stage keeps…`: 無効化された代行者も登録の無い代行者も `Active:true` |
  | 〃 | `Context.ActorChain` へ `actors[:0]` を渡す | 同テスト: チェーンが空で届く |
  | 〃 | 種別と識別子を `agent:researcher` の 1 本へ連結する | 同テスト: `Type` が空になり各段が分離されていない |
  | `AUTHZEN-FGA-FAIL-CLOSED` | 評価器の error を `Permit: true` に変える | `…FailsClosedWhenTheDecisionCannotBeAnswered/an evaluator that cannot answer…`: 200 で `permitted:true` |
  | 〃 | 評価器 `nil` を許可で返す | `…/a missing evaluator…`: 200 で `permitted:true` |
  | 〃 | 関係評価のストア error を `Permitted: true` の判定に差し替える | `…/an unreachable relation store…`: 200 で `permitted:true` |
  | 〃 | handler が判定の error を握って `permitted: true` を返す | **5 区画のうち 4 つが同時に落ちる**。原因ごとに別の区画が観測していることが、この 1 件で読める |
  | 〃 | 深さ上限で `return true`（開いて諦める） | `…/a graph deeper than the traversal bound…`: 上限を超えた連なりが許可になる |
  | 〃 | `DefaultMaxDepth` を 8 から 64 へ上げる | 同区画: 実質的に上限が無くなり許可になる |
  | `RFC8693-FGA-ACTOR-AND` | チェーンを積ではなく和にする | `…RequiresTheRelationOnEveryActorInTheChain`: 2 段のうち 1 段しか持たない 2 通りが許可になる |
  | 〃 | チェーンの先頭 1 段だけを判定する | 同テスト: `subject=true researcher=true assistant=false` が許可になる |
  | 〃 | `SubjectPermitted` を無条件に `true` にする | 同テスト: `subject=false researcher=true assistant=true` が許可になる |
  | 〃 | チェーンの判定結果を捨てる | 同テスト: `subject=true` の 3 通りがすべて許可になる |
  | `AUTHZEN-FGA-SEARCH` | 既定の上限を 500 から 100000 へ上げる | `…IsSubjectFixedBoundedAndReportsTruncation/the default traversal bound…`: 501 件がそのまま返る |
  | 〃 | `truncated` の判定を `false` に固定する | 同区画: 打ち切らず 501 件が返る |
  | 〃 | 結果の `Truncated` だけを `false` にする | 同区画: 500 件で止まっているのに完全な一覧に見える |
  | 〃 | 許可されたものだけを返す絞り込みを外す | `…/the search returns exactly what the fixed subject reaches`: `[d1 d2 d3]` が返る |
- **Verification Results**:
  - `mise run verify` - passed
