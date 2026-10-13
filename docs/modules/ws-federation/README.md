# WsFederation

## 責務と境界

受動的な WS-Federation と能動的な WS-Trust の STS について、RP の信頼関係、AD FS 互換の `federationmetadata.xml`、MEX、RST と RSTR を扱う。

| 扱わないもの | 担当 |
| --- | --- |
| プロトコルに依存しないクレームの発行 | `ClaimMapping` |
| 署名鍵のライフサイクル | `SigningKeys` |
| SAML 2.0 の SP との信頼関係 | `SAML` |
| ブラウザーのログインセッションと UsernameToken の資格情報の検証の部品 | `Authentication` |
| Application への割り当て | `Application` |

## モデル

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `WsFedRelyingParty` | `wtrealm`、許可する `wreply` の集合、audience、トークンの型（SAML 1.1 または SAML 2.0）、`ClaimMappingPolicy`、Entra のフェデレーションの定型設定 | `Tenant` を `tenant_id` で参照する。`Application` から参照される |

RP のトークンの型は、指定がなければ SAML 1.1 とする。
Entra と AD FS の WS-Fed のデフォルトに合わせるためである。

## 公開する契約

HTTP の操作とモデルの形は TypeSpec の `WS-Federation` のタグが定める。
プロトコルのエンドポイント（`/wsfed`、`federationmetadata.xml`、`/trust/mex`、`/trust/usernamemixed`）は、[WsFederation の標準仕様](standards.md)が定める範囲で動く。
次の表は、それ以外にほかのモジュールと結ぶ契約である。

| 契約 | 相手 | 向き | 内容 |
| --- | --- | --- | --- |
| `RelyingPartyRepository` | `Application` のプロトコル設定 | このモジュールが提供する | アプリケーションに属する RP の設定を読み書きする |
| `tokens_saml` の Assertion の組み立てと署名 | `SAML` の SSO | このモジュールが提供する | SAML 1.1 と SAML 2.0 の Assertion を組み立て、XML 署名を付ける |
| `IssueClaimsWithFloor` | `ClaimMapping` が提供する | このモジュールが使う | RP のポリシーでクレームを発行する |
| XML 署名の資格情報 | `SigningKeys` が提供する | このモジュールが使う | テナントの `XmlFederationSigning` の鍵と証明書を発行のたびに得る |
| ドメインイベント | 監査と下流が購読する | このモジュールが発行する | `WsFedSignInIssued`、`WsFedSignInRejected`、`WsTrustTokenIssued`、`WsTrustTokenRejected` |

## 機能

| 機能 | 内容 |
| --- | --- |
| [RP と Entra フェデレーションの管理](relying-party/README.md) | RP の登録、参照、削除と、Entra のドメインフェデレーションの定型設定 |
| [パッシブサインイン](passive-sign-in/README.md) | `wsignin1.0` によるトークンの発行と、`wsignout1.0`、`wsignoutcleanup1.0` によるサインアウト |
| [WS-Trust の能動的 STS](active-sts/README.md) | `/trust/usernamemixed` の `Issue` |

| 文書 | 内容 |
| --- | --- |
| [WsFederation の用語集](glossary.md) | このモジュールでの語義 |
| [WsFederation の標準仕様](standards.md) | 採用する外部標準仕様 |

## 設計

| 設計領域 | 区分 | 参照先 | 内容 |
| --- | --- | --- | --- |
| アーキテクチャ設計 | 固有設計 | [担当文書](design/architecture.md#文脈と範囲) | 文脈と範囲、解決戦略、構成要素、実行時の流れ |
| アプリケーション設計 | 委譲 | [標準仕様](standards.md#web-services-federation-language-ws-federation-version-12) | 管理 API はシステムのアプリケーション設計に従い、プロトコルの形は標準仕様が定める |
| データ設計 | 委譲 | [モデル](#モデル) | RP の項目は仕様のモデルが、テーブルの定義はスキーマファイルが扱う |
| セキュリティ設計 | 固有設計 | [担当文書](design/security.md#テナント単位の署名) | テナント単位の署名、認可の境界 |
| 信頼性設計 | 委譲 | [RP と Entra フェデレーションの管理の設計](relying-party/README.md#信頼性) | Entra のフェデレーションの障害の切り分けはRP と Entra フェデレーションの管理の設計が扱い、それ以外に固有の仕組みはない |
| 検証設計 | 固有設計 | [担当文書](design/verification.md#パースの境界のファズテスト) | パースの境界のファズテスト |

| 文書 | 内容 |
| --- | --- |
| [設計判断](design/decisions.md) | 代替案を比べた重要な判断 |
| [フェデレーションメタデータ](design/federation-metadata.md) | `federationmetadata.xml` と MEX が広告する内容と、署名証明書の公開 |
