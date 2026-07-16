---
depends_on: []
status: completed
authors: ["tn"]
risk: low
created_at: 2026-07-04
---

# Prometheus メトリクスエンドポイントと認証ゴールデンシグナルを公開し SLO 計測を可能にする

## Motivation
現状 idmagic は OpenTelemetry トレーシング (wi-107) の導入は計画しているが、
Prometheus/OpenMetrics 形式で収集できるアプリケーションメトリクスを一切公開して
いない。ADR-018 のリテンション表はメトリクス時系列を Prometheus で 30 日保持し
SLO 計測ウィンドウとする前提を置いているが、その計測点が実装されていない。

本番 IdP の運用では、トレース (個別リクエストの追跡) とは別に、集約された
ゴールデンシグナル — login 成功/失敗率、token 発行レイテンシとエラー率、
login throttle ヒット、依存先 (DB/Valkey) の健全性 — を時系列で継続監視し、
アラートと SLO の根拠にする必要がある。Prometheus scrape 可能な /metrics
エンドポイントと主要カウンタ/ヒストグラムを提供する。

## Scope
- **scl**: spec/contexts/system.yaml objectives に MetricsExposition を追加し、/metrics の公開、RED (Rate/Errors/Duration) を満たす HTTP メトリクスと認証ゴールデンシグナル (login/token/throttle) のカタログを宣言する。
- **go**: OpenTelemetry metrics または prometheus client を用いて /metrics (OpenMetrics) を公開する。ADR-017 (OTel を観測インターフェースとする) と整合させる。, HTTP RED メトリクス (リクエスト数・エラー・レイテンシ histogram) をミドルウェアで収集する。既存 http_request_aborts_total の abort 分類 label と整合させる。, 認証ドメインのゴールデンシグナル (login 成否 counter、token 発行 counter/latency、throttle ヒット counter) を計装する。
- **monitoring**: docker compose / deploy に Prometheus scrape 設定例と最小ダッシュボードを追加する。

## Out of Scope
- 分散トレーシング本体 (既存 wi-107)。
- アラートルール / PagerDuty 等の通知連携。
- 負荷試験資産 (既存 wi-11 の範囲)。

## Plan
- [[ADR-017-opentelemetry-as-observability-interface]] に従い、Go codeはOTel Meterを使い、Prometheus exporterが`/metrics`をOpenMetricsとして公開する。別prometheus clientで二重registryを作らない。
- `/metrics`はapplication API route/tenant middlewareから分離したsystem endpointとし、公開範囲（loopback/management networkまたはauth）をdeploy policyで制御する。tenant/user/client/path実値をlabelにしない。
- HTTP REDはroute template/method/status class、loginはoutcome/reason class/method、tokenはgrant/outcome、throttleはpolicy/outcomeをlabelとする。duration histogram bucketはSCL latency objectives周辺を識別できる値にする。
- 既存`http_request_aborts_total`等のmetricをinventoryし、rename時は互換期間かmigration noteを設ける。counterは確定したoutcome pointで一度だけ記録し、retry/middlewareで二重countしない。
- Prometheus scrape/recording/alert/dashboardはwi-11 operational assetsがconsumerになるため、本WIでmetric contractと最小compose exampleを完成させ、cluster manifestはwi-11へ委譲する。

## Tasks
- [x] T001 [Inventory/SCL] 既存metricとSCL objectivesを照合し、name/type/unit/labels/buckets/ownerをcatalog化してMetricsExpositionを再生成する。SCL 3.0 (ADR-103) では `objectives` が観測可能な SLI/SLO 専用のため、`MetricsExposition` は system.yaml の `interfaces`(endpoint契約)として追加し、authentication.yaml に既存 oauth2.yaml latency/error-rate objectives と同型の `LoginLatency`/`LoginErrorRate` objectives を新設した。
- [x] T002 [Bootstrap] shared observabilityにOTel MeterProvider/Prometheus exporterを追加し、management `/metrics` routeとshutdownをcomposition rootへ接続する。`observability.Metrics`(OTel Prometheus exporter reader、専用registry)を追加し、`OBSERVABILITY` (OTLP push tracing/metrics) とは独立に常時構築、`server.Deps.MetricsHandler` 経由で `/metrics` を配線した。
- [x] T003 [HTTP RED] route-template request count/error/duration/in-flightをmiddlewareで一度だけ計測する。`support.MetricsMiddleware` を追加し、recovered panic の最終status(500)も含めて観測できるよう `RecoverMiddleware` の外側に登録した。abort分類は既存 `HTTPAbortMetrics` の実装(`observability.Metrics`)をcomposition rootへ接続して有効化した(従来 nil で no-op だった箇所)。
- [x] T004 [Authentication] login outcome/method、token grant/outcome/duration、throttle hit/store failureを各usecase確定点へ計装する。集約抑制(`AuthenticationEventAggregated`)とは独立に、確定した失敗/成功/throttled決定点でカウンタを記録する。
- [x] T005 [Cardinality] forbidden labels/values、series budget、histogram bucketsのunit/contract testsとstatic review checklistを追加する。`observability` パッケージに scrape 出力の label 存在検証、`support` パッケージに route-template 検証のユニットテストを追加した。
- [x] T006 [Monitoring] compose scrape、RED/auth recording+alert rules、最小dashboardをcatalog名だけで作りwi-11の入力にする。`infra/docker/prometheus.yml`、`prometheus-rules.yml`(promtoolで構文検証済み)、`grafana-dashboard.json`を追加した。
- [x] T007 [Verify] OpenMetrics scrape、success/failure/throttle fixtureのdelta、tenant/user非露出を検証する。単体テスト(`observability`, `support`)と実HTTPハンドラ経由のe2eテスト(`server_test`: login success/invalid_credentials/throttled、token client_credentials)を追加し、全てpass。

## Verification
- just verify
- 手動: /metrics を curl し、login/token/throttle と HTTP RED のメトリクスが OpenMetrics 形式で出力されることを確認する。

## Risk Notes
メトリクスの label カーディナリティ (tenant / client_id 等) が過大にならないよう、
高カーディナリティ次元は慎重に選ぶ。ADR-017/018 との整合 (OTel を第一の観測 IF と
するか prometheus client を併用するか) を実装前に確認する。

## Completion
- **Completed At**: 2026-07-17
- **Summary**:
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
- **Verification Results**:
  - `just yaml-check` (SCL/work-item/ID部分) - passed
  - `just format-go` / `just lint-go` - passed (0 issues)
  - `just test-go-race` - passed (全パッケージ、新規テスト含む)
  - `just verify-ui` (`just verify` 内の UI 一式: format/lint/typecheck/build/vitest) - passed
  - `docker run ... promtool check rules/config` - passed (`infra/docker/prometheus-rules.yml`, 9 rules)
  - 手動: `docker compose -f infra/docker/docker-compose.dev.yaml config` で compose 構文を確認
  - 注記: `just verify` 全体は、本WIと無関係な並行作業 (wi-232 executable-architecture,
    `tools/yaml-check/schemas/architecture.schema.json` 移行中) による `ARCHITECTURE.md` schema
    不整合で失敗する。本WIのスコープ (SCL/Go/monitoring assets) に閉じた検証はすべて green。
- **Affected Guarantees State**:
  `/metrics` は OTLP push tracing/metrics の有効・無効に関わらず常時到達可能になり、HTTP RED と
  認証ゴールデンシグナルが tenant_id/user_id/client_id を label に含めない形で観測可能になった。
  `oauth2.yaml` の latency/error-rate objectives と新設した `authentication.yaml` の
  `LoginLatency`/`LoginErrorRate` objectives は、この計測点により実測可能な状態になった。
  wi-11 は本WIの metric catalog と compose 資産を前提に、cluster 向け ServiceMonitor/PrometheusRule/
  dashboard 展開へ進める。
