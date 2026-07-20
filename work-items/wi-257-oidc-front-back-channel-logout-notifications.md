---
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-19
depends_on: []
change_kind: feature
initial_context:
  scl:
    OAuth2:
      - models.OAuth2Client
      - models.ClientSession
      - models.LogoutNotification
      - models.LogoutNotificationState
      - models.FrontChannelLogoutTarget
      - models.DiscoveryDocument
      - interfaces.FrontChannelLogout
      - interfaces.BackChannelLogout
      - interfaces.CheckSessionIframe
    Jobs:
      - models.JobKind
  source:
    - backend/oauth2/domain/client.go
    - backend/oauth2/usecases/admin_clients.go
    - backend/oauth2/usecases/exchange_code.go
    - backend/oauth2/handlers_http/end_session_handler.go
    - backend/oauth2/handlers_http/discovery_handler.go
    - backend/jobs/domain/job.go
    - backend/jobs/usecases/handler_registry.go
    - backend/cmd/idmagic-worker/worker.go
  tests:
    - backend/oauth2/usecases
    - backend/oauth2/handlers_http/end_session_hint_test.go
  stop_before_reading:
    - backend/saml
    - backend/wsfederation
affected_spec:
  - { context: OAuth2, kind: model, element: OAuth2Client }
  - { context: OAuth2, kind: model, element: ClientSession }
  - { context: OAuth2, kind: model, element: LogoutNotification }
  - { context: OAuth2, kind: model, element: FrontChannelLogoutTarget }
  - { context: OAuth2, kind: model, element: DiscoveryDocument }
  - { context: OAuth2, kind: interface, element: FrontChannelLogout }
  - { context: OAuth2, kind: interface, element: BackChannelLogout }
  - { context: OAuth2, kind: interface, element: CheckSessionIframe }
  - { context: OAuth2, kind: standard_requirement, standard: OpenIDConnectFrontChannelLogout, requirement: OIDC-FRONTCHANNEL-IFRAME }
  - { context: OAuth2, kind: standard_requirement, standard: OpenIDConnectBackChannelLogout, requirement: OIDC-BACKCHANNEL-LOGOUT-TOKEN }
  - { context: Jobs, kind: model, element: JobKind }
---

# OIDC Front-Channel / Back-Channel Logout の通知配送を実装する

## Motivation
[[wi-28-session-management-and-oidc-logout-completion]] の T004/T005 で、session
revoke に伴う refresh token family の失効と `/end_session` の `id_token_hint`
検証は実運用相当まで完成した。しかし、接続済み RP (Relying Party) へ「ユーザーが
ログアウトしたこと」を伝播する OpenID Connect Front-Channel Logout 1.0 /
Back-Channel Logout 1.0 は未実装であり、`spec/contexts/oauth2.yaml` の
`standards.OpenIDConnectFrontChannelLogout` / `OpenIDConnectBackChannelLogout` は
既に `adoption: required` として宣言済みである (wi-28 T001)。宣言した標準と実装の
不一致を解消するため、通知配送を実装する。

Keycloak / Okta / Google 相当の IdP では、ユーザーが idmagic からログアウトした際に
接続先アプリ (RP) 側のセッションも連動して終了することが期待される。これが無いと、
idmagic 上ではログアウト済みでも RP 側では認証済みのまま残り続ける。

## Scope
以下は wi-28 の T001 (SCL-first) で既に `spec/contexts/oauth2.yaml` /
`spec/contexts/jobs.yaml` に追加・コミット済みであり、本 WI はそれらに対する
Go 実装を担当する (追加の SCL 変更が必要になった場合のみ `scl-change` に戻る)。

- `models.ClientSession` — sid が発行結果を渡した RP (client_id) の参加記録。
- `models.LogoutNotification` / `models.LogoutNotificationState` /
  `state_machines.LogoutNotificationLifecycle` — back-channel logout 配送の
  outbox 行とその状態機械。
- `models.FrontChannelLogoutTarget` — front-channel logout の iframe target。
- `models.OAuth2Client` の `backchannel_logout_uri` /
  `backchannel_logout_session_required` / `frontchannel_logout_uri` /
  `frontchannel_logout_session_required`。
- `models.DiscoveryDocument` の `frontchannel_logout_supported` /
  `frontchannel_logout_session_supported` / `backchannel_logout_supported` /
  `backchannel_logout_session_supported` / `check_session_iframe`。
- `interfaces.FrontChannelLogout` (internal) / `interfaces.BackChannelLogout`
  (internal) / `interfaces.CheckSessionIframe` (public)。
- `spec/contexts/jobs.yaml` の `models.JobKind.backchannel_logout_delivery`。

Go 実装スコープ:
- 管理者向けクライアント編集 (`RegisterClient` / `UpdateAdminOAuth2Client` /
  admin client handler) に上記 4 メタデータフィールドの CRUD を追加する。
- token 発行成功時 (`ExchangeCodeForToken`、authorization_code グラント) に
  `ClientSession` を upsert する。
- ローカル revoke 確定後 (`/end_session` の local logout、wi-28 T005 が実装した
  経路) に、対象 sid の `ClientSession` から `backchannel_logout_uri` 登録済み
  RP ごとの `LogoutNotification` を作成し、Jobs (`kind=backchannel_logout_delivery`)
  へ enqueue する。
- logout token signer (`iss`, `sub`, `aud`, `iat`, `jti`,
  `events: {http://schemas.openid.net/event/backchannel-logout: {}}`, `sid`) を
  実装する。
- `BackChannelLogout` job handler (Jobs 経由の retry、2xx=成功、その他/timeout/
  接続失敗=再試行) を実装し `cmd/idmagic-worker` に登録する。
- `FrontChannelLogout` (iframe target 一覧の算出) を実装し、`/end_session`
  応答へ埋め込む。
- `CheckSessionIframe` (静的ページ、`session_state` 相関アルゴリズムは実装しない
  — ADR-127 決定8) を実装する。
- `LogoutNotification` の状態遷移を監査可能にする (配送成功/失敗の追跡)。

## Out of Scope
- CAEP / Shared Signals。別 WI で扱う (wi-28 と同じ整理)。
- access token の即時失効 (denylist)。ADR-127 決定7 のとおり scope 外。
- `check_session_iframe` の `session_state` salted hash 相関アルゴリズム
  (ADR-127 決定8、Draft 28 のため `adoption: optional`)。
- SAML / WS-Federation の Single Logout。本 WI は OIDC のみを扱う。

## Plan
- 設計判断は [[ADR-127-oidc-session-binding-and-logout-propagation]] に既に
  記録済み (決定5: back-channel 配送は Jobs 経由、決定6: front-channel は配送
  保証なしの計算結果、決定8: check_session_iframe は最小実装)。本 WI では
  ADR-127 と矛盾しない実装のみを行い、新規の設計判断が必要になった場合のみ
  ADR-127 に追記するか新規 ADR を起こす。
- `LogoutNotification` の SCL モデルは `sub` (対象ユーザー) を持たない
  (session 状態の複製を避けるため)。しかし logout token は `sub` claim を
  必須とする (`OIDC-BACKCHANNEL-LOGOUT-TOKEN`)。ワーカープロセス内でテナント別
  issuer やユーザーを再解決する複雑さを避けるため、`sub` と `iss` は
  `LogoutNotification` 作成時点 (HTTP リクエストコンテキスト内、
  `tenancy.Issuer`/session 解決が可能な時点) に Jobs の job params
  (`{"notification_id":..., "sub":..., "iss":...}`) として運ぶ。job params は
  Jobs 側の永続化にそのまま乗るため、追加の永続化は不要。
- `logout_token_jti` は `LogoutNotification` 作成時に一度だけ生成し、retry の
  たびに再生成しない (同一論理配送に対し複数の jti が生まれることを避ける)。
- `ClientSession` は index のみを持つため、front/back-channel 双方の対象解決は
  `ClientSession.ListBySid(sid)` → 各 `client_id` の `OAuth2Client` を引いて
  `backchannel_logout_uri`/`frontchannel_logout_uri` の有無で振り分ける。
- 永続化は `LogoutNotification` も `ClientSession` も memory + postgres の
  両方を実装する (`refresh_tokens.sid` と同じ理由: 通知配送はプロセス再起動を
  跨いで再試行される必要があり、Jobs 自体が postgres 永続化されている以上、
  参照先の `LogoutNotification` もメモリのみでは再起動後に追跡不能になる)。

## Tasks
- [ ] T001 [Client Metadata] `domain.OAuth2Client` に4フィールドを追加し、
      `RegisterClient` / `UpdateAdminOAuth2Client` / admin client handler /
      memory・postgres adapter (`clients` テーブルへのカラム追加) で
      CRUD できるようにする。
- [ ] T002 [ClientSession] `domain.ClientSession` + `ports.ClientSessionStore`
      (Upsert/ListBySid) を追加し、memory・postgres 両adapterを実装する。
      `ExchangeCodeForToken` (authorization_code グラント成功時、sid が
      non-nil のときのみ) から upsert する。
- [ ] T003 [LogoutNotification] `domain.LogoutNotification` /
      `LogoutNotificationState` + `ports.LogoutNotificationStore`
      (Save/FindByID) を追加し、memory・postgres 両adapterを実装する。
      local logout (wi-28 T005 の `end_session_handler.go` 経路) から、
      対象 sid の `ClientSession` を引いて `backchannel_logout_uri` 登録済み
      RP ごとに `LogoutNotification` を作成し、Jobs へ enqueue する use case
      を実装する。
- [ ] T004 [Logout Token] logout token signer (`iss`/`sub`/`aud`/`iat`/`jti`/
      `events`/`sid`) を実装し、`BackChannelLogout` job handler
      (`kind=backchannel_logout_delivery`) を `cmd/idmagic-worker` に登録する。
      2xx=Delivered、それ以外/timeout/接続失敗は Jobs の attempts/max_attempts
      に応じて Pending (再試行) または Failed (dead-letter) に遷移させる。
- [ ] T005 [Front-Channel] `FrontChannelLogout` use case
      (対象 sid の `ClientSession` から `frontchannel_logout_uri` 登録済み
      RP の iframe target 一覧を算出、`frontchannel_logout_session_required`
      なら iss/sid クエリパラメータを付与) を実装し、`/end_session` の応答へ
      埋め込む。
- [x] T006 [Session Management] `CheckSessionIframe` (`GET /session/check`) —
      RED: `TestCheckSessionIframe_noSession_respondsChanged` /
      `TestCheckSessionIframe_validSession_respondsUnchanged` を先に 404 で
      fail 確認 (`backend/oauth2/handlers_http/check_session_iframe_handler_test.go`)
      → GREEN (`check_session_iframe_handler.go`)。静的ページ + 現在の browser
      cookie が有効な LoginSession に解決できるかどうかだけを埋め込んで返す
      最小実装 (ADR-127 決定8)。`d.AuthnResolver.Resolve` の結果 (nil または
      `AuthenticationPending`) を fail-safe 側 ("changed") に倒す。
      route を `backend/oauth2/handlers_http/routes.go` に登録し、
      `TestAssembledRoutesMatchGeneratedOpenAPI` の `GET /session/check` 差分を
      解消した (T007 verification の一部を前倒しで満たす)。wi-56 のブランチ作業中に
      発見した SCL/実装 drift の修正として先行実装。
- [ ] T007 [Verify] 複数 RP への配送、一時的配送失敗からの再試行、
      max_attempts 到達による dead-letter、同一 `LogoutNotification` の
      retry を跨いだ jti 不変性、ワーカー再起動後の配送継続を検証する。
      `GET /session/check` の route 契約差分は T006 で解消済み。

## Verification
- `just yaml-check`
- `just test-go` / `just lint-go` / `just build-go`
- 手動: `backchannel_logout_uri` を登録した test RP に対し、session revoke で
  logout token が POST されることを確認する。
- 手動: `frontchannel_logout_uri` を登録した RP への iframe が `/end_session`
  応答に含まれることを確認する。

## Risk Notes
`LogoutNotification`/`ClientSession` を postgres へ正しく永続化しないと、
ワーカー再起動時に配送中の通知を追跡できなくなる (Jobs 自体は永続化されるが
参照先の notification 行が失われると job params だけでは復元できない属性
(`target_uri` 等) が欠落する)。

logout token 配送は RP という外部境界への outbound HTTP であり、RP 側の
応答本体を解釈しない (2xx/非2xx の判定のみ) ため、未信頼入力のパースリスクは
無い。id_token_hint 検証 (wi-28 T005) と同様の理由で fuzz/property test は
本 WI でも採用しない。

`sub`/`iss` を job params (JSONB) 経由で運ぶ設計は、Jobs の汎用性
(ADR-117: Jobs は generic durable queue) を維持しつつ、ワーカー内でのテナント別
issuer 再解決という複雑さを避けるための選択。将来 Jobs 側で機微情報を job params
に含めることが問題になった場合は再検討する。
