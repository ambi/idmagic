---
status: completed
archived_from: dc0961d035553d99cd2b9ce5a472240293d05838
---

# 管理一覧で正確な件数とページ位置を表示し短い cursor で先頭・末尾へ移動できるようにする

管理一覧5 interfaceへexact count metadata、first/prev/next/last Link、端数を保つend anchor、compact v3 stateless
cursorを追加し、v0/v2 decode互換を維持した。Usersはfilter非依存のexact `total_users` とactive既定表示へ切り替え、
5画面の共通navigation、URL履歴・reload・共有、一覧内refresh後のmetadata同期を実装した。ADR-160、SCL、OpenAPI、
sqlc生成物を同期し、bounded contextやdirectory conventionは変更していないためarchitecture recordの更新は不要だった。

全文は `git show dc0961d035553d99cd2b9ce5a472240293d05838:<このファイルのパス>` で読む。
