# クライアント

## 概要

この文書は、OAuth クライアントの登録、メタデータの解決、クライアントの認証、クライアントシークレットのライフサイクルの仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 動的クライアント登録、Client ID Metadata Documents によるクライアントの解決、トークンエンドポイントでのクライアントの認証、管理者によるクライアントの作成と更新と削除、クライアントシークレットの追加と失効とローテーション |
| 行為者 | 登録済みのクライアント、テナント管理者 |
| 扱わないもの | クライアントに到達してよいかという関門は `Application` が、トークンの発行は[トークン](../token/README.md)が扱う |

## モデル

`token_endpoint_auth_method` は、`private_key_jwt`、`tls_client_auth`、`none`、`client_secret_post`、`client_secret_basic` の 5 種類に対応する。
`client_secret_jwt` は意図的に除く。

- **判断**：`client_secret_jwt` を除く理由は、[クライアントの認証から client_secret_jwt を除く](../design/decisions.md#クライアントの認証から-client_secret_jwt-を除く)。
- **判断**：HTTPS の URL の形式の `client_id` を解決するため、Dynamic Client Registration に加えて、クライアントの登録情報を永続化せずに使える Client ID Metadata Documents に対応する。

## 状態遷移

### ClientSecretCredentialLifecycle

クライアントシークレット資格情報は発行時に `Active` となり、期限到達で `Expired`、管理者による個別の失効で `Revoked` となる。`Revoked` は期限切れより優先して表示する。`Expired` は保存した状態ではなく、`expires_at` を過ぎたことの判定である。

| State | Kind | Meaning |
|---|---|---|
| Active | initial | クライアント認証に使える |
| Expired | terminal | `expires_at` に達した。認証には使えない |
| Revoked | terminal | 管理者が個別に失効させた。期限切れより優先して表示する |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | Expire | now() >= expires_at | Expired |  |
| Active | ClientSecretRevoked | — | Revoked |  |

| State | 資格情報の個別の失効 | 有効期間の経過 |
|---|---|---|
| Active | → Revoked | → Expired |
| Expired | 拒否：400 invalid_request | 何もしない |
| Revoked | 何もしない | 何もしない |

## 操作

### クライアントによる動的クライアント登録

#### REQ-OAUTH2-016 動的クライアント登録は `client_id` を採番して返す

- クライアントが正しいメタデータで動的クライアント登録を要求したとき、OAuth2 は、`client_id` を採番し、201 と `client_id`、confidential のクライアントには `client_secret` を返し、`ClientRegistered` を発行する。
- 読めない本文を受けた場合、OAuth2 は、400 と `invalid_request` で拒否する。
- `redirect_uris` のない登録か、検証を通らないメタデータか `jwks_uri` を受けた場合、OAuth2 は、400 と `invalid_client_metadata` で拒否し、クライアントを登録しない。
- **例**：EX-OAUTH2-016-01、EX-OAUTH2-016-02

#### REQ-OAUTH2-017 クライアントメタデータの取得では公開 IP へ直接接続する

- 登録のない HTTPS の URL の形式の `client_id` を受けたとき、OAuth2 は、環境のプロキシを使わずに、DNS で検査した公開 IP へ直接接続して Client ID Metadata Document を取得し、検証に成功した文書をクライアントとして使い、`ClientIdMetadataDocumentResolved` を発行する。
- メタデータのホストがプライベート、ループバック、リンクローカル、CGNAT（100.64.0.0/10）の IP に解決された場合、OAuth2 は、その IP へ接続せず、未知の `client_id` として拒否し、`ClientIdMetadataDocumentRejected` を発行する。
- 文書を取得できないか検証できない場合、OAuth2 は、未知の `client_id` として拒否し、`ClientIdMetadataDocumentRejected` を発行する。
- **例**：EX-OAUTH2-017-01、EX-OAUTH2-017-02

### クライアントによるトークンエンドポイントでの認証

#### REQ-OAUTH2-007 不正なクライアント認証は invalid_client で一律拒否される

- 誤った `client_secret`、未知の `client_id`、登録と異なる認証方式のクライアント認証を受けた場合、OAuth2 は、理由を区別せずに 401 と `invalid_client` で拒否する。
- **例**：EX-OAUTH2-007-01、EX-OAUTH2-007-02

#### REQ-OAUTH2-028 改ざんされた client_assertion は invalid_client で拒否される

- `private_key_jwt` のクライアントの `client_assertion` を検証するとき、OAuth2 は、アルゴリズムの許可リスト、発行者、subject、audience、上限付きの有効期間、リプレイの防止を、Discovery の広告と同じ規則で確かめる。
- 署名を登録した鍵で検証できないか、いずれかの規則を満たさない `client_assertion` を受けた場合、OAuth2 は、401 と `invalid_client` で拒否する。
- **例**：EX-OAUTH2-028-01

### 管理者によるクライアントの管理

#### REQ-OAUTH2-035 管理者は所属テナントのクライアントを作成・更新・削除できる

- 管理者がクライアントを作成したとき、OAuth2 は、201 と、confidential のクライアントには一度だけ `client_secret` を返し、`AdminOAuth2ClientCreated` を発行する。
- 管理者がクライアントを一覧か取得したとき、OAuth2 は、200 と所属テナントのクライアントだけを返す。
- 管理者がクライアントを更新したとき、OAuth2 は、200 と更新したクライアントを返し、`AdminOAuth2ClientUpdated` を発行する。
- 管理者がクライアントを削除したとき、OAuth2 は、204 を返し、`AdminOAuth2ClientDeleted` を発行する。
- 不正なページの指定か読めない本文を受けた場合、OAuth2 は、400 と `invalid_request` で拒否する。
- 検証を通らないメタデータを受けた場合、OAuth2 は、400 と `invalid_client_metadata` で拒否する。
- 存在しないか別のテナントの `client_id` の取得、更新、削除を受けた場合、OAuth2 は、存在しない `client_id` と同じ 404 と `client_not_found` で拒否し、クライアントを変えない。
- Application が所有するクライアントの更新か削除を受けた場合、OAuth2 は、409 と `application_owned_protocol` で拒否する。
- **例**：EX-OAUTH2-035-01、EX-OAUTH2-035-02

### 管理者によるクライアントシークレットの管理

#### REQ-OAUTH2-036 管理者は Application から期限付きクライアントシークレットを追加発行し、個別に失効できる

- 管理者が `client_secret_basic` か `client_secret_post` の confidential のクライアントへ `expires_in_days` を指定してシークレットを追加発行したとき、OAuth2 は、新しいシークレットを一度だけ返し、その資格情報を `expires_at` 付きの `Active` にし、既存の資格情報の期限と状態を変えず、`ClientSecretIssued` を発行する。
- 複数の `Active` の資格情報を持つクライアントの認証を受けたとき、OAuth2 は、どの `Active` の資格情報のシークレットも受け付ける。
- 管理者が `Active` の資格情報を個別に失効させたとき、OAuth2 は、その資格情報を `Revoked` にし、以後そのシークレットを `invalid_client` で拒否し、`ClientSecretRevoked` を発行する。
- 管理者が `Revoked` の資格情報を再び失効させたとき、OAuth2 は、成功を返し、`ClientSecretRevoked` を発行しない。
- `ClientSecretIssued` と `ClientSecretRevoked` を発行するとき、OAuth2 は、操作者、クライアント、資格情報、期限の非機密のメタデータだけを載せる。
- 1 から 730 の範囲の外の `expires_in_days` を受けた場合、OAuth2 は、400 と `invalid_request` で拒否する。
- `Active` の資格情報がすでに 2 件あるクライアントへの追加発行を受けた場合、OAuth2 は、422 と `client_secret_limit_exceeded` で拒否し、既存の資格情報を変えない。
- `private_key_jwt`、mTLS、公開のクライアントへの追加発行か失効を受けた場合、OAuth2 は、400 と `invalid_request` で拒否する。
- 別のクライアントの `credential_id`、存在しない `credential_id`、期限の切れた資格情報の失効を受けた場合、OAuth2 は、400 と `invalid_request` で拒否する。
- **例**：EX-OAUTH2-036-01、EX-OAUTH2-036-02、EX-OAUTH2-036-03、EX-OAUTH2-036-04、EX-OAUTH2-036-05、EX-OAUTH2-036-06

#### REQ-OAUTH2-037 管理者は互換インターフェースからクライアントシークレットを無停止でローテーションできる

- 管理者が `grace_days` を指定してシークレットをローテーションしたとき、OAuth2 は、新しいシークレットを一度だけ返し、以前のシークレットを `grace_until` まで受け付け、操作者、クライアント、`grace_until` だけを載せた `ClientSecretRotated` を発行する。
- `grace_until` を過ぎた以前のシークレットのクライアント認証を受けた場合、OAuth2 は、401 と `invalid_client` で拒否する。
- 0 から 30 の範囲の外の `grace_days` を受けた場合、OAuth2 は、400 と `invalid_request` で拒否する。
- `private_key_jwt`、mTLS、公開のクライアントのローテーションを受けた場合、OAuth2 は、400 と `invalid_request` で拒否する。
- **例**：EX-OAUTH2-037-01、EX-OAUTH2-037-02、EX-OAUTH2-037-03

## セキュリティ上の考慮

クライアントの認証は `/token`、`/introspect`、`/revoke` などで行う。
認証に失敗した理由は区別せず、一律に `invalid_client` を返す。
`private_key_jwt` の検証は、アルゴリズムの許可リスト、発行者、subject、audience、上限付きの Assertion の有効期間、リプレイの防止という固定の規則に従い、Discovery の広告とサーバーでの強制を一致させる。

クライアントの管理（`admin:clients_manage`）は、`admin` ロールを持つ、有効かつ認証済みのユーザーが、所属テナントに対して行う。

## 実現方式

### アーキテクチャ

#### Client ID Metadata Documents（CIMD）による登録不要のクライアント解決

RFC 7591 の Dynamic Client Registration に加え、パスを持つ `https` URL 形式の `client_id` は `OAuth2Client` Repository ではなく、クライアントがホストする Client ID Metadata Document からその場で解決する。解決結果は永続化しない。Repository に `client_id` がない場合は文書を取得して 5 分間キャッシュし、他の処理経路と同じ `OAuth2Client` の形へ変換する。これにより、`redirect_uri` の照合、同意画面の描画、PKCE、スコープの処理に CIMD 固有の分岐は不要になる。統合点は、クライアントの Repository を包み、ID による取得だけを上書きする Decorator である。Repository で見つかれば取得前に終了し、保存、削除、一覧、資格情報の一覧は変更せず委譲する。組み立て地点で一度だけ接続するため、認可要求、PAR、クライアント認証の各処理は CIMD を意識しない。

取得には、`tokens_jose.JWKResolver` が `jwks_uri` に使うものと同じ SSRF 対策済みダイヤラー `shared/security/safehttp` を使う。HTTPS のみ、DNS 解決後にパブリック IP だけを許可、検証済み IP への直接接続、環境プロキシの不使用、リダイレクト回数・タイムアウト・レスポンスボディサイズの上限を強制する。プロキシは検査済みの接続経路の外で最終送信先を解決して接続し、トランスポート層の SSRF 境界を迂回するため、直接接続が必要である。共通パッケージでは 2 つの取得処理を別々に実装せず、1 つの堅牢化した実装の背後に置く。MVP が受け入れるのは、`token_endpoint_auth_method` を省略するか `none` と宣言する文書だけであり、それ以外はフェイルクローズに拒否する。文書の `client_id` フィールドは取得元 URL と完全に一致しなければならない。解決したクライアントの `scope` は文書の自己宣言値（デフォルトは `openid`）とし、新しい管理者管理カタログではなく RFC 7591 DCR と同じ自己宣言型の信頼モデルを使う。CIMD で解決したクライアントは `Application` に関連付けない。自己登録した DCR クライアントと同じであり、`ApplicationGate` は Application レコードがない場合をフェイルクローズに拒否せず、許可として扱う。

### セキュリティ

#### クライアント認証

`token_endpoint_auth_method` は `private_key_jwt`、`tls_client_auth`、`none`、`client_secret_post`、`client_secret_basic` の 5 種類に対応する。FAPI 級の非対称認証から従来の共有シークレット方式までを覆い、新しいクライアントを最も強い選択肢へ誘導しながら、既存環境の移行を継続できるようにする。`client_secret_jwt`（HMAC）は意図的に実装しない。`private_key_jwt` を利用できる場合、対称鍵による代替は能力を増やさずリスクだけを増やすためである。クライアント認証の失敗では `client_id` の登録有無を明かさず、常に `401 invalid_client` を返す。未登録の `client_id` にも同程度の検証コストを課し、タイミングオラクルを避ける。

`handlers_http` の `private_key_jwt` 検証は固定した規則の集合に従い、Discovery Metadata での広告とサーバーの実際の検査を一致させる。署名アルゴリズムは `PS256` と `ES256` だけを許可し（`none` と HMAC は不可）、`iss == sub == client_id`、このサーバーの発行者またはエンドポイント URL に一致する `audience`、クライアントが登録したインラインの `jwks` または `jwks_uri` から解決した署名鍵、上限付きのアサーション有効期間、`jti` の一度限りの利用を検査する。`jti` は DPoP と TTL および監査上の意味が異なるため、DPoP のリプレイストアとは別のストアに置く。同じリクエストで `client_assertion` と Basic 認証またはシークレット認証を組み合わせた場合は、RFC 6749 §2.3 に従い `invalid_request` として拒否する。
