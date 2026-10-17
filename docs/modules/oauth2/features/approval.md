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

- **判断**：承認を CIBA で実装する理由は、[Agent の操作に対する人間の承認を CIBA で実装する](../decisions.md#agent-の操作に対する人間の承認を-ciba-で実装する)。

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

## 実現方式

### アーキテクチャ

#### エージェント操作に対するバックチャネルでの人による承認（CIBA）

送金、データ削除、外部公開など重大な行為を始める自律型エージェントには、その場にいない人間による事前承認を要求できる。`POST /bc-authorize`（OpenID CIBA Core）により、認証済みクライアントは帯域外で承認リクエストを起票できる。`/token` の `urn:openid:params:grant-type:ciba` グラントは、人間が判断するまで要求を保留する。どの Agent にこの承認を義務付けるかはガバナンス層が決定し、OAuth2 モジュールは `AgentKind.Supervised` だけを理由にすべてのグラントを一律に拒否しない。

CIBA は別の認証方式ではなく、OAuth 2.0 上の承認機能としてモデル化し、同意やステップアップ認証を置き換えない。同意は長命な `(subject, client_id)` のスコープグラントのままであり、承認リクエストは 1 つの行為に対する短命な判断である。ステップアップ認証はバックチャネルフローに置き換えられず、承認対象の行為を保護する。アカウントポータルは判断を記録する前に再認証を要求する。

判断を保持するレコードは CIBA 固有の形にしない。OpenID AuthZEN の Access Request and Approval Profile が定義する、認可判断の前提条件を要求、追跡、充足、再評価するモデルに合わせ、Aggregate は UUID をキーとする `ApprovalRequest` とする。`auth_req_id` は SHA-256 の照合用ダイジェストだけを保存する 32 バイトのベアラーシークレットであり、`interval_seconds` と `last_polled_at` は転送方式に固有のフィールドである。判断と同時ポーリングを 1 つのストア境界で直列化するため、これらを同じ永続化レコードに置く。アカウントポータルは UUID で承認リクエストを指すため、人間向けインターフェースへベアラーシークレットは渡らない。

`Pending → Approved | Denied | Expired` と `Approved → Consumed` は一方向である。発行ではデバイスグラントの `Exchange` と同じ比較交換をストア上で行い、まだ `approved` のレコードだけを変更する。そのため、同時に行われた 2 回のポーリングが両方ともトークンを発行することはない。`/token` は未承認のすべての状態をフェイルクローズに扱う。`pending` のポーリングは `authorization_pending`、`interval` より速いポーリングは `slow_down`（`interval` に 5 秒を加算）、拒否は `access_denied`、期限切れは `expired_token`、消費済みリクエストの 2 回目の交換は `invalid_grant` を返す。

実装し、メタデータで広告する配信モードは `poll` だけである。`ping` は接続せず拡張点として残し、通知基盤が必要になる `push` は対象外とする。`user_code` は未対応として広告する。承認画面はすでに User の認証済みセッションとステップアップ認証の背後にあるため、その前へ 2 つ目の弱い共有シークレットを追加しても利点がない。リクエストの有効期間はデフォルトで 300 秒とし、正の `requested_expiry` は 600 秒以下だけを受け入れ、それを超える値または 0 以下の値は拒否する。ポーリング間隔には別の規約を導入せず、デバイスグラントの 5 秒および 5 秒ずつ増加する設定を再利用する。帯域外通知は、テンプレートキーを 1 個追加した既存のテナント上書き可能な通知カタログを再利用し、パスワードリセットやセキュリティ警告と同じ方法で人間へ承認リクエストを届ける。

承認する User は `/bc-authorize` で、`login_hint`（ユーザー名またはメールアドレス）と検証済みの `id_token_hint` のいずれか正確に 1 個から解決する。ヒントがない場合や複数ある場合は `invalid_request` とする。解決できないヒント、`active` ではない User、別テナントの User はすべて同じ `unknown_user_id` にまとめ、存在するアカウントの探索にエンドポイントを悪用できないようにする。生成したリクエストはその User にバインドし、判断にはアカウントセッション、ステップアップ認証、CSRF 検証を要求する。別の User のリクエストは一覧に表示せず、ID で直接指定しても拒否する。`binding_message` だけではどのリクエストかは分かっても操作内容が分からないため、承認画面にはエージェント、クライアント、要求されたスコープ、`authorization_details` を表示する。`authorization_details` の表示には RAR の人間向けレンダリングを再利用する。

承認リクエストは別の短命なストアではなく、他の揮発性 OAuth2 レコードと同じ `UNLOGGED` PostgreSQL テーブルおよびメモリアダプターへ保存し、共通の一時データ削除処理で消去する。1 種類のレコードのために 2 つ目のストレージ技術を導入しても、既存の規約にはない利点を得られないためである。
