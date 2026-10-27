# ワークフローの実行

## 概要

この文書は、User の変更を契機に WorkflowRun を作り、`worker` がそのステップを実行する仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | User の変更と同じトランザクションでの WorkflowRun の捕捉、Job の関連付け、ステップの実行、再試行、同じ User の実行の直列化 |
| 行為者 | テナント管理者（User の変更の契機）、System（ディスパッチャーと `worker`） |
| 扱わないもの | ワークフローの定義と無効化は[ライフサイクルワークフローの定義](lifecycle-workflow.md)が扱う |

## モデル

トリガーは、User を変更した後の状態に対して評価し、属性の変更の前後の比較だけは変更の差分を使う。
同じ発火の事象は、重複排除のキー（`tenant_id`、`workflow_id`、`revision`、`source_occurrence_id`、`target_user_id`）で一つの WorkflowRun に収束する。

1 回の試行では、未完了のステップをすべて定義順に実行し、失敗したステップがあっても止めない。
再試行では `failed` のステップだけを再実行し、`changed` または `no_op` になったステップは飛ばす。

- **判断**：失敗しても止めず補償もしない理由は、[部分的な失敗で止めず補償もしない](../decisions.md#部分的な失敗で止めず補償もしない)。

## 状態遷移

### WorkflowRunLifecycle

`WorkflowRun` は `queued` で作成し、ディスパッチャーが `job_id` を関連付けて最初のステップを開始すると `running` に遷移する。1 回の試行では、未完了のステップを定義順にすべて実行する。すべてのステップが成功すると `succeeded` で終了する。失敗したステップが残る間は Jobs の再試行に任せて `running` のままにし、Job の試行の上限に達した時点で、成功と失敗が混在していれば `partially_failed`、成功が 1 つもなければ `failed` で終了する。`no_op` は成功として扱う。`partially_failed` と `failed` の実行は、管理者の再試行で `queued` に戻る。ワークフローを無効化すると、未開始の `queued` の実行は `canceled` になり、`running` の実行は現在のステップのチェックポイント後、次のステップを始める前に `canceled` になる。遷移の表の管理者による再試行は遷移の契機の名前であり、ドメインイベントとしては発行しない。

<!-- spec:state-machine backend/idgovernance/testdata/WorkflowRunLifecycle.statemachine.json -->

| State | Kind | Meaning |
| --- | --- | --- |
| queued | initial | 作成直後または再試行の後。ディスパッチャーが `job_id` を関連付けるのを待つ |
| running | — | 未完了のステップを定義順に実行している |
| succeeded | terminal | 全ステップが成功した |
| partially_failed | — | 成功と失敗が混在した。管理者が再試行できる |
| failed | — | 成功したステップが 1 つもなかった。管理者が再試行できる |
| canceled | terminal | ワークフローの無効化により、開始前またはステップの区切りで打ち切った |

| From | Event | Guard | To | Effects |
| --- | --- | --- | --- | --- |
| queued | LifecycleWorkflowRunStarted | — | running |  |
| running | LifecycleWorkflowRunSucceeded | — | succeeded |  |
| running | LifecycleWorkflowRunPartiallyFailed | — | partially_failed |  |
| running | LifecycleWorkflowRunFailed | — | failed |  |
| queued | LifecycleWorkflowRunCanceled | — | canceled |  |
| running | LifecycleWorkflowRunCanceled | — | canceled |  |
| partially_failed | 管理者による再試行 | — | queued |  |
| failed | 管理者による再試行 | — | queued |  |

| State | 実行の開始 | 試行の終了 | ワークフローの無効化 | 管理者による再試行 |
| --- | --- | --- | --- | --- |
| queued | → running（同じ User の先行する実行が終わっている）<br>何もしない（先行する実行が終わっていない） | 何もしない | → canceled | 拒否：400 invalid_request |
| running | 何もしない | → succeeded（すべてのステップが成功）<br>何もしない（失敗したステップが残り、試行が上限未満）<br>→ partially_failed（試行が上限に達し、成功と失敗が混在）<br>→ failed（試行が上限に達し、成功が一つもない） | → canceled（次のステップの前） | 拒否：400 invalid_request |
| succeeded | 何もしない | 何もしない | 何もしない | 拒否：400 invalid_request |
| partially_failed | 何もしない | 何もしない | 何もしない | → queued（ワークフローが `enabled`）<br>拒否：400 invalid_request（ワークフローが `enabled` でない） |
| failed | 何もしない | 何もしない | 何もしない | → queued（ワークフローが `enabled`）<br>拒否：400 invalid_request（ワークフローが `enabled` でない） |
| canceled | 何もしない | 何もしない | 何もしない | 拒否：400 invalid_request |

<!-- /spec:state-machine -->

## 操作

### User の変更による実行の捕捉

#### REQ-IDGOVERNANCE-004 User の変更と WorkflowRun の捕捉は同じ整合性境界で確定し、キュー投入の障害から回復する

- `user_created`、`user_attributes_changed`、`user_status_changed` のどれかに一致する `enabled` のワークフローがある間、User を変更したとき、IdGovernance は、変更後の User に対してトリガーを評価し、User の変更と同じトランザクションで、重複排除のキー（`tenant_id`、`workflow_id`、`revision`、`source_occurrence_id`、`target_user_id`）を持つ `queued` の WorkflowRun と `pending` の WorkflowStep を確定させる。
- 同じ発火の事象を再び配信されたとき、IdGovernance は、既存の WorkflowRun に収束させ、WorkflowStep を重複して作らない。
- WorkflowRun を確定したとき、IdGovernance は、`lifecycle_workflow_run` の Job をキューへ投入し、WorkflowRun に関連付ける。
- キューへの投入が一時的に失敗した場合、IdGovernance は、WorkflowRun を `job_id` のない `queued` のまま残す。
- `worker` の定期のディスパッチャーが `job_id` のない `queued` の WorkflowRun を走査したとき、IdGovernance は、重複しない Job を関連付ける。
- **例**：EX-IDGOVERNANCE-004-01、EX-IDGOVERNANCE-004-02

#### REQ-IDGOVERNANCE-006 値が変わらない属性更新と動的グループのアクション指定は境界条件として扱われる

- 監視する属性の値が変わらない User の更新を受けた場合、IdGovernance は、`user_attributes_changed` のトリガーを発火させず、WorkflowRun を作らない。
- `add_group_member` のアクションに動的グループを指定したワークフローの保存を要求された場合、IdGovernance は、400 と `invalid_request` で拒否する。
- **例**：EX-IDGOVERNANCE-006-01

### worker による WorkflowRun の実行

#### REQ-IDGOVERNANCE-003 管理者が定義したワークフローが部署変更に応じてグループとアプリケーションの割り当てを更新する

- 管理者が監視する属性を更新したとき、IdGovernance は、トリガーのスナップショットに変わった項目（`changed_fields`）を持つ WorkflowRun を作る。
- `worker` が WorkflowRun を実行したとき、IdGovernance は、最初のステップを始める前に `running` にして `LifecycleWorkflowRunStarted` を発行し、未完了のステップを定義順にすべて実行し、ステップごとの結果（`changed`、`no_op`、`failed`）を記録する。
- すべてのステップが `changed` または `no_op` になったとき、IdGovernance は、WorkflowRun を `succeeded` にし、`LifecycleWorkflowRunSucceeded` を発行する。
- **例**：EX-IDGOVERNANCE-003-01

#### REQ-IDGOVERNANCE-005 すでに目的の状態にある操作は `no_op` として扱われる

- 対象がすでに目的の状態の間、`worker` がステップを実行したとき、IdGovernance は、ステップを `no_op` とし、所属や割り当ての重複を作らず、成功として数える。
- **例**：EX-IDGOVERNANCE-005-01

#### REQ-IDGOVERNANCE-009 一時的な失敗は再試行時に成功済みのステップを飛ばして収束する

- 失敗したステップが残り、Job の試行の回数が上限未満の場合、IdGovernance は、Jobs に再試行できるエラーを返し、WorkflowRun を `running` のまま残す。
- Jobs が WorkflowRun の Job を再試行したとき、IdGovernance は、`changed` と `no_op` のステップを再実行せず、`failed` のステップだけを再実行する。
- ステップが失敗したとき、IdGovernance は、機密の情報を除いたエラーコードを記録し、`LifecycleWorkflowStepFailed` を発行し、残りのステップの実行を続ける。
- Job の試行の上限に達し、成功と失敗のステップが混在する場合、IdGovernance は、WorkflowRun を `partially_failed` にし、`LifecycleWorkflowRunPartiallyFailed` を発行する。
- Job の試行の上限に達し、成功したステップが一つもない場合、IdGovernance は、WorkflowRun を `failed` にし、`LifecycleWorkflowRunFailed` を発行する。
- 管理者が `partially_failed` または `failed` の WorkflowRun を再試行したとき、IdGovernance は、`failed` のステップを `pending` に戻し、WorkflowRun を `queued` にして Job を投入し、200 と WorkflowRun を返す。
- `partially_failed` と `failed` 以外の WorkflowRun か、`enabled` でないワークフローの WorkflowRun の再試行を要求された場合、IdGovernance は、400 と `invalid_request` で拒否する。
- **例**：EX-IDGOVERNANCE-009-01

#### REQ-IDGOVERNANCE-010 同一ユーザーの WorkflowRun は発火順に直列化され、副作用の前に再検証される

- 同じ User の先行する WorkflowRun が終端の状態にない間、`worker` が後続の `queued` の WorkflowRun を取り出したとき、IdGovernance は、後続のアクションを始めず、Jobs に再試行させる。
- `worker` が各アクションを実行するとき、IdGovernance は、実行の直前に対象の User と Group または Application を同じテナントの中で取り直す。
- 削除済みか別のテナントのリソースを参照するステップを実行する場合、IdGovernance は、機密の情報を除いたエラーコードと `failed` の結果をチェックポイントに記録する。
- **例**：EX-IDGOVERNANCE-010-01

#### REQ-IDGOVERNANCE-008 通知に失敗してもアクセス剥奪操作は前進する

- メールの通知のステップが失敗した場合、IdGovernance は、そのステップを `failed` として記録し、同じ実行のアクセスを剥がすほかのステップ（`disable_user`、`remove_group_member` を含む）を実行し、補償しない。
- **例**：EX-IDGOVERNANCE-008-01

## セキュリティ上の考慮

WorkflowRun の実行は、管理者の権限を借りない。
対象の User、Group、Application は、いずれも実行の時点でワークフローと同じテナントの中で再取得する。
実行までに対象が削除または移動していれば、そのステップを失敗として記録する。
ワークフローがテナントの境界を越える経路は、定義の側にも実行の側にもない。

## 実現方式

### アーキテクチャ

IdManagement は、User を変更するときに、このモジュールが実装する `UserMutationCommitter` を呼ぶ。
`UserMutationCommitter` は変更の前後の User から WorkflowRun と WorkflowStep を計画し、User と一緒に一つのトランザクションで保存する。
モジュールをまたぐトランザクションを別に要求せずに、User だけが更新されて対応する実行が作られない事態を防ぐ。

保存した WorkflowRun は `job_id` のない `queued` で残る。
`worker` のディスパッチャーは、一定の間隔で `queued` の WorkflowRun を 100 件ずつ読み、重複しない `lifecycle_workflow_run` の Job を関連付ける。
Job の投入に失敗しても、次の走査で関連付け直す。

| アクション | 記録の正を持つモジュールへの書き込み |
| --- | --- |
| `enable_user`、`disable_user` | `UserLifecycle` のポートを通して、IdManagement の User の操作で行う。管理 API で止めたときと同じイベント、記憶済みの端末の失効、下流への通知、所有する Agent の無効化が伴う。削除予約中の User への操作は IdManagement が拒否し、ステップは変更なしとして扱う |
| `assign_application`、`unassign_application` | `ApplicationAssignments` のポートを通して、Application の割り当ての操作で行う |
| `add_group_member`、`remove_group_member` | `GroupMembership` のポートを通して、IdManagement の所属操作で行う。実際の変更に所属のイベントと下流へのプロビジョニング通知が伴う |
| `set_required_action`、`clear_required_action` | `UserRequiredActions` のポートを通して、IdManagement の必須操作の付与と解除を行う。実際の変更に対応するイベントが伴う |
| `send_email` | 共有の通知の部品で送る。対象の User に検証済みのプライマリのメールアドレスがなければ、ステップは失敗する |

アクションによる User の変更は `UserMutationCommitter` を通らないので、新しい WorkflowRun を計画しない。
`UserLifecycle` と `UserRequiredActions` の実装も、IdManagement の User の操作を `UserMutationCommitter` なしで呼ぶ。
アクションからトリガーへの循環は、この経路の分離で断たれる。

操作のポートは IdGovernance が定義し、IdManagement が実装する。
bootstrap が管理 API と同じ保存先、イベントの配信点、通知先を組み立て、Worker が実行ハンドラーへ渡す。
実行前の評価は Repository の読み取りを使い、状態の変更は操作のポートへ渡す。
評価後に状態が変わっても、操作の冪等な結果に応じて `changed` または `no_op` を記録する。
変更が必要な手順にポートがなければ、`dependency_unavailable` で失敗する。

### 信頼性

ステップは、実行するたびに結果をチェックポイントとして記録する。
プロセスが止まっても、Job の再試行は記録した結果を読み、`failed` と未完了のステップだけを実行する。
各ステップの前にワークフローの状態を読み直し、無効化または削除されていれば、次のステップの前で打ち切る。
