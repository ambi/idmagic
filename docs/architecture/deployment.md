# 配備アーキテクチャ

## 参照トポロジー

参照トポロジは製品中立であり、特定クラウドへの適用済み構成を表さない。

```text
Clients
  |
Gateway / load balancer ---- React static assets
  |
Stateless API replicas (N)
  |
PostgreSQL shared state ---- Worker replicas by execution lane
  |
Externally scheduled batch processes
```

ゲートウェイは TLS、同一オリジン、経路制御、健全な API レプリカへの分散を担い、テナントの正となる状態を持たない。
API レプリカは PostgreSQL の共有状態から同じセキュリティ判断を行う。
ワーカーは `latency_sensitive`、`default`、`bulk` の実行レーンごとに独立して増減し、バッチは一回限りの処理として外部スケジューラーが起動する。

## リポジトリの配備形態

Docker Compose はローカル開発と訓練、Kubernetes の base と overlay は汎用的なクラスター配備、`infra/deploy/gcp/` は GCP の例を持つ。
実際の本番環境がいずれの参照構成を採用しているかは、このリポジトリだけからは確認できない。

Kubernetes の本番 overlay は API と UI の三レプリカ、レーン別ワーカー、HorizontalPodAutoscaler、PodDisruptionBudget、NetworkPolicy を宣言する。
PostgreSQL 自体の配備はこのリポジトリの Kubernetes 資材に含まれず、運用基盤が接続先と冗長化を提供する。

## 配備の不変条件

- 複数 API レプリカでは `PERSISTENCE=postgres` を使う。
- API のレプリカ数は HPA が所有し、Deployment の固定値と競合させない。
- 秘密は環境ごとの外部 Secret から注入し、構成資材へ値を書かない。
- API の受付停止をロードバランサーへ伝えてから、処理中のリクエストを退避させる。
- スキーマ変更は新旧のアプリケーションが同時に動ける拡張と縮小の段階に分ける。

負荷に応じた拡張は[拡張設計](../design/performance/scaling.md)、障害時の配置と切替は[可用性設計](../design/reliability/availability.md)が持つ。
