---
status: completed
archived_from: 0fca46fb861aad86ad8b000c8feebf5172954e84
---

# idmagic の横断アダプタ実装ディレクトリを infrastructure に改名する

`internal/platform/` を `internal/infrastructure/` に改名し、top-level の
`infra/` を `deploy/` に改名した。Go import path、README、ADR、実行パスを更新した。
SCL、HTTP route、DB schema、公開 API は変更していない。

全文は `git show 0fca46fb861aad86ad8b000c8feebf5172954e84:<このファイルのパス>` で読む。
