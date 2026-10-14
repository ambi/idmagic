---
status: completed
archived_from: ed11f1540939b63772b44205974683834b62dac1
---

# idmagic の Kubernetes・監視・負荷試験資産を整備する

System の運用 scenario を追加し、API/UI gateway/relay を分離する Kubernetes base と dev/prod overlay、
probe・PDB・NetworkPolicy・external Secret reference を整備した。実在する HTTP RED、login、token metric
だけから PrometheusRule、Grafana dashboard、任意適用の ServiceMonitor を追加した。k6 は authorization
code + S256 PKCE、refresh rotation、client credentials を development seed の固定 UUID client で実行する。
`demo.sh` と frontend の開発手順に残っていた旧 `demo-client` ID を UUID へ同期した。

全文は `git show ed11f1540939b63772b44205974683834b62dac1:<このファイルのパス>` で読む。
