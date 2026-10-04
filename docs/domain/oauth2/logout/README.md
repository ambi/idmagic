# ログアウト

## 概要

この文書は、RP-Initiated Logout とバックチャネルのログアウトの仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | `id_token_hint` によるセッションとクライアントの特定、登録済みの `post_logout_redirect_uri` への復帰、セッションの失効の時のログアウトトークンの配信 |
| 行為者 | 登録済みのクライアント、ResourceOwner、System（セッションの失効） |
| 扱わないもの | ログインセッションの失効そのものは `Authentication` が扱う |

## モデル

RP-Initiated Logout の `/end_session` は、ブラウザーの遷移で使う GET と、フォームの送信で使う POST の両方を提供する。
同じログアウトの処理へ到達するが HTTP の binding が異なるので、公開の契約ではそれぞれに一意の `operationId` を与える。

- **判断**：`sid` のクレームに `LoginSession.id` を使う理由は、[sid のクレームに LoginSession の ID をそのまま使う](../design/decisions.md#sid-のクレームに-loginsession-の-id-をそのまま使う)。

## 状態遷移

### LogoutNotificationLifecycle

LogoutNotification のライフサイクル。Deliver で成功確定、Exhaust で max_attempts 到達による最終失敗確定 (dead-letter)。Jobs 側の Retry は Pending のまま attempts のみ増やす (状態遷移ではない)。

| State | Kind | Meaning |
|---|---|---|
| Pending | initial | 配信待ち。Jobs 側の再試行中もこの状態にとどまる |
| Delivered | terminal | RP が 2xx を返した |
| Failed | terminal | `max_attempts` に達した。配信不能として確定する |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Pending | Deliver | — | Delivered |  |
| Pending | Exhaust | — | Failed |  |

## 操作

### クライアントによる RP-Initiated Logout

#### REQ-OAUTH2-023 RP-Initiated Logout は登録済み post_logout_redirect_uri にだけ戻す

#### REQ-OAUTH2-024 RP-Initiated Logout は id_token_hint からセッションとクライアントを特定する

### セッションの失効によるバックチャネルのログアウト

#### REQ-OAUTH2-025 セッション失効時は `backchannel_logout_uri` を登録済みの RP へログアウトトークンを配信する
