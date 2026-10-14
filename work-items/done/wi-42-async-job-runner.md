---
status: completed
archived_from: 61f5d0fe18c17eadc4a23f1b9048389f098c55ca
---

# テナント境界を保つ汎用非同期ジョブ基盤の core runtime を導入する

テナント境界を保つ汎用非同期ジョブ基盤の core runtime を実装した。
ADR-098 (PostgreSQL `FOR UPDATE SKIP LOCKED` によるリース方式 durable
queue)、ADR-099 (`idmagic-worker` プロセス分離・at-least-once + 冪等・
リース失効再取得・backoff + dead-letter・graceful drain)、ADR-100
(params/result は平文 JSONB・30日 TTL) を決定。`spec/contexts/jobs.yaml`
に `Jobs` bounded context を追加し、Go 側に `backend/jobs`
(domain/ports/usecases/adapters(memory・postgres)) と、新規プロセス
`idmagic-worker` (`backend/cmd/idmagic-worker`,
`backend/bootstrap/worker.go`) を実装した。API プロセスから
テナント横断の retention sweep goroutine を worker プロセスへ移管した。
疎通確認用の `noop_echo` JobKind とハンドラを追加し、enqueue →
Queued → Running → Succeeded、worker crash 後のリース失効再取得、
max_attempts 超過での dead-letter 確定を自動テストで確認した。

全文は `git show 61f5d0fe18c17eadc4a23f1b9048389f098c55ca:<このファイルのパス>` で読む。
