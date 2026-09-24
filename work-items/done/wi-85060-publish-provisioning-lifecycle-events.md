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
  reason: 監査へ Provisioning のライフサイクルイベントが記録されるようになり、割り当て解除による無効化と必須属性の欠落時の振る舞いも変わるため、運用者へ知らせる必要がある。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-85060-publish-provisioning-lifecycle-events.md }
initial_context:
  specification:
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-002
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-003
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-005
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-010
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-011
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-014
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-018
    - docs/domain/provisioning/states.md
  typespec:
    - IdMagic.Contract.ProvisioningConnectionRegistered
    - IdMagic.Contract.UserProvisioned
    - IdMagic.Contract.ConnectionQuarantined
  source:
    - backend/provisioning/domain/events.go
    - backend/provisioning/domain/delivery.go
    - backend/provisioning/usecases/admin.go
    - backend/provisioning/usecases/deliver.go
    - backend/provisioning/usecases/dispatcher.go
    - backend/provisioning/usecases/job_handler.go
    - backend/provisioning/client_scim/mapping.go
    - backend/provisioning/db_memory/repositories.go
    - backend/provisioning/db_postgres/provisioning.sql
    - backend/provisioning/handlers_http/routes.go
    - backend/provisioning/module.go
    - backend/cmd/idmagic-worker/worker.go
    - backend/cmd/internal/bootstrap/audit_event_record.go
    - backend/jobs/usecases/runner.go
  tests:
    - backend/provisioning/e2e_capture_delivery_test.go
    - backend/provisioning/handlers_http/admin_connection_handler_test.go
    - backend/provisioning/usecases/deliver_test.go
    - backend/provisioning/usecases/job_handler_test.go
    - backend/provisioning/usecases/dispatcher_test.go
    - backend/provisioning/domain/events_test.go
  stop_before_reading:
    - frontend
    - backend/provisioning/source_idmanagement
affected_spec:
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-002 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-003 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-004 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-005 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-007 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-008 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-009 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-010 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-011 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-014 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-017 }
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-018 }
primary_use_cases:
  - id: connection-admin-events
    requirement: REQ-PROVISIONING-002
    observable_result: 管理 API で接続を登録、資格情報を更新、隔離を解除すると、それぞれ ProvisioningConnectionRegistered、ProvisioningCredentialRotated、ProvisioningConnectionQuarantineCleared が一度ずつ、TypeSpec と同じ camelCase の項目名で発行される。拒否された操作では何も発行されない。
    unit_test: { path: backend/provisioning/usecases/admin_events_test.go, name: TestAdminConnectionOperationsEmitTheirEventAfterSaving, task: test-go-race }
    e2e_test: { path: backend/provisioning/handlers_http/admin_connection_events_test.go, name: TestAdminConnectionAPIEmitsLifecycleEvents, task: test-go-race }
    unit_fault_model: 保存に失敗したのにイベントを発行する、または資格情報を渡さない更新でも ProvisioningCredentialRotated を発行する。
    e2e_fault_model: HTTP アダプターが support.Deps.Emit をユースケースへ渡さず、管理操作のイベントが監査へ届かない。
  - id: delivery-lifecycle-events
    requirement: REQ-PROVISIONING-003
    observable_result: ディスパッチャーが配信にジョブを関連付けると配信が in_flight になって ProvisioningDeliveryStarted が発行され、ジョブの実行で下流への反映が成功すると配信が succeeded になって UserProvisioned が発行される。同じ配信のジョブが再実行されても下流へ再送せず、イベントも増えない。
    unit_test: { path: backend/provisioning/usecases/job_handler_test.go, name: TestProvisioningDeliveryHandler_EmitsTheTransitionEventOnceAfterSucceeding, task: test-go-race }
    e2e_test: { path: backend/provisioning/e2e_lifecycle_events_test.go, name: TestE2E_DispatchAndDeliveryEmitStartedThenProvisioned, task: test-go-race }
    unit_fault_model: 状態を succeeded にする前にイベントを発行する、または終端状態の配信を再実行して同じイベントを二度発行する。
    e2e_fault_model: Module がディスパッチャーとジョブハンドラーへ発行ポートを渡さないか、関連付けで配信が pending のまま残る。
  - id: unassignment-deprovisions-downstream
    requirement: REQ-PROVISIONING-005
    observable_result: 有効なままの User の割り当てを解除した deactivate 配信は、下流へ active=false を送り、UserDeprovisioned（action=deactivate）を発行する。
    unit_test: { path: backend/provisioning/usecases/deliver_test.go, name: TestExecuteDelivery_DeactivateSendsInactiveEvenWhenTheUserIsActive, task: test-go-race }
    e2e_test: { path: backend/provisioning/e2e_lifecycle_events_test.go, name: TestE2E_UnassignmentSendsActiveFalseAndEmitsDeprovisioned, task: test-go-race }
    unit_fault_model: deactivate 配信が User の現在の active 属性をそのまま送る。
    e2e_fault_model: 割り当て解除から作られた配信が下流へ active=true を送り、UserDeprovisioned だけが残る。
  - id: dead-letter-and-quarantine-events
    requirement: REQ-PROVISIONING-010
    observable_result: 最後の試行が失敗すると配信が dead_letter になって UserProvisioningFailed が一度発行され、その失敗で連続失敗数が閾値に達すると接続が隔離されて ConnectionQuarantined が一度発行される。
    unit_test: { path: backend/provisioning/usecases/job_handler_test.go, name: TestProvisioningDeliveryHandler_TerminalFailureEmitsFailedThenQuarantinedOnce, task: test-go-race }
    e2e_test: { path: backend/provisioning/e2e_lifecycle_events_test.go, name: TestE2E_ExhaustedDeliveryEmitsFailedAndQuarantined, task: test-go-race }
    unit_fault_model: 非終端の失敗でも UserProvisioningFailed を発行する、または隔離済みの接続で ConnectionQuarantined を再発行する。
    e2e_fault_model: 下流が失敗し続けても、監査に配信の失敗と接続の隔離が残らない。
  - id: required-mapping-fails-closed
    requirement: REQ-PROVISIONING-018
    observable_result: 必須の属性マッピングを解決できない配信は、下流へ何も送らず、試行回数が残っていても dead_letter になって UserProvisioningFailed を発行する。
    unit_test: { path: backend/provisioning/usecases/job_handler_test.go, name: TestProvisioningDeliveryHandler_UnresolvedRequiredMappingDeadLettersOnFirstAttempt, task: test-go-race }
    e2e_test: { path: backend/provisioning/e2e_lifecycle_events_test.go, name: TestE2E_MissingRequiredAttributeFailsClosedWithoutRetry, task: test-go-race }
    unit_fault_model: 必須属性の解決失敗を一時的な失敗として扱い、max_attempts まで再試行する。
    e2e_fault_model: SCIM クライアントの解決失敗が型付きのエラーとして伝わらず、ジョブハンドラーが終端と判定できない。
---

# 宣言済みの Provisioning ライフサイクルイベントを状態遷移とともに発行する

## 動機

Provisioning のシナリオは接続登録、配信開始、成功、失敗、隔離、隔離解除、資格情報ローテーションのイベントを観測結果として宣言する。
イベント型は `domain/events.go` に存在するが、本番のユースケース、ハンドラー、アダプターはそれらを発行していない。
型の存在だけでは監査や購読側は状態遷移を観測できない。

準備段階で、イベントを事実どおりに発行する前提となる次の欠陥も見つかった。

- イベント型に JSON タグが無く、ワイヤ表現の項目名が TypeSpec の `tenantId` などではなく `TenantID` になる。監査記録はテナント ID を `tenantId` から取り出すため、テナントの絞り込みからも漏れる。
- `AttachJob` は `job_id` を設定するだけで、配信を `in_flight` へ遷移させない（`states.md` は `pending → in_flight` を `ProvisioningDeliveryStarted` の遷移と宣言する）。
- `deactivate` 配信は User の現在の `active` 属性をそのまま送る。有効なままの User の割り当て解除（`REQ-PROVISIONING-005`）では下流へ `active=true` が届き、無効化されない。
- 必須の属性マッピングを解決できない配信は、一時的な失敗と同じく `max_attempts` まで再試行する（`REQ-PROVISIONING-018` はフェイルクローズを求める）。

これらを残したまま発行すると、状態と合わないイベントが監査に残る。そのため、本件で併せて直す。

## 対象範囲

- 規範が宣言する Provisioning ライフサイクルイベントを、対応する状態の保存の後に発行する。
  - 管理操作: `ProvisioningConnectionRegistered`、`ProvisioningCredentialRotated`、`ProvisioningConnectionQuarantineCleared`
  - 配信: `ProvisioningDeliveryStarted`、`UserProvisioned`、`UserDeprovisioned`、`GroupPushed`、`UserProvisioningFailed`
  - 接続の健全性: `ConnectionQuarantined`
- イベントのワイヤ表現を TypeSpec の項目名へ合わせる。
- ジョブの関連付けで配信を `in_flight` にする。
- `deactivate` 配信で `active=false` を送り、下流に未作成の User への `deactivate` は何も送らずに成功とする。
- 必須属性の解決失敗を終端の失敗として扱う。
- 終端状態（`succeeded`、`dead_letter`）の配信のジョブが再実行されても、下流へ再送せずイベントも発行しない。

## 対象外

- `FullResyncCompleted`。これは resync 追跡を要するため [[wi-22987-track-full-resync-completion]] が扱う。
- 新しいイベント種類や外部イベントブローカー製品の導入。
- シナリオに書かれていないイベントの公開。`ProvisioningConnectionUpdated`、`ProvisioningConnectionDisabled`、`ProvisioningConnectionDeleted` はシナリオが観測結果として宣言していないため発行しない。`GroupMembershipPushed` は `membership_add` / `membership_remove` の配信を作る経路が存在しないため発行しない。Group の配信は `states.md` の遷移に従い `GroupPushed` を発行する。
- 誤削除ガードの閾値判定（`REQ-PROVISIONING-011` の前半）。full resync が deprovision の件数を数えて隔離する仕組みは実装されておらず、本件はイベントの発行を扱う。隔離は既存の連続失敗による経路で発行する。仕組み自体は [[wi-61629-quarantine-connections-on-the-accidental-deletion-guard]] が扱う。
- Jobs の再試行間隔を `Retry-After` に従わせること（`REQ-PROVISIONING-009` の前半）。`RetryableError.RetryAfter` を Jobs の `NextRetryRunAt` へ渡す口が Jobs に無く、Jobs 側の変更を要する。本件は再試行中も `in_flight` のまま保たれ、回復後に `UserProvisioned` が発行されることを扱う。

## 設計

イベントは状態遷移の副産物ではなく、外部から観測できる結果である。
各ユースケースが任意の通知コールバックを直接持つ設計は、配線の欠落と二重発行を増やす。
そこで発行の入口を管理側と worker 側に一つずつ置き、それぞれ状態の保存が成功した後にだけ呼ぶ。

### 型と操作

- 発行ポートは他の Context と同じ `func(spec.DomainEvent)` とする。本番では `bootstrap.Dependencies.NewEmitFunc` が EventSink と監査へ書く。
- `usecases.AdminDeps.Emit func(spec.DomainEvent)`。HTTP アダプターは `support.Deps.Emit` を渡す。
- `usecases.DispatcherDeps.Emit func(spec.DomainEvent)` と `usecases.JobHandlerDeps.Emit func(spec.DomainEvent)`。`Module.DispatcherDeps(jobRepo, quotaRepo, emit)` と `Module.JobHandlerDeps(attrSource, memberSource, newTargetClient, emit)` の引数にして、配線の欠落をコンパイルで検出する。
- `ExecuteDelivery(ctx, deps, tenantID, deliveryID, now) (spec.DomainEvent, error)`。配信を `succeeded` に保存した後、その遷移を表すイベントを返す。下流へ何も送らなかった場合（下流に無い User の削除や無効化、発火元の消失、終端状態の配信の再実行）は `nil` を返す。発行はジョブハンドラーが行う。
- `ports.ErrRequiredAttributeUnresolved`。SCIM クライアントの `BuildResource` が必須属性を解決できないときに包んで返す。ジョブハンドラーは `errors.Is` でこれを見分け、試行回数を問わず終端とする。
- ドメインイベントの構造体には TypeSpec と同じ camelCase の JSON タグを付け、`At` は `json:"-"` とする（`MarshalDomainEvent` が `occurredAt` を付ける）。

### 呼び出しの順序

| 操作 | 保存 | 発行 |
| --- | --- | --- |
| `RegisterConnection` | `ConnectionRepo.Register` | `ProvisioningConnectionRegistered` |
| `UpdateConnection`（資格情報あり） | `ConnectionRepo.Update` | `ProvisioningCredentialRotated`（新しい `credentialId`） |
| `ResumeConnection` | `ConnectionRepo.Update` | `ProvisioningConnectionQuarantineCleared` |
| `DispatchPendingDeliveries` | `AttachJob`（`pending → in_flight`、`job_id` の設定） | 関連付けに成功した配信だけ `ProvisioningDeliveryStarted` |
| ジョブの成功 | `UpdateStatus(succeeded)`、連続失敗数の解除 | `ExecuteDelivery` が返したイベント |
| ジョブの終端の失敗 | `UpdateStatus(dead_letter)` | `UserProvisioningFailed` |
| 連続失敗が閾値に到達 | 接続の `Quarantine` と `Update` | 未隔離から隔離へ変わったときだけ `ConnectionQuarantined` |

成功時の発行を連続失敗数の解除の後に置くのは、解除に失敗したジョブを Jobs が再試行し、その再試行で発行させるためである。
終端状態の配信は `ExecuteDelivery` が下流を呼ばずに `nil` を返すので、再試行でもイベントは一度だけになる。

イベントの発行は保存と同じトランザクションにはしない。
既存の `NewEmitFunc` は発行の失敗をログへ落とす設計であり、Provisioning だけ別の保証を持たせる理由は無い。
横断的なトランザクションは使わず、保存の成功を発行の前提にすることで「イベントだけが残る」誤りを防ぐ。

### 採用しない代替案

- `DeliverDeps.Emit` を追加し、`ExecuteDelivery` の内部で発行する案。worker 側の発行口が `DeliverDeps` と `JobHandlerDeps` の二つになり、片方だけ配線する誤りが起き得る。
- 必須属性の解決失敗を `ExecuteDelivery` で `dead_letter` にする案。終端の判定と連続失敗の扱いがジョブハンドラーと二か所に分かれる。
- 必須属性の解決失敗を連続失敗として数える案。これは User の属性の欠落であり、下流の健全性を表さない。1 人の属性の欠落で接続が隔離されないよう、数えない。

## 計画

1. イベントのワイヤ表現を TypeSpec に合わせる（単体 RED → GREEN）。
2. 管理操作のイベントを単体 RED、HTTP 経由の E2E RED から実装する。
3. `AttachJob` の `in_flight` 遷移と `ProvisioningDeliveryStarted` を、memory と PostgreSQL の両方で実装する。
4. `ExecuteDelivery` の遷移イベント、`deactivate` の `active=false`、終端状態の再実行の抑止を実装する。
5. ジョブハンドラーの発行、フェイルクローズ、隔離の発行を実装する。
6. worker と Module の配線を変え、E2E を GREEN にする。
7. 変異テストと手作業の故障注入、`mise run verify` を行う。

## タスク

- [x] T001 [Readiness] 既存イベント基盤と Provisioning の状態遷移を対応付ける。
- [x] T002 [Acceptance] 登録、配信成功・失敗、隔離、資格情報ローテーションのイベント RED を確認する。
- [x] T003 [Domain] イベントペイロードと一度だけの発行条件を単体 RED から実装する。
- [x] T004 [Use Cases] 接続管理、配信、ジョブ処理を発行ポートへ接続する。
- [x] T005 [Verify] 故障注入、変異、対象パッケージ、`mise run verify` を実行する。

各 RED、GREEN、故障注入では `mise run test-go-test -- <package> <test>` を使う。
一つの振る舞いが GREEN になったら `mise run test-go-package -- <package>` と `mise run lint-go` を実行し、パッケージをまたいだ後は `mise run test-go-changed` を実行する。

## 検証

- `mise run test-go-package -- ./backend/provisioning/usecases`
- `mise run test-go-package -- ./backend/provisioning/handlers_http`
- `mise run test-go-package -- ./backend/provisioning`
- `mise run test-go-mutation -- backend/provisioning/usecases`
- `mise run verify`

## リスク

状態が変わったのにイベントが無い、またはイベントだけが残ると、監査と再試行処理が実際の状態を誤認する。
状態読み戻しと発行記録を対にして検証し、発行の削除・二重化・順序変更を故障として確認する。
`deactivate` 配信の変更により、これまで割り当て解除後も下流で有効のままだったアカウントが、次の配信で無効化される。

## 完了

- **Completed At**: 2026-09-24
- **Summary**:
  `mise run spec-diff` は main に対する規範の差分が無いと報告した。規範は変えず、既に宣言されていた振る舞いへ実装を合わせた。
  管理 API の接続の登録、資格情報の更新、隔離の解除は、保存の後に `ProvisioningConnectionRegistered`、`ProvisioningCredentialRotated`、`ProvisioningConnectionQuarantineCleared` を発行する。
  ディスパッチャーはジョブの関連付けで配信を `in_flight` にし、関連付けに成功した worker だけが `ProvisioningDeliveryStarted` を発行する。
  `ExecuteDelivery` は `succeeded` の保存後に遷移イベント（`UserProvisioned`、`UserDeprovisioned`、`GroupPushed`）を返し、ジョブハンドラーが連続失敗数の解除の後に発行する。終端状態の配信は再実行されても下流へ送らない。
  最後の試行の失敗は `dead_letter` の保存後に `UserProvisioningFailed` を、隔離へ移った失敗は `ConnectionQuarantined` を一度だけ発行する。
  `deactivate` 配信は `active=false` を送り、下流に無い User は作らない。必須属性の解決失敗は `ports.ErrRequiredAttributeUnresolved` として伝わり、最初の試行で `dead_letter` になる。これは連続失敗に数えない。
  イベントの JSON 項目名を TypeSpec に合わせた。発行ポートは `Module.DispatcherDeps` と `Module.JobHandlerDeps` の引数にして、worker は `NewEmitFunc` を渡す。
  `EX-PROVISIONING-003-01`、`004-01`、`005-01`、`010-01`、`014-01`、`017-01`、`018-01` を被覆負債から外した。`EX-PROVISIONING-011-01` は閾値判定が未実装のため、[[wi-61629-quarantine-connections-on-the-accidental-deletion-guard]] を起票して負債の担当を付け替えた。
- **Primary Use Case Evidence**:
  - id: connection-admin-events
    unit_red: 発行ポートの `AdminDeps.Emit` を追加しただけの状態で、`TestAdminConnectionOperationsEmitTheirEventAfterSaving` が `events = [], want exactly one` で失敗した。
    e2e_red: テスト `TestAdminConnectionAPIEmitsLifecycleEvents` が `events = 0, want 3` で失敗した。HTTP アダプターが `support.Deps.Emit` をユースケースへ渡していなかった。
    unit_fault_injection: 登録のイベントを `ConnectionRepo.Register` の前へ移す故障と、資格情報の有無を問わずローテーションを発行する故障で、`TestAdminConnectionOperationsEmitTheirEventAfterSaving` が失敗した。
    e2e_fault_injection: アダプターの `adminDeps()` が `Emit` を渡さない状態が上の E2E RED であり、同じテストが失敗した。
  - id: delivery-lifecycle-events
    unit_red: テスト `TestProvisioningDeliveryHandler_EmitsTheTransitionEventOnceAfterSucceeding` が `emitted = [], createCalls = 1` で、`TestDispatchPendingDeliveries_StartsTheDeliveryAndEmitsStartedAfterAttaching` が `status = pending, event = <nil>` で失敗した。
    e2e_red: E2E はユースケースの実装の後に書いた。代わりに、`Module.JobHandlerDeps` から `Emit` を外す、`Module.DispatcherDeps` から `Emit` を外す、memory の `AttachJob` が `in_flight` にしない、の各故障で `TestE2E_DispatchAndDeliveryEmitStartedThenProvisioned` が失敗することを確認した。
    unit_fault_injection: 配信処理 `ExecuteDelivery` の終端状態のガードを外すと `TestProvisioningDeliveryHandler_EmitsTheTransitionEventOnceAfterSucceeding` が、関連付けの成否を問わず発行すると `TestDispatchPendingDeliveries_EmitsNothingWhenAnotherWorkerAttachedFirst` が失敗した。
    e2e_fault_injection: 上の 3 つの配線と遷移の故障で `TestE2E_DispatchAndDeliveryEmitStartedThenProvisioned` が失敗した。
  - id: unassignment-deprovisions-downstream
    unit_red: テスト `TestExecuteDelivery_DeactivateSendsInactiveEvenWhenTheUserIsActive` が `updateCalls = 1, active sent = true` で失敗した。
    e2e_red: E2E はユースケースの実装の後に書いた。`deactivate` で `active=false` を設定する行を外すと、`TestE2E_UnassignmentSendsActiveFalseAndEmitsDeprovisioned` が失敗した。
    unit_fault_injection: 実装前の状態が「User の active をそのまま送る」故障そのものであり、上の単体 RED で検出した。下流で消えた User の無効化で再作成しない分岐を外すと、`TestExecuteDelivery_DeactivateOfAUserGoneDownstreamDoesNotRecreateIt` が失敗した。
    e2e_fault_injection: 同じ行を外す故障と、`Module.JobHandlerDeps` から `Emit` を外す故障で、`TestE2E_UnassignmentSendsActiveFalseAndEmitsDeprovisioned` が失敗した。
  - id: dead-letter-and-quarantine-events
    unit_red: テスト `TestProvisioningDeliveryHandler_TerminalFailureEmitsFailedThenQuarantinedOnce` が `emitted = [], want [UserProvisioningFailed UserProvisioningFailed ConnectionQuarantined UserProvisioningFailed]` で失敗した。
    e2e_red: E2E はユースケースの実装の後に書いた。`Module.JobHandlerDeps` から `Emit` を外すと、`TestE2E_ExhaustedDeliveryEmitsFailedAndQuarantined` が失敗した。
    unit_fault_injection: 失敗イベント `UserProvisioningFailed` の発行を `dead_letter` の保存の前へ移す故障と、隔離済みの接続でも失敗ごとに `ConnectionQuarantined` を発行する故障で、同じ単体テストが失敗した。
    e2e_fault_injection: 上の配線の故障で `TestE2E_ExhaustedDeliveryEmitsFailedAndQuarantined` が失敗した。
  - id: required-mapping-fails-closed
    unit_red: テスト `TestProvisioningDeliveryHandler_UnresolvedRequiredMappingDeadLettersOnFirstAttempt` が、ハンドラーが必須属性の解決失敗のエラーを Jobs へ返して失敗した。`TestBuildResource_RequiredMissingFailsClosed` も、エラーが `ports.ErrRequiredAttributeUnresolved` を包んでいないため失敗した。
    e2e_red: E2E はユースケースの実装の後に書いた。SCIM クライアントのエラーを `%w` ではなく `%v` で包むと、`TestE2E_MissingRequiredAttributeFailsClosedWithoutRetry` が失敗した。
    unit_fault_injection: フェイルクローズの失敗を連続失敗として数える故障で、`TestProvisioningDeliveryHandler_UnresolvedRequiredMappingDeadLettersOnFirstAttempt` が失敗した。
    e2e_fault_injection: 上の `%v` の故障と、`Module.JobHandlerDeps` から `Emit` を外す故障で、`TestE2E_MissingRequiredAttributeFailsClosedWithoutRetry` が失敗した。
- **Change-Resistance Results**:
  `mise run test-go-mutation -- backend/provisioning/usecases` は 172 件の変異を見つけ、142 件を試して 127 件を検出し、15 件が生き残った（30 件は被覆外）。
  生き残った変異のうち、本件で変更した関数にあるのは `job_handler.go` 64 行目の `now == nil` の反転だけである。単体テストは常に `Now` を渡すため既定の時計の分岐を通らず、観測上は等価である。残る 14 件は本件で変更していない行（`UpdateConnection` の他の項目、`validateCredential`、`StartFullResync`、`RetryDelivery`、`credentialMetadata`、`CaptureLifecycleEvent`、`adoptExistingUser`、`pushGroupMembers`）にある。
  変異器が表現できない故障は、上の各主要ユースケースに記録したとおり手で注入し、すべて検出した。
  イベントのワイヤ表現は、タグを付ける前に `TestProvisioningEvents_MarshalWithTheContractFieldNames` が全イベントの項目名の不一致で失敗することを確認した。
  検出できない故障として、ジョブの成功時に遷移イベントを連続失敗数の解除より前に発行する並べ替えがある。どちらの順序でも配信の状態は保存済みであり、違いは解除に失敗した場合の再試行にしか現れない。
- **Verification Results**:
  - `mise run verify` - 成功
  - `mise run test-go-changed` - 成功
  - `mise run lint-go` - 成功
  - `mise run test-ui-e2e` - 実行していない。変更は worker の配信処理と監査へのイベントに限られ、ブラウザーが受け取る応答の形は変えていない。
