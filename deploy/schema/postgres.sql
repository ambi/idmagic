-- User identity policy (ADR-082):
-- - The User's canonical identifier is the domain column users.id (global,
--   unique across tenants). The OIDC `sub` claim, SAML NameID, WS-Fed subject,
--   and SCIM resource references are protocol-facing projections of users.id;
--   the protocol vocabulary `sub` is not used as a storage identity.
-- - A User's own identifier is `id`; a reference to a User from another table is
--   `user_id` (an owner reference is `owner_user_id`).

-- tenant_id key policy (ADR-082, simplified by ADR-083): users.id and
-- clients.client_id are system-generated globally unique identifiers, so child
-- rows reference them by their global key and tenant-scoped composite foreign keys
-- are no longer used. Keep tenant_id on a table only when it serves search, a
-- constraint/uniqueness, retention, or audit; do not add it just because tenant is
-- reachable through a globally unique parent. Cases:
-- - tenant-owned aggregate / tenant-scoped config: carry tenant_id, often as part
--   of the PK or a unique key (users, groups, clients, applications, agents,
--   signing_keys, application_categories, saml_service_providers,
--   wsfed_relying_parties, *_sign_in_policies).
-- - external tenant-scoped natural key: tenant_id is part of the PK because the
--   external id is only unique within a tenant (scim_user_refs, scim_group_refs on
--   (tenant_id, scim_id)).
-- - child of a globally unique parent: rely on the global key (user_id / client_id)
--   and omit tenant_id unless per-tenant search/retention needs it. Omitted:
--   consents, application_orderings, mfa_factors, password_history,
--   password_reset_tokens, email_change_tokens, group_members. Kept for per-tenant
--   search: refresh_tokens (tenant token indexes).
-- - append-only / audit / outbox / throttling: decide by emit-time tenant, query
--   boundary, and retention (audit_events, authentication_event_buckets, outbox).

-- Timestamp policy:
-- - Every table has created_at.
-- - Tables whose rows can be updated after creation have updated_at.
-- - Insert-only/delete-only rows do not have updated_at.
-- - Domain timestamps such as issued_at, granted_at, occurred_at, expires_at,
--   revoked_at, first_seen, and last_seen keep their domain meaning;
--   they do not replace created_at.

-- Column type policy (ADR-084):
-- - Strings: never use unconstrained varchar. Unbounded values are TEXT; values
--   with a spec/UI/ops limit get TEXT + CHECK(char_length) or varchar(N)
--   (the limits themselves are decided in wi-128).
-- - JSONB is for external-spec-derived metadata, claim/policy config, and
--   append-only audit/outbox payloads. Values needing join/filter/FK/
--   uniqueness or a lifecycle state machine are not kept inside JSONB
--   (users.lifecycle is the flagged normalization candidate).
-- - TIMESTAMPTZ stores microsecond precision as the source of truth; do not round
--   in schema. Second-precision rounding happens only at external protocol
--   boundaries (SCIM/SAML/WS-Fed formatting).
-- - Ids that idmagic generates internally use UUID: users.id, clients.client_id,
--   groups.id, agents.id, audit_events.id, scim_tokens.id, and the already-UUID
--   refresh_tokens/applications/application_categories keys, plus every FK column
--   that references them (user_id, owner_user_id, group_id, agent_id, client_id,
--   subject_id). Go keeps these as string; base.go registers a text codec for the
--   uuid OID so UUID columns read/write as string. Ids whose value is decided
--   externally stay TEXT: entity_id, wtrealm, scim_id, kid. tenants.id is a UUID
--   surrogate key; the mutable URL slug lives in tenants.realm (ADR-085). Non-FK
--   tenant_id columns (audit_events, authentication_event_buckets)
--   stay TEXT and hold the UUID as string (audit_events also carries a ''
--   tenantless sentinel).
-- - Finite value sets default to TEXT + CHECK; PostgreSQL enums are avoided due to
--   migration friction. CHECK-less finite columns are constraint-addition
--   candidates, added per-column with matching Go validation, not in bulk.

-- tenants (ADR-085): id is an immutable UUID surrogate key referenced by every
-- tenant_id FK; realm is the mutable URL slug shown in /realms/{realm}/ and the
-- OIDC issuer, unique and renameable.
CREATE TABLE tenants (
    id UUID PRIMARY KEY,
    realm TEXT NOT NULL,
    display_name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    disabled_at TIMESTAMPTZ,
    CONSTRAINT tenants_realm_unique UNIQUE (realm),
    CONSTRAINT tenants_realm_format CHECK (
        realm <> 'admin' AND realm ~ '^[a-z0-9][a-z0-9-]{0,62}$'
    )
);

-- tenant_brandings (wi-89, ADR-096): 1:1 hosted UI branding config per tenant, kept
-- in its own table rather than columns on tenants so per-feature config growth does
-- not bloat the core tenant row (same reasoning as tenant_user_attribute_schemas).
-- Absence of a row, or all-NULL columns, means branding is unset and callers fall
-- back to system defaults.
CREATE TABLE tenant_brandings (
    tenant_id UUID PRIMARY KEY,
    product_name TEXT,
    logo_object_key TEXT,
    logo_url TEXT,
    favicon_object_key TEXT,
    favicon_url TEXT,
    primary_color TEXT,
    accent_color TEXT,
    footer_link_1_label TEXT,
    footer_link_1_url TEXT,
    footer_link_2_label TEXT,
    footer_link_2_url TEXT,
    footer_text TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT tenant_brandings_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT tenant_brandings_primary_color_format CHECK (primary_color IS NULL OR primary_color ~ '^#[0-9a-fA-F]{6}$'),
    CONSTRAINT tenant_brandings_accent_color_format CHECK (accent_color IS NULL OR accent_color ~ '^#[0-9a-fA-F]{6}$'),
    CONSTRAINT tenant_brandings_footer_link_1_complete CHECK ((footer_link_1_label IS NULL) = (footer_link_1_url IS NULL)),
    CONSTRAINT tenant_brandings_footer_link_2_complete CHECK ((footer_link_2_label IS NULL) = (footer_link_2_url IS NULL)),
    CONSTRAINT tenant_brandings_footer_link_1_label_length CHECK (footer_link_1_label IS NULL OR char_length(footer_link_1_label) <= 80),
    CONSTRAINT tenant_brandings_footer_link_2_label_length CHECK (footer_link_2_label IS NULL OR char_length(footer_link_2_label) <= 80),
    CONSTRAINT tenant_brandings_footer_link_1_url_format CHECK (footer_link_1_url IS NULL OR footer_link_1_url ~ '^https://'),
    CONSTRAINT tenant_brandings_footer_link_2_url_format CHECK (footer_link_2_url IS NULL OR footer_link_2_url ~ '^https://')
);

-- tenant_branding_assets (wi-89, ADR-096): validated logo / favicon blobs for
-- tenant branding. Same shape as application_icons (ADR-073) but kept in a
-- separate table / object_key space so branding asset ownership never crosses
-- with Application icon storage. kind distinguishes logo vs favicon within one
-- tenant.
CREATE TABLE tenant_branding_assets (
    tenant_id UUID NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('logo', 'favicon')),
    object_key TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    data BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, kind, object_key),
    CONSTRAINT tenant_branding_assets_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE TABLE clients (
    tenant_id UUID NOT NULL,
    client_id UUID PRIMARY KEY,
    client_secret_hash TEXT,
    client_name TEXT,
    client_type TEXT NOT NULL CHECK (client_type IN ('public', 'confidential')),
    redirect_uris JSONB NOT NULL,
    grant_types JSONB NOT NULL,
    response_types JSONB NOT NULL DEFAULT '[]'::jsonb,
    token_endpoint_auth_method TEXT NOT NULL,
    scope TEXT NOT NULL,
    jwks_uri TEXT,
    jwks JSONB,
    tls_client_auth_subject_dn TEXT,
    id_token_signed_response_alg TEXT NOT NULL DEFAULT 'PS256',
    require_pushed_authorization_requests BOOLEAN NOT NULL DEFAULT FALSE,
    dpop_bound_access_tokens BOOLEAN NOT NULL DEFAULT FALSE,
    fapi_profile TEXT NOT NULL DEFAULT 'none',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    first_party BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT clients_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT
);

CREATE TABLE users (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    preferred_username TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    name TEXT,
    given_name TEXT,
    family_name TEXT,
    email TEXT,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    mfa_enrolled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    roles JSONB NOT NULL DEFAULT '[]'::jsonb,
    lifecycle JSONB NOT NULL DEFAULT jsonb_build_object('status', 'active'),
    attributes JSONB NOT NULL DEFAULT '{}'::jsonb,
    CONSTRAINT users_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT,
    CONSTRAINT users_tenant_id_unique UNIQUE (tenant_id, id)
);

CREATE UNIQUE INDEX users_preferred_username_active_idx
    ON users (tenant_id, preferred_username)
    WHERE lifecycle->>'status' <> 'deleted';

CREATE TABLE mfa_factors (
    user_id UUID NOT NULL,
    type TEXT NOT NULL,
    secret TEXT,
    label TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ,
    PRIMARY KEY (user_id, type),
    CONSTRAINT mfa_factors_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- WebAuthn / Passkey credential (wi-26 / ADR-087)。1 ユーザーが複数持てるため credential_id を
-- 主キーとし、mfa_factors とは別テーブルとする。public_key は COSE 公開鍵 (base64url)。
CREATE TABLE webauthn_credentials (
    credential_id TEXT PRIMARY KEY,
    user_id UUID NOT NULL,
    public_key TEXT NOT NULL,
    sign_count BIGINT NOT NULL DEFAULT 0,
    transports TEXT[] NOT NULL DEFAULT '{}',
    aaguid TEXT,
    label TEXT,
    backup_eligible BOOLEAN NOT NULL DEFAULT false,
    backup_state BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ,
    CONSTRAINT webauthn_credentials_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX webauthn_credentials_user_id_idx ON webauthn_credentials (user_id);

-- backup recovery code (wi-26 / ADR-087)。平文は保存せず code_hash (SHA-256 hex) のみ。
-- consumed_at が非 NULL なら使用済みで再利用不可。再生成は user 単位で全置換する。
CREATE TABLE recovery_codes (
    user_id UUID NOT NULL,
    code_hash TEXT NOT NULL,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    consumed_at TIMESTAMPTZ,
    PRIMARY KEY (user_id, code_hash),
    CONSTRAINT recovery_codes_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE consents (
    user_id UUID NOT NULL,
    client_id UUID NOT NULL,
    scopes JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    granted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    PRIMARY KEY (user_id, client_id),
    CONSTRAINT consents_client_fkey
        FOREIGN KEY (client_id)
        REFERENCES clients(client_id) ON DELETE RESTRICT,
    CONSTRAINT consents_user_fkey
        FOREIGN KEY (user_id)
        REFERENCES users(id) ON DELETE RESTRICT
);

CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY,
    hash TEXT NOT NULL,
    family_id UUID NOT NULL,
    parent_id UUID,
    client_id UUID NOT NULL,
    user_id UUID NOT NULL,
    scopes JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    issued_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    absolute_expires_at TIMESTAMPTZ NOT NULL,
    revoked BOOLEAN NOT NULL DEFAULT FALSE,
    rotated BOOLEAN NOT NULL DEFAULT FALSE,
    sender_constraint JSONB,
    CONSTRAINT refresh_tokens_hash_key UNIQUE (hash),
    CONSTRAINT refresh_tokens_parent_id_fkey
        FOREIGN KEY (parent_id) REFERENCES refresh_tokens(id) ON DELETE NO ACTION,
    CONSTRAINT refresh_tokens_client_fkey
        FOREIGN KEY (client_id)
        REFERENCES clients(client_id) ON DELETE RESTRICT,
    CONSTRAINT refresh_tokens_user_fkey
        FOREIGN KEY (user_id)
        REFERENCES users(id) ON DELETE RESTRICT
);

CREATE INDEX refresh_tokens_family_id_idx ON refresh_tokens (family_id);
CREATE INDEX refresh_tokens_user_id_idx ON refresh_tokens (user_id);
CREATE INDEX refresh_tokens_client_id_idx ON refresh_tokens (client_id);

CREATE TABLE signing_keys (
    kid TEXT PRIMARY KEY,
    tenant_id UUID NOT NULL,
    alg TEXT NOT NULL,
    provider TEXT NOT NULL DEFAULT 'Postgres',
    key_usage TEXT NOT NULL DEFAULT 'Signing',
    public_jwk JSONB NOT NULL,
    private_jwk JSONB NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    rotated_at TIMESTAMPTZ,
    archived_at TIMESTAMPTZ,
    CONSTRAINT signing_keys_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX signing_keys_single_active_idx
    ON signing_keys (tenant_id, active)
    WHERE active;

CREATE TABLE outbox (
    id BIGSERIAL PRIMARY KEY,
    event_type TEXT NOT NULL,
    topic TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ,
    published_to TEXT,
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT
);

CREATE INDEX outbox_unpublished_idx ON outbox (id) WHERE published_at IS NULL;

CREATE TABLE password_history (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    encoded TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT password_history_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX password_history_user_id_created_at_idx
    ON password_history (user_id, created_at DESC, id DESC);

CREATE TABLE password_reset_tokens (
    token_hash TEXT PRIMARY KEY,
    user_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT password_reset_tokens_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX password_reset_tokens_user_id_idx ON password_reset_tokens (user_id);
CREATE INDEX password_reset_tokens_expires_at_idx ON password_reset_tokens (expires_at);

CREATE TABLE groups (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    roles JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT groups_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT,
    CONSTRAINT groups_tenant_id_id_unique UNIQUE (tenant_id, id),
    CONSTRAINT groups_tenant_name_key UNIQUE (tenant_id, name)
);

CREATE TABLE group_members (
    group_id UUID NOT NULL,
    user_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (group_id, user_id),
    CONSTRAINT group_members_group_id_fkey
        FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    CONSTRAINT group_members_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX group_members_user_id_idx ON group_members (user_id);

CREATE TABLE tenant_user_attribute_schemas (
    tenant_id UUID PRIMARY KEY,
    attributes JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT tenant_user_attribute_schemas_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE TABLE email_change_tokens (
    token_hash TEXT PRIMARY KEY,
    user_id UUID NOT NULL,
    new_email TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT email_change_tokens_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX email_change_tokens_user_id_idx ON email_change_tokens (user_id);
CREATE INDEX email_change_tokens_expires_at_idx ON email_change_tokens (expires_at);

CREATE TABLE audit_events (
    id UUID PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    type TEXT NOT NULL,
    user_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    occurred_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX audit_events_tenant_occurred_idx
    ON audit_events (tenant_id, occurred_at DESC);
CREATE INDEX audit_events_type_idx ON audit_events (type);
CREATE INDEX audit_events_user_id_idx ON audit_events (user_id) WHERE user_id IS NOT NULL;

CREATE TABLE authentication_event_buckets (
    tenant_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    key_hash TEXT NOT NULL,
    window_start TIMESTAMPTZ NOT NULL,
    count BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    first_seen TIMESTAMPTZ NOT NULL,
    last_seen TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (tenant_id, kind, key_hash, window_start)
);

CREATE INDEX authentication_event_buckets_window_idx
    ON authentication_event_buckets (tenant_id, window_start DESC);

-- 相関 salt (wi-145 / ADR-046)。username / IP の相関ハッシュ (SaltedHash) と
-- throttle / bucket の keyHash に使う per-tenant secret。tenant salt により
-- cross-tenant で相関を集約しない。初回取得時に generate-on-first-use する。
CREATE TABLE tenant_correlation_salts (
    tenant_id TEXT PRIMARY KEY,
    salt BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 監査イベントの sidecar 検索属性 (wi-145)。1 行 = (event, attr_name, transform 済み値)。
-- attr_name は AuditSearchRegistry の Field。PII 属性は hash / 丸め済みで平文は入らない
-- (平文は audit_events.payload 側にのみ、失敗イベント限定で短期保持される)。
-- audit_events の削除に追随するよう ON DELETE CASCADE。
CREATE TABLE audit_event_search_attributes (
    event_id UUID NOT NULL REFERENCES audit_events(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL,
    attr_name TEXT NOT NULL,
    attr_value TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (event_id, attr_name)
);

-- eq / in の等値照合を (tenant, attr_name, attr_value) で index し、occurred_at で降順走査する。
CREATE INDEX audit_event_search_attributes_lookup_idx
    ON audit_event_search_attributes (tenant_id, attr_name, attr_value, occurred_at DESC);

CREATE TABLE agents (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    kind TEXT NOT NULL DEFAULT 'supervised',
    owner_user_id UUID NOT NULL,
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'disabled', 'killed')),
    roles JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    disabled_at TIMESTAMPTZ,
    killed_at TIMESTAMPTZ,
    CONSTRAINT agents_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT,
    CONSTRAINT agents_owner_fkey
        FOREIGN KEY (owner_user_id)
        REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT agents_tenant_id_id_unique UNIQUE (tenant_id, id),
    CONSTRAINT agents_tenant_name_key UNIQUE (tenant_id, name)
);

CREATE TABLE agent_credential_bindings (
    agent_id UUID NOT NULL,
    client_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (agent_id, client_id),
    CONSTRAINT agent_credential_bindings_client_id_key UNIQUE (client_id),
    CONSTRAINT agent_credential_bindings_agent_id_fkey
        FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE,
    CONSTRAINT agent_credential_bindings_client_fkey
        FOREIGN KEY (client_id)
        REFERENCES clients(client_id) ON DELETE RESTRICT
);

CREATE INDEX agent_credential_bindings_client_idx
    ON agent_credential_bindings (client_id);

CREATE TABLE authorization_detail_types (
    tenant_id UUID NOT NULL,
    type TEXT NOT NULL,
    description TEXT,
    schema JSONB NOT NULL DEFAULT jsonb_build_object('rules', jsonb_build_array()),
    display_template TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'Enabled'
        CHECK (state IN ('Enabled', 'Disabled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, type),
    CONSTRAINT authorization_detail_types_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT
);

CREATE TABLE applications (
    tenant_id UUID NOT NULL,
    application_id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    status TEXT NOT NULL,
    icon_url TEXT NOT NULL DEFAULT '',
    icon_object_key TEXT NOT NULL DEFAULT '',
    launch_url TEXT NOT NULL DEFAULT '',
    bindings JSONB NOT NULL DEFAULT '[]'::jsonb,
    category_ids TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT applications_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT
);

CREATE TABLE application_icons (
    application_id UUID NOT NULL,
    object_key TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    data BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (application_id, object_key),
    CONSTRAINT application_icons_application_fkey
        FOREIGN KEY (application_id)
        REFERENCES applications (application_id) ON DELETE CASCADE
);

CREATE TABLE application_sign_in_policies (
    application_id UUID PRIMARY KEY,
    rules JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT application_sign_in_policies_application_fkey
        FOREIGN KEY (application_id)
        REFERENCES applications (application_id) ON DELETE CASCADE
);

CREATE TABLE tenant_default_sign_in_policies (
    tenant_id UUID PRIMARY KEY,
    rules JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT tenant_default_sign_in_policies_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE TABLE application_assignments (
    application_id UUID NOT NULL,
    subject_type TEXT NOT NULL,
    subject_id UUID NOT NULL,
    visibility TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (application_id, subject_type, subject_id),
    CHECK (subject_type IN ('user', 'group')),
    CHECK (visibility IN ('visible', 'hidden')),
    CONSTRAINT application_assignments_application_fkey
        FOREIGN KEY (application_id)
        REFERENCES applications (application_id) ON DELETE CASCADE
);

CREATE INDEX application_assignments_subject_idx
    ON application_assignments (subject_type, subject_id);

CREATE TABLE saml_service_providers (
    tenant_id UUID NOT NULL,
    entity_id TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    acs_urls JSONB NOT NULL DEFAULT '[]'::jsonb,
    slo_url TEXT NOT NULL DEFAULT '',
    audience TEXT NOT NULL DEFAULT '',
    claim_policy JSONB NOT NULL,
    sign_assertion BOOLEAN NOT NULL DEFAULT TRUE,
    sign_response BOOLEAN NOT NULL DEFAULT FALSE,
    want_authn_requests_signed BOOLEAN NOT NULL DEFAULT FALSE,
    authn_request_signing_certificate_pem TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, entity_id),
    CONSTRAINT saml_service_providers_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT
);

CREATE TABLE wsfed_relying_parties (
    tenant_id UUID NOT NULL,
    wtrealm TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    reply_urls JSONB NOT NULL DEFAULT '[]'::jsonb,
    audience TEXT NOT NULL DEFAULT '',
    token_type TEXT NOT NULL DEFAULT '',
    claim_policy JSONB NOT NULL,
    entra_profile JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, wtrealm),
    CONSTRAINT wsfed_relying_parties_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT
);

CREATE TABLE application_orderings (
    user_id UUID PRIMARY KEY,
    application_ids TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT application_orderings_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE application_categories (
    tenant_id UUID NOT NULL,
    category_id UUID NOT NULL,
    name TEXT NOT NULL,
    position INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, category_id),
    CONSTRAINT application_categories_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT
);

CREATE TABLE scim_tokens (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    token_hash TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ,
    CONSTRAINT scim_tokens_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT
);

CREATE TABLE scim_user_refs (
    tenant_id UUID NOT NULL,
    scim_id TEXT NOT NULL,
    user_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, scim_id),
    CONSTRAINT scim_user_refs_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT,
    CONSTRAINT scim_user_refs_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE scim_group_refs (
    tenant_id UUID NOT NULL,
    scim_id TEXT NOT NULL,
    group_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, scim_id),
    CONSTRAINT scim_group_refs_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT,
    CONSTRAINT scim_group_refs_group_id_fkey
        FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
);

-- jobs (wi-126, ADR-098/099/100): durable async job queue. tenant-owned aggregate
-- (ARCHITECTURE.md tenant_id 4-category rule), so tenant_id is required even for
-- an aggregate with no natural-key parent. status/kind are closed vocabularies
-- normative in spec/contexts/jobs.yaml, enforced here via CHECK. params/result are
-- opaque per-JobKind payloads (ADR-100: plain JSONB, no at-rest encryption in this
-- WI; terminal rows are purged after a TTL by the worker's relocated retention
-- sweep, not by a dedicated Job).
CREATE TABLE jobs (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    kind TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'canceled')),
    params JSONB NOT NULL,
    result JSONB,
    error TEXT,
    attempts INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL,
    dedup_key TEXT,
    lease_owner TEXT,
    lease_expires_at TIMESTAMPTZ,
    run_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT jobs_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT
);

-- Claim scan (ADR-098 SKIP LOCKED): due StatusQueued jobs ordered by run_at, plus
-- StatusRunning jobs whose lease expired (JobLeaseExclusivity reclaim).
CREATE INDEX jobs_claimable_idx ON jobs (run_at) WHERE status = 'queued';
CREATE INDEX jobs_lease_expiry_idx ON jobs (lease_expires_at) WHERE status = 'running';

-- JobHandlerIdempotency: at most one non-terminal Job per (tenant_id, dedup_key).
CREATE UNIQUE INDEX jobs_tenant_dedup_key_active_idx
    ON jobs (tenant_id, dedup_key)
    WHERE dedup_key IS NOT NULL AND status IN ('queued', 'running');
