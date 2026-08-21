# Jobs Glossary

| Term | Definition | Aliases |
|---|---|---|
| JobLease | `worker` が Job を Running にする際に確保する排他権。`lease_owner`（`worker` 識別子）と `lease_expires_at` を持ち、ハートビートで更新する。ハートビートが途絶えて期限切れになると、別の `worker` が再取得できる。 | lease, リース |
| DeadLetter | `attempts` が `max_attempts` に達して `Failed` に確定した Job。再試行されず、エラーを保持したまま調査対象として残る。 | 配信不能 |
| Queued | Job が投入され、まだ `worker` に取得されていない初期状態。`run_at` を過ぎると取得できる。 | queued |
| Running | `worker` がリースを確保し、ハンドラーを実行中の状態。 | running |
| Succeeded | ハンドラが正常終了した終端状態。 | succeeded |
| Failed | `attempts` が `max_attempts` を超えて配信不能に確定した終端状態。 | failed |
| Canceled | 終端状態に達する前に取消された終端状態。 | canceled |
| Claim | `worker` が実行可能な Job を取得し、自身を `lease_owner` として Running へ遷移させる。 | claim |
| Complete | `worker` がハンドラーの正常終了を報告し、Job を Succeeded に確定する。 | complete |
| Fail | `worker` がハンドラーの失敗を報告する。`attempts` が `max_attempts` 未満なら Retry、以上なら配信不能（Failed）に確定する。 | fail |
| Retry | 失敗後、バックオフを経て再試行のため Queued に戻す遷移。 | retry |
| Cancel | 終端状態に達していない Job を取消す。 | cancel |
| ExecutionLane | JobKind の登録情報が一意に決める実行枠の区分。レイテンシーや資源特性が異なる JobKind 間で、取得処理と `worker` の実行枠を分離するために使う。投入元はレーンを指定できない。`latency_sensitive` / `default` / `bulk` の 3 種類がある。 | lane, 実行レーン |
| Developer | 標準開発環境でこのリポジトリを動かす開発者。 |  |
| System | Jobs の永続キューと `worker` の実行環境そのもの。人間の操作者を伴わない技術的な主体を指す。 |  |
