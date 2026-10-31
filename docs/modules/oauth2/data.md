# OAuth2 のデータ

この文書は、OAuth2 の Aggregate を扱う。

## 集約

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `OAuth2Client` | クライアントのメタデータ、認証方式、クライアントシークレットの資格情報、`ClaimMappingPolicy` | `Tenant` を `tenant_id` で参照する。`Application` から参照される |
| `Consent` | `(subject, client_id)` ごとの付与済みのスコープの集合 | `OAuth2Client` と User を参照する |
| `AuthorizationRequest`、`AuthorizationCodeRecord`、`PARRecord` | 認可の要求、認可コード、プッシュした要求 | `OAuth2Client` を参照する |
| `DeviceAuthorization` | デバイスコードとユーザーコード | `OAuth2Client` を参照する |
| `RefreshTokenRecord` | リフレッシュトークンとそのファミリー | `OAuth2Client` と User を参照する |
| `ApprovalRequest` | 人間の承認の判断と CIBA のポーリングの記録 | `OAuth2Client` と User を参照する |
| `McpResourceServer`、`AuthorizationDetailType` | MCP のリソースサーバーと、`authorization_details` の型 | `Tenant` を `tenant_id` で参照する |
| `AgentGuardrailPolicy` | Agent の発行の上限とテナントのデフォルト | `Tenant` を `tenant_id` で、`Agent` を `agent_id` で参照し、Agent の削除で消える |
| `CrossAppAccessIssuePolicy`、`CrossAppAccessRedeemPolicy` | ID-JAG の発行と受け入れの許可 | `Tenant` を `tenant_id` で参照する。クライアントは `client_id` で、接続は Authentication の `IdentityProviderConnection` を `provider_id` で参照する |

`ClaimMappingPolicy` は `OAuth2Client` の内側の値であり、独立した Aggregate ではない。
