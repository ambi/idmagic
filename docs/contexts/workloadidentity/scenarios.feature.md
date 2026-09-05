# Feature: WorkloadIdentity Scenarios

## Rule: REQ-WORKLOADIDENTITY-001 登録済みの信頼設定を使ってワークロードトークンを Agent 資格情報に交換できる

Primary actor: `System`

### Example: EX-WORKLOADIDENTITY-001-01 通常経路

- Given テナント "tenant-a" に発行者 "https://issuer.example" の WorkloadTrustBundle "prod-cluster" が `Enabled` で登録済みである
- And "prod-cluster" 配下に、主体パターン "spiffe://example.org/ns/prod/sa/*" を Agent "checkout-bot" に対応付ける `Enabled` の AgentWorkloadBinding が存在する
- And Agent "checkout-bot" は `Active` で、AgentCredentialBinding を介して OAuth2Client に関連付けられている
- When `sub` が "spiffe://example.org/ns/prod/sa/worker-1" である有効な JWT-SVID を `subject_token` として Token Exchange を呼ぶ
- Then VerifyWorkloadAttestation が Agent "checkout-bot" の関連付け先 `client_id` を `sub` とする WorkloadIdentityGrant を返す
- Then WorkloadTokenExchanged が発行され、関連付け先 Agent の資格情報として有効期間の短い IdMagic アクセストークンが発行される

## Rule: REQ-WORKLOADIDENTITY-002 未登録の発行者を拒否する

Primary actor: `System`

### Example: EX-WORKLOADIDENTITY-002-01 通常経路

- Given テナント "tenant-a" に発行者 "https://issuer.example" の WorkloadTrustBundle は登録されていない
- When `iss` が "https://unknown-issuer.example" である JWT-SVID を `subject_token` として Token Exchange を呼ぶ
- Then VerifyWorkloadAttestation が `reason=unregistered_issuer` の WorkloadAttestationRejectedError で拒否し、同じ理由の WorkloadAttestationRejected を発行する

## Rule: REQ-WORKLOADIDENTITY-003 署名が不正なアテステーションを拒否する

Primary actor: `System`

### Example: EX-WORKLOADIDENTITY-003-01 通常経路

- Given テナント "tenant-a" に発行者 "https://issuer.example" の WorkloadTrustBundle "prod-cluster" が `Enabled` で登録済みである
- When `iss` に "https://issuer.example" を指定しているが、登録済み JWKS では署名を検証できない JWT を `subject_token` として Token Exchange を呼ぶ
- Then VerifyWorkloadAttestation が `reason=invalid_signature` の WorkloadAttestationRejectedError で拒否する

## Rule: REQ-WORKLOADIDENTITY-004 期限切れのアテステーションを拒否する

Primary actor: `System`

### Example: EX-WORKLOADIDENTITY-004-01 通常経路

- Given テナント "tenant-a" に発行者 "https://issuer.example" の WorkloadTrustBundle "prod-cluster" が `Enabled` で登録済みである
- When `exp` が過去の時刻である JWT-SVID を `subject_token` として Token Exchange を呼ぶ
- Then VerifyWorkloadAttestation が `reason=expired` の WorkloadAttestationRejectedError で拒否する

## Rule: REQ-WORKLOADIDENTITY-005 複数の関連付けに一致して Agent を一意に決められない主体を拒否する

Primary actor: `System`

### Example: EX-WORKLOADIDENTITY-005-01 通常経路

- Given "prod-cluster" 配下に、"spiffe://example.org/ns/prod/sa/*" を Agent "a" に、"spiffe://example.org/ns/prod/sa/worker-*" を Agent "b" に対応付ける 2 つの `Enabled` AgentWorkloadBinding が存在する
- When `sub` が "spiffe://example.org/ns/prod/sa/worker-1" である有効な JWT-SVID を使って Token Exchange を呼ぶ
- Then VerifyWorkloadAttestation が `reason=ambiguous_match` の WorkloadAttestationRejectedError で拒否する

## Rule: REQ-WORKLOADIDENTITY-006 対応先 Agent が Killed になった後は交換を拒否する

Primary actor: `System`

### Example: EX-WORKLOADIDENTITY-006-01 通常経路

- Given AgentWorkloadBinding の対応先 Agent "checkout-bot" が KillAgent によって `killed` に遷移済みである
- When `sub` がパターンに一致する有効な JWT-SVID を使って Token Exchange を呼ぶ
- Then VerifyWorkloadAttestation が `reason=agent_not_active` の WorkloadAttestationRejectedError で拒否する

## Rule: REQ-WORKLOADIDENTITY-007 他テナントの信頼設定は利用できない

Primary actor: `System`

### Example: EX-WORKLOADIDENTITY-007-01 通常経路

- Given テナント "tenant-b" に発行者 "https://issuer.example" の WorkloadTrustBundle が登録済みである
- And テナント "tenant-a" には同じ発行者の WorkloadTrustBundle が存在しない
- When テナント "tenant-a" の実行コンテキストで、`iss` が "https://issuer.example" である JWT-SVID を `subject_token` として Token Exchange を呼ぶ
- Then VerifyWorkloadAttestation が `reason=unregistered_issuer` の WorkloadAttestationRejectedError で拒否し、テナント "tenant-b" の登録内容は参照されない

## Rule: REQ-WORKLOADIDENTITY-008 管理者は信頼設定を登録・無効化・再有効化できる

Primary actor: `TenantAdministrator`

### Example: EX-WORKLOADIDENTITY-008-01 通常経路

- Given 管理者としてテナントに認証済みである
- When 発行者 "https://issuer.example" と JWKS の取得元を指定して RegisterWorkloadTrustBundle を呼ぶ
- Then WorkloadTrustBundleConfigured が発行され、WorkloadTrustBundle が `enabled` として作成される
- When 作成した WorkloadTrustBundle に対して DisableWorkloadTrustBundle を呼ぶ
- Then WorkloadTrustBundleDisabled が発行され、以後この信頼設定に属する関連付けは交換に使えなくなる
- When EnableWorkloadTrustBundle を呼ぶ
- Then WorkloadTrustBundleEnabled が発行され、`enabled` に戻る

### Example: EX-WORKLOADIDENTITY-008-02 `jwks_uri` と `jwks` のどちらも指定しない

- Given 管理者としてテナントに認証済みである
- When 発行者 "https://issuer.example" と JWKS の取得元を指定して RegisterWorkloadTrustBundle を呼ぶ
- But `jwks_uri` と `jwks` のどちらも指定しない
- Then RegisterWorkloadTrustBundle が InvalidRequestError で拒否される

### Example: EX-WORKLOADIDENTITY-008-03 同じテナント内に同じ発行者の WorkloadTrustBundle がすでに存在する

- Given 管理者としてテナントに認証済みである
- When 発行者 "https://issuer.example" と JWKS の取得元を指定して RegisterWorkloadTrustBundle を呼ぶ
- But 同じテナント内に同じ発行者の WorkloadTrustBundle がすでに存在する
- Then RegisterWorkloadTrustBundle が InvalidRequestError で拒否される

## Rule: REQ-WORKLOADIDENTITY-009 管理者は他テナントの Agent への関連付けを作成できない

Primary actor: `TenantAdministrator`

### Example: EX-WORKLOADIDENTITY-009-01 通常経路

- Given テナント "tenant-a" に WorkloadTrustBundle "prod-cluster" が登録済みである
- And Agent "other-tenant-agent" はテナント "tenant-b" に属する
- When "prod-cluster" 配下に `agent_id="other-tenant-agent"` を指定して CreateAgentWorkloadBinding を呼ぶ
- Then CreateAgentWorkloadBinding が InvalidRequestError で拒否され、関連付けは作成されない

## Rule: REQ-WORKLOADIDENTITY-010 信頼設定と関連付けの管理は管理者に限られる

Primary actor: `TenantAdministrator`

### Example: EX-WORKLOADIDENTITY-010-01 通常経路

- Given "alice" は認証済みだが `admin` ロールを持たない
- When "alice" が WorkloadTrustBundle の登録を要求する
- Then AccessDeniedError で拒否される
- Then WorkloadTrustBundle は作成されず、既存の信頼設定と関連付けの状態も変わらない

### Example: EX-WORKLOADIDENTITY-010-02 "alice" が信頼設定の更新、無効化、再有効化、削除、JWKS の再取得、または関連付けの作成・無効化・再有効化・削除を要求する

- Given "alice" は認証済みだが `admin` ロールを持たない
- When "alice" が WorkloadTrustBundle の登録を要求する
- But "alice" が信頼設定の更新、無効化、再有効化、削除、JWKS の再取得、または関連付けの作成・無効化・再有効化・削除を要求する
- Then AccessDeniedError で拒否される
