---
status: completed
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-04
priority: p1
depends_on: []
change_kind: bugfix
evidence_policy: risk-based-v4
documentation_impact:
  level: upgrade_note
  reason: worker のプロセスが SET の `iss` に使う `ISSUER` を読むようになり、運用者は worker にも api と同じ値を設定する必要がある。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-21671-react-to-revocation-events-emitted-by-the-worker.md }
    - { kind: upgrade_note, path: docs/releases/upgrades/wi-21671-react-to-revocation-events-emitted-by-the-worker.md }
initial_context:
  specification:
    - docs/domain/scenarios.feature.md#REQ-PLATFORM-001
    - docs/domain/sharedsignals/revocation/acceptance.feature.md#REQ-SHAREDSIGNALS-007
  typespec: []
  source:
    - backend/sharedsignals/usecases/revocation.go
    - backend/sharedsignals/usecases/project.go
    - backend/shared/http/server_http/routes.go
    - backend/idmanagement/deps_http/deps.go
    - backend/cmd/internal/bootstrap/audit_event_record.go
    - backend/cmd/internal/bootstrap/user_lifecycle.go
    - backend/cmd/internal/bootstrap/apiconfig.go
    - backend/cmd/internal/bootstrap/workerconfig.go
    - backend/cmd/idmagic-worker/worker.go
    - backend/cmd/idmagic/server.go
    - backend/oauth2/token/usecases/introspect_token.go
  tests:
    - backend/cmd/idmagic-worker/worker_lifecycle_workflow_test.go
    - backend/sharedsignals/usecases/project_test.go
    - backend/oauth2/token/usecases/introspect_token_agent_revocation_test.go
  stop_before_reading: [frontend, backend/shared/http/server_http/cross_context_examples_test.go]
affected_spec:
  - { path: docs/domain/scenarios.feature.md, requirement: REQ-PLATFORM-001, impact: conforms }
  - { path: docs/domain/sharedsignals/revocation/acceptance.feature.md, requirement: REQ-SHAREDSIGNALS-007, impact: conforms }
primary_use_cases:
  - id: worker-disable-user-revokes-agent-tokens
    requirement: REQ-PLATFORM-001
    observable_result: worker が実行するワークフローの disable_user で User を止めると、所有する Agent の失効エポックが進み、その前に発行されたトークンはイントロスペクションの失効判定で無効になり、Agent ごとに AgentAccessRevoked が発行される。
    boundary: acceptance
    test: { path: backend/cmd/idmagic-worker/worker_lifecycle_workflow_test.go, name: TestWorkerDisableUserStepRevokesTheTokensOfTheAgentsTheUserOwns, task: test-go-race }
    fault_model: UserLifecycleCommands の Emit が反応器を通らず、監査への記録だけで終わる。
  - id: worker-disable-user-propagates-revocation
    requirement: REQ-SHAREDSIGNALS-007
    observable_result: 同じ disable_user で、session-revoked を購読する有効な Transmit ストリームへ、署名済みの SET を載せた pending の配送が Agent ごとに作られる。
    boundary: acceptance
    test: { path: backend/cmd/idmagic-worker/worker_lifecycle_workflow_test.go, name: TestWorkerDisableUserStepQueuesTheRevocationForTransmitStreams, task: test-go-race }
    fault_model: worker の反応器が AgentAccessRevoked を外部への伝播の投影へ渡さない、または署名に使う鍵と issuer を受け取らない。
  - id: scim-deactivation-revokes-agent-tokens
    requirement: REQ-PLATFORM-001
    observable_result: api の SCIM の取り込みが User を止めると、所有する Agent の失効エポックが進む。
    boundary: acceptance
    test: { path: backend/cmd/idmagic/sourcing_wiring_test.go, name: TestSCIMUserLifecycleAdvancesTheRevocationEpochOfOwnedAgents, task: test-go-race }
    fault_model: api の SCIM の取り込みへ渡す UserLifecycleCommands が、管理 API と違って反応器を持たない。
---

# worker が発行した Agent と所有者のイベントでも失効エポックを進める

## 動機

Agent と所有者のイベント（`AgentKilled`、`AgentDisabled`、`AgentCredentialUnbound`、`UserDisabled`、`UserSoftDeleted`、`UserDeleted`）に反応して失効エポックを進める `AgentRevocationReactor` は、`api` のプロセスの配信点（`backend/shared/http/server_http/routes.go`）にだけ組み込まれている。
`worker` のプロセスの配信点（`NewEmitFunc`）は反応器を持たない。

`worker` で実行する操作、たとえばライフサイクルワークフローの `disable_user` の手順が User を無効化すると、IdManagement は配下の Agent を無効化してイベントを発行するが、失効エポックは進まず、外部の受信側への CAEP のイベントの伝播も起きない。
REQ-PLATFORM-001 は、所有者の無効化がログイン、既存のセッション、配下のエージェントのトークンを同時に閉じることを保証している。

wi-26063 で SharedSignals と IdGovernance の内部設計をコードと照合して見つけ、`docs/domain/sharedsignals/design/risks.md` と `docs/domain/identity-governance/design/risks.md` に載せた。

## 対象範囲

- 失効エポックの反応を、`worker` を含むすべてのプロセスの配信点で働かせる。
- ワークフローの `disable_user` で、配下の Agent のトークンがイントロスペクションで無効になることを確かめる。
- 二つの `design/risks.md` の該当の行を消す。

## 対象外

- 配信不能の配送のやり直し。wi-16152 が扱う。

## 設計

### 欠陥の所在

六つのイベントを発行するのは IdManagement の `AdminUserDeps` と `AdminAgentDeps` を通る操作だけで、その `Emit` を組み立てる場所は二つある。

| 組み立てる場所 | 使う経路 | 反応器 |
| --- | --- | --- |
| `idmanagement/deps_http.Deps.ReactiveEmit` | 管理 API の User と Agent の操作 | `server_http/routes.go` が組み立てて渡す |
| `bootstrap.Dependencies.UserLifecycleCommands` | `worker` のワークフローの `disable_user`、`api` の SCIM の取り込み | なし（欠陥） |

起票時は `worker` だけの欠陥と見ていたが、`api` の SCIM の取り込みも同じ `UserLifecycleCommands` を通るので、同じく失効エポックが進まない。

### 採用する設計

反応器の組み立てを `sharedsignals/usecases` の一つのコンストラクターにまとめ、二つの組み立て場所の両方から使う。

```go
// sharedsignals/usecases
type RevocationReactorDeps struct {
    EpochRepo ssports.AgentRevocationEpochRepository
    AgentRepo agentports.AgentRepository
    Projector ProjectorDeps
    // Emit は派生イベント（RevocationEpochAdvanced、AgentAccessRevoked）の記録先。
    Emit func(spec.DomainEvent)
}
func NewAgentRevocationReactor(deps RevocationReactorDeps) *AgentRevocationReactor

// cmd/internal/bootstrap
func (d *Dependencies) AgentRevocationReactor(emit func(spec.DomainEvent)) *ssusecases.AgentRevocationReactor
func (d *Dependencies) UserLifecycleCommands(emit func(spec.DomainEvent), actor string) userusecases.UserLifecycleCommands
```

- `NewAgentRevocationReactor` の `Emit` は、派生イベントを記録し、`AgentAccessRevoked` なら 5 秒の期限で `ProjectAgentAccessRevoked` を呼び、失敗を記録して続ける。これは `routes.go` にあった閉包そのものである。
- `UserLifecycleCommands` の `Emit` は、イベントを記録した後に反応器を 5 秒の期限で呼び、その誤りを返す。管理 API の `ReactiveEmit` と同じく fail-closed にする。
- 作用の境界: 時刻はイベントの `At`、署名は `SigningKeys` の `KeyStore`、SET の `iss` は `Dependencies.Issuer`、永続化は SharedSignals の Repository である。

### 発見した前提: worker の issuer

SET の `iss` に使う `ISSUER` は `APIConfig` だけが読み、`worker` は知らない。
`ISSUER` を `SharedConfig` へ移し、`Assemble` が `Dependencies.Issuer` に入れる。
`api` は `shared.Issuer` を使う。既定値と検証（絶対 URL）は変えない。
`worker` に設定しないと既定値の `http://localhost:8080` で署名されるので、アップグレードノートで設定を求め、Docker Compose の `worker` にも設定する。

### 採用しない代替案

| 代替案 | 採用しない理由 |
| --- | --- |
| `NewEmitFunc` に反応器を入れる | 戻り値がないので fail-closed にできない。管理 API の `ReactiveEmit` と二重に反応する |
| 反応器を `bootstrap` で組み立てて `server_http.Deps` へ渡す | `server_http` のテストが組み立てを自前で再現する必要が生じる。組み立ての規則は SharedSignals のコンストラクターに置けば一か所になる |

## 計画

1. `worker` のテストで、失効エポックと SET の配送が欠けることを RED として確かめる。SCIM の経路も同じく確かめる。
2. `ISSUER` を共有設定へ移す。
3. 反応器のコンストラクターを作り、`routes.go` と `UserLifecycleCommands` から使う。
4. 設計文書のリスクの行と実行時の流れを直す。

## タスク

- [x] T001 [Acceptance] `TestWorkerDisableUserStepRevokesTheTokensOfTheAgentsTheUserOwns`、`TestWorkerDisableUserStepQueuesTheRevocationForTransmitStreams`、`TestSCIMUserLifecycleAdvancesTheRevocationEpochOfOwnedAgents` で RED を確かめる（REQ-PLATFORM-001、REQ-SHAREDSIGNALS-007）。検査は `mise run test-go-test -- ./backend/cmd/idmagic-worker <name>` と `mise run test-go-test -- ./backend/cmd/idmagic <name>`。
  - RED: `revocation epoch of deploy-bot = <nil>, <nil>; want advanced`、`deliveries = 0, want one per agent (2)`、`revocation epoch = <nil>, <nil>; want advanced for OwnerDisabled`。
- [x] T002 [App] `ISSUER` を `SharedConfig` へ移す。検査は `mise run test-go-package -- ./backend/cmd/internal/bootstrap`。`TestLoadSharedConfigReadsIssuer` と `TestAssembleHandsTheIssuerToTheDependencies` で固定する。
- [x] T003 [App] `NewAgentRevocationReactor` を作り、`routes.go` と `UserLifecycleCommands` から使う。検査は T001 のテストと `mise run test-go-package -- ./backend/sharedsignals/usecases`、`./backend/shared/http/server_http`。
  - 仕様にない振る舞いの分類: `UserLifecycleCommands` の反応の誤りを呼び出し元へ返す fail-closed は、管理 API の `ReactiveEmit` の既存の設計に合わせた。管理 API 側も要件とテストを持たないので、この記録では要件にせず（b）、経路をそろえるにとどめる。
- [x] T004 [Docs] 二つの `design/risks.md` の行を消し、SharedSignals のアーキテクチャの実行時の流れを直し、構成の参照を生成し直し、リリース文書を書く。
- [x] T005 [Verify] 変異試験と `mise run verify` を通す。

## 検証

- `mise run test-go-changed`
- `mise run test-go-mutation -- backend/sharedsignals/usecases`
- `mise run verify`

## リスク

反応器が二重に組み込まれると、同じイベントでエポックを二度進めようとする。
エポックの前進は単調で同じ時刻を進めないので害はないが、組み込みは経路ごとに一か所にする。
管理 API は `ReactiveEmit`、それ以外は `UserLifecycleCommands` だけが反応器を呼ぶ。

## 完了

- **Completed At**: 2026-10-04
- **Summary**:
  `mise run spec-diff` は main に対する規範仕様の差分を報告しない（`no normative specification change against main`）。REQ-PLATFORM-001 と REQ-SHAREDSIGNALS-007 を変えず、実装を合わせた。
  管理 API の外から User を止める経路（`worker` のワークフローの `disable_user` と `api` の SCIM の取り込み）が使う `UserLifecycleCommands` が、管理 API と同じ反応器を発行の後に呼ぶようになり、所有する Agent の失効エポックが進み、SET の配送が作られる。
  反応器の組み立ては `sharedsignals/usecases.NewAgentRevocationReactor` の一つにまとめた。
  SET の `iss` に使う `ISSUER` を共有の設定へ移し、`worker` も読む。
  二つの `design/risks.md` から該当の行を消し、SharedSignals のアーキテクチャの実行時の流れを直した。
- **Primary Use Case Evidence**:
  - id: worker-disable-user-revokes-agent-tokens
    red: 実装前、TestWorkerDisableUserStepRevokesTheTokensOfTheAgentsTheUserOwns が `revocation epoch of deploy-bot = <nil>, <nil>; want advanced` で失敗した。
    fault_injection: 組み立ての地点の `UserLifecycleCommands` の `Emit` から `reactor.React` の呼び出しを外すと、同じテストが同じ失敗をした。
  - id: worker-disable-user-propagates-revocation
    red: 実装前、TestWorkerDisableUserStepQueuesTheRevocationForTransmitStreams が `deliveries = 0, want one per agent (2)` で失敗した。
    fault_injection: bootstrap の投影へ渡す `Issuer` を空にすると、また `NewAgentRevocationReactor` から `ProjectAgentAccessRevoked` の呼び出しを外すと、同じテストが失敗した。
  - id: scim-deactivation-revokes-agent-tokens
    red: 実装前、TestSCIMUserLifecycleAdvancesTheRevocationEpochOfOwnedAgents が `revocation epoch = <nil>, <nil>; want advanced for OwnerDisabled` で失敗した。
    fault_injection: 組み立ての地点の `UserLifecycleCommands` の `Emit` から `reactor.React` の呼び出しを外すと、同じテストが同じ失敗をした。
- **Change-Resistance Results**:
  `mise run test-go-mutation -- backend/sharedsignals/usecases` は 125 件中 112 件を検出した。
  生き残った 13 件は `admin_streams.go`、`deliver.go`、`project.go`、`receive.go`、`transmit.go` の既存の分岐で、この記録は触れていない。
  初回は新しい `NewAgentRevocationReactor` の 3 件がパッケージ内のテストから未到達だったため、TestNewAgentRevocationReactor_RecordsAndProjectsTheRevocation を足した。
  投影の誤りを `React` の誤りとして返す誤実装を注入すると、このテストの `ProjectionFailureDoesNotFailTheRevocation` が `React: issuer: is required` で失敗した。
  `Assemble` が `Dependencies.Issuer` を写さない誤実装は TestAssembleHandsTheIssuerToTheDependencies が検出した。
  `UserLifecycleCommands` の反応の誤りを呼び出し元へ返す fail-closed は、管理 API と同じく要件もテストも持たない。この記録の範囲では要件にしていない。
- **Verification Results**:
  - `mise run verify` - `check-repository` だけが失敗した。アップグレードノートの「既定」の表記を「デフォルト」に直し、`mise run check-repository` が成功した。ほかのゲート（DB を含む `test-go-race`、`lint-go`、UI）は成功した。
  - `mise run test-ui-e2e` - 実行していない。変更はバックエンドの組み立てと設定に限られ、ブラウザーへ届かない。
