---
depends_on: []
status: pending
authors: ["tn"]
risk: low
created_at: 2026-07-04
priority: p2
change_kind: feature
affected_spec:
  - { path: spec/contexts/system/scenarios.md, requirement: REQ-SYSTEM-001 }
---

# OpenTelemetry 分散トレーシングを統合し、リクエスト追跡とボトルネック検出を可能にする

## Motivation
現在 idmagic は Prometheus を用いたメトリクス監視（[[wi-112-prometheus-metrics-and-authentication-golden-signals]]）をカバーしているが、
個別の認証リクエストや設定変更がどのようなコールスタックや外部呼び出しを経て
レイテンシを発生させているかという「トランザクション追跡（分散トレーシング）」が欠落している。
IdP は外部 SAML/OIDC フェデレーション、PostgreSQL、非同期ジョブによる監査イベントの書き込み
など複数のコンポーネントと通信するため、本番環境での障害調査やボトルネック特定には、
W3C Trace Context を用いたコンテキスト伝播と OpenTelemetry Tracing の統合が不可欠である。

## Scope
- **go**: OpenTelemetry Go SDK (otel/trace) を API サーバーおよび UI ゲートウェイに導入する。, HTTP ハンドラー middleware にトレースインスツルメンテーション (otelhttp) を統合する。, W3C Trace Context ヘッダ (traceparent) のパースとダウンストリームへの伝播を実装する。, PostgreSQL (pgx) へ otel インスツルメンテーションを導入し、DB クエリをスパンとして可視化する。, 認証の成否、トークン発行、例外エラー発生時に、トレース情報（スパン属性）にエラーフラグやメタデータを付与する。
- **monitoring**: OpenTelemetry Collector 経由でトレースデータを Jaeger または外部 APM (Datadog 等) へエクスポートする設定を追加する。

## Out of Scope
- 特定の商用 APM ベンダー向けライブラリの直接導入（OTel 標準のみを使用する）。
- プロファイラ（pprof 等）の常時監視統合。

## Plan
- `spec/observability.md` と既存 `backend/shared/observability/telemetry_otlp/otel.go` を拡張し、global SDKを各contextが直接設定しない。server composition rootがTracerProvider/propagator/resourceを構築してshutdownを所有する。
- ingressはotelhttp/Echo middlewareでW3C traceparent/tracestateを抽出し、route template、method、statusを低cardinality属性にする。request_idは別の相関値としてspan/log/event metadataへ付与するが、trace IDの代用にしない。
- usecaseは重要なorchestrationだけmanual spanを持ち、domain entity/modelはOTel依存をimportしない。pgxと外部HTTP呼び出しは公式instrumentationまたはport wrapperでchild spanを作る。
- 非同期の境界は PostgreSQL のジョブキューだけである（[[wi-305-remove-external-message-brokers]] で外部ブローカーと outbox リレーは撤廃済み）。`Job` の `params` に trace context を明示伝播し、worker 側で producer span へ link した consumer span を作る。business payloadへtrace headerを混ぜない。
- attribute allowlistでtenant/user/client/token/IP/SQL bind値を禁止またはhashed分類にし、samplingはparent-based ratio + error tailのcollector側方針とする。exporter障害はrequestを失敗させずbounded queue/drop metricで観測する。

## Tasks
- [ ] T001 [Spec] 現行OTel初期化、HTTP/pgx/Jobs境界を棚卸しし、span taxonomy、async propagation、attribute/redaction/samplingを確定して `spec/observability.md` に記録する。
- [ ] T002 [Bootstrap] shared observabilityにTracerProvider/OTLP exporter/resource/propagator/shutdownとdisabled no-op modeを実装する。
- [ ] T003 [HTTP] Echo ingress/client instrumentation、route template/status/error、request_id log correlationを追加する。
- [ ] T004 [Storage/Usecase] pgx instrumentationと選定usecase manual spansを追加し、SQL値/PIIをrecordしない。
- [ ] T005 [Async] `Job` の `params` へ trace context を載せ、worker 側の producer/consumer span と link、retry attempt 属性を実装する。
- [ ] T006 [Collector] composeのOTel Collectorへtrace pipeline/sampling/exportを追加し、Jaeger等local backendはprofileで起動する。
- [ ] T007 [Verify] browser request→DB→job workerのsingle trace、remote parent、error/retry、exporter outage/backpressure、attribute PII scanを検証する。

## Verification
- `mise run verify-go`
- 手動: ローカル Docker Compose 環境で OpenTelemetry Collector と Jaeger を起動し、ログインを実行した際に DB クエリがネストしたスパンとして Jaeger UI 上に正しく描画されることを確認する。

## Risk Notes
トレーシングの導入は本番環境の CPU / メモリオーバーヘッドを増やす懸念があるため、
サンプリングレートを動的に構成可能にし、開発環境では 100%、本番環境では 5〜10% 程度に
抑えられる仕組みを用意する。
