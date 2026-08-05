---
context: idmanagement
updated_at: 2026-08-06
---

# Architecture: idmanagement

## Overview

The `IdManagement` context owns the tenant-scoped catalog of principals: the `User`, `Group`, and
`Agent` aggregates, and the attribute schema user profiles are validated against. It does not own
credential verification or login sessions (`Authentication`) or OAuth2 client credentials and token
issuance (`OAuth2`) — it owns the principal records those contexts authenticate against and issue
tokens for. `User`, `Group`, and `Agent` are separate feature vertical slices (`user/`, `group/`,
`agent/`), each with its own domain, ports, use cases, and adapters, per the feature-slice split
recorded in [ADR-130](../../decisions/ADR-130-idmanagement-feature-vertical-slice.md). Read user
lifecycle first, then the attribute model user profiles are built from, then `Group`, then `Agent`.

## User Lifecycle: Deletion and Anonymization

Deletion is anonymization, not physical removal. `User.lifecycle.status` transitions to `Deleted` — a
terminal state with no transition back, reachable from any prior status — and the aggregate is
rewritten in place rather than dropped, because `AdminAuditEvent` and other append-only records
reference `sub` and a hard delete would break that reference while also erasing the operational
distinction between "deleted" and merely "disabled"
([ADR-036](../../decisions/ADR-036-user-deletion-and-anonymization.md)). `sub` is retained forever and
never reused.

The tombstone replacement clears, atomically, every field that could re-identify or re-authenticate the
user: `preferred_username` becomes `deleted:<sub>`, `name`/`given_name`/`family_name`/`email` are
cleared, `email_verified` and `mfa_enrolled` reset to `false`, `password_hash` is emptied, `roles`
becomes empty, the entire sparse `attributes` map is cleared, and `lifecycle.status` becomes `Deleted`
(ADR-036, refined by [ADR-039](../../decisions/ADR-039-user-profile-shape.md) §6 once profile attributes
moved into the attribute bag). `preferred_username` is freed for reuse once tombstoned: a partial
unique index scoped to non-deleted rows is what keeps the tombstone value collision-free against future
users while still letting the freed name be claimed again (ADR-036).

Deletion cascades synchronously to every aggregate a deleted user must no longer reach through:
`Consent`, `RefreshTokenRecord`, `LoginSession`, `PasswordHistory`, `MfaFactor`, and active
`DeviceAuthorization` records are all removed for that `sub`. The PostgreSQL-backed cascade runs inside
one transaction; Valkey-backed session/device-code state is deleted per-store, since that state is
volatile by nature and a short inconsistency window there is an acceptable trade against transaction
complexity (ADR-036).

Delete is idempotent — calling it again on an already-tombstoned user is a no-op that returns success
without re-emitting the audit event, so retries and concurrent admin actions never surface as failures
or duplicate the audit trail (ADR-036). A self-destruct guard rejects a delete where the actor and
target are the same principal and the target holds `admin` or `system_admin`, since an admin deleting
their own privileged account is not a path any interactive flow needs to allow (ADR-036). Every delete
emits a `UserDeleted` audit event carrying `actorSub`/`targetSub`/`reason`/`occurredAt`; because `sub`
and the tombstone persist, "who deleted what, and when" stays reconstructable after the anonymization
(ADR-036).

## User Profile: Thin Core and Attribute Bag

`User` keeps a typed core limited to what identity, authentication, and RBAC need at the type level —
`sub`, `tenant_id`, `preferred_username`, `password_hash`, `email`, `email_verified`, `mfa_enrolled`,
`roles`, `name`/`given_name`/`family_name`, `lifecycle`, and timestamps. Giving every user ~25
rarely-used optional OIDC/SCIM fields at the type and storage level was found to bloat the model for
tenants that use almost none of them
([ADR-039](../../decisions/ADR-039-user-profile-shape.md)). Every other profile attribute — remaining
OIDC §5.1 optional claims (`middle_name`, `nickname`, `picture`, `phone_number`, `address_*`, …),
SCIM-style organizational attributes (`title`, `department`, `manager_sub`, …), and tenant-defined
custom fields — lives in a single sparse `attributes: Map<String, AttributeValue>`, where only keys
that actually carry a value consume space. OIDC's `address` claim is stored as flat keys
(`address_formatted`, `address_locality`, …) rather than a nested structure, keeping `AttributeValue` a
plain sum type (string/number/boolean/date/string array); it is reassembled into the nested `address`
object only when UserInfo/ID Token claims are built (ADR-039).

Lifecycle is a single source of truth: `User.lifecycle.status`
(`Active`/`Disabled`/`Locked`/`Staged`/`Suspended`/`Deleted`) plus `status_changed_at` replaced separate
`disabled_at`/`deleted_at` columns, since "when did this transition happen" already lives in the
timestamped `UserDisabled`/`UserDeleted` audit events and a second copy of that timestamp on the
aggregate was redundant (ADR-039). Only `status == Active` authenticates; every other status —
including the zero-value, which resolves to `Active` by default — is treated as non-authenticating.

### Attribute Definitions (`UserAttributeDef`)

Both the OIDC/SCIM built-in attributes and tenant-defined custom attributes are governed by the same
`UserAttributeDef` mechanism, so admins configure one schema shape instead of two
([ADR-040](../../decisions/ADR-040-user-custom-attribute-policy.md)). Definitions come from two tiers
that combine into one effective schema:

- a **builtin catalog**, `BuiltinUserAttributeDefs()`, defined in code and shared by every tenant — the
  OIDC §5.1 optional claims and SCIM `enterprise:User`-equivalent organizational attributes;
- a **tenant schema**, `TenantUserAttributeSchema`, a separate aggregate keyed by `tenant_id` rather
  than embedded in the `Tenant` aggregate, because its schema churns faster than tenant settings, is a
  candidate for its own table later, and needs an explicit cascade path on tenant deletion (ADR-040).

Effective definitions are builtin ∪ tenant; a tenant schema that redefines a builtin key is rejected
outright. Each `UserAttributeDef` carries a `key` (snake_case, letter-first), a `type`
(`string`/`number`/`boolean`/`date`/`string_array`), `required`, `editable_by_user`, an optional
`claim_name`/`oidc_scope` pair that only takes effect at `visibility == claim_exposed`, and `visibility`
itself — one of `private`/`self_readable`/`admin_readable`/`claim_exposed`, of which only
`claim_exposed` is ever disclosed to a relying party. `pii` defaults to `true`: unless a definition
explicitly opts out, its stored and audited values are SHA-256 hashed rather than kept in the clear, a
safe-by-default choice that puts the visibility ceiling ahead of tenant convenience (ADR-040).

`ValidateAttributes` checks a `User.attributes` map against the effective schema before it is
persisted — rejecting undefined keys, missing required values, and type mismatches, and enforcing that
each `AttributeValue` populates only the field its declared `type` selects. The self-service path
(`UpdateUserProfile` / `/api/account/profile`) additionally restricts writes to
`editable_by_user == true` attributes and merges by key rather than replacing the whole map, so a
self-service edit cannot overwrite admin-managed attributes it has no business touching; it also
discloses only `self_readable`/`claim_exposed` attributes back to the user (ADR-040). On deletion, the
entire `attributes` map is cleared along with the typed core, so the sparse bag never outlives the
tombstone (ADR-036 refined by ADR-039 §6).

## Group Aggregate and Effective Roles

`Group` is a tenant-scoped aggregate — `(id, tenant_id, name, description?, roles[], created_at,
updated_at?)` — introduced so a bundle of roles ("sales team = `catalog:read` + `invoice:read`") can be
granted and revoked as one unit instead of editing every affected user's `roles` individually on every
reorg ([ADR-038](../../decisions/ADR-038-group-aggregate-and-effective-roles.md)). `id` is an immutable
generated `group_<uuid>`; `name` is an editable display name unique within the tenant, enforced by a
`(tenant_id, name)` unique index. Cross-tenant membership is rejected outright: `AddMember` loads the
target `User` and refuses if it is absent or belongs to a different tenant.

A user's effective roles are `user.roles ∪ ⋃_{g ∈ user.groups} g.roles` — a plain union, sorted and
deduplicated, with no subtraction or precedence rules, because a deny/minus operator would add
evaluation-order complexity for a case a flat union already covers (ADR-038). When a user belongs to no
group, effective roles collapse back to `user.roles`, so introducing `Group` changes nothing for
existing accounts. Two surfaces resolve against effective roles rather than raw `user.roles`: the admin
console's RBAC gates and the `/account` self-view, so a user can see which of their effective
permissions come from group membership. `User.roles` itself is kept as an individual override path for
users not covered by any group. Roles are not projected into token claims by default — that mapping
does not exist yet for either individual or group-derived roles, and is deliberately out of scope here
(ADR-038).

Membership operations are idempotent: adding an existing member or removing a non-member is a no-op
that does not re-emit a domain event, matching how Okta and Keycloak treat their membership APIs
(ADR-038). Group CRUD and membership changes emit both an `AdminAuditEvent` and one of
`GroupCreated`/`GroupUpdated`/`GroupDeleted`/`GroupMemberAdded`/`GroupMemberRemoved`; deleting a group
cascades its memberships, emitting `GroupMemberRemoved` per member before the final `GroupDeleted`.

## Agent Principal

`Agent` is a third first-class principal type alongside `User` and the credential primitives `OAuth2`
owns. It exists because retrofitting agent-specific concerns onto `OAuth2Client` would leave
autonomous/supervised AI agents indistinguishable from generic M2M clients for audit and policy
purposes, while giving agents an independent credential and cryptographic surface would double an
attack surface that already exists on `OAuth2Client`
([ADR-048](../../decisions/ADR-048-agent-as-first-class-non-human-principal.md)). IdManagement owns the
aggregate itself — identity, ownership, lifecycle, and credential binding. The delegation mechanics
that let an agent act as an actor in a token-exchange chain belong to `OAuth2` and are covered in that
context's design record.

The `Agent` aggregate holds `(id, tenant_id, display_name, kind, status, owner, purpose, created_at,
updated_at, disabled_at?, killed_at?)`. `id` is a URL-safe slug; `kind` distinguishes `autonomous` from
`supervised` agents, a declared statement of how much human oversight the agent's actions get.
Registration, lookup, and mutation are all tenant-scoped, matching the tenant boundary the rest of
IdManagement's aggregates follow (ADR-032/ADR-034).

An `Agent` carries no credential primitives of its own — it binds to one or more existing `OAuth2Client`
registrations through `AgentCredentialBinding`, so a single credential and key-management surface serves
both generic M2M clients and agents, and `Agent` adds only the ownership, purpose, and lifecycle layer
on top (ADR-048). Every agent is required to have an owner (a `User` or an owning `Group`); an unowned
agent cannot be registered, and owner offboarding is designed to propagate to the agents that owner owns
rather than leaving orphaned non-human identities behind.

Lifecycle is `active`/`disabled`/`killed`. `disabled` is a reversible operational stop; `killed` is a
one-way emergency stop. Both are enforced fail-closed at the token-issuance boundary each binding's
`OAuth2Client` flows through: an agent whose status is not `active` gets no new token, and any ambiguity
in that check resolves toward not issuing rather than issuing — the same posture ADR-048 requires of
kill-switch handling generally. `AgentRegistered`/`AgentUpdated`/`AgentDisabled`/`AgentEnabled`/
`AgentDeleted`/`AgentOwnerChanged` are emitted to the existing audit/outbox path. Agent CRUD and the
kill-switch are gated by a dedicated `AdminAgentsManage` permission rather than reusing generic admin
roles.
