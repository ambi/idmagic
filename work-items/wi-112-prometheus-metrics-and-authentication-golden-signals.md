---
depends_on: []
status: pending
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
- [ ] T001 [Inventory/SCL] 既存metricとSCL objectivesを照合し、name/type/unit/labels/buckets/ownerをcatalog化してMetricsExpositionを再生成する。
- [ ] T002 [Bootstrap] shared observabilityにOTel MeterProvider/Prometheus exporterを追加し、management `/metrics` routeとshutdownをcomposition rootへ接続する。
- [ ] T003 [HTTP RED] route-template request count/error/duration/in-flight/abortをmiddlewareで一度だけ計測する。
- [ ] T004 [Authentication] login outcome/method、token grant/outcome/duration、throttle hit/store failureを各usecase確定点へ計装する。
- [ ] T005 [Cardinality] forbidden labels/values、series budget、histogram bucketsのunit/contract testsとstatic review checklistを追加する。
- [ ] T006 [Monitoring] compose scrape、RED/auth recording+alert rules、最小dashboardをcatalog名だけで作りwi-11の入力にする。
- [ ] T007 [Verify] OpenMetrics scrape、success/failure/throttle fixtureのdelta、concurrent request、tenant/user非露出、exporter disabled/errorを検証する。

## Verification
- just verify
- 手動: /metrics を curl し、login/token/throttle と HTTP RED のメトリクスが OpenMetrics 形式で出力されることを確認する。

## Risk Notes
メトリクスの label カーディナリティ (tenant / client_id 等) が過大にならないよう、
高カーディナリティ次元は慎重に選ぶ。ADR-017/018 との整合 (OTel を第一の観測 IF と
するか prometheus client を併用するか) を実装前に確認する。
