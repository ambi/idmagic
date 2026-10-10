# 機能要件

この文書は、IdMagic の機能要件を機能群の単位で列挙し、各機能群を実現するモジュールへ割り当てる。
IdMagic は、人とワークロードのアイデンティティを管理し、認証、認可、フェデレーション、プロビジョニング、監査を一つのプロダクト境界で提供する。

この文書は ISO/IEC/IEEE 29148 のシステム要件にあたり、利用者が何を行えるかを機能群の粒度で示す。
個々の振る舞いは、割り当て先のモジュールの機能仕様が `REQ-*` の要件として定める。
この文書が機能群より細かい振る舞いを書かないのは、同じ要件を二か所に書くと片方だけが更新されるためである。

| 機能群 | 利用者が行えること | 実現するモジュール |
| --- | --- | --- |
| テナントとアイデンティティの管理 | テナントを作成して構成し、利用者、グループ、エージェント、ロールを管理する。CSV で一括して取り込み、書き出す。ライフサイクルワークフローで入社、異動、退職に合わせて処理する | [Tenancy](../modules/tenancy/README.md)、[Identity Management](../modules/identity-management/README.md)、[Identity Governance](../modules/identity-governance/README.md) |
| 人の認証とセッション | パスワード、TOTP、WebAuthn パスキー、リカバリーコードで認証し、MFA、セッション、信頼済みデバイス、サインイン履歴を管理する。上流の OpenID Connect、SAML 2.0、WS-Federation の IdP と連携してサインインする | [Authentication](../modules/authentication/README.md) |
| アプリケーションへのサインオン | OAuth 2.0、OpenID Connect、SAML 2.0、WS-Federation で連携アプリケーションへサインオンし、同意と承認を管理する | [OAuth2](../modules/oauth2/README.md)、[SAML](../modules/saml/README.md)、[WS-Federation](../modules/ws-federation/README.md) |
| アプリケーション、クレーム、権限の管理 | アプリケーションを登録して利用者に割り当て、発行するクレームを対応付け、きめ細かな権限を関係で判定する | [Application](../modules/application/README.md)、[Claim Mapping](../modules/claim-mapping/README.md)、[Authorization](../modules/authorization/README.md) |
| 上流と下流のシステムとの同期 | SCIM で上流から利用者とグループを取り込み、下流の SaaS へ SCIM でプロビジョニングする | [Sourcing](../modules/sourcing/README.md)、[Provisioning](../modules/provisioning/README.md) |
| ワークロードの認証と継続的なアクセス評価 | エージェントの実行環境を外部のアテステーションで認証し、Shared Signals と CAEP でエージェントの失効をほぼリアルタイムに送受信する | [Workload Identity](../modules/workloadidentity/README.md)、[Shared Signals](../modules/sharedsignals/README.md) |
| 鍵、API トークン、非同期処理、監査 | 署名鍵とデータ暗号鍵を管理し、API アクセストークンを発行して失効させ、非同期ジョブを実行し、監査イベントを検索する | [Signing Keys](../modules/signing-keys/README.md)、[Data Keys](../modules/data-keys/README.md)、[API Tokens](../modules/api-tokens/README.md)、[Jobs](../modules/jobs/README.md)、[Audit](../modules/audit/README.md) |
| システムの構成と初期データ | ホステッドの認証画面と API の経路を提供し、表示言語を解決し、起動時設定で構成し、過負荷のときは優先度の低い要求から受け付けを止める。初期データを投入する | [System](../modules/system/README.md)、[Seeding](../modules/seeding/README.md) |
