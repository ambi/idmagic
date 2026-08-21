# OAuth2 State Transitions

## ClientSecretCredentialLifecycle

クライアントシークレット資格情報は発行時に `Active` となり、期限到達で `Expired`、管理者による個別の失効で `Revoked` となる。`Revoked` は期限切れより優先して表示する。

| State | Kind | Meaning |
|---|---|---|
| Active | initial | クライアント認証に使える |
| Expired | terminal | `expires_at` に達した。認証には使えない |
| Revoked | terminal | 管理者が個別に失効させた。期限切れより優先して表示する |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Active | Expire | now() >= expires_at | Expired |  |
| Active | ClientSecretRevoked | — | Revoked |  |

## AuthorizationCodeFlow

`/authorize` から `/token` に至る認可リクエストのライフサイクル。

| State | Kind | Meaning |
|---|---|---|
| Received | initial | `/authorize` が要求を受け取った。まだ検証していない |
| AuthenticationPending | — | 要求は妥当で、主体を決めるログインを待っている |
| Rejected | terminal | 検証、認証、同意のいずれかで拒否した |
| Authenticated | — | 主体が決まった。同意の要否をこれから判定する |
| Expired | terminal | 有効期間内に次の段へ進まなかった |
| ConsentPending | — | 要求スコープが既存の同意で覆えず、同意画面の応答を待っている |
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

## DeviceCodeFlow

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

## ApprovalRequestLifecycle

人間の承認を待つ ApprovalRequest のライフサイクル。Pending から Approved / Denied / Expired へ一方向に進み、Consumed へ到達できるのは Approved からだけである。Consume は保存層の CAS でちょうど一度だけ成立し、並行するポーリングが二重にトークンを得ることはない。

| State | Kind | Meaning |
|---|---|---|
| Pending | initial | 起票済み。人間の判断を待っている |
| Approved | — | 人間が承認した。1 回だけトークンへ引き換えられる |
| Denied | terminal | 人間が拒否した |
| Expired | terminal | 判断または引き換えの前に有効期間が切れた |
| Consumed | terminal | 承認をトークンへ引き換えた。CAS によりちょうど一度だけ成立する |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Pending | Approve | now() < expires_at | Approved |  |
| Pending | Deny | — | Denied |  |
| Pending | Expire | — | Expired |  |
| Approved | Consume | now() < expires_at | Consumed |  |
| Approved | Expire | — | Expired |  |

## RefreshTokenLifecycle

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

## LogoutNotificationLifecycle

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

## AuthorizationCodeRecordLifecycle

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

## ConsentLifecycle

同意レコードのライフサイクル。GDPR Art.7(3) により Granted → Revoked が可能。

| State | Kind | Meaning |
|---|---|---|
| Granted | initial | 付与済みスコープ集合が有効である |
| Revoked | terminal | 利用者が取り消した。以後の認可には使えない |
| Expired | terminal | 付与から 365 日が経過した |

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| Granted | RevokeConsent | — | Revoked |  |
| Granted | Expire | — | Expired |  |

## PARRecordLifecycle

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
