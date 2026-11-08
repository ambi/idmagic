namespace IdMagic.Contract {

@doc("同じ信頼設定に同じ主体のパターンの関連付けが存在するため、作成を拒否した。RFC 9457 の Problem Details で、type は urn:idmagic:error:agent_workload_binding_pattern_conflict である。")
model AgentWorkloadBindingPatternConflictError is ProblemDetails;

@doc("指定した信頼設定が存在しないか、別のテナントに属する。RFC 9457 の Problem Details で、type は urn:idmagic:error:workload_trust_bundle_not_found である。")
model WorkloadTrustBundleNotFoundError is ProblemDetails;

@doc("指定した関連付けが存在しないか、別のテナントに属する。RFC 9457 の Problem Details で、type は urn:idmagic:error:agent_workload_binding_not_found である。")
model AgentWorkloadBindingNotFoundError is ProblemDetails;

@doc("求めた WorkloadTrustBundle の `name` は、テナントですでに使われている。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:workload_trust_bundle_name_conflict` である。")
model WorkloadTrustBundleNameConflictError is ProblemDetails;


@doc("求めた WorkloadTrustBundle の `issuer` は、テナントですでに登録されている。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:workload_trust_bundle_issuer_conflict` である。")
model WorkloadTrustBundleIssuerConflictError is ProblemDetails;

@doc("求めた `spiffe` の WorkloadTrustBundle の `trust_domain` は、テナントのほかの `spiffe` の信頼設定ですでに使われている。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:workload_trust_bundle_trust_domain_conflict` である。")
model WorkloadTrustBundleTrustDomainConflictError is ProblemDetails;


@doc("`spiffe` の WorkloadTrustBundle の `trust_domain` が、SPIFFE のトラストドメインの構文を満たさない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:workload_trust_bundle_invalid_trust_domain` である。")
model WorkloadTrustBundleInvalidTrustDomainError is ProblemDetails;

@doc("IdManagement モジュールが公開する Agent の主体を、公開契約として写したもの。バインディングの検証に必要な属性だけを含む。")
model AgentRef {
  id: string;
  tenant_id: string;
  @doc("Active、Disabled、Killed のいずれか。Active 以外への結び付けはフェイルクローズで拒否する。")
  status: string;
}


@doc("外部のアテステーションのトークンの検証がフェイルクローズで失敗した。未登録の発行者、不正な署名、期限切れのトークン、オーディエンスの不一致、上限を超える TTL、SPIFFE ID の構文の違反、トラストドメインの不一致、パターンの不一致、曖昧な一致、または有効でない結び付き先の Agent のいずれかである。これはドメイン条件であり、HTTP ではトークン交換の InvalidGrantError としてクライアントに届く。")
model WorkloadAttestationRejectedError {

}


@doc("`jwks_uri` も `jwks` も与えられていない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:workload_trust_bundle_jwks_required` である。")
model WorkloadTrustBundleJwksRequiredError is ProblemDetails;


@doc("`name` が空である。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:workload_trust_bundle_name_required` である。")
model WorkloadTrustBundleNameRequiredError is ProblemDetails;


@doc("`issuer` が空である。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:workload_trust_bundle_issuer_required` である。")
model WorkloadTrustBundleIssuerRequiredError is ProblemDetails;


@doc("`accepted_audiences` が空である。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:workload_trust_bundle_audiences_required` である。")
model WorkloadTrustBundleAudiencesRequiredError is ProblemDetails;


@doc("`max_subject_token_ttl_seconds` が正の値でない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:workload_trust_bundle_invalid_ttl` である。")
model WorkloadTrustBundleInvalidTtlError is ProblemDetails;


@doc("指定した `agent_id` が、テナントの Agent へ解決しない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:agent_workload_binding_agent_not_found` である。")
model AgentWorkloadBindingAgentNotFoundError is ProblemDetails;


@doc("`subject_pattern` が空である。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:agent_workload_binding_pattern_required` である。")
model AgentWorkloadBindingPatternRequiredError is ProblemDetails;


@doc("WorkloadTrustBundle が掛ける検証の種別。`oidc` は OIDC 互換の JWT として検証する。`spiffe` は SPIFFE の JWT-SVID の規則（`sub` の SPIFFE ID、トラストドメインの一致、`kid` の必須、`typ`、`alg`）を足し、`iss` を任意にする。登録の後は変えられない。")
enum WorkloadTrustBundleProfile {
  oidc: "oidc",
  spiffe: "spiffe",
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


@doc("外部のアテステーションの発行者に対する、テナントに属する信頼の設定。`issuer` はテナントの中で一意で、`profile` が `oidc` なら必須である。`profile` が `spiffe` なら `trust_domain` が `spiffe` の信頼設定の間で一意である。`jwks_uri` とインラインの `jwks` の少なくとも一方が必要である。")
model WorkloadTrustBundle {
  @doc("生成した識別子（UUID v4）。変わらない。")
  @minLength(1)
  @maxLength(64)
  id: string;
  @doc("属するテナント。テナントをまたぐ参照は禁止する。")
  tenant_id: string = "default";
  @doc("管理用の表示名。テナントの中で一意である。")
  @minLength(1)
  @maxLength(100)
  name: string;
  @doc("検証の種別。登録の後は変わらない。")
  profile: WorkloadTrustBundleProfile = WorkloadTrustBundleProfile.oidc;
  @doc("`spiffe` では SPIFFE のトラストドメイン、`oidc` では運用者が発行者のまとまりに付けるラベル。")
  @minLength(1)
  @maxLength(255)
  trust_domain: string;
  @doc("外部のアテステーションのトークンの `iss` のクレームが一致しなければならない発行者の URL。テナントの中で一意である。`oidc` では必須、`spiffe` では任意で、省くと `iss` を確かめない。")
  issuer?: url;
  @doc("JWKS の取得元。https でなければならず、Web PKI で検証する。`spiffe` では SPIFFE Bundle Endpoint の `https_web` の方式の URL を指定でき、`https_spiffe` の方式には対応しない。")
  jwks_uri?: url;
  @doc("`jwks_uri` の代わりに使うインラインの JWKS。`spiffe` では SPIFFE バンドルを指定でき、`use` が `jwt-svid` の鍵だけを署名の検証に使う。")
  jwks?: JWKS;
  @doc("受け付けるオーディエンス。外部のアテステーションのトークンの `aud` のクレームは、少なくとも一つと一致しなければならない。")
  accepted_audiences: string[];
  @doc("外部のアテステーションのトークン自身の `exp - iat` の上限。超えるとフェイルクローズで拒否する。")
  max_subject_token_ttl_seconds: int64 = 3600;
  status: WorkloadTrustBundleStatus = WorkloadTrustBundleStatus.enabled;
  created_at: utcDateTime;
  updated_at?: utcDateTime;
  @doc("JWKS の取得に最後に成功した時刻。最後に正常だったキャッシュがどれだけ新しいかの判断に使う。")
  jwks_cached_at?: utcDateTime;
}


@doc("WorkloadTrustBundle の下の対応付けで、外部のサブジェクトに対する glob のパターンを同じテナントの Agent へ射影する。`agent_id` は、`tenant_id` が WorkloadTrustBundle のものと一致する既存の Agent を指さなければならない（WorkloadIdentityReferencesStayTenantLocal）。")
model AgentWorkloadBinding {
  @doc("生成した識別子（UUID v4）。変わらない。")
  @minLength(1)
  @maxLength(64)
  id: string;
  @doc("属するテナント。`WorkloadTrustBundle.tenant_id` と一致する。")
  tenant_id: string = "default";
  @doc("属する WorkloadTrustBundle の id。")
  trust_bundle_id: string;
  @doc("外部のアテステーションのトークンの `sub` のクレームと照合する glob のパターン。`spiffe://example.org/ns/*/sa/agent-*` や `system:serviceaccount:prod-agents:*` など。一つの `trust_bundle_id` の中での重複は禁止する。")
  @minLength(1)
  @maxLength(500)
  subject_pattern: string;
  @doc("射影する先の Agent。`tenant_id` が WorkloadTrustBundle のものと一致する既存の Agent でなければならない。")
  agent_id: string;
  status: AgentWorkloadBindingStatus = AgentWorkloadBindingStatus.enabled;
  created_at: utcDateTime;
  updated_at?: utcDateTime;
  disabled_at?: utcDateTime;
}


@doc("管理者が WorkloadTrustBundle を登録するために送る JSON のリクエスト。")
model WorkloadTrustBundleRegisterRequest {
  @minLength(1)
  @maxLength(100)
  name: string;
  @doc("検証の種別。省くと `oidc` になる。")
  profile?: WorkloadTrustBundleProfile;
  @minLength(1)
  @maxLength(255)
  trust_domain: string;
  @doc("`oidc` では必須、`spiffe` では任意である。")
  issuer?: url;
  jwks_uri?: url;
  jwks?: JWKS;
  accepted_audiences: string[];
  max_subject_token_ttl_seconds?: int64;
}


@doc("管理者が、WorkloadTrustBundle の `name`、`jwks_uri`、`jwks`、`accepted_audiences`、`max_subject_token_ttl_seconds` を更新するために送る JSON のリクエスト。省いたフィールドは現在の値を保つ。`profile`、`issuer`、`trust_domain` は変えられず、変えるには登録し直す。")
model WorkloadTrustBundleUpdateRequest {
  @minLength(1)
  @maxLength(100)
  name?: string;
  jwks_uri?: url;
  jwks?: JWKS;
  accepted_audiences?: string[];
  max_subject_token_ttl_seconds?: int64;
}


@doc("WorkloadTrustBundle の表現。管理 API の一覧と詳細のエンドポイントは同じ表現を返す。`jwks` からはシークレットを含まず、公開鍵だけを含む。")
model WorkloadTrustBundleResponse {
  id: string;
  tenant_id: string;
  name: string;
  profile: WorkloadTrustBundleProfile;
  trust_domain: string;
  @doc("`oidc` では常に返す。`spiffe` では登録した場合だけ返す。")
  issuer?: url;
  jwks_uri?: url;
  @doc("インラインの `jwks` を設定しているかどうか。`jwks` そのものは返さない。")
  has_inline_jwks: boolean;
  accepted_audiences: string[];
  max_subject_token_ttl_seconds: int64;
  status: WorkloadTrustBundleStatus;
  created_at: utcDateTime;
  updated_at?: utcDateTime;
  jwks_cached_at?: utcDateTime;
}


@doc("管理 API の WorkloadTrustBundle の一覧。")
model WorkloadTrustBundleListResponse {
  trust_bundles: WorkloadTrustBundleResponse[];
}


@doc("RefreshWorkloadTrustBundleJWKS のレスポンスで、JWKS を実際に取得できたかどうかを示す。テナントが設定の誤りを問題になる前に見つけられるようにするための試験の操作である。")
model WorkloadTrustBundleRefreshResponse {
  reachable: boolean;
  @doc("署名の検証に使う鍵の数。`spiffe` の信頼設定で SPIFFE バンドルを読んだ場合は、`use` が `jwt-svid` の鍵の数である。")
  key_count?: int64;
  jwks_cached_at?: utcDateTime;
}


@doc("管理者が AgentWorkloadBinding を作成するために送る JSON のリクエスト。")
model AgentWorkloadBindingCreateRequest {
  @minLength(1)
  @maxLength(500)
  subject_pattern: string;
  @minLength(1)
  agent_id: string;
}

@doc("関連付けのパターンと Agent の部分更新。省いた項目は保つ。")
model AgentWorkloadBindingUpdateRequest {
  @minLength(1)
  @maxLength(500)
  subject_pattern?: string;
  @minLength(1)
  agent_id?: string;
}

@doc("保存せずに主体の一致と入力中のパターンの重なり候補を試す。")
model AgentWorkloadBindingPreviewRequest {
  subject?: string;
  @minLength(1)
  @maxLength(500)
  subject_pattern?: string;
  exclude_binding_id?: string;
}

@doc("有効な関連付けの一致と近似による重なり候補。候補が空でも重なりがない保証はしない。")
model AgentWorkloadBindingPreviewResponse {
  matches: AgentWorkloadBindingResponse[];
  possible_overlaps: AgentWorkloadBindingResponse[];
  candidate_matches?: boolean;
}

@doc("発行者の URL から Discovery 文書を読み込む入力。")
model WorkloadIssuerMetadataRequest {
  issuer: url;
}

@doc("検証済みの入力候補。信頼設定は保存しない。")
model WorkloadIssuerMetadataResponse {
  issuer: url;
  jwks_uri: url;
}

@doc("発行者情報の取得先、応答または issuer の検証に失敗した。type は urn:idmagic:error:workload_issuer_metadata_invalid である。")
model WorkloadIssuerMetadataInvalidError is ProblemDetails;

@doc("管理者が関連付けのパターンまたは Agent を更新した。")
model AgentWorkloadBindingUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  trustBundleId: string;
  bindingId: string;
  agentId: string;
}


@doc("AgentWorkloadBinding の表現。管理 API の一覧と詳細のエンドポイントは同じ表現を返す。")
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


@doc("管理 API の AgentWorkloadBinding の一覧。")
model AgentWorkloadBindingListResponse {
  bindings: AgentWorkloadBindingResponse[];
}


@doc("OAuth 2.0 のトークン交換のグラントから渡される、生の外部のアテステーションのトークンとその検証のコンテキスト。VerifyWorkloadAttestation の入力である。")
model WorkloadAttestation {
  @doc("生の文字列としての外部のアテステーションのトークン。たとえば JWT-SVID。")
  subject_token: string;
  @doc("トークン交換の `resource` パラメーター（RFC 8707）。監査イベントに残す。")
  requested_audience: string;
}


@doc("外部のアテステーションが検証を通った後に射影された IdMagic の主体。OAuth 2.0 のトークン交換のユースケースはこれをサブジェクトとして扱い、`client_id` が指す結び付き先の OAuth2Client を自身のリポジトリで解決して、スコープとグラントの種類を絞り込む上限を得る。WorkloadIdentity 自体は OAuth2Client を知らない。")
model WorkloadIdentityGrant {
  agent_id: string;
  @doc("結び付き先の Agent が保持する AgentCredentialBinding の `client_id`。発行するトークンの `sub` になる。")
  client_id: string;
  trust_bundle_id: string;
  binding_id: string;
}


@doc("管理者が新しい WorkloadTrustBundle を登録した。")
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


@doc("管理者が WorkloadTrustBundle を無効にした。以降、その下のどのバインディングも交換に使えない。")
model WorkloadTrustBundleDisabled {
  occurredAt: utcDateTime;
  tenantId: string;
  trustBundleId: string;
}


@doc("管理者が、無効にした WorkloadTrustBundle を再び有効にした。")
model WorkloadTrustBundleEnabled {
  occurredAt: utcDateTime;
  tenantId: string;
  trustBundleId: string;
}


@doc("管理者が WorkloadTrustBundle を削除した。その下の AgentWorkloadBinding も一緒に削除した。")
model WorkloadTrustBundleDeleted {
  occurredAt: utcDateTime;
  tenantId: string;
  trustBundleId: string;
}


@doc("管理者が RefreshWorkloadTrustBundleJWKS を呼び出し、JWKS の取得を再び試みた。")
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


@doc("管理者が AgentWorkloadBinding を無効にした。")
model AgentWorkloadBindingDisabled {
  occurredAt: utcDateTime;
  tenantId: string;
  bindingId: string;
}


@doc("管理者が、無効にした AgentWorkloadBinding を再び有効にした。")
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


@doc("外部のアテステーションが検証を通り、結び付き先の Agent の資格情報として IdMagic のトークンを発行した。")
model WorkloadTokenExchanged {
  occurredAt: utcDateTime;
  tenantId: string;
  trustBundleId: string;
  bindingId: string;
  agentId: string;
  audience: string;
}


@doc("外部のアテステーションのトークンの検証をフェイルクローズで拒否した。")
model WorkloadAttestationRejected {
  occurredAt: utcDateTime;
  tenantId: string;
  @doc("unregistered_issuer、trust_bundle_disabled、jwks_unavailable、invalid_signature、expired、audience_mismatch、ttl_exceeded、invalid_spiffe_id、trust_domain_mismatch、no_binding_match、ambiguous_match、agent_not_active、agent_unbound のいずれか。")
  reason: string;
  @doc("発行者が登録済みだったときだけ設定する。")
  trustBundleId?: string;
}

}
