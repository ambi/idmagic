# トークン

## 概要

この文書は、トークンの発行、ローテーション、検証、失効、送信者制約、委譲の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | アクセストークン、リフレッシュトークン、ID トークンの発行、リフレッシュトークンのローテーション、イントロスペクション、失効、UserInfo、DPoP と mTLS の送信者制約、`client_credentials`、Token Exchange による委譲、保護されたリソースの認証エラー |
| 行為者 | 登録済みのクライアント、Agent、リソースサーバー |
| 扱わないもの | 認可コードの発行は[認可](authorization.md)が、承認による発行は[承認リクエスト](approval.md)が、署名鍵は `SigningKeys` が、トークンに載せるクレームは `ClaimMapping` が扱う |

## モデル

| 判断 | 内容 |
| --- | --- |
| トークンの形式 | アクセストークンは自己完結した JWT、リフレッシュトークンはデータベースに保存する不透明な参照をデフォルトとする |
| リフレッシュトークンのローテーション | 交換のたびにローテーションし、ローテーション済みのトークンが提示されたら、公開クライアントと confidential クライアントのどちらでもファミリー全体を失効させる |
| 送信者制約 | DPoP をデフォルトとし、クライアントの PKI を運用するクライアントには mTLS も選択肢として提供する |
| Agent | アイデンティティとライフサイクルを持つ第一級のプリンシパルだが、独自の資格情報は持たず、既存の `OAuth2Client` の登録に関連付ける |
| 代理の行為 | OAuth 2.0 Token Exchange として実装し、偽装ではなく、元の `sub` と `act` の Agent を保つ委任をデフォルトとする |
| Agent の権限 | 粗いスコープではなく RFC 9396 の `authorization_details` で表し、送金の上限などの制約を宣言する。後続の Token Exchange では権限を狭めることだけを許す |

- **判断**：トークンの形式の理由は、[アクセストークンを JWT にしリフレッシュトークンを不透明な参照にする](../decisions.md#アクセストークンを-jwt-にしリフレッシュトークンを不透明な参照にする)。
- **判断**：委譲の方式の理由は、[ユーザーの代理の行為を Token Exchange による委任で表す](../decisions.md#ユーザーの代理の行為を-token-exchange-による委任で表す)。

## 状態遷移

### RefreshTokenLifecycle

RefreshToken のライフサイクル。Rotate で子トークンに引き継がれ、Revoke で失効、Expire で期限切れ。Rotated 後も家族失効により Revoked へ遷移しうる（RFC 9700 §4.14）。

<!-- spec:state-machine backend/oauth2/testdata/RefreshTokenLifecycle.statemachine.json -->

| State | Kind | Meaning |
| --- | --- | --- |
| Active | initial | 提示するとローテーションしてトークンを再発行できる |
| Rotated | — | 子トークンへ引き継いだ。再提示は再利用として family 全体の失効を招く |
| Revoked | terminal | 利用者の操作、ログアウト、または family 失効で無効にした |
| Expired | terminal | スライディング期限または絶対期限に達した |

| From | Event | Guard | To | Effects |
| --- | --- | --- | --- | --- |
| Active | Rotate | now() < absolute_expires_at | Rotated |  |
| Active | RevokeToken | — | Revoked |  |
| Active | Expire | — | Expired |  |
| Rotated | RevokeToken | — | Revoked |  |
| Rotated | Expire | — | Expired |  |

| State | リフレッシュトークンのローテーション | トークンの失効 | 有効期間の経過 |
| --- | --- | --- | --- |
| Active | → Rotated（絶対期限内）<br>拒否：400 invalid_grant（絶対期限切れ） | → Revoked | → Expired |
| Rotated | → Revoked（再利用としてファミリー全体を失効させ、400 invalid_grant を返す） | → Revoked | → Expired |
| Revoked | 拒否：400 invalid_grant | 何もしない | 何もしない |
| Expired | 拒否：400 invalid_grant | 何もしない | 何もしない |

<!-- /spec:state-machine -->

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

#### REQ-OAUTH2-021 リフレッシュトークンは `refresh_token` のグラントを許可したクライアントに `offline_access` スコープを付与したときだけ発行する

- `refresh_token` のグラントを許可したクライアントが `offline_access` を含むスコープの認可コードか `device_code` を交換したとき、OAuth2 は、リフレッシュトークンを返し、`RefreshTokenIssued` を発行する。
- `offline_access` を含まないスコープの認可コードか `device_code` を交換したとき、OAuth2 は、リフレッシュトークンを返さない。
- `refresh_token` のグラントを許可していないクライアントが認可コードか `device_code` を交換したとき、OAuth2 は、リフレッシュトークンを返さず、応答とアクセストークンの `scope` から `offline_access` を除く。
- **判断**：トークンエンドポイントは許可していないグラントを `unauthorized_client` で拒否するので、そのクライアントへ渡すリフレッシュトークンは一度も使えず、漏えいの影響だけを持つ。応答の `scope` は、RFC 6749 の 5.1 に従い、実際に付与したスコープを示す。
- **例**：EX-OAUTH2-021-01、EX-OAUTH2-021-02、EX-OAUTH2-021-03

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

#### REQ-OAUTH2-055 Agent に束縛したクライアントのトークン要求は、鍵の所有を証明する認証方式のときだけ受け付ける

- `token_endpoint_auth_method` が `private_key_jwt` か `tls_client_auth` の、Agent に束縛したクライアントがトークンエンドポイントで認証したとき、OAuth2 は、グラントの種類によらず認証方式による拒否をせずに要求を処理する。
- `token_endpoint_auth_method` が `client_secret_basic`、`client_secret_post`、`none` のクライアントが Agent に束縛されている場合、OAuth2 は、401 と `invalid_client` で拒否し、トークンを発行せず、Agent、クライアント、認証方式、グラントの種類を載せた `AgentClientAuthenticationRejected` を発行する。
- 束縛した Agent のないクライアントの要求を受けたとき、OAuth2 は、認証方式による拒否をしない。
- **判断**：束縛の後でクライアントの認証方式が変わっても拒否できるように、束縛の作成ではなく要求ごとに判定する（REQ-IDMANAGEMENT-074 の束縛時の拒否は、誤った構成を早く知らせる補助である）。応答はほかの Agent の発行拒否（REQ-OAUTH2-046）と同じ `invalid_client` にし、束縛の有無を外部から区別させない。
- **例**：EX-OAUTH2-055-01、EX-OAUTH2-055-02、EX-OAUTH2-055-03

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
- 無効か失効した `subject_token` か `actor_token`、`may_act` に含まれない行為者、ポリシーが拒否した交換を受けた場合、OAuth2 は、400 と `invalid_grant` で拒否し、`TokenExchangeRejected` を発行する。
- `subject_token` か `actor_token` を、イントロスペクションと同じ失効の判定（REQ-OAUTH2-011、REQ-OAUTH2-012）で確かめたとき、OAuth2 は、`active=false` になるトークンを失効したトークンとして扱う。
- **例**：EX-OAUTH2-048-01、EX-OAUTH2-048-02、EX-OAUTH2-048-03、EX-OAUTH2-048-04

#### REQ-OAUTH2-060 本人の付与がない Agent は利用者を代行できない

- サブジェクトが利用者の Token Exchange を受けたとき、OAuth2 は、`subject_token` と `actor_token` の発行時の Agent と、要求したクライアントに束縛された Agent のすべてについて、本人の付与を確かめる。
- どの Agent にも、`Granted` で期限内で、発行するスコープを含む付与があるとき、OAuth2 は、拒否せずに要求を処理し、それらの付与の `last_used_at` を更新する。
- 付与がないか、`Granted` でないか、期限切れか、発行するスコープを含まない Agent がある場合、OAuth2 は、400 と `invalid_grant` で拒否し、トークンを発行せず、`TokenExchangeRejected` を発行する。
- 付与を読めない場合、OAuth2 は、トークンを発行しない。
- 関わる Agent がない Token Exchange か、サブジェクトが登録されたクライアントの Token Exchange か、ワークロード ID 連携の Token Exchange を受けたとき、OAuth2 は、付与を確かめない。
- **判断**：`client_credentials` のトークンは別のクライアントの交換の後に `sub` と `client_id` が異なるので、両者の比較では利用者を見分けられない。サブジェクトを登録されたクライアントとして解決できなければ、利用者として付与を求める。
- **判断**：クライアントの同意では代わりにしない。同意はクライアントが利用者としてアクセスすることへの許可であり、Agent の代行への許可ではない。

#### REQ-OAUTH2-056 送信者制約付きの `subject_token` の交換は、同じ鍵の証明を伴うときだけ受け付け、同じ制約を引き継ぐ

- Token Exchange を受けたとき、OAuth2 は、`subject_token` の送信者制約と要求が示した DPoP Proof または mTLS の証明書から、次の表に従って発行するトークンの送信者制約を決める。

| `subject_token` の送信者制約 | 要求が示した証明 | 結果 |
| --- | --- | --- |
| なし | なし | 送信者制約のないトークンを発行する |
| なし | DPoP または mTLS | 示した鍵（両方なら DPoP）に結び付けたトークンを発行する |
| あり | `subject_token` と同じ方式の同じ鍵を含む | `subject_token` と同じ `cnf` のトークンを発行する |
| あり | `subject_token` と同じ方式の同じ鍵を含まない | 400 と `invalid_grant` で拒否する |

- OAuth2 は、外部ワークロードのアテステーションの `subject_token` を、送信者制約のない `subject_token` として扱う。
- 表によって Token Exchange を拒否したとき、OAuth2 は、トークンを発行せず、`AccessTokenIssued` と `TokenExchanged` を発行せず、`TokenExchangeRejected` を発行する。
- **判断**：クライアント認証とは別に、元のトークンを使う権利を鍵の所有で示させる。証明を省けるなら、交換を許可されたクライアントが盗んだ送信者制約付きのトークンを、別の鍵か Bearer のトークンへ書き換えられる。委譲先の鍵への付け替えは、委譲の許可を発行する仕組みがないため受け付けない。

### クライアントによる ID-JAG の取得

#### REQ-OAUTH2-066 発行の許可があるクライアントは ID Token を ID-JAG に交換できる

- `requested_token_type` が `urn:ietf:params:oauth:token-type:id-jag` で、認証したクライアント、`audience`、`resource` が `CrossAppAccessIssuePolicy` と一致し、`subject_token` が有効な ID Token である Token Exchange を受けたとき、OAuth2 は、`typ` が `oauth-id-jag+jwt` で有効期間が 300 秒の JWT を `IdJagSigning` の鍵で署名して返す。
- OAuth2 は、ID-JAG の `iss` を自身の発行者、`sub` を `subject_token` の `sub`、`aud` を `audience`、`client_id` をポリシーの `target_client_id`、`resource` を要求のとおりの値にし、`subject_token` の `auth_time`、`amr`、`acr` を引き継ぐ。
- ポリシーの `release_email` が有効で、利用者のメールアドレスが確認済みのとき、OAuth2 は、ID-JAG に `email` を載せる。
- `scope` を省いた要求を受けたとき、OAuth2 は、ポリシーのスコープすべてを与える。
- `scope` を指定した要求を受けたとき、OAuth2 は、ポリシーのスコープとの共通部分を与え、結果を応答の `scope` と ID-JAG の `scope` に載せる。
- `authorization_details` を指定した要求を受けたとき、OAuth2 は、ポリシーの `authorization_detail_types` に含まれる型の詳細だけを ID-JAG と応答に載せる。
- ID-JAG を発行したとき、OAuth2 は、応答の `issued_token_type` を ID-JAG の URN、`token_type` を `N_A` にし、リフレッシュトークンを返さず、クライアント、`audience`、リソース、スコープを載せた `IdJagIssued` を発行する。
- **判断**：IdP を介した委任の監査を発行の時点で残す。ID-JAG の本文と利用者のデータはイベントへ載せない。

#### REQ-OAUTH2-067 発行の許可の外にある ID-JAG の要求は拒否される

- 認証しないか `public` のクライアントの要求を受けた場合、OAuth2 は、`invalid_client` で拒否する。
- `actor_token` を含む要求か、`audience` が 1 個でない要求を受けた場合、OAuth2 は、400 と `invalid_request` で拒否する。
- `subject_token_type` が `urn:ietf:params:oauth:token-type:id_token` でない要求を受けた場合、OAuth2 は、400 と `invalid_request` で拒否する。
- `subject_token` の `typ` ヘッダーがあるか、`token_use` が `id_token` でないか、署名か `iss` か `exp` が正しくないか、`aud` が認証したクライアントだけでないか、利用者の失効エポックが `subject_token` の `iat` より後である場合、OAuth2 は、400 と `invalid_grant` で拒否する。
- `audience` がテナントのどの `CrossAppAccessIssuePolicy` にもない場合、OAuth2 は、400 と `invalid_target` で拒否する。
- `resource` がないか、ポリシーの `resources` にない `resource` を含む場合、OAuth2 は、400 と `invalid_target` で拒否する。
- 認証したクライアントのポリシーがその `audience` にないか、与えるスコープが空であるか、利用者が停止しているか、`relying_party_client_id` の Application への利用者のアクセスが許可されない場合、OAuth2 は、400 と `access_denied` で拒否する。
- 要求したクライアントに Agent が束縛され、その区分が人間の承認を要する場合、OAuth2 は、REQ-OAUTH2-050 と同じ判定で拒否する。
- 要求したクライアントに束縛された Agent に本人の代行の付与がない場合、OAuth2 は、400 と `access_denied` で拒否する。
- 要求を拒否したとき、OAuth2 は、ID-JAG を返さず、`IdJagIssued` を発行せず、理由を載せた `IdJagRejected` を発行する。

#### REQ-OAUTH2-068 ID-JAG の送信者制約は要求の証明から決まる

- 有効な DPoP Proof を伴う要求を受けたとき、OAuth2 は、ID-JAG の `cnf.jkt` をその鍵の JWK SHA-256 Thumbprint にする。
- DPoP Proof を伴わず、mTLS でクライアントを認証した要求を受けたとき、OAuth2 は、ID-JAG の `cnf.x5t#S256` をその証明書の thumbprint にする。
- どちらも伴わない要求を受けたとき、OAuth2 は、`cnf` のない ID-JAG を発行する。
- OAuth2 は、一つの ID-JAG に DPoP と mTLS の両方の束縛を載せない。

### クライアントによる ID-JAG の償還

#### REQ-OAUTH2-069 受け入れの許可がある ID-JAG だけをアクセストークンへ償還する

- `grant_type` が `urn:ietf:params:oauth:grant-type:jwt-bearer` か `urn:ietf:params:oauth:grant-type:jwt-dpop` で、`assertion` が次の条件をすべて満たす ID-JAG である要求を受けたとき、OAuth2 は、アクセストークンを発行する。
  - `typ` が `oauth-id-jag+jwt`、`alg` が RS256、ES256、PS256 のどれかで、発行者の `jwks_uri` の鍵で署名を検証できる。
  - `iss` が自身の発行者ではなく、テナントの有効な OIDC の `IdentityProviderConnection` の発行者のうち、ただ一つと一致する。
  - `aud` が自身の発行者だけである。
  - `iat`、`exp`、`jti` があり、`iat` が現在から 60 秒より先でなく、`exp` に 60 秒を足した時刻が現在より後で、`exp` と `iat` の差が 600 秒以下である。
  - `client_id` が認証したクライアントである。
  - `act` がない。
  - `resource` があり、値のすべてがテナントの Active な `McpResourceServer` である。
  - その接続と認証したクライアントの組の `CrossAppAccessRedeemPolicy` がある。
  - `sub` がその接続の `FederatedIdentity` から有効な利用者へ解決できる。
- `jti` をまだ受け付けていないとき、OAuth2 は、発行者と `jti` の組を `exp` に 60 秒を足した時刻まで記憶し、同じ組を再び受けたら拒否する。
- 署名か `typ`、`iss`、`aud`、時刻、`jti`、`client_id`、`act`、`resource` の条件を満たさない場合、OAuth2 は、400 と `invalid_grant` で拒否する。
- ポリシーがないか、`sub` を解決できないか、利用者が停止しているか、与えるスコープが空である場合、OAuth2 は、400 と `access_denied` で拒否する。
- 認証しないか `public` のクライアントの要求を受けた場合、OAuth2 は、`invalid_client` で拒否する。
- 要求を拒否したとき、OAuth2 は、トークンを発行せず、その要求で検証に失敗した ID-JAG の `jti` を記憶せず、理由を載せた `IdJagRejected` を発行する。
- **判断**：§9.4 の明示の設定は設けず、自身が発行した ID-JAG を常に拒否する。同じデプロイメントの別テナントは発行者が異なるので受け入れられる。

#### REQ-OAUTH2-070 償還で発行するトークンは ID-JAG とテナントの許可の範囲に限る

- ID-JAG を償還するとき、OAuth2 は、`sub` を解決した利用者、`aud` を `resource` のすべて、スコープを ID-JAG の `scope`、`CrossAppAccessRedeemPolicy` のスコープ、クライアントのスコープ、リソースのスコープの共通部分にしたアクセストークンを発行する。
- OAuth2 は、ID-JAG の `authorization_details` のうち登録した型で検証を通るものだけをトークンと応答に載せる。
- OAuth2 は、償還の応答でリフレッシュトークンを返さない。
- 償還でトークンを発行したとき、OAuth2 は、`AccessTokenIssued` と、リソースごとの `ResourceScopedTokenIssued` と、発行者とクライアントを載せた `IdJagRedeemed` を発行する。
- 要求したクライアントに Agent が束縛され、その区分が人間の承認を要する場合、OAuth2 は、REQ-OAUTH2-050 と同じ判定で拒否する。
- 要求したクライアントに束縛された Agent に本人の代行の付与がない場合、OAuth2 は、400 と `access_denied` で拒否する。

#### REQ-OAUTH2-071 送信者制約付きの ID-JAG は束縛した鍵の証明でだけ償還できる

- OAuth2 は、ID-JAG の `cnf` と要求の証明から、次の表に従って結果を決める。

| ID-JAG の `cnf` | `grant_type` | 要求の証明 | 結果 |
| --- | --- | --- | --- |
| `jkt` | `jwt-dpop` | 同じ鍵の DPoP Proof | 同じ鍵に結び付けたトークンを発行する |
| `jkt` | `jwt-dpop` | 別の鍵の DPoP Proof | 400 と `invalid_grant` で拒否する |
| `jkt` | `jwt-dpop` | DPoP Proof なし | 400 と `invalid_dpop_proof` で拒否する |
| `jkt` | `jwt-bearer` | 問わない | 400 と `invalid_grant` で拒否する |
| `x5t#S256` | `jwt-bearer` | 同じ証明書 | 同じ証明書に結び付けたトークンを発行する |
| `x5t#S256` | `jwt-bearer` | 別の証明書か証明書なし | 400 と `invalid_grant` で拒否する |
| ほかの方式か、`jkt` 以外の `jwt-dpop` | 問わない | 問わない | 400 と `invalid_grant` で拒否する |
| なし | `jwt-bearer` | DPoP Proof または mTLS | 示した鍵（両方なら DPoP）に結び付けたトークンを発行する |
| なし | `jwt-bearer` | なし | 送信者制約のないトークンを発行する |
| なし | `jwt-dpop` | 問わない | 400 と `invalid_grant` で拒否する |

- **判断**：DPoP に束縛した assertion を `jwt-bearer` で受けると、束縛が Bearer へ格下げされる。proof の失敗で `jti` を記憶しないのは、クライアントが nonce に応じて同じ ID-JAG で再試行できるようにするためである。

### 管理者によるアプリ間アクセスの許可の管理

#### REQ-OAUTH2-072 管理者はアプリ間アクセスの許可を登録、一覧、削除できる

- 管理者が要求したとき、OAuth2 は、自身のテナントの `CrossAppAccessIssuePolicy` と `CrossAppAccessRedeemPolicy` を登録、一覧、削除する。
- 同じクライアントと `audience` の発行の許可か、同じ接続とクライアントの受け入れの許可を登録しようとした場合、OAuth2 は、409 と `cross_app_access_policy_exists` で拒否する。
- 存在しないクライアントか接続を指す許可、`resources` か `scopes` が空の許可、`audience` か `resources` が絶対 URI でない許可を登録しようとした場合、OAuth2 は、400 で拒否する。
- 許可を登録または削除したとき、OAuth2 は、`CrossAppAccessPolicyCreated` か `CrossAppAccessPolicyDeleted` を発行する。
- 許可を削除したとき、OAuth2 は、以後の発行と償還を拒否し、発行済みの ID-JAG とアクセストークンを有効期限まで失効させない。
- ほかのテナントの許可か存在しない許可の削除を要求された場合、OAuth2 は、404 と `cross_app_access_policy_not_found` で拒否する。

### リソースサーバーによるイントロスペクション

#### REQ-OAUTH2-011 失効済みトークンのイントロスペクションは `active=false` だけを返す

- リソースサーバーが有効なトークンをイントロスペクトしたとき、OAuth2 は、`active=true` とクレームを返し、`TokenIntrospected` を発行する。
- 失効、ローテーション、期限切れのトークンか、未知のトークンをイントロスペクトされたとき、OAuth2 は、`active=false` だけを返し、ほかのフィールドを返さない。
- **例**：EX-OAUTH2-011-01

#### REQ-OAUTH2-012 停止した Agent が発行に関わったトークンはイントロスペクションで active=false になる

- アクセストークンを発行するとき、OAuth2 は、要求したクライアントに束縛された Agent、トークン交換の `subject_token` と `actor_token` の発行時の Agent、ワークロード ID 連携が名指す Agent を、発行時の Agent として `agent_chain` のクレームに載せる。
- 発行時の Agent（`agent_chain` と `agent_id` のクレームが名指す Agent）のどれかの revocation epoch より前に発行したトークンをイントロスペクトされたとき、OAuth2 は、`active=false` だけを返す。
- 発行時の Agent のどれかが削除されているか `Active` でないトークンをイントロスペクトされたとき、OAuth2 は、`active=false` だけを返す。
- トークンのクライアントに現在束縛された Agent の revocation epoch より前に発行したトークンをイントロスペクトされたとき、OAuth2 は、`active=false` だけを返す。
- 発行時の Agent がすべて `Active` で、関わるすべての revocation epoch より後に発行したトークンをイントロスペクトされたとき、OAuth2 は、`active=true` とクレームを返す。
- 発行時の Agent の状態か revocation epoch を読めない場合、OAuth2 は、トークンを有効として返さない。
- **判断**：束縛は解除と再束縛で変わるので、現在の束縛ではなく、トークンが発行時に記録した Agent で判定する。現在の束縛も読むのは、`agent_chain` を持たない以前のトークンに従来の保証を残すためである。
- **判断**：エポックに加えて状態も読む。削除した Agent はエポックを残さず、発行の判定と署名の間に止めた Agent は、署名の時刻がエポックより後になりうるからである。
- **判断**：発行時刻は `iat` の秒に切り捨てて比べるので、エポックと同じ秒に発行したトークンは無効の側に倒れる。
- **例**：EX-OAUTH2-012-01、EX-OAUTH2-012-02

#### REQ-OAUTH2-061 本人が付与を撤回した Agent の代行トークンはイントロスペクションで active=false になる

- `act` を持ち、サブジェクトが登録されたクライアントでないトークンをイントロスペクトされたとき、OAuth2 は、発行時の Agent（REQ-OAUTH2-012）のそれぞれについて、そのサブジェクトの付与を確かめる。
- 発行時の Agent のどれかの付与がないか、`Granted` でないか、期限切れか、トークンのスコープを含まないか、`granted_at` より前に発行したトークンをイントロスペクトされたとき、OAuth2 は、`active=false` だけを返す。
- 同じ Agent の別の利用者の付与を撤回したとき、OAuth2 は、そのトークンの判定を変えない。
- 付与を読めない場合、OAuth2 は、トークンを有効として返さない。
- **判断**：失効エポックは進めない。エポックは Agent 単位なので、進めると同じ Agent の他の利用者の代行まで止まる。
- **判断**：`granted_at` と発行時刻は秒に切り捨てて比べ、同じ秒に発行したトークンを有効にする。付与の直後の交換を拒否しないためである。

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
- `jti` が失効のリストに載っているか、REQ-OAUTH2-012 が `active=false` とするトークンか、REQ-SHAREDSIGNALS-012 が無効とする User のトークンを提示された場合、OAuth2 は、401 と `invalid_token` で拒否する。
- 失効のリストに載らず、REQ-OAUTH2-012 が `active=true` とするトークンを提示されたとき、OAuth2 は、認証を成立させ、スコープとロールの境界で判定させる。
- **例**：EX-OAUTH2-047-01、EX-OAUTH2-047-02、EX-OAUTH2-047-03

## セキュリティ上の考慮

到達できる範囲は、トークンのスコープが決める。
`account:*` を含むユーザーに紐づくスコープは、User の subject を持たないグラント（`client_credentials` や、subject を伴わない Token Exchange）では発行しない。

代行（Token Exchange）は権限を広げない。
`act` チェーンの上のすべての actor が有効であり、要求するスコープと `authorization_details` が元の権限の部分集合であることを求める。
チェーンの深さはテナントの `max_delegation_depth`（システムのデフォルトは 3）を超えられず、上書きを解決できない場合は交換を拒否する。

## 実現方式

### アーキテクチャ

JWT のクレームの組み立て、署名鍵の選択、発行したトークンの検証は OAuth2 の内部アダプターが担い、起動時はルートパッケージから組み立てる。
PS256 の署名処理だけは SharedSignals も使う共有ライブラリに置き、鍵の所有型を秘密鍵と `kid` に変換するのは各利用側で行う。

#### トークン形式：JWT アクセストークンと不透明なリフレッシュトークン

アクセストークンはデフォルトで自己完結型の JWT（RFC 9068）として、リフレッシュトークンはデータベースに記録を持つ不透明な参照として発行する。この非対称性は意図的である。リソースサーバーが多い場合、リクエストごとに `/introspect` を呼ぶ方式は拡張しにくい。そのため、リソースサーバーが JWKS だけで検証できる JWT をデフォルトとし、即時失効できない期間を短い TTL（600 秒）で制限する。一方、リフレッシュトークンにはローテーションとファミリー全体の失効が必要なため、データベースで管理する。リフレッシュトークンは平文ではなく SHA-256 ハッシュで保存する。送信者制約（`cnf`）や現在の失効状態を確認したいリソースサーバー向けに `/introspect` を公開するが、JWT アクセストークンの通常の検証経路にはしない。

#### エージェント主体とトークン交換による委譲

`Agent` は `User` や `OAuth2Client` と異なる第一級のプリンシパルであり、アイデンティティ、所有関係、目的、緊急停止を含むライフサイクルを持つ。ただし独自の資格情報の基本要素は持たず、1 個以上の既存の `OAuth2Client` 登録に束縛するため、エージェントのガバナンスに重複する第 2 の資格情報・暗号の仕組みは不要である。すべてのエージェントは所有者 (`User` またはグループ) を必須とし、所有者の離任をエージェントのアクセスへ連鎖する。すべてのトークン発行経路で `status` (`active` / `disabled` / `killed`) を安全側に検査し、状態を解決できない場合は発行を許可しない。所有者の離任の連鎖は、エージェントの `status` を書き換える一度きりの状態遷移ではなく、発行のたびに所有者を解決して有効性を確かめる評価として実装する。所有者を解決できない場合、または所有者が無効化・削除されている場合は発行しない。発行のたびの評価は、所有者の停止に伴う Agent の無効化（REQ-IDMANAGEMENT-081）が届かなかったデプロイでもガードを失わないための安全網である。所有者が復帰しても、無効化した Agent は `Disabled` のまま残り、管理者が再有効化するまで発行を再開しない。既に発行済みのトークンは、この評価ではなく、発行時の Agent の状態と SharedSignals の失効エポック（REQ-OAUTH2-012）が無効化する。アクセストークンのクレームは任意のプリンシパル種別の標識を持ち、既存のトークン利用者を壊さず、リソースサーバーと AuthZEN のポリシー層がエージェント向けに発行したトークンを区別できるようにする。

ユーザーの代理行為は `/token` の OAuth 2.0 Token Exchange（RFC 8693）として実装する。デフォルトの結果はなりすましではなく委任である。交換後のトークンは元のユーザーを `sub` に保ち、現在の行為者であるエージェントを `act` クレームに記録する。RFC 8693 §4.1 に従って以前の行為者を内側へ入れ子にするため、下位エージェントへの委任チェーンを追跡できる。なりすまし（`act` を削除して `sub` を置換）は、クライアントまたはエージェントへ明示的に許可した場合だけ利用できる。判断できない場合は、監査証跡を保つ委任をデフォルトとする。`may_act` と AuthZEN ポリシーが、許可する行為者、対象者、深さの組を共同で制御する。交換時には、結果を単一の対象へ狭める `resource` を必須とする（RFC 8707）。委任の最大深度で `act` チェーンの長さを制限する。上限はテナントごとに下げられるが、システムデフォルトを超えて上げることはできない。テナント設定から認可の境界を緩める経路を作らないための非対称性であり、パスワードポリシーの上書きと同じ扱いである。テナントの委譲ポリシーを解決できない場合は、デフォルトへ退避せず交換を拒否する。

交換後のトークンが自律実行と利用者の代理のどちらであるかは、新しい永続状態ではなく `act` チェーンとプリンシパル種別から導出する。導出は 1 つの関数に閉じ、イントロスペクションのレスポンスと監査イベントの双方がそれを通る。リソースサーバーと監査の担当者が同じ規則を各自で書き直すと解釈がずれ、しかもその食い違いは調査のときに最も見つけにくい形で現れるからである。導出値であるため、モードがクレームと食い違う第二の真実は生まれない。交換後のトークンは短命であり、リフレッシュトークンを発行しない。継続には再交換が必要なため、失効が有効に働く。subject token の送信者制約は、REQ-OAUTH2-056 に従って交換後のトークンへ引き継ぎ、鍵の所持証明を失わない。判定はユースケースがポリシーの評価の後、署名の前に 1 つの関数で一度だけ行い、グラントや行為者の種類ごとに別の照合を書かない。

#### エージェントの権限に対する Rich Authorization Requests

粗い OAuth スコープでは「口座 X から最大 $100 を送金」のような権限を表せないため、`/authorize`、`/par`、`/token` (トークン交換の grant を含む) は RFC 9396 `authorization_details` を受け入れ、広いスコープではなく構造化され上限のある権限を要求で宣言できるようにする。テナントごとに事前登録した `type` だけを受け入れ、各詳細をスキーマに対して安全側に検証する。未登録の種別やスキーマの不一致は部分的に受理せず拒否する。発行または交換するトークンが持てるのはユーザーが同意した内容の部分集合だけであり、後続の交換はその部分集合をさらに狭めることしかできず、広げられない。検査に使う半順序 (対象の包含、上限の単調な減少) は登録済みのスキーマ自体が定義する。同意 UI は生の JSON ではなく、スキーマに結び付いた人間向けテンプレートから各詳細を描画する。リソースサーバーは IdP が発行または introspect した詳細だけを信頼境界とし、付与内容を再解釈または拡張してはならない。`type` と粗い `scope` が同じ領域で重なる場合は構造化された詳細の上限を優先し、`authorization_details` で制限済みの領域を `scope` が再び広げる要求は拒否する。

### データ

#### 有効期間、セキュリティ、保持期間の設定

プロトコルの時間値とセキュリティパラメーター、すなわち認可コードの TTL（60 秒、一度だけ使用可能）、PAR の `request_uri` の TTL（600 秒、一度だけ使用可能）、アクセストークンの TTL（600 秒）、ID トークンの TTL（3,600 秒）、リフレッシュトークンの TTL（スライディング 14 日、絶対期限 30 日）、デバイスコードとユーザーコードの TTL（600 秒）、デフォルトのポーリング間隔（5 秒、`slow_down` ごとに 5 秒加算）、クライアント認証とコード交換の流量制限、DPoP の時刻ずれとリプレイ期間、同意レコードの保持期間（7 年）は、プロダクト目標ではなく 1 か所にまとめて記録する。これらはエラーバジェットの意味を持つ可用性・レイテンシー SLO ではなく、プロトコル、セキュリティ、運用の設定だからである。単一のモデルまたは状態インターフェースで自然に強制できる値は制約、ガード、契約として表す。複数リクエストにまたがる流量制限や、ライフサイクルにまたがる保持期間など、単一要素に属さない値は共有設定レコードを正とする。

### セキュリティ

#### リフレッシュトークンのローテーションと再利用検知

リフレッシュトークンは使用するたびにローテーションする。提示されたトークンを `rotated` にし、`parent_id` を持つ新しいトークンを発行する。1 回の認可コード交換から派生したすべてのトークンは `family_id` を共有する。すでにローテーション済みのトークンが提示された場合は再利用としてリクエストを拒否し、その `family_id` に属する全トークンを失効させ、`RefreshTokenReuseDetected` 監査イベントを発行する。運用と監査を単純に保つため、パブリッククライアントとコンフィデンシャルクライアントへ一様に適用する。2 つのブラウザータブなどによる正当な同時利用もリプレイと区別せず、一方だけが成功し、他方は再利用と判定する。猶予期間に伴う複雑さと外部状態のコストを避ける代わりに、まれに再ログインを求めることを受け入れる。`absolute_expires_at` は発行時に 30 日で固定し、ローテーションでは延長しない。

#### 送信者制約付きトークン：DPoP と mTLS

DPoP（RFC 9449）は TLS 終端プロキシの変更なしに Web アプリケーション、SPA、ネイティブクライアントで同じように動作するため、送信者制約のデフォルト方式とする。すでにクライアント PKI を運用する組織、特に FAPI や金融系のクライアントには、mTLS（RFC 8705）も選択肢として提供する。FAPI 2.0 プロファイルを宣言するクライアントは少なくとも一方を使い、一般プロファイルのクライアントは `dpop_bound_access_tokens` で選択する。DPoP Proof の検証では、`jwk` ヘッダーの署名、`htm` / `htu` / `iat` / `jti`、上限付きの時刻ずれ、`jti` のリプレイ期間を検査する。発行するトークンは JWK のサムプリントを `cnf.jkt` に持つ。

`htu` の期待値は、トークンエンドポイント、`/userinfo`、保護リソースのどれでも同じ形とし、テナントの正規ロケーションの origin にリクエストのパスを継いだ、クエリとフラグメントを含まない絶対 URL とする（RFC 9449 §4.2）。path style のテナントでは `https://{host}/realms/{realm}/...`、subdomain style のテナントでは `https://{realm}.{base}/...` になる。正規 issuer 全体ではなく origin だけを使うのは、path style の issuer がすでに `/realms/{realm}` を含み、リクエストのパスへ継ぐと prefix が二重になるためである。期待値はリクエストの `Host` ヘッダーや転送ヘッダーからは組まない。正規ロケーションは discovery が広告する URL と同じ出所なので、TLS 終端プロキシの配下でもクライアントが送る URL と一致し、ヘッダーの偽装で期待値を動かせない。期待値を決められない場合や、Proof が `htu` を持たない場合は拒否する。

Proof の検証はパラメーター化せず、エンドポイントの種類で分ける。バインドすべき対象が 2 種類で異なるためである。トークンエンドポイントでは対象のアクセストークンがまだ存在しないため、Proof は `ath` を持たない。保護リソースでは Proof が `ath = base64url(SHA-256(提示された access token))` を持ち、Introspection 後の表現ではなく、クライアントが提示した生のトークン文字列と定数時間で比較する。`ath` がなければ Proof は鍵の所持だけを示し、同じ `htm` / `htu` と鍵に対して発行したすべてのトークンで使い回せる。これは長時間動作し、複数のリソースサーバーをまたぎ、委任の各段で異なるトークンを持つエージェントにとって特に重要である。保護リソースで `ath` がない場合は、猶予フラグの背後で許容せず拒否する。オプトアウトをデフォルトにすると、バインドされていない Proof を受け入れることになり、まさに防ぐべき失敗だからである。mTLS の検証では、TLS 終端プロキシが検証済みのクライアント証明書を渡すことを信頼し、登録済みの `tls_client_auth_subject_dn` と照合して、発行トークンを `cnf.x5t#S256` でバインドする。`/userinfo` は提示された証明書のサムプリントがトークンの `cnf` と一致する場合だけ受け入れる。アクセストークンとリフレッシュトークンの両方が送信者制約を持つ。リフレッシュトークンではストアレコードの `sender_constraint` フィールドに保持するため、Proof of Possession はローテーション後も維持される。`/introspect` のレスポンスにも `cnf` を含め、リソースサーバーがリクエストごとに DPoP Proof を再検証できるようにする。
