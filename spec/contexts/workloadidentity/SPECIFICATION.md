---
context: workloadidentity
updated_at: 2026-08-11
---

# WorkloadIdentity Specification

## Overview

自律エージェントランタイム向けの workload identity federation を所有する。テナントが登録した外部 attestation 発行者の信頼設定 (WorkloadTrustBundle) と、外部 subject を既存 Agent へ写す mapping (AgentWorkloadBinding) を管理する。OIDC 互換 JWT (Kubernetes projected ServiceAccount token・クラウド instance identity token・SPIFFE JWT-SVID) を第一級の attestation 種別とし、idmagic 自身は SPIRE server/agent を同梱・運用しない (relying party 側に徹する)。検証を通過した外部 attestation を OAuth2 の token-exchange grant (RFC 8693) へ subject として渡し、専用の資格情報経路は新設しない。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| WorkloadTrustBundle | テナントが登録する外部 attestation 発行者の信頼設定。trust domain・issuer・JWKS 取得元 (または inline JWKS)・受理する audience・受理する外部 SVID の最大 TTL を束ねる。登録済み issuer のみを信頼する (trust-on-first-use を許さない)。 |  |
| AgentWorkloadBinding | WorkloadTrustBundle 配下で、外部 subject に対する glob pattern を同一テナントの既存 Agent へ写す 1 対多の mapping 行。pattern にマッチしない subject、または複数の Enabled binding に曖昧にマッチする subject は fail-closed で拒否する。 |  |
| JwtSvid | OIDC 互換の JWT を wire 形式とする外部 attestation token。Kubernetes projected ServiceAccount token・クラウドの instance identity (OIDC) token・SPIFFE の JWT-SVID を包含する第一級の subject_token 種別。X.509-SVID (mTLS) は将来の拡張。 | JWT-SVID |
| TrustDomain | WorkloadTrustBundle が束ねる論理グループ名。SPIFFE trust domain、または運用者が定める発行者グループのラベル。 |  |
| FailClosed | 未登録 issuer・署名不正・期限切れ・pattern 不一致・ambiguous match・束縛先 Agent が Active でない、のいずれかに該当する場合、常に交換を拒否する方針。判定漏れは「交換しない」側へ倒す。 |  |
| System | WorkloadIdentity の検証 usecase そのものを指す、人間の操作者を伴わない技術的な主体。 |  |

## State Transitions

### WorkloadTrustBundleLifecycle

登録時に enabled として作成される。無効化で disabled に遷移し、以後配下の binding は交換に使えなくなる (fail-closed)。再有効化で enabled に戻せる。削除は状態遷移ではなく行そのものを取り除く終端操作で、配下の binding を cascade で削除する。

Initial: `enabled`
Terminal: none

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| enabled | WorkloadTrustBundleDisabled | — | disabled |  |
| disabled | WorkloadTrustBundleEnabled | — | enabled |  |

### AgentWorkloadBindingLifecycle

作成時に enabled として作成される。無効化で disabled に遷移し以後の交換に使えなくなる。再有効化で enabled に戻せる。削除は状態遷移ではなく行そのものを取り除く終端操作。

Initial: `enabled`
Terminal: none

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| enabled | AgentWorkloadBindingDisabled | — | disabled |  |
| disabled | AgentWorkloadBindingEnabled | — | enabled |  |

## Authorization Boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.

## Design

### Internal Interfaces

#### VerifyWorkloadAttestation
OAuth2 の token-exchange grant (subject_token_type=JWT-SVID URN) から呼ぶ published interface。外部 attestation token を登録済み WorkloadTrustBundle で検証し (署名・iss・aud・exp・TTL 上限)、テナント内の AgentWorkloadBinding で subject を一意に Agent へ写し、束縛先 Agent が Active であることを確認する。いずれかに失敗すれば fail-closed で拒否する。idmagic 自身は SPIRE server/agent を実装しない (relying party 側に徹する)。
- Input invariant: trust_bundle_registered_for_issuer(context.tenant_id, input.attestation.subject_token)
- Input invariant: workload_attestation_signature_and_claims_valid(context.tenant_id, input.attestation.subject_token)
- Input invariant: !ambiguous_binding_match(context.tenant_id, input.attestation.subject_token)
- Input invariant: bound_agent_is_active(context.tenant_id, input.attestation.subject_token)
- Result invariant: output.grant == null || output.grant.client_id != ''

## Scenarios

### REQ-WORKLOADIDENTITY-001: 登録済みtrustbundle経由でワークロードトークンをAgent資格情報に交換できる
- ACTOR System
- GIVEN テナント "tenant-a" に issuer "https://issuer.example" の WorkloadTrustBundle "prod-cluster" が Enabled で登録済みである
- GIVEN "prod-cluster" 配下に subject pattern "spiffe://example.org/ns/prod/sa/*" を Agent "checkout-bot" へ写す AgentWorkloadBinding が Enabled で存在する
- GIVEN Agent "checkout-bot" は Active で、AgentCredentialBinding 経由で OAuth2Client に束縛済みである
- WHEN sub "spiffe://example.org/ns/prod/sa/worker-1" を持つ有効な JWT-SVID を subject_token として token-exchange を呼ぶ
- THEN VerifyWorkloadAttestation が Agent \"checkout-bot\" の束縛先 client_id を sub とする WorkloadIdentityGrant を返す
- THEN WorkloadTokenExchanged が発火し、束縛先 Agent の資格情報として短命な idmagic access token が発行される

### REQ-WORKLOADIDENTITY-002: 未登録issuerは拒否される
- ACTOR System
- GIVEN テナント "tenant-a" に issuer "https://issuer.example" の WorkloadTrustBundle は登録されていない
- WHEN iss "https://unknown-issuer.example" の JWT-SVID を subject_token として token-exchange を呼ぶ
- THEN VerifyWorkloadAttestation が WorkloadAttestationRejectedError (reason=unregistered_issuer) で拒否し WorkloadAttestationRejected が reason=unregistered_issuer で発火する

### REQ-WORKLOADIDENTITY-003: 署名が不正なattestationは拒否される
- ACTOR System
- GIVEN テナント "tenant-a" に issuer "https://issuer.example" の WorkloadTrustBundle "prod-cluster" が Enabled で登録済みである
- WHEN iss "https://issuer.example" を詐称するが登録済み JWKS で検証できない署名を持つ JWT を subject_token として token-exchange を呼ぶ
- THEN VerifyWorkloadAttestation が WorkloadAttestationRejectedError (reason=invalid_signature) で拒否する

### REQ-WORKLOADIDENTITY-004: 期限切れのattestationは拒否される
- ACTOR System
- GIVEN テナント "tenant-a" に issuer "https://issuer.example" の WorkloadTrustBundle "prod-cluster" が Enabled で登録済みである
- WHEN exp が過去時刻の JWT-SVID を subject_token として token-exchange を呼ぶ
- THEN VerifyWorkloadAttestation が WorkloadAttestationRejectedError (reason=expired) で拒否する

### REQ-WORKLOADIDENTITY-005: 複数bindingに曖昧にマッチするsubjectは拒否される
- ACTOR System
- GIVEN "prod-cluster" 配下に "spiffe://example.org/ns/prod/sa/*" → Agent "a" と "spiffe://example.org/ns/prod/sa/worker-*" → Agent "b" の 2 つの Enabled AgentWorkloadBinding が存在する
- WHEN sub "spiffe://example.org/ns/prod/sa/worker-1" を持つ有効な JWT-SVID で token-exchange を呼ぶ
- THEN VerifyWorkloadAttestation が WorkloadAttestationRejectedError (reason=ambiguous_match) で拒否する

### REQ-WORKLOADIDENTITY-006: 束縛先AgentがKilled後は拒否される
- ACTOR System
- GIVEN AgentWorkloadBinding の写像先 Agent "checkout-bot" が KillAgent により killed へ遷移済みである
- WHEN sub がマッチする有効な JWT-SVID で token-exchange を呼ぶ
- THEN VerifyWorkloadAttestation が WorkloadAttestationRejectedError (reason=agent_not_active) で拒否する

### REQ-WORKLOADIDENTITY-007: 他テナントのtrustbundleは利用できない
- ACTOR System
- GIVEN テナント "tenant-b" に issuer "https://issuer.example" の WorkloadTrustBundle が登録済みである
- GIVEN テナント "tenant-a" には同じ issuer の WorkloadTrustBundle が存在しない
- WHEN テナント "tenant-a" のコンテキストで iss "https://issuer.example" の JWT-SVID を subject_token として token-exchange を呼ぶ
- THEN VerifyWorkloadAttestation が WorkloadAttestationRejectedError (reason=unregistered_issuer) で拒否し tenant-b の登録は見えない

### REQ-WORKLOADIDENTITY-008: 管理者がtrustbundleを登録・無効化・再有効化できる
- ACTOR TenantAdministrator
- GIVEN 管理者としてテナントに認証済みである
- WHEN issuer "https://issuer.example" と JWKS 取得元を指定して RegisterWorkloadTrustBundle を呼ぶ
  - ALT jwks_uri と jwks のいずれも指定しない → RegisterWorkloadTrustBundle が InvalidRequestError で拒否される
  - ALT 同一テナント内に同じ issuer の WorkloadTrustBundle が既に存在する → RegisterWorkloadTrustBundle が InvalidRequestError で拒否される
- THEN WorkloadTrustBundleConfigured が発火し、WorkloadTrustBundle が enabled として作成される
- WHEN 作成した WorkloadTrustBundle に対して DisableWorkloadTrustBundle を呼ぶ
- THEN WorkloadTrustBundleDisabled が発火し、以後この bundle 配下の binding は交換に使えなくなる
- WHEN EnableWorkloadTrustBundle を呼ぶ
- THEN WorkloadTrustBundleEnabled が発火し enabled に戻る

### REQ-WORKLOADIDENTITY-009: 管理者が他テナントのAgentへbindingを作成できない
- ACTOR TenantAdministrator
- GIVEN テナント "tenant-a" に WorkloadTrustBundle "prod-cluster" が登録済みである
- GIVEN Agent "other-tenant-agent" はテナント "tenant-b" に属する
- WHEN "prod-cluster" 配下に agent_id="other-tenant-agent" の CreateAgentWorkloadBinding を呼ぶ
- THEN CreateAgentWorkloadBinding が InvalidRequestError で拒否され binding は作成されない
