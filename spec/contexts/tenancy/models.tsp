namespace IdMagic.Contract {

@doc("The per-tenant resource ceilings, hard and soft. A system administrator may override them per tenant.")
model TenantQuota {
  users: int64;
  groups: int64;
  agents: int64;
  applications: int64;
  oauth2_clients: int64;
  active_sessions: int64;
  consents: int64;
  active_jobs: int64;
  @doc("The ceiling on SsfStreams, transmit and receive counted together.")
  ssf_streams: int64;
  audit_events_retained: int64;
  export_artifacts_bytes: int64;
}


@doc("How much of each resource a tenant currently uses.")
model TenantUsage {
  users: int64;
  groups: int64;
  agents: int64;
  applications: int64;
  oauth2_clients: int64;
  active_sessions: int64;
  consents: int64;
  active_jobs: int64;
  ssf_streams: int64;
  audit_events_retained: int64;
  export_artifacts_bytes: int64;
}


@doc("The JSON request in which a system_admin overrides a tenant's Quota.")
model TenantQuotaUpdateRequest {
  users?: int64;
  groups?: int64;
  agents?: int64;
  applications?: int64;
  oauth2_clients?: int64;
  active_sessions?: int64;
  consents?: int64;
  active_jobs?: int64;
  ssf_streams?: int64;
  audit_events_retained?: int64;
  export_artifacts_bytes?: int64;
}


@doc("The tenant state. `active` runs normally; `disabled` stops the tenant accepting any protocol traffic, so `/authorize`, `/token`, and the rest all give invalid_request.")
enum TenantStatus {
  Active: "active",
  Disabled: "disabled",
}


@doc("The shape of a tenant's canonical location. A tenant has exactly one, and the issuer, the cookie scope, and the WebAuthn RP ID all follow from it. `path` is `{base}/realms/{realm}` on a shared host, and is the default because it needs neither wildcard DNS nor a certificate. `subdomain` is the tenant-specific host `{realm}.{tenant_base_domain}`, and it is what makes the features that need a tenant-specific origin work: `__Host-` cookies, a per-tenant WebAuthn RP ID, and a per-tenant branded origin. In a deployment where `tenant_base_domain` is unset, `subdomain` cannot be chosen.")
enum TenantEndpointStyle {
  Path: "path",
  Subdomain: "subdomain",
}


@doc("The per-tenant password policy overrides. It is the schema for replacing the standard PasswordPolicy values of the\nAuthentication context at the tenant resolution port. A field left out inherits the global default, and on the\nevaluation paths of change_password, reset_password, and administrative user creation a tenant value may only move in\nthe stricter direction from the global one.")
model PasswordPolicyOverride {
  @doc("The minimum length; at or above the global default is recommended.")
  min_length?: int64;
  @doc("The maximum length.")
  max_length?: int64;
  @doc("How many past passwords are checked against (PasswordHistoryNoReuse).")
  history_depth?: int64;
  @doc("How many days a password stays valid. Zero, or leaving it out, disables expiry, which is the default in line with\nNIST SP 800-63B-4. In a tenant that sets a positive value, a user whose password is past its expiry is directed into the\n`update_password` required action after their next sign-in succeeds; the authentication itself does not fail.\nThe system ceiling is 30 to 3650 days, and a value outside that is refused with PolicyOverrideWeakerError.")
  @minValue(0)
  max_age_days?: int64;
}

@doc("The tenant (realm) aggregate. Clients, Users, Consents, policies, and branding all stay inside this authorization boundary.")
model Tenant {
  @minLength(1)
  id: string;
  @minLength(1)
  @maxLength(63)
  @pattern("^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$")
  realm: string;
  @minLength(1)
  @maxLength(200)
  display_name: string;
  status: TenantStatus = TenantStatus.Active;
  endpoint_style: TenantEndpointStyle = TenantEndpointStyle.Path;
  password_policy_override?: PasswordPolicyOverride;
  @doc("Tenant override for the maximum Token Exchange (RFC 8693) `act` chain depth. When omitted, the system default applies.\nThe override may only tighten the limit; PolicyOverrideWeakerError rejects values above the system default.\nThis asymmetric rule matches password_policy_override and prevents tenant settings from widening an authorization boundary.")
  @minValue(1)
  max_delegation_depth?: int64;
  @doc("How many seconds a trusted device -- \"remember this device\" in the Authentication context -- stays valid.\nZero, or leaving it out, disables the feature for the whole tenant: no device is remembered and no second factor is skipped.\nThe system ceiling is 7,776,000 seconds (90 days), and a larger value is refused with PolicyOverrideWeakerError.")
  @minValue(0)
  trusted_device_max_age_seconds?: int64;
  @pattern("^[a-z]{2}$")
  default_locale?: string;
  created_at: utcDateTime;
  updated_at: utcDateTime;
  disabled_at?: utcDateTime;
}


@doc("The set of custom attribute definitions of one tenant. The built-in attributes live in code (BuiltinUserAttributeDefs), and this set holds only what the tenant added. It is a separate aggregate rather than something embedded in the tenant aggregate, and it cascades when the tenant is deleted.")
model TenantUserAttributeSchema {
  @doc("The tenant it belongs to.")
  tenant_id: string = "default";
  @doc("The definitions, whose keys are unique and collide with no built-in attribute.")
  attributes: UserAttributeDef[];
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("The set of Group custom attribute definitions of one tenant. Group has nothing corresponding to the built-in catalogue of TenantUserAttributeSchema, so this set is the effective definition in full. It is a separate aggregate rather than something embedded in the tenant aggregate, and it cascades when the tenant is deleted.")
model TenantGroupAttributeSchema {
  @doc("The tenant it belongs to.")
  tenant_id: string = "default";
  @doc("The definitions, whose keys are unique.")
  attributes: GroupAttributeDef[];
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("The current values of the PasswordPolicy of the Authentication context. They are embedded in the API response so the\ntenant settings UI can show the user, as concrete numbers, what actually applies when the tenant overrides nothing.\nThey are at the same time the threshold at which an override is rejected as weaker.")
model PasswordPolicyDefaults {
  @doc("The `min_length` of the PasswordPolicy of the Authentication context.")
  min_length: int64;
  @doc("The `max_length` of the PasswordPolicy of the Authentication context.")
  max_length: int64;
  @doc("The `history_depth` of the PasswordPolicy of the Authentication context.")
  history_depth: int64;
  @doc("The `max_age_days` of the PasswordPolicy of the Authentication context. Leaving it out means 0, no expiry, which\nmatches how an older client that does not know the field reads it.")
  max_age_days?: int64;
}


@doc("What `/api/admin/settings` returns. It is a projection for the administration UI rather than the tenant itself, and\nbesides the tenant_id (a UUID), the realm (a URL slug), the display name, and the password policy override it returns the\nstandard PasswordPolicy values of the Authentication context, used both for display and for the weaker-policy gate.")
model AdminSettingsResponse {
  @doc("The immutable UUID key of the tenant.")
  tenant_id: string;
  @doc("The slug that appears in the URL `/realms/{realm}/`. It is used to build the SCIM endpoint URLs and the like.")
  realm: string;
  display_name: string;
  password_policy_override?: PasswordPolicyOverride;
  password_policy_defaults: PasswordPolicyDefaults;
  @doc("Tenant override for the maximum Token Exchange `act` chain depth. When omitted, the system default applies.")
  @minValue(1)
  max_delegation_depth?: int64;
  @doc("Delegation depth applied when the tenant has no override, and the ceiling that an override cannot exceed.\nThe response exposes this value so the administrative UI can display the concrete limit and reject an excessive value before sending it.")
  max_delegation_depth_default: int64;
  @doc("How many seconds a trusted device stays valid. Zero, or leaving it out, disables the feature.")
  @minValue(0)
  trusted_device_max_age_seconds?: int64;
  @doc("The ceiling on how many seconds a trusted device may stay valid. It is returned so the administration UI can state the\nlimit concretely and reject a larger value before sending it.")
  trusted_device_max_age_seconds_ceiling: int64;
  @doc("The tenant's default locale. Unset, the system default locale is used.")
  default_locale?: string;
  @doc("The locales the catalogue ships translations for. It is returned so the administration UI can build the choices for the default locale and for notification templates.")
  supported_locales: string[];
  quota: TenantQuota;
  usage: TenantUsage;
}


@doc("The canonical endpoints an administrator needs to configure a relying party or an API client against the OAuth 2.0 authorization server and OpenID Connect provider. The discovery and authorization server metadata are the machine-readable authority; the individual URLs are returned for configuring by hand.")
model AdminOAuthEndpointSet {
  openid_configuration: string;
  oauth_authorization_server: string;
  authorization_endpoint: string;
  token_endpoint: string;
  userinfo_endpoint: string;
  jwks_uri: string;
  revocation_endpoint: string;
  introspection_endpoint: string;
  end_session_endpoint: string;
  registration_endpoint: string;
  pushed_authorization_request_endpoint: string;
  device_authorization_endpoint: string;
}


@doc("The public part of the active X.509 certificate IdMagic uses to sign new XML federation messages. It carries no private key, and the metadata remains the authority for every certificate inside a rotation overlap.")
model AdminFederationCertificate {
  download_url: string;
  @doc("The SHA-256 fingerprint, as uppercase hex separated by colons.")
  fingerprint_sha256: string;
  not_before: utcDateTime;
  not_after: utcDateTime;
}


@doc("The canonical details of the tenant's shared SAML IdP, to feed into an SP.")
model AdminSamlIdentityProviderEndpointSet {
  entity_id: string;
  metadata_url: string;
  sso_url: string;
  slo_url: string;
  signing_certificate: AdminFederationCertificate;
}


@doc("The canonical details of the tenant, to feed into a WS-Federation RP or Microsoft Entra domain federation.")
model AdminWsFederationEndpointSet {
  realm: string;
  metadata_url: string;
  passive_logon_url: string;
  active_logon_url: string;
  metadata_exchange_url: string;
}


@doc("The canonical base URLs an external management client, a SCIM client, or a self-service client uses together with an API token.")
model AdminApiEndpointSet {
  management_api_base_url: string;
  scim_base_url: string;
  account_api_base_url: string;
}


@doc("The read-only projection of `/api/admin/integration-endpoints`. Everything is derived from the canonical issuer of the request tenant alone, so the UI needs to know nothing about paths, subdomains, or custom domains. It carries no client secret, token, or private key.")
model AdminIntegrationEndpointCatalog {
  issuer: string;
  oauth: AdminOAuthEndpointSet;
  saml: AdminSamlIdentityProviderEndpointSet;
  ws_federation: AdminWsFederationEndpointSet;
  apis: AdminApiEndpointSet;
}


@doc("What `/api/admin/tenant/user_attribute_schema` returns. Besides the tenant's own custom attribute definitions\n(`attributes`) it returns the built-in catalogue code holds (`builtin`) for reference, so the UI can list every attribute\nwhile avoiding a key collision.")
model TenantUserAttributeSchemaResponse {
  tenant_id: string;
  @doc("The custom definitions the tenant may edit.")
  attributes: UserAttributeDef[];
  @doc("The read-only catalogue code provides (BuiltinUserAttributeDefs).")
  builtin: UserAttributeDef[];
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("The JSON request replacing the tenant's custom attribute definitions outright. A key that collides with a built-in attribute, and a duplicate key, are both refused in the use case.")
model TenantUserAttributeSchemaUpdateRequest {
  @doc("The new list of custom definitions. An empty array deletes them all.")
  attributes: UserAttributeDef[];
}


@doc("What `/api/admin/tenant/group_attribute_schema` returns. Group has no built-in catalogue, so unlike the User version it carries no `builtin` field.")
model TenantGroupAttributeSchemaResponse {
  tenant_id: string;
  @doc("The Group custom definitions the tenant may edit.")
  attributes: GroupAttributeDef[];
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("The JSON request replacing the tenant's Group custom attribute definitions outright. A duplicate key is refused in the use case.")
model TenantGroupAttributeSchemaUpdateRequest {
  @doc("The new list of custom definitions. An empty array deletes them all.")
  attributes: GroupAttributeDef[];
}


@doc("The kinds of image asset a TenantBranding holds.")
enum TenantBrandingAssetKind {
  Logo: "logo",
  Favicon: "favicon",
}


@doc("A safe external link shown in the hosted UI footer, in a fixed order. The label is rendered as plain text and the URL may only be HTTPS. When one is present, both `label` and `url` have to be non-empty; giving only one of the two is refused.")
model TenantFooterLink {
  @doc("The link text shown to the user. It is never interpreted as HTML or Markdown.")
  @minLength(1)
  @maxLength(80)
  label: string;
  @doc("Where the link goes. Only the https scheme is on the allow list.")
  @minLength(1)
  @maxLength(2048)
  @pattern("^https://")
  url: string;
}


@doc("The per-tenant hosted UI branding settings. They are not embedded in the Tenant aggregate but held as a separate\nentity, for the same reason as TenantUserAttributeSchema: sparse per-tenant configuration is kept apart from the Tenant\nitself. Every field is optional, and a missing row, or a row with every field unset, means branding is unset and\nGetTenantBranding falls back to the system default.\nThe colours, links, and text accept only a limited set of tokens; arbitrary CSS or HTML cannot be accepted structurally.")
model TenantBranding {
  @doc("The tenant it belongs to.")
  tenant_id: string;
  @doc("The name shown on the sign-in screen and elsewhere. An empty string means unset, and the system default (IdMagic) is shown.")
  @maxLength(80)
  product_name?: string;
  @doc("The opaque key of the logo image object stored inside the tenant boundary. An administrator never enters it directly; it comes from UploadTenantBrandingAsset with `kind=Logo`.")
  logo_object_key?: string;
  @doc("The internal serving URL the IdP builds from the stored logo.")
  logo_url?: string;
  @doc("The opaque key of the favicon image object stored inside the tenant boundary. It comes from UploadTenantBrandingAsset with `kind=Favicon`.")
  favicon_object_key?: string;
  @doc("The internal serving URL the IdP builds from the stored favicon.")
  favicon_url?: string;
  @doc("The primary brand colour. It is injected only as the CSS custom property `--tenant-brand-primary`. Anything in `#rrggbb` form can be saved whatever its contrast ratio, and an empty string normalizes to unset and returns to the system default. Whether the palette is legible is the tenant's judgement.")
  @pattern("^#[0-9a-fA-F]{6}$")
  primary_color?: string;
  @doc("The accent brand colour. It is injected only as the CSS custom property `--tenant-brand-accent`. Anything in `#rrggbb` form can be saved whatever its contrast ratio, and an empty string normalizes to unset and returns to the system default. Whether the palette is legible is the tenant's judgement.")
  @pattern("^#[0-9a-fA-F]{6}$")
  accent_color?: string;
  @doc("The first footer link. The order is fixed as footer_link_1 then footer_link_2.")
  footer_link_1?: TenantFooterLink;
  @doc("The second footer link. The order is fixed as footer_link_1 then footer_link_2.")
  footer_link_2?: TenantFooterLink;
  @doc("The supplementary text shown in the footer, a copyright notice for instance. It is rendered as plain text, protected by nothing more than the renderer's default escaping.")
  @maxLength(280)
  footer_text?: string;
  created_at: utcDateTime;
  @doc("It advances on every update. It serves as the version and the ETag of the public response.")
  updated_at: utcDateTime;
}


@doc("The publicly safe projection GetTenantBranding, UpdateTenantBranding, and the asset upload and delete return.\nAn unset field stands for the system default, which the client resolves to `IdMagic` and the like.")
model TenantBrandingResponse {
  product_name?: string;
  logo_url?: string;
  favicon_url?: string;
  primary_color?: string;
  accent_color?: string;
  footer_link_1?: TenantFooterLink;
  footer_link_2?: TenantFooterLink;
  footer_text?: string;
  @doc("Used as the version and the ETag once branding has been set at least once. Left out while it is unset.")
  updated_at?: utcDateTime;
}


@doc("The JSON request in which an administrator updates the branding text, colours, and links. Image assets are handled separately, through UploadTenantBrandingAsset and DeleteTenantBrandingAsset. Only the fields given are changed, and passing null returns that field to unset. `primary_color` and `accent_color` return to unset on an empty string as well.")
model TenantBrandingUpdateRequest {
  product_name?: string;
  primary_color?: string;
  accent_color?: string;
  footer_link_1?: TenantFooterLink;
  footer_link_2?: TenantFooterLink;
  footer_text?: string;
}


@doc("How branding is represented after an image asset has been uploaded or deleted.")
model TenantBrandingUploadResponse {
  branding: TenantBrandingResponse;
}


@doc("A tenant administrator updated the branding: its text, colours, links, or image assets.")
model TenantBrandingUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  changedFields: string[];
}


@doc("The fixed identifier naming what a notification is for. The catalogue holds a built-in default template in Japanese and English for every one of these keys, and a tenant cannot add a key. AccountSecurityAlert covers every kind of security notification under one key, and what happened is expressed through the `event_description` and `device_summary` substitutions.")
enum NotificationTemplateKey {
  PasswordReset: "password_reset",
  EmailVerification: "email_verification",
  EmailChangeConfirmation: "email_change_confirmation",
  AccountSecurityAlert: "account_security_alert",
  AgentActionApprovalRequest: "agent_action_approval_request",
  LifecycleWorkflowNotification: "lifecycle_workflow_notification",
}


@doc("One notification template override by a tenant, unique on `(tenant_id, template_key, locale)`. A combination with no row uses the built-in default (BuiltinNotificationTemplate). Only the subject, the plain text body, the HTML body, and the sender display name can be overridden; replacing the whole HTML document or the CSS is not accepted. The substitution variables a body may use are confined to the permitted set of its template_key.")
model NotificationTemplate {
  @doc("The tenant it belongs to.")
  tenant_id: string;
  template_key: NotificationTemplateKey;
  @doc("The locale this override applies to. Only a locale the catalogue ships translations for is accepted.")
  @minLength(1)
  @pattern("^[a-z]{2}$")
  locale: string;
  @doc("The mail subject. It is treated as a single line and may hold no newline.")
  @minLength(1)
  @maxLength(200)
  subject: string;
  @doc("The plain text body. It is never interpreted as HTML.")
  @minLength(1)
  @maxLength(8000)
  body_text: string;
  @doc("The HTML body. It is a fragment that goes inside `<body>`; the doctype, the `<head>`, the character encoding, and the styling of the body container all come from the system-supplied outer frame. Substituted values are always escaped before expansion. It is always saved together with the text body, and an override of only one of the two cannot be made.")
  @minLength(1)
  @maxLength(20000)
  body_html: string;
  @doc("The sender display name. An empty string means unset, and the system default sender display is used. The email address itself is a server setting and cannot be overridden.")
  @maxLength(80)
  from_display_name?: string;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("One row of the template list. For each key and locale pair it returns whether it has been overridden and the subject currently in force.")
model NotificationTemplateSummary {
  template_key: NotificationTemplateKey;
  locale: string;
  @doc("True when a tenant override exists; false when the built-in default is in force.")
  customized: boolean;
  @doc("The subject currently in force: the override if there is one, the built-in default otherwise.")
  subject: string;
  @doc("When the override was last updated, if there is one. Left out when only the built-in default is in force.")
  updated_at?: utcDateTime;
}


@doc("The template list. It returns every key of the catalogue against every supported locale, overridden or not.")
model NotificationTemplateListResponse {
  templates: NotificationTemplateSummary[];
  supported_locales: string[];
}


@doc("How one template is represented for editing. It returns the wording currently in force, the built-in default wording, and the permitted set of substitution variables together, so the administration UI can show the difference from the default and offer \"restore the default\" without another request.")
model NotificationTemplateDetail {
  template_key: NotificationTemplateKey;
  locale: string;
  customized: boolean;
  subject: string;
  body_text: string;
  body_html: string;
  from_display_name?: string;
  @doc("The built-in default subject.")
  default_subject: string;
  default_body_text: string;
  default_body_html: string;
  @doc("The permitted set of substitution variable names for this template_key.")
  placeholders: string[];
  updated_at?: utcDateTime;
}


@doc("The request saving a template override. The four fields are replaced as a set -- this is not a partial update -- and the subject, the text body, and the HTML body are all required. A substitution variable outside the permitted set means nothing is saved and the request is refused.")
model NotificationTemplateUpdateRequest {
  @minLength(1)
  @maxLength(200)
  subject: string;
  @minLength(1)
  @maxLength(8000)
  body_text: string;
  @minLength(1)
  @maxLength(20000)
  body_html: string;
  @maxLength(80)
  from_display_name?: string;
}


@doc("The request rendering unsaved wording against sample values. A field left out uses the wording currently in force.")
model NotificationTemplatePreviewRequest {
  subject?: string;
  body_text?: string;
  body_html?: string;
  from_display_name?: string;
}


@doc("The rendering with the sample values expanded. No mail is actually sent.")
model NotificationTemplatePreviewResponse {
  subject: string;
  body_text: string;
  body_html: string;
  from_display_name?: string;
}


@doc("The result of a test send. The recipient is fixed to the acting administrator's own verified email address and cannot be named in the request.")
model NotificationTemplateTestSendResponse {
  delivered: boolean;
  @doc("The address the message actually went to, the caller's own. It is returned so the UI can state plainly where it was sent.")
  to: string;
}


@doc("A tenant administrator saved a notification template override. The wording itself is not recorded; only the key and locale it applies to.")
model NotificationTemplateUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  templateKey: string;
  locale: string;
}


@doc("A tenant administrator deleted a notification template override and returned to the built-in default.")
model NotificationTemplateReset {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  templateKey: string;
  locale: string;
}


@doc("The JSON request in which an administrator registers a new tenant. `realm` is a URL-safe slug, and the server mints the UUID key (`id`).")
model TenantCreateRequest {
  @minLength(1)
  @maxLength(63)
  @pattern("^[a-z0-9][a-z0-9-]{0,62}$")
  realm: string;
  @minLength(1)
  @maxLength(200)
  display_name: string;
}


@doc("The JSON request in which an administrator updates a tenant's display name or password policy override. `id` and `status` are left alone; a status change goes through the dedicated disable and enable endpoints, and `default_locale` through the tenant's own settings endpoint, which is where the server reads it.")
model TenantUpdateRequest {
  @minLength(1)
  @maxLength(200)
  display_name?: string;
  @doc("Left out, the existing value is kept. Pass null explicitly to clear it.")
  password_policy_override?: PasswordPolicyOverride;
  @doc("Maximum delegation depth. Omission preserves the current value, and zero restores the system default. Values above the system default are rejected.")
  @minValue(0)
  max_delegation_depth?: int64;
  @doc("How many seconds a trusted device stays valid. Left out, the existing value is kept; 0 turns the feature off again.\nA value above the system ceiling of 7,776,000 seconds (90 days) is refused.")
  @minValue(0)
  trusted_device_max_age_seconds?: int64;
}


@doc("The JSON request in which a tenant administrator updates their own tenant's settings. It is TenantUpdateRequest plus `default_locale`: the locale is the tenant's own choice, so the endpoint fixed to `context.tenant_id` takes it while the cross-tenant administration endpoint does not.")
model AdminSettingsUpdateRequest {
  @minLength(1)
  @maxLength(200)
  display_name?: string;
  @doc("Left out, the existing value is kept. Pass null explicitly to clear it.")
  password_policy_override?: PasswordPolicyOverride;
  @doc("Maximum delegation depth. Omission preserves the current value, and zero restores the system default. Values above the system default are rejected.")
  @minValue(0)
  max_delegation_depth?: int64;
  @doc("How many seconds a trusted device stays valid. Left out, the existing value is kept; 0 turns the feature off again.\nA value above the system ceiling of 7,776,000 seconds (90 days) is refused.")
  @minValue(0)
  trusted_device_max_age_seconds?: int64;
  @doc("The tenant's default locale. Left out, the existing value is kept; an empty string returns it to the system default. An unsupported locale is refused.")
  default_locale?: string;
}


@doc("How a tenant is represented. The list and the detail views of the administration UI return the same representation.")
model TenantSummaryResponse {
  @doc("The immutable UUID key of the tenant.")
  id: string;
  @doc("The slug identifier that appears in the URL `/realms/{realm}/`.")
  realm: string;
  display_name: string;
  status: TenantStatus;
  password_policy_override?: PasswordPolicyOverride;
  @minValue(1)
  max_delegation_depth?: int64;
  @doc("How many seconds a trusted device stays valid. Zero, or leaving it out, disables the feature.")
  @minValue(0)
  trusted_device_max_age_seconds?: int64;
  default_locale?: string;
  quota: TenantQuota;
  usage: TenantUsage;
  created_at: utcDateTime;
  updated_at?: utcDateTime;
  disabled_at?: utcDateTime;
}


@doc("The tenant list of the administration API.")
model TenantListResponse {
  tenants: TenantSummaryResponse[];
}


@doc("A system_admin created a new tenant.")
model TenantCreated {
  occurredAt: utcDateTime;
  actorUserId: string;
  tenantId: string;
}


@doc("A system_admin updated a tenant's display name.")
model TenantUpdated {
  occurredAt: utcDateTime;
  actorUserId: string;
  tenantId: string;
  changedFields: string[];
}


@doc("A system_admin overrode a tenant's resource quota.")
model TenantQuotaUpdated {
  occurredAt: utcDateTime;
  actorUserId: string;
  tenantId: string;
}


@doc("A tenant administrator updated the custom attribute definitions (TenantUserAttributeSchema). No value is kept; only the list of custom attribute keys after the change is recorded.")
model TenantUserAttributeSchemaUpdated {
  occurredAt: utcDateTime;
  actorUserId: string;
  tenantId: string;
  attributeKeys: string[];
}


@doc("A tenant administrator updated the Group custom attribute definitions (TenantGroupAttributeSchema). No value is kept; only the list of custom attribute keys after the change is recorded.")
model TenantGroupAttributeSchemaUpdated {
  occurredAt: utcDateTime;
  actorUserId: string;
  tenantId: string;
  attributeKeys: string[];
}


@doc("A system_admin disabled a tenant. Every protocol route of that tenant is refused with invalid_request.")
model TenantDisabled {
  occurredAt: utcDateTime;
  actorUserId: string;
  tenantId: string;
}


@doc("A system_admin re-enabled a disabled tenant.")
model TenantEnabled {
  occurredAt: utcDateTime;
  actorUserId: string;
  tenantId: string;
}


@doc("The tenant's `password_policy_override` is weaker than the global floor policy, or its `max_age_days` is outside the system ceiling. RFC 9457 Problem Details with `type` = `urn:idmagic:error:policy_override_weaker`.")
model PolicyOverrideWeakerError is ProblemDetails;


@doc("The `user_attribute_schema` definition is invalid: a bad type, a duplicate key, a collision with a built-in attribute, and so on. RFC 9457 Problem Details with `type` = `urn:idmagic:error:invalid_attribute_schema`, the same code the group schema rejection uses; the endpoint tells the two apart.")
model InvalidUserAttributeSchemaError is ProblemDetails;


@doc("The `group_attribute_schema` definition is invalid: a bad type, a duplicate key, and so on. RFC 9457 Problem Details with `type` = `urn:idmagic:error:invalid_attribute_schema`, the same code the user schema rejection uses; the endpoint tells the two apart.")
model InvalidGroupAttributeSchemaError is ProblemDetails;

}
