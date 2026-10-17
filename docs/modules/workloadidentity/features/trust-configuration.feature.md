# Feature: 信頼設定と関連付けの管理の例

## Rule: REQ-WORKLOADIDENTITY-008 管理者は信頼設定を登録・無効化・再有効化できる

### Example: EX-WORKLOADIDENTITY-008-01 通常経路

- Given 管理者としてテナントに認証済みである
- When 発行者 "https://issuer.example" と JWKS の取得元を指定して RegisterWorkloadTrustBundle を呼ぶ
- Then WorkloadTrustBundleConfigured が発行され、WorkloadTrustBundle が `enabled` として作成される
- When 作成した WorkloadTrustBundle に対して DisableWorkloadTrustBundle を呼ぶ
- Then WorkloadTrustBundleDisabled が発行され、以後この信頼設定に属する関連付けは交換に使えなくなる
- When EnableWorkloadTrustBundle を呼ぶ
- Then WorkloadTrustBundleEnabled が発行され、`enabled` に戻る

### Scenario Outline: 条件ごとの結果

- Given 管理者としてテナントに認証済みである
- When 発行者 "https://issuer.example" と JWKS の取得元を指定して RegisterWorkloadTrustBundle を呼ぶ
- But <condition>
- Then <result>

#### Examples:

  | example_id | condition | result |
  | --- | --- | --- |
  | EX-WORKLOADIDENTITY-008-02 | `jwks_uri` と `jwks` のどちらも指定しない | RegisterWorkloadTrustBundle が WorkloadTrustBundleJwksRequiredError で拒否される |
  | EX-WORKLOADIDENTITY-008-03 | 同じテナント内に同じ発行者の WorkloadTrustBundle がすでに存在する | RegisterWorkloadTrustBundle が WorkloadTrustBundleIssuerConflictError で拒否される |

## Rule: REQ-WORKLOADIDENTITY-009 管理者は他テナントの Agent への関連付けを作成できない

### Example: EX-WORKLOADIDENTITY-009-01 通常経路

- Given テナント "tenant-a" に WorkloadTrustBundle "prod-cluster" が登録済みである
- And Agent "other-tenant-agent" はテナント "tenant-b" に属する
- When "prod-cluster" 配下に `agent_id="other-tenant-agent"` を指定して CreateAgentWorkloadBinding を呼ぶ
- Then CreateAgentWorkloadBinding が AgentWorkloadBindingAgentNotFoundError で拒否され、関連付けは作成されない

## Rule: REQ-WORKLOADIDENTITY-010 信頼設定と関連付けの管理は管理者に限られる

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
