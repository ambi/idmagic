# Saml のアーキテクチャ

この文書は、Saml の文脈、解決戦略、構成要素、実行時の流れを扱う。
システム全体の構成は[論理アーキテクチャ](../../../design/architecture/logical.md)と[ランタイムアーキテクチャ](../../../design/architecture/runtime.md)に従う。

## 文脈と範囲

このモジュールがほかのモジュールと結ぶ契約は、仕様の[公開する契約](../README.md#公開する契約)が定める。
このモジュールから外への依存の向きは次のとおりである。

| 相手 | 向き | 実現方式 |
| --- | --- | --- |
| `WS-Federation` | SSO が Assertion を組み立て、署名する | 相手の `tokens_saml` の構築器と署名器を再利用する |
| `ClaimMapping` | SSO が SP のポリシーでクレームを組み立てる | 相手の `IssueClaimsWithFloor` と属性の解決を呼ぶ |
| `SigningKeys` | 署名とメタデータが、テナントとプロファイルの資格情報を読む | 用途 `XmlFederationSigning` と、プロファイルの ID をスコープにして相手の `KeyStore` を使う |
| `Authentication` | SSO と SLO がログインセッションを読み、破棄する | 相手のセッションの部品を使う |
| `Application` | SSO が、対象のユーザーの割り当てを確かめる | 相手の割り当ての読み取りを使う |

## 解決戦略

| 方針 | 理由 |
| --- | --- |
| Web Browser SSO Profile に絞る | 署名ラッピングの攻撃への露出を小さくする。詳細は[判断](decisions.md#対応の範囲を-web-browser-sso-profile-に限る) |
| XML の処理と Assertion の組み立てを自作しない | 検証済みのライブラリと、WS-Fed の部品を使う。詳細は[判断](decisions.md#xml-の処理と-assertion-の組み立てを自作しない) |
| 相互運用の安全性の検証をドメインの層に集約する | 宛先、発行者、audience の検証を一か所で、フェイルクローズで行う |

## 構成要素

コードは機能スライスを持たず、一つの層の構成である。
機能仕様に対応するコードは、要件の ID を `//spec:covers` で名指すテストからたどる。

| 層 | 責務 |
| --- | --- |
| `domain` | `SamlServiceProvider`、`SamlIdentityProviderProfile`、AuthnRequest の検証、ドメインイベント |
| `ports` | SP とプロファイルの Repository、AuthnRequest の再送の記録 |
| `usecases` | サインインとログアウトの判定 |
| `responses_saml` | SAMLResponse とプロトコルのエラーの組み立て |
| `metadata_saml` | IdP のメタデータの組み立て |
| `handlers_http` | プロトコルのエンドポイントと、テナント単位の管理 API |
| `db_postgres`、`db_memory` | `ports` の PostgreSQL の実装と、テストとローカルの構成で使うメモリの実装 |
| `testing_contract` | 二つの実装が同じ契約を満たすことを確かめる共通のテスト |

## 実行時の流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| SSO | ブラウザーの `/saml/sso` または `/saml/idp/{profile_id}/sso` への要求 | `api` が、ログインセッションがなければログインの画面へ誘導し、あれば検証して SAMLResponse を ACS へ POST させる | [SAML の SSO の設計](../sso/design.md) |
| SLO | ブラウザーの `/saml/slo` への要求 | `api` がローカルセッションを破棄する | [SAML の SSO](../sso/README.md) |
| メタデータと証明書 | SP の取得 | `api` が、要求のたびにプロファイルの証明書から組み立てる | [IdP プロファイルとメタデータ](../idp-profile/README.md) |
| 管理 API | 解決済みのテナントの `/api/admin/v1/saml/...` への要求 | `api` が、ハンドラーから同期的に保存する | [サービスプロバイダーの管理](../service-provider/README.md) |
