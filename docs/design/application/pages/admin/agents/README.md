# Agent 管理

この文書は、Agent 管理のページ一覧を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[UI ページ](../../README.md) / [管理コンソール](../README.md)

## ページ一覧

| ページ | URL | 目的 |
| --- | --- | --- |
| [Agent 一覧](../../../../../modules/identity-management/pages/admin-agents.md) | `/admin/agents` | Agent を選び、状態と責任者を確認する |
| [Agent の棚卸し](../../../../../modules/identity-management/pages/admin-agent-inventory.md) | `/admin/agents/inventory` | 責任者、資格情報、利用状態の問題を見つける |
| [Agent を登録](../../../../../modules/identity-management/pages/admin-agent-create.md) | `/admin/agents/new` | 責任者と初期の資格情報・アクセスを指定する |
| [Agent 詳細](../../../../../modules/identity-management/pages/admin-agent-detail.md) | `/admin/agents/$agentId` | Agent の責任者、状態と利用を管理する |
| [Agent を編集](../../../../../modules/identity-management/pages/admin-agent-edit.md) | `/admin/agents/$agentId/edit` | 表示情報と責任者を更新する |
| [Agent の資格情報](../../../../../modules/identity-management/pages/admin-agent-credentials.md) | `/admin/agents/$agentId/credentials` | 認証に使う資格情報を作成または関連付ける |
| [Agent のリソース](../../../../../modules/identity-management/pages/admin-agent-resources.md) | `/admin/agents/$agentId/resources` | リソースへの割り当てを確認する |
| [Agent のリソースを編集](../../../../../modules/identity-management/pages/admin-agent-resources-edit.md) | `/admin/agents/$agentId/resources/edit` | 対象リソースと関係の割り当てを変更する |
| [Agent のアクセス概要](../../../../../modules/identity-management/pages/admin-agent-access-summary.md) | `/admin/agents/$agentId/access` | Agent に関連する認証、委任、リソースへの権限を確認する |
| [MCP リソースサーバー](../../../../../modules/oauth2/pages/admin-mcp-resource-servers.md) | `/admin/mcp-resource-servers` | MCP の接続先、認証とアクセス規則を管理する |
| [MCP リソースサーバーを登録](../../../../../modules/oauth2/pages/admin-mcp-resource-server-create.md) | `/admin/mcp-resource-servers/new` | 接続先と認証・アクセス規則を登録する |
| [MCP リソースサーバーを編集](../../../../../modules/oauth2/pages/admin-mcp-resource-server-edit.md) | `/admin/mcp-resource-servers/$resourceServerId/edit` | 接続先とツール・アクセス規則を更新する |
| [MCP サーバーの Agent アクセス](../../../../../modules/oauth2/pages/admin-mcp-agent-access.md) | `/admin/mcp-resource-servers/$resourceServerId/access` | 接続可能な Agent とアクセス設定を確認する |
| [Agent のガードレールを編集](../../../../../modules/oauth2/pages/admin-agent-guardrail-edit.md) | `/admin/agents/$agentId/guardrail/edit` | Agent 固有の制約を保存し、継承するデフォルト値と区別する |
