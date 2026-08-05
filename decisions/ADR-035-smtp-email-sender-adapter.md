---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-035: 本番運用可能な EmailSender adapter (SMTP)

## コンテキスト

ADR-030 で `EmailSender` port と `ConsoleEmailSender` (stdout) /
`NoopEmailSender` (テスト) を導入したが、本番系の adapter は意図的に「別 ADR」
として保留した。結果として `bootstrap/server.go` が `ConsoleEmailSender` を
hardcode しており、idmagic を本番環境にデプロイしても forgot-password の
リセットリンクが実ユーザに届かない (`/api/auth/forgot_password` は 204 を返す
が、メールは サーバ stdout に出るだけ)。後続の email verification / breach
通知 / step-up via email も同じ port に乗る前提なので、production adapter が
無いままだと積み増した瞬間に複数機能が同時に死ぬ。

## 決定

`EmailSender` port の本番 adapter は SMTP のみを採用し、プロバイダ別 HTTP SDK は採用しない —
主要プロバイダ (SendGrid / Resend / Mailgun / Postmark / AWS SES) は全て SMTP relay を公式提供して
おり、SMTP 1 本で到達できる範囲に対して、プロバイダごとの依存・認証情報形・error 形の増加は
port の抽象に見合わない。認証は PLAIN 1 方式のみを、TLS (implicit / starttls) の下でだけ許可する
(CRAM-MD5 / LOGIN / OAUTHBEARER は非採用) — 複数 auth 方式のサポートは adapter を肥大化させる
一方、主要プロバイダはすべて PLAIN over TLS で足りる。送信失敗は ADR-030 の fail-open 方針を継承し、
use case には伝えず戻り値の bool で表現する。retry / queue は adapter 内に持たない — fail-open の
方針上、失敗の吸収は use case 側の「再送リンク要求」UX に委ねる。メール本文は CRLF/NUL 除去・
HTML エスケープ・件名 RFC 2047 エンコードで正規化し、利用者由来の文字列がヘッダ注入や内容注入の
経路にならないようにする。

現在の設計は [`backend/authentication/ARCHITECTURE.md`](../backend/authentication/ARCHITECTURE.md)
の Password lifecycle セクションにある。

## 影響

- 新ファイル: `idmagic/internal/adapters/notification/smtp_email_sender.go`
  (+ unit test)。
- `idmagic/internal/bootstrap/email.go` (新) で env → adapter 切替。
  既定は `console`、`smtp` 選択時は `SMTP_HOST` / `SMTP_FROM` を必須にする。
- 既存 `ConsoleEmailSender` / `NoopEmailSender` は dev / test 用として保持。
- 環境変数表 (`idmagic/README.md` の `### 設定`) に `EMAIL_SENDER` /
  `SMTP_HOST` / `SMTP_PORT` / `SMTP_USERNAME` / `SMTP_PASSWORD` / `SMTP_FROM`
  / `SMTP_TLS` / `SMTP_TIMEOUT_SECONDS` を追加。
- SCL: `EmailSender` port のシグネチャは無変更。`events.EmailSent` の
  wire 形式も無変更。component / interface / permission / objective は無変更。

## 却下した代替案

- **SendGrid / Mailgun / Resend / Postmark / AWS SES の REST API SDK を直接
  叩く**: SMTP 1 本で同じ仕事ができるため。`EmailSender` の port は単純な
  片方向送信なので REST 専用 SDK が提供する高度な機能 (テンプレート、変数
  置換、サブスクリプション管理) は使わない。

- **adapter 内に retry / outbox を入れる**: 冪等性キー設計と queue が必要に
  なる。fail-open + use case 側の再送導線で十分。

- **複数の SMTP auth 方式 (CRAM-MD5 / LOGIN / OAUTHBEARER) をサポート**:
  主要プロバイダはすべて PLAIN over TLS をサポートする。複数方式を許すと
  adapter が「どの方式を試すか」を決める必要があり複雑化する。

- **`SMTP_PASSWORD` をファイルから読む経路 (`SMTP_PASSWORD_FILE` 等)**: k8s
  Secret / Docker secret はファイル → env 変換で配布されるのが常で、adapter
  に持ち込む必要は無い。

- **display name 付き `From` をサポートする (`"idmagic" <…>`)**: parse の余地
  を残すと設定ミスで送信不能になりやすい。bare address で十分。display name
  が必要になったら別 env で受ける ADR を改訂する。
