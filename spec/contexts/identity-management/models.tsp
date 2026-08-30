namespace IdMagic.Contract {

@doc("The sum type discriminator of an attribute value. Published-language stub for a definition the Tenancy context owns.")
enum AttributeType {
  String: "string",
  Number: "number",
  Boolean: "boolean",
  Date: "date",
  StringArray: "string_array",
}


@doc("How a Group's membership is managed. It cannot be changed after creation.")
enum GroupMembershipType {
  manual: "manual",
  dynamic: "dynamic",
}


@doc("The one path that created a GroupMember.")
enum GroupMembershipSource {
  manual: "manual",
  dynamic_rule: "dynamic_rule",
}

@doc("The tenant-scoped role bundle aggregate. It grants its `roles` to every member User at once. `id` is an immutable generated identifier and `name` an editable label unique within the tenant; a CSV row resolves its target by `id` first and falls back to `name`, so a row carrying both a known `id` and a `name` held by a different Group is refused rather than resolved. `email` is a contact address and is never used to authenticate, and `attributes` is a sparse bag of custom attributes validated against the tenant's TenantGroupAttributeSchema; both round-trip through CSV, `attributes` as `custom:<key>` columns. Deletion is immediate and irreversible: Group has no reservation state, so a CSV row asks for it through the explicit `lifecycle_action` column rather than through a target-state column.")
model Group {
  @minLength(1)
  @maxLength(64)
  id: string;
  tenant_id: string = "default";
  @minLength(1)
  @maxLength(100)
  name: string;
  @maxLength(500)
  description?: string;
  @doc("The contact email address of the group. Unlike `User.email` it has no authentication or verification flow.")
  @maxLength(254)
  email?: string;
  @doc("The custom attributes, validated against the tenant's TenantGroupAttributeSchema.")
  attributes?: Record<AttributeValue>;
  roles: string[];
  membership_type: GroupMembershipType = GroupMembershipType.manual;
  created_at: utcDateTime;
  updated_at?: utcDateTime;
}


@doc("The definition of one custom attribute of a Group. Unlike the UserAttributeDef of User, Group has no built-in OIDC or SCIM catalogue, no self-service editor, and no claim exposure, so this is the reduced form carrying only the type and whether it is required.")
model GroupAttributeDef {
  @minLength(1)
  key: string;
  @maxLength(100)
  label?: string;
  type: AttributeType;
  multi_valued: boolean = false;
  required: boolean = false;
}


@doc("The state of a full re-evaluation for one rule version.")
enum DynamicMembershipEvaluationStatus {
  queued: "queued",
  running: "running",
  succeeded: "succeeded",
  failed: "failed",
}


@doc("The operational state of an Agent. `active` runs normally, `disabled` is a reversible suspension, and `killed` is an emergency stop that is one-way and terminal. Anything but `active` issues no new token, fail-closed.")
enum AgentStatus {
  Active: "active",
  Disabled: "disabled",
  Killed: "killed",
}


@doc("How autonomous an Agent is. `autonomous` runs on its own; `supervised` runs under human supervision, and obtains a token only through an issuance path that records a human approval (REQ-OAUTH2-050). The requirement is decided by the absence of `autonomous` rather than the presence of `supervised`, so a value the server does not recognise requires approval too.")
enum AgentKind {
  Autonomous: "autonomous",
  Supervised: "supervised",
}


@doc("`roles` contains a role the server does not know. RFC 9457 Problem Details with `type` = `urn:idmagic:error:invalid_role`.")
model InvalidRoleError is ProblemDetails;


@doc("The operator tried to delete their own User. RFC 9457 Problem Details with `type` = `urn:idmagic:error:self_delete_forbidden`.")
model SelfDeleteForbiddenError is ProblemDetails;


@doc("The operator tried to disable their own User. RFC 9457 Problem Details with `type` = `urn:idmagic:error:self_disable_forbidden`.")
model SelfDisableForbiddenError is ProblemDetails;


@doc("The User that RestoreAdminUser targets is not in the `pending_deletion` state. RFC 9457 Problem Details with `type` = `urn:idmagic:error:not_pending_deletion`.")
model UserNotPendingDeletionError is ProblemDetails;


@doc("RestoreAdminUser was called after the deletion grace period had elapsed. RFC 9457 Problem Details with `type` = `urn:idmagic:error:restore_grace_expired`.")
model RestoreGracePeriodExpiredError is ProblemDetails;


@doc("A custom attribute value does not satisfy the tenant attribute schema. RFC 9457 Problem Details with `type` = `urn:idmagic:error:invalid_attribute`, the same code the group attribute rejection uses; the endpoint tells the two apart.")
model InvalidUserAttributeError is ProblemDetails;


@doc("A custom attribute value on a Group does not satisfy the tenant Group attribute schema (TenantGroupAttributeSchema). RFC 9457 Problem Details with `type` = `urn:idmagic:error:invalid_attribute`, the same code the user attribute rejection uses; the endpoint tells the two apart.")
model InvalidGroupAttributeError is ProblemDetails;


@doc("SetUserRequiredAction was given a RequiredAction value the server does not know. RFC 9457 Problem Details with `type` = `urn:idmagic:error:invalid_required_action`.")
model InvalidRequiredActionError is ProblemDetails;


@doc("The `name` of a Group is empty. RFC 9457 Problem Details with `type` = `urn:idmagic:error:group_name_required`.")
model GroupNameRequiredError is ProblemDetails;


@doc("The dynamic group rule (a CEL expression) failed to compile or validate: an unknown attribute, a non-boolean result, a syntax error, and so on. RFC 9457 Problem Details with `type` = `urn:idmagic:error:invalid_dynamic_group_rule`.")
model InvalidDynamicGroupRuleError is ProblemDetails;


@doc("The `name` of an Agent is empty. RFC 9457 Problem Details with `type` = `urn:idmagic:error:agent_name_required`.")
model AgentNameRequiredError is ProblemDetails;


@doc("`owner_user_id` is absent for an Agent kind that requires an owner. RFC 9457 Problem Details with `type` = `urn:idmagic:error:agent_owner_required`.")
model AgentOwnerRequiredError is ProblemDetails;


@doc("The `owner_user_id` given does not resolve to a User in the tenant. RFC 9457 Problem Details with `type` = `urn:idmagic:error:agent_owner_not_found`.")
model AgentOwnerNotFoundError is ProblemDetails;


@doc("`kind` is absent from an Agent registration. The kind decides at runtime whether the Agent can obtain a token without human approval, so no default value stands in for it. RFC 9457 Problem Details with `type` = `urn:idmagic:error:agent_kind_required`.")
model AgentKindRequiredError is ProblemDetails;


@doc("`kind` is not a value AgentKind declares. It is rejected rather than rounded to a known one, because rounding would decide the approval requirement on the caller's behalf. RFC 9457 Problem Details with `type` = `urn:idmagic:error:invalid_agent_kind`.")
model InvalidAgentKindError is ProblemDetails;


@doc("The email address given is not a well-formed address. RFC 9457 Problem Details with `type` = `urn:idmagic:error:invalid_email`.")
model InvalidEmailError is ProblemDetails;


@doc("The new email address given is the same as the current one. RFC 9457 Problem Details with `type` = `urn:idmagic:error:email_unchanged`.")
model EmailUnchangedError is ProblemDetails;


@doc("An export column named in the request is not on the allow list for the target. RFC 9457 Problem Details with `type` = `urn:idmagic:error:invalid_columns`.")
model InvalidExportColumnsError is ProblemDetails;


@doc("The export target kind is not supported. RFC 9457 Problem Details with `type` = `urn:idmagic:error:invalid_target`.")
model InvalidExportTargetError is ProblemDetails;


@doc("The status filter given is not permitted for the export target. RFC 9457 Problem Details with `type` = `urn:idmagic:error:invalid_filter`.")
model InvalidExportFilterError is ProblemDetails;


@doc("The operational lifecycle attributes of a User. `status` is the single source of truth and matches the\nUserLifecycle state machine in the states section; the old `disabled_at` and `deleted_at` were folded into\n`status` plus `status_changed_at`. A tombstone is `status == deleted`.")
model UserLifecycle {
  status: UserStatus = UserStatus.Active;
  @doc("When the current status was reached. The detailed transition history lives in the audit events.")
  status_changed_at?: utcDateTime;
  last_login_at?: utcDateTime;
  password_changed_at?: utcDateTime;
  required_actions?: RequiredAction[];
}


@doc("The value of one attribute, as a sum type: only the field `type` names is set. It is shared by the built-in OIDC attributes and the tenant-defined custom ones.")
model AttributeValue {
  type: AttributeType;
  string?: string;
  number?: float64;
  boolean?: boolean;
  date?: string;
  string_array?: string[];
}

@doc("The canonical user aggregate of the IdP. Protocol subjects are projected from this id, and profile extensions are kept in the sparse `attributes`.")
model User {
  @minLength(1)
  id: string;
  tenant_id: string = "default";
  @minLength(1)
  @maxLength(100)
  preferred_username: string;
  password_hash?: string;
  @maxLength(200)
  name?: string;
  @maxLength(100)
  given_name?: string;
  @maxLength(100)
  family_name?: string;
  @maxLength(254)
  email?: string;
  email_verified: boolean = false;
  mfa_enrolled: boolean = false;
  roles: string[] = #[];
  lifecycle: UserLifecycle;
  attributes?: Record<AttributeValue>;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("The JSON request in which an administrator creates a user. The PasswordPolicy of the Authentication context applies to the initial password.")
model AdminUserCreateRequest {
  @minLength(1)
  @maxLength(100)
  preferred_username: string;
  password: string;
  @maxLength(200)
  name?: string;
  @maxLength(254)
  email?: string;
  email_verified: boolean = false;
  roles: string[] = #[];
}


@doc("The JSON request in which an administrator updates a user's mutable profile and roles. `id` and `password_hash` are left alone.")
model AdminUserUpdateRequest {
  @minLength(1)
  @maxLength(100)
  preferred_username?: string;
  @maxLength(200)
  name?: string;
  @maxLength(200)
  given_name?: string;
  @maxLength(200)
  family_name?: string;
  @maxLength(254)
  email?: string;
  email_verified?: boolean;
  roles?: string[];
  @doc("Replaces the whole attribute bag when present, validated against the tenant's TenantUserAttributeSchema. Unlike the self-service UpdateUserProfile, an administrator may set attributes an end user cannot edit.")
  attributes?: Record<AttributeValue>;
}


@doc("The user list of the administration API, together with the exact total number of users in the tenant, excluding the deleted ones and independent of `query` and `status`.")
model AdminUserListResponse {
  users: AdminUserResponse[];
  @doc("The exact number of Users in the tenant whose lifecycle status is not `deleted`. It is unaffected by `query` and `status`.")
  total_users?: int64;
}


@doc("How the administration API represents a user. `password_hash` is deliberately not part of the contract.")
model AdminUserResponse {
  id: string;
  @doc("The tenant it belongs to.")
  tenant_id: string;
  preferred_username: string;
  name?: string;
  email?: string;
  email_verified: boolean;
  mfa_enrolled: boolean;
  roles: string[];
  @doc("The actions forced at the next sign-in.")
  required_actions?: RequiredAction[];
  @doc("When authentication last completed.")
  last_login_at?: utcDateTime;
  @doc("When the password was last changed.")
  password_changed_at?: utcDateTime;
  @doc("The previous-generation field holding when the user was disabled. `User.lifecycle.status` plus `status_changed_at` is the single source of truth, and this field survives only for backward compatibility of the current response. It is an existing gap that needs its own cleanup, outside the scope of this migration.")
  disabled_at?: utcDateTime;
  created_at: utcDateTime;
  @doc("It equals `created_at` at creation and is updated on every administrative operation afterwards.")
  updated_at: utcDateTime;
}


@doc("The JSON request of SetUserRequiredAction. `action` is the wire value of the RequiredAction enum.")
model AdminRequiredActionRequest {
  action: RequiredAction;
}


@doc("What `/api/account/profile` returns (self-service): a display projection of the calling user themselves.\n`attributes` carries only what they may read of their own (`self_readable` or `claim_exposed`), and\n`editable_attributes` lists the definitions with `editable_by_user=true`, which the UI uses to render its form.")
model AccountProfileResponse {
  id: string;
  preferred_username: string;
  name?: string;
  given_name?: string;
  family_name?: string;
  email?: string;
  email_verified: boolean;
  mfa_enrolled: boolean;
  status: UserStatus;
  attributes: Record<AttributeValue>;
  editable_attributes: UserAttributeDef[];
}


@doc("The self-service profile update request. It accepts only the display name and the attributes marked\n`editable_by_user=true`. `attributes` merges per key -- the keys given are upserted and the rest are left alone -- and the\nadministrator-managed attributes (status, roles, organization) are out of its reach.")
model AccountProfileUpdateRequest {
  name?: string;
  given_name?: string;
  family_name?: string;
  attributes?: Record<AttributeValue>;
}


@doc("What the CSV user import endpoints return when a preview or an apply has been accepted. The job runs asynchronously; poll GetAdminUserImport with the `id` for progress and results.")
model UserImportJobRef {
  @doc("Job id, used to poll GetAdminUserImport and to start an apply from a preview.")
  id: string;
  @doc("Job status, from the Jobs context lifecycle.")
  status: string;
  mode: UserImportMode;
}


@doc("The public view of a CSV user import job. A preview causes no side effects; an apply joins a succeeded preview of the same tenant on its `source_sha256`. Row errors name the column and a stable code, never the value, and neither passwords nor hashes are read or stored.")
model UserImportJob {
  @doc("Job id.")
  id: string;
  @doc("Job status, from the Jobs context lifecycle.")
  status: string;
  mode: UserImportMode;
  @doc("Counts and digests the job produced. Absent until the job has run.")
  result?: UserImportResult;
  @doc("One page of row errors, paginated by the standard cursor headers. Empty while the job is still running.")
  errors?: UserImportRowError[];
}


@doc("Counts and digests a finished CSV user import job produced.")
model UserImportResult {
  @doc("SHA-256 of the uploaded CSV. An apply must name a preview with the same digest.")
  source_sha256: string;
  @doc("For an apply, the preview job it was started from.")
  preview_job_id?: string;
  total_rows: int64;
  created_rows: int64;
  updated_rows: int64;
  unchanged_rows: int64;
  @doc("Compatibility field for older clients: created_rows + updated_rows + unchanged_rows. New clients use the per-operation counts.")
  accepted_rows?: int64;
  rejected_rows: int64;
  @doc("Reference to the stored error artifact, when the job produced one.")
  error_artifact_ref?: string;
  @doc("SHA-256 of the stored error artifact.")
  error_artifact_sha256?: string;
  @doc("How many row errors the job produced in total, not how many this page carries.")
  error_total: int64;
}


@doc("A safe validation or application error for one CSV row. It returns no value, only the column name and a stable code.")
model UserImportRowError {
  row: int64;
  column?: string;
  code: string;
}


@doc("The stage of a CSV user import job. A preview is a read-only plan; an apply replans a succeeded preview payload against the current state and applies it.")
enum UserImportMode {
  dry_run: "dry_run",
  preview: "preview",
  apply: "apply",
}


@doc("The configurable safety policy every CSV export and import shares, User and Group alike. It defaults to 100,000 data rows, 64 MiB per file, and 64 KiB per field. A successful export always sits inside the same policy, so it can be previewed as it is. The ceilings do not express tenant capacity; they are an operational boundary on a single asynchronous artifact.")
model CsvTransferPolicy {
  max_rows: int64;
  max_bytes: int64;
  max_field_bytes: int64;
}


@doc("The row operation a Group CSV asks for beyond an upsert. It is a write-only column: a Group export always writes it empty, so re-applying an unedited export plans every row as unchanged. The vocabulary is closed -- Group has no reservation state, so `delete` is the only value, and an unknown one refuses the row rather than being rounded to a known one.")
enum GroupCsvLifecycleAction {
  delete: "delete",
}


@doc("What the CSV group import endpoints return when a preview or an apply has been accepted. The job runs asynchronously; poll GetAdminGroupImport with the `id` for progress and results.")
model GroupImportJobRef {
  @doc("Job id, used to poll GetAdminGroupImport and to start an apply from a preview.")
  id: string;
  @doc("Job status, from the Jobs context lifecycle.")
  status: string;
  mode: GroupImportMode;
}


@doc("The public view of a CSV group import job. A preview causes no side effects; an apply joins a succeeded preview of the same tenant on its `source_sha256`. Row errors name the column and a stable code, never the value.")
model GroupImportJob {
  @doc("Job id.")
  id: string;
  @doc("Job status, from the Jobs context lifecycle.")
  status: string;
  mode: GroupImportMode;
  @doc("Counts and digests the job produced. Absent until the job has run.")
  result?: GroupImportResult;
  @doc("One page of row errors, paginated by the standard cursor headers. Empty while the job is still running.")
  errors?: GroupImportRowError[];
}


@doc("Counts and digests a finished CSV group import job produced. `deleted_rows` and `deleted_memberships` are reported apart from the other operations because deletion is irreversible and cascades to every membership of the Group.")
model GroupImportResult {
  @doc("SHA-256 of the uploaded CSV. An apply must name a preview with the same digest.")
  source_sha256: string;
  @doc("For an apply, the preview job it was started from.")
  preview_job_id?: string;
  total_rows: int64;
  created_rows: int64;
  updated_rows: int64;
  unchanged_rows: int64;
  @doc("How many rows ask to delete a Group, or deleted one.")
  deleted_rows: int64;
  @doc("How many memberships those deletions release. It counts the cascade the deletion carries, not rows of the CSV.")
  deleted_memberships: int64;
  rejected_rows: int64;
  @doc("Reference to the stored error artifact, when the job produced one.")
  error_artifact_ref?: string;
  @doc("SHA-256 of the stored error artifact.")
  error_artifact_sha256?: string;
  @doc("How many row errors the job produced in total, not how many this page carries.")
  error_total: int64;
}


@doc("A safe validation or application error for one CSV group row. It returns no value, only the column name and a stable code.")
model GroupImportRowError {
  row: int64;
  column?: string;
  code: string;
}


@doc("The stage of a CSV group import job. A preview is a read-only plan; an apply replans a succeeded preview payload against the current state and applies it.")
enum GroupImportMode {
  preview: "preview",
  apply: "apply",
}


@doc("The resource kind a CSV export targets. The column definitions are closed within a per-kind allow list, so an unreviewed attribute cannot leave on its own.")
enum DataExportTargetKind {
  user: "user",
  group: "group",
  group_membership: "group_membership",
}


@doc("The output format of an export. At first, CSV (RFC 4180) alone.")
enum DataExportFormat {
  csv: "csv",
}


@doc("The state of a resource export job, matching DataExportLifecycle. `succeeded` means the file can be downloaded; `expired` means the file itself has been purged by its TTL and only the metadata and the audit trail remain.")
enum DataExportStatus {
  queued: "queued",
  running: "running",
  succeeded: "succeeded",
  failed: "failed",
  canceled: "canceled",
  expired: "expired",
}


@doc("One column of the allow list for an export kind. `key` is the stable machine name, which is also the CSV header; `label` is for display alone; and `pii` says whether it holds personal information. The User allow list holds 13 built-in core columns plus `attr:<key>` -- 27 built-in extension attributes shared by every tenant, drawn from the OIDC profile, phone, and address claims and the SCIM enterprise organizational attributes -- and `custom:<key>`, the tenant-defined attributes resolved per tenant from the effective TenantUserAttributeSchema. The Group allow list is the set of import-compatible columns -- `id`, `name`, `description`, `email`, `membership_type`, `roles`, `dynamic_rule_expression`, `dynamic_rule_enabled`, `lifecycle_action`, `created_at`, `updated_at` -- plus `custom:<key>`, the tenant-defined attributes resolved per tenant from the TenantGroupAttributeSchema. Group has no built-in attribute catalogue to union, so it carries no `attr:<key>` counterpart. An unedited Group export can be previewed as it is. Sensitive values such as `password_hash`, secrets, tokens, recovery codes, and MFA secrets are on no allow list at all.")
model DataExportColumn {
  key: string;
  label: string;
  pii: boolean = false;
}


@doc("The input starting an export. The target kind comes from the endpoint path and so is not in the body. It carries the output format, the column selection (a subset of the allow list), and the filter carried over from the list screen. A column key outside the allow list is refused.")
model DataExportRequest {
  format: DataExportFormat = DataExportFormat.csv;
  columns: string[];
  filter?: Record<string>;
}


@doc("The public view of a resource export job, which doubles as its auditable record. It carries neither the CSV itself nor any PII value: only the target kind, the selected column keys, the counts, the byte size, the state, the expiry, and who asked for it.")
model DataExportJob {
  id: string;
  status: DataExportStatus;
  target: DataExportTargetKind;
  format: DataExportFormat;
  requested_columns: string[];
  filename?: string;
  total_rows?: int64;
  byte_size?: int64;
  error_code?: string;
  requested_by: string;
  created_at: utcDateTime;
  completed_at?: utcDateTime;
  expires_at?: utcDateTime;
  downloadable: boolean = false;
}


@doc("The finished export file returned on download. `content` is the CSV itself and may hold PII, so it is returned as a `Content-Disposition: attachment` only to an authorized administrator, and only for a job that has succeeded and not yet expired.")
model DataExportFile {
  filename: string;
  content_type: string;
  byte_size: int64;
  content: string;
}


@doc("Why an export failed. It returns no value, only a stable code.")
model DataExportError {
  code: string;
}


@doc("An administrator asked for a resource export and the job was accepted. It carries no CSV value: only the target kind, the selected column keys, and who asked.")
model DataExportRequested {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  exportId: string;
  target: DataExportTargetKind;
  requestedColumns: string[];
}


@doc("A worker started generating a resource export.")
model DataExportStarted {
  occurredAt: utcDateTime;
  tenantId: string;
  exportId: string;
  target: DataExportTargetKind;
}


@doc("A resource export finished generating and the file became downloadable. Only the row count and the byte size are recorded; no value is.")
model DataExportSucceeded {
  occurredAt: utcDateTime;
  tenantId: string;
  exportId: string;
  target: DataExportTargetKind;
  totalRows: int64;
  byteSize: int64;
}


@doc("Generating a resource export failed, and the incomplete file terminated without becoming downloadable.")
model DataExportFailed {
  occurredAt: utcDateTime;
  tenantId: string;
  exportId: string;
  target: DataExportTargetKind;
  errorCode: string;
}


@doc("An administrator cancelled a resource export before it terminated.")
model DataExportCanceled {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  exportId: string;
}


@doc("An administrator downloaded a finished export file. It is kept in the audit trail as a high-risk operation that extracts PII in bulk.")
model DataExportDownloaded {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  exportId: string;
  target: DataExportTargetKind;
}


@doc("The retention period elapsed and the export file itself was purged. The metadata and the audit trail remain.")
model DataExportExpired {
  occurredAt: utcDateTime;
  tenantId: string;
  exportId: string;
}


@doc("What `/api/account/summary` returns, the home of the end-user account portal. It is the calling user's own\naccount summary, a separate contract from the AccountContext of the administration shell (`/api/auth/account`).\nIt carries no roles, and returns the last sign-in, the password change time, the MFA state, and the outstanding\nrequired actions for the summary card of the portal home.")
model AccountSummary {
  id: string;
  preferred_username: string;
  name?: string;
  email?: string;
  email_verified: boolean;
  mfa_enrolled: boolean;
  status: UserStatus;
  last_login_at?: utcDateTime;
  password_changed_at?: utcDateTime;
  required_actions: RequiredAction[];
}


@doc("The request raising a primary email change (self-service). It only sends a confirmation link to the\nnew address; `User.email` does not change until ConfirmEmailChange completes.")
model AccountEmailChangeRequest {
  new_email: string;
}


@doc("The request confirming the one-time link sent to the new address (self-service). The token is itself the\nproof that the new address is owned, so no authenticated session is required, following the same principle as password reset.")
model AccountEmailChangeConfirm {
  @minLength(1)
  token: string;
}


@doc("An end user's export of their own personal data (GDPR Article 15, the right of access). At this stage it is\ngenerated synchronously and gathers the profile and the connected applications (consents) available from the current API.")
model AccountDataExport {
  exported_at: utcDateTime;
  profile: AccountProfileResponse;
  consents: AccountConsentResponse[];
}


@doc("The membership relation between a User and a Group. It is unique on `group_id` and `user_id`.")
model GroupMember {
  group_id: string;
  user_id: string;
  source: GroupMembershipSource = GroupMembershipSource.manual;
  @doc("For `dynamic_rule`, the `DynamicGroupRule.version` that was evaluated.")
  rule_version?: int64;
  created_at: utcDateTime;
}


@doc("The restricted CEL rule paired one-to-one with a dynamic Group. The expression returns a boolean, and an unknown attribute, a function outside the permitted set, and an oversized expression can none of them be saved.")
model DynamicGroupRule {
  group_id: string;
  tenant_id: string;
  @minLength(1)
  @maxLength(4096)
  expression: string;
  enabled: boolean = false;
  version: int64;
  referenced_attributes: string[];
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("The auditable result of a full re-evaluation of a rule version and the membership it changed.")
model DynamicMembershipEvaluation {
  id: string;
  tenant_id: string;
  group_id: string;
  rule_version: int64;
  job_id?: string;
  status: DynamicMembershipEvaluationStatus;
  added_count: int64 = 0;
  removed_count: int64 = 0;
  unchanged_count: int64 = 0;
  error_count: int64 = 0;
  created_at: utcDateTime;
  completed_at?: utcDateTime;
}


@doc("The input creating or updating a CEL rule.")
model DynamicGroupRuleRequest {
  @minLength(1)
  @maxLength(4096)
  expression: string;
}


@doc("The input for a read-only preview of an unsaved CEL against at most 100 users.")
model DynamicGroupPreviewRequest {
  expression: string;
  user_ids: string[];
}


@doc("Per User, whether the rule matched and how that differs from the current membership. It carries no attribute value.")
model DynamicGroupPreviewResult {
  user_id: string;
  matched: boolean;
  change: string;
  error_code?: string;
}


@doc("The JSON request in which an administrator creates a group.")
model GroupCreateRequest {
  @minLength(1)
  @maxLength(100)
  name: string;
  @maxLength(500)
  description?: string;
  @maxLength(254)
  email?: string;
  attributes?: Record<AttributeValue>;
  roles?: string[];
  membership_type?: GroupMembershipType;
  dynamic_rule?: DynamicGroupRuleRequest;
}


@doc("The JSON request in which an administrator updates a group's name, description, email, attributes, or roles. A field left out keeps its current value.")
model GroupUpdateRequest {
  @minLength(1)
  @maxLength(100)
  name?: string;
  @maxLength(500)
  description?: string;
  @maxLength(254)
  email?: string;
  attributes?: Record<AttributeValue>;
  roles?: string[];
}


@doc("How a group is represented. The list and the detail views of the administration UI return the same representation.")
model GroupSummaryResponse {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  email?: string;
  attributes?: Record<AttributeValue>;
  roles: string[];
  member_count: int64;
  membership_type: GroupMembershipType;
  dynamic_rule?: DynamicGroupRule;
  created_at: utcDateTime;
  updated_at?: utcDateTime;
}


@doc("The group list of the administration API.")
model GroupListResponse {
  groups: GroupSummaryResponse[];
}


@doc("How a group member is represented.")
model GroupMemberResponse {
  user_id: string;
  preferred_username: string;
  created_at: utcDateTime;
}


@doc("The optional request body naming `group_id` when a membership is created from a screen such as the user detail view.")
model GroupAddMemberRequest {
  group_id: string;
}


@doc("The groups a named User belongs to and the breakdown of their effective roles. The direct roles\n(`user.roles`) and the roles inherited from groups are returned separately alongside their union, so the\nadministration UI can make effective roles easy to follow. `effective_roles` is `direct_roles` united with\n`group_roles`; because it is derived across both User and Group, it is computed and exposed as a field of this\nresponse rather than as a constraint on the User model.")
model UserGroupsResponse {
  groups: GroupSummaryResponse[];
  @doc("`user.roles`.")
  direct_roles: string[];
  @doc("The union of the roles inherited from the groups they belong to.")
  group_roles: string[];
  @doc("`direct_roles` united with `group_roles`.")
  effective_roles: string[];
}


@doc("The non-human identity principal, scoped to a tenant. An Agent holds no credential of its own: it obtains tokens through the OAuth2Client an AgentCredentialBinding binds it to. `owner_user_id` names the User accountable for it, and issuance resolves that owner on every request rather than freezing the result into `status` (REQ-OAUTH2-046). `status` gates issuance operationally and `kind` gates it by supervision: anything but `active` issues no new token, and anything but `autonomous` issues one only through a path that records a human approval (REQ-OAUTH2-050). Both gates are fail-closed and independent — an `active` `supervised` Agent still needs the approval.")
model Agent {
  @minLength(1)
  @maxLength(64)
  id: string;
  tenant_id: string = "default";
  @minLength(1)
  @maxLength(100)
  name: string;
  @maxLength(500)
  description?: string;
  kind: AgentKind;
  @doc("The User accountable for the Agent. Offboarding them stops new issuance without changing `status`.")
  owner_user_id: string;
  status: AgentStatus = AgentStatus.Active;
  roles: string[] = #[];
  created_at: utcDateTime;
  updated_at: utcDateTime;
  @doc("When the reversible suspension took effect. Absent unless `status` is `disabled`.")
  disabled_at?: utcDateTime;
  @doc("When the emergency stop took effect. Absent unless `status` is `killed`; once set, it never clears.")
  killed_at?: utcDateTime;
}


@doc("The binding between an Agent and an OAuth2Client of the same tenant, unique on `agent_id` and `client_id`. An Agent obtains tokens using the credentials of the Client it is bound to. The `client_id` must name an OAuth2Client whose `tenant_id` matches the Agent's (AgentReferencesStayTenantLocal). One `client_id` cannot be bound to more than one Agent, which BindAgentCredential checks in its preconditions.")
model AgentCredentialBinding {
  agent_id: string;
  client_id: string;
  created_at: utcDateTime;
}


@doc("The JSON request in which an administrator registers an Agent. Leaving `owner_user_id` out makes the caller the owner. `kind` is required and has no default: it decides whether the Agent needs a human approval to obtain a token, and either default value would be wrong — `supervised` would register Agents that silently cannot obtain a token, `autonomous` would let an omission grant unsupervised issuance.")
model AgentRegisterRequest {
  @minLength(1)
  @maxLength(100)
  name: string;
  @maxLength(500)
  description?: string;
  @doc("Required. An omission is AgentKindRequiredError and an unknown value InvalidAgentKindError; neither falls back to a default.")
  kind: AgentKind;
  @doc("Left out, the caller's id becomes the owner.")
  owner_user_id?: string;
  roles?: string[];
}


@doc("The JSON request in which an administrator updates an Agent's name, description, kind, `owner_user_id`, or roles. A field left out keeps its current value. Changing `owner_user_id` transfers ownership.")
model AgentUpdateRequest {
  @minLength(1)
  @maxLength(100)
  name?: string;
  @maxLength(500)
  description?: string;
  @doc("Left out, the current kind stays. An unknown value is InvalidAgentKindError.")
  kind?: AgentKind;
  @doc("Changing it transfers ownership (AgentOwnerChanged).")
  owner_user_id?: string;
  roles?: string[];
}


@doc("The JSON request binding an OAuth2Client to an Agent.")
model AgentCredentialBindRequest {
  @minLength(1)
  client_id: string;
}


@doc("How an Agent is represented. The list and the detail views of the administration UI return the same representation.")
model AgentSummaryResponse {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  kind: AgentKind;
  owner_user_id: string;
  status: AgentStatus;
  roles: string[];
  @doc("The id of the OAuth2Client it is bound to.")
  client_ids: string[];
  created_at: utcDateTime;
  updated_at?: utcDateTime;
  disabled_at?: utcDateTime;
  killed_at?: utcDateTime;
}


@doc("The Agent list of the administration API.")
model AgentListResponse {
  agents: AgentSummaryResponse[];
}


@doc("An external IdP identity was linked to a local user (account linking).")
model FederationLinked {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  provider: string;
}


@doc("An external IdP identity was unlinked from a local user.")
model FederationUnlinked {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  provider: string;
}


@doc("An end user raised a change of their primary email and a confirmation link was sent to the new address\n(self-service). The email itself does not change until ConfirmEmailChange.\n`newEmailHash` is a SHA-256, so the plaintext never reaches the audit log.")
model EmailChangeRequested {
  occurredAt: utcDateTime;
  @doc("The tenant the target user belongs to.")
  tenantId: string;
  userId: string;
  @doc("The SHA-256 of the new email, as hex.")
  newEmailHash: string;
}


@doc("The one-time link sent to the new address was confirmed and the primary email was settled\n(self-service). `email_verified` becomes true, and a `verify_email` required action, if there is one, is\ncleared automatically.")
model EmailChanged {
  occurredAt: utcDateTime;
  @doc("The tenant the target user belongs to.")
  tenantId: string;
  userId: string;
}


@doc("An administrator created a user. Neither the initial password nor `password_hash` is in the audit event.")
model UserCreated {
  occurredAt: utcDateTime;
  @doc("The tenant the operation acted in. It matches both the actor and the target.")
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
}


@doc("An administrator updated a user's profile or roles.")
model UserUpdated {
  occurredAt: utcDateTime;
  @doc("The tenant the operation acted in. It matches both the actor and the target.")
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
  changedFields: string[];
}


@doc("An administrator disabled a user, refusing both new sign-ins and the use of existing sessions for them.")
model UserDisabled {
  occurredAt: utcDateTime;
  @doc("The tenant the operation acted in. It matches both the actor and the target.")
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
}


@doc("An administrator re-enabled a disabled user.")
model UserEnabled {
  occurredAt: utcDateTime;
  @doc("The tenant the operation acted in. It matches both the actor and the target.")
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
}


@doc("An administrator soft-deleted a user. The status moved to `pending_deletion` and the PII, Consents, RefreshTokens, and Sessions were preserved. `reason` is a free-text note the administrator may leave.")
model UserSoftDeleted {
  occurredAt: utcDateTime;
  @doc("The tenant the operation acted in. It matches both the actor and the target.")
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
  reason?: string;
}


@doc("An administrator restored a user in `pending_deletion`. The status returned to `active`, and they can sign in again with their PII and credentials intact.")
model UserRestored {
  occurredAt: utcDateTime;
  @doc("The tenant the operation acted in. It matches both the actor and the target.")
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
}


@doc("A user was purged, either by an administrator or by the automatic purge once the grace period had elapsed. The PII has been anonymized and the related aggregates cascade-deleted. `reason` is a free-text note the administrator may leave, and is `auto_purge` for an automatic one.")
model UserDeleted {
  occurredAt: utcDateTime;
  @doc("The tenant the operation acted in. It matches both the actor and the target.")
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
  reason?: string;
}


@doc("An administrator gave a user an action to be forced at their next sign-in (a RequiredAction). `action` is an enum value and is safe to keep in the audit trail in plaintext.")
model UserRequiredActionSet {
  occurredAt: utcDateTime;
  @doc("The tenant the operation acted in. It matches both the actor and the target.")
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
}


@doc("A forced action was cleared. Besides an administrator clearing it explicitly, it also fires when `update_password` is cleared automatically because the user changed their own password, and in that case the actor id is the target's own.")
model UserRequiredActionCleared {
  occurredAt: utcDateTime;
  @doc("The tenant the operation acted in. It matches both the actor and the target.")
  tenantId: string;
  @doc("Who cleared it. For an automatic clearing, the target themselves.")
  actorUserId: string;
  targetUserId: string;
}


@doc("An administrator created a group.")
model GroupCreated {
  occurredAt: utcDateTime;
  @doc("The tenant the group belongs to.")
  tenantId: string;
  actorUserId: string;
  groupId: string;
}


@doc("An administrator updated a group's name, description, or roles.")
model GroupUpdated {
  occurredAt: utcDateTime;
  @doc("The tenant the group belongs to.")
  tenantId: string;
  actorUserId: string;
  groupId: string;
  changedFields: string[];
}


@doc("An administrator deleted a group. The memberships were released by cascade, each with its own GroupMemberRemoved.")
model GroupDeleted {
  occurredAt: utcDateTime;
  @doc("The tenant the group belongs to.")
  tenantId: string;
  actorUserId: string;
  groupId: string;
}


@doc("An administrator made a User a member of a group. It is not emitted when they were already a member, so the operation is idempotent.")
model GroupMemberAdded {
  occurredAt: utcDateTime;
  @doc("The tenant the group belongs to.")
  tenantId: string;
  actorUserId: string;
  groupId: string;
  userId: string;
}


@doc("An administrator removed a User from a group, or the membership was released by the group's deletion. It is not emitted when they were not a member, so the operation is idempotent.")
model GroupMemberRemoved {
  occurredAt: utcDateTime;
  @doc("The tenant the group belongs to.")
  tenantId: string;
  actorUserId: string;
  groupId: string;
  userId: string;
}


@doc("An administrator updated the CEL rule of a dynamic group to a new version.")
model DynamicGroupRuleUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  groupId: string;
  ruleVersion: int64;
}


@doc("An administrator activated a validated rule and scheduled a full re-evaluation.")
model DynamicGroupRuleEnabled {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  groupId: string;
  ruleVersion: int64;
}


@doc("An administrator deactivated a rule and invalidated the old dynamic membership at once.")
model DynamicGroupRuleDisabled {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  groupId: string;
  ruleVersion: int64;
}


@doc("The full re-evaluation of a rule version finished. It carries no attribute value, only how many memberships changed.")
model DynamicMembershipEvaluated {
  occurredAt: utcDateTime;
  tenantId: string;
  groupId: string;
  ruleVersion: int64;
  addedCount: int64;
  removedCount: int64;
  unchangedCount: int64;
  errorCount: int64;
}


@doc("A worker started a full re-evaluation of dynamic membership.")
model DynamicMembershipEvaluationStarted {
  occurredAt: utcDateTime;
  tenantId: string;
  groupId: string;
  ruleVersion: int64;
}


@doc("A full re-evaluation of dynamic membership failed, and the rule version stayed fail-closed.")
model DynamicMembershipEvaluationFailed {
  occurredAt: utcDateTime;
  tenantId: string;
  groupId: string;
  ruleVersion: int64;
}


@doc("An administrator registered an Agent.")
model AgentRegistered {
  occurredAt: utcDateTime;
  @doc("The tenant the Agent belongs to.")
  tenantId: string;
  actorUserId: string;
  agentId: string;
}


@doc("An administrator updated an Agent's name, description, kind, or roles.")
model AgentUpdated {
  occurredAt: utcDateTime;
  @doc("The tenant the Agent belongs to.")
  tenantId: string;
  actorUserId: string;
  agentId: string;
  changedFields: string[];
}


@doc("An administrator suspended an Agent reversibly.")
model AgentDisabled {
  occurredAt: utcDateTime;
  @doc("The tenant the Agent belongs to.")
  tenantId: string;
  actorUserId: string;
  agentId: string;
}


@doc("An administrator returned a suspended Agent to service.")
model AgentEnabled {
  occurredAt: utcDateTime;
  @doc("The tenant the Agent belongs to.")
  tenantId: string;
  actorUserId: string;
  agentId: string;
}


@doc("An administrator killed an Agent. This is one-way and terminal; there is no way back.")
model AgentKilled {
  occurredAt: utcDateTime;
  @doc("The tenant the Agent belongs to.")
  tenantId: string;
  actorUserId: string;
  agentId: string;
}


@doc("An administrator deleted an Agent. Its bindings were released by cascade.")
model AgentDeleted {
  occurredAt: utcDateTime;
  @doc("The tenant the Agent belongs to.")
  tenantId: string;
  actorUserId: string;
  agentId: string;
}


@doc("An administrator transferred the owner (`owner_user_id`) of an Agent.")
model AgentOwnerChanged {
  occurredAt: utcDateTime;
  @doc("The tenant the Agent belongs to.")
  tenantId: string;
  actorUserId: string;
  agentId: string;
  previousOwnerUserId: string;
  newOwnerUserId: string;
}


@doc("An administrator bound an OAuth2Client to an Agent.")
model AgentCredentialBound {
  occurredAt: utcDateTime;
  @doc("The tenant the Agent belongs to.")
  tenantId: string;
  actorUserId: string;
  agentId: string;
  clientId: string;
}


@doc("An administrator unbound an OAuth2Client from an Agent.")
model AgentCredentialUnbound {
  occurredAt: utcDateTime;
  @doc("The tenant the Agent belongs to.")
  tenantId: string;
  actorUserId: string;
  agentId: string;
  clientId: string;
}


@doc("The User the request targets does not exist in this tenant. A User in another tenant is treated as nonexistent. RFC 9457 Problem Details with `type` = `urn:idmagic:error:user_not_found`.")
model UserNotFoundError is ProblemDetails;


@doc("The tenant already has as many export jobs running as it may have at once, so another one cannot start. Retrying after the running jobs finish succeeds, which is why the response is 429 rather than the 422 of QuotaExceededError. RFC 9457 Problem Details with `type` = `urn:idmagic:error:active_job_quota_exceeded`, its own code: the concurrency ceiling and the tenant resource quota are separate concepts, and a client that saw only `quota_exceeded` could not tell whether retrying would ever succeed.")
model ActiveJobQuotaExceededError is ProblemDetails;


@doc("The user import job named by the request does not exist in this tenant. RFC 9457 Problem Details with `type` = `urn:idmagic:error:user_import_not_found`.")
model UserImportNotFoundError is ProblemDetails;


@doc("The user import service is not configured, so an import can neither be started nor read. RFC 9457 Problem Details with `type` = `urn:idmagic:error:user_import_unavailable`.")
model UserImportUnavailableError is ProblemDetails;


@doc("The group import job named by the request does not exist in this tenant. RFC 9457 Problem Details with `type` = `urn:idmagic:error:group_import_not_found`.")
model GroupImportNotFoundError is ProblemDetails;


@doc("The group import service is not configured, so an import can neither be started nor read. RFC 9457 Problem Details with `type` = `urn:idmagic:error:group_import_unavailable`.")
model GroupImportUnavailableError is ProblemDetails;

}
