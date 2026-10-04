# 認可

## 概要

この文書は、認可エンドポイントでの認可の判断と、認可コードと PAR の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 認可コードフロー、PKCE、Pushed Authorization Requests、認可コードの単回の交換、`nonce` の伝播 |
| 行為者 | 登録済みのクライアント、ResourceOwner |
| 扱わないもの | 同意の判定は[同意](../consent/README.md)が、トークンの発行は[トークン](../token/README.md)が、利用者の認証は `Authentication` が扱う |

## モデル

`AuthorizationRequest`、`AuthorizationCodeRecord`、`PARRecord` のライフサイクルは、条件分岐ではなく宣言的な遷移表で表す。
PKCE の要否はクライアントごとのメタデータ（`require_pkce`）で定め、`code_challenge_method` は `S256` だけを許す。
認可コードは、PKCE の有無によらず一度だけ使える短命な値（60 秒以下）とする。
`request_uri` は一度だけ使え、TTL は 600 秒以下とする。

- **判断**：PKCE を一律に強制しない理由は、[PKCE の必須化を公開クライアントと FAPI のクライアントに限る](../design/decisions.md#pkce-の必須化を公開クライアントと-fapi-のクライアントに限る)。
- **判断**：Pushed Authorization Requests は FAPI 2.0 のクライアントで必須、そのほかでは任意とし、最も強い保証の要るクライアントの `/authorize` で、URL の改ざんと未認証の要求の偽造を防ぐ。
- **判断**：遷移表で表す理由は、[認可とデバイスのライフサイクルを宣言的な遷移表で表す](../design/decisions.md#認可とデバイスのライフサイクルを宣言的な遷移表で表す)。

## 状態遷移

### AuthorizationCodeFlow

`/authorize` から `/token` に至る認可リクエストのライフサイクル。ログインの完了は、`Received` の要求を一つの処理の中で `AuthenticationPending`、`Authenticated`、`CodeIssued` へ進める。

| State | Kind | Meaning |
|---|---|---|
| Received | initial | `/authorize` が要求を受け取った。まだ検証していない |
| AuthenticationPending | — | 要求は妥当で、主体を決めるログインを待っている |
| Rejected | terminal | 検証、認証、同意のいずれかで拒否した |
| Authenticated | — | 主体が決まった。同意の要否をこれから判定する |
| Expired | terminal | 有効期間内に次の段へ進まなかった |
| ConsentPending | — | 要求スコープが既存の同意で覆えず、同意画面のレスポンスを待っている |
| CodeIssued | — | 認可コードを発行し、`/token` での引き換えを待っている |
| Consented | — | 要求スコープを覆う同意が揃った |
| Exchanged | terminal | 認可コードを引き換えた |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Received | Validate | — | AuthenticationPending |  |
| Received | Reject | — | Rejected |  |
| Received | Expire | — | Expired |  |
| AuthenticationPending | AuthenticateUser | — | Authenticated |  |
| AuthenticationPending | Reject | — | Rejected |  |
| AuthenticationPending | Expire | — | Expired |  |
| Authenticated | RequestConsent | — | ConsentPending |  |
| Authenticated | IssueCode | — | CodeIssued |  |
| ConsentPending | GrantConsent | — | Consented |  |
| ConsentPending | Reject | — | Rejected |  |
| ConsentPending | Expire | — | Expired |  |
| Consented | IssueCode | — | CodeIssued |  |
| CodeIssued | RedeemCode | — | Exchanged |  |
| CodeIssued | Expire | — | Expired |  |

| State | 認証の開始 | 利用者の認証 | 同意の要求 | 同意の付与 | 認可コードの発行 | 認可コードの交換 | 認可の拒否 | 有効期間の経過 |
|---|---|---|---|---|---|---|---|---|
| Received | → AuthenticationPending | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_grant | → Rejected | → Expired |
| AuthenticationPending | 拒否：400 invalid_request | → Authenticated | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_grant | → Rejected | → Expired |
| Authenticated | 拒否：400 invalid_request | 拒否：400 invalid_request | → ConsentPending | 拒否：400 invalid_request | → CodeIssued | 拒否：400 invalid_grant | 拒否：400 invalid_request | 何もしない |
| ConsentPending | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_request | → Consented | 拒否：400 invalid_request | 拒否：400 invalid_grant | → Rejected | → Expired |
| Consented | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_request | → CodeIssued | 拒否：400 invalid_grant | 拒否：400 invalid_request | 何もしない |
| CodeIssued | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_request | → Exchanged | 拒否：400 invalid_request | → Expired |
| Rejected | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_grant | 何もしない | 何もしない |
| Expired | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_grant | 何もしない | 何もしない |
| Exchanged | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_grant | 何もしない | 何もしない |

### AuthorizationCodeRecordLifecycle

発行された AuthorizationCode 本体のライフサイクル。AuthorizationCodeFlow（AuthorizationRequest 側）の Exchanged に対応するのが Redeemed。期限を過ぎて一度も提示されなかったコードは `Issued` のまま残り、提示された時点で `Expired` になる。

| State | Kind | Meaning |
|---|---|---|
| Issued | initial | 発行済み。1 回だけ引き換えられる |
| Redeemed | terminal | 引き換え済み。再提示は再利用として拒否する |
| Expired | terminal | 有効期間 (60 秒以下) を過ぎた |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Issued | RedeemCode | now() < expires_at | Redeemed |  |
| Issued | Expire | — | Expired |  |

| State | 認可コードの交換 |
|---|---|
| Issued | → Redeemed（期限内）<br>→ Expired（期限切れ。400 invalid_grant を返す） |
| Redeemed | 拒否：400 invalid_grant |
| Expired | 拒否：400 invalid_grant |

### PARRecordLifecycle

PAR で発行した `request_uri` のライフサイクル。`/authorize` から一度だけ参照できる（RFC 9126）。`Expired` は保存した状態ではなく、`expires_at` を過ぎたことの判定である。

| State | Kind | Meaning |
|---|---|---|
| Stored | initial | `request_uri` を発行した。`/authorize` から 1 回だけ参照できる |
| Used | terminal | `/authorize` が参照した |
| Expired | terminal | 有効期間 (600 秒以下) を過ぎた |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Stored | Use | now() < expires_at | Used |  |
| Stored | Expire | — | Expired |  |

| State | request_uri による認可の要求 | 有効期間の経過 |
|---|---|---|
| Stored | → Used | → Expired |
| Used | 拒否：400 invalid_request_uri | 何もしない |
| Expired | 拒否：400 invalid_request_uri | 何もしない |

## 操作

### クライアントと利用者による認可コードフロー

#### REQ-OAUTH2-005 認可コードフローでアクセストークンと ID トークンを取得できる

- クライアントが `response_type=code`、登録済みの `redirect_uri`、`S256` の `code_challenge`、クライアントに登録したスコープで `/authorize` を要求したとき、OAuth2 は、10 分の有効期間の `Received` の認可リクエストを保存し、ログインか同意へ進ませる。
- `scope` のない認可の要求を受けたとき、OAuth2 は、`openid` を要求したものとして扱う。
- 認証済みの利用者が Application の割り当てと実効のサインインポリシーを満たし、要求したスコープを同意が覆うとき、OAuth2 は、60 秒以下の有効期間の認可コードを発行し、`state` と発行者の識別子とともに `redirect_uri` へ返し、`AuthorizationCodeIssued` を発行する。
- クライアントが期限内の認可コードを、同じクライアント、同じ `redirect_uri`、一致する `code_verifier` で交換したとき、OAuth2 は、コードを `Redeemed` にし、アクセストークンと、`openid` のスコープには ID トークンを返し、`offline_access` のスコープにはリフレッシュトークンを返し、`AuthorizationCodeRedeemed`、`AccessTokenIssued` を発行する。
- `prompt=none` でセッションがない場合、OAuth2 は、画面もログインへのリダイレクトも出さずに、`state` と発行者の識別子を含む `login_required` を登録済みの `redirect_uri` へ返す。
- 未登録の `redirect_uri` か `client_id` のない要求を受けた場合、OAuth2 は、リダイレクトせずに `invalid_request` のエラーの画面を返す。
- 単一値の認可のパラメーターが重複するか、`prompt` に重複か未対応のトークンがあるか、`none` がほかの `prompt` のトークンと併用され、登録済みの `redirect_uri` を安全に確定できる場合、OAuth2 は、認可コードを発行せずに、`state` と発行者の識別子を含む `invalid_request` を `redirect_uri` へ返す。
- 単一値の認可のパラメーターが重複するか、`prompt` に重複か未対応のトークンがあるか、`none` がほかの `prompt` のトークンと併用され、登録済みの `redirect_uri` を安全に確定できない場合、OAuth2 は、認可コードを発行せずに、リダイレクトしないで `invalid_request` のエラーの画面を返す。
- `code_challenge` がないか `code_challenge_method` が `S256` でない要求を受けた場合、OAuth2 は、`invalid_request` で拒否する。
- `response_type` が `code` でない要求を受けた場合、OAuth2 は、`unsupported_response_type` で拒否する。
- 未知の `client_id` を受けた場合、OAuth2 は、`invalid_client` で拒否する。
- 認可コードのグラントを登録していないクライアントの要求を受けた場合、OAuth2 は、`unauthorized_client` で拒否する。
- クライアントに登録していないスコープの要求を受けた場合、OAuth2 は、`invalid_scope` で拒否する。
- 一致しない `code_verifier`、異なるクライアントか `redirect_uri`、未知の認可コードの交換を受けた場合、OAuth2 は、400 と `invalid_grant` で拒否し、トークンを発行しない。
- 引き換え済みの認可コードの交換を受けた場合、OAuth2 は、400 と `invalid_grant` で拒否し、そのコードから発行したファミリーのトークンをすべて失効させ、`RefreshTokenReuseDetected` と `TokenRevoked` を発行する。
- 発行から 60 秒を超えた認可コードの交換を受けた場合、OAuth2 は、コードを `Expired` にし、400 と `invalid_grant` で拒否する。
- 交換の時点で利用者が `Active` でない場合、OAuth2 は、400 と `invalid_grant` で拒否する。
- **例**：EX-OAUTH2-005-01、EX-OAUTH2-005-02、EX-OAUTH2-005-03、EX-OAUTH2-005-04、EX-OAUTH2-005-05、EX-OAUTH2-005-06、EX-OAUTH2-005-07、EX-OAUTH2-005-08

#### REQ-OAUTH2-022 認可リクエストの nonce は ID トークンに伝播する

- `nonce` を含む認可リクエストから発行した認可コードを交換したとき、OAuth2 は、同じ値の `nonce` のクレームを ID トークンに載せる。
- **例**：EX-OAUTH2-022-01

#### REQ-OAUTH2-015 認可コードの並行交換はちょうど一方だけ成功する

- 同じ認可コードを並行に交換されたとき、OAuth2 は、ストアの単位でちょうど一方の交換だけを成立させ、`AuthorizationCodeRedeemed` を発行する。
- 並行の交換で後れた側の場合、OAuth2 は、400 と `invalid_grant` で拒否し、そのコードのファミリーのトークンをすべて失効させ、`RefreshTokenReuseDetected` と `TokenRevoked` を発行する。
- **例**：EX-OAUTH2-015-01

### クライアントによる Pushed Authorization Requests

#### REQ-OAUTH2-009 PAR で送信した認可リクエストを request_uri 経由で実行する

- 認証したクライアントが認可のパラメーターを PAR のエンドポイントへ送ったとき、OAuth2 は、パラメーターを保存し、201 と一度だけ使える `request_uri` と 600 以下の `expires_in` を返し、`PARStored` を発行する。
- クライアントが期限内の `request_uri` と任意の `client_id` だけで `/authorize` を要求したとき、OAuth2 は、PAR の記録を `Used` にし、保存したパラメーターで認可を進める。
- 使用済みか期限切れか別のテナントの `request_uri` を受けた場合、OAuth2 は、`invalid_request_uri` で拒否する。
- `request_uri` と、併用を許可しないフロントチャネルの認可のパラメーターが混在するか、`client_id` が PAR と一致しない要求を受けた場合、OAuth2 は、認可コードを発行せずに `invalid_request` で拒否する。
- PAR を必須とする FAPI のクライアントが PAR なしで `/authorize` を要求した場合、OAuth2 は、`invalid_request` で拒否する。
- **例**：EX-OAUTH2-009-01、EX-OAUTH2-009-02

### クライアントと利用者による構造化された権限の要求

#### REQ-OAUTH2-053 認可詳細の要求は登録済みの種類だけを受け付け、明示の同意を求める

- クライアントがテナントに登録した種類の `authorization_details` を含む認可を要求したとき、OAuth2 は、種類の規則で検証した要求を保存し、要求した種類を載せた `AuthorizationDetailsRequested` を発行する。
- `authorization_details` を含む認可を受けたとき、OAuth2 は、過去のスコープの同意で同意の画面を省かずに、明示の同意を求める。
- 利用者が `authorization_details` を含む要求に同意したとき、OAuth2 は、同意した内容を認可コードとアクセストークンへ引き継ぎ、`AuthorizationDetailsConsented` を発行する。
- `type` のない要素、登録していない種類、種類の規則を満たさない要素を含む `authorization_details` を受けた場合、OAuth2 は、400 と `invalid_authorization_details` で拒否する。

### クライアントによるリソースの指定

#### REQ-OAUTH2-054 リソースの指定は登録済みの有効な MCP のリソースサーバーに audience を限る

- クライアントがテナントに登録した `Active` な MCP のリソースサーバーを `resource` に一つだけ指定して認可、PAR、トークンを要求したとき、OAuth2 は、アクセストークンの audience をその `resource` だけにし、`ResourceScopedTokenIssued` を発行する。
- リフレッシュトークンをローテーションするとき、OAuth2 は、初回の発行で結び付けた `resource` を引き継ぐ。
- 二つ以上の `resource`、登録していないか無効な `resource` を受けた場合、OAuth2 は、400 と `invalid_target` で拒否し、`ResourceAudienceRejected` を発行する。
- トークンの要求の `resource` が認可の要求で結び付けた `resource` と異なる場合、OAuth2 は、400 と `invalid_target` で拒否し、`ResourceAudienceRejected` を発行する。
- リソースサーバーが許可するスコープを超える要求を受けた場合、OAuth2 は、400 と `invalid_scope` で拒否する。

## セキュリティ上の考慮

`/authorize` は、ブラウザーのログインセッションで主体を決め、Application の割り当てと実効のサインインポリシーを満たしたうえで、`(subject, client_id)` の同意が要求したスコープを覆う場合にだけ、認可コードを発行する。
