# デバイス認可

## 概要

この文書は、入力の制約のある端末のための OAuth 2.0 Device Authorization Grant の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | デバイスコードとユーザーコードの発行、利用者による承認、ポーリングによるトークンの取得 |
| 行為者 | 登録済みのクライアント、ResourceOwner |
| 扱わないもの | 利用者の認証は `Authentication` が扱う |

## 状態遷移

### DeviceCodeFlow

RFC 8628 デバイス認可グラントのライフサイクル。device_code と user_code がペアで進む。承認と拒否の要求は、`Issued` の記録をまずユーザーコードの入力として `UserCodeEntered` へ進め、同じ要求の中で続けて `Approved` か `Denied` へ進める。`Expired` は、記録の状態にかかわらず `expires_at` を過ぎたことの判定でもある。

| State | Kind | Meaning |
|---|---|---|
| Issued | initial | `device_code` と `user_code` を発行した。利用者の入力を待っている |
| UserCodeEntered | — | 利用者が `user_code` を入力した。承認の判断を待っている |
| Expired | terminal | 有効期間内に引き換えなかった |
| Approved | — | 利用者が承認した。`/token` での引き換えを待っている |
| Denied | terminal | 利用者が拒否した |
| Exchanged | terminal | 承認済みコードをトークンへ引き換えた |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Issued | EnterUserCode | — | UserCodeEntered |  |
| Issued | Expire | — | Expired |  |
| UserCodeEntered | Approve | — | Approved |  |
| UserCodeEntered | Deny | — | Denied |  |
| UserCodeEntered | Expire | — | Expired |  |
| Approved | Exchange | — | Exchanged |  |
| Approved | Expire | — | Expired |  |

| State | 承認 | 拒否 | トークンの交換 | 有効期間の経過 |
|---|---|---|---|---|
| Issued | → UserCodeEntered（同じ要求で続けて承認する） | → UserCodeEntered（同じ要求で続けて拒否する） | 拒否：400 authorization_pending（間隔を守った）<br>拒否：400 slow_down（間隔より短い） | → Expired |
| UserCodeEntered | → Approved | → Denied | 拒否：400 authorization_pending（間隔を守った）<br>拒否：400 slow_down（間隔より短い） | → Expired |
| Approved | 拒否：400 invalid_request | 拒否：400 invalid_request | → Exchanged | → Expired |
| Denied | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 access_denied | 何もしない |
| Exchanged | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_grant | 何もしない |
| Expired | 拒否：400 expired_token | 拒否：400 expired_token | 拒否：400 expired_token | 何もしない |

## 操作

### クライアントと利用者によるデバイス認可フロー

#### REQ-OAUTH2-027 デバイス認可フローでアクセストークンを取得できる

- `device_code` のグラントを許可したクライアントがデバイス認可を要求したとき、OAuth2 は、`Issued` の記録を作り、200 と `device_code`、`user_code`、`verification_uri`、`verification_uri_complete`、`expires_in`、5 秒の `interval` を返し、`DeviceAuthorizationRequested` を発行する。
- `scope` のないデバイス認可を受けたとき、OAuth2 は、`openid` を要求したものとして扱う。
- 認証済みの利用者が有効な `user_code` を承認したとき、OAuth2 は、記録を `Approved` にして利用者と認証の時刻を結び付け、200 を返し、`DeviceAuthorizationApproved` を発行する。
- 認証済みの利用者が有効な `user_code` を拒否したとき、OAuth2 は、記録を `Denied` にし、200 を返し、`DeviceAuthorizationDenied` を発行する。
- クライアントが `Approved` の `device_code` を交換したとき、OAuth2 は、記録を `Exchanged` にし、アクセストークン、ID トークン、リフレッシュトークンを返し、`AccessTokenIssued` と `RefreshTokenIssued` を発行する。
- 承認の前の `device_code` の交換を受けたとき、OAuth2 は、400 と `authorization_pending` を返し、ポーリングの時刻を記録する。
- `interval` より短い間隔の交換を受けた場合、OAuth2 は、400 と `slow_down` で拒否し、`interval` を延ばす。
- 有効期間を過ぎた `device_code` か `user_code` を受けた場合、OAuth2 は、400 と `expired_token` で拒否する。
- 拒否された `device_code` の交換を受けた場合、OAuth2 は、400 と `access_denied` で拒否する。
- 交換済みの `device_code`、未知の `device_code`、別のクライアントか別のテナントの `device_code` の交換を受けた場合、OAuth2 は、400 と `invalid_grant` で拒否する。
- 承認した利用者が `Active` でない交換を受けた場合、OAuth2 は、400 と `invalid_grant` で拒否し、トークンを発行しない。
- 未知の `client_id` のデバイス認可を受けた場合、OAuth2 は、`invalid_client` で拒否する。
- `device_code` のグラントを許可していないクライアントのデバイス認可を受けた場合、OAuth2 は、400 と `unauthorized_client` で拒否する。
- クライアントに登録していないスコープのデバイス認可を受けた場合、OAuth2 は、400 と `invalid_scope` で拒否する。
- 未知の `user_code`、別のテナントの `user_code`、承認か拒否の済んだ `user_code` の承認か拒否を受けた場合、OAuth2 は、400 と `invalid_request` で拒否する。
- 未認証の利用者の承認か拒否を受けた場合、OAuth2 は、401 と `authentication_required` で拒否する。
- **例**：EX-OAUTH2-027-01、EX-OAUTH2-027-02、EX-OAUTH2-027-03、EX-OAUTH2-027-04
