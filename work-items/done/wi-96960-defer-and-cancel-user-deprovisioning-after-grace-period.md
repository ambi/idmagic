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
  reason: 猶予期間を設定した接続で、User の削除が下流へ届く時期が変わることを運用者へ知らせる必要がある。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-96960-defer-and-cancel-user-deprovisioning-after-grace-period.md }
initial_context:
  specification:
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-006
    - docs/domain/provisioning/glossary.md
    - docs/domain/provisioning/internals.md
  typespec: []
  source:
    - backend/provisioning/usecases/capture.go
    - backend/provisioning/usecases/dispatcher.go
    - backend/provisioning/usecases/notify_adapters.go
    - backend/provisioning/ports/repositories.go
    - backend/provisioning/db_memory/repositories.go
    - backend/provisioning/db_postgres/provisioning.sql
    - backend/provisioning/module.go
    - backend/idmanagement/user/usecases/admin_users.go
    - infra/schema/postgres.sql
  tests:
    - backend/provisioning/e2e_lifecycle_events_test.go
    - backend/provisioning/e2e_capture_delivery_test.go
  stop_before_reading: [frontend, spec/contexts]
affected_spec:
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-006 }
primary_use_cases:
  - id: deprovision-after-grace-period
    requirement: REQ-PROVISIONING-006
    observable_result: on_delete=delete、grace_period_days=7 の接続で User を削除すると、7 日が経つまでディスパッチャーは delete の配信を作らず下流へ何も送らない。7 日の経過後にディスパッチャーが delete の配信を作り、worker が下流へ DELETE を送って UserDeprovisioned（action=delete）を発行する。
    unit_test: { path: backend/provisioning/usecases/dispatcher_test.go, name: TestDispatchPendingTasks_MaterializesAScheduledDeprovisionOnlyOnceDue, task: test-go-race }
    e2e_test: { path: backend/provisioning/e2e_lifecycle_events_test.go, name: TestE2E_DeletionWithAGracePeriodSendsDELETEOnlyAfterItElapses, task: test-go-race }
    unit_fault_model: 期限の比較を誤り、期限の前に予約を配信へ変える、または期限に達しても変えない。
    e2e_fault_model: 捕捉が猶予期間を無視して削除の直後に delete の配信を作る、またはディスパッチャーが予約を配信へ変えないため下流へ DELETE が届かない。
  - id: cancel-deprovision-on-reassignment
    requirement: REQ-PROVISIONING-006
    observable_result: 猶予期間中に User が同じ Application へ再び割り当てられると、予約していた delete は取り消され、期限の経過後も下流へ DELETE を送らない。
    unit_test: { path: backend/provisioning/usecases/capture_test.go, name: TestCaptureLifecycleEvent_ReassignmentCancelsOnlyThatConnectionsScheduledDeprovision, task: test-go-race }
    e2e_test: { path: backend/provisioning/e2e_lifecycle_events_test.go, name: TestE2E_ReassignmentWithinTheGracePeriodCancelsTheDELETE, task: test-go-race }
    unit_fault_model: 割り当ての追加が予約を取り消さない、または別の接続の予約まで取り消す。
    e2e_fault_model: 割り当て通知の経路が取消に届かず、期限の経過後に下流へ DELETE が送られる。
---

# 猶予期間を持つ User 削除の下流配信を延期し、再割り当て時に取り消す

## 動機

`EX-PROVISIONING-006-01` と `EX-PROVISIONING-006-02` は、削除後 7 日間は delete を配信せず、猶予期間中の再割り当てで予約済みの配信を取り消すことを宣言する。
現行の `CaptureLifecycleEvent` は `TriggerUserDeleted` をただちに delete の `ProvisioningDelivery` へ変換するため、この規範に一致しない。

## 対象範囲

- `REQ-PROVISIONING-006` の猶予期間中の非送出、経過後の delete 配信、再割り当てによる取消を実装する。
- 配信予定と取消を永続化し、worker が期限到来を処理できる境界を設ける。
- 実時間を直接読むのではなく、期限判定へ明示した時刻を渡す。
- 規範 ID を参照する単体テストと、下流への DELETE を観測する受け入れテストを追加する。

## 対象外

- `on_delete=deactivate` と `on_delete=none` の既存意味の変更。猶予期間を適用するのは delete へ変換された User の削除だけとする。
- User 以外の猶予期間と、`on_unassign=delete` の割り当て解除への猶予期間の適用。規範は User の削除だけを宣言している。
- 既存の下流 SCIM クライアントのプロトコル変更。
- `RestoreUser` による復元で予約を取り消すこと。復元はいま Provisioning へ何も通知しておらず、通知の追加は IdManagement 側の変更になる。規範の例は再割り当てだけを宣言している。
- 期限到来時に接続の設定（状態、`on_delete`、`delete_users`）を再評価すること。予約は削除時点の判断を保存し、接続の削除だけが外部キーの連鎖で予約を消す。

## 設計

猶予期間は即時の `ProvisioningDelivery` を遅延実行する問題ではなく、再割り当てで取り消せる予約状態を持つ問題である。
予約を通常の pending 配信へ偽装すると、worker が期限前に送出した場合と取消対象を区別できない。
期限、接続、User、元イベントのバージョンを型付きの予約として保存し、期限到来時にだけ配信を生成する。

再割り当ては同じ User と Application の有効な予約だけを取り消す。
異なる接続の予約と、再割り当てより新しいバージョンの予約は取り消さない。

### ドメイン型（`backend/provisioning/domain/scheduled_deprovision.go`）

```go
type ScheduledDeprovisionStatus string // scheduled | materialized | cancelled

type ScheduledDeprovision struct {
    ID, TenantID, ConnectionID, UserID string
    SourceVersion int64        // 削除イベントのバージョン。作る配信の source_version になる
    DueAt         time.Time    // 削除時刻 + grace_period_days × 24 時間
    Status        ScheduledDeprovisionStatus
    DeliveryID    *string      // materialized で作った配信
    CreatedAt, UpdatedAt time.Time
}

func NewScheduledDeprovision(id, tenantID, connectionID, userID string, version int64, deletedAt time.Time, graceDays int) *ScheduledDeprovision
func (s ScheduledDeprovision) IsDue(now time.Time) bool // scheduled かつ now >= DueAt
func (s ScheduledDeprovision) Delivery(id string, now time.Time) *ProvisioningDelivery // operation=delete, pending
```

### 永続化の境界（`ports.ProvisioningDeliveryRepository` へ追加）

予約は配信の前段であり、実体化で「予約を materialized にする」と「配信を挿入する」を同時に行う必要がある。
同じリポジトリへ置けば、PostgreSQL では一文の CTE、メモリでは一つのロックで原子性を保てる。
別リポジトリにすると、取消と実体化の競合で、取り消した予約から配信が生まれる窓ができる。

```go
ScheduleDeprovision(ctx, s *domain.ScheduledDeprovision) (created bool, err error)            // 同じ (tenant, connection, user) に scheduled があれば作らない
CancelScheduledDeprovisions(ctx, tenantID, connectionID, userID string, beforeVersion int64, now time.Time) (int, error)
ListDueDeprovisions(ctx, now time.Time, limit int) ([]*domain.ScheduledDeprovision, error)
MaterializeDeprovision(ctx, s *domain.ScheduledDeprovision, d *domain.ProvisioningDelivery) (bool, error) // scheduled のときだけ遷移し配信を挿入する
```

### ユースケース

- `CaptureLifecycleEvent`：`TriggerUserDeleted` が `OperationDelete` へ変換され、`GracePeriodDays > 0` なら配信の代わりに予約を保存する。`TriggerAssignmentAdded` では、対象の Application と User の予約のうち、今回のバージョンより古い scheduled を取り消してから、従来どおり create の配信を作る。
- `DispatchPendingDeliveries`：期限に達した予約を先に配信へ変え、その後に従来どおり未関連付けの配信へジョブを関連付ける。worker は既存の周期処理でこの関数を呼ぶため、新しい配線は増えない。時刻は引数の `now` で受け取る。識別子の生成はユースケースで行う。

## 計画

1. 受け入れ RED を二つの E2E で確認する（期限前に配信が作られる、再割り当て後も DELETE が送られる）。
2. ドメイン型と単体テストを GREEN にする。
3. メモリの配信リポジトリへ予約を追加し、捕捉と取消、ディスパッチャーの実体化を GREEN にする。
4. PostgreSQL のスキーマ、クエリ、リポジトリを追加し、`check-schema` と PostgreSQL のリポジトリテストで確かめる。
5. 内部設計文書とデータベース設計の表を更新する。
6. 変異テスト、対象パッケージ、`mise run verify` を実行する。

## タスク

- [x] T001 [Readiness] 予約状態と期限処理の所有境界を決める。
- [x] T002 [Acceptance] `EX-PROVISIONING-006-01` と `-02` の E2E RED を確認する（`TestE2E_DeletionWithAGracePeriodSendsDELETEOnlyAfterItElapses`、`TestE2E_ReassignmentWithinTheGracePeriodCancelsTheDELETE`）。
- [x] T003 [Domain] 猶予予約の期限判定の単体 RED を GREEN にする（`EX-PROVISIONING-006-01`）。
- [x] T004 [Use Cases] 削除捕捉、再割り当て、期限到来を接続する（`TestDispatchPendingDeliveries_MaterializesAScheduledDeprovisionOnlyOnceDue`、`TestCaptureLifecycleEvent_ReassignmentCancelsOnlyThatConnectionsScheduledDeprovision`）。
- [x] T005 [Adapters] PostgreSQL の予約テーブルとリポジトリを追加する。
- [x] T006 [Docs] 内部設計とデータベース設計を更新する。
- [x] T007 [Verify] 変異、対象パッケージ、`mise run verify` を実行する。

使う検査の手順:

- RED、GREEN、フォールト注入：`mise run test-go-test -- ./backend/provisioning <test>`、`mise run test-go-test -- ./backend/provisioning/usecases <test>`
- 振る舞いの GREEN 後：`mise run test-go-package -- ./backend/provisioning/usecases`、`mise run lint-go`
- パッケージをまたいだ後：`mise run test-go-changed`
- スキーマ：`mise run sqlc-generate`、`mise run check-schema`、`mise run check-schema-tables`

## 検証

- `mise run test-go-package -- ./backend/provisioning/usecases`
- `mise run test-go-package -- ./backend/provisioning`
- `mise run test-go-mutation -- backend/provisioning/usecases`
- `mise run verify`

## リスク

期限前の送出と取消漏れは、外部システムでの誤削除を招く。
期限の直前・直後と再割り当ての競合を、予約の状態遷移と明示時刻を使うテストで固定する。
実体化と取消の競合は、scheduled のときだけ遷移する条件付き更新と配信の挿入を一文で行うことで閉じる。
実体化が失敗した場合は予約が scheduled のまま残り、次の周期で再試行される。

## 完了

- **Completed At**: 2026-09-24
- **Summary**:
  `mise run spec-diff` は main に対する規範仕様の差分なしを報告した。`REQ-PROVISIONING-006` は宣言済みで、この作業は実装をその宣言へ合わせた。
  `on_delete=delete` かつ `grace_period_days` が 1 以上の接続では、User の削除から配信を作らず `ScheduledDeprovision` を保存する。ディスパッチャーが期限に達した予約を delete の配信へ変え、worker が下流へ DELETE を送る。猶予期間中の同じ Application への割り当ては、その接続の古い予約だけを取り消す。
  予約は `provisioning_scheduled_deprovisions` に保存し、実体化は予約の条件付き遷移と配信の挿入を一つの SQL 文で行う。内部設計、データベース設計、被覆負債の一覧（`EX-PROVISIONING-006-01`、`-02` を除去）、リリースノートを更新した。
- **Primary Use Case Evidence**:
  - id: deprovision-after-grace-period
    unit_red: ディスパッチャーが予約を配信へ変えないため、TestDispatchPendingDeliveries_MaterializesAScheduledDeprovisionOnlyOnceDue が、期限ちょうどの周期処理の後に配信が一件もないことを報告して失敗した。
    e2e_red: 捕捉が猶予期間を無視して直ちに delete を配信したため、TestE2E_DeletionWithAGracePeriodSendsDELETEOnlyAfterItElapses が「downstream received DELETE /Users/remote-user-1 before the grace period elapsed」で失敗した。
    unit_fault_injection: IsDue の比較を `!now.Before(DueAt)` から `now.After(DueAt)` へ変えると、期限ちょうどで配信が作られず TestDispatchPendingDeliveries_MaterializesAScheduledDeprovisionOnlyOnceDue が失敗した。
    e2e_fault_injection: DispatchPendingDeliveries から materializeDueDeprovisions の呼び出しを外すと、下流へ DELETE が届かず TestE2E_DeletionWithAGracePeriodSendsDELETEOnlyAfterItElapses が失敗した。
  - id: cancel-deprovision-on-reassignment
    unit_red: 削除が予約ではなく配信を作ったため、TestCaptureLifecycleEvent_ReassignmentCancelsOnlyThatConnectionsScheduledDeprovision が、削除の直後に app-1 へ配信が作られたことを報告して失敗した。
    e2e_red: 再割り当ての後も削除が配信されたため、TestE2E_ReassignmentWithinTheGracePeriodCancelsTheDELETE が「downstream received DELETE /Users/remote-user-1 after the reassignment cancelled it」で失敗した。
    unit_fault_injection: メモリ実装の取消から接続の条件を外すと、app-2 の予約まで取り消され TestCaptureLifecycleEvent_ReassignmentCancelsOnlyThatConnectionsScheduledDeprovision が失敗した。
    e2e_fault_injection: 捕捉の取消の分岐を `if false` にすると、期限後に DELETE が送られ TestE2E_ReassignmentWithinTheGracePeriodCancelsTheDELETE が失敗した。
- **Change-Resistance Results**:
  `mise run test-go-mutation -- backend/provisioning/usecases` の初回は、変更行に三つの生存変異を残した。取消のエラー検査の反転（割り当ての create 配信が作られなくなる）と、実体化のエラー検査の反転（最初の予約だけを実体化して戻る）は不足であり、再割り当て後の create 配信と、二件の予約を一度に実体化することの表明を加えて検出した。
  再実行で残った capture.go の `Save` のエラー検査の反転は、既存の行であり接続が一つのテストでは等価に振る舞う。この作業の範囲外として残す。
  PostgreSQL 実装は、実 DB 上で一意の有効予約、期限の境界、二度目の実体化の拒否、取消済み予約から配信が作られないことを TestProvisioningDeliveryRepository_ScheduledDeprovisionLifecycle と TestProvisioningDeliveryRepository_CancelledDeprovisionIsNeverMaterialized で確かめた。
- **Verification Results**:
  - `mise run verify` - 成功
  - `mise run lint-go` - 成功
  - `mise run test-go-changed` - 成功
  - `mise run check-schema-tables` - 成功
  - `mise run check-schema` - 実行不可。この環境の Docker CLI が `docker compose` プラグインを見つけられないため（`~/.docker/cli-plugins/docker-compose` のリンク先が存在しない）。スキーマは embedded-postgres のテストで適用して確かめた。
  - `mise run test-ui-e2e` - 対象外。変更はブラウザーへ届かない。
