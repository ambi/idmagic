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
