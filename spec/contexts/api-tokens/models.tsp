namespace IdMagic.Contract {

@doc("API アクセストークンに付与できる scope の正準リスト。<resource>:<action> 形式 (read=参照系 / write=変更系)。scim:* は SCIM 2.0 inbound provisioning API 専用の scope で、users/groups × read/write に分ける。SCIM discovery endpoint (ServiceProviderConfig / ResourceTypes / Schemas) は scim:* のいずれかを持てば 参照できる。Application aggregate とその protocol binding は applications:*、独立した protocol/resource 管理 API は oauth-clients:* / authorization-detail-types:* / mcp-resource-servers:* / saml:* / wsfed:* / provisioning:* / workload-identity:* / shared-signals:* に分ける。アカウント自己管理 account:* は発行者本人の自己管理 API 専用で、read / write と MFA / session / consent / password の sensitive facet を分離する。公開・プロトコル・step-up・email verification endpoint は scope 対象外。")
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


@doc("管理用 API / SCIM API / 発行者本人の account API を認証する tenant-scoped JWT アクセストークンの管理 record。scopes が空、expires_at を過ぎた、または revoked_at が設定された token は認可されない。JWT 本文は発行時に一度だけ返し永続化しない。")
model ApiToken {
  tenant_id: string;
  @doc("token を発行した active User。account:* 操作の対象は常にこの User に固定する。")
  user_id: string;
  id: string;
  jti: string;
  @doc("realm が所有する built-in public client id。管理発行では OAuth client の事前登録を要求しない。")
  client_id: string;
  scopes: ApiTokenScope[];
  audience: string;
  dpop_jkt?: string;
  description: string;
  created_at: utcDateTime;
  expires_at?: utcDateTime;
  revoked_at?: utcDateTime;
}


@doc("一覧・発行応答で返すトークンのメタデータ。JWT 本文を含まない。")
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


@doc("有効な API アクセストークンから解決した tenant、発行者 subject、scope の認証コンテキスト。")
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
