# ApiTokens

## 責務と境界

管理 API と SCIM API の認証に使う、テナント単位の API アクセストークン（`idmagic_pat_` 接頭辞）の発行、失効、一覧と、スコープの語彙を扱う。
トークンに付与したスコープの集合は認証の時点で解決し、記録の正を持つ各 Context が操作の認可に使う。

| 扱わないもの | 担当 |
| --- | --- |
| SCIM を含む各 API のエンドポイント | 記録の正を持つ各 Context |
| 操作ごとに必要なスコープの割り当て | TypeSpec の `x-api-token-scopes` と、各 Context の管理 API |
| JWT の発行、署名、イントロスペクション、`/revoke` | `OAuth2` のトークンの部品 |

この Context が提供するのは、トークンとスコープの語彙という横断的な認証の基盤だけである。

## モデル

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `ApiToken` | `jti`、発行者、説明、スコープの集合、audience、DPoP の束縛、発行時刻、有効期限、失効時刻 | `Tenant` を `tenant_id` で、発行者の User を `user_id` で参照する |

`ApiTokenScope` は `<resource>:<action>` の形で、`read` は参照、`write` は変更の操作を許可する。
`scim:` で始まるスコープは、SCIM 2.0 のプロビジョニング API のリソースと操作を表す。
スコープの語彙は TypeSpec の `ApiTokenScope` が定める。

- **判断**：発行時に一度だけ返す JWT は、どこにも保存しない。永続化するのは `jti` とライフサイクルのメタデータだけである。したがって、発行済みのトークンの一覧から本文を取り出す経路はなく、紛失への手段は再発行だけになる。

## 公開する契約

HTTP の操作とモデルの形は TypeSpec の `API Tokens` のタグが定める。
次の表は、それ以外にほかの Context と結ぶ契約である。

| 契約 | 相手 | 向き | 内容 |
| --- | --- | --- | --- |
| `Authenticate` と `ApiTokenPrincipal` | 管理 API と SCIM API の認証の部品 | この Context が提供する | 提示された JWT を検証し、テナント、発行者、組み込みの `client_id`、スコープの集合を返す |
| `IntrospectAccessToken`、`RevokeByJTI` | `OAuth2` のイントロスペクションと `/revoke` | この Context が提供する | 管理発行のトークンの状態を返し、失効させる |
| `TokenIssuer`、`TokenIntrospector` | `OAuth2` が提供する | この Context が使う | RFC 9068 の JWT を発行し、署名と期限を検証する |

## 機能

| 機能 | 内容 |
| --- | --- |
| [API アクセストークンの発行と失効](token-management/README.md) | 管理画面での構成、発行、一覧、失効 |
| [API アクセストークンの認証と認可](authentication/README.md) | 提示されたトークンの認証と、管理 API の粒度スコープによる認可 |

| 文書 | 内容 |
| --- | --- |
| [ApiTokens の用語集](glossary.md) | この Context での語義 |
| [ApiTokens の標準仕様](standards.md) | 採用する外部標準仕様 |
| [ApiTokens の設計](design/README.md) | 話題ごとの設計と重要な判断 |
