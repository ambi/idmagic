# Saml

## 責務と境界

SAML 2.0 の IdP として、SP の信頼、IdP プロファイル、IdP メタデータ、AuthnRequest と Response、AssertionConsumerService、Single Logout を扱う。
Web Browser SSO Profile に基づき、SP 起点と IdP 起点の SSO を提供する。

| 扱わないもの | 担当 |
| --- | --- |
| プロトコルに依存しないクレームの対応付け | `ClaimMapping` |
| Assertion の組み立てと XML 署名の部品 | 共有ライブラリ `backend/shared/security/tokens_saml`。WS-Federation と共有する |
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

- **判断**：`shared` と `dedicated` を一つのモデルで表す理由は、[IdP プロファイルを一つの共有できるモデルで表す](decisions.md#idp-プロファイルを一つの共有できるモデルで表す)。

## 公開する契約

HTTP の操作とモデルの形は TypeSpec の `SAML` のタグが定める。
SSO、SLO、メタデータ、署名証明書は、テナントのデフォルトのプロファイルの標準の探索の経路（`/saml/*`）と、名前付きの IdP プロファイルの経路（`/saml/idp/{profile_id}/*`）をどちらも提供する。
SSO と SLO は、さらに HTTP-Redirect の GET と HTTP-POST の POST を分ける。
同じプロトコルの操作でも HTTP のメソッドとパスが異なるので、公開の契約では各経路に一意の `operationId` を与える。
次の表は、それ以外にほかのモジュールと結ぶ契約である。

| 契約 | 相手 | 向き | 内容 |
| --- | --- | --- | --- |
| `ServiceProviderRepository` | `Application` のプロトコル設定 | このモジュールが提供する | アプリケーションに属する SP の設定を読み書きする |
| `IssueClaimsWithFloor` | `ClaimMapping` が提供する | このモジュールが使う | SP のポリシーでクレームを発行する |
| XML 署名の資格情報 | `SigningKeys` が提供する | このモジュールが使う | テナントとプロファイルの `XmlFederationSigning` の鍵と証明書を発行のたびに得る |
| ドメインイベント | 監査と下流が購読する | このモジュールが発行する | `SamlSignInIssued`、`SamlSignInRejected`、`SamlLogout` |

## 機能

| 機能 | 内容 |
| --- | --- |
| [IdP プロファイルとメタデータ](features/idp-profile.md) | IdP プロファイルの管理と、メタデータと署名証明書の公開 |
| [サービスプロバイダーの管理](features/service-provider.md) | SP の登録、参照、削除 |
| [SAML の SSO](features/sso.md) | SP 起点と IdP 起点の SSO と Single Logout |

## 文書

| 設計領域 | 区分 | 参照先 | 内容 |
| --- | --- | --- | --- |
| アーキテクチャ設計 | 固有設計 | [担当文書](architecture.md#文脈と範囲) | 文脈と範囲、解決戦略、構成要素、実行時の流れ |
| アプリケーション設計 | 委譲 | [標準仕様](standards.md#assertions-and-protocols-for-the-oasis-security-assertion-markup-language-saml-v20) | 管理 API はシステムのアプリケーション設計に従い、プロトコルの形は標準仕様が定める |
| データ設計 | 委譲 | [モデル](#モデル)、[SAML の SSO の設計](features/sso.md#データ) | Aggregate とその関係は仕様のモデルが、AuthnRequest の再送の記録はSAML の SSO の設計が扱う |
| セキュリティ設計 | 固有設計 | [担当文書](security.md#テナントとプロファイルの単位の署名) | テナントとプロファイルの単位の署名、XML の処理 |
| 信頼性設計 | 委譲 | [SAML の SSO の設計](features/sso.md#信頼性) | 再送の記録先の障害の扱いはSAML の SSO の設計が扱う |
| 検証設計 | 固有設計 | [担当文書](verification.md#パースの境界のファズテスト) | パースの境界のファズテスト、実体参照の回帰テスト |

| 文書 | 内容 |
| --- | --- |
| [Saml の用語集](glossary.md) | このモジュールでの語義 |
| [Saml の標準仕様](standards.md) | 採用する外部標準仕様 |
| [設計判断](decisions.md) | 代替案を比べた重要な判断 |
