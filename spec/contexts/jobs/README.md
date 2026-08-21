# Jobs

テナント境界を保つ汎用非同期ジョブ基盤として、永続キューと `worker` の実行環境を担う。対象は、非同期処理に共通する投入、永続化、取得、リース、ハートビート、再試行、配信不能への退避、キャンセルであり、業務処理そのものは扱わない。

`JobKind` のパラメーターを解釈して副作用を起こす処理は、利用側 Context のユースケースに置く。`backend/cmd/idmagic-worker/worker.go` が起動時にそれらのハンドラーを一覧へ登録する。この分離により、ジョブ基盤は利用側 Context の業務ロジックに依存しない。

API プロセスはジョブを投入するが実行せず、`worker` プロセスはジョブを実行するが HTTP を提供しない。キューを操作する HTTP エンドポイントは持たないので、テナント管理者や API アクセストークンから直接ジョブへ到達する経路は無い。

| File | Content |
|---|---|
| [glossary.md](glossary.md) | この Context での語義 |
| [states.md](states.md) | 状態と遷移 |
| [decisions.md](decisions.md) | 設計判断 |
| [internals.md](internals.md) | 機構の説明 |
| [scenarios.md](scenarios.md) | 受け入れシナリオ |
