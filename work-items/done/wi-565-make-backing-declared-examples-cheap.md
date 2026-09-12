---
depends_on: []
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-12
priority: p2
change_kind: maintenance
spec_impact: { kind: none, reason: "テスト基盤、台帳の書式、報告タスク、エージェント指示だけを変える。製品の振る舞いも公開契約も規範文書も動かさない。規範文書へ観測点を書き込む案は検討したうえで取り下げた。理由は Design に書く。" }
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 変わるのはエージェント向けの指示と開発用の道具だけであり、利用者向けの契約も運用手順も動かない。
  references: []
initial_context:
  specification: []
  typespec: []
  source:
    - tools/check/src/check-documents.ts
    - tools/check/src/normative-coverage.ts
    - tools/check/src/gherkin-scenarios.ts
    - tools/check/src/coverage-debt-ratchet.ts
    - tools/coverage-debt-report/src/main.ts
    - tools/check/example-coverage-debt.json
    - mise.toml
    - AGENTS.md
    - .claude/skills/implement-work-item/SKILL.md
    - docs/development/specification-first-workflow.md
    - DOCUMENTATION_GUIDE.md
    - SPECIFICATION_FORMAT.md
  tests:
    - backend/shared/http/server_http/api_token_standards_test.go
    - backend/oauth2/handlers_http/oauth2_scope_examples_test.go
    - backend/shared/http/server_http/token_issuance_standards_test.go
    - backend/authentication/handlers_http/account_consent_refusal_effects_test.go
    - tools/check/src/normative-coverage.test.ts
  stop_before_reading:
    - backend/oauth2/db_postgres
    - frontend
    - infra
---

# 宣言済みの具体例にテストを対応付ける 1 件あたりの費用を、テスト基盤と台帳と索引の側から下げる

## Motivation

[[wi-538-back-oauth2-examples-with-tests]] は OAuth2 の具体例 6 件を消化し、そのうえで残る 99 件を 5 つの子 work item へ割った。割った理由は 1 件あたりの所要が高かったことである。しかし所要が高い原因は具体例の側ではなく、**消化する側の道具の側にあった。** 分割は原因ではなく症状への対処であり、固定費を 5 倍にする。

原因は測定できる。

| 事実 | 測定値 |
| --- | --- |
| `Register(e, Deps{…})` を手で組み立てているテストファイル | 94 |
| `Deps` のフィールド数 | 44 |
| 配線先の in-memory リポジトリのディレクトリ | 39 |
| `api_token_standards_test.go` の行数 | 1003 |
| `token_issuance_standards_test.go` の行数 | 857 |
| 非テストの Go が名指す `REQ-*` id の総数 | 82（OAuth2 は 20） |
| `scenarios.feature.md` が持つ構造化された項目 | `Primary actor:` のみ |
| 台帳の行数と、そのうち `reason` が同一文字列の行 | 568 / 568 |

**wi-538 が EX-OAUTH2-001-01 を子項目へ回した理由は、主張の難しさではなく、認可コードと `/token` と account リソースサーバーを同時に配線したスタックが 94 個のどれにも無かったことである。** 観測したい振る舞いではなく fixture の有無が work item の範囲を決めた。これが本項目の出発点である。

加えて、wi-538 は故障注入 5 件を手書きした。`mise run test-go-mutation` が `mise.toml:444` に既にあり、[仕様先行の開発ワークフロー](../../docs/development/specification-first-workflow.md)の Mutation testing 節がその使い方と読み方を定めているのに、である。指示が届いていない場所に置かれていた。

## Scope

- **テストスタックの組み立てを 1 箇所へ寄せる。** `backend/shared/http/testing_stack` を新設し、`Register` が要求する配線を option の合成で組み立てられるようにする。既定は製品の組み立てと同じにする。保存先は型付きの handle として公開し、拒否が防いだ効果を読み直せるようにする。
- **移行は新規と、本項目が触った呼び出し元だけにとどめる。** 94 個すべての書き換えは行わない。`oauth2_scope_examples_test.go` を新基盤へ載せ替え、**EX-OAUTH2-001-01 が option の合成だけで到達できることを実測で示す**。示せなければ設計が足りていない。
- **台帳に、導出できない判断だけを持たせる。** `blocked_by` と `finding` を任意項目として追加し、検査を通す。導出できるもの（経路、パッケージ、拒否かどうか）は持たせない。索引が出すためである。
- **具体例 1 件から観測点を引ける索引タスクを作る。** `mise run spec-route -- <EX または REQ の id>` が、規則、当の具体例の Given/When/Then、契約が宣言する候補 operation とそのメソッド・パス・スコープ、同じ規則の隣の id を名指している既存テストを返す。`tools/coverage-debt-report` が既に持っている解析を共有する。
- **変異テストを既定の手順として指示へ書く。** `AGENTS.md` に 1 行、`implement-work-item` スキルの証拠収集の段に 1 文。
- **work item を量で分割しない方針を `implement-work-item` スキルへ書く。**
- **wi-538 が起こした 5 つの子 work item を 1 つへ畳み直す。**
- **Context ごとの兄弟記録 17 件を、新しい道具を使う形へ更新する。** 統合はしない。理由は Design に書く。

## Out of Scope

- 94 個の既存 fixture の全面移行。本項目が触らない呼び出し元はそのまま残す。移行は具体例を消化する work item が、その都度触る範囲で行う。
- 具体例そのものの消化。[[wi-559-back-oauth2-remaining-examples-with-tests]] が持つ。
- `scenarios.feature.md` への観測点の記載。検討して取り下げた。理由は Design に書く。
- 台帳への `entry_point` などの導出可能な項目の追加。索引が出す。
- 行カバレッジ率の目標または閾値。
- CI のゲートへ `spec-route` や `test-go-mutation` を足すこと。どちらも報告であって門ではない。

## Design

### 規範文書へ観測点を書かない理由

案として「各 Example に `Observed at: POST /api/admin/v1/clients` を必須にする」を検討した。経路探索の費用は確かにゼロになる。取り下げたのは次の 2 点による。

第 1 に、[DOCUMENTATION_GUIDE](../../DOCUMENTATION_GUIDE.md) §5 は HTTP ルートを TypeSpec の持ち物と定めている。ルートを `scenarios.feature.md` へ書けば、同じ事実の一次情報源が 2 つになる。

第 2 に、19 コンテキストの `scenarios.feature.md` は `backend/` も `frontend/` も 1 箇所も参照していない。規範は実装を知らない、という分担が実際に守られている。`domain:backend/oauth2/authorization` のような観測点はこれを崩す。

必要なのは事実の記載ではなく、**導出**である。TypeSpec は operation ごとにメソッド、パス、宣言するエラーの直和、スコープを持つ。具体例の `Then` はそのエラーの型を名指す。両者を突き合わせれば観測点は計算できる。だから `spec-route` を作る。

### 3 つの道具の分担

境界は「導出できるか」で引く。

| 道具 | 持つもの | 持たない理由 |
| --- | --- | --- |
| `mise run spec-route` | 規則、具体例の本文、候補 operation とメソッド・パス・スコープ、隣の id を名指す既存テスト | すべて TypeSpec とテスト本文から計算できる。保存すると古くなる |
| `example-coverage-debt.json` の `blocked_by` と `finding` | 読んで初めて分かった判断。先に決着すべき work item と、実装が具体例と食い違う事実 | 計算できない。今日は完了した work item の散文にしか無く、次の読み手が同じ測定をやり直す |
| `backend/shared/http/testing_stack` | `Register` へ渡す配線の組み立てと、保存先の型付き handle | — |

`entry_point` を台帳へ持たせないのはこの分担による。`spec-route` が出せるものを台帳へ書くと、台帳が古くなったときにどちらが正か分からなくなる。

### `testing_stack` の option の切り方

44 フィールドを 44 個の option に置き換えても費用は下がらない。option は**具体例が観測したい入口**の単位で切る。

| option | 配線されるもの |
| --- | --- |
| `WithOAuth2Clients()` | クライアントの保存先と管理 API |
| `WithAuthorizationCodeFlow()` | 認可リクエスト、認可コード、PAR、同意、認証文脈の解決 |
| `WithTokenIssuance()` | 署名鍵、署名器、リフレッシュ、失効リスト |
| `WithApiTokens()` | 管理発行トークンの記録と overlay 付きイントロスペクション |
| `WithAccountApi()` | account 用の同意 API と利用者の保存先 |
| `WithSaml()` | SAML の保存先と管理 API |

既定は製品の組み立てと同じにする。とくに `WithApiTokens()` は `TokenIntrospector` に生の署名検証器ではなく `apitokenusecases` の overlay を渡す。`api_token_standards_test.go` が 10 行かけて警告しているとおり、ここを間違えると失効したトークンが有効に見え、それは製品の欠陥ではなく組み立ての違いになる。この既定を 1 箇所へ寄せることが共有化の主目的である。

`Stack` は保存先を型付きの field として公開する。応答だけを読むテストは拒否を書いてから保存する実装を見分けられないので、**効果を読み直せることが基盤の要件**である。

### 設計が足りているかの判定

`EX-OAUTH2-001-01` を到達可能にできるかで判定する。この具体例は認可コード + PKCE の交換とその後の account リソースサーバーの参照を 1 本で言っていて、[[wi-538-back-oauth2-examples-with-tests]] が「94 個のどのスタックにも無い」として子項目へ回した唯一の件である。

```go
stack := testing_stack.New(t,
    testing_stack.WithAuthorizationCodeFlow(),
    testing_stack.WithTokenIssuance(),
    testing_stack.WithAccountApi())
```

これで組み上がらなければ option の切り方が間違っている。本項目は到達可能であることを示すところまでを持ち、具体例そのものの消化は [[wi-559-back-oauth2-remaining-examples-with-tests]] が持つ。

### Context ごとの兄弟記録を統合しない理由

`wi-539` から `wi-557` は Context ごとに 1 記録で、残り 467 件を持つ。件数は 4 件から 71 件まで開いている。**統合しない。**

分割の可否は量ではなく意味で決まる。この 17 記録を割ったのは量ではなく Context であり、Context は境界づけられた文脈そのものである。それぞれが自分の `scenarios.feature.md` と自分の入口と自分の fixture を持つ。2 つの Context を 1 記録へ入れることは、読み手が片方だけを受け入れられない差分を作ることであり、`implement-work-item` へ書いた規則が「割ってよい」と言っている側に当たる。

件数の少ない記録（`cross-context` の 4 件、`data-keys` の 10 件）にも同じことが言える。固定費は確かに割高だが、寄せ先として意味のある相手がいない。固定費を下げたいなら、寄せるのではなく固定費そのものを下げる。本項目がやっているのはそれである。

代わりに 17 記録すべてへ「作業の進め方」を足した。どれも本項目より前に書かれていて、`spec-route` も `testing_stack` も `test-go-mutation` も知らないままだった。知らなければ、各記録が同じ経路探索をやり直す。

### 親の測定値の誤記

書き写しの誤りが雛形ごと 22 記録へ伝播していた。[[wi-496-burn-down-the-example-coverage-debt]] は「16 件のうち、注記だけで済んだのは 4 件（25 %）である」と測っているのに、子記録の Motivation はすべて「2 件だけ」と書いていた。同じ記録の Risk Notes には正しい 4 件があるので、誤りは Motivation の 1 文に限られる。

誤りの向きは所見を実際より強く見せる側だった。22 記録すべてを親の数字へ揃え、内訳（3 件はテストが 1 つも無く、9 件は既存テストへ観測を足す必要があった）も親から写した。

### 畳み直し

[[wi-538-back-oauth2-examples-with-tests]] が起こした wi-559 から wi-563 を 1 つへ畳む。割った根拠は 1 件あたりの所要であり、本項目がその所要を下げる以上、根拠が消える。5 記録は readiness と Design と Completion の固定費を 5 回払わせるだけになる。

割ってよいのは意味が 2 つあるときだけである。量は理由にならない。これを `implement-work-item` スキルへ書く。

## Plan

1. wi-559 から wi-563 を 1 つへ畳み、wi-538 の参照を更新する。
2. 指示を直す。`AGENTS.md` の表に変異テストの行、スキルに変異テストの 1 文と分割の方針。
3. 台帳の `blocked_by` を RED から実装し、EX-OAUTH2-003-04 の判断を台帳へ移す。
4. `spec-route` を RED から実装する。
5. `testing_stack` を RED から実装し、`oauth2_scope_examples_test.go` を載せ替え、EX-OAUTH2-001-01 の組み立てを実測する。

## Tasks

- [x] T001 [Records] wi-559 から wi-563 を [[wi-559-back-oauth2-remaining-examples-with-tests]] へ畳み、wi-538 の Design と Completion の参照を更新する。`mise run check-work-items`。
- [x] T002 [Docs] `AGENTS.md` に変異テストの行を足し、`implement-work-item` スキルへ変異テストと分割方針を書く。`mise run check`。
- [x] T003 [Tooling] 台帳の `blocked_by` と `finding` を受け付け、存在しない work item を名指したら落ちるようにする。`mise run test-tools-file -- tools/check/src/normative-coverage.test.ts`。
- [x] T004 [Tooling] `mise run spec-route -- <id>` を作る。`tools/coverage-debt-report` の解析を共有する。
- [x] T005 [Tests] `backend/shared/http/testing_stack` を作り、`oauth2_scope_examples_test.go` を `backend/oauth2/handlers_http` へ載せ替える。`mise run test-go-package -- ./backend/oauth2/handlers_http`。
- [x] T006 [Tests] `EX-OAUTH2-001-01` のスタックが option の合成だけで組み上がることを実測する。
- [x] T007 [Records] Context ごとの兄弟記録 17 件へ「作業の進め方」を足し、親の測定値の誤記を直す。`mise run check-work-items`。
- [x] T008 [Verify] `mise run verify`。

## Verification

- `mise run spec-route -- EX-OAUTH2-001-01` が、規則、候補 operation、隣の id を名指す既存テストを返す。
- `mise run check-spec` が、`blocked_by` と `finding` を持つ台帳に対して通る。存在しない work item を `blocked_by` に書くと落ちる。
- `mise run test-go-package -- ./backend/shared/http/server_http`
- `mise run verify`

## Risk Notes

- **共有スタックが製品と違う組み立てを配ってしまう。** `api_token_standards_test.go` は「overlay を外した配線では失効したトークンが有効に見える。それは製品の欠陥ではなく組み立ての違いである」と 10 行かけて警告している。共有化はこの間違いを 1 箇所に集める代わり、間違えたときの影響を全呼び出し元へ広げる。既定を製品の組み立てと一致させ、一致していることを検査で読む。
- **option が増えすぎて、結局どれを呼ぶか分からなくなる。** 44 フィールドを 44 個の option に置き換えるだけなら費用は下がらない。option は「具体例が観測したい入口」の単位で切る。
- **索引が推測を出して、それが根拠として扱われる。** `report-coverage-debt` の分類が根拠にならないのと同じ理由である。`spec-route` の出力は読む順を決める材料であり、台帳から外す根拠にはしない。出力そのものにそう書く。
- **畳み直した work item が、また量を理由に割られる。** 割ってよいのは意味が 2 つあるときだけである。スキルへ書くのはそのためである。

## Completion

- **Completed At**: 2026-09-12
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。規範も製品コードも 1 行も動いていない。
  変わったのは、宣言済みの具体例へテストを対応付ける作業の 1 件あたりの費用である。
  観測点を探す作業は `mise run spec-route` が計算するようになり、`Register` と同じ配線は
  `backend/shared/http/testing_stack` の option 合成で建つようになり、読んで初めて分かった判断は
  台帳の行そのもの (`blocked_by`、`finding`) に残るようになった。
  wi-538 が量を理由に割った 5 記録は 1 つへ畳み直し、割らない方針を `implement-work-item` へ書いた。
  Context ごとの兄弟記録 17 件は統合せず、新しい道具を使う形へ更新した。
- **Acceptance RED Evidence**:
  - **Test**: `mise run spec-route -- EX-OAUTH2-001-01`
  - **Requirement**: N/A: 規範も製品の振る舞いも変えないため、対応する `REQ-*` が無い。
  - **Observed Failure**: タスクが存在せず、mise が `... and 98 more tasks` を出して終了した。
    実装後は同じ引数で、規則 REQ-OAUTH2-001、具体例の 5 ステップ、契約が `/token` で結合した
    候補 operation (`Token: POST /token [path /token]`)、および隣の id を名指す既存テスト 2 本
    (`refusal_effects_test.go` の EX-OAUTH2-001-02、`endpoint_refusal_effects_test.go` の
    EX-OAUTH2-001-03) を返す。これは wi-538 でこの 1 件について手で導いた答えと一致する。
  - **Detection Reason**: この作業に観測可能な境界があるとすれば「id を渡したら観測点が返るか」であり、
    それはタスクの有無そのものである。返る内容の正しさは別に単体テストで固定した
    (`tools/spec-route/src/route.test.ts`、12 件)。
- **Unit RED Evidence**:
  - **Test**: `mise run test-tools-file -- check/src/normative-coverage.test.ts`
    (`checkNormativeCoverage: 台帳が持つ判断`)
  - **Requirement**: N/A: 台帳は検査の道具であり、規範の要求ではない。
  - **Observed Failure**: 3 件が落ちた。存在しない work item を名指した `blocked_by` が通ってしまう件、
    `finding` の無い `blocked_by` が通ってしまう件、空の `finding` が通ってしまう件である。
    実装後は 22 件すべて緑。
  - **Detection Reason**: この 3 つの規則は「読んだ結果を残したふり」を落とすためにある。
    `reason` が既に平の場合を守っているので、新しい規則が守るのは
    「誰かが測ったのに、その結果が次の読み手に届かない」場合だけである。
    配線した側も観測した。実台帳の `EX-OAUTH2-003-04` の `blocked_by` を
    `wi-999-does-not-exist` に差し替えると `mise run check-spec` が
    `EX-OAUTH2-003-04 is blocked by wi-999-does-not-exist, which is not a work item.` で落ちる。
- **Change-Resistance Results**:
  リスクは medium である。追加した Go は配線であり、追加した TypeScript は純粋な結合である。
  それぞれに合う方法を使った。
  1. **変異テスト（新しい規則どおり、手書きより先に道具を使った）。**
     `mise run test-go-mutation -- backend/shared/http/testing_stack` は初回 Killed 16 / Lived 2 /
     Not covered 0 を返した。生存 2 件はどちらも「同意の保存先を作る冪等ガード」の条件反転で、
     `WithAuthorizationCodeFlow` と `WithAccountApi` が互いを覆い隠していた。等価変異ではなく
     観測の穴だったので、各 option を単独で渡すテストを足した
     (`TestEachOptionThatNeedsConsentsWiresItOnItsOwn`)。再実行して Killed 18 / Lived 0。
  2. **手書きの故障注入（変異器が表現できない「配線の差し替え」）。**
     `WithApiTokens` が `OAuth2.TokenIntrospector` へ渡す overlay を生の署名検証器へ戻すと、
     `TestApiTokenIntrospectionSeesTheRevocationRecord` が
     `失効したトークンが active=true である` で落ちる。
     **この観測は 1 度書き直している。** 最初は失効トークンで管理 API を叩く形にしていたが、
     故障を入れても落ちなかった。管理 API の入口は `ApiTokens` module 側の照合を通るためで、
     overlay が効くのは `/introspect` だけだった。組み立ての違いは、それが効く入口でしか読めない。
  3. `requireAdminApiTokenScope` の先頭へ `return nil` を置いて粒度スコープを素通しにすると、
     `testing_stack` へ移行したあとの 3 本
     (`TestOAuth2AdminOperationsFollowTheGranularScopes`、
     `TestOAuth2ReadScopeCannotChangeOAuth2Clients`、
     `TestOAuth2ScopeOfAnotherResourceCannotReachTheOperation`) がそろって落ちる。
     移行の前後で検出能力が変わっていないことの観測である。
  4. `mise run lint-go` が、基盤の初版が `Deps.UserRepo`、`Deps.KeyStore`、`Deps.TokenIssuer` という
     移行期の互換入力を使っていることを検出した (staticcheck SA1019 が 3 件)。
     bootstrap は module しか設定しないので、互換入力を使う基盤は製品と違う経路の組み立てを
     全呼び出し元へ配ることになる。module 経由へ直した。**この項目の主目的そのものに対する検出**
     であり、共有化の危険を検査が先に捕まえたことになる。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run test-go-mutation -- backend/shared/http/testing_stack` - Killed 18 / Lived 0 / Not covered 0
  - `mise run check-work-items` - passed (559 record(s))
  - `mise run test-ui-e2e` - 実行していない。フロントエンドも製品の Go コードも 1 行も変わっていない。

### 直した道具の欠陥

- **`mise run test-go-package` が故障注入に対して偽陰性を返した。** `-count 1` が無いため、
  製品側へ故障を入れて同じパッケージを回すと `ok (cached)` が返る。
  本項目の作業中に実際に 1 度誤読し、検出できているものを「検出できない」と判断しかけた。
  `-count 1` を足した。キャッシュへの書き込みは残るので、最後のゲートへ引き継がれる分は変わらない
  （`-count 1` の直後に素で回すと `(cached)` になることを実測した）。
- **`mise run test-tools-file` が無かった。** 1 ファイルだけ回す手段が無く、`bun test` の直接実行へ
  倒れる。AGENTS.md の「一般的な操作にタスクがなければ追加する」に従って追加した。

### 費用がどう変わるか

wi-538 が 6 件で測ったとき、1 件あたりの作業は「観測点を探す → fixture を探す → 無いので建てるか諦める」
だった。3 つとも別の道具が引き取る。

| wi-538 で払っていたもの | いま |
| --- | --- |
| `routes.go`、`operations_gen.go`、ハンドラを読んで観測点を決める | `mise run spec-route -- <id>` |
| 94 個から使える fixture を探し、無ければ足す | `testing_stack.New(t, ...)` の option 合成 |
| 故障注入を手書きする | `mise run test-go-mutation`（表現できない分だけ手書き） |
| 読んで分かった判断を散文に書き、次の読み手が測り直す | 台帳の `blocked_by` と `finding` |

`EX-OAUTH2-001-01` はこの変化の 1 件目である。wi-538 では「94 個のどの fixture にも無い」ことを理由に
子項目へ回したが、いまは option 3 つで建つ (`TestAuthorizationCodeAndAccountApiComposeIntoOneStack`)。
消化そのものは [[wi-559-back-oauth2-remaining-examples-with-tests]] が持つ。

### 残っている 467 件の配置

| 記録 | Context | 件数 |
| --- | --- | --- |
| [[wi-559-back-oauth2-remaining-examples-with-tests]] | oauth2 | 99 |
| [[wi-539-back-authentication-examples-with-tests]] | authentication | 71 |
| [[wi-540-back-identity-management-examples-with-tests]] | identity-management | 66 |
| [[wi-541-back-system-examples-with-tests]] | system | 45 |
| [[wi-542-back-tenancy-examples-with-tests]] | tenancy | 39 |
| [[wi-543-back-application-examples-with-tests]] | application | 32 |
| [[wi-544-back-authorization-examples-with-tests]] | authorization | 31 |
| [[wi-545-back-provisioning-examples-with-tests]] | provisioning | 27 |
| [[wi-546-back-sourcing-examples-with-tests]] | sourcing | 25 |
| [[wi-547-back-jobs-examples-with-tests]] | jobs | 21 |
| [[wi-548-back-identity-governance-examples-with-tests]] | identity-governance | 21 |
| [[wi-549-back-sharedsignals-examples-with-tests]] | sharedsignals | 20 |
| [[wi-551-back-api-tokens-examples-with-tests]] | api-tokens | 16 |
| [[wi-552-back-signing-keys-examples-with-tests]] | signing-keys | 14 |
| [[wi-553-back-audit-examples-with-tests]] | audit | 14 |
| [[wi-554-back-ws-federation-examples-with-tests]] | ws-federation | 11 |
| [[wi-555-back-data-keys-examples-with-tests]] | data-keys | 10 |
| [[wi-557-back-cross-context-examples-with-tests]] | (cross-context) | 4 |
| [[wi-564-name-the-refusal-a-cross-tenant-oauth-admin-token-gets]] | oauth2 (食い違い) | 1 |

台帳の 568 件のうち、この表が 566 件を持つ。残る 2 件は SAML の消化済み分に対応する。
