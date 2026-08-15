---
context: workloadidentity
updated_at: 2026-08-15
---

# WorkloadIdentity Specification

## Overview

自律エージェントの実行環境に対するワークロードアイデンティティフェデレーションを所有する。テナントが登録した外部アテステーション発行者の信頼設定 (`WorkloadTrustBundle`) と、外部の主体を既存の `Agent` に対応付ける `AgentWorkloadBinding` を管理する。主なアテステーション形式は、Kubernetes の投影 ServiceAccount トークン、クラウドのインスタンスアイデンティティトークン、SPIFFE JWT-SVID などの OIDC 互換 JWT である。

IdMagic は SPIRE のサーバーやエージェントを同梱・運用せず、外部アテステーションを検証する RP として動作する。検証済みのアテステーションは OAuth2 Token Exchange グラント (RFC 8693) の subject として渡し、長期シークレットを配布する専用の資格情報経路は設けない。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| WorkloadTrustBundle | テナントが登録する外部アテステーション発行者の信頼設定。トラストドメイン、発行者、JWKS の取得元またはインライン JWKS、受理する audience、外部 SVID の最大 TTL をまとめる。事前に登録した発行者だけを信頼し、Trust On First Use は許可しない。 |  |
| AgentWorkloadBinding | `WorkloadTrustBundle` の配下で、外部主体に対する glob パターンを同じテナントの既存 `Agent` に対応付けるレコード。パターンに一致しない主体や、複数の有効な関連付けに一致するため対象を一意に決められない主体は、フェイルクローズで拒否する。 |  |
| JwtSvid | OIDC 互換 JWT を通信形式とする外部アテステーショントークン。Kubernetes の投影 ServiceAccount トークン、クラウドのインスタンスアイデンティティトークン、SPIFFE JWT-SVID を包含する、主要な `subject_token` 種別。X.509-SVID（mTLS）は将来の拡張とする。 | JWT-SVID |
| TrustDomain | `WorkloadTrustBundle` がまとめる論理グループ名。SPIFFE トラストドメイン、または運用者が定める発行者グループのラベル。 |  |
| FailClosed | 未登録の発行者、署名不正、期限切れ、パターン不一致、一意に決められない一致、対応先の `Agent` が `Active` でない場合のいずれでも、交換を拒否する方針。判定できない場合も「交換しない」側に倒す。 |  |
| System | `WorkloadIdentity` の検証ユースケースそのものを指す、人間の操作者を伴わない技術的な主体。 |  |

## State Transitions

### WorkloadTrustBundleLifecycle

登録時に `enabled` として作成する。無効化すると `disabled` に遷移し、それ以降は配下の関連付けを交換に使えない。再有効化すれば `enabled` に戻せる。削除は状態遷移ではなくレコードそのものを取り除く終端操作であり、配下の関連付けもカスケード削除する。

Initial: `enabled` Terminal: none

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| enabled | WorkloadTrustBundleDisabled | — | disabled |  |
| disabled | WorkloadTrustBundleEnabled | — | enabled |  |

### AgentWorkloadBindingLifecycle

作成時は `enabled` とする。無効化すると `disabled` に遷移し、それ以降の交換には使えない。再有効化すれば `enabled` に戻せる。削除は状態遷移ではなく行そのものを取り除く終端操作である。

Initial: `enabled` Terminal: none

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| enabled | AgentWorkloadBindingDisabled | — | disabled |  |
| disabled | AgentWorkloadBindingEnabled | — | enabled |  |

## Authorization Boundary

`WorkloadTrustBundle` と `AgentWorkloadBinding` の登録、更新、無効化、削除は、`admin` ロールを持つ、有効かつ認証済みのユーザーだけが所属テナントに対して行える。関連付けの対象にできる `Agent` も同じテナントのものに限る。API アクセストークンでは、ロールに加えて `workload-identity:read` が信頼設定と関連付けの参照だけを、`workload-identity:write` がその変更を許可する。JWKS の再取得は保存する鍵素材を差し替えるため変更系に置く。

交換の経路そのものは管理者の権限を通らない。外部ワークロードが提示するのはアテステーショントークンだけであり、得られる権限は登録済みの信頼設定と関連付けが定めた `Agent` のものに固定される。トークンの内容が対応先の `Agent` を変えることはなく、未登録の発行者を実行時に信頼することもない (Trust On First Use を許可しない)。

## Design

### Internal Interfaces

#### VerifyWorkloadAttestation
OAuth2 Token Exchange グラント（`subject_token_type` は JWT-SVID URN）から呼び出す公開インターフェースである。外部アテステーショントークンの署名、`iss`、`aud`、`exp`、TTL 上限を登録済みの `WorkloadTrustBundle` で検証する。次に、テナント内の `AgentWorkloadBinding` から対応する `Agent` を一意に特定し、その `Agent` が `Active` であることを確認する。いずれかの検証に失敗した場合はフェイルクローズで拒否する。
- Input invariant: trust_bundle_registered_for_issuer(context.tenant_id, input.attestation.subject_token)
- Input invariant: workload_attestation_signature_and_claims_valid(context.tenant_id, input.attestation.subject_token)
- Input invariant: !ambiguous_binding_match(context.tenant_id, input.attestation.subject_token)
- Input invariant: bound_agent_is_active(context.tenant_id, input.attestation.subject_token)
- Result invariant: output.grant == null || output.grant.client_id != ''

## Scenarios

### REQ-WORKLOADIDENTITY-001: 登録済みの信頼設定を使ってワークロードトークンを Agent 資格情報に交換できる
- ACTOR System
- GIVEN テナント "tenant-a" に発行者 "https://issuer.example" の WorkloadTrustBundle "prod-cluster" が `Enabled` で登録済みである
- GIVEN "prod-cluster" 配下に、主体パターン "spiffe://example.org/ns/prod/sa/*" を Agent "checkout-bot" に対応付ける `Enabled` の AgentWorkloadBinding が存在する
- GIVEN Agent "checkout-bot" は `Active` で、AgentCredentialBinding を介して OAuth2Client に関連付けられている
- WHEN `sub` が "spiffe://example.org/ns/prod/sa/worker-1" である有効な JWT-SVID を `subject_token` として Token Exchange を呼ぶ
- THEN VerifyWorkloadAttestation が Agent "checkout-bot" の関連付け先 `client_id` を `sub` とする WorkloadIdentityGrant を返す
- THEN WorkloadTokenExchanged が発行され、関連付け先 Agent の資格情報として有効期間の短い IdMagic アクセストークンが発行される

### REQ-WORKLOADIDENTITY-002: 未登録の発行者を拒否する
- ACTOR System
- GIVEN テナント "tenant-a" に発行者 "https://issuer.example" の WorkloadTrustBundle は登録されていない
- WHEN `iss` が "https://unknown-issuer.example" である JWT-SVID を `subject_token` として Token Exchange を呼ぶ
- THEN VerifyWorkloadAttestation が `reason=unregistered_issuer` の WorkloadAttestationRejectedError で拒否し、同じ理由の WorkloadAttestationRejected を発行する

### REQ-WORKLOADIDENTITY-003: 署名が不正なアテステーションを拒否する
- ACTOR System
- GIVEN テナント "tenant-a" に発行者 "https://issuer.example" の WorkloadTrustBundle "prod-cluster" が `Enabled` で登録済みである
- WHEN `iss` に "https://issuer.example" を指定しているが、登録済み JWKS では署名を検証できない JWT を `subject_token` として Token Exchange を呼ぶ
- THEN VerifyWorkloadAttestation が `reason=invalid_signature` の WorkloadAttestationRejectedError で拒否する

### REQ-WORKLOADIDENTITY-004: 期限切れのアテステーションを拒否する
- ACTOR System
- GIVEN テナント "tenant-a" に発行者 "https://issuer.example" の WorkloadTrustBundle "prod-cluster" が `Enabled` で登録済みである
- WHEN `exp` が過去の時刻である JWT-SVID を `subject_token` として Token Exchange を呼ぶ
- THEN VerifyWorkloadAttestation が `reason=expired` の WorkloadAttestationRejectedError で拒否する

### REQ-WORKLOADIDENTITY-005: 複数の関連付けに一致して Agent を一意に決められない主体を拒否する
- ACTOR System
- GIVEN "prod-cluster" 配下に、"spiffe://example.org/ns/prod/sa/*" を Agent "a" に、"spiffe://example.org/ns/prod/sa/worker-*" を Agent "b" に対応付ける 2 つの `Enabled` AgentWorkloadBinding が存在する
- WHEN `sub` が "spiffe://example.org/ns/prod/sa/worker-1" である有効な JWT-SVID を使って Token Exchange を呼ぶ
- THEN VerifyWorkloadAttestation が `reason=ambiguous_match` の WorkloadAttestationRejectedError で拒否する

### REQ-WORKLOADIDENTITY-006: 対応先 Agent が Killed になった後は交換を拒否する
- ACTOR System
- GIVEN AgentWorkloadBinding の対応先 Agent "checkout-bot" が KillAgent によって `killed` に遷移済みである
- WHEN `sub` がパターンに一致する有効な JWT-SVID を使って Token Exchange を呼ぶ
- THEN VerifyWorkloadAttestation が `reason=agent_not_active` の WorkloadAttestationRejectedError で拒否する

### REQ-WORKLOADIDENTITY-007: 他テナントの信頼設定は利用できない
- ACTOR System
- GIVEN テナント "tenant-b" に発行者 "https://issuer.example" の WorkloadTrustBundle が登録済みである
- GIVEN テナント "tenant-a" には同じ発行者の WorkloadTrustBundle が存在しない
- WHEN テナント "tenant-a" の実行コンテキストで、`iss` が "https://issuer.example" である JWT-SVID を `subject_token` として Token Exchange を呼ぶ
- THEN VerifyWorkloadAttestation が `reason=unregistered_issuer` の WorkloadAttestationRejectedError で拒否し、テナント "tenant-b" の登録内容は参照されない

### REQ-WORKLOADIDENTITY-008: 管理者は信頼設定を登録・無効化・再有効化できる
- ACTOR TenantAdministrator
- GIVEN 管理者としてテナントに認証済みである
- WHEN 発行者 "https://issuer.example" と JWKS の取得元を指定して RegisterWorkloadTrustBundle を呼ぶ
  - ALT `jwks_uri` と `jwks` のどちらも指定しない → RegisterWorkloadTrustBundle が InvalidRequestError で拒否される
  - ALT 同じテナント内に同じ発行者の WorkloadTrustBundle がすでに存在する → RegisterWorkloadTrustBundle が InvalidRequestError で拒否される
- THEN WorkloadTrustBundleConfigured が発行され、WorkloadTrustBundle が `enabled` として作成される
- WHEN 作成した WorkloadTrustBundle に対して DisableWorkloadTrustBundle を呼ぶ
- THEN WorkloadTrustBundleDisabled が発行され、以後この信頼設定に属する関連付けは交換に使えなくなる
- WHEN EnableWorkloadTrustBundle を呼ぶ
- THEN WorkloadTrustBundleEnabled が発行され、`enabled` に戻る

### REQ-WORKLOADIDENTITY-009: 管理者は他テナントの Agent への関連付けを作成できない
- ACTOR TenantAdministrator
- GIVEN テナント "tenant-a" に WorkloadTrustBundle "prod-cluster" が登録済みである
- GIVEN Agent "other-tenant-agent" はテナント "tenant-b" に属する
- WHEN "prod-cluster" 配下に `agent_id="other-tenant-agent"` を指定して CreateAgentWorkloadBinding を呼ぶ
- THEN CreateAgentWorkloadBinding が InvalidRequestError で拒否され、関連付けは作成されない
