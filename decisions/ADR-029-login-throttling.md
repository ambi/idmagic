---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-029: ログイン試行のスロットリングとユーザー名列挙対策

## コンテキスト

ロードマップの「ブルートフォース防御」項目を埋める。bundled common-password 辞書
(ADR-026) と外部漏洩データベース検査 (ADR-028) は **新規 / 変更** のパスワードに
作用し、**既存アカウントへのオンライン パスワード推測 / credential stuffing** には
何も効かない。NIST SP 800-63B-4 §5.2.2 は verifier に「consecutive failed
authentication attempts on a single account」のレート制限を要求しており、
本 ADR はその要件と現場慣行 (OWASP ASVS v4.0.3 §2.2.1) を満たす最小スライスを定義する。

SCL には `rate_limit_per_minute` policy kind と `ClientAuthFailureRateLimit`
(client_secret 失敗) / `AuthorizationCodeRedemptionFailureRateLimit` (code 交換失敗)
の語彙が既に予約されているが、**ログインエンドポイントへの enforcement は未実装**。
本 ADR でログイン経路の port と adapter、SCL annotation、ADR を同時に整える。

## 決定

per-account と per-IP を独立にカウントする二軸スロットリングを採用する (per-account: 10 失敗 /
15 分 → 15 分ロック、per-IP: 30 失敗 / 15 分 → 15 分ロック)。どちらか一方でも NIST §5.2.2 の
"consecutive failed authentication attempts" 要件を満たせないため両軸が要る — per-account のみでは
credential stuffing（1 アカウントあたり数試行で次へ移る攻撃）を捕まえられない。ユーザー名列挙対策
として、未存在 username でも固定 sentinel ハッシュで verify を回し、throttle counter は user 存在
チェックの前に username 文字列で increment する。永続ロックは採用しない — NIST が明示的に
"do not implement permanent lockouts" を推奨しており、攻撃者が任意の victim username を継続的に
失敗させて恒久ロックへ追い込む DoS を防ぐため、時間で解けるロックに統一する。クライアント IP は
デフォルトで `X-Forwarded-For` を信頼しない (`TRUSTED_FORWARDED_HOPS` opt-in) — 偽装による
per-IP throttle 回避のほうが設定ミスによる IP 誤認より致命的なため。

現在の設計は [`backend/authentication/ARCHITECTURE.md`](../backend/authentication/ARCHITECTURE.md)
の Login throttling セクションにある。

## 影響

- 新 port `LoginAttemptThrottle` (`tryAcquire` / `recordFailure` / `recordSuccess`)。
- 新 adapter:
  - `InMemoryLoginAttemptThrottle` (memory / テスト)
  - `ValkeyLoginAttemptThrottle` (本番、`INCR` + `EXPIRE` + `SET NX EX` で lock)
- `authentication-routes.ts` に IP 抽出 + throttle 配線 + sentinel verify。
- `LoginThrottled` event を `DomainEvent` 判別ユニオン / SCL `events` /
  `infra/event-routing.yaml` に追加。
- SCL `objectives.LoginThrottlePolicy` にしきい値を記録する。
- `TRUSTED_FORWARDED_HOPS` は実行環境固有の設定として実装側で管理する。
- セッションマネージャやパスワード検証ロジックは触らない（純粋にレイヤを増やす）。

## 却下した代替案

- **per-account のみで per-IP を持たない**: credential stuffing は 1 アカウント
  あたり 1〜数試行で次のアカウントへ移るため per-account では捕まらない。
- **永続ロック (failed attempts >= N で管理者が解除するまで)**: 自身を DoS
  できる脆弱性になる (攻撃者が任意の victim username を継続的に失敗させて
  恒久ロックに追い込む)。NIST も明示的に "do not implement permanent
  lockouts" を推奨。時間で解ける lockout に統一する。
- **`X-Forwarded-For` をデフォルトで信頼**: 偽装で per-IP を回避できる。
  プロキシ段数が運用ごとに違うため安全側に倒し、明示 opt-in にする。
- **カウンタを username ではなく sub で集計**: 未登録 username の試行が
  カウントされず、存在しないユーザの探索を許す。username 文字列で揃える。
- **CAPTCHA を同時導入**: 上記。スコープを切り、まずレート制限で NIST
  要件を満たす。
