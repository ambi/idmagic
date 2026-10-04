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

RFC 8628 デバイス認可グラントのライフサイクル。device_code と user_code がペアで進む。

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

## 操作

### クライアントと利用者によるデバイス認可フロー

#### REQ-OAUTH2-027 デバイス認可フローでアクセストークンを取得できる
