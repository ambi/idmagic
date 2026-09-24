---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-24
priority: p2
depends_on: []
change_kind: feature
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: 捕捉を呼ばない経路の変更も、照合によって下流へ反映されるようになる。運用者は照合の周期を設定で変えられる。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-92540-guarantee-provisioning-deliveries-by-reconciliation.md }
initial_context:
  specification:
    - docs/domain/scenarios.feature.md#REQ-PLATFORM-003
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-019
    - docs/domain/provisioning/internals.md
    - docs/domain/provisioning/decisions.md
  typespec: []
  source:
    - backend/provisioning/usecases/capture.go
    - backend/provisioning/usecases/execute_task.go
    - backend/provisioning/usecases/admin.go
    - backend/provisioning/domain/task.go
    - backend/provisioning/ports/repositories.go
    - backend/provisioning/module.go
    - backend/idmanagement/user/ports/user_repository.go
    - backend/cmd/idmagic-worker/worker.go
    - backend/cmd/internal/bootstrap/workerconfig.go
  tests:
    - backend/provisioning/e2e_capture_task_test.go
    - backend/provisioning/usecases/capture_test.go
  stop_before_reading: [frontend, backend/provisioning/client_scim, backend/provisioning/handlers_http]
primary_use_cases:
  - id: reconcile-uncaptured-deactivation
    requirement: REQ-PLATFORM-003
    observable_result: 書き込み時の捕捉を通らずに無効化された User が、次の照合で下流へ `active=false` として反映される。
    unit_test: { path: backend/provisioning/domain/reconcile_test.go, name: TestPlanReconciliation_DeactivatesAUserDisabledWithoutCapture, task: test-go-race }
    e2e_test: { path: backend/provisioning/e2e_reconcile_test.go, name: TestE2E_ReconcileDeactivatesAUserDisabledWithoutCapture, task: test-go-race }
    unit_fault_model: 照合の計算が、リンクの有効状態と User の有効状態の食い違いを差分として扱わない。
    e2e_fault_model: 照合のユースケースが、計算した差分をプロビジョニングタスクとして保存しない。
affected_spec:
  - { path: docs/domain/scenarios.feature.md, requirement: REQ-PLATFORM-003 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-019 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-003 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-004 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-005 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-017 }
  - { path: docs/domain/provisioning/standards.md, requirement: RFC7644-OUT-FILTERING }
---

# プロビジョニングタスクの作成を、書き込み時の捕捉ではなく定期的な照合で保証する

## 動機

`REQ-PLATFORM-003` は「プロビジョニングタスクの行は発火元の変更と同時にコミットまたはロールバックする」と定めていた。
[[wi-22350-capture-provisioning-deliveries-in-the-mutation-transaction]] は、管理 API の経路でこれをトランザクションの作用範囲によって実現しようとした。
しかし、その過程で次のことが分かり、取り消した。

| 観察 | 内容 |
| --- | --- |
| 捕捉が経路ごと抜けている | LifecycleWorkflow による有効化と無効化（`idgovernance/usecases/lifecycle_workflow_dispatcher.go`）、SCIM による取り込み（`sourcing/scim/usecases/users.go`）、CSV インポート、利用者自身のプロフィール変更は、User を書き換えてもプロビジョニングタスクを作らない。障害がなくても、毎回確実に下流とずれる |
| 書き込み時の保証は漏れる | 記録の正を書き換える経路は今後も増える。経路ごとに捕捉を正しく呼ぶ前提は、新しい経路が増えるたびに破れる |
| 横断的な仕組みの量が見合わない | 原子性のために、全保存先を context のトランザクションへ参加させる仕組みと、memory 側の書き戻しの仕組みが必要になった。それでも上の漏れには効かない |
| 同一 DB の外は囲めない | Valkey に置く状態や外部の呼び出しは、どのみちトランザクションに入らない |

Provisioning のプロビジョニングタスクは状態ベースである。
下流へ送る属性は実行の時点の User から読み、`RemoteResourceLink` が下流に何があるかを記録している。
そのため、あるべき状態と下流へ反映済みの状態を突き合わせれば、取りこぼしの原因を問わずに回収できる。
Jobs が「配送は少なくとも 1 回、冪等性はハンドラーの責務」とし、dispatcher が未投入のプロビジョニングタスクの行を拾い直すのと同じ考え方である。

## 対象範囲

- `REQ-PLATFORM-003` を、書き込み時の原子性ではなく、照合による最終的な反映の保証へ改める。
- Provisioning に照合のルール `REQ-PROVISIONING-019` を加える。
- 接続ごとに、スコープ内の User のあるべき状態と、下流へ反映済みの状態を突き合わせ、差分に対して `ProvisioningTask` を作る照合を実装し、`worker` の定期ループとして走らせる。
- `RemoteResourceLink` に、下流で有効として反映したかどうかを記録する。User の削除を下流へ送ったらリンクを消す。
- 書き込み時の捕捉（`notifyProvisioning`）は、反映の遅延を短くする近道として best-effort のまま残す。
- `tools/check/example-coverage-debt.json` から `EX-PLATFORM-003-01` と `EX-PLATFORM-003-02` を外し、改訂後の具体例をテストで裏付ける。
- `docs/domain/provisioning/internals.md` と `decisions.md` の書き込み時の記録の記述を現状に合わせる。

## 対象外

- 捕捉が抜けている各経路（LifecycleWorkflow、SCIM の取り込み、CSV インポート、プロフィール変更）への個別の捕捉の追加。照合が回収するので、遅延が問題になる経路が分かってから別の項目で扱う。
- Group の照合。まず User で照合の形を確かめてから、同じ形で広げる。
- 誤削除ガード（`REQ-PROVISIONING-011`）。照合が作る無効化と削除にもガードを効かせる方針だが、ガード自体は [[wi-61629-quarantine-connections-on-the-accidental-deletion-guard]] が実装する。照合は Full Resync と同じ「1 回の同期」として数えられる形で無効化と削除を作る。
- Full Resync を照合の計算の上へ作り直すこと。別の項目で扱う。
- `updated_at` の高水位線による差分の走査。まず全件の走査で照合の形を確かめ、読み取りの量が問題になってから導入する。
- プロビジョニングタスクの実行（`worker` による下流への反映）の信頼性。既存の Jobs の再試行が担う。

## 設計

### 解決した問い

| 問い | 決定 |
| --- | --- |
| 照合の範囲 | 接続ごとに全件を走査する。1 回の照合で 1 接続に作るプロビジョニングタスクの数に上限を設け、超えた分は次の照合が拾う |
| 反映済みの状態の持ち方 | `RemoteResourceLink` に保持する。バージョンの `last_synced_version` に加えて、下流で有効として反映したかどうかの `active` を記録する |
| 誤削除ガードとの関係 | 効かせる。ガードの実装は wi-61629 が担う |
| 照合の実行場所 | ディスパッチャーと同じ `worker` の定期ループ。周期は `PROVISIONING_RECONCILE_INTERVAL`（デフォルト 5 分） |
| Full Resync の作り直し | 別の項目で扱う |

### 反映済みの状態に `active` が必要な理由

バージョンだけを比べると、User 自身が変わらない状態の変化を見分けられない。
割り当ての解除では User の `updated_at` は変わらないので、「スコープ外でリンクがある」という形でしか差分が見えない。
無効化を送ったあとも、リンクは残る。
リンクに下流での有効状態がなければ、照合は同じ無効化を周期ごとに作り続ける。
再び割り当てたときも、User のバージョンはリンクより新しくないので、再有効化が作られない。

### 照合の計算

照合の計算は、作用を持たない純粋な関数として `backend/provisioning/domain` に置く。

```go
// ReconcileUser は照合が 1 人の User について読む事実である。
type ReconcileUser struct {
    ID      string
    Deleted bool  // 削除済み。リンクを持つ User だけが削除済みで現れる
    Active  bool  // User のステータスが active である
    InScope bool  // 接続のスコープ（all_users または割り当て）に入っている
    Version int64 // User の updated_at の UnixNano
}

// ReconcileTask は照合が User ごとに読む、確定していないか失敗したプロビジョニングタスクである。
type ReconcileTask struct {
    Status        ProvisioningTaskStatus
    SourceVersion int64
}

type ReconcileInput struct {
    Connection ProvisioningConnection
    Users      []ReconcileUser
    Links      map[string]RemoteResourceLink // User の ID をキーとする
    Tasks      map[string][]ReconcileTask    // User の ID をキーとする
    Limit      int
}

// ReconcileAction は照合が作るもの 1 件である。Schedule が true なら、プロビジョニングタスクではなく猶予期間つき削除の予約を作る。
type ReconcileAction struct {
    UserID    string
    Operation ProvisioningOperation
    Schedule  bool
}

func PlanReconciliation(in ReconcileInput) []ReconcileAction
```

| あるべき状態 | 反映済みの状態 | 作るもの |
| --- | --- | --- |
| スコープ内で有効 | リンクなし | `create`（`create_users` が有効なとき） |
| スコープ内で有効 | リンクが無効 | `update`（再有効化。`update_users` が有効なとき） |
| スコープ内で有効 | リンクが有効、User のバージョンがリンクより新しい | `update`（`update_users` が有効なとき） |
| スコープ内で無効 | リンクが有効 | `deactivate`（`deactivate_users` が有効なとき） |
| スコープ外 | リンクあり | `on_unassign` に従う。`deactivate` はリンクが有効なときだけ作る |
| 削除済み | リンクあり | `on_delete` に従う。`delete` で `grace_period_days` が 1 以上なら予約を作る |
| 上記以外 | — | なし |

次の User は飛ばす。

- `pending` または `in_flight` のプロビジョニングタスクがある。書き込み時の捕捉や前回の照合が作ったものがまだ終わっていない。
- `dead_letter` のプロビジョニングタスクがあり、そのバージョンが User のバージョン以上である。前回の失敗から User が変わっていないので、作り直しても同じく失敗する。管理者の再試行か、User の変更を待つ。

結果は User の ID の順に並べ、`Limit` 件で打ち切る。

### 作用の境界

`usecases.ReconcileConnections(ctx, deps ReconcileDeps, limit int, now time.Time) (created int, err error)` が作用を担う。

| 作用 | 境界 |
| --- | --- |
| 接続の読み取り | `ProvisioningConnectionRepository.ListActive`（全テナント、有効で隔離されていない接続） |
| User の読み取り | `userports.UserRepository.FindAll`、リンクだけが残る User は `FindBySubIncludingDeleted` |
| 割り当ての読み取り | `appports.AssignmentRepository.ListByApplication` |
| リンクの読み取り | `RemoteResourceLinkRepository.ListByConnection` |
| 未確定と失敗のタスクの読み取り | `ProvisioningTaskRepository.ListUnsettledByConnection` |
| 書き込み | `ProvisioningTaskRepository.Save`、`ScheduleDeprovision` |
| 時刻、識別子 | 引数の `now`、`spec.NewUUIDv4` |

照合が作るプロビジョニングタスクのバージョンは `now.UnixNano()` とする。
書き込み時の捕捉も同じく時刻をバージョンにしているので、バージョンの比較はどちらも User の `updated_at` と同じ時間軸に乗る。
当初は捕捉と照合のバージョンを User の版に揃えて冪等キーで二重作成を防ぐ計画だった（旧 T004）。
しかし、未確定のタスクがある User を飛ばせば、同じ変更を二重に作ることはない。
加えて、冪等キーは操作を含まないので、同じバージョンで `create` と `deactivate` が衝突して片方が消える。
このため揃えない。

### 実行側の変更

- `RemoteResourceLink.ApplySync` に `active` を渡す。`create` と `update` は下流へ送った `active` 属性を、`deactivate` は `false` を記録する。
- User の `delete` を下流へ送ったら、リンクを消す（`RemoteResourceLinkRepository.Delete`）。リンクなしが「下流に何もない」を意味するためである。

### 採用しない代替案

- **書き込み時の原子性（wi-22350 の方式）。** 経路ごとの漏れに効かず、全保存先を作用範囲へ参加させる仕組みが要る。取り消した実装の設計は wi-22350 に残っている。
- **すべての経路へ捕捉を追加して回る。** 今ある漏れは塞げるが、新しい経路が増えるたびに同じ漏れが再発する。
- **User の変更イベントを outbox に積み、それを中継する。** プロビジョニングタスクの行そのものが outbox の役割を果たしており、経路ごとに積み忘れる問題は変わらない。
- **反映済みの状態をプロビジョニングタスクの行から都度求める。** 照合のたびに接続の全タスクを読むことになり、最後に成功した操作を選ぶ規則も要る。

## 計画

1. 仕様を改め、具体例を宣言する。
2. 捕捉を呼ばない経路で無効化した User が、照合で反映されることを E2E の RED として観測する。
3. 照合の計算を純粋な関数として実装する。
4. リンクの `active` と削除時のリンクの除去を実装する。
5. リポジトリ、ユースケース、`worker` の定期ループを組み立てる。
6. 文書と台帳を更新する。

## タスク

- [x] T001 [Spec] `REQ-PLATFORM-003` を改め、`REQ-PROVISIONING-019` を加える。
- [x] T002 [Acceptance] 捕捉を呼ばない経路で User を無効化しても、プロビジョニングタスクが作られないことを RED として観測する。
- [x] T003 [Domain] 照合の計算を純粋な関数として実装する。
- [x] T004 [Domain] リンクの `active` と、削除時のリンクの除去を実装する。
- [x] T005 [Adapter] リポジトリの読み取りと、照合のユースケースを実装する。
- [x] T006 [Infrastructure] 照合を `worker` の定期ループとして組み立てる。
- [x] T007 [Docs] Provisioning の内部設計と設計判断を現状に合わせ、被覆台帳を更新する。
- [x] T008 [Verify] 変更を検証する。

## 検証

- `mise run test-go-test -- ./backend/provisioning/domain TestPlanReconciliation`
- `mise run test-go-test -- ./backend/provisioning TestE2E_Reconcile`
- `mise run test-go-mutation -- backend/provisioning/domain`
- `mise run check-spec`
- `mise run verify`

## リスク

- **照合が大量のプロビジョニングタスクを一度に作る。** 初回の照合や長い停止の後は差分が大きい。1 回に作る数の上限で抑える。誤削除ガードは wi-61629 が加える。
- **照合と書き込み時の捕捉が同じ変更を二重に扱う。** 未確定のタスクがある User を飛ばす。照合が読んだ後に捕捉がタスクを作る競合では二重になり得るが、`update` と `deactivate` は下流で冪等であり、`create` は 409 で既存リソースへ関連付く。
- **照合の走査が DB を圧迫する。** 周期を設定で制限する。高水位線による差分の走査は、読み取りの量が問題になってから導入する。
- **削除済み User の読み取りがテナントを越える。** `FindBySubIncludingDeleted` はテナントを取らないので、読んだ User のテナントが接続のテナントと一致することを確かめる。

## 完了

- **Completed At**: 2026-09-25
- **Summary**:
  `mise run spec-diff` は、追加したシナリオ `REQ-PROVISIONING-019`、変更したシナリオ `REQ-PLATFORM-003`、`REQ-PROVISIONING-003`、`REQ-PROVISIONING-017`、変更した標準行 `RFC7644-OUT-FILTERING` を挙げた。
  `REQ-PLATFORM-003` は、プロビジョニングタスクを発火元の変更と同じトランザクションで作る保証から、書き込み時の捕捉または定期的な照合のどちらかでプロビジョニングタスクにする保証へ変わった。
  具体例は、捕捉による通常経路、捕捉の失敗を照合が回収する経路、捕捉を呼ばない経路を照合が回収する経路の 3 つになった。
  `REQ-PROVISIONING-019` は照合の規則として、リンクのない有効な User への `create`、割り当てを解除された User への `on_unassign` に従う無効化、未確定のタスクがある User を飛ばすこと、猶予期間つき削除を予約にすることを宣言した。
  `REQ-PROVISIONING-003` と `REQ-PROVISIONING-017` は、捕捉が発火元のコミットの後に行われる現状に合わせて Given の文面だけを改めた。
  `RFC7644-OUT-FILTERING` は、照合（Reconciliation）と紛れる「照合属性」を「一致判定の属性」へ言い換えただけである。
  実装では、`RemoteResourceLink` に `active` を加え、User の削除を下流へ送ったらリンクを消すようにし、`worker` に `PROVISIONING_RECONCILE_INTERVAL`（デフォルト 5 分）ごとの照合のループを加えた。
  書き込み時の捕捉と照合のバージョンを揃える当初の計画（旧 T004）は、未確定のタスクがある User を飛ばすことで二重作成を防げるため、採らなかった。
- **Primary Use Case Evidence**:
  - id: reconcile-uncaptured-deactivation
    unit_red: 照合の計算が空の結果を返す段階で、TestPlanReconciliation_DeactivatesAUserDisabledWithoutCapture が deactivate を期待して失敗した。
    e2e_red: 照合のユースケースが何も作らない段階で、TestE2E_ReconcileDeactivatesAUserDisabledWithoutCapture と TestE2E_ReconcileRecoversAFailedCapture が created = 0, want 1 deactivation で失敗した。
    unit_fault_injection: 変異テストで reconcile.go の 12 件の変異をすべて検出した。未確定のタスクを飛ばす判定を外すと、TestPlanReconciliation_SkipsAUserWithAnUnsettledTask と TestPlanReconciliation_RetriesADeadLetteredUserOnlyAfterTheUserChanges が失敗した。
    e2e_fault_injection: 照合のユースケースが差分を保存しないようにすると、TestE2E_ReconcileDeactivatesAUserDisabledWithoutCapture が created = 0, want 1 deactivation で失敗した。
- **Change-Resistance Results**:
  `mise run test-go-mutation -- backend/provisioning/domain` は 74 件中 72 件を検出した。
  生き残った 2 件は既存の `ProvisioningConnection.Validate` と `ProvisioningTask.Validate` の境界の変異で、今回の変更の外にある。
  変異器が表現できない配線の除去は手で注入した。
  削除済み User を照合の入力へ加える処理を外すと、TestReconcileConnections_SchedulesTheDeletionOfADeletedUserWithAGracePeriod が失敗した。
  リンクへ `active` を記録する処理とリンクの削除は、実装前に TestExecuteTask_RecordsTheDownstreamActiveStateOnTheLink と TestExecuteTask_DeleteRemovesTheLink の失敗として観測した。
  `worker` の照合のループの配線は、ビルドでだけ確かめている。
- **Verification Results**:
  - `mise run test-go-changed` - 成功（PostgreSQL のテストを飛ばさないよう、サンドボックスの外で実行）
  - `mise run lint-go` - 成功
  - `mise run check-spec` - 成功
  - `mise run verify` - 成功（サンドボックスの外で実行）
  - `mise run test-ui-e2e` - 未実行。画面と API の契約は変えていない
