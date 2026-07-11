---
depends_on: []
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-06-20
---

# 認証系 endpoint に rate limit / CAPTCHA / bot mitigation を導入する

## Motivation
ADR-029 の login throttle は account/password 試行に対する防御だが、
本番 IdP では `/authorize` `/token` `/par` `/device_authorization`
`/api/auth/*` `/api/auth/password_reset/*` の全体に bot / abuse 防御が必要になる。

Keycloak も brute force / reCAPTCHA / endpoint threat mitigation を持つ。
idmagic も protocol endpoint ごとの一般 rate limit、CAPTCHA の optional
step、bot 判定の拡張ポイントを持つべきである。

## Scope
- **decision**:
  - 新規 ADR: rate limit の key、閾値、fail-open/fail-closed、CAPTCHA を 要求する条件を定義する。CAPTCHA provider は adapter として扱い、SCL は capability だけを保存する。
- **scl**:
  - EndpointRateLimitPolicy / BotChallenge / BotChallengeVerified を追加する。
  - Authorize / Token / PAR / DeviceAuthorization / PasswordResetRequest に objective を追加する。
  - `bot_challenge_required` エラーを browser API 用に追加する。
- **go**:
  - RateLimiter port を追加し、memory / Valkey adapter を実装する。
  - `/token` は client_id + IP + tenant、`/authorize` は IP + tenant + client_id、 password reset は identifier hash + IP で制限する。
  - CAPTCHA provider port を追加し、初期 adapter は noop + hCaptcha/Turnstile のどちらか 1 つに絞る。
  - 429 には `Retry-After` を返し、OAuth endpoint は仕様上の error response と整合させる。
- **ui**:
  - login / password reset / device verification 画面で bot challenge が要求された場合だけ表示する。
  - 通常時は CAPTCHA を出さない。
- **documentation**:
  - README に rate limit の環境変数、Valkey 必須範囲、CAPTCHA provider 設定を書く。

## Out of Scope
- ML による bot スコアリング。
- WAF / CDN ルールの本番投入。
- account lockout policy の再設計。既存 login throttle は維持する。
- device fingerprinting の永続追跡。

## Plan
- [[ADR-029-login-throttling]] と [[ADR-077-shared-login-throttle-store-and-ephemeral-state-ha]] の既存 login throttle を捨てず、汎用 endpoint limiter へ抽象化する。認証失敗の user-key bucket と、要求到達時の IP/client/endpoint bucket は別 policy として合成する。
- token、authorize、passwordless開始、signup等の route ごとに bucket key、window、burst、fail mode を SCL objectives/config へ宣言する。`postgres_valkey` は Valkey の atomic script、memory は単一 process テスト実装とする。
- CAPTCHA は limiter の代替ではなく、browser route で soft threshold を超えた場合だけ要求する challenge adapter とする。token endpoint 等の machine flow は CAPTCHA を返さず OAuth error/429 と `Retry-After` で制限する。
- provider token は server-side で検証し、hostname/action/score/TTL を確認する。provider 障害時は protected browser operation を fail-closed としつつ、既に成立した session/token introspection まで停止させない。
- label/監査には raw username/IP/CAPTCHA token を残さず、既存 tenant-salted correlation hash と低 cardinality reason を再利用する。

## Tasks
- [ ] T001 [Inventory/SCL] 現行 login throttle の key/threshold/fail-closed 実装を棚卸しし、route 別 limiter policy、429/OAuth error、CAPTCHA challenge scenario と objective を追加して再生成する。
- [ ] T002 [Port] atomic `Allow/Consume/Reset` limiter port と challenge verifier port を定義し、clock、hashed subject、policy ID だけを渡す。
- [ ] T003 [Valkey] window/burst を1操作で判定する script と namespaced key/TTL を実装し、複数 replica 競合・Valkey 障害を integration test する。
- [ ] T004 [Memory] 同じ contract の deterministic memory adapter を追加し、既存 login throttle test を共通 contract test へ移す。
- [ ] T005 [HTTP] authorize/token/login/password-reset 等の Scope 記載 endpoint に limiter を配置し、`Retry-After`、OAuth error、proxy-aware client IP の信頼境界を実装する。
- [ ] T006 [CAPTCHA] provider adapter と browser transaction への challenge-required state を実装し、action/hostname/score/replay を検証する。
- [ ] T007 [UI] challenge-required 応答でのみ CAPTCHA を表示し、期限・provider error・再試行を login flow を失わず処理する。
- [ ] T008 [Observability/Verify] hashed key の監査、hit/provider-error metric を追加し、分散 load、IPv6/proxy spoofing、username enumeration、machine flow 非CAPTCHAを検証する。

## Verification
- `just test-go`
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just lint-ui`
- `just build-ui`
- 手動: `/token` に同一 client_id/IP で閾値超過リクエストを投げ、429 と Retry-After が返ることを確認する。
- 手動: login で閾値超過後に CAPTCHA が表示され、成功後に通常 flow へ戻ることを確認する。

## Risk Notes
rate limit は強すぎると正規ユーザーを止め、弱すぎると攻撃を止められない。
tenant override と metric を先に入れ、運用で閾値を調整できる形にする。
