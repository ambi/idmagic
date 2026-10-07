namespace IdMagic.Contract {

@doc("API アクセストークンに付与できるスコープの正式な一覧。`<resource>:<action>` の形で書き、参照には read、変更には write を使う。`scim:*` のスコープは SCIM 2.0 の受信プロビジョニング API だけに属し、ユーザーとグループを read と write に分ける。SCIM のディスカバリーエンドポイント（ServiceProviderConfig、ResourceTypes、Schemas）は、どの `scim:*` スコープでも読める。Application 集約とそのプロトコルのバインディングは `applications:*` の下にあり、単独のプロトコルとリソースの管理 API は `oauth-clients:*`、`authorization-detail-types:*`、`mcp-resource-servers:*`、`saml:*`、`wsfed:*`、`provisioning:*`、`workload-identity:*`、`shared-signals:*` に分かれる。`account:*` のスコープはトークンの発行者自身のセルフサービス API に属し、read と write を分けたうえで、MFA、セッション、同意、パスワードという機微な側面を個別のスコープにする。公開、プロトコル、ステップアップ、メール確認のエンドポイントはスコープを求めない。")
enum ApiTokenScope {
  users_read: "users:read",
  users_write: "users:write",
  groups_read: "groups:read",
  groups_write: "groups:write",
  agents_read: "agents:read",
  agents_write: "agents:write",
  sessions_read: "sessions:read",
  sessions_write: "sessions:write",
  consents_read: "consents:read",
  consents_write: "consents:write",
  lifecycle_workflows_read: "lifecycle-workflows:read",
  lifecycle_workflows_write: "lifecycle-workflows:write",
  tenants_read: "tenants:read",
  tenants_write: "tenants:write",
  settings_read: "settings:read",
  settings_write: "settings:write",
  signing_keys_read: "signing-keys:read",
  signing_keys_write: "signing-keys:write",
  audit_read: "audit:read",
  jobs_read: "jobs:read",
  jobs_cancel: "jobs:cancel",
  applications_read: "applications:read",
  applications_write: "applications:write",
  oauth_clients_read: "oauth-clients:read",
  oauth_clients_write: "oauth-clients:write",
  authorization_detail_types_read: "authorization-detail-types:read",
  authorization_detail_types_write: "authorization-detail-types:write",
  mcp_resource_servers_read: "mcp-resource-servers:read",
  mcp_resource_servers_write: "mcp-resource-servers:write",
  saml_read: "saml:read",
  saml_write: "saml:write",
  wsfed_read: "wsfed:read",
  wsfed_write: "wsfed:write",
  provisioning_read: "provisioning:read",
  provisioning_write: "provisioning:write",
  workload_identity_read: "workload-identity:read",
  workload_identity_write: "workload-identity:write",
  shared_signals_read: "shared-signals:read",
  shared_signals_write: "shared-signals:write",
  scim_users_read: "scim:users:read",
  scim_users_write: "scim:users:write",
  scim_groups_read: "scim:groups:read",
  scim_groups_write: "scim:groups:write",
  account_read: "account:read",
  account_write: "account:write",
  account_mfa_write: "account:mfa:write",
  account_sessions_write: "account:sessions:write",
  account_consents_write: "account:consents:write",
  account_password_write: "account:password:write",
}


@doc("テナントに属する JWT アクセストークンの管理用の記録。このトークンは管理 API、SCIM API、発行者自身のアカウント API の認証に使う。スコープが空になったとき、`expires_at` を過ぎたとき、`revoked_at` が設定されたときは認可しない。JWT そのものは発行時に一度だけ返し、保存しない。")
model ApiToken {
  tenant_id: string;
  @doc("トークンを発行した有効な User。`account:*` の操作は常にこの User だけに作用する。")
  user_id: string;
  id: string;
  jti: string;
  @doc("レルムが所有する組み込みの公開クライアントの id。管理 API からトークンを発行するときに、OAuth クライアントを先に登録する必要はない。")
  client_id: string;
  scopes: ApiTokenScope[];
  audience: string;
  dpop_jkt?: string;
  description: string;
  created_at: utcDateTime;
  expires_at?: utcDateTime;
  revoked_at?: utcDateTime;
}


@doc("一覧と発行のレスポンスが返すトークンのメタデータ。JWT そのものは含まない。")
model ApiTokenMetadata {
  id: string;
  jti: string;
  description: string;
  scopes: ApiTokenScope[];
  user_id: string;
  audience: string;
  dpop_jkt?: string;
  created_at: utcDateTime;
  expires_at?: utcDateTime;
  revoked_at?: utcDateTime;
}


@doc("有効な API アクセストークンから解決した認証コンテキスト。テナント、発行者のサブジェクト、スコープからなる。")
model ApiTokenPrincipal {
  tenant_id: string;
  user_id: string;
  scopes: ApiTokenScope[];
  client_id: string;
  audience: string;
  token_id: string;
  issued_at: utcDateTime;
  expires_at?: utcDateTime;
  dpop_jkt?: string;
}

}
