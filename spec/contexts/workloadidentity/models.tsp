namespace IdMagic.Contract {

@doc("IdManagement が publish する Agent principal を、束縛検証に必要な属性だけで表す published language stub。")
model AgentRef {
  id: string;
  tenant_id: string;
  @doc("Active / Disabled / Killed。Active 以外への束縛は fail-closed で拒否する。")
  status: string;
}


@doc("外部 attestation token の検証が fail-closed で失敗した (未登録 issuer・署名不正・期限切れ・audience 不一致・TTL 超過・pattern 不一致・ambiguous match・束縛先 Agent が Active でない)。")
model WorkloadAttestationRejectedError {

}


@doc("jwks_uri と jwks のいずれも指定されていない。")
model WorkloadTrustBundleJwksRequiredError {

}


@doc("name が空である。")
model WorkloadTrustBundleNameRequiredError {

}


@doc("issuer が空である。")
model WorkloadTrustBundleIssuerRequiredError {

}


@doc("accepted_audiences が空である。")
model WorkloadTrustBundleAudiencesRequiredError {

}


@doc("max_subject_token_ttl_seconds が正の値でない。")
model WorkloadTrustBundleInvalidTtlError {

}


@doc("指定した agent_id がテナント内の Agent に解決できない。")
model AgentWorkloadBindingAgentNotFoundError {

}


@doc("subject_pattern が空である。")
model AgentWorkloadBindingPatternRequiredError {

}


@doc("WorkloadTrustBundleLifecycle の状態。")
enum WorkloadTrustBundleStatus {
  enabled: "enabled",
  disabled: "disabled",
}


@doc("AgentWorkloadBindingLifecycle の状態。")
enum AgentWorkloadBindingStatus {
  enabled: "enabled",
  disabled: "disabled",
}


@doc("テナントスコープの外部 attestation 発行者信頼設定。issuer は tenant 内で一意。jwks_uri と jwks (inline) は少なくとも一方が必須。")
model WorkloadTrustBundle {
  @doc("生成識別子。UUID v4。不変。")
  @minLength(1)
  @maxLength(64)
  id: string;
  @doc("所属テナント ID。cross-tenant 参照は禁止。")
  tenant_id: string = "default";
  @doc("テナント内で一意な管理用表示名。")
  @minLength(1)
  @maxLength(100)
  name: string;
  @doc("SPIFFE trust domain、または運用者が定める発行者グループのラベル。")
  @minLength(1)
  @maxLength(255)
  trust_domain: string;
  @doc("外部 attestation token の iss claim と一致しなければならない発行者 URL。テナント内で一意。")
  issuer: url;
  @doc("JWKS 取得元。https 必須。")
  jwks_uri?: url;
  @doc("inline JWKS。jwks_uri の代替。")
  jwks?: unknown;
  @doc("外部 attestation token の aud claim が最低 1 つ一致しなければならない受理 audience 集合。")
  accepted_audiences: string[];
  @doc("外部 attestation token 自身の (exp - iat) に許容する上限。超過は fail-closed で拒否する。")
  max_subject_token_ttl_seconds: int64 = 3600;
  status: WorkloadTrustBundleStatus = WorkloadTrustBundleStatus.enabled;
  created_at: utcDateTime;
  updated_at?: utcDateTime;
  @doc("JWKS の最終取得成功時刻。last-known-good キャッシュの鮮度判定に使う。")
  jwks_cached_at?: utcDateTime;
}


@doc("WorkloadTrustBundle 配下で外部 subject の glob pattern を同一テナントの Agent へ写す mapping。agent_id は WorkloadTrustBundle と同一 tenant_id の既存 Agent でなければならない (WorkloadIdentityReferencesStayTenantLocal)。")
model AgentWorkloadBinding {
  @doc("生成識別子。UUID v4。不変。")
  @minLength(1)
  @maxLength(64)
  id: string;
  @doc("所属テナント ID。WorkloadTrustBundle.tenant_id と一致する。")
  tenant_id: string = "default";
  @doc("属する WorkloadTrustBundle の id。")
  trust_bundle_id: string;
  @doc("外部 attestation token の sub claim に対する glob pattern (例: spiffe://example.org/ns/*/sa/agent-*、system:serviceaccount:prod-agents:*)。同一 trust_bundle_id 内で重複禁止。")
  @minLength(1)
  @maxLength(500)
  subject_pattern: string;
  @doc("写像先の Agent id。WorkloadTrustBundle と同一 tenant_id の既存 Agent でなければならない。")
  agent_id: string;
  status: AgentWorkloadBindingStatus = AgentWorkloadBindingStatus.enabled;
  created_at: utcDateTime;
  updated_at?: utcDateTime;
  disabled_at?: utcDateTime;
}


@doc("管理者が WorkloadTrustBundle を登録する JSON リクエスト。")
model WorkloadTrustBundleRegisterRequest {
  @minLength(1)
  @maxLength(100)
  name: string;
  @minLength(1)
  @maxLength(255)
  trust_domain: string;
  issuer: url;
  jwks_uri?: url;
  jwks?: unknown;
  accepted_audiences: string[];
  max_subject_token_ttl_seconds?: int64;
}


@doc("管理者が WorkloadTrustBundle の name / jwks_uri / jwks / accepted_audiences / max_subject_token_ttl_seconds を更新する JSON リクエスト。省略フィールドは現状維持。issuer / trust_domain は不変 (変更したい場合は登録し直す)。")
model WorkloadTrustBundleUpdateRequest {
  @minLength(1)
  @maxLength(100)
  name?: string;
  jwks_uri?: url;
  jwks?: unknown;
  accepted_audiences?: string[];
  max_subject_token_ttl_seconds?: int64;
}


@doc("WorkloadTrustBundle 表現。管理 API の一覧・詳細で同じ表現を返す。jwks の秘密情報は含まない (公開鍵材料のみ)。")
model WorkloadTrustBundleResponse {
  id: string;
  tenant_id: string;
  name: string;
  trust_domain: string;
  issuer: url;
  jwks_uri?: url;
  @doc("inline jwks が設定されているかどうか (jwks 本体は返さない)。")
  has_inline_jwks: boolean;
  accepted_audiences: string[];
  max_subject_token_ttl_seconds: int64;
  status: WorkloadTrustBundleStatus;
  created_at: utcDateTime;
  updated_at?: utcDateTime;
  jwks_cached_at?: utcDateTime;
}


@doc("管理 API の WorkloadTrustBundle 一覧。")
model WorkloadTrustBundleListResponse {
  trust_bundles: WorkloadTrustBundleResponse[];
}


@doc("RefreshWorkloadTrustBundleJWKS の応答。実際に JWKS を取得できたかどうかを返す (テナントが設定ミスを事前に発見するための test 操作)。")
model WorkloadTrustBundleRefreshResponse {
  reachable: boolean;
  key_count?: int64;
  jwks_cached_at?: utcDateTime;
}


@doc("管理者が AgentWorkloadBinding を作成する JSON リクエスト。")
model AgentWorkloadBindingCreateRequest {
  @minLength(1)
  @maxLength(500)
  subject_pattern: string;
  @minLength(1)
  agent_id: string;
}


@doc("AgentWorkloadBinding 表現。管理 API の一覧・詳細で同じ表現を返す。")
model AgentWorkloadBindingResponse {
  id: string;
  tenant_id: string;
  trust_bundle_id: string;
  subject_pattern: string;
  agent_id: string;
  status: AgentWorkloadBindingStatus;
  created_at: utcDateTime;
  updated_at?: utcDateTime;
  disabled_at?: utcDateTime;
}


@doc("管理 API の AgentWorkloadBinding 一覧。")
model AgentWorkloadBindingListResponse {
  bindings: AgentWorkloadBindingResponse[];
}


@doc("OAuth2 の token-exchange grant から渡す、外部 attestation token の生値と検証コンテキスト。VerifyWorkloadAttestation の input。")
model WorkloadAttestation {
  @doc("外部 attestation token (JWT-SVID 等) の生の文字列。")
  subject_token: string;
  @doc("token-exchange の resource パラメータ (RFC 8707)。監査イベントに残す。")
  requested_audience: string;
}


@doc("検証を通過した外部 attestation を写した idmagic principal。OAuth2 の token-exchange usecase はこれを subject として扱い、client_id が指す束縛済み OAuth2Client を自身の repository で解決して scope / grant_types のダウンスコープ上限を得る (WorkloadIdentity は OAuth2Client の詳細を持たない)。")
model WorkloadIdentityGrant {
  agent_id: string;
  @doc("束縛先 Agent が保持する AgentCredentialBinding の client_id。発行トークンの sub になる。")
  client_id: string;
  trust_bundle_id: string;
  binding_id: string;
}


@doc("管理者が WorkloadTrustBundle を新規登録した。")
model WorkloadTrustBundleConfigured {
  occurredAt: utcDateTime;
  tenantId: string;
  trustBundleId: string;
  issuer: string;
}


@doc("管理者が WorkloadTrustBundle のメタデータを更新した。")
model WorkloadTrustBundleUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  trustBundleId: string;
}


@doc("管理者が WorkloadTrustBundle を無効化した。以後この bundle 配下の binding は交換に使えない。")
model WorkloadTrustBundleDisabled {
  occurredAt: utcDateTime;
  tenantId: string;
  trustBundleId: string;
}


@doc("管理者が無効化済みの WorkloadTrustBundle を再有効化した。")
model WorkloadTrustBundleEnabled {
  occurredAt: utcDateTime;
  tenantId: string;
  trustBundleId: string;
}


@doc("管理者が WorkloadTrustBundle を削除した。配下の AgentWorkloadBinding は cascade で削除する。")
model WorkloadTrustBundleDeleted {
  occurredAt: utcDateTime;
  tenantId: string;
  trustBundleId: string;
}


@doc("管理者が RefreshWorkloadTrustBundleJWKS を呼び、JWKS の再取得を試みた。")
model WorkloadTrustBundleJWKSRefreshed {
  occurredAt: utcDateTime;
  tenantId: string;
  trustBundleId: string;
  reachable: boolean;
}


@doc("管理者が AgentWorkloadBinding を作成した。")
model AgentWorkloadBindingCreated {
  occurredAt: utcDateTime;
  tenantId: string;
  trustBundleId: string;
  bindingId: string;
  agentId: string;
}


@doc("管理者が AgentWorkloadBinding を無効化した。")
model AgentWorkloadBindingDisabled {
  occurredAt: utcDateTime;
  tenantId: string;
  bindingId: string;
}


@doc("管理者が無効化済みの AgentWorkloadBinding を再有効化した。")
model AgentWorkloadBindingEnabled {
  occurredAt: utcDateTime;
  tenantId: string;
  bindingId: string;
}


@doc("管理者が AgentWorkloadBinding を削除した。")
model AgentWorkloadBindingDeleted {
  occurredAt: utcDateTime;
  tenantId: string;
  bindingId: string;
}


@doc("外部 attestation の検証を通過し、束縛先 Agent の資格情報として idmagic token を発行した。")
model WorkloadTokenExchanged {
  occurredAt: utcDateTime;
  tenantId: string;
  trustBundleId: string;
  bindingId: string;
  agentId: string;
  audience: string;
}


@doc("外部 attestation token の検証が fail-closed で拒否された。")
model WorkloadAttestationRejected {
  occurredAt: utcDateTime;
  tenantId: string;
  @doc("unregistered_issuer / trust_bundle_disabled / jwks_unavailable / invalid_signature / expired / audience_mismatch / ttl_exceeded / no_binding_match / ambiguous_match / agent_not_active / agent_unbound のいずれか。")
  reason: string;
  @doc("issuer が登録済みだった場合のみ設定する。")
  trustBundleId?: string;
}

}
