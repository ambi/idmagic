---
id: idp-wi-108-database-connection-resilience-circuit-breaker
title: "データベースおよび Valkey への接続レジリエンス強化とサーキットブレイカーの導入"
created_at: 2026-07-04
completed_at: 2026-07-04
authors: ["tn"]
status: completed
risk: medium
scope:
  - System:objectives:DatabaseResilience
  - System:objectives:ValkeyResilience
---

# Motivation
現在 idmagic は PostgreSQL と Valkey をデータ永続・セッション管理に使用しているが、
これらへの接続プーリング設定の動的調整や、一時的な瞬断に対する自動再接続・リトライポリシーが未整備である。
本番の HA 構成下では、一時的な DB 高負荷やネットワーク瞬断が発生した際、接続待ちのリクエストが滞留して
Thundering Herd 問題を引き起こし、最終的に IdP 全体がクラッシュ（メモリ枯渇等）しうる。
また、Valkey のような ephemeral state 用キャッシュが一時停止した際、ログインスロットルやセッション状態の
取得で永久にハングするのを防ぐため、タイムアウト制御とサーキットブレイカー（Circuit Breaker）パターン、
および障害発生時のグレイスフル劣化（安全なエラー画面へのフォールバック）が不可欠である。

# Scope
- **go**: PostgreSQL (pgx) の接続プールパラメータ（MaxConns, MinConns, MaxConnIdleTime, MaxConnLifetime）を Config 構造体（WI-103）から制御可能にする。, DB 問い合わせおよび Valkey 操作に対して適切なコンテキストタイムアウトを設定する。, Valkey / PostgreSQL 接続クライアントへサーキットブレイカー（例: go-kit/kit/circuitbreaker 等、または自作の軽量実装）を組み込む。, サーキットがオープン（障害検知）した際、即座に 503 Service Unavailable を返す、またはセッションが必要ない特定のパブリックエンドポイント（例: JWKS / 公開メタデータ）はメモリキャッシュを利用してレスポンスし続けるグレイスフル劣化を実装する。, 接続エラー発生時の Exponential Backoff を用いた再接続リトライ処理を接続プール層に追加する。

# Out of Scope
- アクティブ・アクティブな複数プライマリ DB の同期制御。
- 外部のプロキシ層（PgBouncer 等）の直接設定管理。

# Verification
- just verify-go
- 手動: ローカル Docker Compose で PostgreSQL / Valkey を意図的に停止、またはパケットロスを発生させ、IdP プロセスが即座にタイムアウトエラーを返し、かつ DB 復旧後に自動でリクエストが正常処理に戻ることを確認する。

# Risk Notes
サーキットブレイカーの判定パラメータ（エラー率の閾値、クールダウン期間）の設計が甘いと、
一時的なエラーでサービス全体が過剰に遮断されてしまうリスクがある。
初期値はコンフィグから調整できるようにし、メトリクスを通じてトリガー状況を観測可能にする。

# Completion
- **Completed At**: 2026-07-04
- **Summary**:
  自作の軽量かつスレッドセーフな `CircuitBreaker` (`internal/shared/resilience/circuitbreaker.go`) を実装。
  `ResilientDB` ラッパーによる PostgreSQL 接続へのタイムアウトとサーキットブレイカー、および
  `resilience.RetryWithBackoff` による接続時の Exponential Backoff リトライを実装した。各リポジトリ
  (20 ファイル) の `pgxpool.Pool` 直接参照を `DB` インターフェースに変更し、ラップされた
  `ResilientDB` を DI した。Valkey クライアントには `redis.Hook` を用いた透過的なサーキットブレイカー
  およびコンテキストタイムアウトを注入した。JWKS および Discovery メタデータの公開エンドポイントでは
  `sync.Map` を用いたインメモリキャッシュでグレイスフル劣化できるようにした。
- **Verification Results**:
  - `just verify-go`
    - result: passed
