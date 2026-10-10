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

| State | ログアウトトークンの配信 |
|---|---|
| Pending | → Delivered（RP が 2xx を返した）<br>何もしない（一時的に失敗し、再試行が残る）<br>→ Failed（`max_attempts` に達した） |
| Delivered | 何もしない |
| Failed | 何もしない |

## 操作

### クライアントによる RP-Initiated Logout

#### REQ-OAUTH2-023 RP-Initiated Logout は登録済み post_logout_redirect_uri にだけ戻す

- クライアントが登録済みの `post_logout_redirect_uri` を指定して `/end_session` を GET か POST で呼んだとき、OAuth2 は、ローカルのセッションを終えた後に、`state` を付けた `post_logout_redirect_uri` へ 302 でリダイレクトする。
- クライアントを特定できない `/end_session` を受けたとき、OAuth2 は、ローカルのセッションを終えた後に、サインアウトの状態の画面へ 303 でリダイレクトする。
- 同じセッションにフロントチャネルのログアウトを登録した RP があるとき、OAuth2 は、各 RP のログアウトの URI を隠れた iframe で読み込み、その後に戻り先へ遷移させる 200 の HTML を返す。
- 登録していない `post_logout_redirect_uri` を受けた場合、OAuth2 は、400 と `invalid_request` で拒否し、リダイレクトしない。
- **例**：EX-OAUTH2-023-01、EX-OAUTH2-023-02

#### REQ-OAUTH2-024 RP-Initiated Logout は id_token_hint からセッションとクライアントを特定する

- 署名、`iss`、`aud`、`sub`、`sid` の検証を通る `id_token_hint` を受けたとき、OAuth2 は、`sid` の示す LoginSession を失効させ、同じ `sid` を持つすべてのクライアントのリフレッシュトークンを `Revoked` にし、セッションの Cookie を消す。
- `exp` を過ぎただけの `id_token_hint` を受けたとき、OAuth2 は、`exp` を理由に拒否せずに `sid` からセッションを解決する。
- `id_token_hint` のない `/end_session` を受けたとき、OAuth2 は、ブラウザーの Cookie の示すセッションを終える。
- `id_token_hint` の `aud` が指定した `client_id` と一致しないか、署名を IdMagic の署名鍵で検証できないか、`sid`、`sub`、`aud` のどれかがないか、`sub` が `sid` のセッションの主体と一致しない場合、OAuth2 は、400 と `invalid_request` で拒否し、どのセッションもリフレッシュトークンも失効させない。
- **例**：EX-OAUTH2-024-01、EX-OAUTH2-024-02、EX-OAUTH2-024-03、EX-OAUTH2-024-04、EX-OAUTH2-024-05、EX-OAUTH2-024-06

### セッションの失効によるバックチャネルのログアウト

#### REQ-OAUTH2-025 セッション失効時は `backchannel_logout_uri` を登録済みの RP へログアウトトークンを配信する

- `/end_session` でセッションを終えたとき、OAuth2 は、そのセッションで認可した `backchannel_logout_uri` を登録済みの RP ごとに `Pending` の LogoutNotification を作り、配信のジョブを登録する。
- 配信のジョブを実行したとき、OAuth2 は、署名済みのログアウトトークンを `backchannel_logout_uri` へ送り、RP が 2xx を返した通知を `Delivered` にする。
- 配信が一時的に失敗した場合、OAuth2 は、通知を `Pending` のまま試行の回数を増やして再試行し、ローカルのセッションとリフレッシュトークンの失効を取り消さない。
- `max_attempts` まで再試行しても配信に成功しない場合、OAuth2 は、通知を `Failed` にし、ローカルの失効を取り消さない。
- 配信のジョブを登録できない場合、OAuth2 は、エラーを記録し、ログアウトの応答を失敗させない。
- **例**：EX-OAUTH2-025-01、EX-OAUTH2-025-02、EX-OAUTH2-025-03
