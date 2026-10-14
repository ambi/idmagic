---
status: completed
archived_from: c8d1c0d0835de05f950bdc3d4bc87c7da3addb73
---

# Docker なしの標準開発環境で durable worker を稼働させる

Docker を使わず embedded PostgreSQL と miniredis を共有 TCP endpoint として起動し、API・worker・Vite を別プロセスで lifecycle 管理する標準 `just dev` を実装した。従来の API + UI memory mode は `just dev-memory` に分離し、durable jobs 非対応を明示した。fresh PostgreSQL で default tenant より先に signing key を作成していた bootstrap 順序も修正した。

全文は `git show c8d1c0d0835de05f950bdc3d4bc87c7da3addb73:<このファイルのパス>` で読む。
