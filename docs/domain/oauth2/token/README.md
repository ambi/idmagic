# トークン

## 概要

この文書は、トークンの発行、ローテーション、検証、失効、送信者制約、委譲の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | アクセストークン、リフレッシュトークン、ID トークンの発行、リフレッシュトークンのローテーション、イントロスペクション、失効、UserInfo、DPoP と mTLS の送信者制約、`client_credentials`、Token Exchange による委譲、保護されたリソースの認証エラー |
| 行為者 | 登録済みのクライアント、Agent、リソースサーバー |
| 扱わないもの | 認可コードの発行は[認可](../authorization/README.md)が、承認による発行は[承認リクエスト](../approval/README.md)が、署名鍵は `SigningKeys` が、トークンに載せるクレームは `ClaimMapping` が扱う |

## モデル

| 判断 | 内容 |
| --- | --- |
| トークンの形式 | アクセストークンは自己完結した JWT、リフレッシュトークンはデータベースに保存する不透明な参照をデフォルトとする |
| リフレッシュトークンのローテーション | 交換のたびにローテーションし、ローテーション済みのトークンが提示されたら、公開クライアントと confidential クライアントのどちらでもファミリー全体を失効させる |
| 送信者制約 | DPoP をデフォルトとし、クライアントの PKI を運用するクライアントには mTLS も選択肢として提供する |
| Agent | アイデンティティとライフサイクルを持つ第一級のプリンシパルだが、独自の資格情報は持たず、既存の `OAuth2Client` の登録に関連付ける |
| 代理の行為 | OAuth 2.0 Token Exchange として実装し、偽装ではなく、元の `sub` と `act` の Agent を保つ委任をデフォルトとする |
| Agent の権限 | 粗いスコープではなく RFC 9396 の `authorization_details` で表し、送金の上限などの制約を宣言する。後続の Token Exchange では権限を狭めることだけを許す |

- **判断**：トークンの形式の理由は、[アクセストークンを JWT にしリフレッシュトークンを不透明な参照にする](../design/decisions.md#アクセストークンを-jwt-にしリフレッシュトークンを不透明な参照にする)。
- **判断**：委譲の方式の理由は、[ユーザーの代理の行為を Token Exchange による委任で表す](../design/decisions.md#ユーザーの代理の行為を-token-exchange-による委任で表す)。

## 状態遷移

### RefreshTokenLifecycle

RefreshToken のライフサイクル。Rotate で子トークンに引き継がれ、Revoke で失効、Expire で期限切れ。Rotated 後も家族失効により Revoked へ遷移しうる（RFC 9700 §4.14）。

| State | Kind | Meaning |
|---|---|---|
| Active | initial | 提示するとローテーションしてトークンを再発行できる |
| Rotated | — | 子トークンへ引き継いだ。再提示は再利用として family 全体の失効を招く |
| Revoked | terminal | 利用者の操作、ログアウト、または family 失効で無効にした |
| Expired | terminal | スライディング期限または絶対期限に達した |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | Rotate | now() < absolute_expires_at | Rotated |  |
| Active | RevokeToken | — | Revoked |  |
| Active | Expire | — | Expired |  |
| Rotated | RevokeToken | — | Revoked |  |
| Rotated | Expire | — | Expired |  |

## 操作

### クライアントによるトークンの発行

#### REQ-OAUTH2-001 ユーザーに紐づく OAuth グラントは account スコープのアクセストークンを発行できる

#### REQ-OAUTH2-021 リフレッシュトークンは `offline_access` スコープを付与したときだけ発行する

#### REQ-OAUTH2-026 client_credentials グラントで M2M トークンが発行される

#### REQ-OAUTH2-010 DPoP 証明付き要求はセンダー制約付きトークンを発行する

#### REQ-OAUTH2-046 所有者がオフボードされた Agent は client_credentials で新しいトークンを取得できない

#### REQ-OAUTH2-034 トークンエンドポイントはテナント境界を越えた資格情報を受理しない

#### REQ-OAUTH2-039 KeyProvider の障害時は新しいトークンの発行を拒否する

### クライアントによるリフレッシュトークンのローテーション

#### REQ-OAUTH2-006 リフレッシュトークンをローテーションして新しいトークンを得る

#### REQ-OAUTH2-018 絶対有効期限を過ぎたリフレッシュトークンはローテーションできない

### クライアントによるトークンの交換

#### REQ-OAUTH2-048 テナントが定めた委譲深さの上限を超えるトークン交換は拒否される

### リソースサーバーによるイントロスペクション

#### REQ-OAUTH2-011 失効済みトークンのイントロスペクションは `active=false` だけを返す

#### REQ-OAUTH2-012 キルスイッチ作動後の Agent トークンはイントロスペクションで active=false になる

#### REQ-OAUTH2-049 イントロスペクションと監査は同じ規則で委譲モードを示す

### クライアントによるトークンの失効

#### REQ-OAUTH2-019 クライアントは自分のトークンを失効できる

### クライアントによる UserInfo の取得

#### REQ-OAUTH2-013 UserInfo は openid スコープのトークンに sub を返す

#### REQ-OAUTH2-020 失効したアクセストークンによる UserInfo の取得は `invalid_token` で拒否される

### リソースサーバーによるトークンの検証

#### REQ-OAUTH2-029 mTLS バインド AT は同じ証明書のリクエストでのみ受理される

#### REQ-OAUTH2-045 保護リソースの DPoP Proof は ath でアクセストークンに結び付けられる

#### REQ-OAUTH2-044 Bearer 保護リソースの認証エラーはメタデータ URL を提示する

#### REQ-OAUTH2-047 管理コンソールとアカウントポータルの Bearer 認証も失効判定を通る

## セキュリティ上の考慮

到達できる範囲は、トークンのスコープが決める。
`account:*` を含むユーザーに紐づくスコープは、User の subject を持たないグラント（`client_credentials` や、subject を伴わない Token Exchange）では発行しない。

代行（Token Exchange）は権限を広げない。
`act` チェーンの上のすべての actor が有効であり、要求するスコープと `authorization_details` が元の権限の部分集合であることを求める。
チェーンの深さはテナントの `max_delegation_depth`（システムのデフォルトは 3）を超えられず、上書きを解決できない場合は交換を拒否する。
