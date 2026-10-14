---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# 依存ヘルスを検査する liveness/readiness/startup probe と SIGTERM 時の接続ドレインを整備する

Kubernetes 準拠のヘルスプローブエンドポイント（livez/readyz/startupz）の分離と、SIGTERM受信時の接続ドレイン猶予期間（既定5秒）を設けた graceful shutdown 協調動作を実装し、E2Eテストを追加。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。
