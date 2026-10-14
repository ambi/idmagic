---
status: completed
archived_from: 63b46e1b8c0daf4281ef92d5012cc8ea453b8aad
---

# Prometheus メトリクスエンドポイントと認証ゴールデンシグナルを公開し SLO 計測を可能にする

`GET /metrics` (system.yaml `MetricsExposition` interface) を追加し、`go.opentelemetry.io/otel/exporters/prometheus`
(OTel MeterProvider + Prometheus registry、既存の OTLP push tracing/metrics `Provider` とは独立)で
OpenMetrics 形式のスクレイプを常時提供するようにした。HTTP RED (`http_requests_total` /
`http_request_duration_seconds` / `http_requests_in_flight`) を route-template ラベルの新規 middleware
(`support.MetricsMiddleware`) で全route共通計測し、既存だが未接続だった `HTTPAbortMetrics`
(`http_request_aborts_total` / `operation_detached_completion_failures_total`) を composition root へ実配線した。
認証ゴールデンシグナルとして `authn_login_attempts_total`(outcome/reason_class/method)、
`authn_login_throttle_total`(policy/outcome)、`oauth2_token_issuance_total` /
`oauth2_token_issuance_duration_seconds`(grant_type/outcome) を、監査イベントの集約抑制とは独立した
確定decision pointへ計装した。authentication.yaml に `LoginLatency`/`LoginErrorRate` objectives
(oauth2.yaml の既存 latency/error-rate objectives と同型)を新設し、`/metrics` で計測可能にした。
`infra/docker/` に compose scrape 設定、promtool で構文検証済みの recording/alert rules、
最小 Grafana dashboard を追加し、wi-11 (Kubernetes/monitoring 資産) の入力とした。

全文は `git show 63b46e1b8c0daf4281ef92d5012cc8ea453b8aad:<このファイルのパス>` で読む。
