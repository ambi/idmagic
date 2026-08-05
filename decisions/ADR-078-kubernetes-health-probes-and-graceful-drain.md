---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-078: 依存ヘルスを検査する liveness/readiness/startup probe と SIGTERM 時の接続ドレインを整備する

## コンテキスト
現在の idmagic のヘルスチェック `/health` は、起動構成ラベル（persistence / event_sink / observability / authzen）をそのまま JSON で返すだけで、PostgreSQL や Valkey などの依存の実際の接続性を検査していない。
このエンドポイントを Kubernetes の liveness と readiness プローブに流用すると、DB 瞬断で Pod が不必要に再起動ループに陥ったり、接続不全の Pod へトラフィックがルーティングされ続けたりする。
そのため、Kubernetes の標準に合わせ、(1) 生存性 `/livez`、(2) トラフィック受入可否 `/readyz`、(3) 起動完了 `/startupz` を分離し、SIGTERM 受信時に readiness プローブを即 unready に落として接続ドレインを行う仕組みが必要である。

## 決定

`scl.yaml` の `objectives.HealthProbe` / `objectives.ReadinessCheck` / `objectives.GracefulDrain` と該当ソースに反映。

`/health` を liveness と readiness で共用するのをやめ、`/livez`（生存性）・`/readyz`（トラフィック受入可否、`?verbose` で依存ごとの状態語彙を返す）・`/startupz`（起動完了）に分離する。SIGTERM/SIGINT 受信時は `/readyz` を即座に unready へ落とし、ドレイン猶予期間（既定 5秒、`DRAIN_GRACE_PERIOD_SECONDS`）だけ待ってからサーバをシャットダウンする。一時的な DB/Valkey 瞬断が不要な再起動ループと、接続不全 Pod への継続的なトラフィックルーティングの両方を同時に引き起こしていたことが理由で、却下した代替案（`/health` の共用継続）はこの二重の障害を残す。

現在のメカニズムの詳細（各エンドポイントの判定基準、ドレインの手順）は [ARCHITECTURE.md](../ARCHITECTURE.md) の Runtime Composition「Health probes and graceful drain」を参照。

## 却下した代替案
- **`/health` エンドポイントを readiness と liveness で共用し続ける案**:
  - 前述の通り、一時的な DB の瞬断がコンテナの不要な再起動ループを引き起こし、かつ復旧を遅らせるため却下。

## 影響
- `spec/contexts/system.yaml` に `HealthProbe`、`ReadinessCheck`、`GracefulDrain` の objectives を追加。
- `internal/bootstrap/server.go` でのシグナルハンドリングと graceful shutdown 協調処理の追加。
- `internal/shared/adapters/http/server/health_handler.go` 周辺でのプローブ実装と、`Deps` への `DbPing` / `ValkeyPing` の注入。
