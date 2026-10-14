---
status: completed
archived_from: 9aee83f75695a593712e443fe25d8457c032acce
---

# 認証系 endpoint に rate limit を導入する

`/authorize` `/par` `/token` `/device_authorization` `/api/auth/forgot_password`
`/api/auth/login` に汎用の endpoint rate limit を追加した（ADR-157）。既存の per-account/per-IP
login throttle（失敗回数ベース）とは別の、成功/失敗を問わず全リクエストを消費する fixed-window
カウンタで、`backend/shared/ratelimit`（`ports`/`db_memory`/`db_postgres`）に
`EnvelopeCrypto` と同じ横断 technical capability として実装した。永続化は
[[wi-278-consolidate-ephemeral-state-into-postgresql-remove-valkey]] の方針どおり
PostgreSQL 単一（`endpoint_rate_limit_counters`、UNLOGGED、tx+`SELECT FOR UPDATE`）と
memory の 2 adapter のみで、Valkey は追加していない。SCL に `RateLimitedError` (429) と
`EndpointRateLimitPolicy` を追加し、既存 login throttle の scenario がドキュメント上
`AccessDeniedError` としていた誤りも実態 (429) に合わせて訂正した。HTTP 層は
`support.CheckRateLimit`/`WriteRateLimited`/`ExtractClientIP` の共有ヘルパに統一し、
login throttle 側もこの共有ヘルパへリファクタして重複を除いた。閾値は
`RATE_LIMIT_<POLICY>_MAX_REQUESTS`/`_WINDOW_SECONDS` env で運用者が調整できる
（login throttle の閾値は今回も引き続きハードコードのまま、既存挙動を変えていない）。
UI は LoginPage/ForgotPasswordPage/DevicePage の既存 Alert / `localizedErrorMessage`
パターンに `rate_limited` コードと `Retry-After` 由来の待ち時間表示を追加し、新規
コンポーネントは作らなかった。CAPTCHA / bot challenge は今回のスコープ外（Out of Scope
参照、別途必要になれば起票）。

全文は `git show 9aee83f75695a593712e443fe25d8457c032acce:<このファイルのパス>` で読む。
