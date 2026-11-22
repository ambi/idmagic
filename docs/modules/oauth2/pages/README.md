# OAuth2 のページ

この文書は、OAuth2 が主担当となるページの設計を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

[OAuth2 のモジュール設計](../README.md) / [UI 全体の索引](../../../design/application/pages/README.md)

## 同意と承認

| ページ | URL | 目的 |
| --- | --- | --- |
| [同意と Agent 委任の管理](admin-consents.md) | `/admin/consents` | 利用者の同意を確認して取り消し、Agent への委任を確認する |
| [アプリケーションへの同意](consent.md) | `/consent` | 開示する情報と認可詳細を確認して許可または拒否する |
| [デバイスの認可](device.md) | `/device` | デバイスのコードを指定して認可を許可または拒否する |
| [本人の同意と Agent への委任](account-applications.md) | `/account/applications` | アプリケーションへの同意を取り消し、Agent の委任を管理する |
| [本人宛ての承認依頼](account-approvals.md) | `/account/approvals` | Agent の実行や認可の依頼を確認して承認または拒否する |

## 認可

| ページ | URL | 目的 |
| --- | --- | --- |
| [アプリ間アクセスの規則](admin-cross-app-access.md) | `/admin/cross-app-access` | アプリ間の発行と受け入れの規則を管理する |
| [アプリ間の発行規則を追加](admin-cross-app-access-new-issue.md) | `/admin/cross-app-access/new-issue` | 対象アプリと発行できる範囲を登録する |
| [アプリ間の受け入れ規則を追加](admin-cross-app-access-new-redeem.md) | `/admin/cross-app-access/new-redeem` | 受け入れる発行者と利用範囲を登録する |
| [ロール一覧](admin-roles.md) | `/admin/roles` | 定義済みのロールと権限、付与人数を確認する |
| [ロール詳細](admin-role-detail.md) | `/admin/roles/$name` | 一つのロールの権限と付与先を確認する |
| [認可詳細の型](admin-authz-detail-types.md) | `/admin/authorization-detail-types` | 認可詳細のスキーマと利用するアプリを管理する |
| [認可詳細の型を作成](admin-authz-detail-type-create.md) | `/admin/authorization-detail-types/new` | 型の識別子と JSON Schema を登録する |
| [認可詳細の型を編集](admin-authz-detail-type-edit.md) | `/admin/authorization-detail-types/$type/edit` | 既存の型の説明とスキーマを変更する |

## Agent 管理

| ページ | URL | 目的 |
| --- | --- | --- |
| [MCP リソースサーバー](admin-mcp-resource-servers.md) | `/admin/mcp-resource-servers` | MCP の接続先、認証とアクセス規則を管理する |
| [MCP リソースサーバーを登録](admin-mcp-resource-server-create.md) | `/admin/mcp-resource-servers/new` | 接続先と認証・アクセス規則を登録する |
| [MCP リソースサーバーを編集](admin-mcp-resource-server-edit.md) | `/admin/mcp-resource-servers/$resourceServerId/edit` | 接続先とツール・アクセス規則を更新する |
| [MCP サーバーの Agent アクセス](admin-mcp-agent-access.md) | `/admin/mcp-resource-servers/$resourceServerId/access` | 接続可能な Agent とアクセス設定を確認する |
| [Agent のガードレールを編集](admin-agent-guardrail-edit.md) | `/admin/agents/$agentId/guardrail/edit` | Agent 固有の制約を保存し、継承するデフォルト値と区別する |
