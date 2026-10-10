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
| `authorize()`（AuthZEN） | 判断の合成を求めるモジュール（`Authorization`） | このモジュールが提供する | 規則表が要件の論理積を評価する |
| 失効エポックの参照 | `SharedSignals` が提供する | このモジュールが使う | イントロスペクションで Agent の失効を反映する |
| ドメインイベント | 監査と下流が購読する | このモジュールが発行する | 認可、トークン、同意、承認、ログアウトの各イベント |

## 機能

| 機能 | 内容 |
| --- | --- |
| [クライアント](client/README.md) | 登録、メタデータの解決、クライアントの認証、クライアントシークレット |
| [認可](authorization/README.md) | 認可コードフロー、PKCE、PAR |
| [同意](consent/README.md) | 同意の付与、参照、撤回 |
| [デバイス認可](device/README.md) | Device Authorization Grant |
| [トークン](token/README.md) | 発行、ローテーション、イントロスペクション、失効、UserInfo、送信者制約、委譲 |
| [承認リクエスト](approval/README.md) | Agent の操作に対する人間の承認（CIBA） |
| [ログアウト](logout/README.md) | RP-Initiated Logout とバックチャネルのログアウト |
| [プロトコルのエンドポイント](protocol-endpoints/README.md) | Discovery Metadata と流量の制限 |
| [管理 API の認可](admin-access/README.md) | 管理 API のスコープとロールポリシー |

| 文書 | 内容 |
| --- | --- |
| [OAuth2 の用語集](glossary.md) | このモジュールでの語義 |
| [OAuth2 の標準仕様](standards.md) | 採用する外部標準仕様 |
| [OAuth2 の設計](design/README.md) | 設計領域ごとの設計と重要な判断 |
