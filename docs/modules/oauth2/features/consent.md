# 同意

## 概要

この文書は、利用者がクライアントへ与える同意の付与、参照、撤回の仕様である。

| 項目 | 内容 |
| --- | --- |
| 責務 | 既存の同意による同意画面の出し分け、本人による撤回、管理者による参照と撤回、テナントの境界 |
| 行為者 | ResourceOwner、本人、API トークンの発行者、テナント管理者 |
| 扱わないもの | 認可コードの発行は[認可](authorization.md)が扱う |

## モデル

同意は `(subject, client_id)` ごとに、付与済みのスコープの集合として永続化する。
Agent への代行の付与は、同じ同意の `(subject, agent_id)` の行であり、撤回、期限、テナントの境界、クォータの規則を共有する。
一つの行は `client_id` と `agent_id` のどちらか一方だけを持つ。

- **判断**：この単位にする理由は、[同意を subject と client_id の組ごとのスコープの集合として持つ](../decisions.md#同意を-subject-と-client_id-の組ごとのスコープの集合として持つ)。

## 状態遷移

### ConsentLifecycle

同意レコードのライフサイクル。GDPR Art.7(3) により Granted → Revoked が可能。撤回した同意も期限の切れた同意も、利用者が認可で改めて同意すると `Granted` に戻る。`Expired` は保存した状態ではなく、付与から 365 日の `expires_at` を過ぎたことの判定である。遷移の表の `GrantConsent` と `RevokeConsent` は遷移の契機の名前であり、それぞれ `ConsentGranted` と `ConsentRevoked` を発行する。

<!-- spec:state-machine backend/oauth2/testdata/ConsentLifecycle.statemachine.json -->

| State | Kind | Meaning |
| --- | --- | --- |
| Granted | initial | 付与済みスコープ集合が有効である |
| Revoked | — | 利用者か管理者が取り消した。改めて同意するまで認可には使えない |
| Expired | — | 付与から 365 日が経過した。改めて同意するまで認可には使えない |

| From | Event | Guard | To | Effects |
| --- | --- | --- | --- | --- |
| Granted | GrantConsent | — | Granted | スコープの集合と `expires_at` を更新する |
| Revoked | GrantConsent | — | Granted |  |
| Expired | GrantConsent | — | Granted |  |
| Granted | RevokeConsent | — | Revoked |  |
| Revoked | RevokeConsent | — | Revoked |  |
| Expired | RevokeConsent | — | Revoked |  |
| Granted | Expire | — | Expired |  |

| State | 認可での同意 | 同意の撤回 | 有効期間の経過 |
| --- | --- | --- | --- |
| Granted | → Granted | → Revoked | → Expired |
| Revoked | → Granted | → Revoked | 何もしない |
| Expired | → Granted | → Revoked | 何もしない |

<!-- /spec:state-machine -->

## 操作

### 利用者による認可での同意

#### REQ-OAUTH2-008 既存同意の有無に応じて同意画面を出し分ける

- 利用者が `Granted` で期限内の同意のスコープに含まれるスコープだけを認可で要求したとき、OAuth2 は、同意の画面を出さずに認可を続ける。
- 同意がないか、`Granted` でないか、要求したスコープを含まないか、`prompt=consent` か、`authorization_details` を含む認可を受けたとき、OAuth2 は、認可のリクエストを `ConsentPending` にし、同意の画面へ進ませる。
- 第一者のクライアントの認可を受けたとき、OAuth2 は、同意の画面を出さずに認可を続ける。
- 利用者が同意の画面で許可したとき、OAuth2 は、要求したスコープと 365 日後の `expires_at` で同意を `Granted` にし、`ConsentGranted` を発行し、認可を続けさせる。
- 利用者が同意の画面で拒否したとき、OAuth2 は、認可のリクエストを `Rejected` にし、`redirect_uri` へ `access_denied` を返させる。
- `prompt=none` で同意の画面が必要な場合、OAuth2 は、同意の画面を出さずに `redirect_uri` へ `consent_required` を返す。
- 認可のトランザクションを解決できない場合、OAuth2 は、401 と `transaction_unavailable` で拒否する。
- 認証済みのセッションの主体と認可のリクエストの主体が一致しない場合、OAuth2 は、401 と `authentication_required` で拒否する。
- **例**：EX-OAUTH2-008-01、EX-OAUTH2-008-02

### 本人による同意の撤回

#### REQ-OAUTH2-032 ユーザーは接続済みアプリの同意を自分で撤回できる

- 本人が接続済みのアプリを一覧したとき、OAuth2 は、200 と本人の `Granted` の同意だけを返す。
- 本人がクライアントの同意を撤回したとき、OAuth2 は、204 を返し、同意を `Revoked` にして `revoked_at` を記録し、`ConsentRevoked` を発行する。
- 本人が `Granted` の同意を撤回したとき、OAuth2 は、所属テナントの `consents` の使用量を 1 減らす。
- 本人が `Granted` ではない同意を撤回したとき、OAuth2 は、`consents` の使用量を変えない。
- 本人に同意のないクライアントの撤回を受けた場合、OAuth2 は、404 と `consent_not_found` で拒否する。
- **例**：EX-OAUTH2-032-01

#### REQ-OAUTH2-002 API トークン発行者は account 同意スコープで自分の同意だけを操作できる

- 本人の User に固定した API アクセストークンで同意を操作されたとき、OAuth2 は、`account:read` で本人の `Granted` の同意の参照を、`account:consents:write` で本人の同意の撤回を許可する。
- `account:read` だけのトークンで撤回を要求された場合、OAuth2 は、403 と `insufficient_scope` で拒否する。
- 別のテナントのトークンを受けた場合、OAuth2 は、401 と `invalid_token` で拒否する。
- 他の User の同意を名指しした撤回を受けた場合、OAuth2 は、本人の同意として見つからないものとして 404 と `consent_not_found` で拒否する。
- **例**：EX-OAUTH2-002-01、EX-OAUTH2-002-02、EX-OAUTH2-002-03

### 本人による Agent への代行の付与

#### REQ-OAUTH2-057 ユーザーは自分を代行してよい Agent を選んで付与できる

- 本人が付与できる Agent を一覧したとき、OAuth2 は、200 と、所属テナントの `Active` な Agent の識別子、名前、所有者の表示名を返す。
- 本人が `Active` な Agent を選び、スコープと期限を指定して付与したとき、OAuth2 は、201 と付与を返し、`(本人, Agent)` の同意を `Granted` にして `AgentDelegationGranted` を発行する。
- 既に付与のある Agent へ付与したとき、OAuth2 は、スコープ、期限、`granted_at` を置き換える。
- 期限を省いた付与を受けたとき、OAuth2 は、`expires_at` を 365 日後にする。
- 新しい付与か、`Revoked` の付与を置き換えるとき、OAuth2 は、所属テナントの `consents` の使用量を 1 足す。
- `consents` の使用量が上限に達している場合、OAuth2 は、422 と `quota_exceeded` で拒否し、付与を変えない。
- スコープが空か、期限が過去か 365 日より先の付与を受けた場合、OAuth2 は、400 と `invalid_request` で拒否する。
- 所属テナントに存在しないか `Active` でない Agent への付与を受けた場合、OAuth2 は、404 と `agent_not_found` で拒否する。
- Bearer か DPoP のアクセストークンで付与を受けた場合、OAuth2 は、403 と `access_denied` で拒否し、付与を変えない。
- **判断**：トークンで付与できると、`account:write` を持つ Agent が自分自身へ付与して代行の判定を無効にできる。
- **判断**：付与の単位はクライアントではなく Agent である。一体の Agent は複数のクライアントを持つので、クライアントごとに撤回を求めると、撤回したつもりの権限が別のクライアントに残る。

#### REQ-OAUTH2-058 ユーザーは Agent への代行の付与を一覧し、個別に撤回できる

- 本人が代行の付与を一覧したとき、OAuth2 は、200 と、本人の `Granted` の付与の Agent の識別子、名前、所有者の表示名、スコープ、期限、`last_used_at` を返す。
- 本人が Agent の付与を撤回したとき、OAuth2 は、204 を返し、付与を `Revoked` にして `revoked_at` を記録し、`AgentDelegationRevoked` を発行する。
- 本人が `Granted` の付与を撤回したとき、OAuth2 は、所属テナントの `consents` の使用量を 1 減らす。
- 本人が `Granted` ではない付与を撤回したとき、OAuth2 は、`consents` の使用量を変えない。
- 本人に付与のない Agent の撤回を受けた場合、OAuth2 は、404 と `consent_not_found` で拒否する。
- 本人が接続済みのアプリか、管理者が同意を一覧したとき、OAuth2 は、代行の付与を含めない。
- `account:consents:write` のないアクセストークンで撤回を受けた場合、OAuth2 は、403 と `insufficient_scope` で拒否する。

### 管理者による同意の管理

#### REQ-OAUTH2-059 管理者は所属テナントの代行の付与を参照できるが、付与も撤回も代行できない

- 管理者が代行の付与を一覧したとき、OAuth2 は、200 と、所属テナントの付与を利用者と Agent の昇順にページに分けて、`Revoked` と期限切れも含めて返す。
- OAuth2 は、管理者が代行の付与を作成するか撤回する操作を提供しない。
- 不正なページの指定を受けた場合、OAuth2 は、400 と `invalid_request` で拒否する。
- 管理者が代行の付与を一覧したとき、OAuth2 は、別のテナントの付与を含めない。

#### REQ-OAUTH2-031 管理者は所属テナントの同意を参照・撤回できるが付与は代行できない

- 管理者が同意を一覧したとき、OAuth2 は、200 と、所属テナントの同意をページに分けて返す。
- 管理者が利用者とクライアントの組の同意を取得したとき、OAuth2 は、200 と同意を返す。
- 管理者が同意を撤回したとき、OAuth2 は、204 を返し、同意を `Revoked` にして `revoked_at` を記録し、操作した管理者を `actor_user_id` に載せた `ConsentRevoked` を発行する。
- OAuth2 は、管理者が同意を作成するかスコープを広げる操作を提供しない。
- 不正なページの指定を受けた場合、OAuth2 は、400 と `invalid_request` で拒否する。
- 存在しない同意の取得か撤回を受けた場合、OAuth2 は、404 と `consent_not_found` で拒否する。
- **例**：EX-OAUTH2-031-01

#### REQ-OAUTH2-038 同意管理 API は別テナントの同意を公開しない

- 管理者が別のテナントの利用者とクライアントの組の同意の取得か撤回を要求した場合、OAuth2 は、所属テナントに存在しない同意として 404 と `consent_not_found` で拒否し、同意を返さない。
- **例**：EX-OAUTH2-038-01

## セキュリティ上の考慮

同意の管理（`admin:consents_manage`）は、`admin` ロールを持つ、有効かつ認証済みのユーザーが、所属テナントに対して行う。
管理者は同意を参照し撤回できるが、利用者に代わって付与することはできない。

## 実現方式

### アーキテクチャ

#### 同意

同意はクライアント単位や対話単位ではなく、`(subject, client_id)` ごとに付与済みスコープの集合として永続化する。クライアント単位のグラントでは、後から追加された高権限のスコープへ暗黙に拡張され、目的ごとの同意と矛盾する。毎回尋ねると同意疲れを招き、場当たり的な記憶機能を求める圧力が生じる。要求された全スコープが期限内かつ未失効のグラントですでに覆われている場合だけ同意 UI を省略し、新しいスコープがあれば差分だけを強調する UI を表示する。グラントは定期的な再同意の期待に合わせ、付与から 365 日で失効する。`prompt=consent` は既存グラントにかかわらず UI を強制する。同意の取り消しが影響するのは将来の認可だけである。同時にクライアントのリフレッシュトークンを失効させる処理は、リフレッシュトークンローテーションのファミリー失効を再利用する、明示的で別の操作とする。発行済みの短命なアクセストークンは自然な期限切れを待つ。
