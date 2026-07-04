---
id: idp-wi-90-account-security-notification-emails
title: "アカウントのセキュリティ通知メール (サインイン / 認証情報変更アラート) を導入する"
created_at: 2026-07-03
authors: ["tn"]
status: pending
risk: medium
---

# Motivation
idmagic は SMTP email sender ([[wi-6-real-email-sender-adapter]]) と認証
イベント基盤 ([[wi-44-authentication-event-store-and-search]]) を持つが、
メール送信は password reset / email 変更検証に限られ、セキュリティ上の変化を
ユーザに知らせる通知が無い。代表的な IdP / アカウントサービスは security
notification を標準で送る:

- Google: 新しいデバイスからのサインイン通知。
- Okta / Entra: サインイン / factor 変更 / パスワード変更の通知メール。

新デバイスからのサインインや、パスワード / MFA / 連絡先の変更をユーザに通知
することは、アカウント乗っ取りの早期検知に直結する。本 WI は既存の domain
event を購読して best-effort でメール通知を送るディスパッチャと、ユーザの
opt-out 設定を追加する。

# Scope
- **decision**:
  - 新規 ADR: 通知対象イベント (新デバイスからの sign-in / password 変更 / TOTP・WebAuthn の追加・削除 / email・recovery 連絡先の変更 / session 失効) と、 配信方針を決める。配信は既存 email sender / outbox 経由の fire-and-forget で、 失敗しても認証を阻害しない。本文にトークン / 機微を載せず PII を最小化 (概略の IP / UA / 時刻のみ)。ユーザ / テナント設定で opt-out 可否を持つ。
- **scl**:
  - §3.3 interfaces: GetNotificationPreferences / UpdateNotificationPreferences (self) を追加する。通知自体は既存イベントの subscriber として実現し、 新規 interface は最小に留める。
  - §3.2 models: NotificationPreferences を追加する。
  - §3.4 states/events: AccountSecurityNotificationSent を追加する (どの種別の通知を送ったかを監査に残す。本文は残さない)。
  - §3.7 permissions: preference の参照 / 更新は actor.sub に固定する。
- **go**:
  - notification dispatcher (domain event subscriber → email sender) を追加する。 「新デバイス」判定は sign-in activity / 既知 session ([[wi-20-authentication-event-history]] / [[wi-28-session-management-and-oidc-logout-completion]]) を参照する。通知テンプレートを用意する。
  - preference store (port + memory + postgres + migration) を追加する。
- **http**:
  - account portal に通知設定の取得 / 更新エンドポイントを追加する (認証必須・self 固定)。
- **ui**:
  - AccountSecurityPage に通知設定トグル (種別ごと / 一括 opt-out) を追加する。
- **documentation**:
  - README の account portal / 通知節に対象イベントと opt-out を追記する。

# Out of Scope
- SMS / push 通知 (外部 gateway 依存)。
- アプリ内通知センター / 通知履歴の閲覧 UI。
- ダイジェスト / サマリメール。
- admin がテンプレートを編集する機能 (ブランディング / 別 WI)。
- 位置情報 (GeoIP) に基づく詳細なリスクスコアリング。

# Verification
- `just test-go`
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just lint-ui`
- `just build-ui`
- 手動: 新デバイス相当でサインイン → 通知メールが届く。パスワード / TOTP を 変更 → 対応する通知が届く。opt-out した種別は届かないことを確認する。
- 手動: email sender を失敗させても認証 / 変更操作自体は成功することを確認する。

# Risk Notes
通知は「送りすぎるとノイズ、送らなすぎると無意味」でチューニングが要る。加えて
本文への機微漏洩 / 通知自体を使ったスパム送信 (enumeration・メール爆撃) が
リスク。best-effort・opt-out・PII 最小化・本文に機微を載せない方針をテストで
担保し、新デバイス判定は既存 sign-in activity に載せて誤検知を抑える。
