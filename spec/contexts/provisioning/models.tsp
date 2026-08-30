namespace IdMagic.Contract {

@doc("A published stub the Application context owns. A ProvisioningConnection exists at most once per Application.")
model Application {
  id: string;
  tenant_id: string;
}


@doc("Published-language stub for a definition the Application context owns.")
enum AssignmentSubjectType {
  user: "user",
  group: "group",
}


@doc("A published stub the Application context owns, used to resolve the target set when `scope` is `assigned_only`.")
model ApplicationAssignment {
  application_id: string;
  subject_type: AssignmentSubjectType;
  subject_id: string;
}


@doc("No live ProvisioningConnection exists for the `application_id` given, or it belongs to another tenant (which is treated as nonexistent). RFC 9457 Problem Details with `type` = `urn:idmagic:error:provisioning_not_found`.")
model ProvisioningConnectionNotFoundError is ProblemDetails;


@doc("The Application already has a ProvisioningConnection; at most one connection exists per Application. RFC 9457 Problem Details with `type` = `urn:idmagic:error:provisioning_conflict`.")
model ProvisioningConnectionAlreadyExistsError is ProblemDetails;


@doc("No ProvisioningDelivery exists with the `delivery_id` given, or it does not belong to the connection given. RFC 9457 Problem Details with `type` = `urn:idmagic:error:provisioning_not_found`.")
model ProvisioningDeliveryNotFoundError is ProblemDetails;


@doc("RetryProvisioningDelivery was called on a ProvisioningDelivery whose state is not `dead_letter`. RFC 9457 Problem Details with `type` = `urn:idmagic:error:provisioning_conflict`.")
model ProvisioningDeliveryNotRetryableError is ProblemDetails;


@doc("The subject ProvisionOnDemand names is outside the scope of the connection; under `assigned_only` that means the subject is not assigned. RFC 9457 Problem Details with `type` = `urn:idmagic:error:provisioning_conflict`.")
model ProvisioningSubjectNotInScopeError is ProblemDetails;


@doc("How to authenticate to the downstream SCIM service provider. Either way the outbound request carries `Authorization: Bearer`; the method decides where that value comes from. `bearer_token` presents the stored credential directly. `oauth2_client_credentials` exchanges the stored client secret for an access token at the connection's token endpoint (RFC 6749 section 4.4) and presents that, reusing it until shortly before it expires.")
enum ProvisioningAuthMethod {
  bearer_token: "bearer_token",
  oauth2_client_credentials: "oauth2_client_credentials",
}


@doc("How the subjects to deliver are chosen. `assigned_only` reads the ApplicationAssignments; `all_users` is an advanced option covering every User of the tenant.")
enum ProvisioningScope {
  assigned_only: "assigned_only",
  all_users: "all_users",
}


@doc("The enabled state an administrator toggles. As with ApplicationStatus, deletion is not a state -- the row is removed, and the history stays in the ProvisioningConnectionDeleted event.")
enum ProvisioningConnectionStatus {
  active: "active",
  disabled: "disabled",
}


@doc("How healthy the connection looks to the delivery engine. `quarantined` stops delivery until ResumeProvisioningConnection.")
enum ProvisioningHealth {
  ok: "ok",
  degraded: "degraded",
  quarantined: "quarantined",
}


@doc("How the target groups are chosen while `push_groups` is on.")
enum ProvisioningGroupSelection {
  assigned_groups: "assigned_groups",
  explicit: "explicit",
}


@doc("Where the value of an AttributeMappingRule comes from. An `expression` (a function evaluation) is out of scope for the first version; only a direct reference and a fixed value are permitted.")
enum AttributeSourceKind {
  attribute: "attribute",
  constant: "constant",
}


@doc("When an AttributeMappingRule applies. Use `create_only` for a field whose manual change downstream should be respected.")
enum AttributeApplyOn {
  create_and_update: "create_and_update",
  create_only: "create_only",
}


@doc("How a deprovision trigger is translated for the downstream.")
enum ProvisioningDeprovisionAction {
  deactivate: "deactivate",
  delete: "delete",
  none: "none",
}


@doc("Which internal aggregate a ProvisioningDelivery or RemoteResourceLink refers to.")
enum ProvisioningSourceType {
  user: "user",
  group: "group",
}


@doc("The operation a ProvisioningDelivery applies downstream.")
enum ProvisioningOperation {
  create: "create",
  update: "update",
  deactivate: "deactivate",
  delete: "delete",
  membership_add: "membership_add",
  membership_remove: "membership_remove",
}


@doc("The value set of `ProvisioningDelivery.status`. `in_flight` covers everything from the moment the dispatcher attaches a\nJobs.Job until a terminal state is reached, and it stays `in_flight` through attempt retries at the Jobs level -- as with\nWorkflowRunLifecycle, a retry loop one layer down is not reflected in the state at this layer.\nThere is no non-terminal intermediate state called `failed`: `dead_letter` is a terminal state reached exactly once, when\nthe Job has used up `max_attempts`.")
enum ProvisioningDeliveryStatus {
  pending: "pending",
  in_flight: "in_flight",
  succeeded: "succeeded",
  dead_letter: "dead_letter",
}


@doc("Toggles for the operations the connection may perform downstream, following Okta. `sync_password` is out of scope.")
model ProvisioningFeatureFlags {
  create_users: boolean = true;
  update_users: boolean = true;
  deactivate_users: boolean = true;
  delete_users: boolean = false;
  push_groups: boolean = false;
}


@doc("The read-only capabilities the discovery of TestProvisioningConnection caches from the downstream /ServiceProviderConfig. They decide whether to fall back to PUT for a target that does not support PATCH.")
model ProvisioningCapabilities {
  supports_patch: boolean;
  supports_bulk: boolean;
  supports_filter: boolean;
  supports_etag: boolean;
  supports_sort: boolean;
  discovered_at: utcDateTime;
}


@doc("Non-sensitive credential metadata projected for the Application editor. It exposes neither the plaintext nor a hash, the same way ApplicationClientSecretCredentialMetadata does.")
model ProvisioningConnectionCredentialMetadata {
  credential_id: string;
  auth_method: ProvisioningAuthMethod;
  @doc("Token endpoint the client credentials are exchanged at. Present only for `auth_method` = `oauth2_client_credentials`. Not a secret: the secret is the client secret, which this projection never carries.")
  oauth2_token_url?: string;
  @doc("Client identifier presented at the token endpoint. Present only for `auth_method` = `oauth2_client_credentials`.")
  oauth2_client_id?: string;
  @doc("Scope requested at the token endpoint. Present only for `auth_method` = `oauth2_client_credentials`, and only when the connection sets one.")
  oauth2_scope?: string;
  created_at: utcDateTime;
  rotated_at?: utcDateTime;
}


@doc("The write-only credential input RegisterProvisioningConnection and UpdateProvisioningConnection take. The value never appears in a response, an audit log, or an error.")
model ProvisioningCredentialInput {
  auth_method: ProvisioningAuthMethod;
  @doc("Required when `auth_method` is `bearer_token`.")
  bearer_token?: string;
  @doc("Required when `auth_method` is `oauth2_client_credentials`.")
  oauth2_token_url?: url;
  oauth2_client_id?: string;
  oauth2_client_secret?: string;
  oauth2_scope?: string;
}


@doc("How the target groups are chosen while `push_groups` is on, and where their display names come from.")
model GroupPushConfig {
  selection: ProvisioningGroupSelection = ProvisioningGroupSelection.assigned_groups;
  @doc("The explicit list of `Group.id` values used when `selection` is `explicit`.")
  explicit_group_ids?: string[];
  @doc("The IdMagic Group attribute key projected onto the downstream `displayName`.")
  display_name_source: string = "name";
}


@doc("The declarative mapping of one output attribute. `target_path` is a SCIM attribute path, and may be simple, complex, a multi-valued filter, or an enterprise extension.")
model AttributeMappingRule {
  @minLength(1)
  target_path: string;
  source_kind: AttributeSourceKind;
  @doc("Required when `source_kind` is `attribute`: the attribute key on the IdMagic side.")
  source_key?: string;
  @doc("Required when `source_kind` is `constant`. Any JSON value, because the constant is written straight into the target attribute and the remote SCIM schema decides what that attribute may hold.")
  constant_value?: unknown;
  apply_on: AttributeApplyOn = AttributeApplyOn.create_and_update;
  @doc("When true, a value that cannot be resolved fails the delivery, fail-closed.")
  required: boolean = false;
  @doc("Used when the source resolves to nothing. Any JSON value, for the same reason as `constant_value`: the remote SCIM schema decides what the target attribute may hold.")
  default_value?: unknown;
}


@doc("The attributes compared to adopt an existing downstream resource on a 409 conflict. Correlation by `externalId` -- the IdMagic id -- is the primary route; this rule is the secondary one, for resolving a conflict with an existing resource that has no `externalId` set.")
model MatchingRule {
  conflict_match_attribute: string = "userName";
}


@doc("How a deprovision trigger is translated into a downstream action. Disabling a user always maps to deactivate, and that is\nnot configurable; so is removing a group membership mapping to a PATCH that removes members. The defaults lean towards\nthe less destructive option.")
model DeprovisionPolicy {
  @doc("The unassign trigger coming from the Application.")
  on_unassign: ProvisioningDeprovisionAction = ProvisioningDeprovisionAction.deactivate;
  @doc("The user soft-delete or delete trigger. Choosing `delete` purges after `grace_period_days`.")
  on_delete: ProvisioningDeprovisionAction = ProvisioningDeprovisionAction.deactivate;
  @doc("The group deletion or unassign trigger. A group has no equivalent of deactivate, so only `delete` and `none` are permitted.")
  on_group_deleted_or_unassigned: ProvisioningDeprovisionAction = ProvisioningDeprovisionAction.none;
  @minValue(0)
  grace_period_days: int64 = 0;
  @doc("When one sync would produce more deactivates and deletes than this, it is not run and the connection is quarantined instead.")
  accidental_deletion_count_threshold?: int64;
  @doc("The same guard as a share of the connection's subjects rather than a count.")
  @minValue(1)
  @maxValue(100)
  accidental_deletion_percent_threshold?: int64;
}


@doc("The outbound provisioning settings, of which at most one exists per Application. The source of truth is the User and Group\naggregates on the IdMagic side, and the downstream is a mirror. `scope` reuses the existing ApplicationAssignment rather\nthan carrying an assignment model of its own. Making `application_id` the identity expresses the one-connection-per-\nApplication constraint structurally.")
model ProvisioningConnection {
  application_id: string;
  tenant_id: string;
  status: ProvisioningConnectionStatus = ProvisioningConnectionStatus.active;
  @doc("The downstream SCIM v2 base URL. It must be https, and an internal or link-local IP and an arbitrary redirect are refused.")
  base_url: url;
  credential: ProvisioningConnectionCredentialMetadata;
  @doc("Cached by the discovery of TestProvisioningConnection. Unset until that has run.")
  capabilities?: ProvisioningCapabilities;
  feature_flags: ProvisioningFeatureFlags;
  scope: ProvisioningScope = ProvisioningScope.assigned_only;
  @doc("Meaningful only while `feature_flags.push_groups` is on.")
  group_push?: GroupPushConfig;
  @doc("Seeded with the SCIM core User default mappings at registration.")
  attribute_mappings: AttributeMappingRule[];
  matching: MatchingRule;
  deprovision_policy: DeprovisionPolicy;
  @minValue(1)
  rate_limit_per_minute: int64 = 60;
  @minValue(1)
  max_attempts: int64 = 8;
  @doc("Where failures and quarantines are notified.")
  notification_email?: string;
  @minValue(1)
  quarantine_after_consecutive_failures: int64 = 10;
  health: ProvisioningHealth = ProvisioningHealth.ok;
  @doc("The internal counter that decides quarantine. A successful delivery resets it to zero.")
  consecutive_failure_count: int64 = 0;
  last_full_sync_at?: utcDateTime;
  quarantined_at?: utcDateTime;
  quarantine_reason?: string;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("The durable record correlating an IdMagic User or Group with a downstream resource. It is kept so that a 409 or a 404 can be resolved idempotently.")
model RemoteResourceLink {
  connection_id: string;
  tenant_id: string;
  source_type: ProvisioningSourceType;
  @doc("The `User.id` or `Group.id` on the IdMagic side.")
  source_id: string;
  @doc("The id of the downstream SCIM resource.")
  remote_id: string;
  @doc("The `externalId` sent downstream; the IdMagic `source_id` by default.")
  external_id: string;
  @doc("Meaningful only when `capabilities.supports_etag` is true.")
  etag?: string;
  @doc("The `source_version` most recently applied. It is what resolves an out-of-order delivery.")
  last_synced_version: int64;
  updated_at: utcDateTime;
}


@doc("One delivery unit for one internal lifecycle event. The idempotency key\n`(tenant_id, connection_id, source_type, source_id, source_version)` is a uniqueness constraint, and enqueueing the same\nkey again is a no-op on the existing row. The dispatcher attaches `job_id` after enqueueing to Jobs, and it is unset\nuntil then.")
model ProvisioningDelivery {
  id: string;
  tenant_id: string;
  connection_id: string;
  source_type: ProvisioningSourceType;
  source_id: string;
  @doc("A monotonically rising counter the same-transaction capture emits. It is what resolves an out-of-order delivery.")
  source_version: int64;
  operation: ProvisioningOperation;
  status: ProvisioningDeliveryStatus = ProvisioningDeliveryStatus.pending;
  @doc("The `Jobs.JobRef.id` the dispatcher attaches after enqueueing.")
  job_id?: string;
  @doc("The sanitized error of the most recent failure. It carries no secret and no attribute value.")
  last_error?: string;
  created_at: utcDateTime;
  updated_at: utcDateTime;
  completed_at?: utcDateTime;
}


@doc("An administrator registered a new ProvisioningConnection on an Application.")
model ProvisioningConnectionRegistered {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
}


@doc("An administrator updated the settings of a ProvisioningConnection, the credential aside.")
model ProvisioningConnectionUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
}


@doc("An administrator set `ProvisioningConnection.status` to disabled. No new delivery is created.")
model ProvisioningConnectionDisabled {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
}


@doc("An administrator deleted a ProvisioningConnection. As with ApplicationStatus, the row itself is removed.")
model ProvisioningConnectionDeleted {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
}


@doc("An administrator updated the credential of a connection.")
model ProvisioningCredentialRotated {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
  credentialId: string;
}


@doc("The dispatcher attached a Jobs.Job to a ProvisioningDelivery and delivery began, the same shape as LifecycleWorkflowRunStarted.")
model ProvisioningDeliveryStarted {
  occurredAt: utcDateTime;
  tenantId: string;
  connectionId: string;
  deliveryId: string;
  jobId: string;
}


@doc("A user create or update reached the downstream and the delivery settled as succeeded.")
model UserProvisioned {
  occurredAt: utcDateTime;
  tenantId: string;
  connectionId: string;
  deliveryId: string;
  userId: string;
  remoteId: string;
}


@doc("A user deactivate or delete reached the downstream and the delivery settled as succeeded.")
model UserDeprovisioned {
  occurredAt: utcDateTime;
  tenantId: string;
  connectionId: string;
  deliveryId: string;
  userId: string;
  action: ProvisioningDeprovisionAction;
}


@doc("A user delivery used up `max_attempts` and settled as dead_letter. It is not emitted per attempt. A group delivery that dead-letters uses the same event.")
model UserProvisioningFailed {
  occurredAt: utcDateTime;
  tenantId: string;
  connectionId: string;
  deliveryId: string;
  sourceType: ProvisioningSourceType;
  sourceId: string;
  error: string;
}


@doc("A group create, update, deactivate, or delete reached the downstream and the delivery settled as succeeded.")
model GroupPushed {
  occurredAt: utcDateTime;
  tenantId: string;
  connectionId: string;
  deliveryId: string;
  groupId: string;
  remoteId: string;
}


@doc("A group membership change reached the downstream as a PATCH on members and the delivery settled as succeeded.")
model GroupMembershipPushed {
  occurredAt: utcDateTime;
  tenantId: string;
  connectionId: string;
  deliveryId: string;
  groupId: string;
}


@doc("`ProvisioningConnection.health` became quarantined, either from consecutive failures or from crossing the accidental-deletion guard. No further delivery is created.")
model ConnectionQuarantined {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
  reason: string;
  consecutiveFailures: int64;
}


@doc("An administrator lifted the quarantine through ResumeProvisioningConnection. The event was added because without a way out of quarantine there is no recovery.")
model ProvisioningConnectionQuarantineCleared {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
}


@doc("Every subject inside the scope StartFullResync targeted has converged.")
model FullResyncCompleted {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
  totalSubjects: int64;
  succeededCount: int64;
  failedCount: int64;
}

}
