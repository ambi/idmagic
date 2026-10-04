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

`/authorize` から `/token` に至る認可リクエストのライフサイクル。

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
| AuthenticationPending | AuthenticateUser | — | Authenticated |  |
| AuthenticationPending | Reject | — | Rejected |  |
| AuthenticationPending | Expire | — | Expired |  |
| Authenticated | RequestConsent | — | ConsentPending |  |
| Authenticated | IssueCode | — | CodeIssued |  |
| Authenticated | Reject | — | Rejected |  |
| ConsentPending | GrantConsent | — | Consented |  |
| ConsentPending | Reject | — | Rejected |  |
| ConsentPending | Expire | — | Expired |  |
| Consented | IssueCode | — | CodeIssued |  |
| Consented | Reject | — | Rejected |  |
| CodeIssued | RedeemCode | — | Exchanged |  |
| CodeIssued | Expire | — | Expired |  |

### AuthorizationCodeRecordLifecycle

発行された AuthorizationCode 本体のライフサイクル。AuthorizationCodeFlow（AuthorizationRequest 側）の Exchanged に対応するのが Redeemed。

| State | Kind | Meaning |
|---|---|---|
| Issued | initial | 発行済み。1 回だけ引き換えられる |
| Redeemed | terminal | 引き換え済み。再提示は再利用として拒否する |
| Expired | terminal | 有効期間 (60 秒以下) を過ぎた |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Issued | RedeemCode | now() < expires_at | Redeemed |  |
| Issued | Expire | — | Expired |  |

### PARRecordLifecycle

PAR で発行した `request_uri` のライフサイクル。`/authorize` から一度だけ参照できる（RFC 9126）。

| State | Kind | Meaning |
|---|---|---|
| Stored | initial | `request_uri` を発行した。`/authorize` から 1 回だけ参照できる |
| Used | terminal | `/authorize` が参照した |
| Expired | terminal | 有効期間 (600 秒以下) を過ぎた |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Stored | Use | now() < expires_at | Used |  |
| Stored | Expire | — | Expired |  |

## 操作

### クライアントと利用者による認可コードフロー

#### REQ-OAUTH2-005 認可コードフローでアクセストークンと ID トークンを取得できる

#### REQ-OAUTH2-022 認可リクエストの nonce は ID トークンに伝播する

#### REQ-OAUTH2-015 認可コードの並行交換はちょうど一方だけ成功する

### クライアントによる Pushed Authorization Requests

#### REQ-OAUTH2-009 PAR で送信した認可リクエストを request_uri 経由で実行する

## セキュリティ上の考慮

`/authorize` は、ブラウザーのログインセッションで主体を決め、Application の割り当てと実効のサインインポリシーを満たしたうえで、`(subject, client_id)` の同意が要求したスコープを覆う場合にだけ、認可コードを発行する。
