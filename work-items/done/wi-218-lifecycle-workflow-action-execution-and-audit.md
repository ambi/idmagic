---
status: completed
archived_from: 603746f4691c04b04a38dd3631b6313e6010c11e
---

# lifecycle workflow action を冪等実行し監査可能にする

Lifecycle workflow の action executor を worker に接続し、各 action の結果を durable step checkpoint として保存するようにした。対象 user/resource を tenant 内で再検証し、同一 user の run を発火順に直列化する。

全文は `git show 603746f4691c04b04a38dd3631b6313e6010c11e:<このファイルのパス>` で読む。
