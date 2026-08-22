namespace IdMagic.Contract {

@doc("The canonical list of scopes an API access token may carry, written as `<resource>:<action>` (read for queries, write for changes). The `scim:*` scopes belong to the SCIM 2.0 inbound provisioning API alone and split users and groups against read and write. The SCIM discovery endpoints (ServiceProviderConfig, ResourceTypes, Schemas) are readable with any `scim:*` scope. The Application aggregate and its protocol bindings sit under `applications:*`, while the standalone protocol and resource management APIs split into `oauth-clients:*`, `authorization-detail-types:*`, `mcp-resource-servers:*`, `saml:*`, `wsfed:*`, `provisioning:*`, `workload-identity:*`, and `shared-signals:*`. The `account:*` scopes belong to the self-service API of the token's own issuer, separating read from write and carving out the sensitive facets of MFA, sessions, consent, and passwords. Public, protocol, step-up, and email verification endpoints take no scope at all.")
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


@doc("The management record of a tenant-scoped JWT access token, which authenticates the administration API, the SCIM API, and the issuer's own account API. A token is not authorized once its scopes are empty, once `expires_at` has passed, or once `revoked_at` is set. The JWT itself is returned once, at issuance, and never stored.")
model ApiToken {
  tenant_id: string;
  @doc("The active User that issued the token. An `account:*` operation always acts on this User and no other.")
  user_id: string;
  id: string;
  jti: string;
  @doc("The built-in public client id the realm owns. Issuing a token through the administration API does not require an OAuth client to be registered first.")
  client_id: string;
  scopes: ApiTokenScope[];
  audience: string;
  dpop_jkt?: string;
  description: string;
  created_at: utcDateTime;
  expires_at?: utcDateTime;
  revoked_at?: utcDateTime;
}


@doc("The token metadata returned by the list and issue responses. It never carries the JWT itself.")
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


@doc("The authentication context resolved from a valid API access token: the tenant, the issuer's subject, and the scopes.")
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
