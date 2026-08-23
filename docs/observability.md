# Observability

## Request correlation

すべてのリクエストに `request_id` を割り当て、`X-Request-ID` レスポンスヘッダーと、そのリクエストに関するすべてのアプリケーションログへ付与する。`OBSERVABILITY=otel` の場合は `trace_id` と `span_id` も付与する。

`X-Request-ID` はクライアントが制御できるため、既定では受信値を無視して新しい ID を生成する。これにより、クライアントによる ID の偽装や意図的な衝突を防ぐ。`REQUEST_ID_TRUST_INBOUND=true` を設定できるのは、信頼できる境界プロキシがヘッダーを生成または無害化する場合だけである。クライアントの値をそのまま転送するプロキシは信頼してはならない。受信値を利用する場合も長さと文字種を制限し、ヘッダーとログへの注入を防ぐ。

## Metrics

`GET /metrics` は Prometheus と OpenMetrics の形式でメトリクスを公開する。ルートパターンごとの HTTP RED（件数、`status_code` によるエラー率、所要時間、処理中の数）に加え、SLO とアラートに使う認証のゴールデンシグナルを含む。

| Metric | Labels | Verifies |
| --- | --- | --- |
| `http_requests_total`, `http_request_duration_seconds`, `http_requests_in_flight` | `route`, `method`, `status_code` | 接点ごとの遅延とエラー率の目標 |
| `authn_login_attempts_total` | `outcome`, `reason_class`, `method` | ログインの成否というゴールデンシグナル |
| `authn_login_throttle_total` | `policy`, `outcome` | ログインのスロットルの発動率 |
| `endpoint_rate_limit_total` | `policy`, `outcome` | エンドポイントの流量制限の発動率 |
| `oauth2_token_issuance_total`, `oauth2_token_issuance_duration_seconds` | `grant_type`, `outcome` | grant 別の `/token` の発行率と遅延 |
| `http_request_aborts_total`, `operation_detached_completion_failures_total` | `kind` | 中断の扱い |

サービス目標の母集団、時間窓、除外条件、目標値は [capacity.md](capacity.md) が `SLO-*` と `CAP-*` の ID を付けて定める。アラートと負荷試験はその ID を名指しし、数値を再掲しない。Prometheus は HTTP RED メトリクスとスクレイプ状態をその定義に従って集約し、レイテンシー、非 5xx 比率、可用性を評価する。

`idmagic-worker` は自身の `/metrics` を管理専用の別リスナーで公開する。API プロセスの `/metrics` とは別のプロセスかつ別の実体であり、レーンごとに `jobs_claim_latency_seconds`、`jobs_duration_seconds`、`jobs_outcome_total`、`jobs_retry_total`、`jobs_queue_depth` を持つ。取得までの待ち時間と実行そのものにかかった時間は別の指標である。片方だけでは、遅いのが滞留なのか処理なのかを分けられない。`jobs_outcome_total` の `outcome="failed"` は試行上限に達した確定だけを数えるので、配信不能の件数はこれで読む。

ラベルの値は有限の集合に限る。値の種類に上限がない `tenant_id`、`user_id`、`client_id`、解決済みのリクエストパスはラベルにしない。エンドポイントは常に登録するが、起動時に Prometheus の構築が完了するまでは `503` を返す。公開先はループバックアドレス、管理ネットワーク、または認証付きプロキシの背後に限る。

## Logging

アプリケーションログは、`timestamp`、`level`、`service`、`message` と、相関用の `trace_id`、`span_id`、`request_id` を持つ JSON Lines として標準出力へ書く（`backend/shared/logging`）。プロセス自身は他の場所へログを書かない。

**ローカル**（`infra/docker/docker-compose.dev.yaml`）：Promtail は Docker Engine API（`docker_sd_configs`）ですべてのコンテナを検出し、ログを Loki へ送る。ホストのログディレクトリをマウントする必要はなく、Docker ソケットだけを使用する。Grafana には初回起動時に Prometheus と Loki のデータソースとゴールデンシグナルのダッシュボードを設定する。`mise run dev-compose` だけでメトリクスとログを閲覧できる。

**Kubernetes**（`infra/k8s/monitoring/loki/`）：Promtail は DaemonSet として動作し、`kubernetes_sd_configs` で Pod を検出して `/var/log/pods` を追尾する。Loki は永続ボリュームを持つ単一レプリカの StatefulSet として動作する。ファイルシステムへの保存は開発用の既定であり、本番クラスターではオブジェクトストレージを使う保持設定で上書きする。

| Field | Loki treatment | Why |
| --- | --- | --- |
| `service`, `level` | インデックスラベル | 有限の集合である |
| `trace_id`, `span_id`, `request_id` | 構造化メタデータ（ラベルではない） | 値の種類に上限がなく、インデックスラベルにすると組み合わせが爆発する。[Metrics](#metrics) で `tenant_id` と `user_id` をラベルにしない理由と同じ |
