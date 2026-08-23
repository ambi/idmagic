---
status: cancelled
authors: [tn]
risk: low
created_at: 2026-08-08
depends_on: [wi-49-agent-identity-first-class-principal]
completion:
  completed_at: 2026-08-16
  summary: |
    本 work item が立てた二択は [[wi-372-admin-api-granular-scope-enforcement]] (2026-08-15 完了) が
    包含して決着させたため、実施せずに中止する。wi-372 は `IdManagement` コンテキスト全体へ
    `ManagementApiClient` 系 principal を導入し、Agent 管理 interface に `agents:read` / `agents:write`
    を配線した。結果は `docs/contexts/identity-management/SPECIFICATION.md` の REQ-IDMANAGEMENT-025 が
    正本として保持している。本 work item が明示的な判断を求めた `KillAgent` の扱いも wi-372 の Design で
    確定済みで、結論は「`agents:write` に含める」。理由は、削除が既に同じスコープにあり、削除はキルより
    厳密に破壊的であるため、キルだけを外しても防御にならないこと、および漏洩時の懸念よりも
    侵害された Agent の自動化された即時停止という正当な用途を優先すべきことである。
    Motivation が「全体設計は別 work item とする」と書いた相手が wi-372 そのものであった。
  verification:
    - cmd: rg -n '^### REQ-IDMANAGEMENT-025' docs/contexts/identity-management/SPECIFICATION.md
      result: 管理 API クライアントの粒度別スコープ制御シナリオが宣言済みであることを確認した。
---

# Agent 管理 API を `agents:read`/`agents:write` scope 付き API access token からも操作可能にする(または scope を削除する)

## Motivation

`docs/contexts/api-tokens/SPECIFICATION.md` の `ApiTokenScope` には `agents:read`/`agents:write` が正準 scope として定義されているが、`docs/contexts/identity-management/SPECIFICATION.md` の Agent 管理系 interface(`ListAgents`/`GetAgent`/`RegisterAgent`/`UpdateAgent`/`DisableAgent`/`EnableAgent`/`KillAgent`/`DeleteAgent`/`BindAgentCredential`/`UnbindAgentCredential`)は `TenantAdministrator`(人間の管理者セッション)のみで認可されており、これらの scope を消費する `ManagementApiClient` 系ポリシーが存在しない。

`WorkloadIdentity`(`ManagementApiClientReadWorkloadIdentity`/`ManagementApiClientWriteWorkloadIdentity`)、MCP リソースサーバー、`AuthorizationDetailType` 管理などは同種の scope を実際に配線しており、API access token(`api_token`/`oauth2_access_token`)経由での運用自動化が可能になっている。Agent 管理だけがこのパターンから外れており、`agents:read`/`agents:write` scope を持つ API access token を発行しても Agent の一覧・登録・kill 等はできない。

なお調査の過程で、`identity-management.yaml` の `authorization` ブロックには `TenantAdministrator`/`AuthenticatedSelf`/`SelfApiClient` の3principalしか定義されておらず、`ManagementApiClient` 系principal自体が存在しないことが分かった。つまりこれは Agent 固有の欠落ではなく、`IdManagement` コンテキスト全体(User/Group/Session/Consent/LifecycleWorkflow 管理)が API access token からの管理操作に対応していないという、より広い設計状況の一部である。本 work item ではその全体設計をやり直すのではなく、直近の調査で見つかった Agent 管理の scope 未使用という具体的な不整合の解消(配線するか、scope を削除するか)にスコープを絞る。

## Scope

- `docs/contexts/identity-management/SPECIFICATION.md`
  - `authorization.principals` に `ManagementApiClient`(type: Agent、`WorkloadIdentity` コンテキストの定義を踏襲)を追加するかどうかの決定。
  - 追加する場合: `authorization.policies` に `ManagementApiClientReadAgents`(`agents:read`)/`ManagementApiClientWriteAgents`(`agents:write`)を追加し、該当 interface の `access.policies` に配線する。
  - `KillAgent`(一方向・取り消し不能な kill-switch)を `agents:write` に含めるか、`TenantAdministrator` 限定のまま残すかを明示的に決定し、理由を記録する。
  - 不採用の場合: `docs/contexts/api-tokens/SPECIFICATION.md` の `ApiTokenScope` から `agents:read`/`agents:write` を削除する。
- 判断根拠は本ファイルの `## Design` に記録する。

## Out of Scope

- `IdManagement` コンテキスト全体(User/Group/Session/Consent/LifecycleWorkflow)への `ManagementApiClient` 配線。将来必要になった場合は別 work item とする。
- Agent の fail-closed ステータスゲート(`Active`/`Disabled`/`Killed`)や `AgentCredentialBinding` の再設計(`wi-49`/`wi-60` で確立済み)。
- `WorkloadIdentity` コンテキスト側の scope/policy 変更。

## Design

- 検討する2案:
  1. **配線する**: 運用自動化(CI/CD からの Agent 登録、監視ツールからの一覧参照など)を想定し、`ManagementApiClientReadAgents`/`ManagementApiClientWriteAgents` を他コンテキストと同一パターンで追加する。`KillAgent` は破壊力が大きいため、`agents:write` に含めるか `TenantAdministrator` 限定に残すかを別途判断する(推奨: 含める場合も監査ログの充実を前提とする。含めない場合は kill-switch は人間の管理者操作に限定する設計として明記する)。
  2. **削除する**: Agent 管理は常に人間の管理者判断を要する操作とみなし、`agents:read`/`agents:write` scope 自体を `ApiTokenScope` から削除する。
- 実装時にどちらを採るかは、実際の運用自動化ニーズ(Agent 登録・kill を人手を介さず行いたいユースケースがあるか)を確認して決定する。本 work item 時点では判断を確定しない。

## Plan

- 未定(上記 Design の2択を実装セッションの冒頭で確定させてから着手する)。
- 配線する場合: `docs/contexts/identity-management/SPECIFICATION.md` の specification 変更 → `just spec-render` → 対応する Go 実装(`backend/idmanagement/agent/`)への authorizer/ルーティング変更 → テスト。
- 削除する場合: `docs/contexts/api-tokens/SPECIFICATION.md` の specification 変更のみ(scope が他で未使用であることを grep で確認してから削除)。

## Tasks

- [ ] T001 [Decision] 配線する/しないを決定し、本ファイルの `## Design` に確定した結論と根拠を追記する。
- [ ] T002 [Spec] 決定に応じて `docs/contexts/identity-management/SPECIFICATION.md` または `docs/contexts/api-tokens/SPECIFICATION.md` を更新する。
- [ ] T003 [App] (配線する場合のみ) `backend/idmanagement/agent/` の authorizer/ルーティングに scope チェックを実装する。
- [ ] T004 [Verify] `just check-scl` / `just verify-spec` を通す。配線する場合は Go テストも追加・実行する。

## Verification

- `just check-scl`
- `just verify-spec`
- 配線する場合: `agents:read`/`agents:write` scope のみを持つ API access token で Agent 管理系エンドポイントを叩き、許可される操作と拒否される操作(scope 不足、他 tenant の Agent への操作)を確認する統合テストを追加する。
- 削除する場合: `grep -rn "agents:read\|agents:write"` でリポジトリ全体に参照が残っていないことを確認する。

## Risk Notes

- リスクは low。scope の配線または削除のみで、既存の `TenantAdministrator` 経路や fail-closed ステータスゲートには影響しない。
- 誤って `KillAgent` のような破壊的操作を安易に `agents:write` に含めると、漏洩した API access token 1本で全 Agent が停止させられるリスクが生まれるため、Design 決定時に必ず明示的に検討すること。
