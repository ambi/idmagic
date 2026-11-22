# UI ページ

この文書は、利用者の作業で分類したページ設計の入口を示す。

TypeSpec の UI 宣言から `mise run generate-ui-pages` で生成する。手で編集しない。

## 主担当のモジュール

| モジュール | ページ数 |
| --- | --- |
| [ApiTokens](../../../modules/api-tokens/pages/README.md) | 4 |
| [Application](../../../modules/application/pages/README.md) | 6 |
| [Audit](../../../modules/audit/pages/README.md) | 2 |
| [Authentication](../../../modules/authentication/pages/README.md) | 12 |
| [AccessControl](../../../modules/authorization/pages/README.md) | 8 |
| [DataKeys](../../../modules/data-keys/pages/README.md) | 1 |
| [Frontend](../../../modules/frontend/pages/README.md) | 5 |
| [IdGovernance](../../../modules/identity-governance/pages/README.md) | 8 |
| [IdManagement](../../../modules/identity-management/pages/README.md) | 28 |
| [Jobs](../../../modules/jobs/pages/README.md) | 2 |
| [OAuth2](../../../modules/oauth2/pages/README.md) | 18 |
| [Provisioning](../../../modules/provisioning/pages/README.md) | 2 |
| [Saml](../../../modules/saml/pages/README.md) | 4 |
| [SigningKeys](../../../modules/signing-keys/pages/README.md) | 3 |
| [Sourcing](../../../modules/sourcing/pages/README.md) | 4 |
| [Tenancy](../../../modules/tenancy/pages/README.md) | 11 |
| [TokenVault](../../../modules/token-vault/pages/README.md) | 3 |
| [WorkloadIdentity](../../../modules/workloadidentity/pages/README.md) | 5 |
| [WsFederation](../../../modules/ws-federation/pages/README.md) | 4 |

## 領域

| 領域 | カテゴリ | ページ数 |
| --- | --- | --- |
| [認証](authentication/README.md) | サインイン、アカウント復旧、入口と共通表示、プロフィール、同意と承認 | 11 |
| [アカウントポータル](account/README.md) | API トークン、アプリケーション、セキュリティ、入口と共通表示、ライフサイクル、プロフィール、同意と承認、外部サービス接続 | 15 |
| [管理コンソール](admin/README.md) | API トークン、セキュリティ、アプリケーション、監査、フェデレーション、認可、入口と共通表示、ライフサイクル、Agent 管理、ユーザー管理、グループ管理、ジョブ、同意と承認、プロビジョニング、鍵、ディレクトリ連携、設定、外部サービス接続、ワークロード ID | 94 |
| [システムコンソール](system/README.md) | 監査、鍵、ジョブ、テナント | 10 |

## 共通の設計

- [UI の共通規則](../user-interface.md)
- [フロントエンド設計](../frontend.md)
- [共通表示と外部の到達先](../../../modules/frontend/pages/shared.md)
