# WorkloadIdentity Scenarios

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
