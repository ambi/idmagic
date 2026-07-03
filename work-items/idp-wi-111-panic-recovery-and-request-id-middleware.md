---
id: idp-wi-111-panic-recovery-and-request-id-middleware
title: "パニックリカバリと Request ID ミドルウェアを導入し障害の局所化とログ相関を可能にする"
created_at: 2026-07-04
authors: ["tn"]
status: pending
risk: low
---
# Motivation
internal/bootstrap/server.go では echo に登録されるミドルウェアが OTel の
middleware のみで、Recover ミドルウェアが登録されていない。いずれかの HTTP
ハンドラで panic が発生すると、そのリクエストのゴルーチンが巻き戻され接続が
異常終了し、最悪の場合プロセス全体の安定性に波及する。本番 IdP では単一の
想定外入力が認証基盤全体の可用性を脅かしてはならない。

併せて Request ID (X-Request-ID) ミドルウェアが無いため、単一リクエストに紐づく
複数のログ行やクライアント報告を相関する識別子が存在しない。Request ID は
panic 発生時のスタック記録、構造化ログ (idp-wi-109)、監査/トレースの相関の
共通キーとなる基盤である。

# Scope
- **scl**: spec/contexts/system.yaml objectives に RequestFaultIsolation を追加し、ハンドラ panic を 500 に変換してプロセスを継続すること、全リクエストに request_id を付与しレスポンスヘッダ/ログに伝播することを宣言する。
- **go**: echo Recover ミドルウェアを登録し、panic を捕捉して 500 (問題詳細) に変換、スタックとともに構造化ログへ記録する。, echo RequestID ミドルウェアを登録し、X-Request-ID を受理/生成してレスポンスとログコンテキストに伝播する。, 既存の CancellationConsistency / ClientAbortLogClassification (client abort を server error にしない) と整合させ、panic 由来のみを server error として扱う。

# Out of Scope
- 全エンドポイントの RFC 7807 problem+json 統一 (別途の大きな作業)。
- 分散トレーシング本体 (既存 idp-wi-107)。

# Verification
- just verify
- 手動: 意図的に panic するテストハンドラが 500 を返しサーバが継続稼働すること、レスポンスに X-Request-ID が付くことを確認する。

# Risk Notes
Recover が client abort (context.Canceled) を server error として二重計上しないよう、
既存の abort 分類ロジックと順序・責務を整理する必要がある。
