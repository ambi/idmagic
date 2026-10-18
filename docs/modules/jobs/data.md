# Jobs のデータ

この文書は、Jobs の Aggregate を扱う。

## 集約

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `Job` | 種類（`JobKind`）、レーン、状態、`params`、`result`、`error`、`run_at`、`attempts` と `max_attempts`、リースの所有者と期限、`dedup_key` | `Tenant` を `tenant_id` で参照する |

`JobKind` は、ちょうど一つの `ExecutionLane`（`latency_sensitive`、`default`、`bulk`）を、`domain.RegisterKind(kind, lane)` による登録の時点で固定する。
投入する呼び出し元はレーンを指定できない。

- **判断**：`params` と `result` は保存時に暗号化しない。`JobKind` ごとに解釈する不透明な値であり、シークレットを含めてはならないからである。

## 呼び出し側のトランザクションへの参加

Group の CSV の行のようにジョブと利用側の状態を不可分に保存する場合は、`NewEnqueuerInTx` が同じ Tx へ結んだ投入器を作る。
通常の投入と同じユースケースを使い、レーン、デフォルト値、重複排除と `active_jobs` の確認を省かない。
呼び出し側は同じ Tx に結んだクォータとイベントを保持する関数を渡し、新規の投入イベントを確定後だけ発行する。
クォータ拒否は利用側の状態とともにロールバックし、拒否の通知だけをその後に発行する。
Jobs は参加先のトランザクションを確定せず、利用側の確定の責任を引き取らない。
