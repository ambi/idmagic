namespace IdMagic.Contract {

@doc("The query parameters of ListAdminAuditEvents. `tenant_id` comes from the context, so it is not a field here.\n`type` is a `DomainEventEnvelope.type`, the type name the emitting context gave the event.")
model AuditEventQuery {
  @doc("Exact match on `type`, the low-level machine-facing filter. For example, `AccessTokenIssued`.")
  type?: string;
  @doc("Narrows by event category: authentication, success, fail, aggregated, user, group, client, consent, token, tenant, or key.")
  category?: string;
  @doc("Exact match on the actor's user sub. It names the same thing as `actor.id` in `filter` (see AuditActorVsTarget).\nTo narrow by the user an administrative operation acted on, use `target.id` in `filter` instead -- a different concept\nfrom this field. The UI labels it \"user id (actor)\".")
  user_id?: string;
  @doc("Searches by the actor's login name. Events where the real account is always settled -- UserAuthenticated, the OAuth2\nflow events, and the like -- carry no username in their payload, so the search resolves the name to a user id through\nUserRepo and then narrows exactly as `user_id` does. When no such user exists the result is empty; it never falls back\nto ignoring the filter and returning everything. It cannot be used to search authentication failures, where an account\nname that never existed still has to be traceable -- use `actor.username` in `filter` for that.")
  username?: string;
  @doc("`occurred_at` at or after this instant.")
  after?: utcDateTime;
  @doc("`occurred_at` at or before this instant.")
  before?: utcDateTime;
  @doc("Pass through the value embedded in the URL of the response's Link header. It is a stateless, versioned, opaque token\nholding the direction, the anchor, the page number it moves to, and the keyset in a compact binary payload, signed with\nthe tenant, the sort, and the filter as associated data. A new v3 token does not expire with time, and legacy v0 and v2\ntokens are still accepted under their existing rules.")
  cursor?: string;
  @doc("The page size ceiling (default 100, maximum 1000).")
  limit?: int64;
  @doc("Free text matched as a substring against the text columns stored raw (`type`, `outcome`, and the like). PII columns are excluded.")
  q?: string;
  @doc("A conjunction (AND) of structured search terms over the field/operator allow list of the registry. `actor.username` is a\nplaintext match reserved for authentication failures, where an account name that never existed still has to be traceable;\nit is not hashed. To search events where the real account is settled, use the `username` query parameter instead -- a\ndifferent path from this field. `client.ip` is matched in plaintext too, neither truncated nor hashed. `event.type` draws\nfrom the `DomainEventEnvelope.type` names the contexts emit, and `outcome` from the enum\nthe Go `eventOutcome` classification defines (success or failure); both are picked from a list rather than typed freely\n(see AuditActorVsTarget and AuditSearchRegistry). `actor.id` and `actor.username` name the actor while `target.id` names\nthe user an administrative operation acted on -- different meanings. `workflow.id`, `workflow_run.id`, and\n`workflow_step.id` are registry attributes matched with `eq` against the workflowId, runId, and stepIndex of the\nIdGovernance LifecycleWorkflow*, WorkflowRun*, and WorkflowStepFailed events. `actor.type` (`user` or `agent`),\n`agent.id`, `delegation.actor`, `delegation.depth`, and `delegation.mode` are the delegation axes. `delegation.actor`\nnames every principal of the event's `act` chain and matches when any of them matches, so one filter finds the whole chain\nfrom any participant. `delegation.mode` is read from the payload, where the derivation REQ-OAUTH2-049 defines put it;\naudit does not derive it a second time. Events recorded before these axes existed carry none of them and so match no value\non them. Attribute values and email bodies are not in the payload, so they are not searchable either.")
  filter?: AuditEventFilterExpression[];
}


@doc("The comparison operators an audit search filter expression may use. The registry declares which operators each attribute permits. `eq` is an exact match, `in` matches any member of a value set, `contains` is a substring match (used by `q`), and `time_range` bounds `occurred_at`.")
enum AuditEventFilterOperator {
  eq: "eq",
  in: "in",
  contains: "contains",
  time_range: "time_range",
}


@doc("The transform applied to a plaintext value before a search attribute is stored in the sidecar or searched. Today only\n`none` exists, meaning the value is stored raw.")
enum AuditEventSearchTransform {
  none: "none",
}


@doc("A search axis permitted in audit event search, that is, one registry entry. It declares the field/operator allow list and\nso fixes a closed search grammar in place of arbitrary SQL or JSONPath. The Go AuditSearchRegistry is the twin of this\nvalue object. Every attribute, `actor.username` and `client.ip` included, is stored and searched in the sidecar as\nplaintext. `actor.id` and `actor.username` name the actor -- who did it -- while `target.id` names the user an\nadministrative operation acted on -- who it was done to; the same word \"user\" means different things (AuditActorVsTarget).\n`event.type` and `outcome` are picked from a list rather than typed freely: `event.type` from the `DomainEventEnvelope.type` names the contexts emit, and `outcome` from the two values Go's `eventOutcome()` returns, success and failure, with no\nvalue at all for events it does not classify. `actor.type`, `agent.id`, `delegation.actor`, `delegation.depth`, and\n`delegation.mode` are the delegation axes (REQ-AUDIT-005, REQ-AUDIT-006): they separate what an Agent did for a user from\nwhat the user did (AuditDelegationChain).")
model AuditEventSearchAttribute {
  @doc("The canonical dotted name of the search axis, such as `actor.username`, `client.ip`, or `event.type`.")
  field: string;
  @doc("Whether the plaintext may be stored in the sidecar as it is. True for every attribute today.")
  raw_storable: boolean;
  @doc("The transform applied to the plaintext before storing or searching. Only `none` exists today.")
  transform: AuditEventSearchTransform;
  @doc("The allow list of operators this attribute permits.")
  allowed_operators: AuditEventFilterOperator[];
  @doc("Whether one event may carry several values on this axis. `delegation.actor` is the only multi-valued axis today: one\nevent names every principal of its `act` chain, and a filter matches when any of them matches. Every other axis holds at\nmost one value per event.")
  multi_valued: boolean;
  @doc("Whether to offer it as a UI preset in the administrative search builder.")
  ui_visible: boolean;
}


@doc("One term of `AuditEventQuery.filter`: a field and an operator from the registry allow list together with the values, as one\nelement of the conjunction (AND). Arbitrary SQL is not permitted.")
model AuditEventFilterExpression {
  @doc("The `field` of an AuditEventSearchAttribute in the registry.")
  field: string;
  @doc("An operator listed in the attribute's `allowed_operators`.")
  operator: AuditEventFilterOperator;
  @doc("The values, as many as the operator takes: one for `eq`, N for `in`, two for `time_range`. For a PII attribute the server transforms the plaintext input before matching.")
  values: string[];
}


@doc("A DomainEvent as it is exposed to administrators. The payload is structured JSON.\nPII fields such as email and IP are assumed to have been redacted by the EventSink before this point.\nThe store is append-only and keeps events for seven years (GDPR Article 30).")
model AdminAuditEventResponse {
  @doc("The event id the store assigned (UUIDv4).")
  id: string;
  @doc("The tenant the event belongs to.")
  tenant_id: string;
  @doc("The wire name of `DomainEvent.type`.")
  type: string;
  occurred_at: utcDateTime;
  @doc("The event body, shaped as DomainEventPayload: the published fields are contract, and the rest belongs to the emitting context.")
  payload: Record<unknown>;
}


@doc("The result of an audit event search. Pagination navigation lives in the Link response header, not in the body.")
model AdminAuditEventListResponse {
  events: AdminAuditEventResponse[];
}


@doc("The choices that turn the enumerated search axes into pick lists. The UI does not hard-code them; it reads them from the\nsame single source of truth in Go (the category-to-type map, the eventOutcome classification, and the DelegationMode\nderivation REQ-OAUTH2-049 defines).")
model AuditEventSearchOptionsResponse {
  @doc("The `event.type` values that can be picked, that is, `DomainEventEnvelope.type` names. It is the union of the\ncategorized types, the same population the event category picker in the administration UI draws from.")
  event_types: string[];
  @doc("The `outcome` values that can be picked. Today there are two, success and failure, and only authentication events are classified.")
  outcomes: string[];
  @doc("The `actor.type` values that can be picked: `user` and `agent`.")
  actor_types: string[];
  @doc("The `delegation.mode` values that can be picked. They are the members of DelegationMode, so the audit filter and an\nintrospection response name the same modes.")
  delegation_modes: string[];
}

}
