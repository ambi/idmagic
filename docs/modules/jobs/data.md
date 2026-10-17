# Jobs のデータ

この文書は、Jobs の Aggregate を扱う。

## 集約

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `Job` | 種類（`JobKind`）、レーン、状態、`params`、`result`、`error`、`run_at`、`attempts` と `max_attempts`、リースの所有者と期限、`dedup_key` | `Tenant` を `tenant_id` で参照する |

`JobKind` は、ちょうど一つの `ExecutionLane`（`latency_sensitive`、`default`、`bulk`）を、`domain.RegisterKind(kind, lane)` による登録の時点で固定する。
投入する呼び出し元はレーンを指定できない。

- **判断**：`params` と `result` は保存時に暗号化しない。`JobKind` ごとに解釈する不透明な値であり、シークレットを含めてはならないからである。
