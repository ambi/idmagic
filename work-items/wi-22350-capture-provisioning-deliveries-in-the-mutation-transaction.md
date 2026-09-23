---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-23
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/domain/scenarios.feature.md, requirement: REQ-PLATFORM-003 }
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

採用候補は、この境界 port の実装側で Provisioning の capture も同じトランザクションへ入れる形である。
IdManagement から Provisioning への依存を作らない（Context Map の向きは Provisioning → IdManagement）という制約を保つには、コミットの組み立てを composition root 側に置く必要がある。

採用しない代替案:

- **コミット後の best-effort 呼び出しを再試行で補う。** 再試行しても、2 つのコミットの間の障害で capture が失われる状態は残る。規則は「同時に」を求めている。
- **outbox から非同期に配信行を導出する。** 配信行そのものが outbox の役割を果たしており、中継を 1 つ増やすだけになる。

## 計画

着手時に次の問いを解決する。どれも作るものを変える。

1. IdGovernance の capture と Provisioning の capture を、1 つのトランザクションへどう合成するか。単一の committer に両方を持たせるのか、トランザクションを跨いで渡せる単位（Postgres の `pgx.Tx` を運ぶ文脈）を導入するのか。
2. Application の割り当ての保存に、同じ形の境界 port を設けるか。
3. memory の保存先で「ロールバック」をどう観測するか。capture の失敗を注入したとき、User の変更も保存されないことを読める形にする。

## タスク

- [ ] T001 [Acceptance] `EX-PLATFORM-003-02` について、capture の失敗を注入した管理 API の変更が User を変えてしまうことを RED として観測する。
- [ ] T002 [App] User の変更と capture を同一トランザクションで確定させる。
- [ ] T003 [App] 割り当て解除と capture を同一トランザクションで確定させる。
- [ ] T004 [Acceptance] `EX-PLATFORM-003-01` と `EX-PLATFORM-003-02` を `//spec:covers` で名指すテストを置き、台帳から外す。
- [ ] T005 [Verify] 変更を検証する。

## 検証

- `mise run check-spec`
- `mise run test-go-changed`
- `mise run verify`

## リスク

- **memory の組み立てだけで原子性を確かめる。** memory の保存先にはトランザクションがなく、成功経路だけのテストはコミット後の呼び出しでも通る。失敗の注入でロールバックを観測する。
- **依存の向きを逆転させる。** IdManagement が Provisioning を import すると Context Map に違反する。合成は composition root に置く。
