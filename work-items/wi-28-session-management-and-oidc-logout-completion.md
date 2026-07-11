---
depends_on: []
status: pending
authors: ["tn"]
risk: high
created_at: 2026-06-20
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
  - session repository を拡張し、session_id / sub / client_id / amr / acr / started_at / last_seen を保存する。
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
- 現行 `/end_session` は登録済み `post_logout_redirect_uri` と state の検証まで実装済みなので維持し、まず Authentication 所有の login session を永続的に列挙・個別/global revoke できる session record へ拡張する。
- OAuth2 の authorization/refresh token に `sid`/session reference を持たせ、session revoke から refresh family と access-token denylist を終端化する。session 正本を OAuth2 client/RP ごとに複製しない。
- RP-Initiated Logout は `id_token_hint` の issuer/audience/signature/sid を検証して session を特定する。client_id fallback は既存互換として残すが、hint と client の不一致を拒否する。
- front-channel logout は browser iframe/redirect の到達失敗を許容し、back-channel logout は署名済み logout token (`sid`, events, jti, iat) と replay 防止を実装する。ローカル revoke を先に commit し、RP 通知失敗で復活させない。
- account portal の session inventory は [[ADR-042-end-user-account-portal-scope]] と [[ADR-043-account-portal-csrf-and-step-up]] に従い、他 session/global revoke を step-up 対象にする。

## Tasks
- [ ] T001 [SCL] authentication の Session model/lifecycle と oauth2 の sid、id_token_hint、front/back-channel logout metadata・interface・scenario を更新して再生成する。
- [ ] T002 [Domain] session record に tenant/user/auth_time/client/sid/last_seen/revoked_at を定義し、current/other/global revoke の不変条件をテストする。
- [ ] T003 [Persistence] memory/PostgreSQL session repository と Valkey の短命 browser state の責務を分離し、既存 session を安全に移行する schema/query を追加する。
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
