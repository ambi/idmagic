---
status: completed
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-03
priority: p1
depends_on: [wi-21670-move-identity-management-to-the-feature-and-design-layout, wi-18703-align-identity-management-with-its-state-tables]
change_kind: bugfix
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: 所有者の User を止めると Agent も止まり、所有者が止まった Agent の再有効化を管理 API が 409 で拒否するようになる。ライフサイクルワークフローと SCIM による User の停止が IdManagement のイベントと副作用を伴うようになる。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-96797-stop-agents-when-their-owner-stops.md }
initial_context:
  specification:
    - docs/domain/identity-management/principals/agent/README.md
    - docs/domain/identity-management/principals/user/lifecycle.md
    - docs/domain/identity-management/glossary.md
    - docs/design/architecture/logical.md
  typespec: [IdMagic.IdManagement.Operations.EnableAgent]
  source:
    - backend/idmanagement/agent/usecases/admin_agents.go
    - backend/idmanagement/user/usecases/admin_users.go
    - backend/idgovernance/usecases/lifecycle_workflow_dispatcher.go
    - backend/sourcing/scim/usecases/users.go
    - backend/cmd/idmagic-worker/worker.go
  tests: [backend/idmanagement/agent/usecases, backend/idmanagement/user/usecases, backend/idgovernance/usecases, backend/sourcing/scim/usecases]
  stop_before_reading: [frontend]
affected_spec:
  - { path: docs/domain/identity-management/principals/agent/README.md, requirement: REQ-IDMANAGEMENT-081 }
  - { path: docs/domain/identity-management/principals/agent/README.md, requirement: REQ-IDMANAGEMENT-082 }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.EnableAgent }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.AgentOwnerInactiveError }
  - { path: docs/domain/identity-governance/scenarios.feature.md, requirement: REQ-IDGOVERNANCE-008 }
primary_use_cases:
  - id: owner-disable-stops-agents
    requirement: REQ-IDMANAGEMENT-081
    observable_result: 管理者が所有者の User を無効化すると、その User が所有する Active の Agent が Disabled になり、AgentDisabled が発行される。
    unit_test:
      path: backend/idmanagement/user/usecases/user_rules_test.go
      name: TestStoppingAUserDisablesTheAgentsTheyOwn
      task: test-go-race
    e2e_test:
      path: backend/idmanagement/handlers_http/lifecycle_refusal_effects_test.go
      name: TestDisablingTheOwnerDisablesTheirAgents
      task: test-go-race
    unit_fault_model: User を止めるユースケースが、所有する Agent の無効化を呼ばない。
    e2e_fault_model: 管理 API の組み立てが、User のユースケースへ Agent のリポジトリを渡さず、伝播が黙って止まる。
  - id: workflow-and-scim-stop-agents
    requirement: REQ-IDMANAGEMENT-081
    observable_result: ライフサイクルワークフローの disable_user と SCIM の active=false が、IdManagement の無効化を通り、所有する Agent を Disabled にする。
    unit_test:
      path: backend/idgovernance/usecases/scenario_examples_test.go
      name: TestDisableUserActionGoesThroughTheUserLifecyclePort
      task: test-go-race
    e2e_test:
      path: backend/sourcing/scim/usecases/user_lifecycle_test.go
      name: TestScimDeactivationStopsTheUserAndTheAgentsTheyOwn
      task: test-go-race
    unit_fault_model: ワークフローの手順が User を直接保存し、IdManagement のイベントと Agent の伝播を迂回する。
    e2e_fault_model: SCIM の active=false が User を直接保存し、IdManagement のイベントと Agent の伝播を迂回する。
---

# 所有者の User が止まると、その User が所有する Agent を止める

## 動機

Agent は所有者を必須とし、誰も責任を持たない非人間のアイデンティティを残さないことを判断としている。
ところが、所有者の User を無効化しても、削除を予約しても、完全削除しても、Agent は動き続ける。
さらに、ライフサイクルワークフローの `disable_user` と `enable_user`、SCIM の取り込みの `active` と `DELETE` は、IdManagement のユースケースを通らずに User の `status` を直接書く。
そのため、退職による停止の主な経路では、`UserDisabled` などのイベント、記憶済みの端末の失効、動的グループの再評価、下流への通知、削除予約中の User の拒否（REQ-IDMANAGEMENT-046）も行われない。
用語集は Agent の所有者を User または Group としているが、実装は `Active` の User だけを受け付ける。

製品は未リリースなので、あるべき形に直す。

## 対象範囲

- 所有者の User を無効化する、削除を予約する、完全削除すると、その User が所有する `Active` の Agent を `Disabled` にし、`AgentDisabled` を発行する（REQ-IDMANAGEMENT-081）。
- 所有者の User が `Active` でない Agent の再有効化を、409 と `agent_owner_inactive` で拒否する（REQ-IDMANAGEMENT-082）。管理 API の契約に 409 を加える。
- ライフサイクルワークフローの `disable_user` と `enable_user` を、IdGovernance のポートを通して、IdManagement の `SetUserDisabled` で行う。
- SCIM の取り込みの `active` の切り替えを `SetUserDisabled` で、`DELETE` を `SoftDeleteUser` で行う。Sourcing のポートを通す。
- 伝播が途中で失敗したときに、同じ操作の再実行で残った Agent を回収できるようにする。
- 用語集の Agent の所有者を、同じテナントの User に直す。

## 対象外

- 所有者の User を再有効化または復元したときに、Agent を自動で再有効化すること。止めた理由が所有者の停止かどうかを Agent が記録しないので、管理者が確かめてから再有効化する。
- Group を Agent の所有者にすること。
- SCIM の作成で `active=false` の User を作る経路。作成の時点では所有する Agent がない。

## 設計

### IdManagement

```go
// agent/usecases
func DisableAgentsOwnedBy(ctx context.Context, deps AgentOwnerDeps, tenantID, ownerUserID, actorUserID string, now time.Time) error

type AgentOwnerDeps struct {
    AgentRepo agentports.AgentRepository
    Emit      func(spec.DomainEvent) error
}

var ErrAgentOwnerInactive = errors.New("agent owner is not active")
```

`DisableAgentsOwnedBy` は、テナントの Agent のうち所有者が一致し `Active` のものだけを `Disabled` にし、Agent ごとに `AgentDisabled` を発行する。
`Disabled` と `Killed` の Agent は変えない。

`AdminUserDeps` に `AgentRepo` を加える。
`SetUserDisabled`（無効化）、`SoftDeleteUser`、`DeleteUser` は、User を確定した後に `DisableAgentsOwnedBy` を呼ぶ。
`AgentRepo` が配線されていなければ呼ばない。
期限切れの自動の完全削除は `DeleteUser` を通るので、同じく伝わる。

伝播は User の保存と同じトランザクションにせず、再実行で回収する。
すでに `Disabled` の User への無効化と、すでに削除予約中の User への削除の予約は、User を変えずに `DisableAgentsOwnedBy` だけを呼ぶ。
`DeleteUser` は、Tombstone を保存する前に `DisableAgentsOwnedBy` を呼ぶ。途中で失敗しても User が残り、完全削除を再実行できる。

```go
// user/usecases
type UserLifecycleCommands struct {
    Deps  AdminUserDeps
    Actor string
}

func (c UserLifecycleCommands) SetUserDisabled(ctx context.Context, tenantID, userID string, disabled bool, now time.Time) (changed bool, err error)
func (c UserLifecycleCommands) ScheduleUserDeletion(ctx context.Context, tenantID, userID string, now time.Time) error
```

`UserLifecycleCommands` は、管理 API の外の Context が User を止めるための操作である。
テナントを文脈に載せ、`SetUserDisabled` と `SoftDeleteUser` を `Actor` で呼ぶ。
`UserMutationCommitter` は渡さない。
ほかの Context の手順による User の変更から、さらにワークフローの実行を作らないためである（変更前の直接の保存と同じ）。
削除予約中の User への無効化と再有効化（`ErrUserPendingDeletion`）は、`changed=false` として返す。
IdGovernance と Sourcing のポートは、この型が構造的に満たす。
ほかの Context が IdManagement の `usecases` を参照すると境界の検査に反するので、ポートの実装を相手の Context に置かない。

`SetAgentDisabled` の再有効化は、所有者の User を読み、`Active` でなければ `ErrAgentOwnerInactive` を返す。
ハンドラーはそれを 409 と `agent_owner_inactive` に写す。

### IdGovernance

```go
// idgovernance/ports
type UserLifecycle interface {
    SetUserDisabled(ctx context.Context, tenantID, userID string, disabled bool, now time.Time) (changed bool, err error)
}
```

`LifecycleWorkflowExecutorDeps` に `UserLifecycle` を加え、`disable_user` と `enable_user` の手順はそれを呼ぶ。
`changed=false` の手順は変更なしとして扱う。
`UserLifecycle` が配線されていなければ、手順は `dependency_unavailable` で失敗する。

退職のシナリオ（EX-IDGOVERNANCE-008-01）は、削除予約中の User を `Disabled` にする遷移に頼っていた。
削除予約中の User は無効化できないので、トリガーを属性の変化に直す。

### Sourcing

```go
// sourcing/scim/ports
type UserLifecycle interface {
    SetUserDisabled(ctx context.Context, tenantID, userID string, disabled bool, now time.Time) (changed bool, err error)
    ScheduleUserDeletion(ctx context.Context, tenantID, userID string, now time.Time) error
}
```

SCIM の置き換えと部分更新は、`status` 以外の項目を保存した後に、`active` が変わる場合だけ `SetUserDisabled` を呼び、変わった User を読み直して応答する。
`DELETE` は `ScheduleUserDeletion` を呼ぶ。
SCIM は削除予約中の User を見つからないものとして扱うので、削除予約中の User への `active` の要求は 404 のままである。

### SharedSignals

失効の反応器は、所有者の停止（`UserDisabled`、`UserSoftDeleted`、`UserDeleted`）と `AgentDisabled` の両方で Agent の失効エポックを進める。
伝播を加えると、同じ時刻の二つのイベントで同じ Agent を二度失効させ、`AgentAccessRevoked` と CAEP の送信が重複する。
失効エポックの保存先は、同じ時刻への前進を受け入れていたが、仕様は単調な前進を定めている。
同じ時刻への前進を `ErrEpochNotAdvancing` として進めないように、メモリと PostgreSQL の保存先と契約テストを直す。
理由は先に届いた所有者の停止が残る。

### 組み立て

`bootstrap.Dependencies.UserLifecycleCommands(emit, actor)` が、管理 API のハンドラーと同じ依存の集合で `UserLifecycleCommands` を組み立てる。
`worker` は操作者 `lifecycle-workflow` で IdGovernance へ、`api` の起動はイベントの出力先を決めた後に操作者 `scim` で `sourcing.Module.UserLifecycle` へ渡す。

### 採らない案

| 案 | 採らない理由 |
| --- | --- |
| OAuth2 のトークンの発行の時点で所有者を確かめる | Agent の状態は `Active` のまま残り、管理者が見る状態と実際に使えるかが食い違う |
| ワークフローと SCIM の直接の保存の後に、Agent の伝播だけを足す | イベント、端末の失効、動的グループ、下流への通知、削除予約中の拒否の漏れが残る |
| 所有者の再開で Agent も再開する | 所有者の停止の前から止めていた Agent まで再開してしまう |

## 計画

1. 規則、用語集、TypeSpec を変える。
2. IdManagement の伝播と再有効化の拒否を実装する。
3. IdManagement の `UserLifecycleCommands` と IdGovernance のポートを実装し、ワークフローの手順を移す。
4. Sourcing のポートを実装し、SCIM の流れを移す。
5. 組み立てと、設計文書を直す。

## タスク

- [x] T001 [Spec] 規則、用語集、TypeSpec を変える。
  Acceptance RED：`mise run check` が、追加した EX-IDMANAGEMENT-081 と 082 の例をテストが引いていないことを報告した。
- [x] T002 [App] IdManagement の伝播と再有効化の拒否を実装する。
  テストは実装と並行して書いたので、RED は誤りの注入で確かめた（完了の欄）。
  実行：`mise run test-go-package -- <package>`、`mise run lint-go`、`mise run check-contract-drift`。
- [x] T003 [App] IdGovernance のワークフローの手順を IdManagement 経由にする。
  最初はアダプターを IdGovernance に置いたが、`mise run check-boundaries` が IdManagement の `usecases` の参照を拒否した。IdManagement の `UserLifecycleCommands` を組み立ての地点から注入する形に改めた。
  退職のシナリオの `TestLeaverWorkflowRevokesAccessEvenWhenTheNotificationIsBlocked` は、削除予約中の User を `Disabled` にする遷移に頼っていたので失敗した。シナリオと例のトリガーを属性の変化に直した。
- [x] T004 [App] SCIM の取り込みを IdManagement 経由にする。
  `mise run check` が、既存の `sourcing/scim/handlers_http/user_lifecycle_test.go` を上書きして消した EX-SOURCING-002-02〜04 の引用の欠落を検出した。元の内容に戻し、補助関数は別のファイルに置いた。
  変異テストで、`active` に触れない PATCH の初期値の否定が生き残ったので、`TestScimPatchWithoutActiveLeavesTheUserStatusAlone` を加えた。
- [x] T005 [App] 組み立てと設計文書を直す。
  伝播の失敗を再実行で回収できるよう、止まっている User への無効化と削除の予約でも伝播を行い、完全削除は Tombstone の前に伝播するよう改めた（EX-IDMANAGEMENT-081-05）。
  API の組み立てを `sourcingModule` に切り出し、`TestServerWiresScimUserStopsThroughIdManagement` で固定した。
- [x] T006 [Verify] 変更を検証する。
  最初の `mise run verify` で、`TestAdminDisablingAUserClosesEveryPathToItAtOnce` が「AgentAccessRevoked の Agent ごとの件数=map[A1:2 A2:2]」で失敗した。失効エポックの同じ時刻への前進を進めないよう SharedSignals の保存先を直し、再実行で成功した。

## 検証

- `mise run check`、`mise run check-boundaries`
- `mise run verify`

## リスク

- ワークフローと SCIM の User の停止が、これまで行っていなかった副作用（イベント、端末の失効、下流への通知）を伴う。下流の宛先への通知が増える。
- 失効エポックの同じ時刻への前進を進めなくなるので、同じ時刻に重なった二つ目の失効の理由は記録されない。

## 完了

- **Completed At**: 2026-10-03
- **Summary**:
  `mise run spec-diff -- work-item/wi-18703` は、追加した規則 REQ-IDMANAGEMENT-081 と 082、変更した規則 REQ-IDGOVERNANCE-008、追加した TypeSpec の `AgentOwnerInactiveError`、変更した `EnableAgent` を報告する。
  所有者の User を無効化する、削除を予約する、完全削除すると、その User が所有する `Active` の Agent を `Disabled` にする。所有者が `Active` でない Agent の再有効化は 409 と `agent_owner_inactive` で拒否する。
  ライフサイクルワークフローと SCIM の User の停止は、IdManagement の `UserLifecycleCommands` を通り、管理 API と同じイベントと副作用を伴う。
  失効エポックは同じ時刻へは進めず、所有者の停止と Agent の無効化が重なっても Agent ごとの失効は一度だけになる。
- **Primary Use Case Evidence**:
  - id: owner-disable-stops-agents
    unit_red: テスト `TestStoppingAUserDisablesTheAgentsTheyOwn` は実装と並行して書いたので RED を観測していない。下の誤りの注入で検出を確かめた。
    e2e_red: テスト `TestDisablingTheOwnerDisablesTheirAgents` も同じく、誤りの注入で検出を確かめた。
    unit_fault_injection: 伝播の本体 `disableOwnedAgents` を何もしない形にすると、無効化、削除の予約、完全削除の三つとも「Agent deploy-bot の状態=active, want disabled」で失敗した。
    e2e_fault_injection: 管理 API のハンドラーの依存から `AgentRepo` を外すと、「所有者を無効化したのに Agent の状態が active のまま」で失敗した。
  - id: workflow-and-scim-stop-agents
    unit_red: テスト `TestDisableUserActionGoesThroughTheUserLifecyclePort` は実装と並行して書いたので RED を観測していない。下の誤りの注入で検出を確かめた。
    e2e_red: テスト `TestScimDeactivationStopsTheUserAndTheAgentsTheyOwn` も同じく、誤りの注入で検出を確かめた。
    unit_fault_injection: ワークフローの手順を User の直接の保存に戻すと、「calls=[], want [... disabled=true]」で失敗した。
    e2e_fault_injection: SCIM の `active=false` を User の状態の直接の書き換えに戻すと、「Agent の状態=active, want disabled」で失敗した。
- **Change-Resistance Results**:
  `mise run test-go-mutation` の生き残りは、SCIM の usecases で 14 件、Agent の usecases で 2 件、User の usecases で 37 件、IdGovernance の usecases で 10 件である。今回変えた行に残った `users.go` の `active` の初期値と、`admin_users.go` の再実行の伝播の失敗の分岐は、テストを加えて検出させた。残りは今回変えていない箇所（SCIM のフィルター属性と応答の組み立て、Agent の束縛の競合、User の CSV、ワークフローの集計）にある。
- **Verification Results**:
  - `mise run check` - 成功
  - `mise run check-boundaries` - 成功
  - `mise run lint-go` - 成功
  - `mise run verify` - 成功
