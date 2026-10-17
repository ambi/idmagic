# OAuth2

## 責務と境界

OAuth 2.0 と OIDC のプロトコル群の全責務を扱う。
クライアントのメタデータと Dynamic Client Registration、認可の判断（認可、同意、認可コード、PAR、Device Authorization、RP-Initiated Logout）、トークンの発行とライフサイクル（アクセストークン、リフレッシュトークン、ID トークン、イントロスペクション、失効、UserInfo、Proof of Possession）、Discovery Metadata、Authorization Server Metadata、健全性の報告をこのモジュールに集約する。

| 扱わないもの | 担当 |
| --- | --- |
| トークンに載せるクレームの決定 | `ClaimMapping` |
| 署名鍵のライフサイクル | `SigningKeys` |
| 利用者の認証とセッションの保持 | `Authentication` |
| その `client_id` にそもそも到達してよいかという関門 | `Application` |
| Agent のアイデンティティとライフサイクル | `IdManagement` |
| Agent の失効エポック | `SharedSignals` |

このモジュールが受け持つのは、それらの結果をプロトコルの語彙で組み立てて外部へ返す部分である。

## モデル

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `OAuth2Client` | クライアントのメタデータ、認証方式、クライアントシークレットの資格情報、`ClaimMappingPolicy` | `Tenant` を `tenant_id` で参照する。`Application` から参照される |
| `Consent` | `(subject, client_id)` ごとの付与済みのスコープの集合 | `OAuth2Client` と User を参照する |
| `AuthorizationRequest`、`AuthorizationCodeRecord`、`PARRecord` | 認可の要求、認可コード、プッシュした要求 | `OAuth2Client` を参照する |
| `DeviceAuthorization` | デバイスコードとユーザーコード | `OAuth2Client` を参照する |
| `RefreshTokenRecord` | リフレッシュトークンとそのファミリー | `OAuth2Client` と User を参照する |
| `ApprovalRequest` | 人間の承認の判断と CIBA のポーリングの記録 | `OAuth2Client` と User を参照する |
| `McpResourceServer`、`AuthorizationDetailType` | MCP のリソースサーバーと、`authorization_details` の型 | `Tenant` を `tenant_id` で参照する |

`ClaimMappingPolicy` は `OAuth2Client` の内側の値であり、独立した Aggregate ではない。

## 公開する契約

HTTP の操作とモデルの形は TypeSpec の `OAuth 2.0 and OpenID Connect` のタグが、採用する標準の規則は[OAuth2 の標準仕様](standards.md)が定める。
次の表は、それ以外にほかのモジュールと結ぶ契約である。

| 契約 | 相手 | 向き | 内容 |
| --- | --- | --- | --- |
| `TokenIssuer`、`TokenIntrospector` | `ApiTokens` | このモジュールが提供する | RFC 9068 の JWT の発行と検証 |
| `WorkloadTokenVerifier` | `WorkloadIdentity` が実装する | このモジュールが定める | Token Exchange の `subject_token` の外部のアテステーションを検証する |
| `AccountConsents` | `Authentication` のアカウントポータル、`IdManagement` のデータエクスポート | このモジュールが提供する | 本人の `Granted` の同意を一覧し、本人の同意を撤回する |
| `ClientAdministration` | `Application` | このモジュールが提供する | Application が持つクライアントを作成、更新し、シークレットを交換、発行、失効させる |
| `SessionTokenRevoker` | `Authentication` が定める | このモジュールが実装する | ログインセッションの失効に合わせて、その `sid` のリフレッシュトークンを失効させる |
| `PendingAuthorizations` | `Authentication` が定める | このモジュールが実装する | 対話のログインが認証を終えたあと、認可要求のクライアントに応じたサインインの条件を答え、同意かコードの発行へ続ける |
| `SignInSteps` | `Authentication` が実装する | このモジュールが定める | 既存のセッションへの認可で、テナントのデフォルトのポリシーの適用と第二要素の要求を対話のログインへ委ねる |
| `LoginSessions` | `Authentication` が実装する | このモジュールが定める | `/end_session` がログインセッションを解決して終わらせる |
| `authorize()`（AuthZEN） | 判断の合成を求めるモジュール（`Authorization`） | このモジュールが提供する | 規則表が要件の論理積を評価する |
| 失効エポックの参照 | `SharedSignals` が提供する | このモジュールが使う | イントロスペクションで Agent の失効を反映する |
| ドメインイベント | 監査と下流が購読する | このモジュールが発行する | 認可、トークン、同意、承認、ログアウトの各イベント |

## 機能

| 機能 | 内容 |
| --- | --- |
| [クライアント](features/client.md) | 登録、メタデータの解決、クライアントの認証、クライアントシークレット |
| [認可](features/authorization.md) | 認可コードフロー、PKCE、PAR |
| [同意](features/consent.md) | 同意の付与、参照、撤回 |
| [デバイス認可](features/device.md) | Device Authorization Grant |
| [トークン](features/token.md) | 発行、ローテーション、イントロスペクション、失効、UserInfo、送信者制約、委譲 |
| [承認リクエスト](features/approval.md) | Agent の操作に対する人間の承認（CIBA） |
| [ログアウト](features/logout.md) | RP-Initiated Logout とバックチャネルのログアウト |
| [プロトコルのエンドポイント](features/protocol-endpoints.md) | Discovery Metadata と流量の制限 |
| [管理 API の認可](features/admin-access.md) | 管理 API のスコープとロールポリシー |

## 文書

このモジュールが担当する SLO とキャパシティ要件は、[品質要件の割り当て](../../design/architecture/logical.md#品質要件の割り当て)から担当機能、設計、検証へ参照する。

| 設計領域 | 区分 | 参照先 | 内容 |
| --- | --- | --- | --- |
| アーキテクチャ設計 | 固有設計 | [担当文書](architecture.md#文脈と範囲) | 文脈と範囲、解決戦略、構成要素、実行時の流れ |
| アプリケーション設計 | 委譲 | [標準仕様](standards.md#the-oauth-20-authorization-framework) | プロトコルの形は標準仕様と TypeSpec が、管理 API はシステムのアプリケーション設計が扱う |
| データ設計 | 委譲 | [トークンの設計](features/token.md#データ) | トークンの有効期間と保持期間はトークンの設計が、テーブルの定義はスキーマファイルが扱う |
| 信頼性設計 | 委譲 | [トークン](features/token.md#概要)、[ログアウトの設計](features/logout.md#アーキテクチャ) | 署名の提供元の障害はトークンが、ログアウトの配信の再試行はログアウトの設計が扱う |
| 検証設計 | 固有設計 | [担当文書](verification.md#パースの境界のファズテスト) | パースの境界のファズテスト |

| 文書 | 内容 |
| --- | --- |
| [OAuth2 の用語集](glossary.md) | このモジュールでの語義 |
| [OAuth2 の標準仕様](standards.md) | 採用する外部標準仕様 |
| [設計判断](decisions.md) | 代替案を比べた重要な判断 |
