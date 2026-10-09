# WsFederation のアーキテクチャ

この文書は、WsFederation の文脈、解決戦略、構成要素、実行時の流れを扱う。
システム全体の構成は[論理アーキテクチャ](../../../design/architecture/logical.md)と[ランタイムアーキテクチャ](../../../design/architecture/runtime.md)に従う。

## 文脈と範囲

このモジュールがほかのモジュールと結ぶ契約は、仕様の[公開する契約](../README.md#公開する契約)が定める。
このモジュールから外への依存の向きは次のとおりである。

| 相手 | 向き | 実現方式 |
| --- | --- | --- |
| `ClaimMapping` | 発行が、RP のポリシーでクレームを組み立てる | 相手の `usecases` の `IssueClaimsWithFloor` と属性の解決を呼ぶ |
| `SigningKeys` | 発行とメタデータが、テナントの XML 署名の資格情報を読む | `tokens_saml` の署名者が、相手の資格情報を発行のたびに得る |
| `Authentication` | パッシブサインインがログインセッションを読み、能動的な STS が UsernameToken を検証する | 相手のセッションとパスワードの部品、ログインの試行の制限を使う |
| `Application` | パッシブサインインが、対象のユーザーの割り当てを確かめる | 相手の割り当ての読み取りを使う |
| `OAuth2` | 能動的な STS が `MessageID` の再利用を記録する | 相手のクライアントアサーションのリプレイ防止ストアを共有する |

## 解決戦略

| 方針 | 理由 |
| --- | --- |
| 準拠を価値のすべてとし、独自の語彙を足さない | 足せば相互運用を損なう |
| 能動的な対応を狭く保つ | 再送と XML の包み替えへの攻撃面を広げない。詳細は[判断](decisions.md#ws-trust-の能動的な対応を-usernamemixed-の-issue-だけに絞る) |
| メタデータの公開とクレームの対応付けの担当を分ける | クレームの公開ポリシーを、WS-Fed、WS-Trust、SAML で共有する。詳細は[判断](decisions.md#フェデレーションメタデータの公開とクレームの対応付けの担当を分ける) |

## 構成要素

コードは機能スライスを持たず、一つの層の構成である。

| 機能仕様 | 主なユースケースとハンドラー |
| --- | --- |
| [RP と Entra フェデレーションの管理](../relying-party/README.md) | `handlers_http/admin_relying_party_handler.go`、`handlers_http/admin_entra_handler.go`、`domain/entra.go` |
| [パッシブサインイン](../passive-sign-in/README.md) | `usecases/signin.go`、`usecases/signout.go`、`handlers_http/wsfed_handler.go` |
| [WS-Trust の能動的 STS](../active-sts/README.md) | `usecases/wstrust.go`、`handlers_http/wstrust_handler.go`、`requests_wstrust` |
| [フェデレーションメタデータ](federation-metadata.md) | `handlers_http/metadata_handler.go`、`metadata_wsfederation` |

| 層 | 責務 |
| --- | --- |
| `domain` | `WsFedRelyingParty`、`EntraFederationProfile`、ImmutableID の正規化、ドメインイベント |
| `ports` | `RelyingPartyRepository` |
| `usecases` | サインイン、サインアウト、`Issue` の判定 |
| `requests_wstrust` | RST の SOAP エンベロープの解析と RSTR の組み立て |
| `responses_wsfederation` | パッシブサインインの RSTR のフォーム |
| `tokens_saml` | SAML 1.1 と SAML 2.0 の Assertion の組み立てと XML 署名。`SAML` も使う |
| `metadata_wsfederation` | `federationmetadata.xml` と MEX の組み立て |
| `handlers_http` | プロトコルのエンドポイントと、テナント単位の管理 API |
| `db_postgres`、`db_memory` | `ports` の PostgreSQL の実装と、テストとローカルの構成で使うメモリの実装 |
| `testing_contract` | 二つの実装が同じ契約を満たすことを確かめる共通のテスト |

## 実行時の流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| パッシブサインイン | ブラウザーの `/wsfed` への `wsignin1.0` | `api` が、ログインセッションがなければログインの画面へ誘導し、あれば検証して RSTR のフォームを返す | [パッシブサインイン](../passive-sign-in/README.md) |
| 能動的な STS | クライアントの `/trust/usernamemixed` への SOAP の要求 | `api` が、エンベロープの解析、`MessageID` の記録、資格情報の検証、発行を同期的に行う | [WS-Trust の能動的 STS](../active-sts/README.md) |
| メタデータと MEX | RP と Entra の取得 | `api` が、要求のたびにテナントの証明書から組み立てる | [フェデレーションメタデータ](federation-metadata.md) |
| 管理 API | 解決済みのテナントの `/api/admin/v1/wsfed/...` への要求 | `api` が、ハンドラーから同期的に保存する | [RP と Entra フェデレーションの管理](../relying-party/README.md) |
