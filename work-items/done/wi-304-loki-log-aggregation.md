---
status: completed
archived_from: 255f213e3dd474083ef5ddb9355f26d9de671323
---

# アプリケーションログの永続化と検索のための Promtail / Loki 導入

`infra/docker/docker-compose.dev.yaml` に `loki`（`loki-config.yaml`、filesystem storage）、`promtail`（`promtail-config.yaml`、`docker_sd_configs` でコンテナを検出し Docker Engine API 経由でログを取得するため、ホストのログディレクトリはマウント不要）、`grafana`（`grafana-datasources.yml` で Prometheus/Loki を自動登録、`grafana-dashboards.yml` で既存 `grafana-dashboard.json` を自動読込）を追加した。
`infra/k8s/monitoring/loki/` に Loki（StatefulSet + PVC 10Gi + Service + NetworkPolicy）、Promtail（DaemonSet + ClusterRole/Binding + ConfigMap、`kubernetes_sd_configs` で `/var/log/pods` を監視）、および既存 Grafana インスタンス向けの Loki データソース ConfigMap（`grafana-dashboard.yaml` と同じ sidecar 発見規約 `grafana_datasource: "1"`）を追加し、親 `infra/k8s/monitoring/kustomization.yaml` に組み込んだ。`ARCHITECTURE.md` に Logging 節を追加した。
`service`/`level` は Loki のラベル（有限集合）にする一方、`trace_id`/`span_id`/`request_id` は structured metadata として扱い、ラベルにはしない（`ARCHITECTURE.md` Metrics 節の `tenant_id`/`user_id` と同じ理由：無制限のカーディナリティはインデックスを破壊する）。この扱いは work item の Design 記述（`trace_id` 等をラベルとする案）から意図的に外れているが、SCL/実装契約を変更するものではなくインフラ内部の実装判断のため ADR は起こしていない。

全文は `git show 255f213e3dd474083ef5ddb9355f26d9de671323:<このファイルのパス>` で読む。
