# Jobs

テナント境界を保つ汎用非同期ジョブ基盤として、永続キューと `worker` の実行環境を所有する。対象は、非同期処理に共通する投入、永続化、取得、リース、ハートビート、再試行、配信不能への退避、キャンセルであり、業務処理は所有しない。

`JobKind` のパラメーターを解釈して副作用を起こす処理は、利用側 Context のユースケースに置く。`backend/cmd/idmagic-worker/worker.go` が起動時にそれらのハンドラーを一覧へ登録する。この分離により、ジョブ基盤は利用側 Context の業務ロジックに依存しない。

API プロセスはジョブを投入するが実行せず、`worker` プロセスはジョブを実行するが HTTP を提供しない。

```text
API / 利用側のユースケース
  └─ EnqueueJob（レーンは種別から導出）
       └─ JobRepository ──> PostgreSQL の jobs（lane 列、レーンを先頭にしたインデックス）
                                  │
idmagic-worker                    │ レーンごとに独立してポーリング
  ├─ ライフサイクルワークフローのディスパッチャー │（未投入の実行を回収）
  ├─ Runner (lane=latency_sensitive) ── ClaimBatch(lane) <───┘
  ├─ Runner (lane=default)          ── ClaimBatch(lane) <───┘
  ├─ Runner (lane=bulk)             ── ClaimBatch(lane) <───┘
  │    ├─ HandlerRegistry ──> 利用側 Context のユースケース（共有するハンドラー一覧）
  │    ├─ Heartbeat ────────> lease_expires_at を延長
  │    └─ Complete / Fail ──> succeeded / queued（再試行）/ failed
  └─ jobsQueueDepthSamplingLoop ──> レーンごとのキュー深度・実行中ゲージ（10 秒間隔）
```

| File | Content |
|---|---|
| [glossary.md](glossary.md) | この Context での語義 |
| [states.md](states.md) | 状態と遷移 |
| [decisions.md](decisions.md) | 設計判断 |
| [internals.md](internals.md) | 機構の説明 |
| [scenarios.md](scenarios.md) | 受け入れシナリオ |
