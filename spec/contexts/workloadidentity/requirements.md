# WorkloadIdentity Requirements

> This Markdown file is the normative, language-independent home for product requirements. Models and API contracts live in the adjacent TypeSpec source.

## Requirements

### REQ-WORKLOADIDENTITY-001: 登録済みtrustbundle経由でワークロードトークンをAgent資格情報に交換できる
- Actor: System
- Given: テナント "tenant-a" に issuer "https://issuer.example" の WorkloadTrustBundle "prod-cluster" が Enabled で登録済みである
- Given: "prod-cluster" 配下に subject pattern "spiffe://example.org/ns/prod/sa/*" を Agent "checkout-bot" へ写す AgentWorkloadBinding が Enabled で存在する
- Given: Agent "checkout-bot" は Active で、AgentCredentialBinding 経由で OAuth2Client に束縛済みである
- Then: sub "spiffe://example.org/ns/prod/sa/worker-1" を持つ有効な JWT-SVID を subject_token として token-exchange を呼ぶ
- Then: VerifyWorkloadAttestation が Agent \"checkout-bot\" の束縛先 client_id を sub とする WorkloadIdentityGrant を返す
- Then: WorkloadTokenExchanged が発火し、束縛先 Agent の資格情報として短命な idmagic access token が発行される

### REQ-WORKLOADIDENTITY-002: 未登録issuerは拒否される
- Actor: System
- Given: テナント "tenant-a" に issuer "https://issuer.example" の WorkloadTrustBundle は登録されていない
- Then: iss "https://unknown-issuer.example" の JWT-SVID を subject_token として token-exchange を呼ぶ
- Alternative (常に (未登録 issuer)): VerifyWorkloadAttestation が WorkloadAttestationRejectedError (reason=unregistered_issuer) で拒否する → WorkloadAttestationRejected が reason=unregistered_issuer で発火する

### REQ-WORKLOADIDENTITY-003: 署名が不正なattestationは拒否される
- Actor: System
- Given: テナント "tenant-a" に issuer "https://issuer.example" の WorkloadTrustBundle "prod-cluster" が Enabled で登録済みである
- Then: iss "https://issuer.example" を詐称するが登録済み JWKS で検証できない署名を持つ JWT を subject_token として token-exchange を呼ぶ
- Alternative (常に (spoofed / 署名不正)): VerifyWorkloadAttestation が WorkloadAttestationRejectedError (reason=invalid_signature) で拒否する

### REQ-WORKLOADIDENTITY-004: 期限切れのattestationは拒否される
- Actor: System
- Given: テナント "tenant-a" に issuer "https://issuer.example" の WorkloadTrustBundle "prod-cluster" が Enabled で登録済みである
- Then: exp が過去時刻の JWT-SVID を subject_token として token-exchange を呼ぶ
- Alternative (常に (期限切れ)): VerifyWorkloadAttestation が WorkloadAttestationRejectedError (reason=expired) で拒否する

### REQ-WORKLOADIDENTITY-005: 複数bindingに曖昧にマッチするsubjectは拒否される
- Actor: System
- Given: "prod-cluster" 配下に "spiffe://example.org/ns/prod/sa/*" → Agent "a" と "spiffe://example.org/ns/prod/sa/worker-*" → Agent "b" の 2 つの Enabled AgentWorkloadBinding が存在する
- Then: sub "spiffe://example.org/ns/prod/sa/worker-1" を持つ有効な JWT-SVID で token-exchange を呼ぶ
- Alternative (常に (複数 Enabled binding にマッチする、binding collision)): VerifyWorkloadAttestation が WorkloadAttestationRejectedError (reason=ambiguous_match) で拒否する

### REQ-WORKLOADIDENTITY-006: 束縛先AgentがKilled後は拒否される
- Actor: System
- Given: AgentWorkloadBinding の写像先 Agent "checkout-bot" が KillAgent により killed へ遷移済みである
- Then: sub がマッチする有効な JWT-SVID で token-exchange を呼ぶ
- Alternative (常に (kill 後)): VerifyWorkloadAttestation が WorkloadAttestationRejectedError (reason=agent_not_active) で拒否する

### REQ-WORKLOADIDENTITY-007: 他テナントのtrustbundleは利用できない
- Actor: System
- Given: テナント "tenant-b" に issuer "https://issuer.example" の WorkloadTrustBundle が登録済みである
- Given: テナント "tenant-a" には同じ issuer の WorkloadTrustBundle が存在しない
- Then: テナント "tenant-a" のコンテキストで iss "https://issuer.example" の JWT-SVID を subject_token として token-exchange を呼ぶ
- Alternative (常に (cross-tenant)): VerifyWorkloadAttestation が WorkloadAttestationRejectedError (reason=unregistered_issuer) で拒否する (tenant-b の登録は見えない)

### REQ-WORKLOADIDENTITY-008: 管理者がtrustbundleを登録・無効化・再有効化できる
- Actor: TenantAdministrator
- Given: 管理者としてテナントに認証済みである
- Then: issuer "https://issuer.example" と JWKS 取得元を指定して RegisterWorkloadTrustBundle を呼ぶ
- Then: WorkloadTrustBundleConfigured が発火し、WorkloadTrustBundle が enabled として作成される
- Then: 作成した WorkloadTrustBundle に対して DisableWorkloadTrustBundle を呼ぶ
- Then: WorkloadTrustBundleDisabled が発火し、以後この bundle 配下の binding は交換に使えなくなる
- Then: EnableWorkloadTrustBundle を呼ぶと WorkloadTrustBundleEnabled が発火し enabled に戻る
- Alternative (jwks_uri と jwks のいずれも指定しない): RegisterWorkloadTrustBundle が InvalidRequestError で拒否される
- Alternative (同一テナント内に同じ issuer の WorkloadTrustBundle が既に存在する): RegisterWorkloadTrustBundle が InvalidRequestError で拒否される

### REQ-WORKLOADIDENTITY-009: 管理者が他テナントのAgentへbindingを作成できない
- Actor: TenantAdministrator
- Given: テナント "tenant-a" に WorkloadTrustBundle "prod-cluster" が登録済みである
- Given: Agent "other-tenant-agent" はテナント "tenant-b" に属する
- Then: "prod-cluster" 配下に agent_id="other-tenant-agent" の CreateAgentWorkloadBinding を呼ぶ
- Alternative (常に (agent_id が別テナント)): CreateAgentWorkloadBinding が InvalidRequestError で拒否され binding は作成されない

### REQ-WORKLOADIDENTITY-010: ListWorkloadTrustBundles
管理者が所属テナントの WorkloadTrustBundle を一覧する。

### REQ-WORKLOADIDENTITY-011: GetWorkloadTrustBundle
管理者が単一 WorkloadTrustBundle を取得する。別テナントの bundle は未存在として扱う。

### REQ-WORKLOADIDENTITY-012: RegisterWorkloadTrustBundle
管理者が所属テナントに WorkloadTrustBundle を登録する。name はテナント内で一意、issuer もテナント内で一意でなければならない。
- Precondition: input.request.jwks_uri != null || input.request.jwks != null
- Precondition: trust_bundle_issuer_unique_in_tenant(context.tenant_id, input.request.issuer)

### REQ-WORKLOADIDENTITY-013: UpdateWorkloadTrustBundle
管理者が WorkloadTrustBundle の name / jwks_uri / jwks / accepted_audiences / max_subject_token_ttl_seconds を更新する。

### REQ-WORKLOADIDENTITY-014: DisableWorkloadTrustBundle
管理者が WorkloadTrustBundle を無効化する。以後配下の全 binding が交換に使えなくなる (fail-closed)。

### REQ-WORKLOADIDENTITY-015: EnableWorkloadTrustBundle
管理者が無効化済みの WorkloadTrustBundle を再有効化する。

### REQ-WORKLOADIDENTITY-016: DeleteWorkloadTrustBundle
管理者が WorkloadTrustBundle を削除する。配下の AgentWorkloadBinding は cascade で削除する。

### REQ-WORKLOADIDENTITY-017: RefreshWorkloadTrustBundleJWKS
管理者が WorkloadTrustBundle の JWKS 到達性を即時確認する (設定ミスの事前検知)。成功時は jwks_cached_at を更新する。

### REQ-WORKLOADIDENTITY-018: ListAgentWorkloadBindings
管理者が WorkloadTrustBundle 配下の AgentWorkloadBinding を一覧する。

### REQ-WORKLOADIDENTITY-019: CreateAgentWorkloadBinding
管理者が WorkloadTrustBundle 配下に AgentWorkloadBinding を作成する。agent_id は同一テナントの既存 Agent でなければならない (WorkloadIdentityReferencesStayTenantLocal)。同一 trust_bundle_id 内で subject_pattern の完全重複は拒否する。
- Precondition: input.request.agent_id in tenantAgentIds(context.tenant_id)
- Precondition: !subject_pattern_duplicate_in_bundle(input.trust_bundle_id, input.request.subject_pattern)

### REQ-WORKLOADIDENTITY-020: DisableAgentWorkloadBinding
管理者が AgentWorkloadBinding を無効化する。以後この binding は交換に使えなくなる。

### REQ-WORKLOADIDENTITY-021: EnableAgentWorkloadBinding
管理者が無効化済みの AgentWorkloadBinding を再有効化する。

### REQ-WORKLOADIDENTITY-022: DeleteAgentWorkloadBinding
管理者が AgentWorkloadBinding を削除する。

### REQ-WORKLOADIDENTITY-023: VerifyWorkloadAttestation
OAuth2 の token-exchange grant (subject_token_type=JWT-SVID URN) から呼ぶ published interface。外部 attestation token を登録済み WorkloadTrustBundle で検証し (署名・iss・aud・exp・TTL 上限)、テナント内の AgentWorkloadBinding で subject を一意に Agent へ写し、束縛先 Agent が Active であることを確認する。いずれかに失敗すれば fail-closed で拒否する。idmagic 自身は SPIRE server/agent を実装しない (relying party 側に徹する)。
- Precondition: trust_bundle_registered_for_issuer(context.tenant_id, input.attestation.subject_token)
- Precondition: workload_attestation_signature_and_claims_valid(context.tenant_id, input.attestation.subject_token)
- Precondition: !ambiguous_binding_match(context.tenant_id, input.attestation.subject_token)
- Precondition: bound_agent_is_active(context.tenant_id, input.attestation.subject_token)
- Postcondition: output.grant == null || output.grant.client_id != ''

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| WorkloadTrustBundle | テナントが登録する外部 attestation 発行者の信頼設定。trust domain・issuer・JWKS 取得元 (または inline JWKS)・受理する audience・受理する外部 SVID の最大 TTL を束ねる。登録済み issuer のみを信頼する (trust-on-first-use を許さない)。 |  |
| AgentWorkloadBinding | WorkloadTrustBundle 配下で、外部 subject に対する glob pattern を同一テナントの既存 Agent へ写す 1 対多の mapping 行。pattern にマッチしない subject、または複数の Enabled binding に曖昧にマッチする subject は fail-closed で拒否する。 |  |
| JwtSvid | OIDC 互換の JWT を wire 形式とする外部 attestation token。Kubernetes projected ServiceAccount token・クラウドの instance identity (OIDC) token・SPIFFE の JWT-SVID を包含する第一級の subject_token 種別。X.509-SVID (mTLS) は将来の拡張。 | JWT-SVID |
| TrustDomain | WorkloadTrustBundle が束ねる論理グループ名。SPIFFE trust domain、または運用者が定める発行者グループのラベル。 |  |
| FailClosed | 未登録 issuer・署名不正・期限切れ・pattern 不一致・ambiguous match・束縛先 Agent が Active でない、のいずれかに該当する場合、常に交換を拒否する方針。判定漏れは「交換しない」側へ倒す。 |  |
| System | WorkloadIdentity の検証 usecase そのものを指す、人間の操作者を伴わない技術的な主体。 |  |

## State machines

### WorkloadTrustBundleLifecycle

登録時に enabled として作成される。無効化で disabled に遷移し、以後配下の binding は交換に使えなくなる (fail-closed)。再有効化で enabled に戻せる。削除は状態遷移ではなく行そのものを取り除く終端操作で、配下の binding を cascade で削除する。

Initial: `enabled`  
Terminal: none

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| enabled | WorkloadTrustBundleDisabled | "" | disabled |  |
| disabled | WorkloadTrustBundleEnabled | "" | enabled |  |

### AgentWorkloadBindingLifecycle

作成時に enabled として作成される。無効化で disabled に遷移し以後の交換に使えなくなる。再有効化で enabled に戻せる。削除は状態遷移ではなく行そのものを取り除く終端操作。

Initial: `enabled`  
Terminal: none

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| enabled | AgentWorkloadBindingDisabled | "" | disabled |  |
| disabled | AgentWorkloadBindingEnabled | "" | enabled |  |

## Authorization boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.
