# Saml

## 責務と境界

SAML 2.0 の IdP として、SP の信頼、IdP プロファイル、IdP メタデータ、AuthnRequest と Response、AssertionConsumerService、Single Logout を扱う。
Web Browser SSO Profile に基づき、SP 起点と IdP 起点の SSO を提供する。

| 扱わないもの | 担当 |
| --- | --- |
| プロトコルに依存しないクレームの対応付け | `ClaimMapping` |
| Assertion の組み立てと XML 署名の部品 | `WS-Federation` の `tokens_saml`。この Context は再利用する |
| 署名鍵のライフサイクル | `SigningKeys` |
| ブラウザーのログインセッション | `Authentication` |
| Application への割り当て | `Application` |
| 暗号化した Assertion、ECP、SAML の SP としての動作 | 提供しない |

## モデル

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `SamlServiceProvider` | entityID、許可する ACS の URL、署名の要否（Assertion と Response）、要求の署名の検証の鍵、`ClaimMappingPolicy`、割り当てた IdP プロファイル | `Tenant` を `tenant_id` で参照する。`SamlIdentityProviderProfile` を一つだけ参照する。`Application` から参照される |
| `SamlIdentityProviderProfile` | 名前、モード（`shared` または `dedicated`）、サーバーが生成する entityID | `Tenant` を `tenant_id` で参照する。署名の資格情報は、`SigningKeys` の `XmlFederationSigning` の用途でこのプロファイルをスコープとする |

テナントには、変更できない `default` のプロファイルが必ずあり、複数の SP で共有する。

- **判断**：`shared` と `dedicated` を一つのモデルで表す理由は、[IdP プロファイルを一つの共有できるモデルで表す](design/decisions.md#idp-プロファイルを一つの共有できるモデルで表す)。

## 公開する契約

HTTP の操作とモデルの形は TypeSpec の `SAML` のタグが定める。
SSO、SLO、メタデータ、署名証明書は、テナントのデフォルトのプロファイルの標準の探索の経路（`/saml/*`）と、名前付きの IdP プロファイルの経路（`/saml/idp/{profile_id}/*`）をどちらも提供する。
SSO と SLO は、さらに HTTP-Redirect の GET と HTTP-POST の POST を分ける。
同じプロトコルの操作でも HTTP のメソッドとパスが異なるので、公開の契約では各経路に一意の `operationId` を与える。
次の表は、それ以外にほかの Context と結ぶ契約である。

| 契約 | 相手 | 向き | 内容 |
| --- | --- | --- | --- |
| `ServiceProviderRepository` | `Application` のプロトコル設定 | この Context が提供する | アプリケーションに属する SP の設定を読み書きする |
| `IssueClaimsWithFloor` | `ClaimMapping` が提供する | この Context が使う | SP のポリシーでクレームを発行する |
| XML 署名の資格情報 | `SigningKeys` が提供する | この Context が使う | テナントとプロファイルの `XmlFederationSigning` の鍵と証明書を発行のたびに得る |
| ドメインイベント | 監査と下流が購読する | この Context が発行する | `SamlSignInIssued`、`SamlSignInRejected`、`SamlLogout` |

## 機能

| 機能 | 内容 |
| --- | --- |
| [IdP プロファイルとメタデータ](idp-profile/README.md) | IdP プロファイルの管理と、メタデータと署名証明書の公開 |
| [サービスプロバイダーの管理](service-provider/README.md) | SP の登録、参照、削除 |
| [SAML の SSO](sso/README.md) | SP 起点と IdP 起点の SSO と Single Logout |

| 文書 | 内容 |
| --- | --- |
| [Saml の用語集](glossary.md) | この Context での語義 |
| [Saml の標準仕様](standards.md) | 採用する外部標準仕様 |
| [Saml の設計](design/README.md) | 話題ごとの設計と重要な判断 |
