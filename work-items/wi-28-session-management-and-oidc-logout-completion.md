---
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
