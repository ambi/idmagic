---
depends_on: [wi-565-make-backing-declared-examples-cheap]
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-12
priority: p3
change_kind: maintenance
spec_impact: { kind: none, reason: "宣言済みの具体例に、その id を名指しするテストを対応付ける作業である。シナリオは変えない。本項目で直す欠陥は、実装を宣言済みの具体例、TypeSpec の操作説明、decisions.md、states.md へ合わせるものであり、規範を変えない。" }
evidence_policy: risk-based-v3
documentation_impact:
  level: none
  reason: 宣言済みの具体例と既存の検証を対応付け、実装を既存の規範へ合わせる保守作業であり、公開契約と運用手順を変えない。
  references: []
initial_context:
  specification:
    - docs/domain/identity-governance/scenarios.feature.md
    - docs/domain/identity-governance/decisions.md
    - docs/domain/identity-governance/states.md
    - docs/domain/identity-governance/internals.md
  typespec:
    - spec/contexts/identity-governance/main.tsp
  source:
    - backend/idgovernance/domain/lifecycle_workflows.go
    - backend/idgovernance/usecases/lifecycle_workflows.go
    - backend/idgovernance/usecases/lifecycle_workflow_dispatcher.go
    - backend/idgovernance/usecases/lifecycle_workflow_dry_run.go
    - backend/idgovernance/usecases/user_mutation_committer.go
    - backend/idgovernance/db_memory/lifecycle_workflow_runs.go
    - backend/idgovernance/db_memory/lifecycle_workflow_capture.go
    - backend/idgovernance/db_postgres/lifecycle_workflow_capture.go
    - backend/idgovernance/db_postgres/lifecycle_workflow_runs.go
    - backend/idgovernance/handlers_http/admin_lifecycle_workflow_handler.go
    - backend/idgovernance/handlers_http/routes.go
    - backend/shared/http/server_http/routes.go
    - backend/cmd/idmagic-worker/worker.go
    - backend/jobs/usecases/runner.go
    - backend/idmanagement/user/usecases/admin_users.go
    - backend/shared/http/testing_stack/stack.go
    - frontend/src/features/admin-lifecycle-workflows/WorkflowDefinitionForm.tsx
    - frontend/src/features/admin-lifecycle-workflows/AdminLifecycleWorkflowEditorPage.tsx
  tests:
    - backend/idgovernance/domain/lifecycle_workflows_test.go
    - backend/idgovernance/usecases/lifecycle_workflows_test.go
    - backend/idgovernance/usecases/lifecycle_workflow_dispatcher_test.go
    - backend/idgovernance/handlers_http/admin_lifecycle_workflow_handler_test.go
    - frontend/src/features/admin-lifecycle-workflows/WorkflowDefinitionForm.test.tsx
    - frontend/src/features/admin-lifecycle-workflows/AdminLifecycleWorkflowPages.test.tsx
    - tools/check/example-coverage-debt.json
  stop_before_reading:
    - infra
    - backend/idgovernance/db_postgres/lifecycle_workflows.sql.go
---

# IdGovernance が宣言する具体例 21 件にテストを対応付け、被覆台帳から外す

## Motivation

[[wi-496-burn-down-the-example-coverage-debt]] は具体例の被覆台帳の消化単位を Context と決め、測定のうえで残りを Context ごとの子 work item へ割った。本項目はそのうち `docs/domain/identity-governance/scenarios.feature.md` が宣言する 21 件を引き取る。

親項目が `claim-mapping` の 3 件と `workloadidentity` の 13 件で測った結果は、**16 件のうち注記だけで済んだのは 4 件だけ**だというものである。3 件はテストが 1 つも無く、9 件は既存テストへ新しい観測を足す必要があった。件数は作業量の目安にならない。

## Scope

- 21 件を 1 件ずつ確認し、次のいずれかに解決して `tools/check/example-coverage-debt.json` から外す。
  - 当の具体例を検証しているテストが実在する → そのテストに `//spec:covers EX-IDGOVERNANCE-NNN-MM: <この具体例の何を固定しているか>` を足す。
  - 当の具体例を検証しているテストが無い → 書く。具体例が拒否なら、[[wi-392-refusal-tests-assert-the-absent-effect]] が定める形（拒否応答と、拒否が防いだ効果の双方を観測する）で書く。
  - 具体例が宣言されなくなっている → 台帳から外す（検査が落ちて教える）。
- 注記は「何を固定しているか」を書く。id だけの注記は禁止する。
- 実装にバグがあって具体例のとおりに振る舞っていないことが分かった場合は、難しくない修正なら本 work item で修正する。難しい修正なら別 work-item に切り出す（「食い違いの判定」節）。

## Out of Scope

- 他の Context が宣言する具体例。Context ごとに別の work item が持つ。
- `tools/check/standards-coverage-debt.json`。[[wi-495-burn-down-the-standards-coverage-debt]] が消化済みである。
- 既に台帳に載っていない拒否テストが、防いだ効果を観測していない件。[[wi-392-refusal-tests-assert-the-absent-effect]] が持つ。
- シナリオと具体例の追加、削除、書き換え。
- `user_attributes_changed` の監視属性（`watched_attributes`）を属性スキーマと照合すること。`EX-IDGOVERNANCE-007-01` が言うのはフィルターのフィールドであり、監視属性は組み込みの中核フィールド（`name` など）の変更名とも一致させる必要があって、照合の規則が別に要る。
- `internals.md` が述べる Transactional Outbox 経由のトリガー消費。現在の実装は User の保存と同じトランザクションで WorkflowRun を直接捕捉しており、具体例が要求する「同一トランザクションで確定」と「再配信の収束」は満たす。
- 行カバレッジ率の目標または閾値。
- `report-coverage-debt` の分類を根拠にした台帳からの削除。

## 作業の進め方

[[wi-565-make-backing-declared-examples-cheap]] が用意した道具を使う。

- 読む順と観測点は `mise run spec-route -- <id>` が出す。**出力は読む順を決める材料であり、台帳から外す根拠にはしない。**
- 変更した Go の変異は `mise run test-go-mutation -- <package-directory>` で読む。手で書く故障注入は、変異器が表現できない「配線を外す」「既定の分岐を差し替える」種類だけに残す。
- 実装が具体例と食い違って消化できない件は、台帳の当該行へ `blocked_by` と `finding` を書いて残す。

## Design

### 食い違いの判定

具体例と実装の食い違いは、具体例とは別の規範（TypeSpec の操作説明、`decisions.md`、`states.md`）とも照合して、実装の欠陥か具体例の欠陥かを判定した。
すべて、具体例以外の規範も実装と食い違っており、実装の欠陥と判定した。

| 食い違い | 具体例 | 具体例以外の根拠 | 判定 |
| --- | --- | --- | --- |
| 入力の検証失敗を `invalid_workflow`、存在しないワークフローを 404 `workflow_not_found` で返す | `EX-IDGOVERNANCE-006-01`、`-007-01`、`-011-02`、`-012-01` | TypeSpec の全操作が 400 として InvalidRequestError（`urn:idmagic:error:invalid_request`）だけを宣言し、404 を宣言しない。InvalidRequestError の説明は「別テナントの対象を存在しないものとして扱う」場合を含む | 実装の欠陥 |
| `add_group_member` に動的グループを指定しても保存できる | `EX-IDGOVERNANCE-006-01` | 作成画面は動的グループを選択肢から除いている。保存時に拒否する側が欠けている | 実装の欠陥 |
| 属性スキーマに無いフィルターのフィールド、別テナントや存在しないグループとアプリケーションを保存と有効化で拒否しない | `EX-IDGOVERNANCE-007-01`、`-012-01` | `decisions.md`「定義時に別テナントの識別子を指定すれば保存で拒否し」。TypeSpec の EnableLifecycleWorkflow「`current_revision` を検証し、失敗すれば拒否する」 | 実装の欠陥 |
| 無効化しても `running` の WorkflowRun が次のステップを始める | `EX-IDGOVERNANCE-011-01`、`-011-02` | `states.md`「`running` の実行は現在のステップのチェックポイント後、次のステップを始める前に `canceled` になる」。TypeSpec の DisableLifecycleWorkflow と RetryLifecycleWorkflowRun の説明も同じ | 実装の欠陥 |
| 失敗したステップがあると、その試行で WorkflowRun を終端にし、Jobs に再試行させない | `EX-IDGOVERNANCE-009-01` | `states.md`「Job の試行上限に達した時点で……終了する」「Jobs 側で再試行している間は `running` のままにする」 | 実装の欠陥 |

いずれも修正は小さく、本項目で直す。

### 修正の設計

#### エラーの対応付け

`writeLifecycleWorkflowError` の既定の分岐と、ワークフローが見つからない分岐を、400 の `invalid_request` へ揃える。
名前の重複と revision の競合は TypeSpec が 409 として宣言しているので変えない。

#### 定義の参照先の検証

保存（作成と更新）と有効化の直前に、revision の参照先をワークフローと同じテナントで検証する。

```go
// LifecycleWorkflowDeps に加える。
GroupRepo       groupports.GroupRepository
ApplicationRepo appports.ApplicationRepository
AttrSchemaRepo  tenantports.TenantUserAttributeSchemaRepository

func validateRevisionReferences(ctx context.Context, deps LifecycleWorkflowDeps, revision *igdomain.LifecycleWorkflowRevision) error
```

- フィルターのフィールドは、中核フィールド（`preferred_username`、`email`、`status`）、組み込み属性、テナントの属性スキーマのいずれかでなければならない。`AttrSchemaRepo` が nil なら組み込みだけで判定する（`IdManagement` の実効定義と同じ扱い）。
- グループのアクションは、同じテナントに存在し、手動管理のグループでなければならない。
- アプリケーションのアクションは、同じテナントに存在しなければならない。
- 検証に要る保存先が nil のときは拒否する（fail-closed）。

違反は `ErrLifecycleWorkflowInvalidReference` として返し、HTTP では InvalidRequestError になる。

#### ステップ境界での取り消し

`LifecycleWorkflowExecutorDeps` に `WorkflowRepo igports.LifecycleWorkflowRepository` を加える。
ハンドラーは各ステップを始める前にワークフローを同じテナントで読み直し、`enabled` でなければ残りのステップを `canceled` として記録し、WorkflowRun を `canceled` で終えて `LifecycleWorkflowRunCanceled` を発行する。
`queued` から始める前にも同じ判定をする。
`WorkflowRepo` が nil の構成では判定しない。製品の組み立て（`backend/cmd/idmagic-worker/worker.go`）は必ず渡す。組み立てを `lifecycleWorkflowExecutorDeps` へ切り出し、その関数を通したハンドラーが無効化済みワークフローの WorkflowRun を打ち切ることを worker のテストで観測する。

#### 試行上限までの再試行

1 回の試行で失敗したステップがあり、`job.Attempts < job.MaxAttempts` なら、WorkflowRun を `running` のまま残して Jobs へエラーを返す。
Jobs はバックオフ後に同じ Job を再試行し、ハンドラーは `changed` と `no_op` のステップを飛ばして `failed` のステップだけを実行する。
試行上限に達した試行では、従来どおり結果の組み合わせで終端状態を決める。

| 採った案 | 退けた案 | 退けた理由 |
| --- | --- | --- |
| 失敗の種類を区別せず、試行上限まで再試行する | 一時的な失敗（保存先のエラー）だけを再試行し、`blocked` 由来の失敗は即座に終端にする | `states.md` は失敗の種類で分けていない。区別は規範の変更になる |

### テストの置き場所

| 具体例 | 置き場所 |
| --- | --- |
| `EX-IDGOVERNANCE-001-01`、`-02`、`-03`、`-002-01` | 画面の操作は `AdminLifecycleWorkflowPages.test.tsx`。API が返す定義と拒否は `backend/idgovernance/handlers_http` |
| `EX-IDGOVERNANCE-003-01`、`-004-01`、`-004-02`、`-005-01`、`-006-01`、`-008-01`、`-009-01`、`-010-01`、`-011-01`、`-013-01`、`-02`、`-03` | `backend/idgovernance/usecases`。User の変更は `IdManagement` のユースケースを通し、実行は jobs の Runner またはハンドラーを通す |
| `EX-IDGOVERNANCE-004-01` の同一トランザクション | `backend/idgovernance/db_postgres`。埋め込み PostgreSQL で、捕捉の途中の失敗が User の変更も巻き戻すことを観測する |
| `EX-IDGOVERNANCE-007-01`、`-011-02`、`-012-01`、`-014-01`、`-02` | `backend/idgovernance/handlers_http`。テナント "acme" を持つ fixture に広げる |

### 選んだ実行レシピ

| 触った層 | 実行するタスク |
| --- | --- |
| Go の 1 テスト | `mise run test-go-test -- <package> <test>` |
| Go のパッケージ | `mise run test-go-package -- <package>` |
| 埋め込み PostgreSQL を使うテスト | 同じタスクをサンドボックスの外で実行する |
| フロントエンドの 1 ファイル | `mise run test-ui-unit-file -- <file>` |
| 台帳と frontmatter | `mise run check-spec`、`mise run check-work-items` |

## Tasks

- [x] T001 [Acceptance] 21 件を台帳から外し、`mise run check-spec` が未対応 id を名指しで落とすことを確認する。
- [x] T002 [Fix] エラーの対応付けを InvalidRequestError へ揃える。
- [x] T003 [Fix] 保存と有効化で参照先を検証する（`EX-IDGOVERNANCE-006-01`、`-007-01`、`-012-01`）。
- [x] T004 [Fix] ステップ境界で取り消す（`EX-IDGOVERNANCE-011-01`、`-011-02`）。
- [x] T005 [Fix] 試行上限まで再試行する（`EX-IDGOVERNANCE-009-01`）。
- [x] T006 [Run] `EX-IDGOVERNANCE-003-01`、`-004-01`、`-004-02`、`-005-01`、`-008-01`、`-010-01` の観測を書く。
- [x] T007 [DryRun] `EX-IDGOVERNANCE-013-01`、`-02`、`-03` の観測を書く。
- [x] T008 [AdminAPI] `EX-IDGOVERNANCE-012-01`、`-014-01`、`-02` の観測を書く。
- [x] T009 [UI] `EX-IDGOVERNANCE-001-01`、`-02`、`-03`、`-002-01` の観測を書く。
- [x] T010 [Verify] 対象パッケージ、仕様検査、総合検証を通して完了記録を作る。

## Verification

- `mise run check-spec` が、`docs/domain/identity-governance/scenarios.feature.md` の 21 件を `tools/check/example-coverage-debt.json` から外した状態で通る。台帳から外す前に同じ検査が当の id を名指しで落とすことを観測する。
- 予定する Acceptance RED は、台帳から 21 件を外した `mise run check-spec` と、HTTP で動的グループ、未知のフィールド、別テナントのグループを指定した保存と有効化が InvalidRequestError で拒否されることの観測である。
- 予定する Unit RED は、無効化後にハンドラーが次のステップを始めないこと、失敗したステップのある試行が WorkflowRun を `running` のまま Jobs へエラーを返すこと、の `backend/idgovernance/usecases` の観測である。
- 消化したテストの所属パッケージに対する `mise run test-go-package -- <package>`。
- `mise run verify`

## Risk Notes

- **注記だけを足して終わる。** `checkNormativeCoverage` は文字列の一致しか見ないので、読まずに id を貼れば件数は速く減る。注記に「この具体例の何を固定しているか」を書かせることで区別する。
- **`nearby` に分類された件を、テストがある証拠として読む。** 分類は読む順の材料にとどめる。
- **拒否である具体例のテストが、応答の字面だけを見て書かれる。** 新しく書く拒否テストには [[wi-392-refusal-tests-assert-the-absent-effect]] の規範が効く。
- **具体例の `Then` が複数あるのに、観測が 1 つで済まされる。** `Then` の数だけ観測が要る。
- **参照先の検証を nil の保存先で素通りさせる。** 検証に要る保存先が配線されていない構成で保存が通ると、製品の組み立て漏れが見えなくなる。nil は拒否とし、HTTP の組み立てへ保存先を渡す。

## Completion

- **Completed At**: 2026-09-22
- **Summary**:
  `mise run spec-diff` は `main` に対して `no normative specification change` を返した。
  IdGovernance が台帳に残していた 21 件の具体例をすべて実装と照合し、`//spec:covers` を付けたテストへ結び付けて台帳から外した。
  既存テストへディレクティブを足すだけで済んだ件は無かった。
  `EX-IDGOVERNANCE-005-01`、`-013-02`、`-013-03` は既存テストに観測（重ならないメンバーシップ、下書きのトリガー、全アクションの blocked）を足した。
  残りの 18 件は新しいテストを書いた。画面の 4 件は作成画面と編集画面の操作、実行系は IdManagement のユースケースと jobs の Runner を通した観測、管理 API はテナント "acme" を含む `Register` の組み立て、同一トランザクションは埋め込み PostgreSQL である。
  照合で 5 件の食い違いが分かった。具体例以外の規範（TypeSpec の操作説明、`decisions.md`、`states.md`）とも照合し、すべて実装の欠陥と判定して、利用者の指示により本項目で直した。
  管理 API は入力の検証失敗を `invalid_workflow`、存在しないワークフローを 404 で返しており、契約が宣言する InvalidRequestError と食い違っていた。
  保存と有効化は、動的グループ、属性スキーマに無いフィルターのフィールド、別テナントや存在しないグループとアプリケーションを拒否していなかった。
  実行ハンドラーは、無効化されたワークフローの running の WorkflowRun を次のステップへ進めていた。
  失敗したステップのある試行は WorkflowRun をその場で終端にし、Jobs に再試行させていなかった。
- **Acceptance RED Evidence**:
  - **Test**: `mise run check-spec`、`TestCreateRefusesAnActionWithoutAnExistingReference`、`TestSavingAnAddGroupMemberActionOnADynamicGroupIsRefused`、`TestUnknownFieldsAndForeignGroupsCannotBeSavedOrEnabled`、`TestRetryingARunOfADisabledWorkflowStartsNothing`、`TestWorkflowsAndResourcesDoNotCrossTheTenantBoundary`（`backend/idgovernance/handlers_http`）
  - **Requirement**: REQ-IDGOVERNANCE-007
  - **Observed Failure**: 台帳から 21 件を外すと、`check-spec` は `EX-IDGOVERNANCE-001-01` から `EX-IDGOVERNANCE-014-02` までの 21 件を「declared, but no test names it」で名指しして落ちた。修正前の管理 API は、存在しないグループと動的グループを参照する定義を 201 と 200 で保存し、未知のフィルターのフィールドを 200 で保存し、無効化済みワークフローの再試行を `urn:idmagic:error:invalid_workflow` で、別テナントのワークフローの取得を 404 `workflow_not_found` で返して落ちた。
  - **Detection Reason**: 台帳は REQ-IDGOVERNANCE-001 から REQ-IDGOVERNANCE-014 の具体例 21 件を含み、台帳の行を消すだけでは検査を通らない。HTTP のテストは製品と同じ `Register` の組み立てで要求を送り、応答の type に加えて保存先の定義と WorkflowRun と Job を読み直すので、拒否を書いてから保存する実装とも区別できる。
- **Unit RED Evidence**:
  - **Test**: `TestDisablingStopsARunningRunAtTheNextStepBoundary`、`TestARetriedRunOfADisabledWorkflowStartsNoStep`、`TestFailedStepKeepsTheRunRunningWhileAttemptsRemain`、`TestTransientTimeoutIsRetriedWithoutRepeatingCompletedSteps`（`backend/idgovernance/usecases`）
  - **Requirement**: REQ-IDGOVERNANCE-011
  - **Observed Failure**: 修正を外した状態では、1 つ目のステップの実行中に無効化された WorkflowRun のステップが `[changed changed]`（期待は `[changed canceled]`）になり、無効化の後に再試行された WorkflowRun がグループへメンバーを追加して落ちた。失敗したステップのある非最終の試行はエラーを返さず、Runner の Job は `attempts=1` で succeeded になって落ちた。
  - **Detection Reason**: 4 件は REQ-IDGOVERNANCE-011 と REQ-IDGOVERNANCE-009 に属し、実在のハンドラーと jobs の Runner を通して、ステップの結果、グループのメンバー、Job の試行回数を保存先から読むので、ハンドラーの戻り値だけを見るテストでは区別できない継続と中断の違いを検出する。
- **Change-Resistance Results**:
  `mise run test-go-mutation -- backend/idgovernance/usecases` は 152 件の変異のうち 141 件を検出した。最初の実行で、今回足した参照先の検証に 3 件の生存変異（フィルターの有無の判定の否定、グループとアプリケーションの検索エラーの分岐の否定）があり、ユースケース層で検証をたたく `TestSavingValidatesReferencesInTheWorkflowsTenant` を足して検出させた。今回変えた行に残る生存変異は、フィルターの件数の `> 0` を `>= 0` にする等価な変異 1 件だけである。他の 10 件は既存の行（Job の関連付けのエラー分岐、成功数の増減、可視性の分岐、名前の重複判定、取り消しイベントの発行エラー、再試行の検索エラー、捕捉の分岐）にある。
  `mise run test-go-mutation -- backend/idgovernance/handlers_http` は 38 件のうち 37 件を検出した。生存した 1 件は既存の応答組み立てのエラー分岐にある。
  変異器が表現できない故障として次を手で注入し、すべて検出された。worker の組み立てから `WorkflowRepo` を外す形は、`TestWorkerLifecycleWorkflowHandlerCancelsRunsOfADisabledWorkflow` が WorkflowRun の `succeeded` で検出した。管理 API の組み立てから属性スキーマの配線を外す形は、`TestUnknownFieldsAndForeignGroupsCannotBeSavedOrEnabled` がスキーマ独自の属性を使う作成の拒否で検出した。作成画面から動的グループの除外を外す形は、`選べる参照先が無いと先に作るよう案内し、作成させない` が検出した。
- **Verification Results**:
  - `mise run check-spec` - 成功
  - `mise run test-go-package -- ./backend/idgovernance/...` - 成功
  - `mise run test-go-package -- ./backend/idgovernance/db_postgres` - 成功（サンドボックスの外で実行）
  - `mise run test-go-test -- ./backend/cmd/idmagic-worker TestWorkerLifecycleWorkflowHandlerCancelsRunsOfADisabledWorkflow` - 成功
  - `mise run test-ui-unit-file -- src/features/admin-lifecycle-workflows/AdminLifecycleWorkflowPages.test.tsx` - 成功
  - `mise run test-go-changed` - 成功
  - `mise run lint-go` - 成功
  - `mise run check-contract-drift` - 成功
  - `mise run verify` - 成功
  - `mise run test-ui-e2e` - 成功（1 回目は account portal の 1 件が Bun WebKit の既知の不具合 oven-sh/bun#43412 の形で時間切れになり、再実行で 38 件すべて成功）
