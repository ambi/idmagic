# 論理アーキテクチャ

## アーキテクチャ様式

IdMagic は、一つの Go モジュール内で Bounded Context のモデル境界を保つ Modular Monolith である。独立したデータ所有権、担当チーム、サービス目標が必要になるまではサービスへ分割しない。ドメイン層とユースケース層は外部技術へ依存せず、ポートを通じて PostgreSQL、HTTP、通知などのアダプターへ接続する。

コードの依存方向と配置は [構造](../structure.md) が、境界とモジュールを評価する規則は [設計規則](../design/application/design-rules.md) が所有する。論理単位をプロセスへ割り当てる方法は [実行時アーキテクチャ](runtime.md) が所有する。

## Context Map

この図はドメイン上の関係と統合境界を示す。矢印は Supplier から Customer へ向かう。`OHS/PL` は Published Language を伴う Open Host Service、`C/S` は Customer/Supplier、`ACL` は Anti-Corruption Layer を表す。

ドメインイベントの関係は図に描かない。発行する Context は共通のワイヤ表現を使い、組み立て地点にある一つの配信点を通じて Audit と Authentication の通知機構へ事実を渡すため、Context 間の import を生まない。契約となる語彙は [Context 間イベント](../structure.md#context-間イベント) が所有する。

```mermaid
flowchart LR
  Tenancy[Tenancy]
  IdManagement[IdManagement]
  IdGovernance[IdGovernance]
  Authentication[Authentication]
  OAuth2[OAuth2]
  Application[Application]
  ClaimMapping[ClaimMapping]
  Provisioning[Provisioning]
  Sourcing[Sourcing]
  ApiTokens[ApiTokens]
  Jobs[Jobs]
  Seeding[Seeding]
  SigningKeys[SigningKeys]
  DataKeys[DataKeys]
  WsFederation[WsFederation]
  Saml[Saml]
  WorkloadIdentity[WorkloadIdentity]
  SharedSignals[SharedSignals]
  Audit[Audit]
  Authorization[Authorization]
  System[System]

  Tenancy -->|OHS/PL: tenant boundary| IdManagement
  Tenancy -->|OHS/PL: tenant settings| Application
  IdManagement -->|OHS/PL: principals| Authentication
  IdManagement -->|OHS/PL: user mutation commit| IdGovernance
  IdGovernance -->|C/S: governed mutations| IdManagement
  IdManagement -->|OHS/PL: user lifecycle triggers| Provisioning
  Sourcing -->|ACL: authoritative identity| IdManagement
  Authentication -->|OHS/PL: authenticated subject| OAuth2
  Application -->|C/S: protocol binding and gate| OAuth2
  Application -->|C/S: protocol binding and gate| Saml
  Application -->|C/S: protocol binding and gate| WsFederation
  ClaimMapping -->|OHS/PL: released claims| OAuth2
  ClaimMapping -->|OHS/PL: released claims| Saml
  ClaimMapping -->|OHS/PL: released claims| WsFederation
  SigningKeys -->|OHS/PL: signing service| OAuth2
  SigningKeys -->|OHS/PL: XML signing service| Saml
  SigningKeys -->|OHS/PL: XML signing service| WsFederation
  SigningKeys -->|OHS/PL: SET signing service| SharedSignals
  DataKeys -->|OHS/PL: encryption-key lifecycle| Authentication
  WorkloadIdentity -->|ACL: workload attestation| OAuth2
  ApiTokens -->|OHS/PL: API principal| System
  Jobs -->|OHS/PL: durable execution| IdGovernance
  Jobs -->|OHS/PL: durable execution| Provisioning
  Jobs -->|OHS/PL: durable execution| SharedSignals
  Seeding -->|C/S: published commands| Tenancy
  Seeding -->|C/S: published commands| IdManagement
  Seeding -->|C/S: published commands| Application
  OAuth2 -->|OHS/PL: AuthZEN policy evaluation| Authorization
  IdManagement -->|OHS/PL: principal status| Authorization
  System -->|C/S: UI and runtime composition| Authentication
  System -->|C/S: UI and runtime composition| Application
```

## Context の責務

| 仕様上の Context | Subdomain | Go パッケージ | 責務 |
| --- | --- | --- | --- |
| [System](../contexts/system/README.md) | Supporting | `backend/cmd/internal/bootstrap`, `backend/shared/http/server_http`, `frontend/` | 起動、機能選択、経路の組み立て、健全性、画面 |
| [Tenancy](../contexts/tenancy/README.md) | Supporting | `backend/tenancy` | Tenant と realm、テナント設定、属性スキーマ、制御面の管理 |
| [IdManagement](../contexts/identity-management/README.md) | Core | `backend/idmanagement` | User、Group、Agent、プロフィール、アイデンティティのライフサイクル |
| [IdGovernance](../contexts/identity-governance/README.md) | Supporting | `backend/idgovernance` | LifecycleWorkflow のポリシーとオーケストレーション |
| [Authentication](../contexts/authentication/README.md) | Core | `backend/authentication` | 資格情報、MFA、ログインセッション、ステップアップ、認証イベント |
| [OAuth2](../contexts/oauth2/README.md) | Core | `backend/oauth2` | OAuth 2.0 と OIDC のプロトコル、クライアント、同意、トークン |
| [Application](../contexts/application/README.md) | Supporting | `backend/application` | Application、プロトコルのバインディング、割り当て、表示分類 |
| [Authorization](../contexts/authorization/README.md) | Core | `backend/authorization` | 細粒度認可モデル、関係タプル、グラフ評価、整合トークン |
| [Audit](../contexts/audit/README.md) | Supporting | `backend/audit` | 全 Context の監査イベントを統合する Read Model と保持 |
| [ClaimMapping](../contexts/claim-mapping/README.md) | Supporting | `backend/claimmapping` | プロトコル非依存のクレーム開示ポリシーとマッピング |
| [Provisioning](../contexts/provisioning/README.md) | Supporting | `backend/provisioning` | IdMagic を正とする外向き SCIM プロビジョニング |
| [Sourcing](../contexts/sourcing/README.md) | Supporting | `backend/sourcing` | 上流の権威からの内向きアイデンティティ取り込み |
| [ApiTokens](../contexts/api-tokens/README.md) | Generic | `backend/apitoken` | 管理 API と SCIM API のテナント単位アクセストークン |
| [Jobs](../contexts/jobs/README.md) | Generic | `backend/jobs` | テナント境界を保つ非同期ジョブ基盤 |
| [Seeding](../contexts/seeding/README.md) | Supporting | `backend/seeding` | 環境構成のプレビュー、計画、初期データ適用 |
| [SigningKeys](../contexts/signing-keys/README.md) | Supporting | `backend/signingkeys` | 用途別署名鍵、証明書、ローテーション、提供元アダプター |
| [DataKeys](../contexts/data-keys/README.md) | Generic | `backend/datakeys` | 可逆な秘密情報を保護するテナント別 DEK のライフサイクル |
| [WsFederation](../contexts/ws-federation/README.md) | Generic | `backend/wsfederation` | WS-Federation、WS-Trust、メタデータ、RP の信頼 |
| [Saml](../contexts/saml/README.md) | Generic | `backend/saml` | SAML 2.0 IdP、SP の信頼、SSO、SLO、メタデータ |
| [WorkloadIdentity](../contexts/workloadidentity/README.md) | Core | `backend/workloadidentity` | 外部アテステーションと Agent の対応付け |
| [SharedSignals](../contexts/sharedsignals/README.md) | Supporting | `backend/sharedsignals` | SSF、SET、CAEP による継続的アクセス評価と失効 |

各 Subdomain 区分の理由は Context の `decisions.md` が所有する。Context 間の統合では供給側の公開言語を使い、顧客側で語彙の差を翻訳する。PostgreSQL は業務データと短命状態の論理的な正本だが、テーブルを共有して Context 境界を迂回してはならない。
