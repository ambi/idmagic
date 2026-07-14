---
status: accepted
authors: [tn]
created_at: 2026-07-14
---

# ADR-105: System の runtime resilience / HTTP hardening / i18n ツール設定を ARCHITECTURE 層の文書に移す

## コンテキスト

[[ADR-103]] は SCL 3.0 の `objectives` を観測可能な SLI に対する SLO だけに限定し、retention、
security configuration、runtime、logging、アーキテクチャ判断は ADR または `ARCHITECTURE.md` へ
移すことを決定した。`spec/contexts/system.yaml` (SCL 2.0) の `objectives` 18件のうち、以下は
`indicator` / `target` / `window` / `budgeting` を持つ観測可能な比率目標ではなく、単一の設定値・
運用方針の集合である。

- `DatabaseResilience` / `ValkeyResilience`: 接続プール・タイムアウト・サーキットブレイカー設定。
- `CancellationConsistency` / `ClientAbortLogClassification` / `RequestFaultIsolation`: HTTP リクエスト
  の cancellation・panic 分離・request_id 伝播方針。
- `HTTPServerHardening` / `SecurityResponseHeaders` / `FrameAncestorsPolicy`: 境界 HTTP サーバの
  タイムアウト・ボディ上限・セキュリティレスポンスヘッダ設定。
- `BuildInfo` / `StructuredApplicationLog`: ビルドメタデータ露出とアプリケーションログの構造化方針。
- `SharedEphemeralStateHA`: 複数レプリカ運用時の ephemeral 状態共有トポロジ。
- `HealthProbe` / `ReadinessCheck` / `GracefulDrain`: liveness/readiness/startup プローブと drain 方針。
- `TranslationKeyIntegrity` / `FrontendLocalizationCoverage`: ja/en 辞書の translation key 完全性を
  ビルド・テスト時に検証する開発ツール方針。

wi-209 (IdMagic の基盤・identity 系 context を SCL 3.0 に移行する) はこれらを SCL から除去する
移行作業であり、値そのものを変更しない。移行先が無いまま削除すると運用者が参照する既存の
正本を失うため、本 ADR に値を保存し、`spec/contexts/system.yaml` の SCL 3.0 版はこの ADR を指す
記述に留める。

## 決定

以下の設定を IdMagic の runtime hardening / i18n ツール方針の正本として本 ADR に集約する。値は
SCL 2.0 の該当 `objectives` から機械的に転記したものであり、本 ADR 採用時点で新たな決定は行わない。

### 1. データストア接続の resilience

- PostgreSQL 接続プール: `max_conns=50`, `min_conns=2`, `max_conn_idle_time=1800s`,
  `max_conn_lifetime=3600s`, `timeout=5s`。接続リトライは `max_attempts=3`,
  `backoff_factor=2.0`。サーキットブレイカーは `failure_threshold=0.5`, `cooldown=30s`。
  オープン時は 503 で即時遮断するが、JWKS など静的・キャッシュ可能なエンドポイントはメモリ
  キャッシュで縮退運転を継続する。
- Valkey 接続: `timeout=2s`。サーキットブレイカーは `failure_threshold=0.5`, `cooldown=15s`。
  障害時はハングを防ぎ、即時 503 またはメモリへの縮退フォールバックとする。

### 2. HTTP リクエストの cancellation と障害分離

- read-only endpoint は HTTP request context の client abort をそのまま処理中断に使ってよい。
  mutation endpoint は `rollback` / `complete` / `detached_completion` のいずれかの completion
  mode を明示し、認可コード・refresh token family・consent・session・audit/outbox が観測不能な
  中間状態に残らないようにする。
- client abort 由来の `context.Canceled` は server error として計上せず、`server_timeout` /
  `upstream_timeout` と別ラベル (`http_request_aborts_total`) で観測する。
- 個々の HTTP ハンドラの panic は該当リクエストに局所化する。middleware が panic を捕捉し 500 +
  構造化 stack ログへ変換し、プロセスは継続稼働する。全リクエストに `request_id` を付与し、
  レスポンスヘッダとログへ伝播する。`request_id` は既定でアプリが自前生成する secure-by-default
  であり、受信 `X-Request-ID` は `REQUEST_ID_TRUST_INBOUND` を明示的に opt-in したときだけ
  (信頼できる境界プロキシが所有・消毒している前提で) 再利用する。信頼の有無に関わらず受信値は
  許可文字・長さ制限でサニタイズし、ヘッダ/ログインジェクションを防ぐ。

### 3. 境界 HTTP サーバの hardening

- `http.Server` に `read_header_timeout=10s`, `read_timeout=30s`, `write_timeout=60s`,
  `idle_timeout=120s` を設定し、リクエストボディ上限を `1048576` bytes とする (超過は 413)。
  値は env で上書き可能だが既定は本番安全側に倒す (gosec G112 / CWE-400 の低速接続・巨大ボディ
  DoS への下限防御)。volumetric / TLS ハンドシェイク DoS の主防御は前段プロキシが担う。
- 全レスポンスにセキュリティヘッダを middleware で一元付与する: `X-Content-Type-Options: nosniff`、
  `Referrer-Policy: no-referrer`、`Content-Security-Policy` は `default-src 'none'` を基本とし
  `'unsafe-inline'` に依存しない (SAML ACS / WS-Fed の自動 POST フォーム submit script のみ CSP
  hash (sha256) でそのレスポンスに限定して許可し、nonce は用いない)。CSP は `CSP_REPORT_ONLY` で
  enforce/report-only を切替、`CSP_REPORT_URI` で違反レポート収集先を指定できる。
  `Strict-Transport-Security` は TLS 終端層が所有すべきヘッダのため既定 (開発 http) では抑制し、
  `HSTS_ENABLED` で明示的に opt-in したときだけ `max-age=31536000; includeSubDomains` を付与する。
- login / consent / account portal を含む全レスポンスは `frame-ancestors 'none'`
  (併せて `X-Frame-Options: DENY`) とし iframe 埋め込みを禁止する (clickjacking 対策)。
  SAML ACS / WS-Fed の自動 POST フォームは `CSP form-action` にのみ送信先を明示許可し、
  `frame-ancestors` は `'none'` を維持する。per-route の判断はアプリが持ち、プロキシへ委譲しない。

### 4. ビルド情報とアプリケーションログ

- 稼働バイナリの一意性確認のため `version` / `git_commit` / `build_date` / `go_version` を
  ビルド時に埋め込み、ログと OTel resource に露出する。
- アプリケーションログ (debug/info/warn/error) は監査ログと分離し、stdout へ構造化 JSON Lines
  で出力する。必須フィールドは `timestamp` / `level` / `service` / `message`。OTel span context
  有効時は `trace_id` / `span_id` で相関する。PII (email 等) はアプリケーションログに平文出力しない。

### 5. 複数レプリカ運用時の ephemeral state 配置

- `PERSISTENCE=postgres_valkey` の複数レプリカ本番構成では、authorization request / authorization
  code / PAR / device code / login session / DPoP・client-assertion replay / access-token
  denylist / login throttle を Valkey に共有し、per-replica のプロセスメモリに残さない
  (login throttle をメモリに残すとレプリカ数倍に閾値が緩むため)。refresh token / audit event /
  auth-event bucket は PostgreSQL が durable な共有ストアとして持つ。memory adapter は単一
  レプリカ・テスト専用であり、複数レプリカ運用では Valkey が必須である。共有ストア到達不能時、
  login throttle は fail_closed とする。

### 6. ヘルスプローブと graceful drain

- liveness (`/livez`)、readiness (`/readyz`)、startup (`/startupz`) を個別エンドポイントに分離する。
  既存 `/health` は起動構成ラベル返却用に維持する。liveness は自己回復不能な致命的デッドロック等
  のみで fail させ、一時的な依存障害では fail させない。startup は初期化完了時に healthy となる。
- readiness プローブは PostgreSQL / Valkey への Ping を `timeout=1s` で並列実行し、`?verbose` で
  `healthy` / `degraded` / `unavailable` を依存ごとに列挙する。
- `SIGTERM` / `SIGINT` 受信時は readiness を即座に `unavailable` にし、`DRAIN_GRACE_PERIOD_SECONDS`
  (既定 5秒) だけ待機してから HTTP サーバをシャットダウンする。

### 7. フロントエンド i18n 辞書の完全性チェック (開発ツール)

- ja/en 辞書は translation key の完全な対を持つ。ビルドまたはテストで、一方の辞書にのみ存在する
  key、コードから参照されるが辞書に存在しない key を検出して失敗させる。どちらの辞書からも
  参照されない未使用 key は警告として報告する。
- 利用者向け画面文字列は feature-local または共通辞書に ja/en の対として存在させ、代表画面の
  ja/en レンダリングテストと直書き文字列検出でリグレッションを検出する。

## 却下した代替案

- 各設定値を対応する `models` / `interfaces` の `description` に手作業で分散させる: 単一のクロス
  カッティングな運用設定を復数箇所へ分散すると、変更時に同期漏れが起きやすく、`ARCHITECTURE.md`
  的な単一の運用ドキュメントとしての一覧性を失う。
- 新しい `objectives` の kind として残す: [[ADR-103]] が `objectives` を SLO 専用と定めた決定を
  本 ADR の範囲で覆さない。

## 影響

- `spec/contexts/system.yaml` の SCL 3.0 版はこれら 15 件の `objectives` を持たず、本 ADR を
  runtime hardening / i18n ツール設定の正本として参照する。
- 値そのもの (プール上限、タイムアウト、閾値等) は変更しない。実装・runtime 挙動への影響はない。
