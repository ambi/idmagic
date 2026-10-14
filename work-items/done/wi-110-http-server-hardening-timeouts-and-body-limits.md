---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# HTTP サーバに read/write/idle タイムアウトとボディ上限を設定しリソース枯渇 DoS を緩和する

system.yaml に HTTPServerHardening objective を追加し (read_header 10s / read 30s /
write 60s / idle 120s / max_body 1 MiB / oversize=413)、派生 HTML/JSON を再生成した。
internal/bootstrap/hardening.go に env 上書き可能な httpServerHardening 設定を新設し、
未指定・不正値は本番安全なデフォルトへフォールバックする (envDuration/envInt の規約)。
bootstrap/server.go では echo.StartConfig.BeforeServeFunc で基盤 http.Server に
ReadHeaderTimeout / ReadTimeout / WriteTimeout / IdleTimeout を設定し、echo v5 の
BodyLimit middleware を RequestID/Recover の内側・otel/handler の外側に登録して
全リクエストにボディ上限を課した (超過は 413、Content-Length と実読み取りの両面で判定)。
BodyLimit は request_id 付与と panic recover の内側に置き、拒否レスポンスも相関/回復対象
にする。アイコンアップロード上限 (256 KiB, MaxApplicationIconBytes) は既定 1 MiB 内に収まり
回帰しない。有効なタイムアウト値は起動ログに出力し、README に env 表と「エッジプロキシが
volumetric/ハンドシェイク DoS の主防御、アプリはプロキシ非依存の多層防御として自ホップを
守る」責務分担を追記した (WI 本文の Responsibility 節と整合)。
golang.org/x/time は BodyLimit middleware の間接依存として go.mod/go.sum に追加。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
