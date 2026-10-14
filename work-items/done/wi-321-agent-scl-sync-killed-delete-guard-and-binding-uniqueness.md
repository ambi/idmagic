---
status: completed
archived_from: 794133221eda13d8414657d52ad3790398ca5cea
---

# SCL を Agent 関連の既実装不変条件(Killed 削除拒否・client_id 一意性)に同期する

`spec/contexts/identity-management.yaml` の `DeleteAgent` に Killed ステータスの Agent 削除を拒否する `requires: [resource.status != "Killed"]` を追加し、`BindAgentCredential` に client_id の重複束縛を拒否する `requires` を追加した。`AgentCredentialBinding` の description にも一意性の言及を追記した。いずれも `backend/idmanagement/agent/usecases/admin_agents.go`(`DeleteAgent`/`BindCredential`)と `infra/schema/postgres.sql`(`agent_credential_bindings_client_id_key` UNIQUE 制約)で既に強制されている挙動を SCL に反映しただけであり、実装・DB migration の変更は行っていない。

全文は `git show 794133221eda13d8414657d52ad3790398ca5cea:<このファイルのパス>` で読む。
