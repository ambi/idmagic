---
status: completed
authors: [tn]
risk: low
created_at: 2026-07-28
depends_on: []
---

# アプリケーションログの永続化と検索のための Promtail / Loki 導入

## Motivation
現在アプリケーションログは標準出力に JSON Lines 形式で出力されているが（ADR-018）、本番環境（Kubernetes等）やローカルの Docker Compose でそれらを中央集約・永続化し、横断的に検索する仕組みがない。監視・トレース基盤としてすでに Prometheus および OpenTelemetry が導入されており、トレース ID との連携やラベルベースの検索といった親和性の高さから、ログ収集基盤として Promtail と Grafana Loki を採用し、運用性と可観測性を高める必要がある。

## Scope
- `infra/docker/` のローカルインフラ構成（Loki, Promtail および **Grafana** の追加・データソース設定）
- `infra/k8s/monitoring/` の Kubernetes マニフェスト（Loki, Promtail DaemonSet の追加、および Grafana への Loki データソース登録設定）
- `ARCHITECTURE.md` への構成の記録とドキュメント化

## Out of Scope
- アプリケーション側の出力形式（`backend/shared/logging`）の変更（現状のままとする）
- Elasticsearch などの別ロギングスタックの評価

## Design
- **Log Aggregation Stack**: Grafana Loki を採用する。Loki はインデックスを抑え、オブジェクトストレージなどに実体を保存できるため、コスト効率が高い。
- **Agent**: Promtail を使用する。既存の OpenTelemetry Collector とは独立させ、ログ収集に特化させる。Promtail は Docker または Kubernetes ノードのログディレクトリ (`/var/log/containers/` など) を監視し、出力された JSON ログからラベル（`service`, `level`, `trace_id` 等）をパースして Loki に送信する。
- **Web GUI (Grafana)**: 収集したログを閲覧・検索するための UI として Grafana を用いる。
  - ローカル環境 (`docker-compose.dev.yaml`) には現在 Grafana コンテナが存在しないため、新たに `grafana` コンテナを追加し、Loki と Prometheus をデータソースとして自動登録（Provisioning）する。
  - 本番環境 (`infra/k8s/monitoring`) では、Loki サーバーおよびノードごとに配置される Promtail のマニフェストを追加し、既存の Grafana インスタンスに対して Loki データソースを追加構成するマニフェスト（または Secret/ConfigMap）を提供する。

## Plan
1. **Local Infrastructure**: `docker-compose.dev.yaml` に Loki, Promtail, および **Grafana** を追加。`promtail-config.yaml` を作成し、Loki **および既存の Prometheus** をデータソースとして登録する `grafana-datasources.yml`、さらに既存のダッシュボード (`grafana-dashboard.json`) を自動読み込みする `grafana-dashboards.yml` を作成し、Grafana をプロビジョニングする。
2. **Kubernetes Manifests**: `infra/k8s/monitoring/loki/` ディレクトリを作成し、Loki と Promtail (DaemonSet) のベースマニフェスト、および K8s 上の Grafana に Loki データソースを登録するための設定を追加する（※K8s 上の Prometheus連携/ダッシュボードは既存の `monitoring` にあるため、Loki部分のみ追加）。
3. **Documentation**: `ARCHITECTURE.md` を更新し、Loki と Promtail を用いる方針を明記する。

## Tasks
- [x] T001 [Infra] `docker-compose.dev.yaml` に Loki, Promtail, Grafana コンテナを追加。Grafana に Loki と Prometheus の両方をデータソースとしてプロビジョニングし、既存の `grafana-dashboard.json` を表示できるようにする。RED/GREEN: 対象外（Infrastructure/Deploy Pipeline 層、振る舞いを持つ層ではないため test-first 免除）。
- [x] [Verify] ローカルで Grafana 上に Prometheus のダッシュボードが表示されること、および Loki へのログ取り込み・検索ができることを確認する。`docker-grafana-1`/`docker-prometheus-1` を起動し `/api/datasources` で Loki/Prometheus 両方の自動プロビジョニングを確認、`/api/search` でダッシュボードフォルダの自動読込を確認。ADR-018 形式の JSON ログを模したコンテナ（`compose_service=idp` ラベル）を起動し、Loki `/loki/api/v1/query_range` と Grafana 経由のプロキシクエリで `level`/`service` がラベルに、`trace_id`/`span_id`/`request_id` が structured metadata（インデックス外）になっていることを `/loki/api/v1/labels` で確認。
- [x] T002 [Infra] `infra/k8s/monitoring/loki/` に Kubernetes 用の Promtail / Loki マニフェストを作成する。RED/GREEN: 対象外（同上）。
- [x] T003 [Doc] `ARCHITECTURE.md` に Loki / Promtail を用いたログ集約方針を追記する。

## Verification
- ローカル環境: `docker compose -f infra/docker/docker-compose.dev.yaml up -d` を実行し、ブラウザでローカルの Grafana (`http://localhost:3000` 等) にアクセスする。
  - Loki のログ（Explore タブ等）が検索・閲覧できることを確認する。
  - Prometheus を用いた既存のダッシュボード（`idmagic-grafana-dashboard` 等）が自動的にプロビジョニングされ、表示されることを確認する。
- k8s マニフェスト: `kubectl kustomize infra/k8s/monitoring/loki` または `kustomize build` がエラーなくビルドできることを確認する。
- プロジェクト整合性: `just verify` (もしくは `just check-work-items`, `just check-ids`) が通過すること。

## Risk Notes
- Kubernetes マニフェストは環境によって PersistentVolume の StorageClass や権限周りが異なるため、今回は標準的なマニフェストを提供し、実際のデプロイ時には環境別の overlays で上書き可能にしておく。

## Completion
- **Completed At**: 2026-07-29
- **Summary**:
  `infra/docker/docker-compose.dev.yaml` に `loki`（`loki-config.yaml`、filesystem storage）、`promtail`（`promtail-config.yaml`、`docker_sd_configs` でコンテナを検出し Docker Engine API 経由でログを取得するため、ホストのログディレクトリはマウント不要）、`grafana`（`grafana-datasources.yml` で Prometheus/Loki を自動登録、`grafana-dashboards.yml` で既存 `grafana-dashboard.json` を自動読込）を追加した。
  `infra/k8s/monitoring/loki/` に Loki（StatefulSet + PVC 10Gi + Service + NetworkPolicy）、Promtail（DaemonSet + ClusterRole/Binding + ConfigMap、`kubernetes_sd_configs` で `/var/log/pods` を監視）、および既存 Grafana インスタンス向けの Loki データソース ConfigMap（`grafana-dashboard.yaml` と同じ sidecar 発見規約 `grafana_datasource: "1"`）を追加し、親 `infra/k8s/monitoring/kustomization.yaml` に組み込んだ。`ARCHITECTURE.md` に Logging 節を追加した。
  `service`/`level` は Loki のラベル（有限集合）にする一方、`trace_id`/`span_id`/`request_id` は structured metadata として扱い、ラベルにはしない（`ARCHITECTURE.md` Metrics 節の `tenant_id`/`user_id` と同じ理由：無制限のカーディナリティはインデックスを破壊する）。この扱いは work item の Design 記述（`trace_id` 等をラベルとする案）から意図的に外れているが、SCL/実装契約を変更するものではなくインフラ内部の実装判断のため ADR は起こしていない。
- **Verification Results**:
  - `just check-compose` - passed
  - `just check-monitoring`（`infra/k8s/monitoring` と `infra/k8s/monitoring/loki` 双方の kustomize build を含む） - passed
  - `just check` - passed
  - `just verify` - `test-go` の `TestAgentStatusMatchesSCL` のみ failed（下記 Disclosures の通り本 WI と無関係な既存の失敗）、他は passed
  - ローカルで `docker compose up` を実行し、Grafana `/api/datasources` で Prometheus/Loki 両データソースの自動登録、`/api/search` でダッシュボードフォルダの自動読込を確認。実際の `idp`/`worker` コンテナは本 WI と無関係な既存バグ（`infra/schema/postgres.sql` を psqldef 3.11 が解析できず `schema` コンテナが exit 1 する）で起動できなかったため、ADR-018 形式の JSON を出力する模擬コンテナ（`compose_service=idp` ラベル付き）を使い、Promtail のパイプライン（JSON 抽出 → label 昇格 → structured metadata → timestamp 解析）が Loki への実際の取り込みと Grafana 経由のクエリで機能することを確認した。
- **Disclosures**:
  - Out of Scope（`backend/shared/logging` の出力形式変更、Elasticsearch 等の代替スタック評価）は変更していない。
  - `just verify` は本 WI と無関係な既存の失敗が2件ある: (1) `test-go`: `TestAgentStatusMatchesSCL`（Agent ドメインの SCL/Go enum 大文字小文字不一致）、(2) `just check-k8s`: `infra/k8s/base/kustomization.yaml` が存在しない `relay.yaml` を参照している。どちらも本 WI が触れたファイルとは無関係で、修正は別 work item の対象とすべき。
