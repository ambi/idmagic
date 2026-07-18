---
status: accepted
authors: [tn]
created_at: 2026-07-19
---

# ADR-127: OIDC session binding と logout propagation の設計

## コンテキスト

[[wi-253-postgresql-persistent-login-sessions]] は Authentication bounded context の
`LoginSession` を PostgreSQL 単一正本にした。しかし OAuth2/OIDC 側は認可コード・
refresh token・ID Token のいずれにも browser session を指す値を持たず、`RevokeMySession`
/ `RevokeMyOtherSessions` で LoginSession を失効しても、発行済み refresh token は
生き残り、接続済み RP (Relying Party) はユーザーがログアウトしたことを知る手段が
無い ([[wi-28-session-management-and-oidc-logout-completion]])。

`/end_session` は既に存在するが `id_token_hint` を一切検証せず、Cookie が示す
session を無条件に失効するだけで、OIDC RP-Initiated Logout 1.0 が要求する
client 解決・redirect 検証との整合性チェックを欠く。Front-Channel / Back-Channel
Logout、Session Management 1.0 も未実装であり、接続済み RP へログアウトを
伝播する経路が無い。

本 WI は次を決定する: (1) browser session と OAuth2 token / RP をどう相関するか、
(2) `id_token_hint` の検証と client 解決失敗時の挙動、(3) session revoke から
refresh token family・RP 通知への伝播経路、(4) access token の扱い。

## 決定

1. **sid は `LoginSession.id` そのものであり、RP 間で単一・共通の値とする。**
   Keycloak / Okta と同様、OIDC の `sid` claim は「ある RP に対する session」ではなく
   「ブラウザの OP session」を指す。RP ごとに別の sid を発行すると、1 つのブラウザ
   session を複数の相関不能な断片として扱うことになり、session revoke から
   全 RP を辿れなくなる。`AuthorizationRequest.sid` は `authenticate_user` 完了時に
   `AuthenticationContext.session_id` から一度だけ伝搬し、`AuthorizationCodeRecord.sid`
   → `RefreshTokenRecord.sid` → `IdTokenClaims.sid` へそのまま引き継ぐ。session 状態
   そのものは OAuth2 側に複製しない (Authentication が単一正本のまま)。

2. **`ClientSession` は「どの RP に通知すべきか」を解決するための索引に限定する。**
   `(sid, client_id)` を identity とする軽量な参加記録のみを持ち、session の
   属性 (amr/acr/expires_at 等) は複製しない。目的は back/front-channel logout の
   配送先解決であり、認証状態の第二正本にしない。

3. **`RefreshTokenRecord.sid` で family を横断してユーザーの session 単位revoke を
   解決する。** 1 つのブラウザ session から複数 RP (複数 client_id、複数
   refresh token family) が生成されうるため、`family_id` 単位ではなく `sid` 単位で
   `RevokeToken` を一括適用する。Rotate 後も親の `sid` をそのまま引き継ぐため、
   ローテーション履歴に関わらず同一 session 由来の token を漏れなく失効できる。

4. **`id_token_hint` は本 OP が署名した ID Token だけを受理し、fail-closed で
   検証する。** 検証する項目は署名 (登録済み signing key と算法一致)・`iss`・
   `aud` (client_id パラメータが同時に与えられた場合は一致必須。矛盾する
   hint は `invalid_request` で拒否し、cookie 単独の session へフォールバックしない)・
   `sub`・`sid`。**`exp` は検証しない** — ログアウト時点で ID Token が期限切れに
   なっているのが RP 実装として通常であり、多くの実装 (Keycloak 等) も logout 目的
   では expiry を無視する。`id_token_hint` が無い場合は既存の `client_id` +
   browser cookie による解決を fallback として維持する (後方互換)。

5. **back-channel logout の配送は Jobs bounded context の durable job に委譲する
   (kind=`backchannel_logout_delivery`)。** `LogoutNotification` を outbox 行として
   ローカル revoke 確定後に作成し、配送の成否に関わらずローカルの
   session/refresh token 失効を取り消さない。これは [[wi-45-outbound-scim-provisioning]]
   が提案する `ProvisioningDelivery` と同型の「配送は非同期・冪等・retry 可能」
   という設計を踏襲するが、SCIM 側がまだ未実装であるのに対し本 WI は実装対象と
   する。二重の queue 基盤を作らないため、既存の Jobs (EnqueueJob/ClaimJobs/…) を
   そのまま使い、`JobKind` enum に値を 1 つ追加するに留める。

6. **front-channel logout は配送保証を持たない計算結果として扱う。** `FrontChannelLogout`
   は `/end_session` 応答へ埋め込む iframe target の一覧を都度算出する internal
   interface であり、job/outbox を介さない。RP 側 iframe の到達失敗は仕様上
   許容されるものであり、ローカル revoke の成否に影響させない。

7. **Access Token の即時失効 (denylist) は本 WI のスコープ外とする。** Access
   Token は RFC 9068 準拠の自己完結 JWT で TTL 600 秒であり、signature のみで
   検証する (introspection や denylist 参照を resource server 経路に追加していない)。
   session revoke から access token を即時失効させるには、全 access token 検証を
   ストア参照必須に変える大きな設計変更が必要になり、[[wi-253]] が採った
   「計測で必要性が示されるまで cache/consistency 機構を追加しない」という方針と
   矛盾する。本 WI では TTL 600 秒の残存露出を許容し、refresh token family の
   即時失効と RP への logout 通知で実務上十分な保護とする。将来 introspection
   ベースの access token 検証が必要になった場合は別 WI で扱う。

8. **`check_session_iframe` (OIDC Session Management 1.0) は最小実装に留める。**
   本仕様は Draft 28 のまま Final に到達しておらず、主要 IdP でも実装が割れている。
   discovery 広告と「現在の browser cookie が有効な LoginSession に解決できるか」
   だけを返す静的ページに限定し、`session_state` の salted hash 相関アルゴリズムは
   実装しない (`standards.OpenIDConnectSessionManagement` は `adoption: optional`)。

9. **admin 向け session 管理は self-service (`ListMySessions` 等, wi-253) と対で
   `TenantAdministrator` policy の姉妹 interface として追加する。** 新しい
   projection model を作らず既存の `AccountSession` 相当を admin 版
   (`SessionRecord`、current マーカー無し) として複製する。既存の
   `ListUserSignInActivity` (admin, `resource: User, id: input.user_id`) と同じ
   アクセス制御パターンを踏襲する。

## 却下した代替案

- **RP ごとに別々の sid を発行する。** 相関が RP 単位に分断され、1 つの
  ブラウザ session から複数 RP へのログアウト伝播に family/session の
  突き合わせテーブルが別途必要になる。OIDC の sid の意味論 (OP session を指す)
  にも反する。
- **`id_token_hint` の署名検証を省略し issuer/audience だけを見る。** 検証不能な
  hint を信用すると、攻撃者が偽の hint で client 解決を誤誘導し得る。既存の
  fail-closed な session 解決方針 ([[ADR-126]]) と整合させ、署名検証を必須にする。
- **`id_token_hint` 不一致時に hint を無視して cookie にフォールバックする。**
  一部実装はこの寛容な挙動を採るが、client_id と hint の食い違いを黙って
  受理すると、どの client の意図でログアウトしたかが曖昧になる。WI の Plan
  通り拒否を選ぶ。
- **access token 即時失効のための introspection 必須化。** 全 resource server
  呼び出しにストア参照を強制するのは、本 WI のスコープに対して過大な
  アーキテクチャ変更であり、[[wi-253]] の「計測されるまでキャッシュ/一貫性
  機構を追加しない」という判断と矛盾する。TTL 600 秒の残存露出を許容する。
- **back-channel logout 配送を専用の新しい非同期基盤で実装する。** 既存の
  Jobs bounded context がある以上、二重の queue/worker 基盤を作る理由がない。
  `JobKind` の allowlist に 1 値追加するだけで賄う。

## 影響

- `spec/contexts/authentication.yaml`: `models.SessionRecord` /
  `models.SessionRecordListResponse` を追加。`interfaces.ListSessions` /
  `interfaces.RevokeSession` / `interfaces.RevokeUserSessions` (`TenantAdministrator`)
  を追加。`events.SessionEnded` の説明を sid 経由の OAuth2 側伝播に合わせて改訂。
  管理者向け session scenario を追加。
- `spec/contexts/oauth2.yaml`:
  - `models.AuthorizationRequest` / `models.AuthorizationCodeRecord` /
    `models.RefreshTokenRecord` / `models.IdTokenClaims` に `sid` field を追加。
  - `models.OAuth2Client` に `backchannel_logout_uri` /
    `backchannel_logout_session_required` / `frontchannel_logout_uri` /
    `frontchannel_logout_session_required` を追加。
  - 新規 model: `ClientSession` / `LogoutNotification` /
    `LogoutNotificationState` / `FrontChannelLogoutTarget`。新規 state machine:
    `LogoutNotificationLifecycle` (states: `Pending` → `Delivered` | `Failed`)。
  - `interfaces.EndSession` に `id_token_hint` 検証の `requires` を追加。新規
    internal interface: `FrontChannelLogout` / `BackChannelLogout`。新規 public
    interface: `CheckSessionIframe`。
  - `models.DiscoveryDocument` に `frontchannel_logout_supported` /
    `frontchannel_logout_session_supported` / `backchannel_logout_supported` /
    `backchannel_logout_session_supported` / `check_session_iframe` を追加。
  - `standards`: `OpenIDConnectRPInitiatedLogout` に
    `OIDC-LOGOUT-ID-TOKEN-HINT` を追加。新規スタンダード:
    `OpenIDConnectFrontChannelLogout` / `OpenIDConnectBackChannelLogout`
    (adoption: required) / `OpenIDConnectSessionManagement`
    (adoption: optional、Draft 28 のため)。
  - 新規 glossary/event 語彙: `Deliver` / `Exhaust`。
- `spec/contexts/jobs.yaml`: `models.JobKind` に `backchannel_logout_delivery` を追加。
- `spec/scl.yaml`: `OAuth2` の `depends_on` に `Jobs` (`uses: JobRef`) を追加。
- Go 実装 (backend/oauth2, backend/authentication, backend/cmd) と account/admin UI
  は本 ADR のスコープ外で、[[wi-28-session-management-and-oidc-logout-completion]] の
  T004 以降で対応する。
