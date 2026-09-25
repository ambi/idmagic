---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-21
priority: p2
depends_on: []
change_kind: bugfix
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: Full Resync の完了が FullResyncCompleted として記録されるようになり、運用者と監査の読者が全対象の収束を観測できる。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-22987-track-full-resync-completion.md }
initial_context:
  specification:
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-013
    - docs/domain/provisioning/internals.md
    - docs/domain/provisioning/states.md
  typespec: [Provisioning.FullResyncCompleted, Provisioning.StartFullResync]
  source:
    - backend/provisioning/usecases/admin.go
    - backend/provisioning/usecases/job_handler.go
    - backend/provisioning/usecases/execute_task.go
    - backend/provisioning/domain/task.go
    - backend/provisioning/domain/events.go
    - backend/provisioning/ports/repositories.go
    - backend/provisioning/db_postgres/provisioning.sql
    - backend/provisioning/module.go
    - infra/schema/postgres.sql
  tests:
    - backend/provisioning/e2e_lifecycle_events_test.go
    - backend/provisioning/e2e_capture_task_test.go
    - backend/provisioning/handlers_http/admin_connection_events_test.go
  stop_before_reading: [frontend, backend/provisioning/client_scim, backend/cmd]
primary_use_cases:
  - id: full-resync-completes-once
    requirement: REQ-PROVISIONING-013
    observable_result: 管理 API で開始した Full Resync の全プロビジョニングタスクを worker が終端にすると、FullResyncCompleted が対象数と成功数と失敗数を持って一度だけ発行される。
    unit_test: { path: backend/provisioning/usecases/full_resync_test.go, name: TestFullResyncCompletesOnceWhenTheLastTaskSettles, task: test-go-race }
    e2e_test: { path: backend/provisioning/e2e_full_resync_test.go, name: TestE2E_FullResyncEmitsCompletedOnceAllTasksSettle, task: test-go-race }
    unit_fault_model: 完了の判定が、終端でないプロビジョニングタスクや未作成のプロビジョニングタスクが残るうちに完了とみなす、または完了済みの Full Resync を再び完了させる。
    e2e_fault_model: ジョブハンドラーがプロビジョニングタスクの終端化の後に Full Resync の完了判定を呼ばない。
affected_spec:
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-013 }
---

# Full Resync のプロビジョニングタスクの完了を追跡し、全対象が収束したときに完了イベントを発行する

## 動機

`EX-PROVISIONING-013-01` は、scope 内の全 subject にプロビジョニングタスクを作成し、すべてが収束したとき `FullResyncCompleted` を発行すると宣言する。
現行の `StartFullResync` はプロビジョニングタスクを作るだけで、非同期のプロビジョニングタスク群を同じ resync として追跡せず、実装コメントでもイベント未実装を明記している。

## 対象範囲

- Full Resync ごとの対象プロビジョニングタスク、成功、失敗、完了を追跡する状態を実装する。
- すべての対象プロビジョニングタスクが終端状態になった時点で `FullResyncCompleted` を発行する。
- `REQ-PROVISIONING-013` を参照する単体テストと、実配線での最終イベントを観測する受け入れテストを追加する。
- 仕組みを `docs/domain/provisioning/internals.md`、表を `infra/schema/postgres.sql` と `docs/design/data/database.md` に反映する。

## 対象外

- Full Resync の対象選択規則の変更。
- 外部 SCIM の同期アルゴリズムや差分計算の追加。
- 他の Provisioning イベントの発行。
- 取りこぼした完了判定を周期処理で回収すること。
  完了判定はプロビジョニングタスクを終端にしたジョブの中で行い、判定が失敗すればジョブの再実行が判定をやり直す。
  最後の試行で判定だけが失敗した場合は `running` のまま残り、イベントは発行されない。
  早すぎる発行や重複発行は起きないため、回収は必要が生じたときに別の記録で扱う。
- `StartFullResync` の応答へ Full Resync の識別子を加えること。公開契約は変えない。

## 設計

非同期のプロビジョニングタスクの一括処理には、個々の `ProvisioningTask` だけでは表せない親の追跡単位が必要である。
親を `FullResync` とし、開始時に対象数を確定して保存する。
対象のプロビジョニングタスクは別表 `provisioning_full_resync_tasks` で親へ関連付ける。
`provisioning_tasks` の列と `ProvisioningTask` の JSON 表現は変えないため、タスク一覧 API の契約は変わらない。

### ドメインの型と操作

```go
type FullResyncStatus string // "running" | "completed"

type FullResync struct {
    ID, TenantID, ConnectionID string
    Status                     FullResyncStatus
    TotalTasks                 int
    SucceededCount, FailedCount int
    StartedAt                  time.Time
    CompletedAt                *time.Time
}

// FullResyncTally は親へ関連付いた終端のプロビジョニングタスクを数えた結果である。
type FullResyncTally struct{ Succeeded, Failed int }

func NewFullResync(id, tenantID, connectionID string, totalTasks int, now time.Time) *FullResync
// Settle は running で、終端の件数が対象数に達したときだけ completed を返す。
func (r FullResync) Settle(t FullResyncTally, now time.Time) (FullResync, bool)
func (r FullResync) CompletedEvent() *FullResyncCompleted
```

関連付けは一件ずつで対象数を超えないので、終端の件数が対象数に達したことは全件が作られて終端になったことを表す。
このため、`StartFullResync` がプロビジョニングタスクを作り終える前に先頭のプロビジョニングタスクが終わっても完了しない。

### 作用の境界

| 作用 | 境界 |
| --- | --- |
| 識別子の生成 | `spec.NewUUIDv4`（`StartFullResync`） |
| 時刻 | `StartFullResync` の `now` 引数、ジョブハンドラーの `JobHandlerDeps.Now` |
| 永続化 | `ports.ProvisioningTaskRepository` の `SaveFullResync`、`SaveFullResyncTask`、`FindFullResyncByTask`、`TallyFullResync`、`CompleteFullResync` |
| 通知 | `AdminDeps.Emit`、`JobHandlerDeps.Emit` |

`SaveFullResyncTask` はプロビジョニングタスクの挿入と関連付けを一つの SQL 文で行う。
`CompleteFullResync` は `status='running'` のときだけ成立する条件付き更新であり、成立した呼び出しだけがイベントを発行する。
二つのジョブが最後の二件を同時に終えても、各ジョブは自分の終端化を保存した後に数えるので、後に数えたほうが全件の終端を観測し、条件付き更新が発行を一度に限る。

### 採用しない代替案

- `provisioning_tasks` へ `full_resync_id` 列を加える。すべての SELECT とタスクの JSON 表現に波及し、API へ内部の識別子が漏れる危険がある。
- 親に成功数と失敗数を加算していく。ジョブの再実行や管理者の再試行で二重に数え得る。完了時に数え直すほうが冪等である。

## 計画

1. `internals.md` に Full Resync の完了追跡を書き、`mise run check-spec` を通す。
2. E2E RED を確認する。
3. ドメインの `Settle` を単体 RED から実装する。
4. 保存先（memory と Postgres）、`StartFullResync`、ジョブハンドラーの順に接続する。
5. 変異、対象パッケージ、`mise run verify` を実行する。

## タスク

- [x] T001 [Readiness] resync 親状態の所有者とトランザクション境界を決める。
- [x] T002 [Spec] `internals.md`、スキーマ、データベース設計を更新する。
- [x] T003 [Acceptance] `EX-PROVISIONING-013-01` の E2E RED を確認する（`TestE2E_FullResyncEmitsCompletedOnceAllTasksSettle`）。
- [x] T004 [Domain] 全対象の終端判定を単体 RED から実装する（`TestFullResyncSettle*`、`REQ-PROVISIONING-013`）。
- [x] T005 [Use Cases] 開始・プロビジョニングタスク処理・イベント発行を接続する（`TestFullResyncCompletesOnceWhenTheLastTaskSettles`、`EX-PROVISIONING-013-01`）。
- [x] T006 [Adapters] memory と Postgres の保存先を実装する。
- [x] T007 [Verify] 変異、対象パッケージ、`mise run verify` を実行する。

各 RED と GREEN には `mise run test-go-test -- <package> <test>` を使い、振る舞いが GREEN になったら `mise run test-go-package` と `mise run lint-go` を実行する。

## 検証

- `mise run test-go-package -- ./backend/provisioning/domain`
- `mise run test-go-package -- ./backend/provisioning/usecases`
- `mise run test-go-package -- ./backend/provisioning`
- `mise run test-go-mutation -- backend/provisioning/domain`
- `mise run test-go-mutation -- backend/provisioning/usecases`
- `mise run verify`

## リスク

早すぎる完了イベントはまだ下流へ反映していない subject を成功と誤認させ、重複発行は監査と運用の判断を壊す。
複数のプロビジョニングタスク・失敗・再試行を含む状態遷移を、イベントの回数まで含めて検証する。

## 完了

- **Completed At**: 2026-09-25
- **Summary**:
  `mise run spec-diff` は main に対する規範仕様の変更なしを報告した（`REQ-PROVISIONING-013` と `FullResyncCompleted` の宣言は変えていない）。
  `EX-PROVISIONING-013-01` の振る舞いを実装し、被覆の負債から外した。
  `StartFullResync` は重複を除いた対象数を持つ `FullResync` を保存し、各プロビジョニングタスクを一つの SQL 文で挿入して関連付ける。
  ジョブハンドラーはプロビジョニングタスクを `succeeded` または `dead_letter` として保存した後、終端の件数を数え直し、対象数に達していれば `running` のときだけ成立する条件付き更新で完了させ、成立したときだけ `FullResyncCompleted` を発行する。
  対象が 0 件なら `StartFullResync` が開始と同時に完了させる。
  `provisioning_full_resyncs` と `provisioning_full_resync_tasks` の表を追加し、`internals.md` に完了追跡の仕組みを記した。
  タスク一覧 API と `StartFullResync` の応答は変わらない。
- **Primary Use Case Evidence**:
  - id: full-resync-completes-once
    unit_red: StartFullResync とジョブハンドラーが完了を判定しないため、TestFullResyncCompletesOnceWhenTheLastTaskSettles が「FullResyncCompleted = 0 events, want exactly one」で失敗した。ドメインの TestFullResyncSettle* 三件も、完了を返さない仮実装で失敗した。
    e2e_red: 管理 API の Full Resync で作った 3 件がすべて UserProvisioned になっても FullResyncCompleted が発行されず、TestE2E_FullResyncEmitsCompletedOnceAllTasksSettle が「FullResyncCompleted = 0 events」で失敗した。
    unit_fault_injection: Settle から対象数の条件を外すと、開始直後に完了して TestFullResyncCompletesOnceWhenTheLastTaskSettles が失敗した。失敗時の経路から完了判定を外すと、最後の dead_letter の直後に完了していないため同じテストが失敗した。
    e2e_fault_injection: ジョブハンドラーの成功時の経路から完了判定の呼び出しを外すと、TestE2E_FullResyncEmitsCompletedOnceAllTasksSettle が「FullResyncCompleted = 0 events」で失敗した。
- **Change-Resistance Results**:
  - 手書きの障害: `Settle` の `running` 条件を外すと TestFullResyncSettleRefusesEarlyOrRepeatedCompletion/完了済み が、完了の条件付き更新の結果を無視すると、または memory の保存先で条件付き更新を外すと TestFullResyncConcurrentSettlementEmitsOnce が、開始時の完了判定を外すと TestStartFullResyncCompletesAnEmptyScopeImmediately が、Group の重複除去を外すと TestFullResyncCountsADuplicatedSubjectOnce が失敗した。
  - 最初の注入では、失敗時の完了判定を外す障害と重複除去を外す障害が生き残った。前者は再実行が判定をやり直すため最後の失敗の直後の表明を足し、後者は明示指定の Group ID が重複を拒否されないことを確かめて重複のテストを足した。
  - 完了条件の「未決着 0 件」と「終端の件数が対象数に一致」は重複するガードで、前者だけを外す障害はどのテストでも区別できなかった。関連付けは一件ずつで対象数を超えないので後者が前者を含むと判断し、前者を削除した。
  - `mise run test-go-mutation -- backend/provisioning/domain`: `full_resync.go` の変異 4 件はすべて検出された。生存 2 件（`connection.go:256`、`task.go:149`）と未被覆 8 件は、この作業で変更していないコードである。
  - `mise run test-go-mutation -- backend/provisioning/usecases`: `full_resync.go` の変異はすべて検出された。変更した行で生き残ったのは `admin.go:392`（開始時の完了判定のエラー判定の否定）だけで、正常時はどちらも nil を返すため判定の失敗を注入しなければ区別できない。この失敗を起こすテストがない不足として残す。ほかの生存と未被覆は既存の行である。
  - Postgres の保存先は TestProvisioningTaskRepository_FullResyncTalliesAndCompletesOnce で固定した。サンドボックス内では embedded-postgres が起動できずスキップされるため、サンドボックス外で実行し、一時的に入れた失敗が報告されることで実際に走ったことを確かめた。
- **Verification Results**:
  - `mise run test-go-package -- ./backend/provisioning/...` - 成功
  - `mise run test-go-changed` - 成功
  - `mise run lint-go` - 成功
  - `mise run verify` - 成功
