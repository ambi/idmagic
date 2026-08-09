---
depends_on: []
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-06-20
---

# 認証系 endpoint に rate limit を導入する

## Motivation
ADR-029 の login throttle は account/password 試行に対する防御だが、
本番 IdP では `/authorize` `/token` `/par` `/device_authorization`
`/api/auth/*` `/api/auth/password_reset/*` の全体に abuse 防御が必要になる。

Keycloak も brute force / endpoint threat mitigation を持つ。
idmagic も protocol endpoint ごとの一般 rate limit を持つべきである。

CAPTCHA / bot challenge は今回は不要と判断し、Out of Scope とする
（別途必要になった時点で改めて WI を起票する）。

## Scope
- **decision**:
  - 新規 ADR: rate limit の key、閾値、fail-open/fail-closed を定義する（[[ADR-157-endpoint-rate-limit-policy]]）。
- **scl**:
  - EndpointRateLimitPolicy を追加する。
  - Authorize / Token / PAR / DeviceAuthorization / PasswordResetRequest に objective を追加する。
  - `rate_limited` エラー (429) を追加する。
- **go**:
  - RateLimiter port を追加し、memory / postgres adapter を実装する（[[wi-278-consolidate-ephemeral-state-into-postgresql-remove-valkey]] で Valkey は撤去済みのため、Valkey adapter は追加しない）。
  - `/token` は client_id + IP + tenant、`/authorize` は IP + tenant + client_id、 password reset は identifier hash + IP で制限する。
  - 429 には `Retry-After` を返し、OAuth endpoint は仕様上の error response と整合させる。
- **ui**:
  - login / password reset / device verification 画面で 429 応答を受けた場合、既存の Alert / `localizedErrorMessage` パターンに `rate_limited` コードと `Retry-After` 由来の待ち時間表示を追加する。新規コンポーネントは作らない。
- **documentation**:
  - README に rate limit の環境変数・閾値設定を書く。

## Out of Scope
- CAPTCHA / bot challenge（hCaptcha・Turnstile 等）の導入。今回は閾値超過を 429 で止めるところまでとし、soft challenge が必要になれば別 WI で再検討する。
- ML による bot スコアリング。
- WAF / CDN ルールの本番投入。
- account lockout policy の再設計。既存 login throttle は維持する。
- device fingerprinting の永続追跡。

## Plan
- [[ADR-029-login-throttling]] の既存 login throttle を捨てず、汎用 endpoint limiter へ抽象化する。認証失敗の user-key bucket（login throttle）と、要求到達時の IP/client/endpoint bucket（本 WI の rate limiter）は別 policy として合成する。
- token、authorize、passwordless開始、signup等の route ごとに bucket key、window、burst、fail mode を SCL objectives/config へ宣言する。永続化は [[wi-278-consolidate-ephemeral-state-into-postgresql-remove-valkey]] の ephemeral state 方針（PostgreSQL 単一依存、Valkey は使わない）に従う。postgres adapter は `login_throttle_counters` と同型の単文原子操作（`UPDATE ... WHERE window_expires_at <= now() ... RETURNING` / `INSERT ... ON CONFLICT`）、memory は単一 process テスト実装とする。
- fail-closed/fail-open は endpoint ごとに SCL で宣言する。閾値超過時は 429 + `Retry-After` を返し、OAuth endpoint は仕様上の error response と整合させる。
- label/監査には raw username/IP を残さず、既存 tenant-salted correlation hash と低 cardinality reason を再利用する。

## Tasks
- [x] T001 [Inventory/SCL] 現行 login throttle の key/threshold/fail-closed 実装を棚卸しした（port/adapter/schema/HTTP wiring/SCL 記述の現状を確認、`backend/authentication/ARCHITECTURE.md` の stale Valkey 記述を発見・修正）。SCL: `RateLimitedError` (429) と `EndpointRateLimitPolicy` (value_object) を oauth2.yaml に追加、authentication.yaml に published-language stub を追加。Authorize/PushAuthorizationRequest/Token/DeviceAuthorization/SubmitBrowserLogin/RequestPasswordReset に errors・`!context.rate_limited` requires を追加。既存 login throttle scenario の誤った `AccessDeniedError` を実態 (429) に合わせ `RateLimitedError` へ訂正し、新規 rate limit scenario (oauth2 系 endpoint 共通・login IP 単位・password reset identifier+IP 単位) を追加。`just check-scl` / `just check` / `just check-api-compat`（breaking change なし）green、`just scl-render` で派生物再生成済み。
- [x] T002 [Port] `backend/shared/ratelimit/ports.RateLimiter` に単一 `Allow(ctx, tenantID, policyID, key, now)` を定義（`EnvelopeCrypto` と同じ横断 technical capability の置き場所、ADR-157）。`RateLimitConfigs` で policy 別 `MaxRequests`/`WindowSeconds` を束ねる。login throttle の 2 軸 (account/IP) 合成と異なり、本 port は単一 key で足りるため `TryAcquire`/`RecordFailure`/`RecordSuccess` の 3 メソッドではなく `Allow` 1 つに単純化した（ADR-157 却下代替案）。
- [x] T003 [Postgres] RED: `TestRateLimiter`（4th リクエストで拒否）を `RateLimiter` 型未定義で fail 確認 → GREEN。`backend/shared/ratelimit/db_postgres` に `endpoint_rate_limit_counters` への tx + `SELECT FOR UPDATE` の read-modify-write（`login_attempt_throttle.go` と同型パターン、失敗回数ではなく全リクエストを消費）を実装。`TestRateLimiterConcurrentAllow`（MaxRequests=8 に対し 16 並行呼び出しで許可がちょうど 8 件、wi-278 の 16 並行実証と同型）で原子性を実証、`-race` green。`DeleteExpiredBatch` 実装、tenant 分離・window 失効を検証。`just test-go-package`（該当パッケージ）/ `just lint-go` / `just check-architecture` green。
- [x] T004 [Memory] RED: `TestRateLimiterAllowsWithinThresholdThenBlocks` 等 4 tests を `NewRateLimiter` 未定義で fail 確認 → GREEN。`backend/shared/ratelimit/db_memory` に fixed-window カウンタ実装、tenant/policy/key 分離・window 失効・未知 policy の fail-closed error を検証。`just test-go` (該当パッケージ) / `just lint-go` green。`architecture.yaml` に `shared-ratelimit-ports`/`shared-ratelimit-db-memory` を登録、`just check-architecture` green。
- [x] T005 [HTTP] `support_http.CheckRateLimit`/`WriteRateLimited`/`ExtractClientIP` を共有ヘルパとして追加し、Authorize/PushAuthorizationRequest(PAR)/Token/DeviceAuthorization(oauth2/handlers_http)・SubmitBrowserLogin(oauth2/handlers_http)・RequestPasswordReset(authentication/password/handlers_http) の 6 箇所に配線。key 合成は ADR-157 通り（token: client_id+IP、authorize/par: IP+client_id、device_authorization: client_id+IP、password_reset: identifier+IP、login: IP のみ）。`extractClientIP`/`writeLoginThrottled` は共有ヘルパへ委譲するよう既存 login throttle 側もリファクタ（`too_many_requests` → 新 `rate_limited` body に統一、ADR-157 決定7）。bootstrap: `backend/shared/ratelimit` を `notification` と同型の cross-cutting Module として追加（`Dependencies.RateLimit`、memory/postgres 双方の factory、`server.go` で `RATE_LIMIT_<POLICY>_MAX_REQUESTS`/`_WINDOW_SECONDS` env（既定値は `ARCHITECTURE.md` 表）から構築）。`ephemeral.go` に GC 登録。architecture.yaml に `shared-ratelimit-{ports,db-memory,db-postgres}` と依存 edge（bootstrap: composition_root、backend/http-support: published_interface）を追加。**実装中に発見・修正したバグ**: `CheckRateLimit` を単一 error 返却にすると `WriteRateLimited` が成功時 nil を返すため、呼び出し側の `if err != nil { return err }` が block 済みリクエストで発火せず後続処理が続行してしまう（応答は書き込み済みなのに handler が走り続ける）。`(blocked bool, err error)` の2値返却に直し、全呼び出し元を `if blocked, err := ...; err != nil { return err } else if blocked { return nil }` に統一。RED/GREEN で `TestCheckRateLimitBlockedWrites429WithRetryAfter`（err=nil でも blocked=true を要求）として回帰防止。`just build-go`/`lint-go`/`test-go`/`verify-go`(-race)/`check-architecture` 全て green。
- [x] T006 [UI] RED: `errorMessage.test.ts` に `rate_limited` の期待値（辞書値 + retry-after 文の付加）を先に追加し fail 確認 → GREEN。`AuthenticationAPIError` に `retryAfterSeconds`（body の `retry_after_seconds` から）を追加、`localizedErrorMessage` に `rate_limited` → `rateLimited` 辞書キーを追加し、`retryAfterSeconds` 指定時だけ locale ごとの完結した1文（「N秒後にもう一度お試しください。」/ "Try again in N seconds."）を付加する半端翻訳のない形に。LoginPage / ForgotPasswordPage(password reset) / DevicePage(device verification) の catch 節を `localizedErrorMessage(locale, cause.code, cause.message, cause.retryAfterSeconds)` に統一（新規コンポーネントは追加せず既存 Alert パターンのまま）。Go 側 `WriteRateLimited` に `message` フィールド（英語既定文）を追加し、localizedErrorMessage を通さない箇所でも `body.message` フォールバックが誤って networkError にならないようにした。`just typecheck-ui`/`lint-ui`/`build-ui`/`bun test`（i18n・auth-flow 該当分）green。
- [x] T007 [Observability/Verify] `endpoint_rate_limit_total{policy,outcome}` metric を追加（`RecordEndpointRateLimit`、`authn_login_throttle_total` と同型）し `CheckRateLimit` から allowed/rate_limited/store_unavailable を記録、`ARCHITECTURE.md` Metrics 表に追記。hashed key の監査は postgres adapter の `hashRateLimitKey` (SHA-256、`login_throttle_counters` の `hashThrottleIdentifier` と同型) が担保。IPv6/proxy spoofing 対策は `support.ExtractClientIP`（既存 login throttle と共有、`TRUSTED_FORWARDED_HOPS` 信頼境界）を再利用。username enumeration: password reset は既存の「email 存在有無に関わらず 204」を維持したまま rate limit のみ追加（レスポンス形状は変えていない）。手動 E2E: `RATE_LIMIT_AUTHORIZE_MAX_REQUESTS=2` でサーバ起動し `/realms/default/authorize` に3回リクエストして3回目が `429` + `Retry-After: 60` + `{"error":"rate_limited",...}` で拒否されることを確認。分散 load（複数 replica 間の共有カウンタ）は T003 の 16 並行 postgres テストが同一プロセス内並行性として証明、複数プロセス間の実地検証は未実施（postgres 共有ストアの原子性は tx+`SELECT FOR UPDATE` で担保されるため、ロジック上はプロセス数に依存しない）。`just verify`（全チェック並列実行）green。

## Verification
- `just test-go`
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just lint-ui`
- `just build-ui`
- 手動: `/token` に同一 client_id/IP で閾値超過リクエストを投げ、429 と Retry-After が返ることを確認する。
- 手動: login で閾値超過後に 429 と待ち時間メッセージが表示され、待機後に通常 flow へ戻ることを確認する。

## Risk Notes
rate limit は強すぎると正規ユーザーを止め、弱すぎると攻撃を止められない。
tenant override と metric を先に入れ、運用で閾値を調整できる形にする。

## Completion

- **Completed At**: 2026-08-09
- **Summary**:
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
- **Verification Results**:
  - `just verify`（test-go / lint-go / build-ui / check / typecheck-tools / test-tools /
    traceability-strict / lint-ui / format-check-ui / check-api-compat の並列実行）- passed
  - `just verify-go`（lint-go + `-race` 全テスト）- passed
  - `backend/shared/ratelimit/db_postgres` の `-race`（`TestRateLimiterConcurrentAllow`:
    MaxRequests=8 に対し 16 並行呼び出しで許可がちょうど 8 件）- passed（原子性の実証）
  - `just check-architecture`（台帳・実 import 突合含む）- passed
  - `just check-schema`（psqldef 収束）- passed
  - `just typecheck-ui` / `just lint-ui` / `just build-ui` / `bun test`（i18n・auth-flow 該当分）
    - passed
  - 手動 E2E（backend のみ、`RATE_LIMIT_AUTHORIZE_MAX_REQUESTS=2` でサーバ起動し
    `/realms/default/authorize` に3回リクエスト）: 3回目が `429` + `Retry-After: 60` +
    `{"error":"rate_limited","message":"...","retry_after_seconds":60}` で拒否されることを確認
- **Affected Guarantees State**:
  - fail-closed: 全 6 policy が共有カウンタストア到達不能時にエラーを伝播し拒否側に倒れる
    （postgres 到達不能 → `Allow` がエラーを返し `CheckRateLimit` が呼び出し元へエラーを伝播）。
    login throttle の fail-closed（ADR-077/139）は変更していない。
  - 原子性: `endpoint_rate_limit_counters` の read-modify-write は tx +
    `SELECT ... FOR UPDATE` で直列化（`TestRateLimiterConcurrentAllow` で実証）。
  - 監査/プライバシー: key は SHA-256 でハッシュ化してから保存（`hashRateLimitKey`、
    `login_throttle_counters` の `hashThrottleIdentifier` と同型）、raw な client_id/IP/
    identifier は保存しない。
  - username enumeration: password reset の「email 存在有無に関わらず 204」は変更していない
    （rate limit は identifier+IP の組で先に評価するのみ）。
- **Evidence**:
  - 手順: 上記コマンドをローカルで実行（darwin、本リポジトリ作業ツリー）。
  - 対象ソース版: 本セッションでの一連の変更（ADR-157 新規、SCL 更新、
    `backend/shared/ratelimit` 新設、6 HTTP handler 配線、bootstrap 配線、UI 3 画面、
    README、`ARCHITECTURE.md`）。
  - 結果: 上記すべて green。
  - 保存先: git 履歴（本コミット）。大容量ログ・機密は無し。
  - **未実施（開示）**: UI をブラウザで実際に操作しての目視確認は行っていない
    （`bun test` によるユニットテストと `typecheck-ui`/`build-ui` のみ）。password_reset /
    device_authorization / par / login 各 policy の 429 を実際にトリガーする手動 E2E は
    authorize policy でのみ実施し、他 policy は自動テスト（memory/postgres contract test +
    HTTP 層の `CheckRateLimit` unit test）のカバレッジに委ねている。複数プロセス間での
    共有カウンタの実地検証（同一 IP から複数 API replica への分散リクエスト）は未実施。
    CAPTCHA / bot challenge は意図的に Out of Scope。
