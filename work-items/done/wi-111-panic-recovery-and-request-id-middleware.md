---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# パニックリカバリと Request ID ミドルウェアを導入し障害の局所化とログ相関を可能にする

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

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
