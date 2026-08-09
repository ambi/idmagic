---
status: accepted
authors: [tn]
created_at: 2026-08-09
---

# ADR-157: 認証系 protocol endpoint に汎用 rate limit を導入する

## コンテキスト

ADR-029 の login throttle は account/password 試行にのみ効く防御で、`/authorize` `/token` `/par`
`/device_authorization` `/api/auth/password_reset/*` など他の認証系 endpoint には rate limit が無い。
wi-27 の再検討で CAPTCHA / bot challenge は今回スコープ外と判断し（別 WI で再検討）、endpoint 単位の
rate limit のみを対象にする。永続化は ADR-139 が確立した「揮発性状態は全て PostgreSQL、Valkey は
使わない」方針に従う。

## 決定

1. **RateLimiter port を `backend/shared/ratelimit` に新設する。** login throttle 固有ではない横断的
   な技術 capability として、`EnvelopeCrypto`（`backend/shared/security`）と同じ位置づけで置く。
   `Allow(ctx, tenantID, policyID, key, now) (Result{Allowed, RetryAfterSeconds}, error)` の単一
   メソッドで、fixed-window カウンタを 1 回の呼び出しで判定・加算する（login throttle の
   `RecordFailure` と異なり、成功/失敗を問わず全リクエストを消費する）。
2. **アルゴリズムは fixed window とし、token bucket / sliding window は採らない。** 「burst」は window
   内の最大リクエスト数として表現する。`login_throttle_counters` と同じ CAS 実装
   (`UPDATE ... WHERE window_expires_at <= now() ... RETURNING` / `INSERT ... ON CONFLICT`) を踏襲でき、
   実装・検証コストが小さい。
3. **route ごとの bucket key**（wi-27 Scope）: `/token` は `client_id+IP`、`/authorize` `/par` は
   `IP+client_id`、`/device_authorization` は `client_id+IP`、password reset は `identifier_hash+IP`。
   `tenant_id` は key に含めず別列として保持する（決定 4）。
4. **永続化は PostgreSQL 単一（memory はテスト用）、Valkey は使わない。** `endpoint_rate_limit_counters`
   テーブルを追加する。`tenant_id` は opaque key の高頻度 fail-closed lookup として保持する（ADR-082
   §4 / ADR-139 §8 の既存例外に合流）。`UNLOGGED` とする — 全リクエストを数える高 churn カウンタで、
   failover で消えても直後にウィンドウが再構築されるだけで防御が後退しない点が、消失が防御後退に
   直結する `login_throttle_counters` や denylist（いずれも LOGGED）と異なる。
5. **fail-closed を全 policy 一律とする。** login throttle (ADR-077) と揃え、閾値超過判定ができない
   場合はリクエストを拒否する。5 endpoint はいずれも PostgreSQL が既存の hard dependency であり
   （クライアント・トークンの読み書きで既に必須）、fail-closed にしても新しい単一障害点は増えない。
6. **閾値は環境変数で運用者が調整可能にする。** login throttle の閾値は現状ハードコード
   (`backend/cmd/idmagic/server.go`) だが、本 rate limiter は運用中の調整余地を要件とする（wi-27 Risk
   Notes）。デフォルト値と env 変数名は `ARCHITECTURE.md` に記載する。
7. **429 応答は新規 SCL `RateLimitedError` (status 429) + `Retry-After` ヘッダで統一する。** 既存
   login throttle が返す独自 `too_many_requests` body（現状 SCL 未宣言）もこの `RateLimitedError` に
   合わせて宣言・実装を揃える。

## 却下した代替案

- **CAPTCHA / bot challenge の同時導入**: wi-27 再検討でスコープ外と判断（別 WI で起票）。
- **token bucket / sliding window アルゴリズム**: 精度は上がるが実装・検証コストが
  `login_throttle_counters` の CAS パターンから逸脱し、今回のスコープに見合わない。
- **Valkey 等、rate limit 専用の別ストアを新設する**: ADR-139 の「2 つ目の基盤を増やさない」方針に
  正面から反する。
- **login throttle の 2 軸 (account/IP 別 bucket) パターンをそのまま流用する**: 本 rate limiter は
  route ごとに単一の複合 key で足りる。多軸合成が要る呼び出し元は `Allow` を複数回呼べばよく、port を
  複雑化する理由がない。

## 影響

- **SCL**: `spec/contexts/authentication.yaml` / oauth2 context に `RateLimitedError` (429) を追加し、
  Authorize / Token / PAR / DeviceAuthorization / PasswordResetRequest / SubmitBrowserLogin の
  scenario に rate limit 拡張を追加する。
- **schema**: `infra/schema/postgres.sql` に `endpoint_rate_limit_counters` を追加する。
- **adapter**: `backend/shared/ratelimit/{ports,db_memory,db_postgres}` を新設する。
- **wiring**: oauth2 / authentication の該当 HTTP handler、bootstrap (`assembleMemory` /
  `assemblePostgres`)、`idmagic-worker` の `ephemeralSweepLoop` に追加する。
- **ARCHITECTURE.md**: 「Availability and shared state」の ephemeral 一覧、「Database design policy」
  §2 tenant_id retention classes、rate limit の閾値/env 一覧を同期する。
