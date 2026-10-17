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

## 主な流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| SSO | ブラウザーの `/saml/sso` または `/saml/idp/{profile_id}/sso` への要求 | `api` が、ログインセッションがなければログインの画面へ誘導し、あれば検証して SAMLResponse を ACS へ POST させる | [SAML の SSO の設計](features/sso.md#実現方式) |
| SLO | ブラウザーの `/saml/slo` への要求 | `api` がローカルセッションを破棄する | [SAML の SSO](features/sso.md) |
| メタデータと証明書 | SP の取得 | `api` が、要求のたびにプロファイルの証明書から組み立てる | [IdP プロファイルとメタデータ](features/idp-profile.md) |
| 管理 API | 解決済みのテナントの `/api/admin/v1/saml/...` への要求 | `api` が、ハンドラーから同期的に保存する | [サービスプロバイダーの管理](features/service-provider.md) |

## 設計の要点

| 方針 | 理由 |
| --- | --- |
| Web Browser SSO Profile に絞る | 署名ラッピングの攻撃への露出を小さくする。詳細は[判断](decisions.md#対応の範囲を-web-browser-sso-profile-に限る) |
| XML の処理と Assertion の組み立てを自作しない | 検証済みのライブラリと、WS-Fed の部品を使う。詳細は[判断](decisions.md#xml-の処理と-assertion-の組み立てを自作しない) |
| 相互運用の安全性の検証をドメインの層に集約する | 宛先、発行者、audience の検証を一か所で、フェイルクローズで行う |

## 機能一覧

| 機能 | 内容 |
| --- | --- |
| [IdP プロファイルとメタデータ](features/idp-profile.md) | IdP プロファイルの管理と、メタデータと署名証明書の公開 |
| [サービスプロバイダーの管理](features/service-provider.md) | SP の登録、参照、削除 |
| [SAML の SSO](features/sso.md) | SP 起点と IdP 起点の SSO と Single Logout |

## 文書

| 設計領域 | 文書 | 内容 |
| --- | --- | --- |
| アプリケーション設計 | [標準仕様](standards.md#assertions-and-protocols-for-the-oasis-security-assertion-markup-language-saml-v20) | 管理 API はシステムのアプリケーション設計に従い、プロトコルの形は標準仕様が定める |
| データ設計 | [Saml のデータ](data.md#集約)、[SAML の SSO の設計](features/sso.md#データ) | Aggregate とその関係。AuthnRequest の再送の記録はSAML の SSO の設計が扱う |
| セキュリティ設計 | [Saml のセキュリティ](security.md#テナントとプロファイルの単位の署名) | テナントとプロファイルの単位の署名、XML の処理 |
| 信頼性設計 | [SAML の SSO の設計](features/sso.md#信頼性) | 再送の記録先の障害の扱いはSAML の SSO の設計が扱う |
| 検証設計 | [Saml の検証](verification.md#パースの境界のファズテスト) | パースの境界のファズテスト、実体参照の回帰テスト |

| 文書 | 内容 |
| --- | --- |
| [Saml の用語集](glossary.md) | このモジュールでの語義 |
| [Saml の標準仕様](standards.md) | 採用する外部標準仕様 |
| [設計判断](decisions.md) | 代替案を比べた重要な判断 |
