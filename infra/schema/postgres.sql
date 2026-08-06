CREATE TABLE tenants (
    id UUID PRIMARY KEY,
    realm TEXT NOT NULL,
    display_name TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled')),
    default_locale TEXT,
    endpoint_style TEXT NOT NULL DEFAULT 'path'
        CHECK (endpoint_style IN ('path', 'subdomain')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    disabled_at TIMESTAMPTZ,
    CONSTRAINT tenants_realm_unique UNIQUE (realm),
    CONSTRAINT tenants_default_locale_format CHECK (default_locale IS NULL OR default_locale ~ '^[a-z]{2}$'),
    CONSTRAINT tenants_realm_format CHECK (
        realm <> 'admin' AND realm ~ '^[a-z0-9][a-z0-9-]{0,62}$'
    )
);

CREATE TABLE tenant_quotas (
    tenant_id UUID PRIMARY KEY,
    users INT,
    groups INT,
    agents INT,
    applications INT,
    oauth2_clients INT,
    active_sessions INT,
    consents INT,
    active_jobs INT,
    audit_events_retained INT,
    export_artifacts_bytes INT,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE TABLE tenant_usages (
    tenant_id UUID PRIMARY KEY,
    users INT NOT NULL DEFAULT 0 CHECK (users >= 0),
    groups INT NOT NULL DEFAULT 0 CHECK (groups >= 0),
    agents INT NOT NULL DEFAULT 0 CHECK (agents >= 0),
    applications INT NOT NULL DEFAULT 0 CHECK (applications >= 0),
    oauth2_clients INT NOT NULL DEFAULT 0 CHECK (oauth2_clients >= 0),
    active_sessions INT NOT NULL DEFAULT 0 CHECK (active_sessions >= 0),
    consents INT NOT NULL DEFAULT 0 CHECK (consents >= 0),
    active_jobs INT NOT NULL DEFAULT 0 CHECK (active_jobs >= 0),
    audit_events_retained INT NOT NULL DEFAULT 0 CHECK (audit_events_retained >= 0),
    export_artifacts_bytes INT NOT NULL DEFAULT 0 CHECK (export_artifacts_bytes >= 0),
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

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

CREATE TABLE notification_templates (
    tenant_id UUID NOT NULL,
    template_key TEXT NOT NULL CHECK (template_key IN (
        'account_security_alert', 'email_change_confirmation', 'email_verification',
        'lifecycle_workflow_notification', 'password_reset'
    )),
    locale TEXT NOT NULL,
    subject TEXT NOT NULL,
    body_text TEXT NOT NULL,
    body_html TEXT NOT NULL,
    from_display_name TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, template_key, locale),
    CONSTRAINT notification_templates_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT notification_templates_locale_format CHECK (locale ~ '^[a-z]{2}$'),
    CONSTRAINT notification_templates_subject_length CHECK (char_length(subject) BETWEEN 1 AND 200),
    CONSTRAINT notification_templates_body_text_length CHECK (char_length(body_text) BETWEEN 1 AND 8000),
    CONSTRAINT notification_templates_body_html_length CHECK (char_length(body_html) BETWEEN 1 AND 20000),
    CONSTRAINT notification_templates_from_display_name_length
        CHECK (from_display_name IS NULL OR char_length(from_display_name) <= 80)
);

CREATE TABLE tenant_branding_assets (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('logo', 'favicon')),
    content_type TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    data BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT tenant_branding_assets_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX tenant_branding_assets_tenant_kind_idx
    ON tenant_branding_assets (tenant_id, kind);

CREATE TABLE oauth2_clients (
    tenant_id UUID NOT NULL,
    client_id UUID PRIMARY KEY,
    application_id UUID UNIQUE,
    application_protocol_type TEXT NOT NULL DEFAULT 'oidc'
        CHECK (application_protocol_type = 'oidc'),
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
    claim_policy JSONB,
    CONSTRAINT oauth2_clients_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT
);

CREATE TABLE oauth2_client_secrets (
    id UUID PRIMARY KEY,
    client_id UUID NOT NULL,
    secret_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    CONSTRAINT oauth2_client_secrets_client_id_fkey
        FOREIGN KEY (client_id) REFERENCES oauth2_clients(client_id) ON DELETE CASCADE,
    CONSTRAINT oauth2_client_secrets_expiry_after_creation
        CHECK (expires_at IS NULL OR expires_at > created_at)
);

CREATE INDEX oauth2_client_secrets_client_id_idx ON oauth2_client_secrets (client_id);

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
    secret_key_version INTEGER,
    secret_ciphertext BYTEA,
    label TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ,
    PRIMARY KEY (user_id, type),
    CONSTRAINT mfa_factors_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE mfa_enrollment_bypasses (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    issued_by UUID NOT NULL,
    issued_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    expired_at TIMESTAMPTZ,
    CONSTRAINT mfa_enrollment_bypasses_tenant_fkey FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT mfa_enrollment_bypasses_user_fkey FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT mfa_enrollment_bypasses_issuer_fkey FOREIGN KEY (tenant_id, issued_by) REFERENCES users(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT mfa_enrollment_bypasses_expiry CHECK (expires_at > issued_at),
    CONSTRAINT mfa_enrollment_bypasses_terminal CHECK (num_nonnulls(consumed_at, revoked_at, expired_at) <= 1)
);

CREATE UNIQUE INDEX mfa_enrollment_bypasses_active_user_idx
    ON mfa_enrollment_bypasses (tenant_id, user_id)
    WHERE consumed_at IS NULL AND revoked_at IS NULL AND expired_at IS NULL;

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
        REFERENCES oauth2_clients(client_id) ON DELETE RESTRICT,
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
    sid UUID,
    resource TEXT,
    CONSTRAINT refresh_tokens_hash_key UNIQUE (hash),
    CONSTRAINT refresh_tokens_parent_id_fkey
        FOREIGN KEY (parent_id) REFERENCES refresh_tokens(id) ON DELETE NO ACTION,
    CONSTRAINT refresh_tokens_client_fkey
        FOREIGN KEY (client_id)
        REFERENCES oauth2_clients(client_id) ON DELETE RESTRICT,
    CONSTRAINT refresh_tokens_user_fkey
        FOREIGN KEY (user_id)
        REFERENCES users(id) ON DELETE RESTRICT
);

CREATE INDEX refresh_tokens_family_id_idx ON refresh_tokens (family_id);
CREATE INDEX refresh_tokens_user_id_idx ON refresh_tokens (user_id);
CREATE INDEX refresh_tokens_client_id_idx ON refresh_tokens (client_id);
CREATE INDEX refresh_tokens_sid_idx ON refresh_tokens (sid) WHERE sid IS NOT NULL;

CREATE TABLE signing_keys (
    kid TEXT PRIMARY KEY,
    tenant_id UUID NOT NULL,
    alg TEXT NOT NULL,
    provider TEXT NOT NULL DEFAULT 'Postgres',
    key_usage TEXT NOT NULL DEFAULT 'Signing',
    scope_id TEXT NOT NULL DEFAULT 'default',
    public_jwk JSONB NOT NULL,
    private_jwk JSONB NOT NULL,
    certificate_der BYTEA,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    retired_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    archived_at TIMESTAMPTZ,
    CONSTRAINT signing_keys_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT
);

CREATE UNIQUE INDEX signing_keys_single_active_idx
    ON signing_keys (tenant_id, key_usage, scope_id, active)
    WHERE active;

CREATE TABLE tenant_data_encryption_keys (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    version INTEGER NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'retiring', 'disabled', 'destroyed')),
    wrapped_dek BYTEA,
    master_key_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    activated_at TIMESTAMPTZ,
    disabled_at TIMESTAMPTZ,
    destroyed_at TIMESTAMPTZ,
    CONSTRAINT tenant_data_encryption_keys_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT,
    CONSTRAINT tenant_data_encryption_keys_tenant_version_key
        UNIQUE (tenant_id, version)
);

CREATE UNIQUE INDEX tenant_data_encryption_keys_single_active_idx
    ON tenant_data_encryption_keys (tenant_id)
    WHERE status = 'active';

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

CREATE TABLE authentication_sessions (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    auth_time BIGINT NOT NULL,
    amr TEXT[] NOT NULL,
    acr TEXT NOT NULL,
    authentication_pending BOOLEAN NOT NULL DEFAULT FALSE,
    pending_purpose TEXT NOT NULL DEFAULT 'None'
        CHECK (pending_purpose IN ('None', 'Challenge', 'Enrollment')),
    enrollment_deadline TIMESTAMPTZ,
    enrollment_bypass_id UUID,
    step_up_at BIGINT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ,
    revoke_reason TEXT
        CHECK (revoke_reason IS NULL OR revoke_reason IN
            ('logout', 'idle', 'absolute', 'self_revoke', 'admin_revoke',
             'password_change', 'mfa_change', 'other')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT authentication_sessions_revoke_pair
        CHECK ((revoked_at IS NULL) = (revoke_reason IS NULL)),
    CONSTRAINT authentication_sessions_tenant_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT authentication_sessions_user_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX authentication_sessions_active_user_idx
    ON authentication_sessions (tenant_id, user_id, auth_time DESC, id DESC)
    WHERE revoked_at IS NULL;

CREATE INDEX authentication_sessions_expires_at_idx ON authentication_sessions (expires_at);

CREATE TABLE identity_provider_connections (
    id TEXT NOT NULL,
    tenant_id UUID NOT NULL,
    display_name TEXT NOT NULL,
    protocol TEXT NOT NULL CHECK (protocol IN ('oidc', 'saml')),
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled')),
    issuer TEXT NOT NULL,
    client_id TEXT,
    secret_reference TEXT,
    secret_key_version INTEGER,
    secret_ciphertext BYTEA,
    authorization_endpoint TEXT,
    token_endpoint TEXT,
    jwks_uri TEXT,
    saml_sso_url TEXT,
    saml_entity_id TEXT,
    saml_signing_certificates JSONB NOT NULL DEFAULT '[]'::jsonb,
    claim_mapping JSONB NOT NULL,
    linking_policy TEXT NOT NULL CHECK (linking_policy IN ('none', 'verified_email')),
    jit_provisioning BOOLEAN NOT NULL DEFAULT FALSE,
    allowed_email_domains JSONB NOT NULL DEFAULT '[]'::jsonb,
    metadata_refreshed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT identity_provider_connections_tenant_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX identity_provider_connections_tenant_idx
    ON identity_provider_connections (tenant_id);

CREATE TABLE federated_identities (
    tenant_id UUID NOT NULL,
    provider_id TEXT NOT NULL,
    external_subject TEXT NOT NULL,
    local_user_id UUID NOT NULL,
    linked_at TIMESTAMPTZ NOT NULL,
    last_login_at TIMESTAMPTZ,
    PRIMARY KEY (tenant_id, provider_id, external_subject),
    CONSTRAINT federated_identities_tenant_id_provider_id_local_user_id_key
        UNIQUE (tenant_id, provider_id, local_user_id),
    CONSTRAINT federated_identities_provider_fkey
        FOREIGN KEY (provider_id)
        REFERENCES identity_provider_connections(id) ON DELETE RESTRICT,
    CONSTRAINT federated_identities_user_fkey
        FOREIGN KEY (local_user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE federated_login_attempts (
    tenant_id UUID NOT NULL,
    state TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    protocol TEXT NOT NULL CHECK (protocol IN ('oidc', 'saml')),
    nonce TEXT,
    pkce_verifier TEXT,
    request_id TEXT,
    return_to TEXT,
    link_user_id UUID,
    created_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    PRIMARY KEY (tenant_id, state),
    CONSTRAINT federated_login_attempts_provider_fkey
        FOREIGN KEY (provider_id)
        REFERENCES identity_provider_connections(id) ON DELETE CASCADE,
    CONSTRAINT federated_login_attempts_user_fkey
        FOREIGN KEY (link_user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE federated_response_replays (
    tenant_id UUID NOT NULL,
    response_id TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (tenant_id, response_id),
    CONSTRAINT federated_response_replays_tenant_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE TABLE groups (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    roles JSONB NOT NULL DEFAULT '[]'::jsonb,
    membership_type TEXT NOT NULL DEFAULT 'manual' CHECK (membership_type IN ('manual','dynamic')),
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
    source TEXT NOT NULL DEFAULT 'manual' CHECK (source IN ('manual','dynamic_rule')),
    rule_version BIGINT,
    PRIMARY KEY (group_id, user_id),
    CONSTRAINT group_members_group_id_fkey
        FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE,
    CONSTRAINT group_members_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX group_members_user_id_idx ON group_members (user_id);

CREATE TABLE dynamic_group_rules (
    group_id UUID PRIMARY KEY REFERENCES groups(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE RESTRICT,
    expression TEXT NOT NULL CHECK (char_length(expression) BETWEEN 1 AND 4096),
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    version BIGINT NOT NULL CHECK (version > 0),
    referenced_attributes JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT dynamic_group_rules_tenant_id_group_id_key
        UNIQUE (tenant_id, group_id)
);

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

CREATE TABLE tenant_correlation_salts (
    tenant_id TEXT PRIMARY KEY,
    salt BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE audit_event_search_attributes (
    event_id UUID NOT NULL REFERENCES audit_events(id) ON DELETE CASCADE,
    tenant_id TEXT NOT NULL,
    attr_name TEXT NOT NULL,
    attr_value TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (event_id, attr_name)
);

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
        REFERENCES oauth2_clients(client_id) ON DELETE RESTRICT
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

CREATE TABLE mcp_resource_servers (
    tenant_id UUID NOT NULL,
    id UUID PRIMARY KEY,
    resource TEXT NOT NULL,
    name TEXT NOT NULL,
    scopes JSONB NOT NULL,
    state TEXT NOT NULL DEFAULT 'Active'
        CHECK (state IN ('Active', 'Disabled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT mcp_resource_servers_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT,
    CONSTRAINT mcp_resource_servers_tenant_resource_unique
        UNIQUE (tenant_id, resource)
);

CREATE TABLE applications (
    tenant_id UUID NOT NULL,
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('federated', 'weblink', 'service')),
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled')),
    protocol_type TEXT,
    icon_url TEXT NOT NULL DEFAULT '',
    icon_object_key TEXT NOT NULL DEFAULT '',
    launch_url TEXT NOT NULL DEFAULT '',
    category_ids TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT applications_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT,
    CONSTRAINT applications_protocol_kind_check CHECK (
        (kind = 'weblink' AND protocol_type IS NULL)
        OR (kind = 'service' AND protocol_type = 'oidc')
        OR (kind = 'federated' AND protocol_type IN ('oidc', 'saml', 'wsfed'))
    ),
    CONSTRAINT applications_protocol_identity_unique
        UNIQUE (id, tenant_id, protocol_type)
);

CREATE TABLE application_icons (
    id UUID PRIMARY KEY,
    application_id UUID NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    data BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT application_icons_application_fkey
        FOREIGN KEY (application_id)
        REFERENCES applications (id) ON DELETE CASCADE
);

CREATE INDEX application_icons_application_idx
    ON application_icons (application_id);

CREATE TABLE application_sign_in_policies (
    application_id UUID PRIMARY KEY,
    rules JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT application_sign_in_policies_application_fkey
        FOREIGN KEY (application_id)
        REFERENCES applications (id) ON DELETE CASCADE
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
        REFERENCES applications (id) ON DELETE CASCADE
);

CREATE INDEX application_assignments_subject_idx
    ON application_assignments (subject_type, subject_id);

CREATE TABLE saml_identity_provider_profiles (
    tenant_id UUID NOT NULL,
    profile_id TEXT NOT NULL,
    name TEXT NOT NULL,
    mode TEXT NOT NULL CHECK (mode IN ('shared', 'dedicated')),
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, profile_id),
    CHECK ((profile_id = 'default') = is_default),
    CHECK (NOT is_default OR mode = 'shared'),
    CONSTRAINT saml_identity_provider_profiles_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX saml_identity_provider_profiles_single_default_idx
    ON saml_identity_provider_profiles (tenant_id) WHERE is_default;

CREATE FUNCTION create_default_saml_identity_provider_profile()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO saml_identity_provider_profiles
        (tenant_id, profile_id, name, mode, is_default)
    VALUES (NEW.id, 'default', 'Default', 'shared', TRUE);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tenants_create_default_saml_identity_provider_profile
AFTER INSERT ON tenants
FOR EACH ROW EXECUTE FUNCTION create_default_saml_identity_provider_profile();

CREATE TABLE saml_service_providers (
    tenant_id UUID NOT NULL,
    entity_id TEXT NOT NULL,
    idp_profile_id TEXT NOT NULL DEFAULT 'default',
    application_id UUID UNIQUE,
    application_protocol_type TEXT NOT NULL DEFAULT 'saml'
        CHECK (application_protocol_type = 'saml'),
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
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT,
    CONSTRAINT saml_service_providers_idp_profile_fkey
        FOREIGN KEY (tenant_id, idp_profile_id)
        REFERENCES saml_identity_provider_profiles (tenant_id, profile_id) ON DELETE RESTRICT
);

CREATE TABLE wsfed_relying_parties (
    tenant_id UUID NOT NULL,
    wtrealm TEXT NOT NULL,
    application_id UUID UNIQUE,
    application_protocol_type TEXT NOT NULL DEFAULT 'wsfed'
        CHECK (application_protocol_type = 'wsfed'),
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

ALTER TABLE oauth2_clients
ADD CONSTRAINT oauth2_clients_application_fkey
    FOREIGN KEY (application_id, tenant_id, application_protocol_type)
    REFERENCES applications(id, tenant_id, protocol_type) ON DELETE CASCADE;

ALTER TABLE saml_service_providers
ADD CONSTRAINT saml_service_providers_application_fkey
    FOREIGN KEY (application_id, tenant_id, application_protocol_type)
    REFERENCES applications(id, tenant_id, protocol_type) ON DELETE CASCADE;

ALTER TABLE wsfed_relying_parties
ADD CONSTRAINT wsfed_relying_parties_application_fkey
    FOREIGN KEY (application_id, tenant_id, application_protocol_type)
    REFERENCES applications(id, tenant_id, protocol_type) ON DELETE CASCADE;

CREATE INDEX oauth2_clients_application_id_idx
    ON oauth2_clients (application_id) WHERE application_id IS NOT NULL;
CREATE INDEX saml_service_providers_application_id_idx
    ON saml_service_providers (application_id) WHERE application_id IS NOT NULL;
CREATE INDEX wsfed_relying_parties_application_id_idx
    ON wsfed_relying_parties (application_id) WHERE application_id IS NOT NULL;

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
    id UUID NOT NULL,
    name TEXT NOT NULL,
    position INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, id),
    CONSTRAINT application_categories_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT
);

CREATE TABLE api_tokens (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    jti TEXT NOT NULL,
    client_id TEXT NOT NULL,
    scopes TEXT[] NOT NULL,
    audience TEXT NOT NULL,
    dpop_jkt TEXT,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    CONSTRAINT api_tokens_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT,
    CONSTRAINT api_tokens_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT api_tokens_jti_key UNIQUE (jti),
    CONSTRAINT api_tokens_scopes_valid CHECK (
        scopes <@ ARRAY[
            'users:read', 'users:write',
            'groups:read', 'groups:write',
            'agents:read', 'agents:write',
            'sessions:read', 'sessions:write',
            'consents:read', 'consents:write',
            'lifecycle-workflows:read', 'lifecycle-workflows:write',
            'tenants:read', 'tenants:write',
            'settings:read', 'settings:write',
            'signing-keys:read', 'signing-keys:write',
            'audit:read',
            'applications:read', 'applications:write',
            'oauth-clients:read', 'oauth-clients:write',
            'authorization-detail-types:read', 'authorization-detail-types:write',
            'mcp-resource-servers:read', 'mcp-resource-servers:write',
            'saml:read', 'saml:write',
            'wsfed:read', 'wsfed:write',
            'provisioning:read', 'provisioning:write',
            'scim:users:read', 'scim:users:write',
            'scim:groups:read', 'scim:groups:write',
            'account:read', 'account:write',
            'account:mfa:write', 'account:sessions:write',
            'account:consents:write', 'account:password:write'
        ]::TEXT[]
    )
);

CREATE INDEX api_tokens_tenant_id_created_at_idx ON api_tokens (tenant_id, created_at);
CREATE INDEX api_tokens_tenant_id_jti_idx ON api_tokens (tenant_id, jti);

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

CREATE TABLE jobs (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    kind TEXT NOT NULL,
    lane TEXT NOT NULL DEFAULT 'default' CHECK (lane IN ('latency_sensitive', 'default', 'bulk')),
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

CREATE INDEX jobs_claimable_idx ON jobs (lane, run_at) WHERE status = 'queued';
CREATE INDEX jobs_lease_expiry_idx ON jobs (lane, lease_expires_at) WHERE status = 'running';

CREATE UNIQUE INDEX jobs_tenant_dedup_key_active_idx
    ON jobs (tenant_id, dedup_key)
    WHERE dedup_key IS NOT NULL AND status IN ('queued', 'running');

CREATE TABLE lifecycle_workflows (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL CHECK (status IN ('draft', 'enabled', 'disabled', 'archived')),
    current_revision BIGINT NOT NULL CHECK (current_revision >= 1),
    enabled_revision BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT lifecycle_workflows_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT,
    CONSTRAINT lifecycle_workflows_tenant_name_unique UNIQUE (tenant_id, name),
    CONSTRAINT lifecycle_workflows_enabled_revision_consistency CHECK (
        (status = 'enabled' AND enabled_revision IS NOT NULL) OR
        (status <> 'enabled' AND enabled_revision IS NULL)
    )
);

CREATE TABLE lifecycle_workflow_revisions (
    workflow_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    revision BIGINT NOT NULL CHECK (revision >= 1),
    trigger JSONB NOT NULL,
    actions JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (workflow_id, revision),
    CONSTRAINT lifecycle_workflow_revisions_workflow_fkey
        FOREIGN KEY (workflow_id) REFERENCES lifecycle_workflows(id) ON DELETE RESTRICT,
    CONSTRAINT lifecycle_workflow_revisions_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT
);

CREATE INDEX lifecycle_workflows_tenant_status_idx ON lifecycle_workflows (tenant_id, status);

CREATE TABLE lifecycle_workflow_runs (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    workflow_id UUID NOT NULL,
    revision BIGINT NOT NULL,
    source_occurrence_id TEXT NOT NULL,
    target_user_id UUID NOT NULL,
    trigger_kind TEXT NOT NULL,
    changed_fields JSONB NOT NULL DEFAULT '[]'::jsonb,
    actions JSONB NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'succeeded', 'partially_failed', 'failed', 'canceled')),
    job_id UUID,
    triggered_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT lifecycle_workflow_runs_tenant_id_fkey FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT,
    CONSTRAINT lifecycle_workflow_runs_workflow_fkey FOREIGN KEY (workflow_id) REFERENCES lifecycle_workflows(id) ON DELETE RESTRICT,
    CONSTRAINT lifecycle_workflow_runs_target_user_fkey FOREIGN KEY (target_user_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT lifecycle_workflow_runs_occurrence_unique UNIQUE (tenant_id, workflow_id, revision, source_occurrence_id, target_user_id)
);

CREATE TABLE lifecycle_workflow_steps (
    run_id UUID NOT NULL,
    step_index INTEGER NOT NULL CHECK (step_index >= 0),
    action JSONB NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('pending', 'changed', 'no_op', 'failed', 'canceled')),
    error_code TEXT,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (run_id, step_index),
    CONSTRAINT lifecycle_workflow_steps_run_fkey FOREIGN KEY (run_id) REFERENCES lifecycle_workflow_runs(id) ON DELETE CASCADE
);

CREATE INDEX lifecycle_workflow_runs_unenqueued_idx ON lifecycle_workflow_runs (triggered_at) WHERE status = 'queued' AND job_id IS NULL;

CREATE TABLE provisioning_connections (
    application_id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('active', 'disabled')),
    base_url TEXT NOT NULL,
    credential_id UUID NOT NULL,
    auth_method TEXT NOT NULL CHECK (auth_method IN ('bearer_token', 'oauth2_client_credentials')),
    credential_secret TEXT NOT NULL,
    credential_created_at TIMESTAMPTZ NOT NULL,
    credential_rotated_at TIMESTAMPTZ,
    capabilities JSONB,
    feature_flags JSONB NOT NULL,
    scope TEXT NOT NULL CHECK (scope IN ('assigned_only', 'all_users')),
    group_push JSONB,
    attribute_mappings JSONB NOT NULL DEFAULT '[]'::jsonb,
    matching JSONB NOT NULL,
    deprovision_policy JSONB NOT NULL,
    rate_limit_per_minute INT NOT NULL DEFAULT 60,
    max_attempts INT NOT NULL DEFAULT 8,
    notification_email TEXT,
    quarantine_after_consecutive_failures INT NOT NULL DEFAULT 10,
    health TEXT NOT NULL CHECK (health IN ('ok', 'degraded', 'quarantined')),
    consecutive_failure_count INT NOT NULL DEFAULT 0,
    last_full_sync_at TIMESTAMPTZ,
    quarantined_at TIMESTAMPTZ,
    quarantine_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT provisioning_connections_application_fkey
        FOREIGN KEY (application_id) REFERENCES applications(id) ON DELETE CASCADE,
    CONSTRAINT provisioning_connections_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE RESTRICT,
    CONSTRAINT provisioning_connections_quarantine_check
        CHECK ((health = 'quarantined') = (quarantined_at IS NOT NULL))
);

CREATE TABLE provisioning_remote_links (
    connection_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    source_type TEXT NOT NULL CHECK (source_type IN ('user', 'group')),
    source_id UUID NOT NULL,
    remote_id TEXT NOT NULL,
    external_id TEXT NOT NULL,
    etag TEXT,
    last_synced_version BIGINT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (connection_id, source_type, source_id),
    CONSTRAINT provisioning_remote_links_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES provisioning_connections(application_id) ON DELETE CASCADE
);

CREATE TABLE provisioning_deliveries (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    connection_id UUID NOT NULL,
    source_type TEXT NOT NULL CHECK (source_type IN ('user', 'group')),
    source_id UUID NOT NULL,
    source_version BIGINT NOT NULL CHECK (source_version >= 1),
    operation TEXT NOT NULL CHECK (operation IN ('create', 'update', 'deactivate', 'delete', 'membership_add', 'membership_remove')),
    status TEXT NOT NULL CHECK (status IN ('pending', 'in_flight', 'succeeded', 'dead_letter')),
    job_id UUID,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    CONSTRAINT provisioning_deliveries_connection_fkey
        FOREIGN KEY (connection_id) REFERENCES provisioning_connections(application_id) ON DELETE CASCADE,
    CONSTRAINT provisioning_deliveries_idempotency_unique
        UNIQUE (tenant_id, connection_id, source_type, source_id, source_version)
);

CREATE INDEX provisioning_deliveries_unenqueued_idx ON provisioning_deliveries (created_at) WHERE status = 'pending' AND job_id IS NULL;
CREATE INDEX provisioning_deliveries_connection_idx ON provisioning_deliveries (connection_id, created_at DESC);

CREATE UNLOGGED TABLE oauth2_authorization_requests (
    id TEXT PRIMARY KEY,
    tenant_id UUID NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT oauth2_authorization_requests_tenant_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);
CREATE INDEX oauth2_authorization_requests_expires_at_idx
    ON oauth2_authorization_requests (expires_at);

CREATE UNLOGGED TABLE oauth2_authorization_codes (
    code TEXT PRIMARY KEY,
    tenant_id UUID NOT NULL,
    state TEXT NOT NULL,
    redeemed_at TIMESTAMPTZ,
    issued_family_id TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT oauth2_authorization_codes_tenant_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);
CREATE INDEX oauth2_authorization_codes_expires_at_idx
    ON oauth2_authorization_codes (expires_at);

CREATE UNLOGGED TABLE oauth2_par_requests (
    request_uri TEXT PRIMARY KEY,
    tenant_id UUID NOT NULL,
    used BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT oauth2_par_requests_tenant_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);
CREATE INDEX oauth2_par_requests_expires_at_idx
    ON oauth2_par_requests (expires_at);

CREATE UNLOGGED TABLE oauth2_device_codes (
    device_code_hash TEXT PRIMARY KEY,
    tenant_id UUID NOT NULL,
    user_code TEXT NOT NULL,
    user_id UUID,
    state TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT oauth2_device_codes_user_code_unique UNIQUE (tenant_id, user_code),
    CONSTRAINT oauth2_device_codes_tenant_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT oauth2_device_codes_user_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX oauth2_device_codes_expires_at_idx
    ON oauth2_device_codes (expires_at);
CREATE INDEX oauth2_device_codes_user_idx
    ON oauth2_device_codes (tenant_id, user_id) WHERE user_id IS NOT NULL;

CREATE UNLOGGED TABLE oauth2_replay_jtis (
    tenant_id UUID NOT NULL,
    kind TEXT NOT NULL,
    jti TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, kind, jti),
    CONSTRAINT oauth2_replay_jtis_tenant_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);
CREATE INDEX oauth2_replay_jtis_expires_at_idx
    ON oauth2_replay_jtis (expires_at);

CREATE TABLE oauth2_access_token_denylist (
    tenant_id UUID NOT NULL,
    jti TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, jti),
    CONSTRAINT oauth2_access_token_denylist_tenant_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);
CREATE INDEX oauth2_access_token_denylist_expires_at_idx
    ON oauth2_access_token_denylist (expires_at);

CREATE UNLOGGED TABLE webauthn_sessions (
    tenant_id UUID NOT NULL,
    session_key TEXT NOT NULL,
    data JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, session_key),
    CONSTRAINT webauthn_sessions_tenant_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);
CREATE INDEX webauthn_sessions_expires_at_idx
    ON webauthn_sessions (expires_at);

CREATE TABLE login_throttle_counters (
    tenant_id UUID NOT NULL,
    kind TEXT NOT NULL,
    identifier_hash TEXT NOT NULL,
    failures INTEGER NOT NULL DEFAULT 0,
    window_expires_at TIMESTAMPTZ NOT NULL,
    locked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, kind, identifier_hash),
    CONSTRAINT login_throttle_counters_tenant_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
) WITH (fillfactor = 80);
CREATE INDEX login_throttle_counters_gc_idx
    ON login_throttle_counters (window_expires_at);

CREATE UNLOGGED TABLE saml_authnrequest_replays (
    tenant_id UUID NOT NULL,
    entity_id TEXT NOT NULL,
    request_id TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (tenant_id, entity_id, request_id),
    CONSTRAINT saml_authnrequest_replays_tenant_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);
CREATE INDEX saml_authnrequest_replays_expires_at_idx
    ON saml_authnrequest_replays (expires_at);
