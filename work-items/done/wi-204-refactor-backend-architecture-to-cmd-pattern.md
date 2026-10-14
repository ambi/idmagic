---
status: completed
archived_from: 0fca46fb861aad86ad8b000c8feebf5172954e84
---

# リポジトリ構造を Screaming Architecture と Go の標準レイアウトに適合させる

Screaming Architecture に従い、`backend/` 直下に配置されていたインフラ層・エントリーポイント起動ロジック (`bootstrap`, `relay`, `devinfra`) をすべて `backend/cmd/` 以下へ移動し整理しました。APIサーバー用とWorker用の起動ロジック (`server.go`, `worker.go`) はそれぞれ完全に独立した `main` パッケージとして `backend/cmd/idmagic/` と `backend/cmd/idmagic-worker/` に配置しました。また、共通のDIコンテナ構築ロジック (`deps.go` 等) は `backend/cmd/internal/bootstrap` 内に集約隔離し、内部の関数群（`assemble`等）をExport化（`Assemble`等）して呼び出すインターフェースへと整備しました。これにより、ドメイン層からの逆依存をコンパイラレベルで遮断しています。また、`ARCHITECTURE.md` の関連記述も新ディレクトリ構造に合わせて更新しました。

全文は `git show 0fca46fb861aad86ad8b000c8feebf5172954e84:<このファイルのパス>` で読む。
