---
depends_on: [wi-625-application-assignment-accepts-any-subject]
status: completed
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-09-19
priority: p1
change_kind: feature
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: LifecycleWorkflow の assign_application と unassign_application が、実際に変えたときだけ ApplicationAssigned と ApplicationUnassigned を監査へ記録し、Provisioning へ通知するようになる。指定と異なる visibility の直接割り当ては指定どおりに更新される。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-628-implement-desired-state-application-assignment.md }
initial_context:
  specification:
    - docs/domain/application/scenarios.feature.md#REQ-APPLICATION-014
    - docs/domain/application/internals.md
    - docs/domain/application/decisions.md
  typespec: []
  source:
    - backend/application/usecases/assignments.go
    - backend/application/ports/repository.go
    - backend/application/module.go
    - backend/idgovernance/usecases/lifecycle_workflow_dispatcher.go
    - backend/idgovernance/domain/lifecycle_workflows.go
    - backend/cmd/idmagic-worker/worker.go
    - backend/shared/http/support_http/application_gate.go
  tests:
    - backend/idgovernance/usecases/scenario_examples_test.go
    - backend/cmd/idmagic-worker/worker_lifecycle_workflow_test.go
    - tools/check/example-coverage-debt.json
  stop_before_reading:
    - frontend
    - backend/application/handlers_http
    - backend/application/db_postgres
affected_spec:
  - { path: docs/domain/application/scenarios.feature.md, requirement: REQ-APPLICATION-014 }
primary_use_cases:
  - id: desired-state-assignment-beside-group-assignment
    requirement: REQ-APPLICATION-014
    observable_result: グループ割り当てを持つ User へ LifecycleWorkflow の assign_application を実行すると直接割り当てが作られ、unassign_application を実行すると直接割り当てだけが消え、グループ割り当ての行は変わらず、フェデレーションの関門は許可を返し続ける。
    unit_test: { path: backend/application/usecases/desired_state_assignments_test.go, name: TestDesiredStateAssignmentsLeaveTheGroupAssignmentUntouched, task: test-go-race }
    e2e_test: { path: backend/cmd/idmagic-worker/worker_lifecycle_workflow_test.go, name: TestWorkerLifecycleWorkflowAppliesTheDirectAssignmentBesideAGroupAssignment, task: test-go-race }
    unit_fault_model: 直接割り当ての解除が主体を問わず Application の割り当てを消してグループ割り当ての行を失うか、作成が subject_type を取り違える。
    e2e_fault_model: worker の実行ハンドラーが Application の割り当て操作を配線せず、ステップが保存先へ直接書くか何も書かない。
  - id: desired-state-assignment-without-difference
    requirement: REQ-APPLICATION-014
    observable_result: 指定どおりの visibility の直接割り当てがすでにある User への assign_application は changed=false を返し、保存も ApplicationAssigned の発行もしない。同じ入力を 2 回適用しても ApplicationAssigned は 1 回だけ発行される。
    unit_test: { path: backend/application/usecases/desired_state_assignments_test.go, name: TestAssignApplicationDesiredStateWithTheSameVisibilityChangesNothing, task: test-go-race }
    e2e_test: { path: backend/cmd/idmagic-worker/worker_lifecycle_workflow_test.go, name: TestWorkerLifecycleWorkflowAppliesTheDirectAssignmentBesideAGroupAssignment, task: test-go-race }
    unit_fault_model: 差分判定が visibility を比べず、または比べた結果を無視して毎回保存し ApplicationAssigned を発行する。
    e2e_fault_model: 実行ハンドラーが割り当ての変更を Application の操作へ渡さず保存先へ直接書き、ApplicationAssigned が監査へ届かない。
---

# あるべき状態を指定する Application 割り当て操作を実装する

## Motivation

`EX-APPLICATION-014-01` と `EX-APPLICATION-014-02` は、IdManagement の LifecycleWorkflow が `AssignApplicationDesiredState` と `UnassignApplicationDesiredState` を呼び、グループ経由の割り当てを変えずに個人への直接割り当てだけを作成または削除すると定める。

`docs/domain/application/decisions.md` も、この 2 つを HTTP に公開せず同じテナント内の識別子しか受け取らない内部インターフェースとして宣言している。

しかし `AssignApplicationDesiredState` も `UnassignApplicationDesiredState` も、リポジトリのどこにも実装が存在しない（[[wi-543-back-application-examples-with-tests]] の測定）。

既存の `AssignApplication` と `UnassignApplication` は、指定した `subject_type` の行をそのまま保存または削除するだけで、あるべき状態との差分も `changed` の応答も持たない。

LifecycleWorkflow が割り当てを繰り返し適用する経路では、差分が無いことを応答で区別できなければ、同じ操作のたびに `ApplicationAssigned` が発行される。

## Scope

- `AssignApplicationDesiredState` と `UnassignApplicationDesiredState` のシグネチャと戻り値を決め、Application Context の内部インターフェースとして実装する。
- 個人への直接割り当て（`subject_type=user`）だけを作成または削除し、グループ割り当て（`subject_type=group`）の行を変えないことを確かめる。
- 指定どおりの `visibility` で直接割り当てが既に存在する場合、保存もイベント発行もせず `changed=false` を返すことを確かめる。
- 直接割り当てを削除した後もグループ割り当てによるフェデレーションが許可されることを確かめる。
- IdManagement の LifecycleWorkflow からこの操作へ到達する配線を決める。

## Out of Scope

- この 2 つの操作を HTTP へ公開すること。決定が内部インターフェースと定めている。
- 既存の `AssignApplication` と `UnassignApplication` の署名変更。
- 主体の実在検査。[[wi-625-application-assignment-accepts-any-subject]] が先に決着させる。
- グループ割り当てそのものの評価規則と、動的グループの所属判定。

## Design

主たるドメイン型は `domain.ApplicationAssignment` であり、`SubjectType`、`SubjectID`、`Visibility` が同一性を決める。

あるべき状態の操作は、`AssignmentRepo` から現在の直接割り当てを読み、指定した `visibility` と突き合わせ、差分があるときだけ保存する。

時刻は入力の `Now` として受け取り、イベント発行は既存の `Emit` port を通す。

判断点は戻り値の形である。

`changed bool` だけを返す案は呼び出し側が単純になるが、何が変わったかを記録できない。

割り当てと `changed` の組を返す案は監査に足りるが、差分が無いときに返す割り当ての意味を決める必要がある。

もう一つの判断点は、LifecycleWorkflow がこの操作をどの port で呼ぶかである。

IdGovernance が Application の usecases を直接呼ぶと Context 間の依存が増えるため、port を宣言して注入する。

### 決定

`changed bool` だけを返す。
操作が成功した後の直接割り当ては、`changed` の値によらず入力どおりであり、割り当てを返しても呼び出し側が持たない情報は増えない。
何が変わったかは、変わったときだけ発行する `ApplicationAssigned` と `ApplicationUnassigned` が監査へ記録する。

| 型または操作 | シグネチャ | 置き場所 |
| --- | --- | --- |
| `DesiredStateAssignments` | `struct { Deps AssignmentDeps; ActorUserID string }` | `backend/application/usecases` |
| `AssignApplicationDesiredState` | `(ctx, tenantID, applicationID, userID string, visibility domain.AssignmentVisibility, now time.Time) (bool, error)` | `DesiredStateAssignments` のメソッド |
| `UnassignApplicationDesiredState` | `(ctx, tenantID, applicationID, userID string, now time.Time) (bool, error)` | `DesiredStateAssignments` のメソッド |
| `ApplicationAssignments` | 上の 2 つのメソッドを持つ interface | `backend/idgovernance/ports` |
| `Module.DesiredStateAssignments` | `(users, groups, emit, actorUserID) usecases.DesiredStateAssignments` | `backend/application`。主体の実在検査を組み立てる |

テナントは `ctx` ではなく引数で受け取る。
worker の実行ハンドラーは `ctx` にテナントを持たず、`WorkflowRun.TenantID` を持つためである。
`tenantID` の外の Application は `ErrApplicationNotFound`、外の User は `ErrSubjectNotFound` で拒否する。

| 操作 | 現在の直接割り当て | 作用 | 戻り値 |
| --- | --- | --- | --- |
| Assign | なし | 保存、`ApplicationAssigned`、Provisioning へ `assignment_added` | `true` |
| Assign | visibility が異なる | `CreatedAt` を保って visibility を更新して保存、`ApplicationAssigned` | `true` |
| Assign | visibility が同じ | なし | `false` |
| Unassign | あり | 削除、`ApplicationUnassigned`、Provisioning へ `assignment_removed` | `true` |
| Unassign | なし | なし（Application の存在も問わない） | `false` |

現在の直接割り当ては `AssignmentRepo.ListBySubjects` に `subject_type=user` の主体だけを渡して読む。
グループ割り当ての行は読みも書きもしない。
visibility だけの変更は Provisioning の対象属性を変えないため通知しない。

LifecycleWorkflow の実行ハンドラーは、assign_application と unassign_application を `LifecycleWorkflowExecutorDeps.ApplicationAssignments` へ渡し、`changed` を `changed` と `no_op` のステップ結果へ写す。
保存先への直接の書き込みはなくす。
事前の評価（dry-run と共有する `EvaluateLifecycleAction`）は、assign_application について visibility まで一致する直接割り当てを `no_op` とする。
これまでは visibility を比べず、異なる visibility の直接割り当てを `no_op` として変えずに残していた。
worker は actor を `lifecycle-workflow` として組み立てる。

## Plan

1. 2 つの操作のシグネチャ、戻り値、イベント発行の条件を決める。
2. グループ割り当てを持つ主体へ直接割り当てを作る経路を RED にする。
3. 差分が無い呼び出しが `changed=false` を返し、保存もイベントも起こさないことを確かめる。
4. 直接割り当ての削除後もグループ割り当てが残り、フェデレーションが許可されることを確かめる。
5. LifecycleWorkflow からの配線を決めて通す。

## Tasks

- [x] T001 [Decision] 2 つの操作のシグネチャ、戻り値、イベント発行の条件を決める。
- [x] T002 [Domain] あるべき状態との差分判定を、保存先を読み直せる形で実装する。REQ-APPLICATION-014。
- [x] T003 [Unit] グループ割り当ての行が変わらないことと、`changed=false` の経路で保存もイベントも起きないことを確かめる。
- [x] T004 [App] 直接割り当ての削除後もフェデレーションが許可されることを確かめる。
- [x] T005 [Adapters] LifecycleWorkflow からこの操作へ到達する port を配線する。
- [x] T006 [Verify] 仕様と被覆の検査を通し、2 件を台帳から外す。

## Verification

- `mise run test-go-package -- ./backend/application/usecases`
- `mise run test-go-package -- ./backend/idgovernance/usecases`
- `mise run test-go-package -- ./backend/cmd/idmagic-worker`
- RED、GREEN、障害注入は `mise run test-go-package -- <package>` で回し、境界をまたいだ後は `mise run test-go-changed` を使う。
- `mise run test-go-mutation -- backend/application/usecases`
- `mise run check-spec`
- `mise run verify`

## Risk Notes

差分判定を誤ると、グループ割り当ての行を直接割り当てとして上書きするか、削除してしまう。

操作の前後でグループ割り当ての行を読み直し、`subject_type` と `visibility` が変わらないことを観測する。

`changed=false` を応答だけで確かめると、保存もイベント発行も進む誤実装を通す。

保存ポートの呼び出し回数とイベントの発行数の双方を観測する。

冪等な呼び出しが毎回イベントを発行すると、監査ログが実際の変化を示さなくなる。

同じ入力を 2 回適用したときのイベント数を固定する。

## Completion

- **Completed At**: 2026-09-24
- **Summary**:
  `mise run spec-diff` は main に対する規範的な仕様変更なしと報告した。
  `REQ-APPLICATION-014` の宣言済みの具体例 2 件を、仕様を変えずに実装で満たした。
  Application Context に内部インターフェース `DesiredStateAssignments` を加え、`AssignApplicationDesiredState` と `UnassignApplicationDesiredState` が User への直接割り当てだけをあるべき状態へそろえ、実際に変えたかを `bool` で返すようにした。差分がなければ保存、イベント発行、Provisioning への通知のいずれも行わない。
  IdGovernance に port `ApplicationAssignments` を宣言し、LifecycleWorkflow の実行ハンドラーは assign_application と unassign_application をこの port へ渡すようにした。保存先への直接の書き込みはなくなり、ワークフローによる割り当ての変更が `ApplicationAssigned` と `ApplicationUnassigned` として監査へ届き、Provisioning へ通知される。
  事前の評価は、assign_application について visibility まで一致する直接割り当てだけを `no_op` とするようにした。
  worker は Application の Module から操作を組み立て、actor を `lifecycle-workflow` とする。
  `tools/check/example-coverage-debt.json` から `EX-APPLICATION-014-01` と `EX-APPLICATION-014-02` を削除し、リリースノートを追加した。
- **Primary Use Case Evidence**:
  - id: desired-state-assignment-beside-group-assignment
    unit_red: "TestDesiredStateAssignmentsLeaveTheGroupAssignmentUntouched は、実装前は `undefined: appusecases.DesiredStateAssignments` でビルドに失敗し、何もしない仮実装では `AssignApplicationDesiredState() = false, <nil>; want true, nil` で失敗した。"
    e2e_red: TestWorkerLifecycleWorkflowAppliesTheDirectAssignmentBesideAGroupAssignment は、実行ハンドラーが保存先へ直接書く変更前の配線で `ApplicationAssigned emitted 0 times, want 1 for two identical runs` により失敗した。
    unit_fault_injection: 解除を `DeleteByApplication` に替えると `group assignment after unassign = <nil>` で、作成時の subject_type を group に替えると `direct assignment = <nil>` で単体テストが失敗した。
    e2e_fault_injection: worker の `ApplicationAssignments` の配線を外すと、`assign step = failed, want changed` で E2E テストが失敗した。
  - id: desired-state-assignment-without-difference
    unit_red: TestAssignApplicationDesiredStateWithTheSameVisibilityChangesNothing は、実装前はビルドに失敗し、仮実装では `first AssignApplicationDesiredState() = false, <nil>; want true, nil` で失敗した。
    e2e_red: 同じ E2E テストが、変更前の直接書き込みで `ApplicationAssigned emitted 0 times` により失敗した。
    unit_fault_injection: visibility の比較を `if false` に替えると、`second AssignApplicationDesiredState() = true, <nil>; want false, nil` で単体テストが失敗した。
    e2e_fault_injection: 実行ハンドラーが Application の操作を通さず保存先へ直接書く変更前の実装で、E2E テストが `ApplicationAssigned` の件数により失敗した。
- **Change-Resistance Results**:
  - `mise run test-go-mutation -- backend/application/usecases` は 184 個の変異を試し、169 個を検出し、15 個が生き残った。`desired_state_assignments.go` の 12 個はすべて検出した。生き残りは `applications.go`、`assignments.go`、`categories.go`、`sign_in_policy.go` の既存コードにあり、この変更の対象外である。
  - `mise run test-go-mutation -- backend/idgovernance/usecases` は 155 個の変異を試し、144 個を検出した。変更した評価条件の `a.ApplicationID == app.ID` を `!=` にする変異が生き残ったため、別の Application への hidden の直接割り当てを前提に加えた。追加後、同じ変異を手で入れると TestLifecycleWorkflowRunHandlerAssignApplicationUpdatesADifferentVisibility が `EvaluateLifecycleAction() = no_op` で失敗した。unassign_application で port が未配線の分岐は被覆されていない。
  - 事前の評価から visibility の比較を外すと、TestLifecycleWorkflowRunHandlerAssignApplicationUpdatesADifferentVisibility が `EvaluateLifecycleAction() = no_op, <nil>; want would_change` で失敗した。
  - Application の操作から visibility の比較を外す障害は、E2E テストでは検出されなかった。実行ハンドラーの事前の評価が同じ直接割り当てを `no_op` と判定し、操作を呼ばないためである。冪等性は事前の評価と操作の 2 か所で守られており、片方だけの故障は E2E では観測できない。操作側の故障は単体テストが検出する。
- **Verification Results**:
  - `mise run test-go-package -- ./backend/application/usecases` - passed
  - `mise run test-go-package -- ./backend/idgovernance/usecases` - passed
  - `mise run test-go-package -- ./backend/cmd/idmagic-worker` - passed
  - `mise run test-go-changed` - passed
  - `mise run check-spec` - passed
  - `mise run lint-go` - 0 issues
  - `mise run verify` - passed
  - `mise run test-ui-e2e` - 未実行。変更は worker の実行経路に閉じており、ブラウザーへ届かない。
- **Left Undone**:
  - `REQ-APPLICATION-014` と `docs/domain/application/internals.md` は LifecycleWorkflow を IdManagement のものと書いているが、実装は IdGovernance が所有する。仕様の文言は変えていない。
