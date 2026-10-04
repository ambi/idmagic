# 承認リクエスト

## 概要

この文書は、Agent の操作に対する人間の承認を、CIBA のバックチャネル認可要求として扱う仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | バックチャネル認可要求の受け付け、対象のユーザーによる承認と拒否、承認の成立の後のトークンの発行、`Supervised` な Agent の承認の要求 |
| 行為者 | Agent、登録済みのクライアント、ResourceOwner（承認する本人） |
| 扱わないもの | `supervised` の Agent に承認を義務付けるかの方針は、ガバナンスの層が決める |

## モデル

承認の判断の記録には、UUID を鍵とし通信の方式に依存しない `ApprovalRequest` を使う。
CIBA の検索のフィールドとポーリングのフィールドは通信の上の記録だが、ストアが判断とポーリングを不可分に直列化できるよう、同じ場所に置く。
承認済みの要求はストアの単位の Compare-and-Set でトークンに交換し、同時のポーリングによる二重の発行を防ぐ。
それ以外の状態は `/token` で拒否する。

CIBA の配信のモードは `poll` だけを実装する。
承認の画面はすでにユーザーのセッションとステップアップ認証で保護されているので、`user_code` は未対応として広告する。

- **判断**：承認を CIBA で実装する理由は、[Agent の操作に対する人間の承認を CIBA で実装する](../design/decisions.md#agent-の操作に対する人間の承認を-ciba-で実装する)。

## 状態遷移

### ApprovalRequestLifecycle

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

| State | 承認 | 拒否 | トークンの交換 | 有効期間の経過 |
|---|---|---|---|---|
| Pending | → Approved（期限内）<br>拒否：400 invalid_request（期限切れ） | → Denied（期限内）<br>拒否：400 invalid_request（期限切れ） | 拒否：400 authorization_pending（間隔を守った）<br>拒否：400 slow_down（間隔より短い）<br>→ Expired（期限切れ。400 expired_token を返す） | → Expired |
| Approved | 拒否：400 invalid_request | 拒否：400 invalid_request | → Consumed（期限内）<br>→ Expired（期限切れ。400 expired_token を返す） | → Expired |
| Denied | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 access_denied | 何もしない |
| Expired | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 expired_token | 何もしない |
| Consumed | 拒否：400 invalid_request | 拒否：400 invalid_request | 拒否：400 invalid_grant | 何もしない |

## 操作

### Agent によるバックチャネル認可要求

#### REQ-OAUTH2-041 バックチャネル認可要求は人間の承認が成立してからトークンを発行する

- CIBA のグラントを許可したクライアントが、`openid` を含む許可済みのスコープと、テナントの `Active` な User を解決する `login_hint` か `id_token_hint` のどちらか一つでバックチャネル認可を要求したとき、OAuth2 は、`Pending` の承認リクエストを作り、200 と `auth_req_id`、`expires_in`、`interval` を返し、`BackchannelAuthRequested` を発行し、対象の User のメールアドレスへ承認の依頼を通知する。
- クライアントが `Approved` で期限内の `auth_req_id` を交換したとき、OAuth2 は、承認リクエストを Compare-and-Set で `Consumed` にし、要求したスコープと承認した User の `sub` のアクセストークンと ID トークンを返し、`AccessTokenIssued` を発行する。
- クライアントが期限を過ぎた `auth_req_id` を交換したとき、OAuth2 は、承認リクエストを `Expired` にし、`BackchannelAuthExpired` を発行し、400 と `expired_token` を返す。
- `Pending` の `auth_req_id` の交換を受けたとき、OAuth2 は、ポーリングの時刻を記録し、400 と `authorization_pending` を返す。
- `interval` より短い間隔の交換を受けた場合、OAuth2 は、400 と `slow_down` で拒否する。
- `scope` がないか `openid` を含まないか、クライアントに登録していないスコープの要求を受けた場合、OAuth2 は、400 と `invalid_scope` で拒否する。
- `login_hint` と `id_token_hint` の両方があるか両方がない要求、整数でない `requested_expiry`、1 から 600 秒の範囲の外の `requested_expiry` を受けた場合、OAuth2 は、400 と `invalid_request` で拒否する。
- 64 文字を超えるか制御文字を含む `binding_message` を受けた場合、OAuth2 は、400 と `invalid_binding_message` で拒否する。
- ヒントが User を解決しないか、別のテナントか `Active` でない User を指す場合、OAuth2 は、400 と `unknown_user_id` で拒否する。
- CIBA のグラントを許可していないクライアントの要求を受けた場合、OAuth2 は、400 と `unauthorized_client` で拒否する。
- **例**：EX-OAUTH2-041-01、EX-OAUTH2-041-02、EX-OAUTH2-041-03、EX-OAUTH2-041-04、EX-OAUTH2-041-05、EX-OAUTH2-041-06、EX-OAUTH2-041-07、EX-OAUTH2-041-08、EX-OAUTH2-041-09、EX-OAUTH2-041-10

#### REQ-OAUTH2-050 `Supervised` な Agent は人間の承認を経ずに新しいトークンを得られない

- `kind` が `supervised` の Agent に束縛したクライアントか、交換に関与する `supervised` の Agent が client_credentials か Token Exchange でトークンを要求した場合、OAuth2 は、400 と `unauthorized_client` で拒否し、トークンを発行せず、判断の根拠とした区分を載せた `AgentApprovalRequired` を発行する。
- `kind` が既知のどの値でもない Agent のトークンの要求を受けた場合、OAuth2 は、承認が必要な側として 400 と `unauthorized_client` で拒否する。
- 承認を経て `supervised` の Agent へ発行したトークンを `subject_token` とする Token Exchange を受けた場合、OAuth2 は、400 と `unauthorized_client` で拒否し、一つの承認を派生のトークンへ継承しない。
- 束縛した Agent のないクライアントか、関与するどの Agent も `autonomous` のクライアントがトークンを要求したとき、OAuth2 は、区分の判定で拒否せずに要求を処理する。
- 対象の User が `supervised` の Agent のバックチャネル認可を承認したとき、OAuth2 は、承認の対象の Agent の識別子を載せた `BackchannelAuthApproved` を発行し、`auth_req_id` の交換でトークンを発行させる。
- **例**：EX-OAUTH2-050-01、EX-OAUTH2-050-02、EX-OAUTH2-050-03、EX-OAUTH2-050-04、EX-OAUTH2-050-05、EX-OAUTH2-050-06、EX-OAUTH2-050-07

#### REQ-OAUTH2-042 承認が成立していない承認リクエストはトークンを発行しない

- 拒否された `auth_req_id` の交換を受けた場合、OAuth2 は、400 と `access_denied` で拒否し、承認リクエストを `Denied` のままにする。
- 交換済みの `auth_req_id`、起票したクライアントと異なるクライアントか別のテナントの交換、未知の `auth_req_id` を受けた場合、OAuth2 は、400 と `invalid_grant` で拒否する。
- 同じ `Approved` の `auth_req_id` を並行に交換された場合、OAuth2 は、ちょうど一方だけにトークンを返し、もう一方を 400 と `invalid_grant` で拒否する。
- 承認の後に Agent が停止されたか無効化されたか、所有者がオフボードされたか、User が `Active` でなくなった `auth_req_id` の交換を受けた場合、OAuth2 は、400 と `invalid_grant` で拒否し、トークンを発行しない。
- **例**：EX-OAUTH2-042-01、EX-OAUTH2-042-02、EX-OAUTH2-042-03、EX-OAUTH2-042-04、EX-OAUTH2-042-05、EX-OAUTH2-042-06、EX-OAUTH2-042-07、EX-OAUTH2-042-08

### 本人による承認の判断

#### REQ-OAUTH2-043 承認リクエストを判断できるのは対象ユーザー本人のステップアップ認証済みセッションだけである

- 本人が保留中の承認リクエストを一覧したとき、OAuth2 は、200 と、本人宛の期限内の `Pending` の承認リクエストだけを、クライアントの表示名、Agent の名前、要求したスコープ、`authorization_details`、`binding_message` とともに返す。
- 本人がステップアップ認証を経たセッションで本人宛の期限内の `Pending` の承認リクエストを承認したとき、OAuth2 は、204 を返し、承認リクエストを `Approved` にし、`BackchannelAuthApproved` を発行する。
- 本人がステップアップ認証を経たセッションで本人宛の期限内の `Pending` の承認リクエストを拒否したとき、OAuth2 は、204 を返し、承認リクエストを `Denied` にし、`BackchannelAuthDenied` を発行する。
- セッションが未認証か認証の途中の場合、OAuth2 は、401 と `authentication_required` で拒否する。
- ステップアップ認証の有効期間を過ぎたセッションで判断を受けた場合、OAuth2 は、403 と `step_up_required` で拒否し、承認リクエストを `Pending` のままにする。
- CSRF トークンが一致しない判断を受けた場合、OAuth2 は、403 と `csrf_failed` で拒否し、承認リクエストを `Pending` のままにする。
- 他人宛か存在しない承認リクエストの判断を受けた場合、OAuth2 は、403 と `access_denied` で拒否する。
- 期限を過ぎたか `Pending` でない承認リクエストの判断か、`approve` でも `deny` でもない判断を受けた場合、OAuth2 は、400 と `invalid_request` で拒否し、記録済みの判断を上書きしない。
- **例**：EX-OAUTH2-043-01、EX-OAUTH2-043-02、EX-OAUTH2-043-03、EX-OAUTH2-043-04、EX-OAUTH2-043-05
