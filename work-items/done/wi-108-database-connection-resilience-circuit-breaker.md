---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# データベースおよび Valkey への接続レジリエンス強化とサーキットブレイカーの導入

自作の軽量かつスレッドセーフな `CircuitBreaker` (`internal/shared/resilience/circuitbreaker.go`) を実装。
`ResilientDB` ラッパーによる PostgreSQL 接続へのタイムアウトとサーキットブレイカー、および
`resilience.RetryWithBackoff` による接続時の Exponential Backoff リトライを実装した。各リポジトリ
(20 ファイル) の `pgxpool.Pool` 直接参照を `DB` インターフェースに変更し、ラップされた
`ResilientDB` を DI した。Valkey クライアントには `redis.Hook` を用いた透過的なサーキットブレイカー
およびコンテキストタイムアウトを注入した。JWKS および Discovery メタデータの公開エンドポイントでは
`sync.Map` を用いたインメモリキャッシュでグレイスフル劣化できるようにした。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
