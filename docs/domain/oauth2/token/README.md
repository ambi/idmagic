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

| State | リフレッシュトークンのローテーション | トークンの失効 | 有効期間の経過 |
|---|---|---|---|
| Active | → Rotated（絶対期限内）<br>拒否：400 invalid_grant（絶対期限切れ） | → Revoked | → Expired |
| Rotated | → Revoked（再利用としてファミリー全体を失効させ、400 invalid_grant を返す） | → Revoked | → Expired |
| Revoked | 拒否：400 invalid_grant | 何もしない | 何もしない |
| Expired | 拒否：400 invalid_grant | 何もしない | 何もしない |

## 操作

### クライアントによるトークンの発行

#### REQ-OAUTH2-001 ユーザーに紐づく OAuth グラントは account スコープのアクセストークンを発行できる

- クライアントに登録し User が同意した `account:*` のスコープを、認可コードかデバイス認可のグラントで交換したとき、OAuth2 は、`sub` を同意した User、audience をレルムの IdMagic API の発行者の識別子にしたアクセストークンを発行する。
- account のリソースサーバーが `account:read` のトークンを受けたとき、OAuth2 は、トークンの subject 本人の参照の操作だけを許可する。
- `client_credentials` か User の subject を持たない Token Exchange で account のスコープを要求された場合、OAuth2 は、400 と `invalid_scope` で拒否する。
- クライアントの許可のスコープか User の同意に account のスコープが含まれない場合、OAuth2 は、account のスコープを発行しない。
- audience がリクエスト先のレルムの IdMagic API を含まない account のスコープのトークンを account のリソースサーバーへ提示された場合、OAuth2 は、401 と `invalid_token` で拒否する。
- ポータルの境界のスコープだけを持ち account のスコープを持たないトークンを受けたとき、OAuth2 は、audience の検査をしない。
- **例**：EX-OAUTH2-001-01、EX-OAUTH2-001-02、EX-OAUTH2-001-03、EX-OAUTH2-001-04

#### REQ-OAUTH2-021 リフレッシュトークンは `offline_access` スコープを付与したときだけ発行する

- `offline_access` を含むスコープの認可コードを交換したとき、OAuth2 は、リフレッシュトークンを返し、`RefreshTokenIssued` を発行する。
- `offline_access` を含まないスコープの認可コードを交換したとき、OAuth2 は、リフレッシュトークンを返さない。
- **例**：EX-OAUTH2-021-01、EX-OAUTH2-021-02

#### REQ-OAUTH2-026 client_credentials グラントで M2M トークンが発行される

- `client_credentials` を登録した confidential のクライアントが許可したスコープのトークンを要求したとき、OAuth2 は、`sub` を `client_id` にしたアクセストークンを、リフレッシュトークンなしで返し、`AccessTokenIssued` を発行する。
- 公開のクライアントが `client_credentials` でトークンを要求した場合、OAuth2 は、400 と `unauthorized_client` で拒否し、トークンを発行しない。
- **例**：EX-OAUTH2-026-01

#### REQ-OAUTH2-010 DPoP 証明付き要求はセンダー制約付きトークンを発行する

- 有効な DPoP 証明を付けてトークンを要求されたとき、OAuth2 は、アクセストークンの `cnf` に DPoP の鍵のサムプリントを結び付け、`token_type` を `DPoP` にし、リフレッシュトークンのセンダー制約を `Dpop` にする。
- `iat` が 60 秒以上古いか、同じ `jti` を再使用した DPoP 証明を受けた場合、OAuth2 は、400 と `invalid_dpop_proof` で拒否する。
- **例**：EX-OAUTH2-010-01、EX-OAUTH2-010-02、EX-OAUTH2-010-03

#### REQ-OAUTH2-046 所有者がオフボードされた Agent は client_credentials で新しいトークンを取得できない

- `Active` な Agent に束縛したクライアントが `client_credentials` でトークンを要求したとき、OAuth2 は、発行のたびに Agent の所有者の User を解決し、`Active` の所有者の Agent にだけトークンを発行する。
- 所有者が `Active` でないか解決できない Agent のクライアントの要求を受けた場合、OAuth2 は、401 と `invalid_client` で拒否し、トークンを発行せず、Agent の `status` を書き換えない。
- 束縛した Agent のないクライアントの要求を受けたとき、OAuth2 は、所有者を解決せずに要求を処理する。
- **例**：EX-OAUTH2-046-01、EX-OAUTH2-046-02、EX-OAUTH2-046-03、EX-OAUTH2-046-04

#### REQ-OAUTH2-034 トークンエンドポイントはテナント境界を越えた資格情報を受理しない

- 別のテナントで発行した認可コード、リフレッシュトークン、`device_code` を受けた場合、OAuth2 は、400 と `invalid_grant` で拒否する。
- 別のテナントに登録した `client_id` のクライアント認証を受けた場合、OAuth2 は、401 と `invalid_client` で拒否する。
- テナントの異なるリフレッシュトークンとクライアントか `sub` を永続化しようとした場合、OAuth2 は、参照整合性のエラーで保存しない。
- **例**：EX-OAUTH2-034-01、EX-OAUTH2-034-02、EX-OAUTH2-034-03、EX-OAUTH2-034-04

#### REQ-OAUTH2-039 KeyProvider の障害時は新しいトークンの発行を拒否する

- テナントの KeyProvider へ到達できない場合、OAuth2 は、新しい署名をせずに 500 と `server_error` で拒否する。
- **例**：EX-OAUTH2-039-01

### クライアントによるリフレッシュトークンのローテーション

#### REQ-OAUTH2-006 リフレッシュトークンをローテーションして新しいトークンを得る

- 所有するクライアントが `Active` で絶対期限内のリフレッシュトークンを、必要な送信者制約の証明とともに交換したとき、OAuth2 は、そのトークンを `Rotated` にし、同じファミリーの新しいリフレッシュトークンとアクセストークンを返し、`RefreshTokenRotated` と `AccessTokenIssued` を発行する。
- `Rotated` のリフレッシュトークンを再び受けた場合、OAuth2 は、400 と `invalid_grant` で拒否し、ファミリーのトークンをすべて `Revoked` にし、`RefreshTokenReuseDetected` と `TokenRevoked` を発行する。
- 所有者でないクライアントのリフレッシュトークンを受けた場合、OAuth2 は、400 と `invalid_grant` で拒否し、ファミリーを失効させ、`RefreshTokenReuseDetected` を発行する。
- 並行のローテーションで後れた場合、OAuth2 は、400 と `invalid_grant` で拒否し、ファミリーを失効させる。
- User が存在しないか `Active` でないリフレッシュトークンを受けた場合、OAuth2 は、400 と `invalid_grant` で拒否し、ファミリーを失効させる。
- 送信者制約の証明が一致しないか、ポリシーが拒否したリフレッシュを受けた場合、OAuth2 は、400 と `invalid_grant` で拒否する。
- **例**：EX-OAUTH2-006-01、EX-OAUTH2-006-02

#### REQ-OAUTH2-018 絶対有効期限を過ぎたリフレッシュトークンはローテーションできない

- `absolute_expires_at` を過ぎたリフレッシュトークンを受けた場合、OAuth2 は、400 と `invalid_grant` で拒否し、トークンを発行しない。
- **例**：EX-OAUTH2-018-01

### クライアントによるトークンの交換

#### REQ-OAUTH2-048 テナントが定めた委譲深さの上限を超えるトークン交換は拒否される

- `act` の入れ子の深さがテナントの `max_delegation_depth`（上書きがなければシステムのデフォルトの 3）以内の Token Exchange を受けたとき、OAuth2 は、元の権限の部分集合のスコープと `authorization_details` で委任のトークンを発行し、発行したトークンの深さと適用した上限を載せた `TokenExchanged` を発行する。
- 深さが上限を超える Token Exchange を受けた場合、OAuth2 は、400 と `invalid_request` で拒否し、理由を載せた `TokenExchangeRejected` を発行する。
- テナントの委譲のポリシーを解決できない場合、OAuth2 は、システムのデフォルトへ退避せずに 400 と `invalid_request` で拒否する。
- `subject_token` のスコープを超えるスコープを要求された場合、OAuth2 は、400 と `invalid_scope` で拒否する。
- `subject_token` の `authorization_details` を超えるか検証を通らない `authorization_details` を要求された場合、OAuth2 は、400 と `invalid_authorization_details` で拒否する。
- 未対応の `subject_token_type` か `actor_token_type`、決められない現在の行為者を受けた場合、OAuth2 は、400 と `invalid_request` で拒否する。
- 無効か失効した `actor_token`、`may_act` に含まれない行為者、ポリシーが拒否した交換を受けた場合、OAuth2 は、400 と `invalid_grant` で拒否し、`TokenExchangeRejected` を発行する。
- **例**：EX-OAUTH2-048-01、EX-OAUTH2-048-02、EX-OAUTH2-048-03、EX-OAUTH2-048-04

### リソースサーバーによるイントロスペクション

#### REQ-OAUTH2-011 失効済みトークンのイントロスペクションは `active=false` だけを返す

- リソースサーバーが有効なトークンをイントロスペクトしたとき、OAuth2 は、`active=true` とクレームを返し、`TokenIntrospected` を発行する。
- 失効、ローテーション、期限切れのトークンか、未知のトークンをイントロスペクトされたとき、OAuth2 は、`active=false` だけを返し、ほかのフィールドを返さない。
- **例**：EX-OAUTH2-011-01

#### REQ-OAUTH2-012 キルスイッチ作動後の Agent トークンはイントロスペクションで active=false になる

- Agent の revocation epoch より前に発行したトークンをイントロスペクトされたとき、OAuth2 は、`active=false` だけを返す。
- Agent の revocation epoch より後に発行したトークンをイントロスペクトされたとき、OAuth2 は、`active=true` とクレームを返す。
- **例**：EX-OAUTH2-012-01、EX-OAUTH2-012-02

#### REQ-OAUTH2-049 イントロスペクションと監査は同じ規則で委譲モードを示す

- `active` なトークンをイントロスペクトされたとき、OAuth2 は、同じ交換が監査へ残したものと同じ規則で導いた `delegation_mode` を返す。
- `act` に subject と異なる行為者がいるとき、OAuth2 は、`delegation_mode` を `on_behalf_of` にする。
- 代行がなく subject が人間でないプリンシパルのとき、OAuth2 は、`delegation_mode` を `autonomous` にする。
- 代行がなく subject が人間の利用者のとき、OAuth2 は、`delegation_mode` を `direct` にする。
- **例**：EX-OAUTH2-049-01、EX-OAUTH2-049-02、EX-OAUTH2-049-03、EX-OAUTH2-049-04

### クライアントによるトークンの失効

#### REQ-OAUTH2-019 クライアントは自分のトークンを失効できる

- クライアントが自分のリフレッシュトークンかアクセストークンを失効させたとき、OAuth2 は、200 を返し、トークンを `Revoked` にし、`TokenRevoked` を発行する。
- 所有者でないクライアントか、未知か無効なトークンの失効を受けたとき、OAuth2 は、トークンを変えずに 200 を返す。
- **例**：EX-OAUTH2-019-01、EX-OAUTH2-019-02

### クライアントによる UserInfo の取得

#### REQ-OAUTH2-013 UserInfo は openid スコープのトークンに sub を返す

- `openid` のスコープを持つ有効なアクセストークンで UserInfo を取得されたとき、OAuth2 は、`sub` と、許可したスコープに対応するクレーム（`profile` には `name` と `preferred_username`）を返す。
- `openid` のスコープを持たないトークンで UserInfo を取得された場合、OAuth2 は、403 と `insufficient_scope` で拒否する。
- **例**：EX-OAUTH2-013-01、EX-OAUTH2-013-02

#### REQ-OAUTH2-020 失効したアクセストークンによる UserInfo の取得は `invalid_token` で拒否される

- 失効したアクセストークンで UserInfo を GET か POST で取得された場合、OAuth2 は、401 と `invalid_token` を `WWW-Authenticate: Bearer` のチャレンジとともに返し、`sub` もユーザーのクレームも返さない。
- **例**：EX-OAUTH2-020-01、EX-OAUTH2-020-02

### リソースサーバーによるトークンの検証

#### REQ-OAUTH2-029 mTLS バインド AT は同じ証明書のリクエストでのみ受理される

- `tls_client_auth` のクライアントが mTLS の証明書を提示してトークンを要求したとき、OAuth2 は、アクセストークンを証明書のサムプリントに結び付ける。
- 結び付けた証明書と同じ証明書を提示した要求を受けたとき、OAuth2 は、要求を受け付ける。
- 結び付けた証明書と異なる証明書を提示した場合、OAuth2 は、401 と `invalid_token` で拒否する。
- **例**：EX-OAUTH2-029-01、EX-OAUTH2-029-02

#### REQ-OAUTH2-045 保護リソースの DPoP Proof は ath でアクセストークンに結び付けられる

- DPoP の鍵に結び付けたアクセストークンを、`ath` がそのトークンの base64url(SHA-256) である同じ鍵の DPoP Proof とともに保護リソースへ提示されたとき、OAuth2 は、要求を受け付ける。
- `ath` を含まないか、`ath` が別のトークンのハッシュである DPoP Proof を保護リソースで受けた場合、OAuth2 は、401 と `invalid_token` で拒否する。
- トークンエンドポイントで `ath` を含まない DPoP Proof を受けたとき、OAuth2 は、`ath` を求めずにトークンを発行する。
- **例**：EX-OAUTH2-045-01、EX-OAUTH2-045-02、EX-OAUTH2-045-03、EX-OAUTH2-045-04

#### REQ-OAUTH2-044 Bearer 保護リソースの認証エラーはメタデータ URL を提示する

- 無効なアクセストークンで保護 API を呼ばれた場合、OAuth2 は、401 を返し、`WWW-Authenticate` に Bearer の `error="invalid_token"` と、レルムの発行者の下の `/.well-known/oauth-protected-resource` を指す `resource_metadata` を引用符付きで含める。
- 必要なスコープのないアクセストークンで保護 API を呼ばれた場合、OAuth2 は、403 を返し、`WWW-Authenticate` に `error="insufficient_scope"`、必要なスコープ、`resource_metadata` を含める。
- クライアントが `resource` を指定せずに Protected Resource Metadata を取得したとき、OAuth2 は、レルムの IdMagic API のメタデータを返し、`ProtectedResourceMetadataServed` を発行する。
- 登録していない `resource` の Protected Resource Metadata を要求された場合、OAuth2 は、400 と `invalid_target` で拒否する。
- **例**：EX-OAUTH2-044-01、EX-OAUTH2-044-02、EX-OAUTH2-044-03

#### REQ-OAUTH2-047 管理コンソールとアカウントポータルの Bearer 認証も失効判定を通る

- 管理コンソールかアカウントポータルの API へ Bearer のアクセストークンを提示されたとき、OAuth2 は、イントロスペクションと同じ失効の判定を通す。
- `jti` が失効のリストに載っているか、Agent の revocation epoch より前に発行したトークンを提示された場合、OAuth2 は、401 と `invalid_token` で拒否する。
- revocation epoch より後に発行したトークンを提示されたとき、OAuth2 は、認証を成立させ、スコープとロールの境界で判定させる。
- **例**：EX-OAUTH2-047-01、EX-OAUTH2-047-02、EX-OAUTH2-047-03

## セキュリティ上の考慮

到達できる範囲は、トークンのスコープが決める。
`account:*` を含むユーザーに紐づくスコープは、User の subject を持たないグラント（`client_credentials` や、subject を伴わない Token Exchange）では発行しない。

代行（Token Exchange）は権限を広げない。
`act` チェーンの上のすべての actor が有効であり、要求するスコープと `authorization_details` が元の権限の部分集合であることを求める。
チェーンの深さはテナントの `max_delegation_depth`（システムのデフォルトは 3）を超えられず、上書きを解決できない場合は交換を拒否する。
