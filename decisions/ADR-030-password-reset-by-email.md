---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-030: メールによるパスワードリセット (forgot-password)

## コンテキスト

これまでのパスワード周りの ADR は **既知の現パスワード** から始まる経路
（change-password / ADR-026 / ADR-027 / ADR-028）と、**生きているセッション**
での再保護（ADR-029）に閉じていた。「パスワードを忘れたユーザを救う経路」が
無く、初回パスワード入力ミスやデバイス紛失で永久にアカウントが失われる。

リセット経路は password-policy / password-history / breached-checker / sentinel
verify と同じ部品を再利用しつつ、認証されていない攻撃者の入口にもなるため、
**列挙対策**・**シングルユース トークン**・**TTL**・**email チャネルの抽象** を
ここで一括して決める。

将来は email verification / magic link / breach 通知 / step-up via email を
全て同じ `EmailSender` port に乗せる。本 ADR が port 設計の最初の使用点。

## 決定

32 バイト乱数トークンを発行し、DB には SHA-256 hash のみ保管する (流出時に再現不可)。TTL は
30 分 — OWASP "Forgot Password Cheat Sheet" / NIST の "short-lived" 条件と、email 確認の現実的な
所要時間の中間。`consume(token)` は原子的に削除するシングルユースとする。認証されていない攻撃者の
入口になるため anti-enumeration を徹底する: 要求エンドポイントは email の登録有無・verified 状態に
関わらず常に 204 を返し、email 送信は best-effort（送信失敗をユーザに返さない fail-open）とする。
`email_verified=true` の user にだけ送る — 検証されていない email に送ると self-registration 導入後
に他人の email を使った乗っ取り経路になる。新パスワードは change-password と同じ検証パイプライン
(履歴・漏洩チェック) を通す。

現在の設計は [`backend/authentication/ARCHITECTURE.md`](../backend/authentication/ARCHITECTURE.md)
の Password lifecycle セクションにある。

## 影響

- 新 port: `EmailSender`, `PasswordResetTokenStore`。
- 新 adapter: `ConsoleEmailSender`, `NoopEmailSender`,
  `InMemoryPasswordResetTokenStore`, `PostgresPasswordResetTokenStore`。
- 新 use case: `requestPasswordReset`, `resetPasswordWithToken`。
- 新 HTTP endpoint: `GET /forgot_password`, `POST /api/auth/forgot_password`,
  `GET /reset_password`, `POST /api/auth/reset_password`。
- 新 SPA page: `ForgotPasswordPage`, `ResetPasswordPage`。i18n キー追加 (ja/en)。
- 新 SCL: `PasswordResetTokenRecord` model, `PasswordResetRequested` event,
  `EmailSent` event, `RequestPasswordReset` / `ResetPasswordWithToken`
  interfaces, `objectives.PasswordResetTokenLifetime`。
- 新マイグレーション: `password_reset_tokens` テーブル。

## 却下した代替案

- **TTL を 24 時間に伸ばす**: ユーザビリティ向上は限定的で、流出時の窓が
  広がる。OWASP は分単位の短さを推奨。

- **email を本文に書く / sub を URL に載せる**: 列挙経路を作る。URL は
  token だけにし、サーバ側で sub を解決する。

- **fail-closed (送信失敗時に エラーを画面に返す)**: 列挙の手がかりを
  与え、SMTP 障害でフロー全体が停止する。リセットは「届かなければ再送」
  でユーザが回復できる経路にする。

- **password reset で email_verified を要求しない**: self-registration が
  実装された時、攻撃者が他人の email で登録して victim の password を奪える
  経路が開く。検証済みのみへ送る方針で先回りする。

- **token を DB に平文で保管**: 流出時に攻撃者がそのまま reset を実行できる。
  ハッシュ化は cost 0 で耐性を一段上げる。
