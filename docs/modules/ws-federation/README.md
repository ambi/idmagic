# WsFederation

## 責務と境界

受動的な WS-Federation と能動的な WS-Trust の STS について、RP の信頼関係、AD FS 互換の `federationmetadata.xml`、MEX、RST と RSTR を扱う。

| 扱わないもの | 担当 |
| --- | --- |
| プロトコルに依存しないクレームの発行 | `ClaimMapping` |
| 署名鍵のライフサイクル | `SigningKeys` |
| SAML 2.0 の SP との信頼関係 | `SAML` |
| ブラウザーのログインセッション、第二要素の照合の手順、UsernameToken の資格情報の検証の部品 | `Authentication` |
| Application への割り当て | `Application` |

## 主な流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| パッシブサインイン | ブラウザーの `/wsfed` への `wsignin1.0` | `api` が、ログインセッションがなければログインの画面へ、Entra が求めた MFA を満たさなければ第二要素の画面へ誘導し、満たせば検証して RSTR のフォームを返す | [パッシブサインイン](features/passive-sign-in.md) |
| 能動的な STS | クライアントの `/trust/usernamemixed` への SOAP の要求 | `api` が、エンベロープの解析、`MessageID` の記録、資格情報の検証、発行を同期的に行う | [WS-Trust の能動的 STS](features/active-sts.md) |
| メタデータと MEX | RP と Entra の取得 | `api` が、要求のたびにテナントの証明書から組み立てる | [フェデレーションメタデータ](#フェデレーションメタデータ) |
| 管理 API | 解決済みのテナントの `/api/admin/v1/wsfed/...` への要求 | `api` が、ハンドラーから同期的に保存する | [RP と Entra フェデレーションの管理](features/relying-party.md) |

### フェデレーションメタデータ

WsFederation が公開する `federationmetadata.xml` と MEX は、パッシブサインインと能動的な STS の両方の利用者が使う。

各テナントの `/{realm}/federationmetadata/2007-06/federationmetadata.xml` で AD FS 互換の `federationmetadata.xml` を公開し、テナントの発行者（デフォルトテナントでは `/realms/default`）を entityID として広告する。
WS-Fed の RP と Microsoft Entra のドメインフェデレーションは、別の導入手順を使わずに、発行者、エンドポイント、署名証明書を検出できる。

| 要素 | 広告する内容 |
| --- | --- |
| `RoleDescriptor` | `SecurityTokenServiceType` と `ApplicationServiceType` の二つ |
| エンドポイント | `PassiveRequestorEndpoint`、`SecurityTokenServiceEndpoint`、`MetadataEndpoint` |
| `KeyDescriptor` | 役割ごとに、現在有効な署名証明書と、有効期限内の検証用の証明書 |

署名証明書は、OAuth と OIDC の JWK の形を再利用せず、WS-* が本来使う X.509 の形で公開する。
鍵の用途、ローテーション、重なりは `SigningKeys` の責務のままであり、メタデータは WS-* の利用者がすでに期待するものを広告すれば足りるからである。
有効期限内の検証用の証明書も広告するので、RP は計画されたローテーションの前後でも検証を続けられる。

`/{realm}/trust/mex` は、能動的な STS の探索として、`usernamemixed` のエンドポイントと、UsernameToken を必須とするその方針を公開する。
MEX は WSDL の方針、束縛、ポートで構成し、MSAL のようなクライアントが方針から束縛、束縛からポートをたどってエンドポイントを選べる形にする。

| 要素 | 内容 |
| --- | --- |
| 方針（`wsp:Policy`） | TLS の `TransportBinding` と、WS-SecurityPolicy 2005/07 の `SignedSupportingTokens` の `UsernameToken`（`WssUsernameToken10`） |
| 束縛（`wsdl:binding`） | 方針を `wsp:PolicyReference` で参照し、SOAP 1.2 の HTTP の転送と、WS-Trust 1.3 の Issue の `soapAction`（`.../200512/RST/Issue`）を持つ |
| ポート（`wsdl:port`） | 束縛を参照し、`soap12:address` と `wsa:EndpointReference` に `usernamemixed` の URL を持つ |
RST と RSTR のやり取り自体は、メタデータの一部ではない。

## 設計の要点

| 方針 | 理由 |
| --- | --- |
| 準拠を価値のすべてとし、独自の語彙を足さない | 足せば相互運用を損なう |
| 能動的な対応を狭く保つ | 再送と XML の包み替えへの攻撃面を広げない。詳細は[判断](decisions.md#ws-trust-の能動的な対応を-usernamemixed-の-issue-だけに絞る) |
| MessageID の再送防止は要求側のポートで受ける | `ports.MessageIDReplayStore` により OAuth のクライアント認証の契約から分ける。組み立て地点は既存のクライアント assertion 用ストアを直接渡し、同じ保存先を使う。`wstrust:` のキーの名前空間と有効期間は WsFederation が決める |
| メタデータの公開とクレームの対応付けの担当を分ける | クレームの公開ポリシーを、WS-Fed、WS-Trust、SAML で共有する。詳細は[判断](decisions.md#フェデレーションメタデータの公開とクレームの対応付けの担当を分ける) |

## 機能一覧

| 機能 | 内容 |
| --- | --- |
| [RP と Entra フェデレーションの管理](features/relying-party.md) | RP の登録、参照、削除と、Entra のドメインフェデレーションの定型設定 |
| [パッシブサインイン](features/passive-sign-in.md) | `wsignin1.0` によるトークンの発行と、`wsignout1.0`、`wsignoutcleanup1.0` によるサインアウト |
| [WS-Trust の能動的 STS](features/active-sts.md) | `/trust/usernamemixed` の `Issue` |

## 文書

| 設計領域 | 文書 | 内容 |
| --- | --- | --- |
| アプリケーション設計 | [標準仕様](standards.md#web-services-federation-language-ws-federation-version-12) | 管理 API はシステムのアプリケーション設計に従い、プロトコルの形は標準仕様が定める |
| データ設計 | [WsFederation のデータ](data.md#集約) | Aggregate とその関係。テーブルの定義はスキーマファイルが扱う |
| セキュリティ設計 | [WsFederation のセキュリティ](security.md#テナント単位の署名) | テナント単位の署名、認可の境界 |
| 信頼性設計 | [RP と Entra フェデレーションの管理の設計](features/relying-party.md#信頼性) | Entra のフェデレーションの障害の切り分けはRP と Entra フェデレーションの管理の設計が扱い、それ以外に固有の仕組みはない |
| 検証設計 | [WsFederation の検証](verification.md#パースの境界のファズテスト) | パースの境界のファズテスト |

| 文書 | 内容 |
| --- | --- |
| [WsFederation の用語集](glossary.md) | このモジュールでの語義 |
| [WsFederation の標準仕様](standards.md) | 標準と Microsoft の製品がどう組み合わさって相互運用が成り立つかのまとめと、採用する外部標準仕様と、Microsoft Entra、Microsoft Graph、MSAL.NET との相互運用で守る要件（出典と参照日付き） |
| [設計判断](decisions.md) | 代替案を比べた重要な判断 |
