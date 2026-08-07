---
status: completed
authors: [claude]
risk: low
created_at: 2026-08-08
depends_on: [wi-49-agent-identity-first-class-principal, wi-60-agent-security-correctness-hardening]
---

# SCL を Agent 関連の既実装不変条件(Killed 削除拒否・client_id 一意性)に同期する

## Motivation

`work-items/done/wi-60-agent-security-correctness-hardening.md` により、以下2点は Go 実装レベルでは既に修正・強制されている:

1. `Killed` ステータスの Agent は `DeleteAgent` で削除できない(`backend/idmanagement/agent/usecases/admin_agents.go` の `DeleteAgent` が `agent.Status == idmdomain.AgentStatusKilled` を検査し `ErrAgentKilled` を返す)。削除を許すと束縛先 `OAuth2Client` が kill-switch を経由せず client_credentials で token を再取得できてしまうため。
2. `AgentCredentialBinding` は `client_id` が複数の Agent に重複束縛されない(Postgres `infra/schema/postgres.sql` の `agent_credential_bindings` テーブルに `UNIQUE (client_id)` 制約があり、`BindCredential` usecase も `FindByClientID` で事前チェック + 挿入後の再チェックを行う)。

しかし `spec/contexts/identity-management.yaml` はこの2つの不変条件を一切記述していなかった(`DeleteAgent` に `requires` ガードなし、`AgentCredentialBinding` のフィールド定義に一意性の言及なし)。Regenerative Architecture は SCL を正本として実装が追従する方針だが、ここでは逆に実装が先行し SCL が追いついていない状態だった。安全上の実害はない(コードは正しく fail-closed に振る舞っている)が、SCL だけを読んで挙動を再現しようとする第三者・将来の実装は Killed Agent 削除や client_id 重複束縛を許してしまう誤った仕様を組んでしまう。

## Scope

- `spec/contexts/identity-management.yaml`
  - `DeleteAgent` interface に `requires: [resource.status != "Killed"]` を追加し、description に理由を明記。
  - `AgentCredentialBinding` entity の description に「client_id は他の Agent と重複して束縛できない」旨を追記。
  - `BindAgentCredential` interface に `requires: [not agentCredentialBindingExistsForClient(context.tenant_id, input.request.client_id)]` を追加。

## Out of Scope

- Go 実装・DB migration の変更(既に正しく実装されている。今回は SCL 側のドキュメント/契約としての同期のみ)。
- `wi-60` で修正されたその他の項目(owner_sub 検証、`requested_token_type` 検証など)の SCL 記述状況の網羅的な監査。今回は Killed 削除拒否と client_id 一意性の2点に限定。

## Design

- SCL の `requires` 式は本コンテキスト内の既存パターン(`resource.status`、`in`/`not` 演算、named predicate 呼び出し)にそのまま従う。新しい形式や DSL 拡張は導入しない。
- `resource.status != "Killed"` は `access.resource.type: Agent, id: input.agent_id` により `resource` が対象 Agent エンティティに解決される既存の authorization 機構をそのまま利用する(`spec/contexts/provisioning.yaml` の `resource.status == "dead_letter"` と同型)。
- `agentCredentialBindingExistsForClient(...)` は `spec/contexts/workloadidentity.yaml` の `trust_bundle_issuer_unique_in_tenant(...)` と同じ、SCL 内で非形式的に使われる述語呼び出し規約(専用の `functions:` ブロックは本リポジトリの SCL には存在せず、命名から意図を読み取る運用)に倣った。

## Tasks

- [x] T001 [SCL] `DeleteAgent` に Killed 拒否の `requires` を追加する。
- [x] T002 [SCL] `AgentCredentialBinding` の description に client_id 一意性を明記する。
- [x] T003 [SCL] `BindAgentCredential` に client_id 重複拒否の `requires` を追加する。
- [x] T004 [Verify] `just check-scl` / `just verify-spec` を通す。

## Verification

- `just check-scl` — 実施済み、`spec/contexts/identity-management.yaml` を含む全 SCL ファイルが ok。
- 実装側 (`backend/idmanagement/agent/usecases/admin_agents.go:334-345` の `DeleteAgent`、同ファイル `385-406` 付近の `BindCredential`、`infra/schema/postgres.sql:685-699` の `UNIQUE (client_id)`)を確認し、追加した SCL の `requires` が実装済み挙動と完全に一致することを確認済み(本 work item のための事前調査で確認)。

## Risk Notes

- リスクは low。SCL への記述追加のみで、実装(Go/DB)は変更しない。追加した `requires` は既存実装の挙動をそのまま記述したものであり、新しい制約を課すものではない。

## Completion

- **Completed At**: 2026-08-08
- **Summary**:
  `spec/contexts/identity-management.yaml` の `DeleteAgent` に Killed ステータスの Agent 削除を拒否する `requires: [resource.status != "Killed"]` を追加し、`BindAgentCredential` に client_id の重複束縛を拒否する `requires` を追加した。`AgentCredentialBinding` の description にも一意性の言及を追記した。いずれも `backend/idmanagement/agent/usecases/admin_agents.go`(`DeleteAgent`/`BindCredential`)と `infra/schema/postgres.sql`(`agent_credential_bindings_client_id_key` UNIQUE 制約)で既に強制されている挙動を SCL に反映しただけであり、実装・DB migration の変更は行っていない。
- **Verification Results**:
  - `just check-scl` - passed
