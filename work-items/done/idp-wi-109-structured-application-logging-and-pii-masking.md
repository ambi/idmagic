---
id: idp-wi-109-structured-application-logging-and-pii-masking
title: "構造化アプリケーションログ (Logger ポート) を導入し ADR-018 のアプリログ半分を実装する"
created_at: 2026-07-04
authors: ["tn"]
status: completed
risk: low
---
# Motivation
ADR-018 は監査ログとアプリケーションログを分離し、アプリログは
「level / service / trace_id / span_id / message を持つ構造化 JSON Lines を
stdout に出力」「x-pii フィールド (email 等) をアプリログに出さない」ことを
採用済み決定としている。しかし実装はいまだ標準ライブラリの log.Printf に留まり、
この決定のアプリログ半分が未実装のままである。

具体的な帰結として:
- 出力が非構造 (プレーンテキスト) で level フィールドを持たないため、
  Loki / OpenSearch でのフィルタリングや OTel Collector の filelog receiver
  による取り込みが機能しない。
- trace_id / span_id が付与されず、wi-107 で導入する分散トレースとログを
  相関できない。
- internal/shared/adapters/notification/smtp_email_sender.go と
  email_sender.go が宛先メールアドレス (x-pii=true) を平文でログ出力しており、
  ADR-018 §4 の PII マスキング要件に違反している (実在のコンプライアンス欠陥)。

本番運用ではインシデント調査・SLO 計測・コンプライアンス監査のいずれにおいても
構造化アプリログが前提となるため、Go 標準の log/slog を用いた Logger ポートと
そのアダプタを整備し、既存の log.Printf 呼び出し全てを置換する。

# Scope
- **scl**: spec/contexts/system.yaml objectives に StructuredApplicationLog を追加し、必須フィールド (timestamp/level/service/trace_id/span_id/message)、PII マスキング (x-pii フィールドをアプリログに出さない)、出力先 (stdout JSON Lines)、レベル集合を宣言する。ADR-018 の decision を SCL objective として明文化する。
- **go**: shared に Logger ポート (Debug/Info/Warn/Error(ctx, msg, attrs...) + Audit は既存 EventSink 経由のまま) を定義する。, log/slog をバックエンドとする slog Logger アダプタを internal/shared/adapters/observability に追加する。JSON Handler、LOG_LEVEL env による level 制御、service/version の固定 attribute、ctx からの trace_id/span_id 注入 (OTel span context) を実装する。, cmd/idmagic / cmd/idmagic-relay / internal/bootstrap / internal/relay および各アダプタの log.Printf / log.Fatal をすべて Logger ポート経由に置換する。, smtp_email_sender.go / email_sender.go の宛先メールアドレスのログ出力を PII マスキング (ドメイン保持のローカルパート伏字等) に是正する。
- **monitoring**: アプリログが stdout JSON Lines として OTel Collector / fluentbit に取り込める形式であることを確認する (ADR-018 §1)。

# Out of Scope
- 監査ログ (DomainEvent / EventSink / outbox) の変更。ADR-018 で監査は別経路であり本 WI はアプリログ半分のみを対象とする。
- OTel logs SDK 経由の非同期ログ配送 (OTLP logs exporter) の統合。まずは stdout JSON Lines を確立し、trace 相関は span context 由来の attribute で担保する。OTLP logs 配送は将来の WI とする。
- CI での check-no-pii-in-logs 静的検査の追加 (ADR-018 で Phase 3 相当)。
- アプリケーションログのメトリクス化・ダッシュボード整備 (別 WI: Prometheus メトリクス)。

# Verification
- just verify
- 手動: LOG_LEVEL=info でサーバを起動し、起動ログ・email 送信ログ・retention sweep ログが 1 行 1 JSON オブジェクト (level/service/message フィールドを持つ) として出力されることを確認する。
- 手動: email 送信経路のログに宛先メールアドレスのローカルパートが平文で出ないことを確認する。
- 手動 (OTel 有効時): OBSERVABILITY=otel でトレース内のログに trace_id / span_id が付与されることを確認する。

# Risk Notes
変更は横断的だが、置換対象の log.Printf は 17 箇所と限定的で、いずれも副作用のない
出力処理である。Logger ポートを noop/console に切り替え可能にし、既存の起動・デモ体験を
壊さないことを回帰の主眼とする。slog は Go 標準ライブラリであり新規依存を増やさない。

# Completion
- **Completed At**: 2026-07-04
- **Summary**:
  ADR-018 のアプリログ半分を実装した。internal/shared/logging に Logger ポートと
  log/slog ベースの JSON Lines アダプタを新設し、必須フィールド (timestamp/level/
  service/message)、LOG_LEVEL によるレベル制御、service/version の固定 attribute、
  OTel span context 由来の trace_id/span_id 注入、slog 既定キーの ADR-018 命名への
  リネーム (time→timestamp, msg→message) を実装した。cmd / bootstrap / relay および
  notification / eventsink / policy の標準 log 呼び出しをすべて Logger 経由へ置換し、
  Echo フレームワークのログ (e.Logger) も同じハンドラに載せて field 規約を統一した。
  email 送信経路 (console / smtp) の宛先アドレス出力を MaskEmail (***@domain) で
  マスクし、ADR-018 §4 の PII 漏洩を是正した。SCL は system.yaml objectives に
  StructuredApplicationLog を追加して決定を明文化し、派生 HTML を再生成した。
- **Verification Results**:
  - just verify
  - go test ./internal/shared/logging
  - 手動: LOG_LEVEL=info でサーバ起動時、email sender / breached checker / server listening / Echo 起動ログが 1 行 1 JSON (timestamp/level/service/message) で出力されることを確認した。
  - 手動: email 送信経路のログで宛先アドレスがローカルパート伏字 (***@domain) となり平文の PII が出ないことをコードと単体テストで確認した。
