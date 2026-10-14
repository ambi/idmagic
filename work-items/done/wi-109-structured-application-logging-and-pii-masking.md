---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# 構造化アプリケーションログ (Logger ポート) を導入し ADR-018 のアプリログ半分を実装する

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

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
