---
depends_on: []
status: pending
authors: ["tn"]
risk: low
created_at: 2026-07-04
---

# OpenTelemetry 分散トレーシングを統合し、リクエスト追跡とボトルネック検出を可能にする

## Motivation
現在 idmagic は Prometheus を用いたメトリクス監視（WI-11）をカバーしているが、
個別の認証リクエストや設定変更がどのようなコールスタックや外部呼び出しを経て
レイテンシを発生させているかという「トランザクション追跡（分散トレーシング）」が欠落している。
IdP は外部 SAML/OIDC フェデレーション、PostgreSQL、Valkey、監査イベントの非同期書き込み
など複数のコンポーネントと通信するため、本番環境での障害調査やボトルネック特定には、
W3C Trace Context を用いたコンテキスト伝播と OpenTelemetry Tracing の統合が不可欠である。

## Scope
- **go**: OpenTelemetry Go SDK (otel/trace) を API サーバーおよび UI ゲートウェイに導入する。, HTTP ハンドラー middleware にトレースインスツルメンテーション (otelhttp) を統合する。, W3C Trace Context ヘッダ (traceparent) のパースとダウンストリームへの伝播を実装する。, PostgreSQL (pgx) および Valkey クライアントへ otel インスツルメンテーションを導入し、DB クエリとキャッシュ操作をスパンとして可視化する。, 認証の成否、トークン発行、例外エラー発生時に、トレース情報（スパン属性）にエラーフラグやメタデータを付与する。
- **monitoring**: OpenTelemetry Collector 経由でトレースデータを Jaeger または外部 APM (Datadog 等) へエクスポートする設定を追加する。

## Out of Scope
- 特定の商用 APM ベンダー向けライブラリの直接導入（OTel 標準のみを使用する）。
- プロファイラ（pprof 等）の常時監視統合。

## Verification
- just verify-go
- 手動: ローカル Docker Compose 環境で OpenTelemetry Collector と Jaeger を起動し、ログインを実行した際に DB クエリや Valkey アクセスがネストしたスパンとして Jaeger UI 上に正しく描画されることを確認する。

## Risk Notes
トレーシングの導入は本番環境の CPU / メモリオーバーヘッドを増やす懸念があるため、
サンプリングレートを動的に構成可能にし、開発環境では 100%、本番環境では 5〜10% 程度に
抑えられる仕組みを用意する。
