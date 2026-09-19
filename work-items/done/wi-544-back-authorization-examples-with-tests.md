---
depends_on: [wi-565-make-backing-declared-examples-cheap]
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-12
priority: p3
change_kind: maintenance
evidence_policy: risk-based-v3
spec_impact: { kind: none, reason: "宣言済みの具体例に、その id を名指しするテストを対応付ける作業である。シナリオも製品の振る舞いも変えない。テストが書けない具体例が見つかった場合、それは実装が具体例のとおりに振る舞っていないということなので、欠陥として個別の work item に切り出す。" }
documentation_impact:
  level: none
  reason: テストと被覆台帳だけを変え、利用者が観測できる振る舞いも公開契約も変えないため。
  references: []
initial_context:
  specification:
    - docs/domain/authorization/scenarios.feature.md#REQ-AUTHORIZATION-001
    - docs/domain/authorization/scenarios.feature.md#REQ-AUTHORIZATION-002
    - docs/domain/authorization/scenarios.feature.md#REQ-AUTHORIZATION-003
    - docs/domain/authorization/scenarios.feature.md#REQ-AUTHORIZATION-004
    - docs/domain/authorization/scenarios.feature.md#REQ-AUTHORIZATION-005
    - docs/domain/authorization/scenarios.feature.md#REQ-AUTHORIZATION-006
    - docs/domain/authorization/scenarios.feature.md#REQ-AUTHORIZATION-007
    - docs/domain/authorization/scenarios.feature.md#REQ-AUTHORIZATION-008
    - docs/domain/authorization/scenarios.feature.md#REQ-AUTHORIZATION-009
    - docs/domain/authorization/scenarios.feature.md#REQ-AUTHORIZATION-010
  typespec: []
  source:
    - backend/authorization/usecases
    - backend/authorization/domain/events.go
    - backend/authorization/db_memory/repository.go
    - backend/authorization/handlers_http/routes.go
    - tools/check/src/normative-coverage.ts
  tests:
    - backend/authorization/usecases/check_access_test.go
    - backend/authorization/domain/model_test.go
    - backend/authorization/domain/evaluator_test.go
    - backend/authorization/handlers_http/routes_test.go
    - backend/authorization/handlers_http/standards_test.go
    - backend/authorization/db_postgres/repository_test.go
  stop_before_reading:
    - frontend
    - spec/contexts/authorization
    - backend/authorization/db_postgres/authorization.sql.go
    - backend/shared/http/testing_stack
---

# Authorization が宣言する具体例 31 件にテストを対応付け、被覆台帳から外す

## Motivation

[[wi-496-burn-down-the-example-coverage-debt]] は具体例の被覆台帳の消化単位を Context と決め、測定のうえで残りを Context ごとの子 work item へ割った。本項目はそのうち `docs/domain/authorization/scenarios.feature.md` が宣言する 31 件を引き取る。

親項目が `claim-mapping` の 3 件と `workloadidentity` の 13 件で測った結果は、**16 件のうちディレクティブだけで済んだのは 4 件だけ**だというものである。3 件はテストが 1 つも無く、9 件は既存テストへ新しい観測を足す必要があった。件数は作業量の目安にならない。

## Scope

- 31 件を 1 件ずつ確認し、次のいずれかに解決して `tools/check/example-coverage-debt.json` から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `//spec:covers EX-AUTHORIZATION-NNN-MM: <この具体例の何を固定しているか>` のディレクティブを足す。
  - 当の具体例を検証しているテストが無い → 書く。具体例が拒否なら、[[wi-392-refusal-tests-assert-the-absent-effect]] が定める形（拒否応答と、拒否が防いだ効果の双方を観測する）で書く。
  - 具体例が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- ディレクティブは「何を固定しているか」を書く。id だけのディレクティブは禁止する。
- 実装が具体例のとおりに振る舞っていないことが分かった場合は、**本 work item では直さず欠陥として切り出す**。

## Out of Scope

- 他の Context が宣言する具体例。Context ごとに別の work item が持つ。
- `tools/check/standards-coverage-debt.json`。[[wi-495-burn-down-the-standards-coverage-debt]] が消化済みである。
- 既に台帳に載っていない拒否テストが、防いだ効果を観測していない件。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。本項目が新しく書く拒否テストは、その規範を満たす形で書く。既存テストにディレクティブを足すだけの件で、そのテストが効果の不在を見ていない場合は、ディレクティブを足したうえで wi-392 の対象として残す。
- シナリオと具体例の追加、削除、書き換え。具体例の記述が実装と食い違うことが判明した場合は規範の変更なので、別の work item が扱う。
- 見つかった実装の欠陥の修正。切り出した先で扱う。
- 行カバレッジ率の目標または閾値。
- `report-coverage-debt` の分類（`named`、`nearby`、`none`）を根拠にした台帳からの削除。分類は読む順を決める材料であり、台帳から外す根拠にはならない。

## 作業の進め方

[[wi-565-make-backing-declared-examples-cheap]] が、この作業の 1 件あたりの費用を下げる道具を用意した。使う。

- 読む順と観測点は `mise run spec-route -- <id>` が出す。規則、当の具体例の本文、契約が宣言する候補 operation とそのメソッド・パス・スコープ、同じ規則の隣の id を名指している既存テストが返る。**出力は読む順を決める材料であり、台帳から外す根拠にはしない。**
- 新しく書くテストは `backend/shared/http/testing_stack` の上に載せる。`Register` と同じ配線が option の合成で建ち、保存先は型付きの field から読み直せる。既存 fixture の全面移行はしない。触る必要が出た範囲だけ移す。
- 変更した Go の変異は `mise run test-go-mutation -- <package-directory>` で読む。手で書く故障注入は、変異器が表現できない「配線を外す」「既定の分岐を差し替える」種類だけに残す。
- 実装が具体例と食い違って消化できない件は、台帳の当該行へ `blocked_by`（先に決着すべき work item）と `finding`（実装が実際に何を返すか）を書いて残す。散文にだけ書くと、次の読み手が同じ測定をやり直す。

## Design

31 件を 1 件ずつ読み、具体例の `Then` と既存テストの観測を突き合わせた結果が次の表である。
`処置` は 3 通りに分かれる。

- **ディレクティブ**：既存テストが `Then` をすべて観測している。id を名指すディレクティブだけを足す。
- **観測追加**：既存テストが同じ入力を通しているが、`Then` の一部を見ていない。観測を足したうえでディレクティブを付ける。
- **新規**：その具体例を通すテストが無い。書く。

| id | 消化先 | 処置 | 足りない観測 |
| --- | --- | --- | --- |
| EX-AUTHORIZATION-001-01 | `usecases/check_access_test.go` | 新規 | 版が単調増加すること、以前の版が書き換わらないこと、`GetAuthorizationModel` が新しい版を最新として返すこと |
| EX-AUTHORIZATION-001-02 | `usecases/check_access_test.go` `TestPutAuthorizationModelRejectsAnInconsistentModel` | ディレクティブ | なし |
| EX-AUTHORIZATION-001-03 | `usecases/check_access_test.go` | 新規 | 書き換え規則が循環するモデルを拒否したときに版が増えないこと |
| EX-AUTHORIZATION-001-04 | `usecases/check_access_test.go` | 新規 | 型名または関係名がフォーマットに反するモデルを拒否したときに版が増えないこと |
| EX-AUTHORIZATION-002-01 | `usecases/check_access_test.go` | 新規 | 追加と削除が 1 回で適用されること、既存の組の再追加が冪等であること、返った整合トークンを以後の判定へ渡せること |
| EX-AUTHORIZATION-002-02 | `usecases/check_access_test.go` `TestWriteRelationTuplesRejectsTheWholeDiff` | ディレクティブ | なし |
| EX-AUTHORIZATION-002-03 | `usecases/check_access_test.go` | 新規 | `direct` が許していない主体型とワイルドカードを含む差分が 1 件も適用されないこと |
| EX-AUTHORIZATION-002-04 | `usecases/check_access_test.go` `TestWriteRelationTuplesRejectsTheWholeDiff` | 観測追加 | 追加と削除の双方に現れる組を拒否したあと、保管庫が空のままであること |
| EX-AUTHORIZATION-002-05 | `usecases/check_access_test.go` | 新規 | モデル未登録のテナントへの書き込みが `ErrModelNotFound` になること |
| EX-AUTHORIZATION-003-01 | `domain/evaluator_test.go` `TestCheckTraversesGroupAndParent`、`TestCheckPathOmitsIdentifiers` | ディレクティブ | なし |
| EX-AUTHORIZATION-003-02 | `domain/evaluator_test.go` `TestCheckTraversesGroupAndParent` | 観測追加 | subject set だけをたどる経路。既存の表は親と group を兼ねる 1 件しか持たない |
| EX-AUTHORIZATION-003-03 | `domain/evaluator_test.go` `TestCheckTraversesGroupAndParent` | 観測追加 | 親オブジェクトだけをたどる経路。同上 |
| EX-AUTHORIZATION-003-04 | `domain/evaluator_test.go` `TestCheckTraversesGroupAndParent` | ディレクティブ | なし |
| EX-AUTHORIZATION-004-01 | `usecases/check_access_test.go` `TestCheckAccessRequiresSubjectAndActorChain` | 観測追加 | 表が言う「スコープが含まれる」を、要求スコープを実際に立てた入力で観測すること |
| EX-AUTHORIZATION-004-02 | `usecases/check_access_test.go` `TestCheckAccessRequiresSubjectAndActorChain` | ディレクティブ | なし |
| EX-AUTHORIZATION-004-03 | `usecases/check_access_test.go` `TestCheckAccessRequiresSubjectAndActorChain` | ディレクティブ | なし |
| EX-AUTHORIZATION-004-04 | `usecases/check_access_test.go` `TestCheckAccessRequiresSubjectAndActorChain` | ディレクティブ | なし |
| EX-AUTHORIZATION-005-01 | `usecases/check_access_test.go` `TestCheckAccessFailsClosed` | 観測追加 | 答えの出ない判定が許可にならないことに加え、拒否した規則名が結果に残ること |
| EX-AUTHORIZATION-005-02 | `domain/evaluator_test.go` `TestCheckDeniesOnCycleAndDepth` | ディレクティブ | なし |
| EX-AUTHORIZATION-005-03 | `domain/evaluator_test.go` `TestCheckDeniesOnCycleAndDepth` | ディレクティブ | なし |
| EX-AUTHORIZATION-005-04 | `usecases/check_access_test.go` `TestCheckAccessFailsClosed` | ディレクティブ | なし |
| EX-AUTHORIZATION-005-05 | `usecases/check_access_test.go` `TestCheckAccessFailsClosed` | ディレクティブ | なし |
| EX-AUTHORIZATION-006-01 | `usecases/check_access_test.go` | 新規 | 別テナントに同じ識別子のタプルがあっても読み出されず、判定が不許可になること |
| EX-AUTHORIZATION-006-02 | `usecases/check_access_test.go` | 新規 | 同じ識別子でも対象テナントは呼び出し元のまま変わらないこと |
| EX-AUTHORIZATION-006-03 | `usecases/check_access_test.go` `TestCheckAccessRejectsUnsatisfiedConsistency` | ディレクティブ | なし |
| EX-AUTHORIZATION-007-01 | `usecases/check_access_test.go` `TestListAccessibleResourcesIsBoundedAndFiltered` | 観測追加 | 列挙が `CheckAccess` と同じ合成を通り、代行チェーンも同じく評価されること |
| EX-AUTHORIZATION-007-02 | `usecases/check_access_test.go` `TestListAccessibleResourcesIsBoundedAndFiltered` | 観測追加 | 上限ちょうどが打ち切りにならないこと。変異で判明 |
| EX-AUTHORIZATION-008-01 | `usecases/check_access_test.go` `TestDeletingAnObjectStopsTheRelationsItCarried` | 観測追加 | 何も消さなかった削除が削除イベントを残さないこと。変異で判明 |
| EX-AUTHORIZATION-009-01 | `usecases/check_access_test.go` `TestCheckAccessAuditKeepsNoIdentifiers` | 観測追加 | 監査イベントがリソース型、関係、許否、拒否理由を持つこと、および主体識別子とタプルの内容がどこにも複製されないこと |
| EX-AUTHORIZATION-010-01 | `handlers_http/routes_test.go` `TestAuthorizationAdminRoutes` | ディレクティブ | なし |
| EX-AUTHORIZATION-010-02 | `handlers_http/routes_test.go` `TestAuthorizationAdminRoutes` | 観測追加 | 拒否された判定の応答が判定を載せないこと。既存テストは空の本文を送って 403 だけを見ていた |

内訳は、ディレクティブだけで済むものが 13 件、観測追加が 10 件、新規が 8 件である。
親項目の測定（16 件中 4 件）より比率は高いが、理由は Authorization の既存テストが `domain` と `usecases` の双方に厚く、規則単位の観測がすでに揃っていることにある。

表の `観測追加` のうち 4 件は、読んだだけでは `ディレクティブ` に見えた。
内訳は、規則単位のテストが複数の具体例を 1 件の入力で兼ねていたもの（`EX-AUTHORIZATION-003-02` と `EX-AUTHORIZATION-003-03`）と、変異検査が生存を報告して分かったもの（`EX-AUTHORIZATION-007-02` と `EX-AUTHORIZATION-008-01`）である。
前者は、親の解決が壊れると 2 つの具体例が同時に落ちて原因を指さない。
後者は、上限ちょうどの境界と、何も消さなかった削除のイベントを、どのテストも見ていなかった。

### 消化先を選ぶ規準

同じ具体例を観測できるテストが複数あるとき、`Then` を全部観測できるもののうち最も内側の境界を選ぶ。
`handlers_http/standards_test.go` は `EX-AUTHORIZATION-004-*` と `EX-AUTHORIZATION-007-*` の入力を HTTP 経由で通しているが、それらの行が固定しているのは `standards.md` の 5 行であり、具体例の `Then`（スコープ、走査の上限、監査の粒度）とは観測点が違う。
同じ id を 2 箇所から名指すと、どちらを直せばよいかが次の読み手に伝わらないので、標準側のテストには足さない。

`EX-AUTHORIZATION-010-*` だけは `handlers_http` に置く。
この 2 件の `Then` は `AccessDeniedError` で拒否することであり、権限の判定は HTTP の入口にしか無いからである。

### 新しく書くテストの配置

新規 8 件はいずれも `usecases` の境界で `Then` を全部観測できる。
`backend/shared/http/testing_stack` は HTTP の組み立てを建てる道具なので、この 8 件には使わない。
`usecases/check_access_test.go` の既存 `harness` は 1 テナント固定なので、テナント境界の 2 件（`EX-AUTHORIZATION-006-01`、`EX-AUTHORIZATION-006-02`）のために、同じ `Store` の上へ 2 つ目のテナントを建てられるよう `harness` を広げる。

## Tasks

- [x] T001 [Spec] N/A: `spec_impact: none`。シナリオも契約も変えない。
- [x] T002 [Acceptance] 31 件を `tools/check/example-coverage-debt.json` から外し、`mise run check-spec` が 31 件を名指しで落とすことを観測する。
- [x] T003 [Domain] `domain/evaluator_test.go` の 6 件を消化する。経路を分ける表の追加 2 件と深さ上限の境界を含む。
- [x] T004 [App] `usecases/check_access_test.go` の既存テストで 15 件を消化する。うち 7 件は観測追加を伴う。
- [x] T005 [App] `usecases/check_access_test.go` へ新規 8 件を書く。
- [x] T006 [Adapters] `handlers_http/routes_test.go` の 2 件を消化する。拒否応答が判定を載せないことの観測を含む。
- [x] T007 [Verify] `mise run test-go-mutation` で変更したテストの検出能力を読み、`mise run verify` を通す。

## Verification

- `mise run check-spec` が、`docs/domain/authorization/scenarios.feature.md` の 31 件を `tools/check/example-coverage-debt.json` から外した状態で通る。台帳から外す前に同じ検査が当の id を名指しで落とすことを、消化ごとに観測する。
- 消化したテストの所属パッケージに対する `mise run test-go-package -- <package>`。
- `mise run verify`

## Risk Notes

- **ディレクティブだけを足して終わる。** `checkNormativeCoverage` はディレクティブの前半に id があるかしか見ないので、読まずに id を貼れば件数は速く減る。減った件数は何も意味しない。ディレクティブに「この具体例の何を固定しているか」を書かせることで区別する。親項目の測定では、16 件のうちディレクティブだけで済んだのは 4 件だけだった。
- **`nearby` に分類された件を、テストがある証拠として読む。** 分類が言っているのは「近傍に他の拒否テストがある」だけである。親項目の測定では、`claim-mapping` の 3 件はすべて `nearby` でありながら 2 件はテストが無かった。分類は読む順の材料にとどめる。
- **拒否である具体例のテストが、応答の字面だけを見て書かれる。** 新しく書く拒否テストには [[wi-392-refusal-tests-assert-the-absent-effect]] の規範が効く。ディレクティブへ「効果の不在を何で観測したか」を書き、後から区別できるようにする。
- **具体例の `Then` が複数あるのに、観測が 1 つで済まされる。** 親項目の測定では、`EX-WORKLOADIDENTITY-008-01` のようにイベントと実際の効果の双方を言う具体例が複数あった。`Then` の数だけ観測が要る。

## 完了

- **Completed At**: 2026-09-19
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を返す。
  シナリオも契約も変えていない。
  変わったのは、`docs/domain/authorization/scenarios.feature.md` が宣言する具体例 31 件のそれぞれを、その id を名指しして何を固定しているかを述べるテストが支えるようになったことである。
  `tools/check/example-coverage-debt.json` の `untested` は 31 件減り、Authorization の行は残っていない。
  ディレクティブだけで済んだのは 13 件で、10 件は既存テストが具体例の `Then` の一部を見ていなかったため観測を足し、8 件はテストが存在しなかったため書いた。
  実装が具体例と食い違う件は 1 件も出なかったので、切り出した欠陥はない。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`（`checkNormativeCoverage`）
  - **Requirement**: REQ-AUTHORIZATION-001
  - **Observed Failure**: 台帳から 31 件を外した時点で exit 1。`EX-AUTHORIZATION-001-01` から `EX-AUTHORIZATION-010-02` までの 31 件それぞれについて `<id> is declared, but no test names it.` を宣言元の行番号つきで出した。他の検査項目に失敗はなく、失敗は 31 件ちょうどだった。
  - **Detection Reason**: `Requirement` 欄は id を 1 つしか取らないので代表を書いた。実際の対象は `REQ-AUTHORIZATION-001` から `REQ-AUTHORIZATION-010` が宣言する具体例 31 件である。検査は id ごとに 1 件の指摘を出すので、消化前に当の id が名指しで落ちることを 31 件すべてについて個別に観測できる。`//spec:covers` ディレクティブの前半に id がある場合しか被覆と数えないため、散文に id を書くだけでは緑にならない。
- **Unit RED Evidence**:
  - **Test**: N/A: 製品コードを変更しないので、実装前に落ちる単体境界が存在しない。
  - **Requirement**: N/A: 同上。既存の `REQ-AUTHORIZATION-001` から `010` の振る舞いは変えていない。
  - **Observed Failure**: 代わりに、足した観測が実装の欠陥を捕まえることを故障注入で観測した。下の Change-Resistance Results に 3 件を記す。
  - **Detection Reason**: 新しい観測の価値は「今も通ること」ではなく「壊れたら落ちること」でしか測れない。テストだけを変える作業では RED が取れないので、実装側を壊して赤を作る向きに置き換えた。
- **Change-Resistance Results**:
  - 変異検査（`mise run test-go-mutation`）。`backend/authorization/usecases` は 38 killed / 6 lived / 2 not covered（efficacy 86.4%、行被覆 85.2%）から 46 killed / 0 lived / 0 not covered（efficacy 100%、行被覆 89.0%）になった。`backend/authorization/domain` は 57 killed / 4 lived（93.4%）から 60 killed / 1 lived（98.4%）になった。
  - 生存していた変異のうち、具体例が名指す振る舞いに載っていた 7 件を読んで観測を足した。書き込みイベントと削除イベントの発行条件（`result.WrittenCount > 0`、`result.DeletedCount > 0`）、列挙の打ち切り境界（`len(candidates) > limit`）、候補が 0 件のときの整合トークン（`out.Consistency == ""`）、探索の深さ上限の境界（`depth >= e.maxDepth`）と、computed_userset・tuple_to_userset の段数の進み方（`depth+1`）である。
  - 残る生存 1 件は `domain/tuple.go:109` の識別子長の境界（`> spec.LengthExternalID`）、未被覆 11 件は `SubjectRef.String` と `RelationTuple.String` の文字列連結である。どちらも 31 件のどの具体例も言及しない振る舞いなので、本項目では追わない。
  - 変異器が表現できない故障注入 3 件。いずれも当該テストだけが落ち、ほかは緑のままだった。
    - 配線を外す: `db_memory.RelationTupleRepository.ListSubjects` からテナントの絞り込みを外し、全テナントのタプルを読ませた。`TestCheckAccessReadsOnlyTheCallersTenant` が `the grant lives in another tenant and must not reach this decision` で落ちた。
    - 既定の分岐を差し替える: `ListAccessibleResources` が内側で呼ぶ `checkAccess` の `audit` を `false` から `true` にした。`TestListAccessibleResourcesIsBoundedAndFiltered` が `enumeration emitted 3 per-check and 1 summary events, want 0 and 1` で落ちた。
    - 効果の順序を入れ替える: `WriteRelationTuples` が `ValidateTuple` の前に 1 件ずつ保管庫へ適用するようにした。`TestWriteRelationTuplesRejectsTheWholeDiff` と `TestWriteRelationTuplesRejectsSubjectFormsTheDirectRuleDoesNotAllow` の 3 つの部分試験が、拒否された差分の適合した側が残っていることを報告して落ちた。
- **Verification Results**:
  - `mise run check-spec` - passed（`157 standard(s), 316 rule(s), 771 example(s), 724 id(s) named by a test`）
  - `mise run lint-go` - passed（0 issues）
  - `mise run test-go-package -- ./backend/authorization/domain` - passed
  - `mise run test-go-package -- ./backend/authorization/usecases` - passed
  - `mise run test-go-package -- ./backend/authorization/handlers_http` - passed
  - `mise run test-go-changed` - passed
  - `mise run verify` - passed
  - `mise run test-ui-e2e` - N/A: Go のテストと被覆台帳だけを変えており、ブラウザーに届く経路を持たない。
