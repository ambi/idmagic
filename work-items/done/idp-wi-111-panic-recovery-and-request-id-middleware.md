---
id: idp-wi-111-panic-recovery-and-request-id-middleware
title: "パニックリカバリと Request ID ミドルウェアを導入し障害の局所化とログ相関を可能にする"
created_at: 2026-07-04
authors: ["tn"]
status: completed
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

# Completion
- **Completed At**: 2026-07-04
- **Summary**:
  RequestFaultIsolation objective を system.yaml に追加し、派生 HTML/JSON を再生成した。
  echo v5 の Recover / RequestID middleware を support パッケージに新設し、bootstrap で
  最外に RequestID、その内側に Recover を登録した (以降の otel / handler の panic とログを
  同じ request_id 配下に置く)。RecoverMiddleware は panic を捕捉して 500 (`internal_error`)
  へ変換し、panic 値・method・path・stack を構造化ログへ記録してプロセスを継続する。
  http.ErrAbortHandler は errors.Is 判定で再 panic し net/http の abort 意味論を保つ。
  client abort (context.Canceled) は returned error であって panic ではないため本経路に
  入らず、ClientAbortLogClassification / ClassifyCancel の分類を崩さない。
  RequestID は secure-by-default で、受信 X-Request-ID を無視して常に自前生成する。
  信頼できる境界プロキシがヘッダを所有・消毒する構成でのみ REQUEST_ID_TRUST_INBOUND=true で
  受信値の再利用を許可する。信頼有無に関わらず受信値はサニタイズ (許可文字 [A-Za-z0-9._-]、
  長さ 128 上限) してヘッダ/ログインジェクションを防ぐ。TRUSTED_FORWARDED_HOPS とは別軸
  (プロキシが XFF を消毒しても X-Request-ID を素通しし得る) のため専用フラグとした。
  request_id は logging パッケージの context 値としてアプリログ全行に付与し、OTel 有効時は
  trace_id/span_id と併記される。README の Configuration に環境変数と Request Correlation
  節 (信頼境界とプロキシ設定例) を追記した。
- **Verification Results**:
  - just verify (yaml-check / lint-go / test-go-race / UI build いずれも green)
  - go test ./internal/shared/adapters/http/support ./internal/shared/logging
  - RequestID: 未送信時は 32桁 hex を生成、非信頼時は受信値を無視 (偽装拒否)、信頼時は
    安全な受信値を再利用しつつ改行/空白/超過長を再生成することを単体テストで確認した。
  - Recover: handler panic が 500 (`internal_error`) を返しつつ後続リクエストが 200 で継続し、
    ログに request_id / stack / panic 値が相関して残ること、http.ErrAbortHandler が再 panic
    してログされないことを単体テストで確認した。
