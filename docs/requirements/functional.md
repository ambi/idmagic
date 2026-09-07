# 機能要求

IdMagic は、人とワークロードのアイデンティティを管理し、認証、認可、フェデレーション、プロビジョニング、監査を一つのプロダクト境界で提供する。
利用者の目的と担当する詳細仕様は次のように割り当てる。

| 利用目的 | 担当する仕様 |
| --- | --- |
| テナントとアイデンティティを管理する | [Tenancy](../contexts/tenancy/)、[Identity Management](../contexts/identity-management/)、[Identity Governance](../contexts/identity-governance/) |
| 人を認証し、セッションと認証要素を管理する | [Authentication](../contexts/authentication/) |
| OAuth 2.0、OpenID Connect、SAML、WS-Federation を通じてサインオンする | [OAuth2](../contexts/oauth2/)、[SAML](../contexts/saml/)、[WS-Federation](../contexts/ws-federation/) |
| アプリケーション、クレーム、権限を管理する | [Application](../contexts/application/)、[Claim Mapping](../contexts/claim-mapping/)、[Authorization](../contexts/authorization/) |
| 上流と下流のシステムにアイデンティティを同期する | [Sourcing](../contexts/sourcing/)、[Provisioning](../contexts/provisioning/) |
| ワークロードを認証し、継続的なアクセス評価を行う | [Workload Identity](../contexts/workloadidentity/)、[Shared Signals](../contexts/sharedsignals/) |
| 鍵、API トークン、非同期処理、監査を管理する | [Signing Keys](../contexts/signing-keys/)、[Data Keys](../contexts/data-keys/)、[API Tokens](../contexts/api-tokens/)、[Jobs](../contexts/jobs/)、[Audit](../contexts/audit/) |
| システムを構成し、初期データを投入する | [System](../contexts/system/)、[Seeding](../contexts/seeding/) |

複数の Context が協調しなければ満たせない振る舞いは[システム横断シナリオ](../scenarios.feature.md)が持つ。
API のモデル、操作、HTTP バインディング、認証機構は `spec/contexts/` の TypeSpec が持つ。
