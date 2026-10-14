---
status: cancelled
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# Agent 管理 API を `agents:read`/`agents:write` scope 付き API access token からも操作可能にする(または scope を削除する)

`docs/domain/api-tokens/SPECIFICATION.md` の `ApiTokenScope` には `agents:read`/`agents:write` が正準 scope として定義されているが、`docs/domain/identity-management/SPECIFICATION.md` の Agent 管理系 interface(`ListAgents`/`GetAgent`/`RegisterAgent`/`UpdateAgent`/`DisableAgent`/`EnableAgent`/`KillAgent`/`DeleteAgent`/`BindAgentCredential`/`UnbindAgentCredential`)は `TenantAdministrator`(人間の管理者セッション)のみで認可されており、これらの scope を消費する `ManagementApiClient` 系ポリシーが存在しない。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。
