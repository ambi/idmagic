---
status: cancelled
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-23
priority: p2
depends_on: []
change_kind: bugfix
evidence_policy: risk-based-v3
initial_context:
  specification:
    - docs/domain/scenarios.feature.md#REQ-PLATFORM-003
    - docs/domain/provisioning/internals.md
    - docs/design/application/design-guidelines.md
  typespec: []
  source:
    - backend/provisioning/ports/capture.go
    - backend/provisioning/usecases/capture.go
    - backend/provisioning/usecases/notify_adapters.go
    - backend/provisioning/module.go
    - backend/idmanagement/user/ports/user_mutation.go
    - backend/idmanagement/user/ports/provisioning_notify.go
    - backend/idmanagement/user/usecases/admin_users.go
    - backend/idgovernance/usecases/user_mutation_committer.go
    - backend/idgovernance/db_postgres/lifecycle_workflow_capture.go
    - backend/idgovernance/db_memory/lifecycle_workflow_capture.go
    - backend/application/usecases/assignments.go
    - backend/application/ports/provisioning_notify.go
    - backend/application/module.go
    - backend/shared/storage/db_postgres/base.go
    - backend/cmd/internal/bootstrap/memory.go
    - backend/cmd/internal/bootstrap/postgres.go
  tests:
    - backend/provisioning/e2e_capture_delivery_test.go
    - backend/idmanagement/user/handlers_http/admin_user_handler_test.go
    - backend/application/handlers_http/application_handler_test.go
    - tools/check/example-coverage-debt.json
  stop_before_reading:
    - frontend
    - backend/idmanagement/group
    - backend/provisioning/client_scim
affected_spec:
  - { path: docs/domain/scenarios.feature.md, requirement: REQ-PLATFORM-003 }
primary_use_cases:
  - id: user-mutation-and-delivery-commit-together
    requirement: REQ-PLATFORM-003
    observable_result: 有効な接続と割り当てがある User を管理 API で作成、無効化、削除すると、変更と同じトランザクションで ProvisioningDelivery が作られて配信が succeeded になる。配信の捕捉が失敗すると要求は失敗し、User は変更前のまま残り、配信行も残らない。
    unit_test: { path: backend/idmanagement/user/usecases/admin_users_transaction_test.go, name: TestSetUserDisabledLeavesUserUnchangedWhenCaptureFails, task: test-go-race }
    e2e_test: { path: backend/provisioning/transactional_capture_e2e_test.go, name: TestAdminUserMutationRollsBackWithItsDelivery, task: test-go-race }
    unit_fault_model: 捕捉を User の保存と別の作用範囲で呼ぶか、捕捉の失敗をログに落として変更を確定させる。
    e2e_fault_model: 組み立てがトランザクションの境界を User の変更経路へ渡さず、捕捉の失敗後も User の変更が保存先に残る。
  - id: unassignment-and-delivery-commit-together
    requirement: REQ-PLATFORM-003
    observable_result: 有効な接続がある Application から User の割り当てを管理 API で解除すると、同じトランザクションで ProvisioningDelivery が作られる。配信の捕捉が失敗すると要求は失敗し、割り当ては残り、配信行も残らない。
    unit_test: { path: backend/application/usecases/assignments_transaction_test.go, name: TestUnassignApplicationKeepsAssignmentWhenCaptureFails, task: test-go-race }
    e2e_test: { path: backend/provisioning/transactional_capture_e2e_test.go, name: TestAdminUnassignmentRollsBackWithItsDelivery, task: test-go-race }
    unit_fault_model: 割り当ての削除と捕捉を別の作用範囲で行うか、捕捉の失敗を握りつぶす。
    e2e_fault_model: Application の組み立てがトランザクションの境界を割り当て経路へ渡さない。
  - id: storage-scope-rolls-back
    requirement: REQ-PLATFORM-003
    observable_result: トランザクションの作用範囲の中で保存した行は、作用範囲が失敗すると memory と PostgreSQL のどちらの保存先からも読めない。
    unit_test: { path: backend/shared/storage/db_memory/transaction_test.go, name: TestTransactorRestoresParticipantsWhenScopeFails, task: test-go-race }
    e2e_test: { path: backend/shared/storage/db_postgres/transaction_test.go, name: TestTransactorRollsBackRepositoryWritesWhenScopeFails, task: test-go-race }
    unit_fault_model: memory の作用範囲が失敗しても参加する保存先の行を書き戻さない。
    e2e_fault_model: 保存先が context のトランザクションを使わずプールへ直接書き、作用範囲の失敗後も行が残る。
---

# Provisioning の配信行を、発火元の変更と同じトランザクションで作る

## 動機

`REQ-PLATFORM-003` は「配信行は発火元の変更と同時にコミットまたはロールバックする」と定める。
`EX-PLATFORM-003-01` は変更と同じトランザクションで `ProvisioningDelivery` が作られることを、`EX-PLATFORM-003-02` は変更がロールバックすれば `ProvisioningDelivery` も作られないことを求める。

現行の実装はこの保証を満たさない。

| 発火元 | 現行の呼び出し | 食い違い |
| --- | --- | --- |
| User の作成、無効化、削除 | `backend/idmanagement/user/usecases/admin_users.go` の `notifyProvisioning` が、User のコミット後に `CaptureLifecycleEvent` を呼ぶ | capture の失敗はログに残るだけで、User の変更は確定したままになる |
| Application の割り当て解除 | `backend/application/usecases/assignments.go` の `notifyProvisioning` が、割り当てのコミット後に呼ぶ | 同上 |

`backend/provisioning/ports/capture.go` は、2 つのコミットの間でプロセスが落ちると capture が失われ回復できないことを、残余のギャップとして記録している。
記録の正が変わったのに配信が作られない状態は、この規則が違反と名指す状態そのものである。

[[wi-557-back-cross-context-examples-with-tests]] は、この 2 件の具体例をテストで裏付けられず、被覆台帳の当該行に本項目を `blocked_by` として残した。

## 対象範囲

- User の変更（作成、無効化、削除）が作る `ProvisioningDelivery` を、User の保存と同じトランザクションで確定させる。
- Application の割り当て解除が作る `ProvisioningDelivery` を、割り当ての保存と同じトランザクションで確定させる。
- memory と Postgres の両方の組み立てで、同じ原子性を観測できるようにする。
- `EX-PLATFORM-003-01` と `EX-PLATFORM-003-02` を、管理 API を入口とするテストで裏付け、`tools/check/example-coverage-debt.json` から外す。

## 対象外

- 配信の実行（`worker` による下流への反映）の信頼性。既存の Jobs の再試行が担う。
- Group の変更が作る配信。規則の具体例は User と割り当てだけを名指している。必要なら別の work item で扱う。
- 規範（シナリオと具体例）の変更。

## 設計

既存の形として、IdGovernance の `UserMutationCommitter` が `igports.UserWorkflowCapture.SaveUserAndRuns` を通じて、User の保存と派生する LifecycleWorkflow の run を 1 つのトランザクションで確定している。
IdManagement は `userports.UserMutationCommitter` という境界 port だけを知り、IdGovernance の型に依存しない。

採用するのは、トランザクションの作用範囲を context で運ぶ形である。
発火元のユースケースが作用範囲を開き、その中で変更の保存と配信の捕捉を呼ぶ。
各 Context の保存先は作用範囲を知らずに書き、組み立てが渡した共有の保存先が作用範囲へ参加させる。
IdManagement、Application、IdGovernance、Provisioning は互いの型を import しない。

### 型と操作

| 置き場所 | 型または操作 | 役割 |
| --- | --- | --- |
| `backend/idmanagement/user/ports` | `Transactor`: `InTransaction(ctx, fn func(context.Context) error) error` | User の変更と配信の捕捉を 1 つの作用範囲で確定させる境界。`nil` のときは作用範囲を開かずに `fn` を呼ぶ |
| `backend/application/ports` | `Transactor`（同じ形） | 割り当ての保存と配信の捕捉を 1 つの作用範囲で確定させる境界 |
| `backend/shared/storage/db_postgres` | `TransactionScopedDB`（`DB` を実装） | context にトランザクションがあれば `Query`、`QueryRow`、`Exec`、`Begin` をそこへ流し、なければ下位の `DB` へ流す。`Begin` はトランザクション内ではセーブポイントになる |
| `backend/shared/storage/db_postgres` | `Transactor{DB DB}` | `BEGIN` し、トランザクションを context へ入れて `fn` を呼び、成功ならコミット、失敗ならロールバックする。すでに作用範囲の中なら `fn` をそのまま呼ぶ |
| `backend/shared/storage/db_memory` | `Snapshotter`: `Snapshot() (restore func())` | memory の保存先が行の集合を写し取り、書き戻す関数を返す |
| `backend/shared/storage/db_memory` | `NewTransactor(participants ...Snapshotter) *Transactor` | 作用範囲を直列化し、開始時に参加する保存先を写し取り、`fn` が失敗したら書き戻す |

### 呼び出しの順序

| 発火元 | 作用範囲の中 | 作用範囲の外（従来どおり） |
| --- | --- | --- |
| User の作成、属性変更、有効化、無効化、削除予約 | `captureUserMutation`（IdGovernance の run を含む）、配信の捕捉 | 動的 Group の同期、パスワード履歴、監査イベント |
| User の削除（匿名化） | 匿名化した User の保存、配信の捕捉 | カスケード、クォータ、監査イベント |
| 割り当ての追加、解除 | 割り当ての保存または削除、配信の捕捉 | 監査イベント |

配信の捕捉は、作用範囲の中で失敗をそのまま返す。
ログに落として成功を返す best-effort の扱いは廃止する。

### 計画の問いへの回答

1. IdGovernance の capture と Provisioning の capture は、context で運ぶ 1 つのトランザクションへ合成する。単一の committer に両方を持たせる案は、IdGovernance か composition root のどちらかに Provisioning の保存形式を持ち込む。PostgreSQL の `UserWorkflowCapture` は `Begin` をそのまま呼び続け、作用範囲の中ではセーブポイントになる。
2. Application の割り当てにも同じ形の `Transactor` を設ける。発火元の Context が境界の型を所有する既存の流儀（`ProvisioningNotifier` が User と Application で別々に宣言されている）に合わせる。
3. memory の「ロールバック」は、参加する保存先の行の集合を開始時に写し取り、失敗時に書き戻して観測可能にする。参加するのは User、LifecycleWorkflow の run、`ProvisioningDelivery`、割り当ての 4 つの保存先である。作用範囲どうしは直列化する。作用範囲の外から同時に書かれた行は、失敗時の書き戻しで失われ得る。memory の組み立ては開発とテストにだけ使うので、この制約を受け入れる。

採用しない代替案:

- **捕捉を「計画」と「保存」に分け、単一の committer が User、run、配信行をまとめて保存する。** PostgreSQL で 3 つの Context の書き込みを 1 つの `pgx.Tx` へ載せるには、結局どこかでトランザクションを受け渡す必要があり、その受け渡しを持つ Context が他の Context の保存形式を知ることになる。

- **コミット後の best-effort 呼び出しを再試行で補う。** 再試行しても、2 つのコミットの間の障害で capture が失われる状態は残る。規則は「同時に」を求めている。
- **outbox から非同期に配信行を導出する。** 配信行そのものが outbox の役割を果たしており、中継を 1 つ増やすだけになる。

## 計画

計画の問いは「設計」の節で解決した。
構造の変更（共有の保存先への作用範囲の追加）と振る舞いの変更（発火元が作用範囲の中で捕捉する）は、別のコミットに分ける。

RED と GREEN の確認には次のレシピを使う。

| 確認 | レシピ |
| --- | --- |
| 1 つのテスト | `mise run test-go-test -- <package> <test>` |
| パッケージ | `mise run test-go-package -- <package>` |
| パッケージをまたいだ後 | `mise run test-go-changed` |
| 振る舞いが GREEN になった後 | `mise run lint-go` |

## タスク

- [ ] T001 [Acceptance] `EX-PLATFORM-003-02` について、捕捉の失敗を注入した管理 API の変更が User と割り当てを変えてしまうことを RED として観測する。
- [ ] T002 [Storage] memory と PostgreSQL の保存先に、context で運ぶトランザクションの作用範囲を設ける。
- [ ] T003 [App] User の変更と捕捉を同じ作用範囲で確定させる。
- [ ] T004 [App] 割り当ての追加と解除を捕捉と同じ作用範囲で確定させる。
- [ ] T005 [Wiring] memory と PostgreSQL の組み立てで作用範囲を IdManagement と Application へ渡す。
- [ ] T006 [Acceptance] `EX-PLATFORM-003-01` と `EX-PLATFORM-003-02` を `//spec:covers` で名指すテストを置き、台帳から外す。
- [ ] T007 [Docs] 設計ガイドラインと Provisioning の内部設計へ、作用範囲の仕組みを反映する。
- [ ] T008 [Verify] 変更を検証する。

## 検証

- `mise run check-spec`
- `mise run test-go-changed`
- `mise run verify`

## リスク

- **memory の組み立てだけで原子性を確かめる。** memory の保存先にはトランザクションがなく、成功経路だけのテストはコミット後の呼び出しでも通る。失敗の注入でロールバックを観測する。
- **依存の向きを逆転させる。** IdManagement が Provisioning を import すると Context Map に違反する。合成は composition root に置く。


## 完了

- **Completed At**: 2026-09-24
- **Summary**:
  取り消す。管理 API の経路で、User と割り当ての変更と配信行を 1 つのトランザクションの作用範囲で確定させる実装を終え、`mise run verify` も通したが、コミットせずに破棄した。
  調査の過程で、LifecycleWorkflow による有効化と無効化、SCIM の取り込み、CSV インポート、利用者自身のプロフィール変更が、User を書き換えても配信行を一切作らないことが分かった。書き込み時の原子性はこの漏れに効かない。そのうえ、全保存先を作用範囲へ参加させる `TransactionScopedDB` と、memory の書き戻しの仕組みという横断的な仕組みが必要になり、得られる保証に見合わなかった。
  配信の保証は、あるべき状態と下流へ反映済みの状態を突き合わせる照合で行う方針に改め、[[wi-92540-guarantee-provisioning-deliveries-by-reconciliation]] が `REQ-PLATFORM-003` の改訂と照合の実装を引き継ぐ。同じ調査で見つかった削除カスケードの再実行不能とクォータの加算の取りこぼしは、[[wi-83071-make-user-deletion-rerunnable-and-recount-quota-usage]] が扱う。
  製品、仕様、文書には変更を加えずに終了する。上の設計の節は、採用しなかった方式の記録として残す。
