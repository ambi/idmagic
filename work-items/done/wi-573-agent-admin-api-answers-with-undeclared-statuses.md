---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-13
priority: p3
depends_on: []
change_kind: bugfix
evidence_policy: risk-based-v3
documentation_impact:
  level: release_note
  reason: Agent 管理 API が実際に返す 404、409、422 を生成クライアントが型付きの応答として扱えるようになり、クライアント資格情報の参照先が見つからない拒否は 404 から 422 へ移る。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-573-agent-admin-api-answers-with-undeclared-statuses.md }
initial_context:
  specification:
    - docs/domain/identity-management/scenarios.feature.md#REQ-IDMANAGEMENT-009
    - docs/design/application/api-guidelines.md
  typespec:
    - IdMagic.IdManagement.Operations.GetAgent
    - IdMagic.IdManagement.Operations.UpdateAgent
    - IdMagic.IdManagement.Operations.DisableAgent
    - IdMagic.IdManagement.Operations.EnableAgent
    - IdMagic.IdManagement.Operations.KillAgent
    - IdMagic.IdManagement.Operations.DeleteAgent
    - IdMagic.IdManagement.Operations.BindAgentCredential
    - IdMagic.IdManagement.Operations.UnbindAgentCredential
    - IdMagic.Contract.OAuth2ClientNotFoundError
  source:
    - backend/idmanagement/agent/handlers_http/admin_agent_handler.go
    - backend/idmanagement/agent/usecases/admin_agents.go
    - backend/shared/http/support_http/auth.go
  tests:
    - backend/idmanagement/handlers_http/scenario_examples_test.go
  stop_before_reading:
    - frontend
primary_use_cases:
  - id: bind-foreign-tenant-credential
    requirement: REQ-IDMANAGEMENT-009
    observable_result: 別テナントの `client_id` を Agent へバインドする要求は 422 の `client_not_found` で拒否され、応答は存在しない `client_id` を指定したときと同じで、関連付けは残らない。
    unit_test: { path: backend/idmanagement/agent/usecases/admin_agents_test.go, name: TestBindCredentialRejectsClientOfAnotherTenant, task: test-go-race }
    e2e_test: { path: backend/idmanagement/handlers_http/scenario_examples_test.go, name: TestBindingAForeignTenantsCredentialLeavesTheAgentUnbound, task: test-go-race }
    unit_fault_model: ユースケースが別テナントの OAuth2Client をテナントを見ずに検索し、バインドを受理する。
    e2e_fault_model: HTTP ハンドラーが参照先の見つからないクライアントを宣言外の 404 で書く、または別テナントと存在しない `client_id` とで応答を書き分ける。
  - id: agent-refusal-statuses
    requirement: REQ-IDMANAGEMENT-009
    observable_result: 存在しない Agent への操作は 404 の `agent_not_found`、停止済み Agent への変更と別 Agent に束縛済みのクライアントのバインドは 409 で拒否され、いずれも TypeSpec の宣言に含まれる。
    unit_test: { path: backend/idmanagement/agent/usecases/admin_agents_test.go, name: TestKillAgentIsIrreversible, task: test-go-race }
    e2e_test: { path: backend/idmanagement/handlers_http/agent_refusal_statuses_test.go, name: TestAgentRefusalsUseDeclaredStatuses, task: test-go-race }
    unit_fault_model: ユースケースが停止済み Agent への変更を拒否しない。
    e2e_fault_model: HTTP ハンドラーがユースケースのエラーを宣言と異なる status へ写す。
affected_spec:
  - { path: docs/domain/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-009 }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.GetAgent }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.UpdateAgent }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.DisableAgent }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.EnableAgent }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.KillAgent }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.DeleteAgent }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.BindAgentCredential }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.UnbindAgentCredential }
---

# Agent 管理 API が、契約の宣言に無い 404 と 409 で拒否する

## Motivation

[API ガイドライン](../../docs/design/application/api-guidelines.md)の「ステータスコードの宣言」は、ハンドラーと手前のガードが返すステータスコードをすべて TypeSpec に宣言すると定める。
Agent 管理 API はこの規則を満たしていない。

`writeAdminAgentError` (`backend/idmanagement/agent/handlers_http/admin_agent_handler.go`) は、次の拒否を宣言に無い status で書く。
名前の重複による 409 `agent_name_conflict` は、[[wi-595-declare-uniqueness-conflict-responses]] が宣言済みである。

| エラー | 書いている status | code | 宣言の状態 |
| --- | --- | --- | --- |
| `ErrAgentNotFound` | 404 | `agent_not_found` | 未宣言 |
| `ErrAgentClientNotFound` | 404 | `client_not_found` | 未宣言 |
| `ErrAgentKilled` | 409 | `agent_killed` | 未宣言 |
| `ErrAgentClientBound` | 409 | `agent_client_already_bound` | 未宣言 |

DisableAgent、EnableAgent、KillAgent、DeleteAgent は、`RequireAdmin` と `VerifyBrowserRequest` を経由するのに 401 を宣言せず、403 も `AccessDeniedError` だけを宣言する。
同じガードを経由する BindAgentCredential は、401 と 4 種の 403 を宣言している。

`EX-IDMANAGEMENT-009-04` は、別テナントの `client_id` のバインドについて「エラー "InvalidRequestError"」を要求する。
しかし、この要求は API ガイドラインとも既存の API とも合わない。

- ガイドラインの「400 と 422 の区別」は、400 を解析できないリクエストに、422 を参照の不整合に割り当てる。本文の `owner_user_id` が見つからない拒否は、既に 422 の `AgentOwnerNotFoundError` である。
- `OAuth2ClientNotFoundError` (`client_not_found`) は、別テナントのクライアントを存在しないものとして同じ応答で答えると宣言している。実装が書く `code` も既にこれである。

`mise run check-status-drift` は現状 0 finding で通る。Agent の 9 operation は `writeAdminAgentError` または `changeAgentStatus` を経由するため、「一部だけ読めた」側に入る。
[[wi-540-back-identity-management-examples-with-tests]] が `EX-IDMANAGEMENT-009-04` を消化しようとしてこのずれを見つけた。

## Scope

- Agent 管理 API の宣言を、API ガイドラインと既存 API の形へ揃える。
  - パスの `agent_id` が見つからない拒否は、404 の `AgentNotFoundError` とする。別テナントの Agent は存在しないものとして扱う。GetAdminUser の `UserNotFoundError` と同じ形である。
  - 本文の `client_id` が見つからない拒否は、422 の `OAuth2ClientNotFoundError` とする。実装の status を 404 から 422 へ移す。
  - 停止済み Agent への変更と、別 Agent に束縛済みのクライアントのバインドは、状態の競合として 409 の `AgentKilledError` と `AgentClientAlreadyBoundError` とする。
  - DisableAgent、EnableAgent、KillAgent、DeleteAgent に 401 と、BindAgentCredential と同じ 403 の union を宣言する。
- `EX-IDMANAGEMENT-009-04` の期待するエラーを `OAuth2ClientNotFoundError` へ訂正し、存在しない `client_id` と同じ応答であることを加える。
- `EX-IDMANAGEMENT-009-04` を `tools/check/example-coverage-debt.json` から外し、`TestBindingAForeignTenantsCredentialLeavesTheAgentUnbound` が具体例 id を名指すようにする。
- `mise run check-status-drift` が Agent の operation を全体まで読めない理由を記録し、HTTP の境界テストで宣言との対応を固定する。

## Out of Scope

- Agent 以外の管理 API が宣言外の status を書いていないかの全数調査。本項目は Agent の 10 operation に閉じる。
- 拒否の `code` (`agent_not_found` など) の改名。
- `mise run check-status-drift` がエラーマッピングヘルパーを追跡する改修。API ガイドラインが、ヘルパーの分岐はユースケースが返すエラーで決まるため追跡しないと定めている。
- 空の `client_id` を 400 と 422 のどちらで拒否するかの見直し。現状どおり `client_not_found` として扱う。

## Design

ユースケースの型と作用の境界は変えない。変更は契約と HTTP の写像に閉じる。

| operation | 追加する宣言 |
| --- | --- |
| GetAgent | 404 `AgentNotFoundError` |
| UpdateAgent | 404 `AgentNotFoundError`、409 に `AgentKilledError` |
| DisableAgent / EnableAgent / KillAgent / DeleteAgent | 401、403 に `CsrfFailedError` `InsufficientScopeError` `InvalidOriginError`、404 `AgentNotFoundError`、409 `AgentKilledError` |
| BindAgentCredential | 404 `AgentNotFoundError`、409 `AgentKilledError` `AgentClientAlreadyBoundError`、422 `OAuth2ClientNotFoundError` |
| UnbindAgentCredential | 404 `AgentNotFoundError` |

`writeAdminAgentError` の `ErrAgentClientNotFound` の分岐だけを 404 から 422 へ移す。

## Tasks

- [x] T001 [Spec] TypeSpec にエラー型と status を宣言し、`EX-IDMANAGEMENT-009-04` を訂正する。
- [x] T002 [Acceptance] `TestBindingAForeignTenantsCredentialLeavesTheAgentUnbound` で 422 と応答の同一性を固定し、RED を確かめる。
- [x] T003 [Acceptance] Agent の拒否 status を HTTP の境界で固定する。
- [x] T004 [Impl] `ErrAgentClientNotFound` の写像を 422 へ移す。
- [x] T005 [Docs] API ガイドラインの適用状況とリリースノートを改める。
- [x] T006 [Verify] 変更を検証する。

## Verification

- `mise run check-spec`
- `mise run check-api-compat`
- `mise run check-status-drift`
- `mise run test-go-package -- ./backend/idmanagement/handlers_http`
- `mise run test-go-package -- ./backend/idmanagement/agent/usecases`
- `mise run verify`

## Risk Notes

- **status を書き換えて、拒否の理由を運ぶ経路を壊す。** 呼び出し元は `type` の URN 接尾辞で分岐する。`code` は変えない。
- **404 が別テナントの Agent の存在を漏らす。** ユースケースはテナントで絞って検索するため、別テナントの Agent と存在しない Agent は同じ 404 になる。境界テストで両者の応答が同じことを確かめる。
- **`client_not_found` の 404 から 422 への移動は、404 で分岐していた呼び出し元を壊す。** 404 は宣言外だったため、生成クライアントはこの分岐を持てなかった。リリースノートで知らせる。

## Completion

- **Completed At**: 2026-09-23
- **Summary**:
  `mise run spec-diff` は `REQ-IDMANAGEMENT-009` の具体例の変更と、`AgentNotFoundError`、`AgentKilledError`、`AgentClientAlreadyBoundError` の 3 個の TypeSpec 宣言の追加を報告した。起票時の方針（実装を宣言済みの 400 へ寄せる）は、API ガイドラインの「400 と 422 の区別」「一意性違反と状態の競合」、および既存 API の 404 の宣言と食い違っていた。このため、ガイドラインと既存 API に合わせて宣言を広げる方向へ改めた。Agent 管理 API 8 operation に 404、409、422 を宣言し、状態変更 4 operation には欠けていた 401 と 403 の union も宣言した。実装の変更は、本文の `client_id` が見つからない拒否を 404 から 422 へ移した 1 箇所だけである。`EX-IDMANAGEMENT-009-04` は `OAuth2ClientNotFoundError` と、存在しない `client_id` と同じ応答であることを要求する形へ訂正し、負債台帳から外した。
- **Primary Use Case Evidence**:
  - id: bind-foreign-tenant-credential
    unit_red: "N/A: ユースケースはテナントで絞ってクライアントを検索しており、`TestBindCredentialRejectsClientOfAnotherTenant` は追加時点から GREEN だった。"
    e2e_red: "`TestBindingAForeignTenantsCredentialLeavesTheAgentUnbound` は実装変更前に `越境した client_id のバインドが status=404 ... want 422 client_not_found` で失敗した。"
    unit_fault_injection: "`BindCredential` のクライアント検索を default テナントに固定すると、`TestBindCredentialRejectsClientOfAnotherTenant` は `expected ErrAgentClientNotFound, got <nil>` で失敗した。"
    e2e_fault_injection: "同じ故障で `TestBindingAForeignTenantsCredentialLeavesTheAgentUnbound` は `status=204 body=, want 422 client_not_found` で失敗した。"
  - id: agent-refusal-statuses
    unit_red: "N/A: 停止済み Agent の拒否は実装済みであり、`TestKillAgentIsIrreversible` は既存の GREEN のテストである。"
    e2e_red: "N/A: 404 と 409 の写像は実装済みであり、`TestAgentRefusalsUseDeclaredStatuses` は既存の振る舞いを宣言どおりに固定した。"
    unit_fault_injection: "`SetAgentDisabled` から停止済みの判定を外すと、`TestKillAgentIsIrreversible` は `expected ErrAgentKilled on enable-after-kill, got <nil>` で失敗した。"
    e2e_fault_injection: "`agent_killed` を 400、`agent_not_found` を 403、`agent_client_already_bound` を 422 へ書き換えると、`TestAgentRefusalsUseDeclaredStatuses` の該当サブテストがそれぞれ失敗した。`mise run test-go-mutation -- ./backend/idmanagement/agent/handlers_http` は、境界テストが別パッケージにあるため 37 変異すべてを未被覆と報告した。このため写像の故障は手で注入した。"
- **Verification Results**:
  - `mise run check-spec`、`mise run check-api-compat`、`mise run check-status-drift` - 成功
  - `mise run test-go-package -- ./backend/idmanagement/handlers_http`、`mise run test-go-package -- ./backend/idmanagement/agent/usecases`、`mise run lint-go` - 成功
  - `mise run verify` - 成功
  - `mise run test-ui-e2e` - 未実行。フロントエンドはこれらのエラーコードを参照しておらず、変更はブラウザーへ届かない。
- **Left Undone**:
  - `mise run check-status-drift` は、Agent の 9 operation を `writeAdminAgentError` と `changeAgentStatus` の中まで読めないままである。API ガイドラインがヘルパーを追跡しないと定めているため、写像は `TestAgentRefusalsUseDeclaredStatuses` で固定した。
  - 越境した `client_id` と存在しない `client_id` の応答を書き分ける故障は、ユースケースが両者を区別しないため注入していない。
