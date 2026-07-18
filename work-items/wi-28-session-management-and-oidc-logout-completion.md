---
depends_on: [wi-253-postgresql-persistent-login-sessions]
status: pending
authors: ["tn"]
risk: high
created_at: 2026-06-20
change_kind: feature
initial_context:
  scl:
    Authentication:
      - models.LoginSession
      - interfaces.ListMySessions
      - interfaces.RevokeMySession
    OAuth2:
      - models.OAuth2Client
      - models.RefreshTokenRecord
      - models.EndSessionParameters
      - interfaces.EndSession
  source:
    - backend/oauth2/domain/client.go
    - backend/oauth2/domain/refresh_token.go
    - backend/oauth2/usecases/exchange_code.go
    - backend/oauth2/adapters/http/end_session_handler.go
  tests:
    - backend/oauth2/usecases/exchange_code_test.go
    - backend/oauth2/adapters/http/end_session_handler_test.go
  stop_before_reading:
    - backend/saml
    - backend/wsfederation
affected_spec:
  - { context: Authentication, kind: model, element: LoginSession }
  - { context: Authentication, kind: interface, element: ListMySessions }
  - { context: OAuth2, kind: model, element: OAuth2Client }
  - { context: OAuth2, kind: model, element: RefreshTokenRecord }
  - { context: OAuth2, kind: model, element: EndSessionParameters }
  - { context: OAuth2, kind: interface, element: EndSession }
---

# セッション管理と OIDC logout を実運用相当に完成させる

## Motivation
`/end_session` は存在するが、production IdP としては user/admin の
session inventory、個別 revoke、OIDC RP-Initiated Logout の厳密化、
Front-Channel Logout、Back-Channel Logout、Session Management 1.0 が足りない。

Keycloak / Okta / Google 相当では、ユーザーがデバイス・セッションを確認し、
管理者が全セッションを失効し、RP に logout を伝播できることが重要になる。

## Scope
- **decision**:
  - 新規 ADR: local session、browser session、client session、refresh token family、 OIDC logout notification の関係を定義する。`id_token_hint` 検証と client 解決の失敗時挙動も明記する。
- **scl**:
  - SessionRecord / ClientSession / LogoutNotification を追加する。
  - ListSessions / RevokeSession / RevokeUserSessions を admin/self interface として追加する。
  - BackChannelLogout / FrontChannelLogout / CheckSessionIframe を追加する。
- **go**:
  - [[wi-253-postgresql-persistent-login-sessions]] が提供する PostgreSQL session 正本を利用する。
  - `/end_session` で `id_token_hint` 署名・iss・aud・sub・sid を検証する。
  - RP ごとの backchannel_logout_uri / frontchannel_logout_uri を client metadata に追加する。
  - logout token を生成し、back-channel logout を retry 付きで送信する。
  - session revoke 時に refresh token family / device code / browser cookie を整合して失効する。
- **ui**:
  - account portal に active sessions / devices を表示する。
  - admin users detail に user sessions と revoke all を表示する。
- **documentation**:
  - README に OIDC logout 対応範囲、RP metadata、既知制約を書く。

## Out of Scope
- CAEP / Shared Signals。別 WI で扱う。
- device trust / managed device inventory。
- native app の platform-specific logout。

## Plan
- 本 WI に durable session repository まで含めると、DB migration、session lifecycle、
  token family、外部 RP 通知、account/admin UI が一つの変更単位に集中する。このため
  PostgreSQL-only session 基盤を [[wi-253-postgresql-persistent-login-sessions]] へ
  切り出し、本 WI はその完了後に OIDC session binding と logout propagation を実装する。
- WI-253 の方針により、ログイン済み session の正本・認証参照・一覧・失効は
  PostgreSQL に一本化する。Valkey に active session を複製せず、本 WI も cache
  invalidation protocol を追加しない。
- OAuth2 の authorization/refresh token に `sid`/session reference を持たせ、session revoke から refresh family と access-token denylist を終端化する。session 正本を OAuth2 client/RP ごとに複製しない。
- RP-Initiated Logout は `id_token_hint` の issuer/audience/signature/sid を検証して session を特定する。client_id fallback は既存互換として残すが、hint と client の不一致を拒否する。
- front-channel logout は browser iframe/redirect の到達失敗を許容し、back-channel logout は署名済み logout token (`sid`, events, jti, iat) と replay 防止を実装する。ローカル revoke を先に commit し、RP 通知失敗で復活させない。
- account portal の session inventory は [[ADR-042-end-user-account-portal-scope]] と [[ADR-043-account-portal-csrf-and-step-up]] に従い、他 session/global revoke を step-up 対象にする。
- `last_seen_at` は認証済みrequestごとに更新せず、WI-253のcoarse touch契約を利用する。
  RP notificationの送達結果はsession行へ埋め込まず、再試行可能なjob/outboxとして分離する。
  local revokeのDB transactionを先に確定し、通知失敗でsessionを復活させない。

## Tasks
- [ ] T000 [Dependency] [[wi-253-postgresql-persistent-login-sessions]] を完了し、
      PostgreSQL session正本、indexed list、冪等revoke、coarse touchを利用可能にする。
- [ ] T001 [SCL] oauth2 の sid、id_token_hint、front/back-channel logout
      metadata・interface・正常/境界/拒否scenarioを更新して再生成する。
- [ ] T004 [Token] ID/access/refresh token と refresh family に sid を伝播し、session revoke 時に refresh family/denylist が冪等に更新される use case を実装する。
- [ ] T005 [RP Logout] `id_token_hint` 検証を既存 end_session handler に追加し、client redirect validation と local logout を use case へ抽出する。
- [ ] T006 [Notifications] client metadata、logout token signer、jti replay guard、front/back-channel delivery を実装し、通知結果を監査する。
- [ ] T007 [Account API/UI] session list/current marker、個別/global revoke endpoint と step-up/CSRF 付き画面を追加する。
- [ ] T008 [Verify] 複数端末・複数RP、通知不能、logout token replay、期限切れ hint、redirect 攻撃、再起動後 revoke を検証する。

## Verification
- `just test-go`
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just lint-ui`
- `just build-ui`
- 手動: 複数ブラウザで login し、account portal から片方の session を revoke すると対象 browser だけが再認証要求になることを確認する。
- 手動: backchannel_logout_uri を持つ test RP に logout token が送信されることを確認する。

## Risk Notes
session、refresh token、browser cookie、RP notification の整合が崩れると
logout したつもりで生き残る経路ができる。状態遷移を SCL に寄せ、session
lifecycle の property test を厚くする。

active sessionをPostgreSQLとValkeyへ二重保存しないことで、最も危険なstale cache経路を
設計から除外する。性能上の懸念はWI-253のindexed queryと負荷ゲートで先に評価する。
`id_token_hint` とlogout tokenは認証・認可に関わる未信頼JWTであるため、署名、期限、
issuer、audience、subject、sid、client不一致、replayのproperty/fuzz test採用を
T005/T006着手時に再評価し、その判断をself-attestへ記録する。
