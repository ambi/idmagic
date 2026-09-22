# 機能要求

IdMagic は、人とワークロードのアイデンティティを管理し、認証、認可、フェデレーション、プロビジョニング、監査を一つのプロダクト境界で提供する。

## 主な機能

| 分類 | 提供する機能 |
| --- | --- |
| 認証要素 | パスワード、TOTP、WebAuthn パスキー、リカバリーコード |
| 上流からの同期 | SCIM による利用者とグループの取り込み |
| 上流 IdP との連携 | OpenID Connect、SAML 2.0、WS-Federation |
| 下流への同期 | SCIM による利用者とグループのプロビジョニング |

## 担当する仕様

| 利用目的 | 担当する仕様 |
| --- | --- |
| テナントとアイデンティティを管理する | [Tenancy](../domain/tenancy/)、[Identity Management](../domain/identity-management/)、[Identity Governance](../domain/identity-governance/) |
| 人を認証し、セッションと認証要素を管理する | [Authentication](../domain/authentication/) |
| OAuth 2.0、OpenID Connect、SAML、WS-Federation を通じてサインオンする | [OAuth2](../domain/oauth2/)、[SAML](../domain/saml/)、[WS-Federation](../domain/ws-federation/) |
| アプリケーション、クレーム、権限を管理する | [Application](../domain/application/)、[Claim Mapping](../domain/claim-mapping/)、[Authorization](../domain/authorization/) |
| 上流と下流のシステムにアイデンティティを同期する | [Sourcing](../domain/sourcing/)、[Provisioning](../domain/provisioning/) |
| ワークロードを認証し、継続的なアクセス評価を行う | [Workload Identity](../domain/workloadidentity/)、[Shared Signals](../domain/sharedsignals/) |
| 鍵、API トークン、非同期処理、監査を管理する | [Signing Keys](../domain/signing-keys/)、[Data Keys](../domain/data-keys/)、[API Tokens](../domain/api-tokens/)、[Jobs](../domain/jobs/)、[Audit](../domain/audit/) |
| システムを構成し、初期データを投入する | [System](../domain/system/)、[Seeding](../domain/seeding/) |
